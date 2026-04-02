package gateway

import (
	"errors"
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

	probeGatewayBind = func(host string, _ int) error {
		switch host {
		case "127.0.0.1":
			return errors.New("loopback unavailable")
		case gatewayFallbackBindHost:
			return nil
		default:
			return nil
		}
	}
	discoverGatewayCIDRs = func() ([]string, error) {
		return []string{"192.168.50.0/24"}, nil
	}

	decision, err := resolveGatewayListenDecision("127.0.0.1", 18790, []string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("resolveGatewayListenDecision() error = %v", err)
	}
	if !decision.AutoFallback {
		t.Fatal("decision.AutoFallback = false, want true")
	}
	if decision.BindHost != gatewayFallbackBindHost {
		t.Fatalf("decision.BindHost = %q, want %q", decision.BindHost, gatewayFallbackBindHost)
	}
	wantCIDRs := []string{"10.0.0.0/8"}
	if !reflect.DeepEqual(decision.AllowedCIDRs, wantCIDRs) {
		t.Fatalf("decision.AllowedCIDRs = %v, want %v", decision.AllowedCIDRs, wantCIDRs)
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
		case gatewayFallbackBindHost:
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
		case gatewayFallbackBindHost:
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
	if !strings.Contains(err.Error(), "no non-loopback interface CIDRs discovered") {
		t.Fatalf("error = %q, want no-interface-cidr failure", err.Error())
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
