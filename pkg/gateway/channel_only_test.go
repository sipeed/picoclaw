package gateway

import "testing"

func TestResolveChannelOnlyListenHostRejectsNonLoopbackFallback(t *testing.T) {
	host, err := resolveChannelOnlyListenHost("256.256.256.256", 1)
	if err == nil {
		t.Fatal("expected error for invalid non-loopback host")
	}
	if host != "" {
		t.Fatalf("host = %q, want empty", host)
	}
}
