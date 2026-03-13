package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	oauthProviderOpenAI            = "openai"
	oauthProviderAnthropic         = "anthropic"
	oauthProviderGoogleAntigravity = "google-antigravity"

	oauthMethodBrowser    = "browser"
	oauthMethodDeviceCode = "device_code"
	oauthMethodToken      = "token"

	oauthFlowPending = "pending"
	oauthFlowSuccess = "success"
	oauthFlowError   = "error"
	oauthFlowExpired = "expired"
)

const (
	oauthBrowserFlowTTL    = 10 * time.Minute
	oauthDeviceCodeFlowTTL = 15 * time.Minute
	oauthTerminalFlowGC    = 30 * time.Minute
)

var oauthProviderOrder = []string{
	oauthProviderOpenAI,
	oauthProviderAnthropic,
	oauthProviderGoogleAntigravity,
}

var oauthProviderMethods = map[string][]string{
	oauthProviderOpenAI:            {oauthMethodBrowser, oauthMethodDeviceCode, oauthMethodToken},
	oauthProviderAnthropic:         {oauthMethodToken},
	oauthProviderGoogleAntigravity: {oauthMethodBrowser},
}

var oauthProviderLabels = map[string]string{
	oauthProviderOpenAI:            "OpenAI",
	oauthProviderAnthropic:         "Anthropic",
	oauthProviderGoogleAntigravity: "Google Antigravity",
}

var (
	oauthNow                      = time.Now
	oauthGeneratePKCE             = auth.GeneratePKCE
	oauthGenerateState            = auth.GenerateState
	oauthBuildAuthorizeURL        = auth.BuildAuthorizeURL
	oauthRequestDeviceCode        = auth.RequestDeviceCode
	oauthPollDeviceCodeOnce       = auth.PollDeviceCodeOnce
	oauthExchangeCodeForTokens    = auth.ExchangeCodeForTokens
	oauthGetCredential            = auth.GetCredential
	oauthSetCredential            = auth.SetCredential
	oauthDeleteCredential         = auth.DeleteCredential
	oauthLoadConfig               = config.LoadConfig
	oauthSaveConfig               = config.SaveConfig
	oauthFetchAntigravityProject  = providers.FetchAntigravityProjectID
	oauthFetchGoogleUserEmailFunc = fetchGoogleUserEmail
)

type oauthFlow struct {
	ID           string
	Provider     string
	Method       string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
	Error        string
	CodeVerifier string
	OAuthState   string
	RedirectURI  string
	DeviceAuthID string
	UserCode     string
	VerifyURL    string
	Interval     int
}

type oauthProviderStatus struct {
	Provider    string   `json:"provider"`
	DisplayName string   `json:"display_name"`
	Methods     []string `json:"methods"`
	LoggedIn    bool     `json:"logged_in"`
	Status      string   `json:"status"`
	AuthMethod  string   `json:"auth_method,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
	AccountID   string   `json:"account_id,omitempty"`
	Email       string   `json:"email,omitempty"`
	ProjectID   string   `json:"project_id,omitempty"`
}

type oauthFlowResponse struct {
	FlowID    string `json:"flow_id"`
	Provider  string `json:"provider"`
	Method    string `json:"method"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Error     string `json:"error,omitempty"`
	UserCode  string `json:"user_code,omitempty"`
	VerifyURL string `json:"verify_url,omitempty"`
	Interval  int    `json:"interval,omitempty"`
}

// registerOAuthRoutes binds OAuth login/logout endpoints to the ServeMux.
func (h *Handler) registerOAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/oauth/providers", h.handleListOAuthProviders)
	mux.HandleFunc("POST /api/oauth/login", h.handleOAuthLogin)
	mux.HandleFunc("GET /api/oauth/flows/{id}", h.handleGetOAuthFlow)
	mux.HandleFunc("POST /api/oauth/flows/{id}/poll", h.handlePollOAuthFlow)
	mux.HandleFunc("POST /api/oauth/logout", h.handleOAuthLogout)
	mux.HandleFunc("GET /oauth/callback", h.handleOAuthCallback)
}

