package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPAllowlist_EmptyCIDRsAllowsAll(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestIPAllowlist_RejectsOutsideCIDR(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs: []string{"192.168.1.0/24"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.RemoteAddr = "10.0.0.8:1234"
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestIPAllowlist_AllowsInsideCIDR(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs: []string{"192.168.1.0/24"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.88:1234"
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestIPAllowlist_AllowsLoopbackWhenBypassEnabled(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:         []string{"192.168.1.0/24"},
		AllowLocalhostBypass: true,
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestIPAllowlist_RejectsLoopbackWhenBypassDisabled(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs: []string{"192.168.1.0/24"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestIPAllowlist_IgnoresXForwardedForFromUntrustedPeer(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:      []string{"192.168.1.0/24"},
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.88")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestIPAllowlist_UsesRightmostUntrustedXForwardedForIPFromTrustedPeer(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:      []string{"192.168.1.0/24"},
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.8:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 192.168.1.88, 10.0.0.9")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestIPAllowlist_UsesRightmostUntrustedIPFromTrustedProxyChain(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:      []string{"192.168.1.0/24"},
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.8:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.88, 10.0.0.9, 10.0.0.10")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestIPAllowlist_DoesNotTrustSpoofedLeftmostLoopbackInXForwardedFor(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:         []string{"192.168.1.0/24"},
		AllowLocalhostBypass: true,
		TrustedProxyCIDRs:    []string{"10.0.0.0/8"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.8:1234"
	req.Header.Set("X-Forwarded-For", "127.0.0.1, 203.0.113.5")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestIPAllowlist_AllTrustedProxyChainFallsBackToLeftmostIP(t *testing.T) {
	h, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:      []string{"192.168.1.0/24"},
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("IPAllowlist() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.8:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.9, 10.0.0.10")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestIPAllowlist_InvalidCIDR(t *testing.T) {
	_, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs: []string{"bad-cidr"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if err == nil {
		t.Fatal("IPAllowlist() expected error for invalid CIDR")
	}
}

func TestIPAllowlist_InvalidTrustedProxyCIDR(t *testing.T) {
	_, err := IPAllowlist(IPAllowlistConfig{
		AllowedCIDRs:      []string{"192.168.1.0/24"},
		TrustedProxyCIDRs: []string{"bad-cidr"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if err == nil {
		t.Fatal("IPAllowlist() expected error for invalid trusted proxy CIDR")
	}
}
