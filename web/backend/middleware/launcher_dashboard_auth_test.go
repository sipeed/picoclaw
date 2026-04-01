package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionCookieValue_Deterministic(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	a := SessionCookieValue(key, "tok-a")
	b := SessionCookieValue(key, "tok-a")
	if a != b || a == "" {
		t.Fatalf("SessionCookieValue mismatch or empty: %q vs %q", a, b)
	}
	c := SessionCookieValue(key, "tok-b")
	if c == a {
		t.Fatal("SessionCookieValue should differ for different tokens")
	}
}

func TestLauncherDashboardAuth_AllowsPublicPaths(t *testing.T) {
	cfg := LauncherDashboardAuthConfig{ExpectedCookie: "deadbeef", Token: "x"}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := LauncherDashboardAuth(cfg, next)

	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/launcher-login", http.StatusTeapot},
		{http.MethodGet, "/assets/index.js", http.StatusTeapot},
		{http.MethodPost, "/api/auth/login", http.StatusTeapot},
		{http.MethodGet, "/api/auth/status", http.StatusTeapot},
		{http.MethodPost, "/api/auth/logout", http.StatusTeapot},
		{http.MethodGet, "/api/auth/logout", http.StatusUnauthorized},
		{http.MethodGet, "/api/config", http.StatusUnauthorized},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}

func TestLauncherDashboardAuth_URLTokenBootstrapGET(t *testing.T) {
	const tok = "secret"
	cfg := LauncherDashboardAuthConfig{ExpectedCookie: "deadbeef", Token: tok}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := LauncherDashboardAuth(cfg, next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?token="+tok, nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("GET /?token=valid: status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/" {
		t.Fatalf("Location = %q, want %q", got, "/")
	}
	if c := rec.Result().Cookies(); len(c) != 1 || c[0].Name != LauncherDashboardCookieName {
		t.Fatalf("expected one session cookie, got %#v", c)
	}

	rec1b := httptest.NewRecorder()
	req1b := httptest.NewRequest(http.MethodGet, "/config?token="+tok+"&keep=1", nil)
	h.ServeHTTP(rec1b, req1b)
	if rec1b.Code != http.StatusSeeOther {
		t.Fatalf("GET /config?token=valid: status = %d", rec1b.Code)
	}
	if got := rec1b.Header().Get("Location"); got != "/config?keep=1" {
		t.Fatalf("Location = %q, want /config?keep=1", got)
	}

	recBad := httptest.NewRecorder()
	reqBad := httptest.NewRequest(http.MethodGet, "/?token=wrong", nil)
	h.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusFound || recBad.Header().Get("Location") != "/launcher-login" {
		t.Fatalf("GET /?token=invalid: code=%d loc=%q", recBad.Code, recBad.Header().Get("Location"))
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/config?token="+tok, nil)
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api with token query: status = %d, want %d", rec2.Code, http.StatusUnauthorized)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/?token=", nil)
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusFound {
		t.Fatalf("GET /?token=empty: status = %d, want redirect", rec3.Code)
	}

	recLogin := httptest.NewRecorder()
	reqLogin := httptest.NewRequest(http.MethodGet, "/launcher-login?token="+tok, nil)
	h.ServeHTTP(recLogin, reqLogin)
	if recLogin.Code != http.StatusSeeOther || recLogin.Header().Get("Location") != "/" {
		t.Fatalf("GET /launcher-login?token=valid: code=%d loc=%q", recLogin.Code, recLogin.Header().Get("Location"))
	}
}

func TestLauncherDashboardAuth_DotDotCannotBypass(t *testing.T) {
	cfg := LauncherDashboardAuthConfig{ExpectedCookie: "deadbeef", Token: "x"}
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("next handler should not run without auth")
	})
	h := LauncherDashboardAuth(cfg, next)

	for _, p := range []string{
		"/assets/../api/config",
		"/launcher-login/../api/config",
		"/./api/config",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%q: status = %d, want %d", p, rec.Code, http.StatusUnauthorized)
		}
	}
}

func TestLauncherDashboardAuth_CookieAndBearer(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xab
	}
	token := "dashboard-secret-9"
	cookieVal := SessionCookieValue(key, token)
	cfg := LauncherDashboardAuthConfig{ExpectedCookie: cookieVal, Token: token}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := LauncherDashboardAuth(cfg, next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: LauncherDashboardCookieName, Value: cookieVal})
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cookie auth: status = %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("bearer auth: status = %d", rec2.Code)
	}
}
