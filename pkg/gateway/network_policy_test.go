package gateway

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeAndValidateCIDRs(t *testing.T) {
	got, err := normalizeAndValidateCIDRs([]string{" 192.168.1.20/24 ", "10.0.0.0/8", "192.168.1.0/24"})
	if err != nil {
		t.Fatalf("normalizeAndValidateCIDRs() error = %v", err)
	}
	want := []string{"10.0.0.0/8", "192.168.1.0/24"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAndValidateCIDRs() = %v, want %v", got, want)
	}
}

func TestNormalizeAndValidateCIDRsInvalid(t *testing.T) {
	_, err := normalizeAndValidateCIDRs([]string{"bad-cidr"})
	if err == nil {
		t.Fatal("normalizeAndValidateCIDRs() expected error for invalid CIDR")
	}
}

func TestResolveGatewayListenDecisionLoopbackFallbackUsesConfiguredCIDRs(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	discoverCalled := false

	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "127.0.0.1":
			return errors.New("loopback unavailable")
		case "::1", "localhost":
			return errors.New("alternative loopback unavailable")
		case gatewayFallbackBindHostV4, gatewayFallbackBindHostV6:
			return nil
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		discoverCalled = true
		return nil, errors.New("must not be called when configured CIDRs are provided")
	}

	decision, err := resolveGatewayListenDecision("127.0.0.1", 18790, []string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("resolveGatewayListenDecision() error = %v", err)
	}
	if !decision.AutoFallback {
		t.Fatal("decision.AutoFallback = false, want true")
	}
	if decision.BindHost != gatewayFallbackBindHostV4 {
		t.Fatalf("decision.BindHost = %q, want %q", decision.BindHost, gatewayFallbackBindHostV4)
	}
	wantCIDRs := []string{"10.0.0.0/8"}
	if !reflect.DeepEqual(decision.AllowedCIDRs, wantCIDRs) {
		t.Fatalf("decision.AllowedCIDRs = %v, want %v", decision.AllowedCIDRs, wantCIDRs)
	}
	if discoverCalled {
		t.Fatal("discoverGatewayCIDRs() called unexpectedly")
	}
}

func TestResolveGatewayListenDecisionLoopbackFallbackUsesDiscoveredCIDRs(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "localhost":
			return errors.New("loopback unavailable")
		case "127.0.0.1", "::1":
			return errors.New("alternative loopback unavailable")
		case gatewayFallbackBindHostV4, gatewayFallbackBindHostV6:
			return nil
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		return []string{"192.168.1.0/24", "10.0.0.0/8"}, nil
	}

	decision, err := resolveGatewayListenDecision("localhost", 18790, nil)
	if err != nil {
		t.Fatalf("resolveGatewayListenDecision() error = %v", err)
	}
	if !decision.AutoFallback {
		t.Fatal("decision.AutoFallback = false, want true")
	}
	wantCIDRs := []string{"192.168.1.0/24", "10.0.0.0/8"}
	if !reflect.DeepEqual(decision.AllowedCIDRs, wantCIDRs) {
		t.Fatalf("decision.AllowedCIDRs = %v, want %v", decision.AllowedCIDRs, wantCIDRs)
	}
}

func TestResolveGatewayListenDecisionNonLoopbackFailure(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	probeGatewayBind = func(host string, _ int) error {
		if host == "192.0.2.1" {
			return errors.New("cannot assign requested address")
		}
		return nil
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		return []string{"10.0.0.0/8"}, nil
	}

	_, err := resolveGatewayListenDecision("192.0.2.1", 18790, nil)
	if err == nil {
		t.Fatal("resolveGatewayListenDecision() expected error")
	}
	if !strings.Contains(err.Error(), "bind 192.0.2.1:18790 failed") {
		t.Fatalf("error = %q, want bind failure", err.Error())
	}
}

func TestResolveGatewayListenDecisionLoopbackFailureNoDiscoveredCIDRs(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "127.0.0.1":
			return errors.New("loopback unavailable")
		case "::1", "localhost":
			return errors.New("alternative loopback unavailable")
		case gatewayFallbackBindHostV4, gatewayFallbackBindHostV6:
			return nil
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		return nil, nil
	}

	_, err := resolveGatewayListenDecision("127.0.0.1", 18790, nil)
	if err == nil {
		t.Fatal("resolveGatewayListenDecision() expected error")
	}
	if !strings.Contains(err.Error(), "no private non-loopback interface CIDRs discovered") {
		t.Fatalf("error = %q, want no-interface-cidr failure", err.Error())
	}
}

func TestResolveGatewayListenDecisionLoopbackFallbackIPv6Preferred(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "::1":
			return errors.New("loopback unavailable")
		case "127.0.0.1", "localhost":
			return errors.New("alternative loopback unavailable")
		case gatewayFallbackBindHostV6:
			return nil
		case gatewayFallbackBindHostV4:
			return errors.New("should not probe IPv4 when IPv6 fallback succeeds")
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		return []string{"192.168.1.0/24"}, nil
	}

	decision, err := resolveGatewayListenDecision("::1", 18790, nil)
	if err != nil {
		t.Fatalf("resolveGatewayListenDecision() error = %v", err)
	}
	if decision.BindHost != gatewayFallbackBindHostV6 {
		t.Fatalf("decision.BindHost = %q, want %q", decision.BindHost, gatewayFallbackBindHostV6)
	}
}

