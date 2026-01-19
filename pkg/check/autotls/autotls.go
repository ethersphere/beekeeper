package autotls

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents check options
type Options struct {
	ExpectedDomain   string // expected domain suffix in SNI (e.g., "local.test")
	TargetNodeGroups []string
	ConnectTimeout   time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ExpectedDomain: "",
		ConnectTimeout: 30 * time.Second,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

// WSSUnderlay represents a parsed WSS underlay address
type WSSUnderlay struct {
	Raw       string
	IP        string
	Port      string
	SNIDomain string
	PeerID    string
	IsIPv6    bool
}

var (
	errNoWSSNodes       = errors.New("no nodes with WSS underlay addresses found")
	errInvalidWSSFormat = errors.New("invalid WSS address format")
	errWSSConnect       = errors.New("WSS connection failed")
)

// Pre-compiled regex patterns for WSS multiaddr parsing
var (
	ipv4WSSPattern = regexp.MustCompile(`^/ip4/([0-9.]+)/tcp/(\d+)/tls/sni/([^/]+)/ws/p2p/(.+)$`)
	ipv6WSSPattern = regexp.MustCompile(`^/ip6/([^/]+)/tcp/(\d+)/tls/sni/([^/]+)/ws/p2p/(.+)$`)
)

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	c.logger.Info("starting AutoTLS check")

	wssNodes, err := c.discoverWSSNodes(ctx, cluster, o.TargetNodeGroups)
	if err != nil {
		return fmt.Errorf("discover WSS nodes: %w", err)
	}

	if len(wssNodes) == 0 {
		return errNoWSSNodes
	}

	c.logger.Infof("found %d nodes with WSS underlays", len(wssNodes))

	nodeNames := make([]string, 0, len(wssNodes))
	for name := range wssNodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	for _, nodeName := range nodeNames {
		underlays := wssNodes[nodeName]
		c.logger.Infof("validating WSS addresses for node %s", nodeName)

		for _, underlay := range underlays {
			parsed, err := parseWSSUnderlay(underlay)
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			if err := c.validateWSSAddress(parsed, o.ExpectedDomain); err != nil {
				return fmt.Errorf("node %s: address %s: %w", nodeName, underlay, err)
			}

			c.logger.Infof("node %s: valid WSS address: %s", nodeName, underlay)
		}
	}

	if err := c.testWSSConnectivity(ctx, cluster, wssNodes, o); err != nil {
		return fmt.Errorf("WSS connectivity test: %w", err)
	}

	c.logger.Info("AutoTLS check completed successfully")
	return nil
}

// discoverWSSNodes finds all nodes that have WSS underlay addresses
func (c *Check) discoverWSSNodes(ctx context.Context, cluster orchestration.Cluster, targetGroups []string) (map[string][]string, error) {
	wssNodes := make(map[string][]string)

	addresses, err := cluster.Addresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cluster addresses: %w", err)
	}

	for groupName, groupAddrs := range addresses {
		// Filter by target groups if specified
		if len(targetGroups) > 0 && !slices.Contains(targetGroups, groupName) {
			continue
		}

		for nodeName, nodeAddrs := range groupAddrs {
			wssUnderlays := filterWSSUnderlays(nodeAddrs.Underlay)
			if len(wssUnderlays) > 0 {
				wssNodes[nodeName] = wssUnderlays
				c.logger.Debugf("node %s has %d WSS underlay(s)", nodeName, len(wssUnderlays))
			}
		}
	}

	return wssNodes, nil
}

// filterWSSUnderlays returns only underlay addresses that contain TLS/WSS components
func filterWSSUnderlays(underlays []string) []string {
	var wss []string
	for _, u := range underlays {
		if isWSSUnderlay(u) {
			wss = append(wss, u)
		}
	}
	return wss
}

// isWSSUnderlay checks if an underlay address is a WSS address
// WSS addresses contain /tls/ and /ws/ components
func isWSSUnderlay(underlay string) bool {
	return strings.Contains(underlay, "/tls/") && strings.Contains(underlay, "/ws/")
}

// parseWSSUnderlay parses a WSS multiaddr into its components
// Expected formats:
//   - IPv4: /ip4/x.x.x.x/tcp/port/tls/sni/domain/ws/p2p/peerID
//   - IPv6: /ip6/::1/tcp/port/tls/sni/domain/ws/p2p/peerID
func parseWSSUnderlay(underlay string) (*WSSUnderlay, error) {
	re := ipv4WSSPattern
	matches := re.FindStringSubmatch(underlay)
	if matches != nil {
		return &WSSUnderlay{
			Raw:       underlay,
			IP:        matches[1],
			Port:      matches[2],
			SNIDomain: matches[3],
			PeerID:    matches[4],
			IsIPv6:    false,
		}, nil
	}

	re = ipv6WSSPattern
	matches = re.FindStringSubmatch(underlay)
	if matches != nil {
		return &WSSUnderlay{
			Raw:       underlay,
			IP:        matches[1],
			Port:      matches[2],
			SNIDomain: matches[3],
			PeerID:    matches[4],
			IsIPv6:    true,
		}, nil
	}

	return nil, fmt.Errorf("%w: %s", errInvalidWSSFormat, underlay)
}

