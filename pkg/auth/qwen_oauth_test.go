package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsQwenOAuthModel(t *testing.T) {
	tests := []struct {
		model  string
		expect bool
	}{
		{"qwen-oauth/coder-model", true},
		{"qwen-oauth/vision-model", true},
		{"qwen-oauth", true},
		{"qwen/coder-model", false},
		{"openai/gpt-4", false},
		{"", false},
		{"my-qwen-oauth-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := IsQwenOAuthModel(tt.model)
			if got != tt.expect {
				t.Errorf("IsQwenOAuthModel(%q) = %v, want %v", tt.model, got, tt.expect)
			}
		})
	}
}

func TestRequestQwenDeviceCode(t *testing.T) {
	expectedDeviceCode := "test-device-code-12345"
	expectedUserCode := "ABC-123"
	expectedVerificationURI := "https://chat.qwen.ai/verify"
	expectedVerificationURIComplete := "https://chat.qwen.ai/verify?code=ABC-123"
	expectedExpiresIn := 300
	expectedInterval := 5

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth2/device/code" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		// Verify PKCE challenge is present
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if r.FormValue("code_challenge") == "" {
			http.Error(w, "missing code_challenge", http.StatusBadRequest)
			return
		}
		if r.FormValue("code_challenge_method") != "S256" {
			http.Error(w, "invalid code_challenge_method", http.StatusBadRequest)
			return
		}

		resp := qwenDeviceAuthorization{
			DeviceCode:              expectedDeviceCode,
			UserCode:                expectedUserCode,
			VerificationURI:         expectedVerificationURI,
			VerificationURIComplete: expectedVerificationURIComplete,
			ExpiresIn:               expectedExpiresIn,
			Interval:                expectedInterval,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(server.URL+"/api/v1/oauth2/device/code", qwenOAuthBaseURL+"/api/v1/oauth2/token")

	_, challenge, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE() error: %v", err)
	}

	da, err := requestQwenDeviceCode(challenge)
	if err != nil {
		t.Fatalf("requestQwenDeviceCode() error: %v", err)
	}

	if da.DeviceCode != expectedDeviceCode {
		t.Errorf("DeviceCode = %q, want %q", da.DeviceCode, expectedDeviceCode)
	}

	if da.UserCode != expectedUserCode {
		t.Errorf("UserCode = %q, want %q", da.UserCode, expectedUserCode)
	}

	if da.VerificationURI != expectedVerificationURI {
		t.Errorf("VerificationURI = %q, want %q", da.VerificationURI, expectedVerificationURI)
	}

	if da.VerificationURIComplete != expectedVerificationURIComplete {
		t.Errorf("VerificationURIComplete = %q, want %q", da.VerificationURIComplete, expectedVerificationURIComplete)
	}

	if da.ExpiresIn != expectedExpiresIn {
		t.Errorf("ExpiresIn = %d, want %d", da.ExpiresIn, expectedExpiresIn)
	}

	if da.Interval != expectedInterval {
		t.Errorf("Interval = %d, want %d", da.Interval, expectedInterval)
	}
}

func TestRequestQwenDeviceCodeInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": "response"}`))
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(server.URL+"/api/v1/oauth2/device/code", qwenOAuthBaseURL+"/api/v1/oauth2/token")

	_, err := requestQwenDeviceCode("test-challenge")
	if err == nil {
		t.Error("expected error for invalid response")
	}

	if !strings.Contains(err.Error(), "missing device_code or user_code") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPollQwenTokenSuccess(t *testing.T) {
	expectedAccessToken := "test-access-token-xyz"
	expectedRefreshToken := "test-refresh-token-abc"
	expectedExpiresIn := 3600
	expectedTokenType := "Bearer"

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth2/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		callCount++

		// First two calls return pending, third returns success
		if callCount < 3 {
			resp := qwenTokenResponse{
				Error:            "authorization_pending",
				ErrorDescription: "User has not yet authorized",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Verify grant_type and code_verifier
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}
		if r.FormValue("code_verifier") == "" {
			http.Error(w, "missing code_verifier", http.StatusBadRequest)
			return
		}

		resp := qwenTokenResponse{
			AccessToken:  expectedAccessToken,
			RefreshToken: expectedRefreshToken,
			ExpiresIn:    expectedExpiresIn,
			TokenType:    expectedTokenType,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(qwenOAuthBaseURL+"/api/v1/oauth2/device/code", server.URL+"/api/v1/oauth2/token")

	// Use very short interval for testing
	tok, err := pollQwenToken("test-device-code", "test-verifier", 1, 30)
	if err != nil {
		t.Fatalf("pollQwenToken() error: %v", err)
	}

	if tok.AccessToken != expectedAccessToken {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, expectedAccessToken)
	}

	if tok.RefreshToken != expectedRefreshToken {
		t.Errorf("RefreshToken = %q, want %q", tok.RefreshToken, expectedRefreshToken)
	}

	if tok.ExpiresIn != expectedExpiresIn {
		t.Errorf("ExpiresIn = %d, want %d", tok.ExpiresIn, expectedExpiresIn)
	}

	if tok.TokenType != expectedTokenType {
		t.Errorf("TokenType = %q, want %q", tok.TokenType, expectedTokenType)
	}
}

func TestPollQwenTokenAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := qwenTokenResponse{
			Error:            "access_denied",
			ErrorDescription: "User denied the request",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(qwenOAuthBaseURL+"/api/v1/oauth2/device/code", server.URL+"/api/v1/oauth2/token")

	_, err := pollQwenToken("test-device-code", "test-verifier", 1, 30)
	if err == nil {
		t.Error("expected error for access_denied")
	}

	if !strings.Contains(err.Error(), "authorization denied") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPollQwenTokenExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := qwenTokenResponse{
			Error:            "expired_token",
			ErrorDescription: "Device code expired",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(qwenOAuthBaseURL+"/api/v1/oauth2/device/code", server.URL+"/api/v1/oauth2/token")

	_, err := pollQwenToken("test-device-code", "test-verifier", 1, 30)
	if err == nil {
		t.Error("expected error for expired_token")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPollQwenTokenSlowDown(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// First call returns slow_down
		if callCount == 1 {
			resp := qwenTokenResponse{
				Error:            "slow_down",
				ErrorDescription: "Please slow down",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Second call returns success
		resp := qwenTokenResponse{
			AccessToken: "test-token",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(qwenOAuthBaseURL+"/api/v1/oauth2/device/code", server.URL+"/api/v1/oauth2/token")

	tok, err := pollQwenToken("test-device-code", "test-verifier", 1, 30)
	if err != nil {
		t.Fatalf("pollQwenToken() error: %v", err)
	}

	if tok.AccessToken != "test-token" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "test-token")
	}
}

func TestRefreshQwenCredentialsSuccess(t *testing.T) {
	expectedAccessToken := "new-access-token"
	expectedRefreshToken := "new-refresh-token"
	expectedExpiresIn := 7200

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth2/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		if r.FormValue("grant_type") != "refresh_token" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		if r.FormValue("refresh_token") != "old-refresh-token" {
			http.Error(w, "invalid refresh_token", http.StatusBadRequest)
			return
		}

		resp := qwenTokenResponse{
			AccessToken:  expectedAccessToken,
			RefreshToken: expectedRefreshToken,
			ExpiresIn:    expectedExpiresIn,
			TokenType:    "Bearer",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(qwenOAuthBaseURL+"/api/v1/oauth2/device/code", server.URL+"/api/v1/oauth2/token")

	oldCred := &AuthCredential{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
		Provider:     "qwen",
		AuthMethod:   "oauth",
	}

	newCred, err := RefreshQwenCredentials(oldCred)
	if err != nil {
		t.Fatalf("RefreshQwenCredentials() error: %v", err)
	}

	if newCred.AccessToken != expectedAccessToken {
		t.Errorf("AccessToken = %q, want %q", newCred.AccessToken, expectedAccessToken)
	}

	if newCred.RefreshToken != expectedRefreshToken {
		t.Errorf("RefreshToken = %q, want %q", newCred.RefreshToken, expectedRefreshToken)
	}

	if newCred.Provider != "qwen" {
		t.Errorf("Provider = %q, want %q", newCred.Provider, "qwen")
	}
}

func TestRefreshQwenCredentialsNoRefreshToken(t *testing.T) {
	cred := &AuthCredential{
		AccessToken: "some-token",
		Provider:    "qwen",
		AuthMethod:  "oauth",
	}

	_, err := RefreshQwenCredentials(cred)
	if err == nil {
		t.Error("expected error for missing refresh token")
	}

	if !strings.Contains(err.Error(), "no refresh token available") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshQwenCredentialsExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer server.Close()

	// Set test endpoint
	SetQwenTestEndpoints(qwenOAuthBaseURL+"/api/v1/oauth2/device/code", server.URL+"/api/v1/oauth2/token")

	cred := &AuthCredential{
		AccessToken:  "old-token",
		RefreshToken: "expired-refresh-token",
		Provider:     "qwen",
		AuthMethod:   "oauth",
	}

	_, err := RefreshQwenCredentials(cred)
	if err == nil {
		t.Error("expected error for expired refresh token")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateQwenTokenSource(t *testing.T) {
	// This test verifies the token source closure works correctly
	// We can't test the full flow without mocking the auth store,
	// but we can verify the function returns a valid closure

	tokenSource := CreateQwenTokenSource()
	if tokenSource == nil {
		t.Fatal("CreateQwenTokenSource() returned nil")
	}

	// Note: The token source will return an error when not authenticated
	// This is expected behavior - we just verify the closure is created
	_, err := tokenSource()
	if err == nil {
		// This is actually OK - it means credentials might exist in the test environment
		// The important thing is that the closure was created successfully
		t.Log("Token source created successfully (credentials may exist in test env)")
	}
}

func TestQwenDeviceAuthorizationStruct(t *testing.T) {
	// Test that the struct can be properly unmarshaled
	jsonData := `{
		"device_code": "dc-12345",
		"user_code": "UC-ABC",
		"verification_uri": "https://example.com/verify",
		"verification_uri_complete": "https://example.com/verify?code=UC-ABC",
		"expires_in": 300,
		"interval": 5
	}`

	var da qwenDeviceAuthorization
	if err := json.Unmarshal([]byte(jsonData), &da); err != nil {
		t.Fatalf("unmarshal qwenDeviceAuthorization error: %v", err)
	}

	if da.DeviceCode != "dc-12345" {
		t.Errorf("DeviceCode = %q, want %q", da.DeviceCode, "dc-12345")
	}
	if da.UserCode != "UC-ABC" {
		t.Errorf("UserCode = %q, want %q", da.UserCode, "UC-ABC")
	}
	if da.ExpiresIn != 300 {
		t.Errorf("ExpiresIn = %d, want %d", da.ExpiresIn, 300)
	}
	if da.Interval != 5 {
		t.Errorf("Interval = %d, want %d", da.Interval, 5)
	}
}

func TestQwenTokenResponseStruct(t *testing.T) {
	// Test success response
	jsonData := `{
		"access_token": "at-12345",
		"refresh_token": "rt-67890",
		"expires_in": 3600,
		"token_type": "Bearer",
		"resource_url": "https://api.example.com"
	}`

	var tr qwenTokenResponse
	if err := json.Unmarshal([]byte(jsonData), &tr); err != nil {
		t.Fatalf("unmarshal qwenTokenResponse error: %v", err)
	}

	if tr.AccessToken != "at-12345" {
		t.Errorf("AccessToken = %q, want %q", tr.AccessToken, "at-12345")
	}
	if tr.RefreshToken != "rt-67890" {
		t.Errorf("RefreshToken = %q, want %q", tr.RefreshToken, "rt-67890")
	}
	if tr.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want %d", tr.ExpiresIn, 3600)
	}

	// Test error response
	errorJSON := `{
		"error": "authorization_pending",
		"error_description": "User has not yet authorized"
	}`

	var tr2 qwenTokenResponse
	if err := json.Unmarshal([]byte(errorJSON), &tr2); err != nil {
		t.Fatalf("unmarshal error response error: %v", err)
	}

	if tr2.Error != "authorization_pending" {
		t.Errorf("Error = %q, want %q", tr2.Error, "authorization_pending")
	}
}
