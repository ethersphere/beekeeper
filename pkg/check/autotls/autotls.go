package autotls

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/cert"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	ma "github.com/multiformats/go-multiaddr"
)

type Options struct {
	AutoTLSGroups       []string
	UltraLightGroup     string
	ForgeDNSAddress     string
	ForgeTLSHostAddress string
	PebbleMgmtURL       string
}

func NewDefaultOptions() Options {
	return Options{
		AutoTLSGroups:   []string{"bee-autotls"},
		UltraLightGroup: "ultra-light",
	}
}

const (
	underlayPollInterval = 2 * time.Second
	connectTimeout       = 30 * time.Second
)

var _ beekeeper.Action = (*Check)(nil)

type Check struct {
	logger logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	c.logger.Info("starting AutoTLS check")

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("get node clients: %w", err)
	}

	autoTLSClients := orchestration.ClientMap(clients).FilterByNodeGroups(o.AutoTLSGroups)
	if len(autoTLSClients) == 0 {
		return fmt.Errorf("no nodes found in AutoTLS groups %v", o.AutoTLSGroups)
	}

	c.logger.Infof("found %d nodes in AutoTLS groups %v", len(autoTLSClients), o.AutoTLSGroups)

	wssNodes, err := c.verifyWSSUnderlays(ctx, autoTLSClients, o.UltraLightGroup)
	if err != nil {
		return fmt.Errorf("verify WSS underlays: %w", err)
	}

	forgeDomain, caCertPEM := c.forgeConfig(ctx, cluster, autoTLSClients, o.PebbleMgmtURL)
	if forgeDomain == "" {
		return fmt.Errorf("could not determine forge domain from node config")
	}

	forgeNodes, err := c.verifyForgeAddressFormat(wssNodes, forgeDomain)
	if err != nil {
		return fmt.Errorf("forge address validation: %w", err)
	}

	if err := c.verifyDNSResolution(ctx, forgeNodes, o.ForgeDNSAddress); err != nil {
		return fmt.Errorf("DNS resolution verification: %w", err)
	}

	if err := c.verifyTLSCertificate(ctx, forgeNodes, caCertPEM, o.ForgeTLSHostAddress); err != nil {
		return fmt.Errorf("TLS certificate verification: %w", err)
	}

	if err := c.testWSSConnectivity(ctx, clients, wssNodes, connectTimeout); err != nil {
		return fmt.Errorf("WSS connectivity test: %w", err)
	}

	if o.UltraLightGroup != "" {
		if err := c.testUltraLightConnectivity(ctx, clients, wssNodes, o.UltraLightGroup, connectTimeout); err != nil {
			return fmt.Errorf("ultra-light connectivity test: %w", err)
		}
	}

	if err := c.testCertificateRenewal(ctx, clients, wssNodes, forgeNodes, caCertPEM, o.ForgeTLSHostAddress, connectTimeout); err != nil {
		return fmt.Errorf("certificate renewal test: %w", err)
	}

	c.logger.Info("AutoTLS check completed successfully")
	return nil
}

func (c *Check) verifyWSSUnderlays(ctx context.Context, autoTLSClients orchestration.ClientList, excludeNodeGroup string) (map[string][]string, error) {
	autoTLS := make(map[string][]string)

	for _, client := range autoTLSClients {
		if excludeNodeGroup != "" && client.NodeGroup() == excludeNodeGroup {
			c.logger.Debugf("skipping %s (node group %s has no WSS underlays)", client.Name(), excludeNodeGroup)
			continue
		}

		nodeName := client.Name()
		var wssUnderlays []string
		for {
			addresses, err := client.Addresses(ctx)
			if err != nil {
				return nil, fmt.Errorf("%s: get addresses: %w", nodeName, err)
			}
			wssUnderlays = filterWSSUnderlays(addresses.Underlay)
			if len(wssUnderlays) > 0 {
				break
			}
			c.logger.Debugf("node %s has no WSS underlays yet, retrying in %v", nodeName, underlayPollInterval)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(underlayPollInterval):
			}
		}

		autoTLS[nodeName] = wssUnderlays
		c.logger.Debugf("node %s has %d WSS underlay(s)", nodeName, len(wssUnderlays))
	}

	return autoTLS, nil
}

func filterWSSUnderlays(underlays []string) []string {
	var wss []string
	for _, u := range underlays {
		maddr, err := ma.NewMultiaddr(u)
		if err != nil {
			continue
		}
		if _, err := maddr.ValueForProtocol(ma.P_TLS); err != nil {
			continue
		}
		if _, err := maddr.ValueForProtocol(ma.P_WS); err != nil {
			continue
		}
		wss = append(wss, u)
	}
	return wss
}