func (h *Handler) handleListOAuthProviders(w http.ResponseWriter, r *http.Request) {
	providersResp := make([]oauthProviderStatus, 0, len(oauthProviderOrder))

	for _, provider := range oauthProviderOrder {
		cred, err := oauthGetCredential(provider)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load credentials: %v", err), http.StatusInternalServerError)
			return
		}

		item := oauthProviderStatus{
			Provider:    provider,
			DisplayName: oauthProviderLabels[provider],
			Methods:     oauthProviderMethods[provider],
			Status:      "not_logged_in",
		}
		if cred != nil {
			item.LoggedIn = true
			item.AuthMethod = cred.AuthMethod
			item.AccountID = cred.AccountID
			item.Email = cred.Email
			item.ProjectID = cred.ProjectID
			if !cred.ExpiresAt.IsZero() {
				item.ExpiresAt = cred.ExpiresAt.Format(time.RFC3339)
			}
			switch {
			case cred.IsExpired():
				item.Status = "expired"
			case cred.NeedsRefresh():
				item.Status = "needs_refresh"
			default:
				item.Status = "connected"
			}
		}

		providersResp = append(providersResp, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"providers": providersResp,
	})
}

func (h *Handler) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Provider string `json:"provider"`
		Method   string `json:"method"`
		Token    string `json:"token"`
	}
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	provider, err := normalizeOAuthProvider(req.Provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	method := strings.ToLower(strings.TrimSpace(req.Method))
	if !isOAuthMethodSupported(provider, method) {
		http.Error(
			w,
			fmt.Sprintf("unsupported login method %q for provider %q", method, provider),
			http.StatusBadRequest,
		)
		return
	}

	switch method {
	case oauthMethodToken:
		token := strings.TrimSpace(req.Token)
		if token == "" {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}

		cred := &auth.AuthCredential{
			AccessToken: token,
			Provider:    provider,
			AuthMethod:  oauthMethodToken,
		}
		if err := h.persistCredentialAndConfig(provider, oauthMethodToken, cred); err != nil {
			http.Error(w, fmt.Sprintf("token login failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"provider": provider,
			"method":   method,
		})
		return

	case oauthMethodDeviceCode:
		cfg := auth.OpenAIOAuthConfig()
		info, err := oauthRequestDeviceCode(cfg)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to request device code: %v", err), http.StatusInternalServerError)
			return
		}

		now := oauthNow()
		flow := &oauthFlow{
			ID:           newOAuthFlowID(),
			Provider:     provider,
			Method:       method,
			Status:       oauthFlowPending,
			CreatedAt:    now,
			UpdatedAt:    now,
			ExpiresAt:    now.Add(oauthDeviceCodeFlowTTL),
			DeviceAuthID: info.DeviceAuthID,
			UserCode:     info.UserCode,
			VerifyURL:    info.VerifyURL,
			Interval:     info.Interval,
		}
		h.storeOAuthFlow(flow)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "ok",
			"provider":   provider,
			"method":     method,
			"flow_id":    flow.ID,
			"user_code":  flow.UserCode,
			"verify_url": flow.VerifyURL,
			"interval":   flow.Interval,
			"expires_at": flow.ExpiresAt.Format(time.RFC3339),
		})
		return

	case oauthMethodBrowser:
		cfg, err := oauthConfigForProvider(provider)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		pkce, err := oauthGeneratePKCE()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate PKCE: %v", err), http.StatusInternalServerError)
			return
		}
		state, err := oauthGenerateState()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate state: %v", err), http.StatusInternalServerError)
			return
		}

		redirectURI := buildOAuthRedirectURI(r)
		authURL := oauthBuildAuthorizeURL(cfg, pkce, state, redirectURI)

		now := oauthNow()
		flow := &oauthFlow{
			ID:           newOAuthFlowID(),
			Provider:     provider,
			Method:       method,
			Status:       oauthFlowPending,
			CreatedAt:    now,
			UpdatedAt:    now,
			ExpiresAt:    now.Add(oauthBrowserFlowTTL),
			CodeVerifier: pkce.CodeVerifier,
			OAuthState:   state,
			RedirectURI:  redirectURI,
		}
		h.storeOAuthFlow(flow)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "ok",
			"provider":   provider,
			"method":     method,
			"flow_id":    flow.ID,
			"auth_url":   authURL,
			"expires_at": flow.ExpiresAt.Format(time.RFC3339),
		})
		return
	default:
		http.Error(w, "unsupported login method", http.StatusBadRequest)
	}
}

