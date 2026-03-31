package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

func TestLauncherAuthLoginAndStatus(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x55
	}
	const tok = "dashboard-test-token-9"
	sess := middleware.SessionCookieValue(key, tok)
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		DashboardToken: tok,
		SessionCookie:  sess,
		TokenHelp: LauncherAuthTokenHelp{
			EnvVarName:    "PICOCLAW_LAUNCHER_TOKEN",
			LogFileAbs:    "/tmp/launcher.log",
			TrayCopyMenu:  true,
			ConsoleStdout: false,
		},
	})

	t.Run("status_unauthenticated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/status", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d", rec.Code)
		}
		var body struct {
			Authenticated bool                   `json:"authenticated"`
			TokenHelp     *LauncherAuthTokenHelp `json:"token_help"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Authenticated || body.TokenHelp == nil {
			t.Fatalf("unexpected body: %+v", body)
		}
		if body.TokenHelp.EnvVarName != "PICOCLAW_LAUNCHER_TOKEN" || body.TokenHelp.LogFileAbs != "/tmp/launcher.log" {
			t.Fatalf("token_help = %+v", body.TokenHelp)
		}
	})

	t.Run("login_ok", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"token":"`+tok+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:12345"
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("login code = %d body=%s", rec.Code, rec.Body.String())
		}
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != middleware.LauncherDashboardCookieName {
			t.Fatalf("cookies = %#v", cookies)
		}
	})

	t.Run("status_authenticated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
		req.AddCookie(&http.Cookie{Name: middleware.LauncherDashboardCookieName, Value: sess})
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"authenticated":true`)) {
			t.Fatalf("body = %s", rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "token_help") {
			t.Fatalf("authenticated response should omit token_help: %s", rec.Body.String())
		}
	})
}

func TestLauncherAuthLogoutRequiresPostAndJSON(t *testing.T) {
	key := make([]byte, 32)
	sess := middleware.SessionCookieValue(key, "tok")
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		DashboardToken: "tok",
		SessionCookie:  sess,
		TokenHelp:      LauncherAuthTokenHelp{EnvVarName: "PICOCLAW_LAUNCHER_TOKEN"},
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil))
	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Fatalf("GET logout: code = %d (expected 404 or 405)", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("wrong content-type: code = %d body=%s", rec2.Code, rec2.Body.String())
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader(`{}`))
	req3.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("POST json logout: code = %d", rec3.Code)
	}
}

func TestLauncherAuthLoginRateLimit(t *testing.T) {
	key := make([]byte, 32)
	const tok = "rate-limit-tok-xxxxxxxx"
	sess := middleware.SessionCookieValue(key, tok)
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		DashboardToken: tok,
		SessionCookie:  sess,
		TokenHelp:      LauncherAuthTokenHelp{EnvVarName: "X"},
	})

	// 11 failing logins by wrong token; each consumes allow() slot after valid JSON.
	wrongBody := `{"token":"wrong"}`
	for i := 0; i < loginAttemptsPerIP; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(wrongBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.5.5:9999"
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("iter %d: want 401 got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(wrongBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.5.5:9999"
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("11th attempt: want 429 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestLoginRateLimiterWindow(t *testing.T) {
	l := newLoginRateLimiter()
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return t0 }
	for i := 0; i < loginAttemptsPerIP; i++ {
		if !l.allow("ip") {
			t.Fatalf("want allow at %d", i)
		}
	}
	if l.allow("ip") {
		t.Fatal("want deny on 11th")
	}
	l.now = func() time.Time { return t0.Add(loginAttemptWindow + time.Second) }
	if !l.allow("ip") {
		t.Fatal("want allow after window")
	}
}

func TestReferrerPolicyMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := middleware.ReferrerPolicyNoReferrer(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
}

func TestLauncherAuthLogoutEmptyBody(t *testing.T) {
	key := make([]byte, 32)
	sess := middleware.SessionCookieValue(key, "tok")
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		DashboardToken: "tok",
		SessionCookie:  sess,
		TokenHelp:      LauncherAuthTokenHelp{EnvVarName: "X"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = http.NoBody
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLauncherAuthLogoutRejectsTrailingJSON(t *testing.T) {
	key := make([]byte, 32)
	sess := middleware.SessionCookieValue(key, "tok")
	mux := http.NewServeMux()
	RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{
		DashboardToken: "tok",
		SessionCookie:  sess,
		TokenHelp:      LauncherAuthTokenHelp{EnvVarName: "X"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader(`{}{}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
}
