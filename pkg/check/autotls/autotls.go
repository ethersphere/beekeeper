package autotls

import (
	"context"
	"fmt"
	"time"

	"strings"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
	ma "github.com/multiformats/go-multiaddr"
)

type Options struct {
	AutoTLSGroup    string
	UltraLightGroup string
}

func NewDefaultOptions() Options {
	return Options{
		AutoTLSGroup:    "bee-autotls",
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

	autoTLSClients := orchestration.ClientMap(clients).FilterByNodeGroups([]string{o.AutoTLSGroup})
	if len(autoTLSClients) == 0 {
		return fmt.Errorf("no nodes found in AutoTLS group %q", o.AutoTLSGroup)
	}

	c.logger.Infof("found %d nodes in AutoTLS group %q", len(autoTLSClients), o.AutoTLSGroup)

	wssNodes, err := c.verifyWSSUnderlays(ctx, autoTLSClients, o.UltraLightGroup)
	if err != nil {
		return fmt.Errorf("verify WSS underlays: %w", err)
	}

	// Extract forge config from the first autotls node's bee config.
	forgeDomain, caCertPEM := c.forgeConfig(cluster, autoTLSClients)
	if forgeDomain == "" {
		return fmt.Errorf("could not determine forge domain from node config")
	}

	forgeNodes, err := c.verifyForgeAddressFormat(wssNodes, forgeDomain)
	if err != nil {
		return fmt.Errorf("forge address validation: %w", err)
	}

	if err := c.verifyDNSResolution(ctx, forgeNodes); err != nil {
		return fmt.Errorf("DNS resolution verification: %w", err)
	}

	if err := c.verifyTLSCertificate(ctx, forgeNodes, caCertPEM); err != nil {
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

	if err := c.testCertificateRenewal(ctx, clients, wssNodes, forgeNodes, caCertPEM, connectTimeout); err != nil {
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

func (c *Check) testCertificateRenewal(ctx context.Context, clients map[string]*bee.Client, wssNodes map[string][]string, forgeNodes map[string][]*forgeUnderlayInfo, caCertPEM string, connectTimeout time.Duration) error {
	const renewalWaitTime = 350 * time.Second // This is configured in beelocal setup (we set certificate to expire in 300 seconds)

	// Snapshot certificate serial numbers before waiting.
	preSerials := c.getCertSerials(ctx, forgeNodes, caCertPEM)
	if len(preSerials) > 0 {
		c.logger.Infof("captured %d certificate serial(s) before renewal wait", len(preSerials))
	} else {
		c.logger.Warning("no TLS endpoints reachable, will fall back to connectivity-only renewal check")
	}

	c.logger.Infof("testing certificate renewal: waiting %v for certificates to expire and renew", renewalWaitTime)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(renewalWaitTime):
	}

	c.logger.Info("wait complete, verifying certificates were renewed")

	// Verify serial numbers changed (proves new certs were issued).
	if len(preSerials) > 0 {
		postSerials := c.getCertSerials(ctx, forgeNodes, caCertPEM)
		var renewed, unchanged int
		for key, preSN := range preSerials {
			postSN, ok := postSerials[key]
			if !ok {
				c.logger.Warningf("%s: endpoint became unreachable after wait", key)
				continue
			}
			if preSN == postSN {
				unchanged++
				c.logger.Warningf("%s: certificate serial unchanged (%s), renewal may not have occurred", key, preSN)
			} else {
				renewed++
				c.logger.Infof("%s: certificate renewed (serial %s -> %s)", key, preSN, postSN)
			}
		}
		if unchanged > 0 && renewed == 0 {
			return fmt.Errorf("no certificates were renewed: %d/%d serials unchanged", unchanged, len(preSerials))
		}
		c.logger.Infof("certificate renewal verified: %d renewed, %d unchanged", renewed, unchanged)
	}

	// Also verify WSS connectivity still works with the new certificates.
	if err := c.testWSSConnectivity(ctx, clients, wssNodes, connectTimeout); err != nil {
		return fmt.Errorf("post-renewal connectivity test failed (certificates may not have been renewed): %w", err)
	}

	c.logger.Info("certificate renewal test passed")
	return nil
}

// forgeConfig extracts the forge domain and appropriate CA certificate from the
// first autotls node's bee configuration. If the CA endpoint indicates pebble
// (test environment), the embedded pebble CA cert is returned. Otherwise, an
// empty string is returned so the system root pool is used.
func (c *Check) forgeConfig(cluster orchestration.Cluster, autoTLSClients orchestration.ClientList) (forgeDomain, caCertPEM string) {
	nodes := cluster.Nodes()
	for _, client := range autoTLSClients {
		node, ok := nodes[client.Name()]
		if !ok || node.Config() == nil {
			continue
		}
		cfg := node.Config()
		forgeDomain = cfg.AutoTLSDomain
		if strings.Contains(cfg.AutoTLSCAEndpoint, "pebble") {
			caCertPEM = k8s.PebbleCertificate
		}
		return forgeDomain, caCertPEM
	}
	return "", ""
}
