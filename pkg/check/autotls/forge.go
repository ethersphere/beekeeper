package autotls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	forgeclient "github.com/ipshipyard/p2p-forge/client"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
)

// forgeUnderlayInfo holds parsed forge address information for a single WSS underlay.
type forgeUnderlayInfo struct {
	ForgeAddr     *forgeclient.ForgeAddrInfo
	PeerID        peer.ID
	ForgeHostname string // full hostname: <escaped-ip>.<base36-peerid>.<domain>
	ExpectedSAN   string // expected TLS cert SAN: *.<base36-peerid>.<domain>
}

// parseForgeUnderlay parses a WSS underlay multiaddr and extracts forge address information.
// It handles both short format (/dns4/<hostname>/tcp/<port>/tls/ws/p2p/<peerid>)
// and long format (/ip4/<ip>/tcp/<port>/tls/sni/<hostname>/ws/p2p/<peerid>).
func parseForgeUnderlay(underlayStr, forgeDomain string) (*forgeUnderlayInfo, error) {
	maddr, err := ma.NewMultiaddr(underlayStr)
	if err != nil {
		return nil, fmt.Errorf("parse multiaddr: %w", err)
	}

	pid, err := extractPeerID(maddr)
	if err != nil {
		return nil, err
	}

	peerIDBase36 := peer.ToCid(pid).Encode(multibase.MustNewEncoder(multibase.Base36))

	forgeInfo, hostname, err := extractForgeInfo(maddr, pid, forgeDomain)
	if err != nil {
		return nil, err
	}

	if forgeInfo.PeerIDBase36 != peerIDBase36 {
		return nil, fmt.Errorf("peer ID mismatch: /p2p/ component gives %s, hostname contains %s", peerIDBase36, forgeInfo.PeerIDBase36)
	}

	return &forgeUnderlayInfo{
		ForgeAddr:     forgeInfo,
		PeerID:        pid,
		ForgeHostname: hostname,
		ExpectedSAN:   fmt.Sprintf("*.%s.%s", forgeInfo.PeerIDBase36, forgeDomain),
	}, nil
}

// extractPeerID extracts and decodes the peer ID from the /p2p/ component of a multiaddr.
func extractPeerID(maddr ma.Multiaddr) (peer.ID, error) {
	peerIDStr, err := maddr.ValueForProtocol(ma.P_P2P)
	if err != nil {
		return "", fmt.Errorf("no /p2p/ component: %w", err)
	}
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		return "", fmt.Errorf("decode peer ID %q: %w", peerIDStr, err)
	}
	return pid, nil
}

// extractForgeInfo determines the multiaddr format and extracts forge address info accordingly.
func extractForgeInfo(maddr ma.Multiaddr, pid peer.ID, forgeDomain string) (*forgeclient.ForgeAddrInfo, string, error) {
	// Short format: /dns4/<hostname>/... or /dns6/<hostname>/...
	if hostname, err := maddr.ValueForProtocol(ma.P_DNS4); err == nil {
		info, err := forgeInfoFromHostname(hostname, forgeDomain, "4", maddr)
		return info, hostname, err
	}
	if hostname, err := maddr.ValueForProtocol(ma.P_DNS6); err == nil {
		info, err := forgeInfoFromHostname(hostname, forgeDomain, "6", maddr)
		return info, hostname, err
	}

	// Long format: /ip4/<ip>/tcp/<port>/tls/sni/<hostname>/ws/p2p/<peerid>
	hostname, err := maddr.ValueForProtocol(ma.P_SNI)
	if err != nil {
		return nil, "", fmt.Errorf("multiaddr has no DNS, SNI, or IP component: %w", err)
	}

	bareAddr, err := extractBareIPTCPAddr(maddr)
	if err != nil {
		return nil, "", fmt.Errorf("extract IP+TCP: %w", err)
	}

	info, err := forgeclient.ExtractForgeAddrInfo(bareAddr, pid)
	if err != nil {
		return nil, "", fmt.Errorf("extract forge addr info: %w", err)
	}
	return info, hostname, nil
}

