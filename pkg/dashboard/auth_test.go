package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	pw := GeneratePassword()
	if len(pw) != 16 {
		t.Fatalf("expected 16 chars, got %d: %q", len(pw), pw)
	}
	for _, c := range pw {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			t.Fatalf("non-alphanumeric char in password: %c", c)
		}
	}
	// Two passwords should differ
	pw2 := GeneratePassword()
	if pw == pw2 {
		t.Error("two generated passwords should not be identical")
	}
}

func TestSignVerifySession(t *testing.T) {
	password := "testpassword123"
	cookie, _ := signSession(password)
	if !verifySession(cookie, password) {
		t.Error("valid session should verify")
	}
}

func TestVerifyExpiredSession(t *testing.T) {
	password := "testpassword123"
	// Sign normally then replace expiry with a past timestamp (0 = epoch)
	cookie, _ := signSession(password)
	parts := strings.SplitN(cookie, ".", 2)
	pastCookie := parts[0] + ".0"
	if verifySession(pastCookie, password) {
		t.Error("expired session should not verify")
	}
}

func TestVerifyTamperedSession(t *testing.T) {
	password := "testpassword123"
	cookie, _ := signSession(password)

	// Tamper with signature
	tampered := "deadbeef" + cookie[8:]
	if verifySession(tampered, password) {
		t.Error("tampered session should not verify")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	cookie, _ := signSession("correct")
	if verifySession(cookie, "wrong") {
		t.Error("session signed with different password should not verify")
	}
}

func TestVerifyInvalidFormats(t *testing.T) {
	tests := []string{
		"",
		"noseparator",
		"abc.",
		".abc",
		"abc.notahexnumber",
	}
	for _, cookie := range tests {
		if verifySession(cookie, "password") {
			t.Errorf("invalid cookie %q should not verify", cookie)
		}
	}
}

func TestAuthMiddlewareRedirect(t *testing.T) {
	called := false
	handler := authMiddleware("secret", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if called {
		t.Error("handler should not be called without auth cookie")
	}
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/dashboard/login" {
		t.Fatalf("expected redirect to /dashboard/login, got %q", loc)
	}
}

func TestAuthMiddlewareValid(t *testing.T) {
	password := "secret"
	called := false
	handler := authMiddleware(password, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	cookie, expiry := signSession(password)
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{
		Name:    cookieName,
		Value:   cookie,
		Expires: expiry,
	})
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("handler should be called with valid auth cookie")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestNoAuthWhenNoPassword(t *testing.T) {
	called := false
	handler := authMiddleware("", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("handler should be called when password is empty (no auth)")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLoginHandler(t *testing.T) {
	password := "mypassword"
	handler := loginHandler(password)

	form := url.Values{"password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %q", loc)
	}
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == cookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("login should set session cookie")
	}
}

func TestLoginHandlerWrong(t *testing.T) {
	password := "mypassword"
	handler := loginHandler(password)

	form := url.Values{"password": {"wrongpassword"}}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (re-serve login), got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Invalid password") {
		t.Error("wrong password should show error message")
	}
}

func TestLogoutHandler(t *testing.T) {
	handler := logoutHandler()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/logout", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/dashboard/login" {
		t.Fatalf("expected redirect to /dashboard/login, got %q", loc)
	}
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == cookieName && c.MaxAge != -1 {
			t.Error("logout should set cookie MaxAge to -1")
		}
	}
}
