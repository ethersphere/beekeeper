package autotls

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

const (
	renewalBuffer       = 2 * time.Minute
	expiredRenewalWait  = 10 * time.Minute // wait for next maintenance tick when cert already expired
	handshakeBlockLimit = 95 * time.Second // certmagic blocks up to 90s on handshake renewal
)

func (c *Check) testCertificateRenewal(ctx context.Context, clients map[string]*bee.Client, wssNodes map[string][]string, forgeNodes map[string][]*forgeUnderlayInfo, caCertPEM, forgeTLSHostAddress string, connectTimeout time.Duration) error {
	if err := c.verifyCertificateRenewal(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress); err != nil {
		return err
	}

	if err := c.testWSSConnectivity(ctx, clients, wssNodes, connectTimeout); err != nil {
		return fmt.Errorf("post-renewal connectivity test failed: %w", err)
	}

	c.logger.Info("certificate renewal test passed")
	return nil
}

func (c *Check) verifyCertificateRenewal(ctx context.Context, forgeNodes map[string][]*forgeUnderlayInfo, caCertPEM, forgeTLSHostAddress string) error {
	before := c.getCertSnapshots(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress)
	if len(before) == 0 {
		c.logger.Warning("no TLS endpoints reachable, skipping serial comparison")
		return nil
	}

	earliest := earliestCertExpiry(before)
	for key, snap := range before {
		c.logger.Infof("%s: serial=%s expires=%s", key, snap.Serial, snap.NotAfter)
	}

	waitDuration := renewalWaitDuration(earliest)
	if time.Until(earliest) <= 0 {
		c.logger.Info("cert already expired, triggering renewal via TLS connections")
		c.triggerRenewalConnections(ctx, forgeNodes, forgeTLSHostAddress, handshakeBlockLimit)
		waitDuration = expiredRenewalWait
	}
	c.logger.Infof("earliest cert expires at %s, waiting %v", earliest, waitDuration)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
	}

	after := c.getCertSnapshots(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress)
	renewed, unchanged := compareCertRenewals(c.logger, before, after, true)
	c.logger.Infof("certificate renewal verified: %d renewed, %d unchanged", renewed, unchanged)

	if renewed == 0 && unchanged > 0 {
		c.logger.Infof("no renewals yet, waiting 1 more minute for certmagic to complete")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Minute):
		}
		after = c.getCertSnapshots(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress)
		renewed, unchanged = compareCertRenewals(c.logger, before, after, false)
		c.logger.Infof("after retry: %d renewed, %d unchanged", renewed, unchanged)
	}

	if renewed == 0 && unchanged > 0 {
		return fmt.Errorf("no certificates renewed: %d/%d serials unchanged after expiry", unchanged, len(before))
	}
	if renewed == 0 && unchanged == 0 && len(before) > 0 {
		return fmt.Errorf("could not verify renewal: all %d endpoints unreachable after expiry", len(before))
	}
	return nil
}

func earliestCertExpiry(before map[string]certSnapshot) time.Time {
	var earliest time.Time
	for _, snap := range before {
		if earliest.IsZero() || snap.NotAfter.Before(earliest) {
			earliest = snap.NotAfter
		}
	}
	return earliest
}

func renewalWaitDuration(earliest time.Time) time.Duration {
	return time.Until(earliest) + renewalBuffer
}

// compareCertRenewals counts endpoints whose serial changed vs unchanged. If logDetails is true,
// logs per-key outcomes (including unreachable endpoints).
func compareCertRenewals(logger logging.Logger, before, after map[string]certSnapshot, logDetails bool) (renewed, unchanged int) {
	for key, pre := range before {
		post, ok := after[key]
		if !ok {
			if logDetails {
				logger.Warningf("%s: endpoint became unreachable after expiry", key)
			}
			continue
		}
		if pre.Serial == post.Serial {
			unchanged++
			if logDetails {
				logger.Warningf("%s: serial unchanged (%s), renewal may not have occurred", key, pre.Serial)
			}
		} else {
			renewed++
			if logDetails {
				logger.Infof("%s: renewed (serial %s -> %s)", key, pre.Serial, post.Serial)
			}
		}
	}
	return renewed, unchanged
}
