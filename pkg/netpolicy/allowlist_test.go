package netpolicy

import (
	"testing"
)

func TestIPAllowlistOpenPolicy(t *testing.T) {
	allowlist, err := NewIPAllowlist(nil)
	if err != nil {
		t.Fatalf("NewIPAllowlist() error = %v", err)
	}
	if !allowlist.IsOpen() {
		t.Fatal("allowlist should be open for empty CIDRs")
	}
	if !allowlist.AllowsRemoteAddr("203.0.113.7:1234") {
		t.Fatal("open policy should allow any remote address")
	}
}

func TestIPAllowlistAllowsInsideCIDR(t *testing.T) {
	allowlist, err := NewIPAllowlist([]string{"192.168.1.0/24"})
	if err != nil {
		t.Fatalf("NewIPAllowlist() error = %v", err)
	}

	if !allowlist.AllowsRemoteAddr("192.168.1.8:1234") {
		t.Fatal("allowlist should allow address inside CIDR")
	}
	if allowlist.AllowsRemoteAddr("10.0.0.8:1234") {
		t.Fatal("allowlist should reject address outside CIDR")
	}
}

func TestIPAllowlistAlwaysAllowsLoopback(t *testing.T) {
	allowlist, err := NewIPAllowlist([]string{"192.168.1.0/24"})
	if err != nil {
		t.Fatalf("NewIPAllowlist() error = %v", err)
	}

	if !allowlist.AllowsRemoteAddr("127.0.0.1:1234") {
		t.Fatal("loopback should always be allowed")
	}
}

func TestClientIPFromRemoteAddrIPv6Zone(t *testing.T) {
	ip := ClientIPFromRemoteAddr("[fe80::1%eth0]:1234")
	if ip == nil {
		t.Fatal("ClientIPFromRemoteAddr() returned nil")
	}
	if got := ip.String(); got != "fe80::1" {
		t.Fatalf("ClientIPFromRemoteAddr() = %q, want %q", got, "fe80::1")
	}
}

func TestNewIPAllowlistInvalidCIDR(t *testing.T) {
	_, err := NewIPAllowlist([]string{"bad-cidr"})
	if err == nil {
		t.Fatal("NewIPAllowlist() expected error for invalid CIDR")
	}
}

func TestIPAllowlistWithZoneAddressInCIDR(t *testing.T) {
	allowlist, err := NewIPAllowlist([]string{"fe80::/10"})
	if err != nil {
		t.Fatalf("NewIPAllowlist() error = %v", err)
	}

	if !allowlist.AllowsRemoteAddr("[fe80::2%eth0]:1234") {
		t.Fatal("allowlist should accept IPv6 link-local with zone")
	}
}

func TestNewIPAllowlistTrimsSkipsAndDedups(t *testing.T) {
	allowlist, err := NewIPAllowlist([]string{
		" 192.168.1.8/24 ",
		"",
		"192.168.1.0/24",
		"   ",
		"10.0.0.0/8",
	})
	if err != nil {
		t.Fatalf("NewIPAllowlist() error = %v", err)
	}
	if allowlist.IsOpen() {
		t.Fatal("allowlist should not be open")
	}
	if len(allowlist.nets) != 2 {
		t.Fatalf("len(allowlist.nets) = %d, want 2", len(allowlist.nets))
	}

	if !allowlist.AllowsRemoteAddr("192.168.1.22:1234") {
		t.Fatal("allowlist should allow deduplicated 192.168.1.0/24 CIDR")
	}
	if !allowlist.AllowsRemoteAddr("10.9.8.7:1234") {
		t.Fatal("allowlist should allow 10.0.0.0/8 CIDR")
	}
	if allowlist.AllowsRemoteAddr("203.0.113.7:1234") {
		t.Fatal("allowlist should reject outside CIDR")
	}
}

func TestNewIPAllowlistAllEmptyEntries(t *testing.T) {
	allowlist, err := NewIPAllowlist([]string{"", "   ", "\t"})
	if err != nil {
		t.Fatalf("NewIPAllowlist() error = %v", err)
	}
	if !allowlist.IsOpen() {
		t.Fatal("allowlist should be open when all entries are empty")
	}
}