// validateWSSAddress validates the WSS address components
func (c *Check) validateWSSAddress(wss *WSSUnderlay, expectedDomain string) error {
	if wss.IP == "" {
		return fmt.Errorf("missing IP address")
	}

	if ip := net.ParseIP(wss.IP); ip == nil {
		return fmt.Errorf("invalid IP address: %s", wss.IP)
	}

	if wss.Port == "" {
		return fmt.Errorf("missing port")
	}

	if wss.SNIDomain == "" {
		return fmt.Errorf("missing SNI domain")
	}

	if expectedDomain != "" && !strings.HasSuffix(wss.SNIDomain, expectedDomain) {
		return fmt.Errorf("SNI domain %q does not have expected suffix %q", wss.SNIDomain, expectedDomain)
	}

	if !isValidAutoTLSSNI(wss.SNIDomain, wss.IP, wss.IsIPv6) {
		c.logger.Warningf("SNI domain %s may not follow AutoTLS naming convention for IP %s", wss.SNIDomain, wss.IP)
	}

	if wss.PeerID == "" {
		return fmt.Errorf("missing peer ID")
	}

	return nil
}

func isValidAutoTLSSNI(sni, ip string, isIPv6 bool) bool {
	var ipWithDashes string
	if isIPv6 {
		ipWithDashes = ip
		if strings.HasPrefix(ipWithDashes, "::") {
			ipWithDashes = "0" + ipWithDashes
		}
		ipWithDashes = strings.ReplaceAll(ipWithDashes, ":", "-")
	} else {
		ipWithDashes = strings.ReplaceAll(ip, ".", "-")
	}
	return strings.HasPrefix(sni, ipWithDashes+".")
}

func (c *Check) testWSSConnectivity(ctx context.Context, cluster orchestration.Cluster, wssNodes map[string][]string, opts Options) error {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("get node clients: %w", err)
	}

	if len(clients) == 0 {
		return fmt.Errorf("no nodes available for connectivity test")
	}

	clientNames := make([]string, 0, len(clients))
	for name := range clients {
		clientNames = append(clientNames, name)
	}
	sort.Strings(clientNames)

	var nonWSSSource *bee.Client
	var nonWSSName string
	var wssSource *bee.Client
	var wssSourceName string

	for _, name := range clientNames {
		client := clients[name]
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

	targetNames := make([]string, 0, len(wssNodes))
	for name := range wssNodes {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	if nonWSSSource != nil {
		c.logger.Infof("testing cross-protocol: %s (non-WSS) to WSS nodes", nonWSSName)
		if err := c.testConnectivity(ctx, nonWSSSource, nonWSSName, wssNodes, targetNames, opts); err != nil {
			return fmt.Errorf("cross-protocol test: %w", err)
		}
	} else {
		c.logger.Warning("no non-WSS nodes available, skipping cross-protocol test")
	}

	if wssSource != nil {
		c.logger.Infof("testing WSS-to-WSS: %s to WSS nodes", wssSourceName)
		if err := c.testConnectivity(ctx, wssSource, wssSourceName, wssNodes, targetNames, opts); err != nil {
			return fmt.Errorf("WSS-to-WSS test: %w", err)
		}
	} else {
		c.logger.Warning("no WSS source nodes available, skipping WSS-to-WSS test")
	}

	return nil
}

func (c *Check) testConnectivity(ctx context.Context, sourceClient *bee.Client, sourceName string, wssNodes map[string][]string, targetNames []string, opts Options) error {
	for _, targetName := range targetNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if targetName == sourceName {
			continue
		}

		underlays := wssNodes[targetName]
		for _, underlay := range underlays {
			c.logger.Infof("testing WSS connection from %s to %s via %s", sourceName, targetName, underlay)

			connectCtx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
			start := time.Now()

			overlay, err := sourceClient.Connect(connectCtx, underlay)
			duration := time.Since(start)
			cancel()

			if err != nil {
				return fmt.Errorf("%w: from %s to %s via %s: %v", errWSSConnect, sourceName, targetName, underlay, err)
			}

			c.logger.Infof("WSS connection successful: %s to %s (overlay: %s, duration: %v)",
				sourceName, targetName, overlay, duration)

			if overlay.Equal(swarm.ZeroAddress) {
				return fmt.Errorf("received zero overlay address after connecting to %s", targetName)
			}

			if err := sourceClient.Disconnect(ctx, overlay); err != nil {
				return fmt.Errorf("failed to disconnect from %s: %w", targetName, err)
			}
		}
	}

	return nil
}