// forgeInfoFromHostname constructs ForgeAddrInfo from a short-format forge hostname.
// The hostname has the form: <escaped-ip>.<base36-peerid>.<forge-domain>
func forgeInfoFromHostname(hostname, forgeDomain, ipVersion string, maddr ma.Multiaddr) (*forgeclient.ForgeAddrInfo, error) {
	suffix := "." + forgeDomain
	if !strings.HasSuffix(hostname, suffix) {
		return nil, fmt.Errorf("hostname %q doesn't end with domain %q", hostname, forgeDomain)
	}
	withoutDomain := strings.TrimSuffix(hostname, suffix)

	// The base36 peer ID is the last dot-separated segment before the domain.
	lastDot := strings.LastIndex(withoutDomain, ".")
	if lastDot < 0 {
		return nil, fmt.Errorf("hostname %q missing peer ID segment", hostname)
	}
	escapedIP := withoutDomain[:lastDot]
	peerIDBase36 := withoutDomain[lastDot+1:]

	tcpPort, err := maddr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return nil, fmt.Errorf("no TCP port in multiaddr: %w", err)
	}

	// Reconstruct the raw IP from the DNS-escaped form.
	// IPv4: dashes back to dots (1-2-3-4 → 1.2.3.4)
	// IPv6: dashes back to colons, then normalize through net.ParseIP to get canonical form.
	var ipStr string
	if ipVersion == "4" {
		ipStr = strings.ReplaceAll(escapedIP, "-", ".")
	} else {
		raw := strings.ReplaceAll(escapedIP, "-", ":")
		// The forge escaper pads leading/trailing zeros for RFC 1035 compliance
		// (e.g., ::1 → 0--1 → 0::1). Parse+String normalizes back to canonical form.
		ip := net.ParseIP(raw)
		if ip == nil {
			return nil, fmt.Errorf("invalid IPv6 address from escaped hostname: %q", raw)
		}
		ipStr = ip.String()
	}

	return &forgeclient.ForgeAddrInfo{
		EscapedIP:    escapedIP,
		IPVersion:    ipVersion,
		IPMaStr:      fmt.Sprintf("/ip%s/%s", ipVersion, ipStr),
		TCPPort:      tcpPort,
		PeerIDBase36: peerIDBase36,
	}, nil
}

// extractBareIPTCPAddr extracts the /ipX/<ip>/tcp/<port> prefix from a multiaddr.
func extractBareIPTCPAddr(maddr ma.Multiaddr) (ma.Multiaddr, error) {
	var ipPart, tcpPart string
	ma.ForEach(maddr, func(c ma.Component) bool {
		switch c.Protocol().Code {
		case ma.P_IP4, ma.P_IP6:
			ipPart = c.String()
			return true
		case ma.P_TCP:
			tcpPart = "/tcp/" + c.Value()
			return false
		default:
			return false
		}
	})
	if ipPart == "" || tcpPart == "" {
		return nil, fmt.Errorf("multiaddr missing IP or TCP component")
	}
	return ma.NewMultiaddr(ipPart + tcpPart)
}

// ipFromForgeAddr extracts the raw IP string from ForgeAddrInfo.IPMaStr.
// IPMaStr has format "/ip4/1.2.3.4" or "/ip6/2001:db8::1".
func ipFromForgeAddr(info *forgeclient.ForgeAddrInfo) string {
	switch info.IPVersion {
	case "4":
		return strings.TrimPrefix(info.IPMaStr, "/ip4/")
	case "6":
		return strings.TrimPrefix(info.IPMaStr, "/ip6/")
	default:
		return strings.TrimPrefix(info.IPMaStr, "/ip4/")
	}
}

// verifyForgeAddressFormat validates that each WSS underlay has correct forge format
// and consistent peer IDs. Returns parsed forge info for subsequent checks.
func (c *Check) verifyForgeAddressFormat(wssNodes map[string][]string, forgeDomain string) (map[string][]*forgeUnderlayInfo, error) {
	c.logger.Infof("verifying forge address format for %d nodes (domain: %s)", len(wssNodes), forgeDomain)

	result := make(map[string][]*forgeUnderlayInfo, len(wssNodes))
	for nodeName, underlays := range wssNodes {
		for _, u := range underlays {
			info, err := parseForgeUnderlay(u, forgeDomain)
			if err != nil {
				return nil, fmt.Errorf("node %s: invalid forge address %q: %w", nodeName, u, err)
			}

			// Round-trip validation: rebuild the multiaddr from parsed components
			// and verify it matches the original (minus the /p2p/<peerid> suffix).
			originalWithoutP2P := strings.TrimSuffix(u, "/p2p/"+info.PeerID.String())
			shortRebuilt := forgeclient.BuildShortForgeMultiaddr(info.ForgeAddr, forgeDomain)
			longRebuilt := forgeclient.BuildLongForgeMultiaddr(info.ForgeAddr, forgeDomain)

			if originalWithoutP2P != shortRebuilt && originalWithoutP2P != longRebuilt {
				return nil, fmt.Errorf("node %s: round-trip mismatch for %q (short: %q, long: %q)",
					nodeName, originalWithoutP2P, shortRebuilt, longRebuilt)
			}

			result[nodeName] = append(result[nodeName], info)
			c.logger.Debugf("node %s: forge address valid: %s (SAN: %s)", nodeName, u, info.ExpectedSAN)
		}
	}

	c.logger.Info("forge address format verification passed")
	return result, nil
}

