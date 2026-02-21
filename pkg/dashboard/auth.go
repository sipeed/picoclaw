package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	cookieName    = "picoclaw_session"
	sessionMaxAge = 24 * time.Hour
	passwordLen   = 16
)

var alphanumeric = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// GeneratePassword returns a random 16-character alphanumeric password.
func GeneratePassword() string {
	b := make([]byte, passwordLen)
	randomBytes := make([]byte, passwordLen)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	for i := range b {
		b[i] = alphanumeric[int(randomBytes[i])%len(alphanumeric)]
	}
	return string(b)
}

func authMiddleware(password string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if password == "" {
			next(w, r)
			return
		}
		cookie, err := r.Cookie(cookieName)
		if err != nil || !verifySession(cookie.Value, password) {
			http.Redirect(w, r, "/dashboard/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func loginPage(password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if password == "" {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		serveLogin(w, "")
	}
}

func loginHandler(password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/dashboard/login", http.StatusFound)
			return
		}

		submitted := r.FormValue("password")
		if !hmac.Equal([]byte(submitted), []byte(password)) {
			serveLogin(w, "Invalid password")
			return
		}

		value, expiry := signSession(password)
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    value,
			Path:     "/dashboard",
			Expires:  expiry,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/dashboard",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.Redirect(w, r, "/dashboard/login", http.StatusFound)
	}
}

func signSession(password string) (string, time.Time) {
	expiry := time.Now().Add(sessionMaxAge)
	expiryHex := fmt.Sprintf("%x", expiry.Unix())
	mac := hmac.New(sha256.New, []byte(password))
	mac.Write([]byte(expiryHex))
	sig := hex.EncodeToString(mac.Sum(nil))
	return sig + "." + expiryHex, expiry
}

func verifySession(cookie, password string) bool {
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		return false
	}
	sig, expiryHex := parts[0], parts[1]

	var expiryUnix int64
	if _, err := fmt.Sscanf(expiryHex, "%x", &expiryUnix); err != nil {
		return false
	}
	if time.Now().Unix() > expiryUnix {
		return false
	}

	mac := hmac.New(sha256.New, []byte(password))
	mac.Write([]byte(expiryHex))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func serveLogin(w http.ResponseWriter, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginHTML, err := staticFiles.ReadFile("static/login.html")
	if err != nil {
		http.Error(w, "login.html not found", http.StatusInternalServerError)
		return
	}
	html := string(loginHTML)
	if errMsg != "" {
		html = strings.Replace(html, `<!--ERROR-->`, `<p class="error">`+errMsg+`</p>`, 1)
	}
	w.Write([]byte(html))
}
