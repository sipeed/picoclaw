package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	wecomFlowTTL             = 5 * time.Minute
	wecomFlowGCAge           = 30 * time.Minute
	wecomQRSourceID          = "picoclaw"
	wecomQRGenerateEndpoint  = "https://work.weixin.qq.com/ai/qc/generate"
	wecomQRQueryEndpoint     = "https://work.weixin.qq.com/ai/qc/query_result"
	wecomQRHTTPTimeout       = 15 * time.Second
	wecomDefaultWebSocketURL = "wss://openws.work.weixin.qq.com"
	wecomPollStartTimeout    = 15 * time.Second
	wecomPollStatusTimeout   = 10 * time.Second
)

const (
	wecomStatusWait      = "wait"
	wecomStatusScanned   = "scaned"
	wecomStatusConfirmed = "confirmed"
	wecomStatusExpired   = "expired"
	wecomStatusError     = "error"
)

type wecomFlow struct {
	ID        string
	SCode     string
	QRDataURI string
	BotID     string
	Status    string
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

type wecomFlowResponse struct {
	FlowID    string `json:"flow_id"`
	Status    string `json:"status"`
	QRDataURI string `json:"qr_data_uri,omitempty"`
	BotID     string `json:"bot_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type wecomQRGenerateResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    struct {
		SCode   string `json:"scode"`
		AuthURL string `json:"auth_url"`
	} `json:"data"`
}

type wecomQRQueryResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    struct {
		Status  string `json:"status"`
		BotInfo struct {
			BotID  string `json:"botid"`
			Secret string `json:"secret"`
		} `json:"bot_info"`
	} `json:"data"`
}

// registerWecomRoutes binds WeCom QR login endpoints to the ServeMux.
func (h *Handler) registerWecomRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/wecom/flows", h.handleStartWecomFlow)
	mux.HandleFunc("GET /api/wecom/flows/{id}", h.handlePollWecomFlow)
}

// handleStartWecomFlow starts a new WeCom QR login flow.
//
//	POST /api/wecom/flows
func (h *Handler) handleStartWecomFlow(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), wecomPollStartTimeout)
	defer cancel()

	session, err := fetchWecomQRCode(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get QR code: %v", err), http.StatusInternalServerError)
		return
	}

	dataURI, err := generateQRDataURI(session.Data.AuthURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate QR image: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	flow := &wecomFlow{
		ID:        newWecomFlowID(),
		SCode:     session.Data.SCode,
		QRDataURI: dataURI,
		Status:    wecomStatusWait,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(wecomFlowTTL),
	}
	h.storeWecomFlow(flow)

	logger.InfoCF("wecom", "QR flow started", map[string]any{"flow_id": flow.ID})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(wecomFlowResponse{
		FlowID:    flow.ID,
		Status:    flow.Status,
		QRDataURI: flow.QRDataURI,
	})
}

// handlePollWecomFlow polls the WeCom API for QR code status and updates the flow.
//
//	GET /api/wecom/flows/{id}
func (h *Handler) handlePollWecomFlow(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.PathValue("id"))
	if flowID == "" {
		http.Error(w, "missing flow id", http.StatusBadRequest)
		return
	}

	flow, ok := h.getWecomFlow(flowID)
	if !ok {
		http.Error(w, "flow not found", http.StatusNotFound)
		return
	}

	if flow.Status == wecomStatusConfirmed ||
		flow.Status == wecomStatusExpired ||
		flow.Status == wecomStatusError {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wecomFlowResponse{
			FlowID: flow.ID,
			Status: flow.Status,
			BotID:  flow.BotID,
			Error:  flow.Error,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), wecomPollStatusTimeout)
	defer cancel()

	statusResp, err := queryWecomQRCodeStatus(ctx, flow.SCode)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wecomFlowResponse{
			FlowID:    flow.ID,
			Status:    flow.Status,
			QRDataURI: flow.QRDataURI,
		})
		return
	}

	switch strings.ToLower(statusResp.Data.Status) {
	case wecomStatusWait:
		// no-op
	case wecomStatusScanned, "scanned":
		h.updateWecomFlowStatus(flowID, wecomStatusScanned)
	case "success":
		if statusResp.Data.BotInfo.BotID == "" || statusResp.Data.BotInfo.Secret == "" {
			h.setWecomFlowError(flowID, "login confirmed but missing bot credentials")
			break
		}
		if saveErr := h.saveWecomBinding(
			statusResp.Data.BotInfo.BotID,
			statusResp.Data.BotInfo.Secret,
		); saveErr != nil {
			h.setWecomFlowError(flowID, fmt.Sprintf("failed to save credentials: %v", saveErr))
			logger.ErrorCF("wecom", "failed to save credentials", map[string]any{"error": saveErr.Error()})
			break
		}
		h.setWecomFlowConfirmed(flowID, statusResp.Data.BotInfo.BotID)
		logger.InfoCF("wecom", "QR login confirmed, credentials saved", map[string]any{
			"flow_id": flowID,
			"bot_id":  statusResp.Data.BotInfo.BotID,
		})
	case wecomStatusExpired:
		h.updateWecomFlowStatus(flowID, wecomStatusExpired)
	}

	flow, _ = h.getWecomFlow(flowID)
	w.Header().Set("Content-Type", "application/json")
	resp := wecomFlowResponse{
		FlowID: flow.ID,
		Status: flow.Status,
		BotID:  flow.BotID,
		Error:  flow.Error,
	}
	if flow.Status == wecomStatusWait || flow.Status == wecomStatusScanned {
		resp.QRDataURI = flow.QRDataURI
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) saveWecomBinding(botID, secret string) error {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg.Channels.WeCom.Enabled = true
	cfg.Channels.WeCom.BotID = botID
	cfg.Channels.WeCom.SetSecret(secret)
	if strings.TrimSpace(cfg.Channels.WeCom.WebSocketURL) == "" {
		cfg.Channels.WeCom.WebSocketURL = wecomDefaultWebSocketURL
	}
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		return err
	}

	status := h.gatewayStatusData()
	gatewayStatus, _ := status["gateway_status"].(string)
	if gatewayStatus != "running" {
		return nil
	}

	if _, err := h.RestartGateway(); err != nil {
		logger.ErrorCF("wecom", "failed to restart gateway after saving binding", map[string]any{
			"error": err.Error(),
		})
	}
	return nil
}

func fetchWecomQRCode(ctx context.Context) (wecomQRGenerateResponse, error) {
	targetURL, err := buildWecomQRGenerateURL(wecomQRGenerateEndpoint, wecomQRSourceID, wecomPlatformCode())
	if err != nil {
		return wecomQRGenerateResponse{}, err
	}

	var resp wecomQRGenerateResponse
	if err := doWecomJSONGet(ctx, targetURL, &resp); err != nil {
		return wecomQRGenerateResponse{}, err
	}
	if resp.ErrCode != 0 {
		return wecomQRGenerateResponse{}, fmt.Errorf(
			"errcode=%d errmsg=%s",
			resp.ErrCode,
			resp.ErrMsg,
		)
	}
	if resp.Data.SCode == "" || resp.Data.AuthURL == "" {
		return wecomQRGenerateResponse{}, fmt.Errorf("response missing scode or auth_url")
	}
	return resp, nil
}

func queryWecomQRCodeStatus(ctx context.Context, scode string) (wecomQRQueryResponse, error) {
	targetURL, err := buildWecomQRQueryURL(wecomQRQueryEndpoint, scode)
	if err != nil {
		return wecomQRQueryResponse{}, err
	}

	var resp wecomQRQueryResponse
	if err := doWecomJSONGet(ctx, targetURL, &resp); err != nil {
		return wecomQRQueryResponse{}, err
	}
	if resp.ErrCode != 0 {
		return wecomQRQueryResponse{}, fmt.Errorf(
			"errcode=%d errmsg=%s",
			resp.ErrCode,
			resp.ErrMsg,
		)
	}
	return resp, nil
}

func buildWecomQRGenerateURL(baseURL, sourceID string, platformCode int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid WeCom QR generate URL: %w", err)
	}

	query := u.Query()
	query.Set("source", sourceID)
	query.Set("sourceID", sourceID)
	query.Set("plat", strconv.Itoa(platformCode))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func buildWecomQRQueryURL(baseURL, scode string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid WeCom QR query URL: %w", err)
	}

	query := u.Query()
	query.Set("scode", scode)
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func doWecomJSONGet(ctx context.Context, targetURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: wecomQRHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8192))
		if readErr != nil {
			return fmt.Errorf("unexpected status %s", resp.Status)
		}
		return fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	return nil
}

func wecomPlatformCode() int {
	switch runtime.GOOS {
	case "darwin":
		return 1
	case "windows":
		return 2
	case "linux":
		return 3
	default:
		return 0
	}
}

func newWecomFlowID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("wc_%d", time.Now().UnixNano())
	}
	return "wc_" + hex.EncodeToString(buf)
}

func (h *Handler) storeWecomFlow(flow *wecomFlow) {
	h.wecomMu.Lock()
	defer h.wecomMu.Unlock()
	h.gcWecomFlowsLocked(time.Now())
	h.wecomFlows[flow.ID] = flow
}

func (h *Handler) getWecomFlow(flowID string) (*wecomFlow, bool) {
	h.wecomMu.Lock()
	defer h.wecomMu.Unlock()
	h.gcWecomFlowsLocked(time.Now())
	flow, ok := h.wecomFlows[flowID]
	if !ok {
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (h *Handler) updateWecomFlowStatus(flowID, status string) {
	h.wecomMu.Lock()
	defer h.wecomMu.Unlock()
	if flow, ok := h.wecomFlows[flowID]; ok {
		flow.Status = status
		flow.UpdatedAt = time.Now()
	}
}

func (h *Handler) setWecomFlowConfirmed(flowID, botID string) {
	h.wecomMu.Lock()
	defer h.wecomMu.Unlock()
	if flow, ok := h.wecomFlows[flowID]; ok {
		flow.Status = wecomStatusConfirmed
		flow.BotID = botID
		flow.UpdatedAt = time.Now()
	}
}

func (h *Handler) setWecomFlowError(flowID, errMsg string) {
	h.wecomMu.Lock()
	defer h.wecomMu.Unlock()
	if flow, ok := h.wecomFlows[flowID]; ok {
		flow.Status = wecomStatusError
		flow.Error = errMsg
		flow.UpdatedAt = time.Now()
	}
}

func (h *Handler) gcWecomFlowsLocked(now time.Time) {
	for id, flow := range h.wecomFlows {
		if flow.Status == wecomStatusWait || flow.Status == wecomStatusScanned {
			if !flow.ExpiresAt.IsZero() && now.After(flow.ExpiresAt) {
				flow.Status = wecomStatusExpired
				flow.UpdatedAt = now
			}
		}
		if flow.Status != wecomStatusWait &&
			flow.Status != wecomStatusScanned &&
			now.Sub(flow.UpdatedAt) > wecomFlowGCAge {
			delete(h.wecomFlows, id)
		}
	}
}