func (h *Handler) handleGetOAuthFlow(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.PathValue("id"))
	if flowID == "" {
		http.Error(w, "missing flow id", http.StatusBadRequest)
		return
	}

	flow, ok := h.getOAuthFlow(flowID)
	if !ok {
		http.Error(w, "flow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(flowToResponse(flow))
}

func (h *Handler) handlePollOAuthFlow(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.PathValue("id"))
	if flowID == "" {
		http.Error(w, "missing flow id", http.StatusBadRequest)
		return
	}

	flow, ok := h.getOAuthFlow(flowID)
	if !ok {
		http.Error(w, "flow not found", http.StatusNotFound)
		return
	}

	if flow.Method != oauthMethodDeviceCode {
		http.Error(w, "flow does not support polling", http.StatusBadRequest)
		return
	}
	if flow.Status != oauthFlowPending {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(flowToResponse(flow))
		return
	}

	cfg := auth.OpenAIOAuthConfig()
	cred, err := oauthPollDeviceCodeOnce(cfg, flow.DeviceAuthID, flow.UserCode)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "pending") {
			updated, _ := h.getOAuthFlow(flowID)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(flowToResponse(updated))
			return
		}
		h.setOAuthFlowError(flowID, fmt.Sprintf("device code poll failed: %v", err))
		updated, _ := h.getOAuthFlow(flowID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(flowToResponse(updated))
		return
	}
	if cred == nil {
		updated, _ := h.getOAuthFlow(flowID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(flowToResponse(updated))
		return
	}

	if err := h.persistCredentialAndConfig(flow.Provider, oauthMethodTokenOrOAuth(flow.Method), cred); err != nil {
		h.setOAuthFlowError(flowID, fmt.Sprintf("failed to save credential: %v", err))
		updated, _ := h.getOAuthFlow(flowID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(flowToResponse(updated))
		return
	}

	h.setOAuthFlowSuccess(flowID)
	updated, _ := h.getOAuthFlow(flowID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(flowToResponse(updated))
}

func (h *Handler) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		renderOAuthCallbackPage(w, "", oauthFlowError, "Missing state", "missing_state")
		return
	}

	flow, ok := h.getOAuthFlowByState(state)
	if !ok {
		renderOAuthCallbackPage(w, "", oauthFlowError, "OAuth flow not found", "flow_not_found")
		return
	}

	if flow.Status != oauthFlowPending {
		renderOAuthCallbackPage(w, flow.ID, flow.Status, "Flow already completed", flow.Error)
		return
	}

	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		if desc := strings.TrimSpace(r.URL.Query().Get("error_description")); desc != "" {
			errMsg += ": " + desc
		}
		h.setOAuthFlowError(flow.ID, errMsg)
		renderOAuthCallbackPage(w, flow.ID, oauthFlowError, "Authorization failed", errMsg)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		h.setOAuthFlowError(flow.ID, "missing authorization code")
		renderOAuthCallbackPage(w, flow.ID, oauthFlowError, "Missing authorization code", "missing_code")
		return
	}

	cfg, err := oauthConfigForProvider(flow.Provider)
	if err != nil {
		h.setOAuthFlowError(flow.ID, err.Error())
		renderOAuthCallbackPage(w, flow.ID, oauthFlowError, "Unsupported provider", err.Error())
		return
	}

	cred, err := oauthExchangeCodeForTokens(cfg, code, flow.CodeVerifier, flow.RedirectURI)
	if err != nil {
		h.setOAuthFlowError(flow.ID, fmt.Sprintf("token exchange failed: %v", err))
		renderOAuthCallbackPage(w, flow.ID, oauthFlowError, "Token exchange failed", err.Error())
		return
	}

	if err := h.persistCredentialAndConfig(flow.Provider, oauthMethodTokenOrOAuth(flow.Method), cred); err != nil {
		h.setOAuthFlowError(flow.ID, fmt.Sprintf("failed to save credential: %v", err))
		renderOAuthCallbackPage(w, flow.ID, oauthFlowError, "Failed to save credential", err.Error())
		return
	}

	h.setOAuthFlowSuccess(flow.ID)
	renderOAuthCallbackPage(w, flow.ID, oauthFlowSuccess, "Authentication successful", "")
}

func (h *Handler) handleOAuthLogout(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Provider string `json:"provider"`
	}
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	provider, err := normalizeOAuthProvider(req.Provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := oauthDeleteCredential(provider); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete credential: %v", err), http.StatusInternalServerError)
		return
	}
	if err := h.syncProviderAuthMethod(provider, ""); err != nil {
		http.Error(w, fmt.Sprintf("failed to update config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"provider": provider,
	})
}