func TestResolveGatewayListenDecisionLoopbackFallbackTriesSecondaryWildcard(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "::1":
			return errors.New("loopback unavailable")
		case "127.0.0.1", "localhost":
			return errors.New("alternative loopback unavailable")
		case gatewayFallbackBindHostV6:
			return errors.New("ipv6 wildcard unavailable")
		case gatewayFallbackBindHostV4:
			return nil
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		return []string{"192.168.1.0/24"}, nil
	}

	decision, err := resolveGatewayListenDecision("::1", 18790, nil)
	if err != nil {
		t.Fatalf("resolveGatewayListenDecision() error = %v", err)
	}
	if decision.BindHost != gatewayFallbackBindHostV4 {
		t.Fatalf("decision.BindHost = %q, want %q", decision.BindHost, gatewayFallbackBindHostV4)
	}
}

func TestResolveGatewayListenDecisionLoopbackUsesAlternativeBeforeWildcard(t *testing.T) {
	origProbe := probeGatewayBind
	origDiscover := discoverGatewayCIDRs
	t.Cleanup(func() {
		probeGatewayBind = origProbe
		discoverGatewayCIDRs = origDiscover
	})

	discoverCalled := false
	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "127.0.0.1":
			return errors.New("configured loopback unavailable")
		case "::1":
			return nil
		case "localhost", gatewayFallbackBindHostV4, gatewayFallbackBindHostV6:
			return errors.New("must not be probed after alternative loopback succeeds")
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		discoverCalled = true
		return nil, errors.New("must not be called when alternative loopback succeeds")
	}

	decision, err := resolveGatewayListenDecision("127.0.0.1", 18790, nil)
	if err != nil {
		t.Fatalf("resolveGatewayListenDecision() error = %v", err)
	}
	if decision.BindHost != "::1" {
		t.Fatalf("decision.BindHost = %q, want %q", decision.BindHost, "::1")
	}
	if len(decision.AllowedCIDRs) != 0 {
		t.Fatalf("decision.AllowedCIDRs = %v, want empty", decision.AllowedCIDRs)
	}
	if discoverCalled {
		t.Fatal("discoverGatewayCIDRs() called unexpectedly")
	}
}

func TestCIDRAllowlistMiddleware(t *testing.T) {
	mw := newCIDRAllowlistMiddleware([]string{"192.168.1.0/24"})
	h, err := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("middleware error = %v", err)
	}

	tests := []struct {
		name       string
		remoteAddr string
		wantStatus int
	}{
		{name: "inside cidr", remoteAddr: "192.168.1.99:1234", wantStatus: http.StatusOK},
		{name: "loopback", remoteAddr: "127.0.0.1:1234", wantStatus: http.StatusOK},
		{name: "outside cidr", remoteAddr: "10.0.0.7:1234", wantStatus: http.StatusForbidden},
		{name: "malformed", remoteAddr: "not-an-ip", wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			req.RemoteAddr = tt.remoteAddr
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestCIDRAllowlistMiddlewareInvalidCIDR(t *testing.T) {
	mw := newCIDRAllowlistMiddleware([]string{"bad-cidr"})
	_, err := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if err == nil {
		t.Fatal("middleware expected error for invalid CIDR")
	}
}

func TestNormalizeIPForMaskIPv4MappedIPv6(t *testing.T) {
	ip := net.ParseIP("::ffff:192.168.10.20")
	mask := net.CIDRMask(24, 32)

	normalized := normalizeIPForMask(ip, mask)
	if got := normalized.String(); got != "192.168.10.20" {
		t.Fatalf("normalizeIPForMask() = %q, want %q", got, "192.168.10.20")
	}

	masked := normalized.Mask(mask)
	if got := (&net.IPNet{IP: masked, Mask: mask}).String(); got != "192.168.10.0/24" {
		t.Fatalf("masked network = %q, want %q", got, "192.168.10.0/24")
	}
}

func TestFallbackBindCandidates(t *testing.T) {
	tests := []struct {
		name string
		host string
		want []string
	}{
		{
			name: "ipv4 loopback",
			host: "127.0.0.1",
			want: []string{gatewayFallbackBindHostV4, gatewayFallbackBindHostV6},
		},
		{name: "ipv6 loopback", host: "::1", want: []string{gatewayFallbackBindHostV6, gatewayFallbackBindHostV4}},
		{name: "localhost", host: "localhost", want: []string{gatewayFallbackBindHostV6, gatewayFallbackBindHostV4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fallbackBindCandidates(tt.host)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("fallbackBindCandidates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlternativeLoopbackHosts(t *testing.T) {
	tests := []struct {
		name string
		host string
		want []string
	}{
		{name: "configured ipv4", host: "127.0.0.1", want: []string{"::1", "localhost"}},
		{name: "configured ipv6", host: "::1", want: []string{"127.0.0.1", "localhost"}},
		{name: "configured localhost", host: "localhost", want: []string{"127.0.0.1", "::1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := alternativeLoopbackHosts(tt.host)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("alternativeLoopbackHosts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSafeFallbackIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "rfc1918 a", ip: "10.1.2.3", want: true},
		{name: "rfc1918 b", ip: "172.20.1.1", want: true},
		{name: "rfc1918 c", ip: "192.168.1.1", want: true},
		{name: "cgnat", ip: "100.64.2.3", want: true},
		{name: "ipv6 ula", ip: "fd12::1", want: true},
		{name: "public ipv4", ip: "8.8.8.8", want: false},
		{name: "public ipv6", ip: "2001:4860:4860::8888", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if got := isSafeFallbackIP(ip); got != tt.want {
				t.Fatalf("isSafeFallbackIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
