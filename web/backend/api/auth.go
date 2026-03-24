package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/web/backend/auth"
)

type AuthHandler struct {
	configStore  *auth.AuthConfigStore
	sessionStore auth.SessionStore
}

func NewAuthHandler(configStore *auth.AuthConfigStore, sessionStore auth.SessionStore) *AuthHandler {
	return &AuthHandler{
		configStore:  configStore,
		sessionStore: sessionStore,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type StatusResponse struct {
	Enabled    bool `json:"enabled"`
	Configured bool `json:"configured"`
	LoggedIn   bool `json:"logged_in"`
}

type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SetupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/auth/status", h.Status)
	mux.HandleFunc("POST /api/auth/setup", h.Setup)
	mux.HandleFunc("POST /api/auth/change-password", h.ChangePassword)
	mux.HandleFunc("GET /api/auth/check", h.Check)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	config := h.configStore.Get()
	if !config.IsConfigured() {
		writeJSONError(w, "authentication not configured", http.StatusServiceUnavailable)
		return
	}

	if req.Username != config.Username || !auth.VerifyPassword(req.Password, config.PasswordHash) {
		writeJSON(w, LoginResponse{Success: false, Message: "invalid credentials"}, http.StatusUnauthorized)
		return
	}

	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		writeJSONError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	_, err = h.sessionStore.Create(sessionID, config.SessionTTL)
	if err != nil {
		writeJSONError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(config.SessionTTL),
	})

	writeJSON(w, LoginResponse{Success: true}, http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r)
	if sessionID != "" {
		h.sessionStore.Delete(sessionID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, LoginResponse{Success: true}, http.StatusOK)
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	config := h.configStore.Get()
	loggedIn := false

	if config.IsConfigured() {
		sessionID := extractSessionID(r)
		if sessionID != "" {
			loggedIn = h.sessionStore.Validate(sessionID)
		}
	}

	writeJSON(w, StatusResponse{
		Enabled:    config.Enabled,
		Configured: config.IsConfigured(),
		LoggedIn:   loggedIn,
	}, http.StatusOK)
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	config := h.configStore.Get()
	if config.IsConfigured() {
		writeJSONError(w, "authentication already configured", http.StatusBadRequest)
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSONError(w, "username and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		writeJSONError(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	if err := h.configStore.SetCredentials(req.Username, req.Password); err != nil {
		writeJSONError(w, "failed to save credentials", http.StatusInternalServerError)
		return
	}

	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		writeJSONError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	newConfig := h.configStore.Get()
	_, err = h.sessionStore.Create(sessionID, newConfig.SessionTTL)
	if err != nil {
		writeJSONError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(newConfig.SessionTTL),
	})

	writeJSON(w, SetupResponse{Success: true}, http.StatusOK)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	config := h.configStore.Get()
	if !config.IsConfigured() {
		writeJSONError(w, "authentication not configured", http.StatusServiceUnavailable)
		return
	}

	sessionID := extractSessionID(r)
	if sessionID == "" || !h.sessionStore.Validate(sessionID) {
		writeJSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !auth.VerifyPassword(req.CurrentPassword, config.PasswordHash) {
		writeJSONError(w, "current password is incorrect", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 6 {
		writeJSONError(w, "new password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	if err := h.configStore.SetCredentials(config.Username, req.NewPassword); err != nil {
		writeJSONError(w, "failed to update password", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]bool{"success": true}, http.StatusOK)
}

func (h *AuthHandler) Check(w http.ResponseWriter, r *http.Request) {
	config := h.configStore.Get()
	if !config.IsConfigured() {
		writeJSON(w, map[string]bool{"authenticated": true}, http.StatusOK)
		return
	}

	sessionID := extractSessionID(r)
	authenticated := sessionID != "" && h.sessionStore.Validate(sessionID)

	writeJSON(w, map[string]bool{"authenticated": authenticated}, http.StatusOK)
}

func extractSessionID(r *http.Request) string {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, map[string]string{"error": message}, status)
}
