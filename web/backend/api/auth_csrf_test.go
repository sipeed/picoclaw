package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLauncherAuthSetupRejectsCrossSiteFirstRun(t *testing.T) {
	store := &fakePasswordStore{}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		PasswordStore: store,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"http://127.0.0.1:18800/api/auth/setup",
		strings.NewReader(`{"password":"CrossSitePwn123!","confirm":"CrossSitePwn123!"}`),
	)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Referer", "https://evil.example/attack")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site setup code = %d body=%s", rec.Code, rec.Body.String())
	}
	if store.initialized || store.password != "" {
		t.Fatalf("cross-site setup mutated store: initialized=%v password=%q", store.initialized, store.password)
	}
}

func TestLauncherAuthSetupAllowsSameOriginFirstRun(t *testing.T) {
	store := &fakePasswordStore{}
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		SessionCookie: "session-cookie-value",
		PasswordStore: store,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"http://127.0.0.1:18800/api/auth/setup",
		strings.NewReader(`{"password":"LocalSetup123!","confirm":"LocalSetup123!"}`),
	)
	req.Header.Set("Origin", "http://127.0.0.1:18800")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("same-origin setup code = %d body=%s", rec.Code, rec.Body.String())
	}
	if !store.initialized || store.password != "LocalSetup123!" {
		t.Fatalf("same-origin setup store: initialized=%v password=%q", store.initialized, store.password)
	}
}
