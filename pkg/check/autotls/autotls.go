package autotls

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	ma "github.com/multiformats/go-multiaddr"
)

type Options struct {
	WSSGroup        string
	UltraLightGroup string
	ConnectTimeout  time.Duration
}

func NewDefaultOptions() Options {
	return Options{
		WSSGroup:        "wss",
		UltraLightGroup: "ultralight",
		ConnectTimeout:  30 * time.Second,
	}
}

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

	wssClients := orchestration.ClientMap(clients).FilterByNodeGroups([]string{o.WSSGroup})
	if len(wssClients) == 0 {
		return fmt.Errorf("no nodes found in WSS group %q", o.WSSGroup)
	}

	c.logger.Infof("found %d nodes in WSS group %q", len(wssClients), o.WSSGroup)

	wssNodes, err := c.verifyWSSUnderlays(ctx, wssClients)
	if err != nil {
		return fmt.Errorf("verify WSS underlays: %w", err)
	}

	if err := c.testWSSConnectivity(ctx, clients, wssNodes, o.ConnectTimeout); err != nil {
		return fmt.Errorf("WSS connectivity test: %w", err)
	}

	if o.UltraLightGroup != "" {
		if err := c.testUltraLightConnectivity(ctx, clients, wssNodes, o.UltraLightGroup, o.ConnectTimeout); err != nil {
			return fmt.Errorf("ultra-light connectivity test: %w", err)
		}
	}

	// if err := c.testCertificateRenewal(ctx, clients, wssNodes, o.ConnectTimeout); err != nil {
	// 	return fmt.Errorf("certificate renewal test: %w", err)
	// }

	c.logger.Info("AutoTLS check completed successfully")
	return nil
}

func (c *Check) verifyWSSUnderlays(ctx context.Context, wssClients orchestration.ClientList) (map[string][]string, error) {
	wssNodes := make(map[string][]string)

	for _, client := range wssClients {
		nodeName := client.Name()
		addresses, err := client.Addresses(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: get addresses: %w", nodeName, err)
		}

		wssUnderlays := filterWSSUnderlays(addresses.Underlay)
		if len(wssUnderlays) == 0 {
			return nil, fmt.Errorf("node %s in WSS group has no WSS underlay addresses", nodeName)
		}

		wssNodes[nodeName] = wssUnderlays
		c.logger.Debugf("node %s has %d WSS underlay(s)", nodeName, len(wssUnderlays))
	}

	return wssNodes, nil
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

		time.Sleep(500 * time.Millisecond)

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

			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func (c *Check) testCertificateRenewal(ctx context.Context, clients map[string]*bee.Client, wssNodes map[string][]string, connectTimeout time.Duration) error {
	const renewalWaitTime = 500 * time.Second // This is configured in beelocal setup (we set certificate to expire in 300 seconds)

	c.logger.Infof("testing certificate renewal: waiting %v then re-testing connectivity", renewalWaitTime)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(renewalWaitTime):
	}

	c.logger.Info("wait complete, re-testing WSS connectivity to verify certificates were renewed")

	if err := c.testWSSConnectivity(ctx, clients, wssNodes, connectTimeout); err != nil {
		return fmt.Errorf("post-renewal connectivity test failed (certificates may not have been renewed): %w", err)
	}

	c.logger.Info("certificate renewal test passed: WSS connectivity still works after wait period")
	return nil
}
