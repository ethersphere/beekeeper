package autotls

import (
	"strings"
	"testing"

	forgeclient "github.com/ipshipyard/p2p-forge/client"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	testPeerID      = "Qmf5QjqDnnqw1CxErtnLpmyMNjpmug2o4G9yeo17HCXg2j"
	testBase36      = "k2k4r8pm5dousbyounf30w8nj0oeiol9x1ywlwspd3sxbw2omqjxiw8w"
	testForgeDomain = "local.test"
)

func TestParseForgeUnderlay_LongFormat(t *testing.T) {
	underlay := "/ip4/10.42.0.23/tcp/1635/tls/sni/10-42-0-23." + testBase36 + "." + testForgeDomain + "/ws/p2p/" + testPeerID

	info, err := parseForgeUnderlay(underlay, testForgeDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.PeerID.String() != testPeerID {
		t.Errorf("peer ID: got %s, want %s", info.PeerID, testPeerID)
	}
	if info.ForgeAddr.PeerIDBase36 != testBase36 {
		t.Errorf("base36: got %s, want %s", info.ForgeAddr.PeerIDBase36, testBase36)
	}
	if info.ForgeAddr.TCPPort != "1635" {
		t.Errorf("port: got %s, want 1635", info.ForgeAddr.TCPPort)
	}
	wantSAN := "*." + testBase36 + "." + testForgeDomain
	if info.ExpectedSAN != wantSAN {
		t.Errorf("SAN: got %s, want %s", info.ExpectedSAN, wantSAN)
	}
}

func TestParseForgeUnderlay_ShortFormat(t *testing.T) {
	hostname := "10-42-0-23." + testBase36 + "." + testForgeDomain
	underlay := "/dns4/" + hostname + "/tcp/1635/tls/ws/p2p/" + testPeerID

	info, err := parseForgeUnderlay(underlay, testForgeDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ForgeHostname != hostname {
		t.Errorf("hostname: got %s, want %s", info.ForgeHostname, hostname)
	}
	if info.ForgeAddr.IPVersion != "4" {
		t.Errorf("ip version: got %s, want 4", info.ForgeAddr.IPVersion)
	}
	if info.ForgeAddr.EscapedIP != "10-42-0-23" {
		t.Errorf("escaped IP: got %s, want 10-42-0-23", info.ForgeAddr.EscapedIP)
	}
}

func TestParseForgeUnderlay_IPv6Long(t *testing.T) {
	underlay := "/ip6/::1/tcp/1635/tls/sni/0--1." + testBase36 + "." + testForgeDomain + "/ws/p2p/" + testPeerID

	info, err := parseForgeUnderlay(underlay, testForgeDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ForgeAddr.IPVersion != "6" {
		t.Errorf("ip version: got %s, want 6", info.ForgeAddr.IPVersion)
	}
}

func TestParseForgeUnderlay_IPv6Short(t *testing.T) {
	hostname := "0--1." + testBase36 + "." + testForgeDomain
	underlay := "/dns6/" + hostname + "/tcp/1635/tls/ws/p2p/" + testPeerID

	info, err := parseForgeUnderlay(underlay, testForgeDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ForgeAddr.IPVersion != "6" {
		t.Errorf("ip version: got %s, want 6", info.ForgeAddr.IPVersion)
	}
	if info.ForgeHostname != hostname {
		t.Errorf("hostname: got %s, want %s", info.ForgeHostname, hostname)
	}
}

func TestParseForgeUnderlay_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		domain  string
		wantErr string
	}{
		{
			name:    "invalid multiaddr",
			input:   "not-a-multiaddr",
			domain:  testForgeDomain,
			wantErr: "parse multiaddr",
		},
		{
			name:    "missing p2p component",
			input:   "/ip4/1.2.3.4/tcp/1635/tls/ws",
			domain:  testForgeDomain,
			wantErr: "no /p2p/ component",
		},
		{
			name:    "no DNS, SNI, or IP",
			input:   "/tcp/1635/p2p/" + testPeerID,
			domain:  testForgeDomain,
			wantErr: "no DNS, SNI, or IP",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseForgeUnderlay(tt.input, tt.domain)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestForgeInfoFromHostname_IPv4(t *testing.T) {
	hostname := "192-168-1-100." + testBase36 + "." + testForgeDomain
	maddr, err := ma.NewMultiaddr("/dns4/" + hostname + "/tcp/443/tls/ws/p2p/" + testPeerID)
	if err != nil {
		t.Fatal(err)
	}

	info, err := forgeInfoFromHostname(hostname, testForgeDomain, "4", maddr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.EscapedIP != "192-168-1-100" {
		t.Errorf("escaped IP: got %s, want 192-168-1-100", info.EscapedIP)
	}
	if info.IPMaStr != "/ip4/192.168.1.100" {
		t.Errorf("IPMaStr: got %s, want /ip4/192.168.1.100", info.IPMaStr)
	}
	if info.TCPPort != "443" {
		t.Errorf("port: got %s, want 443", info.TCPPort)
	}
	if info.PeerIDBase36 != testBase36 {
		t.Errorf("base36: got %s, want %s", info.PeerIDBase36, testBase36)
	}
}

func TestForgeInfoFromHostname_IPv6(t *testing.T) {
	hostname := "0--1." + testBase36 + "." + testForgeDomain
	maddr, err := ma.NewMultiaddr("/dns6/" + hostname + "/tcp/1635/tls/ws/p2p/" + testPeerID)
	if err != nil {
		t.Fatal(err)
	}

	info, err := forgeInfoFromHostname(hostname, testForgeDomain, "6", maddr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.EscapedIP != "0--1" {
		t.Errorf("escaped IP: got %s, want 0--1", info.EscapedIP)
	}
	if info.IPMaStr != "/ip6/::1" {
		t.Errorf("IPMaStr: got %s, want /ip6/::1", info.IPMaStr)
	}
}

func TestForgeInfoFromHostname_WrongDomain(t *testing.T) {
	hostname := "1-2-3-4." + testBase36 + ".wrong.domain"
	maddr, err := ma.NewMultiaddr("/dns4/" + hostname + "/tcp/1635/tls/ws/p2p/" + testPeerID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = forgeInfoFromHostname(hostname, testForgeDomain, "4", maddr)
	if err == nil {
		t.Fatal("expected error for wrong domain")
	}
	if !strings.Contains(err.Error(), "doesn't end with domain") {
		t.Errorf("error %q should mention domain mismatch", err)
	}
}

func TestForgeInfoFromHostname_MissingPeerIDSegment(t *testing.T) {
	hostname := "nopeerid." + testForgeDomain
	maddr, err := ma.NewMultiaddr("/dns4/" + hostname + "/tcp/1635/tls/ws/p2p/" + testPeerID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = forgeInfoFromHostname(hostname, testForgeDomain, "4", maddr)
	if err == nil {
		t.Fatal("expected error for missing peer ID segment")
	}
	if !strings.Contains(err.Error(), "missing peer ID segment") {
		t.Errorf("error %q should mention missing peer ID", err)
	}
}

func TestExtractBareIPTCPAddr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "ipv4 with tcp",
			input: "/ip4/10.42.0.23/tcp/1635/tls/sni/example.com/ws",
			want:  "/ip4/10.42.0.23/tcp/1635",
		},
		{
			name:  "ipv6 with tcp",
			input: "/ip6/::1/tcp/1635/tls/sni/example.com/ws",
			want:  "/ip6/::1/tcp/1635",
		},
		{
			name:    "missing ip",
			input:   "/tcp/1635",
			wantErr: true,
		},
		{
			name:    "missing tcp",
			input:   "/ip4/1.2.3.4",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maddr, err := ma.NewMultiaddr(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := extractBareIPTCPAddr(maddr)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIpFromForgeAddr(t *testing.T) {
	tests := []struct {
		name string
		info *forgeclient.ForgeAddrInfo
		want string
	}{
		{
			name: "ipv4",
			info: &forgeclient.ForgeAddrInfo{IPVersion: "4", IPMaStr: "/ip4/10.42.0.23"},
			want: "10.42.0.23",
		},
		{
			name: "ipv6",
			info: &forgeclient.ForgeAddrInfo{IPVersion: "6", IPMaStr: "/ip6/::1"},
			want: "::1",
		},
		{
			name: "ipv6 full",
			info: &forgeclient.ForgeAddrInfo{IPVersion: "6", IPMaStr: "/ip6/2001:db8::1"},
			want: "2001:db8::1",
		},
		{
			name: "unknown version defaults to ip4 trim",
			info: &forgeclient.ForgeAddrInfo{IPVersion: "99", IPMaStr: "/ip4/1.2.3.4"},
			want: "1.2.3.4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipFromForgeAddr(tt.info)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