func (c *Check) testWSSConnectivity(ctx context.Context, clients map[string]*bee.Client, wssNodes map[string][]string, timeout time.Duration) error {
	var nonWSSSource *bee.Client
	var nonWSSName string
	var wssSource *bee.Client
	var wssSourceName string

	for name, client := range clients {
		if _, hasWSS := wssNodes[name]; hasWSS {
			if wssSource == nil {
				wssSource = client
				wssSourceName = name
			}
		} else {
			if nonWSSSource == nil {
				nonWSSSource = client
				nonWSSName = name
			}
		}
	}

	if nonWSSSource != nil {
		c.logger.Infof("testing cross-protocol: %s (non-WSS) to WSS nodes", nonWSSName)
		if err := c.testConnectivity(ctx, nonWSSSource, nonWSSName, clients, wssNodes, timeout); err != nil {
			return fmt.Errorf("cross-protocol test: %w", err)
		}
	} else {
		c.logger.Warning("no non-WSS nodes available, skipping cross-protocol test")
	}

	if wssSource != nil {
		c.logger.Infof("testing WSS-to-WSS: %s to WSS nodes", wssSourceName)
		if err := c.testConnectivity(ctx, wssSource, wssSourceName, clients, wssNodes, timeout); err != nil {
			return fmt.Errorf("WSS-to-WSS test: %w", err)
		}
	} else {
		c.logger.Warning("no WSS source nodes available, skipping WSS-to-WSS test")
	}

	return nil
}

func (c *Check) testUltraLightConnectivity(ctx context.Context, clients map[string]*bee.Client, wssNodes map[string][]string, ultraLightGroup string, timeout time.Duration) error {
	ultralightClients := orchestration.ClientMap(clients).FilterByNodeGroups([]string{ultraLightGroup})
	if len(ultralightClients) == 0 {
		c.logger.Warningf("no nodes found in ultra-light group %q, skipping ultra-light connectivity test", ultraLightGroup)
		return nil
	}

	c.logger.Infof("found %d nodes in ultra-light group %q", len(ultralightClients), ultraLightGroup)

	for _, client := range ultralightClients {
		nodeName := client.Name()
		c.logger.Infof("testing ultra-light to WSS: %s (no listen addr) to WSS nodes", nodeName)
		if err := c.testConnectivity(ctx, client, nodeName, clients, wssNodes, timeout); err != nil {
			return fmt.Errorf("ultra-light %s to WSS test: %w", nodeName, err)
		}
	}

	return nil
}