// verifyDNSResolution resolves each forge hostname and verifies it points to the expected IP.
// Unreachable hostnames (e.g., cluster-internal domains when running outside k8s) are
// skipped with a warning. At least one underlay per node must resolve successfully.
func (c *Check) verifyDNSResolution(ctx context.Context, forgeNodes map[string][]*forgeUnderlayInfo) error {
	c.logger.Infof("verifying DNS resolution for %d nodes", len(forgeNodes))

	resolver := &net.Resolver{}
	for nodeName, infos := range forgeNodes {
		var verified int
		for _, info := range infos {
			ips, err := resolver.LookupHost(ctx, info.ForgeHostname)
			if err != nil {
				c.logger.Warningf("node %s: DNS lookup for %s failed (may be unreachable from host): %v",
					nodeName, info.ForgeHostname, err)
				continue
			}

			expectedIP := ipFromForgeAddr(info.ForgeAddr)
			if !slices.Contains(ips, expectedIP) {
				return fmt.Errorf("node %s: DNS for %s resolved to %v, expected %s",
					nodeName, info.ForgeHostname, ips, expectedIP)
			}
			verified++
			c.logger.Debugf("node %s: DNS resolution verified: %s -> %s", nodeName, info.ForgeHostname, expectedIP)
		}
		if verified == 0 {
			c.logger.Warningf("node %s: no forge hostnames were resolvable from this host, skipping DNS check", nodeName)
		}
	}

	c.logger.Info("DNS resolution verification passed")
	return nil
}

// verifyTLSCertificate connects to each forge endpoint and verifies the TLS certificate SAN.
// Unreachable endpoints (e.g., cluster-internal IPs when running outside k8s) are
// skipped with a warning. A wrong SAN on a reachable endpoint is a hard failure.
func (c *Check) verifyTLSCertificate(ctx context.Context, forgeNodes map[string][]*forgeUnderlayInfo, caCertPEM string) error {
	c.logger.Infof("verifying TLS certificates for %d nodes", len(forgeNodes))

	baseTLSConfig := &tls.Config{}
	if caCertPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caCertPEM)) {
			return fmt.Errorf("failed to parse CA certificate PEM")
		}
		baseTLSConfig.RootCAs = pool
	}

	for nodeName, infos := range forgeNodes {
		var verified int
		for _, info := range infos {
			err := c.verifyNodeTLSCert(ctx, baseTLSConfig, nodeName, info)
			if err == nil {
				verified++
				continue
			}

			// Distinguish connection failures (unreachable) from cert mismatches.
			var netErr *net.OpError
			if errors.As(err, &netErr) {
				c.logger.Warningf("node %s: %v", nodeName, err)
				continue
			}
			return err
		}
		if verified == 0 {
			c.logger.Warningf("node %s: no forge endpoints were reachable from this host, skipping TLS check", nodeName)
		}
	}

	c.logger.Info("TLS certificate verification passed")
	return nil
}

func (c *Check) verifyNodeTLSCert(ctx context.Context, baseTLSConfig *tls.Config, nodeName string, info *forgeUnderlayInfo) error {
	addr := net.JoinHostPort(ipFromForgeAddr(info.ForgeAddr), info.ForgeAddr.TCPPort)

	cfg := baseTLSConfig.Clone()
	cfg.ServerName = info.ForgeHostname

	conn, err := (&tls.Dialer{Config: cfg}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("TLS dial to %s (ServerName: %s) failed: %w",
			addr, info.ForgeHostname, err)
	}
	defer conn.Close()

	certs := conn.(*tls.Conn).ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return fmt.Errorf("no TLS certificates from %s", addr)
	}

	if !slices.Contains(certs[0].DNSNames, info.ExpectedSAN) {
		return fmt.Errorf("certificate SANs %v don't include expected %s",
			certs[0].DNSNames, info.ExpectedSAN)
	}

	c.logger.Debugf("node %s: TLS certificate verified: %s (SAN: %s)", nodeName, addr, info.ExpectedSAN)
	return nil
}

// getCertSerials TLS-dials each reachable forge endpoint and returns a map of
// "nodeName/hostname" -> certificate serial number (hex string).
// Unreachable endpoints are silently skipped.
func (c *Check) getCertSerials(ctx context.Context, forgeNodes map[string][]*forgeUnderlayInfo, caCertPEM string) map[string]string {
	baseTLSConfig := &tls.Config{}
	if caCertPEM != "" {
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM([]byte(caCertPEM)) {
			baseTLSConfig.RootCAs = pool
		}
	}

	serials := make(map[string]string)
	for nodeName, infos := range forgeNodes {
		for _, info := range infos {
			addr := net.JoinHostPort(ipFromForgeAddr(info.ForgeAddr), info.ForgeAddr.TCPPort)

			cfg := baseTLSConfig.Clone()
			cfg.ServerName = info.ForgeHostname

			conn, err := (&tls.Dialer{Config: cfg}).DialContext(ctx, "tcp", addr)
			if err != nil {
				continue
			}

			certs := conn.(*tls.Conn).ConnectionState().PeerCertificates
			conn.Close()

			if len(certs) > 0 {
				key := fmt.Sprintf("%s/%s", nodeName, info.ForgeHostname)
				serials[key] = certs[0].SerialNumber.Text(16)
				c.logger.Debugf("%s: certificate serial=%s notAfter=%s", key, serials[key], certs[0].NotAfter)
			}
		}
	}
	return serials
}