func renderOAuthCallbackPage(w http.ResponseWriter, flowID, status, title, errMsg string) {
	payload := map[string]string{
		"type":   "picoclaw-oauth-result",
		"flowId": flowID,
		"status": status,
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	payloadJSON, _ := json.Marshal(payload)

	message := title
	if errMsg != "" {
		message = fmt.Sprintf("%s: %s", title, errMsg)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status == oauthFlowSuccess {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}

	_, _ = fmt.Fprintf(
		w,
		"<!doctype html><html><head><meta charset=\"utf-8\"><title>PicoClaw OAuth</title></head><body><script>(function(){var payload=%s;var hasOpener=false;try{if(window.opener&&!window.opener.closed){window.opener.postMessage(payload,window.location.origin);hasOpener=true}}catch(e){}var target='/credentials?oauth_flow_id='+encodeURIComponent(payload.flowId||'')+'&oauth_status='+encodeURIComponent(payload.status||'');setTimeout(function(){if(hasOpener){window.close();return}window.location.replace(target)},800)})();</script><div style=\"font-family:Inter,system-ui,sans-serif;padding:24px\"><h2>%s</h2><p>%s</p><p>You can close this window.</p></div></body></html>",
		string(payloadJSON),
		html.EscapeString(title),
		html.EscapeString(message),
	)
}

func normalizeOAuthProvider(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	switch provider {
	case "antigravity":
		return oauthProviderGoogleAntigravity, nil
	case oauthProviderOpenAI, oauthProviderAnthropic, oauthProviderGoogleAntigravity:
		return provider, nil
	default:
		return "", fmt.Errorf("unsupported provider %q", raw)
	}
}

func isOAuthMethodSupported(provider, method string) bool {
	methods := oauthProviderMethods[provider]
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

func oauthConfigForProvider(provider string) (auth.OAuthProviderConfig, error) {
	switch provider {
	case oauthProviderOpenAI:
		return auth.OpenAIOAuthConfig(), nil
	case oauthProviderGoogleAntigravity:
		return auth.GoogleAntigravityOAuthConfig(), nil
	default:
		return auth.OAuthProviderConfig{}, fmt.Errorf("provider %q does not support browser oauth", provider)
	}
}

func oauthMethodTokenOrOAuth(method string) string {
	if method == oauthMethodToken {
		return oauthMethodToken
	}
	return "oauth"
}

func buildOAuthRedirectURI(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	return fmt.Sprintf("%s://%s/oauth/callback", scheme, r.Host)
}

func flowToResponse(flow *oauthFlow) oauthFlowResponse {
	resp := oauthFlowResponse{
		FlowID:   flow.ID,
		Provider: flow.Provider,
		Method:   flow.Method,
		Status:   flow.Status,
		Error:    flow.Error,
	}
	if !flow.ExpiresAt.IsZero() {
		resp.ExpiresAt = flow.ExpiresAt.Format(time.RFC3339)
	}
	if flow.Method == oauthMethodDeviceCode {
		resp.UserCode = flow.UserCode
		resp.VerifyURL = flow.VerifyURL
		resp.Interval = flow.Interval
	}
	return resp
}

func newOAuthFlowID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("oauth_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (h *Handler) storeOAuthFlow(flow *oauthFlow) {
	now := oauthNow()
	h.oauthMu.Lock()
	defer h.oauthMu.Unlock()

	h.gcOAuthFlowsLocked(now)
	h.oauthFlows[flow.ID] = flow
	if flow.OAuthState != "" {
		h.oauthState[flow.OAuthState] = flow.ID
	}
}

func (h *Handler) getOAuthFlow(flowID string) (*oauthFlow, bool) {
	now := oauthNow()
	h.oauthMu.Lock()
	defer h.oauthMu.Unlock()

	h.gcOAuthFlowsLocked(now)
	flow, ok := h.oauthFlows[flowID]
	if !ok {
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (h *Handler) getOAuthFlowByState(state string) (*oauthFlow, bool) {
	now := oauthNow()
	h.oauthMu.Lock()
	defer h.oauthMu.Unlock()

	h.gcOAuthFlowsLocked(now)
	flowID, ok := h.oauthState[state]
	if !ok {
		return nil, false
	}
	flow, ok := h.oauthFlows[flowID]
	if !ok {
		delete(h.oauthState, state)
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (h *Handler) setOAuthFlowSuccess(flowID string) {
	now := oauthNow()
	h.oauthMu.Lock()
	defer h.oauthMu.Unlock()

	flow, ok := h.oauthFlows[flowID]
	if !ok {
		return
	}
	flow.Status = oauthFlowSuccess
	flow.Error = ""
	flow.UpdatedAt = now
	if flow.OAuthState != "" {
		delete(h.oauthState, flow.OAuthState)
	}
}

func (h *Handler) setOAuthFlowError(flowID, errMsg string) {
	now := oauthNow()
	h.oauthMu.Lock()
	defer h.oauthMu.Unlock()

	flow, ok := h.oauthFlows[flowID]
	if !ok {
		return
	}
	flow.Status = oauthFlowError
	flow.Error = errMsg
	flow.UpdatedAt = now
	if flow.OAuthState != "" {
		delete(h.oauthState, flow.OAuthState)
	}
}

func (h *Handler) gcOAuthFlowsLocked(now time.Time) {
	for id, flow := range h.oauthFlows {
		if flow.Status == oauthFlowPending && !flow.ExpiresAt.IsZero() && now.After(flow.ExpiresAt) {
			flow.Status = oauthFlowExpired
			flow.Error = "flow expired"
			flow.UpdatedAt = now
			if flow.OAuthState != "" {
				delete(h.oauthState, flow.OAuthState)
			}
		}

		if flow.Status != oauthFlowPending && now.Sub(flow.UpdatedAt) > oauthTerminalFlowGC {
			if flow.OAuthState != "" {
				delete(h.oauthState, flow.OAuthState)
			}
			delete(h.oauthFlows, id)
		}
	}
}

func (h *Handler) persistCredentialAndConfig(provider, authMethod string, cred *auth.AuthCredential) error {
	if cred == nil {
		return fmt.Errorf("empty credential")
	}

	cp := *cred
	cp.Provider = provider
	if cp.AuthMethod == "" {
		cp.AuthMethod = authMethod
	}

	if provider == oauthProviderGoogleAntigravity {
		if cp.Email == "" {
			email, err := oauthFetchGoogleUserEmailFunc(cp.AccessToken)
			if err != nil {
				log.Printf("oauth warning: could not fetch google email: %v", err)
			} else {
				cp.Email = email
			}
		}
		if cp.ProjectID == "" {
			projectID, err := oauthFetchAntigravityProject(cp.AccessToken)
			if err != nil {
				log.Printf("oauth warning: could not fetch antigravity project id: %v", err)
			} else {
				cp.ProjectID = projectID
			}
		}
	}

	if err := oauthSetCredential(provider, &cp); err != nil {
		return fmt.Errorf("saving credential: %w", err)
	}
	if err := h.syncProviderAuthMethod(provider, authMethod); err != nil {
		return fmt.Errorf("syncing provider auth config: %w", err)
	}
	return nil
}

func (h *Handler) syncProviderAuthMethod(provider, authMethod string) error {
	cfg, err := oauthLoadConfig(h.configPath)
	if err != nil {
		return err
	}

	switch provider {
	case oauthProviderOpenAI:
		cfg.Providers.OpenAI.AuthMethod = authMethod
	case oauthProviderAnthropic:
		cfg.Providers.Anthropic.AuthMethod = authMethod
	case oauthProviderGoogleAntigravity:
		cfg.Providers.Antigravity.AuthMethod = authMethod
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}

	found := false
	for i := range cfg.ModelList {
		if modelBelongsToProvider(provider, cfg.ModelList[i].Model) {
			cfg.ModelList[i].AuthMethod = authMethod
			found = true
		}
	}

	if !found && authMethod != "" {
		cfg.ModelList = append(cfg.ModelList, defaultModelConfigForProvider(provider, authMethod))
	}

	return oauthSaveConfig(h.configPath, cfg)
}

func modelBelongsToProvider(provider, model string) bool {
	lower := strings.ToLower(strings.TrimSpace(model))
	switch provider {
	case oauthProviderOpenAI:
		return lower == "openai" || strings.HasPrefix(lower, "openai/")
	case oauthProviderAnthropic:
		return lower == "anthropic" || strings.HasPrefix(lower, "anthropic/")
	case oauthProviderGoogleAntigravity:
		return lower == "antigravity" ||
			lower == "google-antigravity" ||
			strings.HasPrefix(lower, "antigravity/") ||
			strings.HasPrefix(lower, "google-antigravity/")
	default:
		return false
	}
}

func defaultModelConfigForProvider(provider, authMethod string) config.ModelConfig {
	switch provider {
	case oauthProviderOpenAI:
		return config.ModelConfig{
			ModelName:  "gpt-5.4",
			Model:      "openai/gpt-5.4",
			AuthMethod: authMethod,
		}
	case oauthProviderAnthropic:
		return config.ModelConfig{
			ModelName:  "claude-sonnet-4.6",
			Model:      "anthropic/claude-sonnet-4.6",
			AuthMethod: authMethod,
		}
	case oauthProviderGoogleAntigravity:
		return config.ModelConfig{
			ModelName:  "gemini-flash",
			Model:      "antigravity/gemini-3-flash",
			AuthMethod: authMethod,
		}
	default:
		return config.ModelConfig{}
	}
}

func fetchGoogleUserEmail(accessToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", err
	}
	if userInfo.Email == "" {
		return "", fmt.Errorf("empty email in userinfo response")
	}
	return userInfo.Email, nil
}