func (c *Check) testConnectivity(ctx context.Context, sourceClient *bee.Client, sourceName string, clients map[string]*bee.Client, wssNodes map[string][]string, timeout time.Duration) error {
	for targetName, underlays := range wssNodes {
		if targetName == sourceName {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		targetClient := clients[targetName]
		targetAddresses, err := targetClient.Addresses(ctx)
		if err != nil {
			return fmt.Errorf("get target %s addresses: %w", targetName, err)
		}
		targetOverlay := targetAddresses.Overlay

		// Disconnect first to ensure we test actual WSS connection.
		// Bee returns 200 OK for both new connections and existing ones,
		// so we must disconnect first to guarantee WSS transport is used.
		c.logger.Infof("disconnecting from %s before WSS test", targetName)
		if err := sourceClient.Disconnect(ctx, targetOverlay); err != nil {
			c.logger.Warningf("failed to disconnect from %s: %v", targetName, err)
		}

		for _, underlay := range underlays {
			c.logger.Infof("testing WSS connection from %s to %s via %s", sourceName, targetName, underlay)

			connectCtx, cancel := context.WithTimeout(ctx, timeout)
			start := time.Now()

			overlay, err := sourceClient.Connect(connectCtx, underlay)
			duration := time.Since(start)
			cancel()

			if err != nil {
				return fmt.Errorf("WSS connection failed from %s to %s via %s: %w", sourceName, targetName, underlay, err)
			}

			c.logger.Infof("WSS connection successful: %s to %s (overlay: %s, duration: %v)",
				sourceName, targetName, overlay, duration)

			if !overlay.Equal(targetOverlay) {
				return fmt.Errorf("overlay mismatch: expected %s, got %s", targetOverlay, overlay)
			}

			if err := sourceClient.Disconnect(ctx, overlay); err != nil {
				c.logger.Warningf("failed to disconnect from %s: %v", targetName, err)
			}
		}
	}

	return nil
}

func (c *Check) testCertificateRenewal(ctx context.Context, clients map[string]*bee.Client, wssNodes map[string][]string, forgeNodes map[string][]*forgeUnderlayInfo, caCertPEM, forgeTLSHostAddress string, connectTimeout time.Duration) error {
	const (
		renewalBuffer       = 2 * time.Minute
		expiredRenewalWait  = 10 * time.Minute // wait for next maintenance tick when cert already expired
		handshakeBlockLimit = 95 * time.Second // certmagic blocks up to 90s on handshake renewal
	)

	before := c.getCertSnapshots(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress)
	if len(before) == 0 {
		c.logger.Warning("no TLS endpoints reachable, skipping serial comparison")
	} else {
		var earliest time.Time
		for key, snap := range before {
			c.logger.Infof("%s: serial=%s expires=%s", key, snap.Serial, snap.NotAfter)
			if earliest.IsZero() || snap.NotAfter.Before(earliest) {
				earliest = snap.NotAfter
			}
		}

		untilExpiry := time.Until(earliest)
		var waitDuration time.Duration
		if untilExpiry <= 0 {
			c.logger.Info("cert already expired, triggering renewal via TLS connections")
			c.triggerRenewalConnections(ctx, forgeNodes, forgeTLSHostAddress, handshakeBlockLimit)
			waitDuration = expiredRenewalWait
		} else {
			waitDuration = untilExpiry + renewalBuffer
		}
		c.logger.Infof("earliest cert expires at %s, waiting %v", earliest, waitDuration)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}

		after := c.getCertSnapshots(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress)
		var renewed, unchanged int
		for key, pre := range before {
			post, ok := after[key]
			if !ok {
				c.logger.Warningf("%s: endpoint became unreachable after expiry", key)
				continue
			}
			if pre.Serial == post.Serial {
				unchanged++
				c.logger.Warningf("%s: serial unchanged (%s), renewal may not have occurred", key, pre.Serial)
			} else {
				renewed++
				c.logger.Infof("%s: renewed (serial %s -> %s)", key, pre.Serial, post.Serial)
			}
		}
		c.logger.Infof("certificate renewal verified: %d renewed, %d unchanged", renewed, unchanged)
		if renewed == 0 && unchanged > 0 {
			c.logger.Infof("no renewals yet, waiting 1 more minute for certmagic to complete")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Minute):
			}
			after = c.getCertSnapshots(ctx, forgeNodes, caCertPEM, forgeTLSHostAddress)
			renewed, unchanged = 0, 0
			for key, pre := range before {
				post, ok := after[key]
				if !ok {
					continue
				}
				if pre.Serial == post.Serial {
					unchanged++
				} else {
					renewed++
				}
			}
			c.logger.Infof("after retry: %d renewed, %d unchanged", renewed, unchanged)
		}
		if renewed == 0 && unchanged > 0 {
			return fmt.Errorf("no certificates renewed: %d/%d serials unchanged after expiry", unchanged, len(before))
		}
		if renewed == 0 && unchanged == 0 && len(before) > 0 {
			return fmt.Errorf("could not verify renewal: all %d endpoints unreachable after expiry", len(before))
		}
	}

	if err := c.testWSSConnectivity(ctx, clients, wssNodes, connectTimeout); err != nil {
		return fmt.Errorf("post-renewal connectivity test failed: %w", err)
	}

	c.logger.Info("certificate renewal test passed")
	return nil
}

// forgeConfig extracts the forge domain and CA certificate from the first autotls
// node's bee configuration. When Pebble is detected, the live root CA is fetched
// from Pebble's management API (since Pebble generates a fresh CA on each start).
// Falls back to the static embedded cert if the fetch fails.
func (c *Check) forgeConfig(ctx context.Context, cluster orchestration.Cluster, autoTLSClients orchestration.ClientList, pebbleMgmtURLOverride string) (forgeDomain, caCertPEM string) {
	nodes := cluster.Nodes()
	for _, client := range autoTLSClients {
		node, ok := nodes[client.Name()]
		if !ok || node.Config() == nil {
			continue
		}
		cfg := node.Config()
		forgeDomain = cfg.AutoTLSDomain
		if strings.Contains(cfg.AutoTLSCAEndpoint, "pebble") {
			mgmtURL := pebbleMgmtURL(cfg.AutoTLSCAEndpoint)
			if pebbleMgmtURLOverride != "" {
				mgmtURL = pebbleMgmtURLOverride
			}
			liveCert, err := fetchPebbleCACert(ctx, mgmtURL)
			if err != nil {
				c.logger.Warningf("failed to fetch live Pebble CA from %s, falling back to static cert: %v", mgmtURL, err)
				caCertPEM = cert.PebbleCertificate
			} else {
				c.logger.Infof("fetched live Pebble CA from %s", mgmtURL)
				caCertPEM = liveCert
			}
		}
		return forgeDomain, caCertPEM
	}
	return "", ""
}

// pebbleMgmtURL derives the Pebble management API URL from the ACME directory endpoint.
// E.g. "https://pebble:14000/dir" -> "https://pebble:15000/roots/0"
func pebbleMgmtURL(acmeEndpoint string) string {
	base := acmeEndpoint
	if i := strings.LastIndex(base, "/"); i > 0 {
		base = base[:i]
	}
	base = strings.Replace(base, ":14000", ":15000", 1)
	return base + "/roots/0"
}

// fetchPebbleCACert fetches the root CA PEM from Pebble's management API.
// Pebble's management endpoint uses a self-signed TLS cert, so we skip verification.
func fetchPebbleCACert(ctx context.Context, url string) (string, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(body), nil
}
