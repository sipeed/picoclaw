package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rsc.io/qr"

	"github.com/sipeed/picoclaw/pkg/channels/weixin"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	weixinFlowTTL   = 5 * time.Minute
	weixinFlowGCAge = 30 * time.Minute
	weixinBaseURL   = "https://ilinkai.weixin.qq.com/"
	weixinBotType   = "3"
)

const (
	weixinStatusWait      = "wait"
	weixinStatusScanned   = "scaned"
	weixinStatusConfirmed = "confirmed"
	weixinStatusExpired   = "expired"
	weixinStatusError     = "error"
)

type weixinFlow struct {
	ID        string
	Qrcode    string // qrcode token from WeChat API (used for status polling)
	QRDataURI string // base64 PNG data URI for display
	AccountID string // IlinkBotID returned on confirmed
	Status    string // wait / scaned / confirmed / expired / error
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

type weixinFlowResponse struct {
	FlowID    string `json:"flow_id"`
	Status    string `json:"status"`
	QRDataURI string `json:"qr_data_uri,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// registerWeixinRoutes binds WeChat QR login endpoints to the ServeMux.
func (h *Handler) registerWeixinRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/weixin/flows", h.handleStartWeixinFlow)
	mux.HandleFunc("GET /api/weixin/flows/{id}", h.handlePollWeixinFlow)
}

// handleStartWeixinFlow starts a new WeChat QR login flow.
//
//	POST /api/weixin/flows
func (h *Handler) handleStartWeixinFlow(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	api, err := weixin.NewApiClient(weixinBaseURL, "", "")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create weixin client: %v", err), http.StatusInternalServerError)
		return
	}

	qrResp, err := api.GetQRCode(ctx, weixinBotType)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get QR code: %v", err), http.StatusInternalServerError)
		return
	}

	dataURI, err := generateQRDataURI(qrResp.QrcodeImgContent)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate QR image: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	flow := &weixinFlow{
		ID:        newWeixinFlowID(),
		Qrcode:    qrResp.Qrcode,
		QRDataURI: dataURI,
		Status:    weixinStatusWait,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(weixinFlowTTL),
	}
	h.storeWeixinFlow(flow)

	logger.InfoCF("weixin", "QR flow started", map[string]any{"flow_id": flow.ID})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(weixinFlowResponse{
		FlowID:    flow.ID,
		Status:    flow.Status,
		QRDataURI: flow.QRDataURI,
	})
}

// handlePollWeixinFlow polls the WeChat API for QR code status and updates the flow.
//
//	GET /api/weixin/flows/{id}
func (h *Handler) handlePollWeixinFlow(w http.ResponseWriter, r *http.Request) {
	flowID := strings.TrimSpace(r.PathValue("id"))
	if flowID == "" {
		http.Error(w, "missing flow id", http.StatusBadRequest)
		return
	}

	flow, ok := h.getWeixinFlow(flowID)
	if !ok {
		http.Error(w, "flow not found", http.StatusNotFound)
		return
	}

	// Return terminal states directly without polling WeChat again
	if flow.Status == weixinStatusConfirmed ||
		flow.Status == weixinStatusExpired ||
		flow.Status == weixinStatusError {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(weixinFlowResponse{
			FlowID: flow.ID,
			Status: flow.Status,
			Error:  flow.Error,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	api, err := weixin.NewApiClient(weixinBaseURL, "", "")
	if err != nil {
		h.setWeixinFlowError(flowID, fmt.Sprintf("client error: %v", err))
		flow, _ = h.getWeixinFlow(flowID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(weixinFlowResponse{FlowID: flow.ID, Status: flow.Status, Error: flow.Error})
		return
	}

	statusResp, err := api.GetQRCodeStatus(ctx, flow.Qrcode)
	if err != nil {
		// Transient error — keep current status, return it
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(weixinFlowResponse{
			FlowID:    flow.ID,
			Status:    flow.Status,
			QRDataURI: flow.QRDataURI,
		})
		return
	}

	switch statusResp.Status {
	case weixinStatusWait:
		// no change

	case weixinStatusScanned:
		h.updateWeixinFlowStatus(flowID, weixinStatusScanned)

	case weixinStatusConfirmed:
		if statusResp.BotToken == "" {
			h.setWeixinFlowError(flowID, "login confirmed but missing bot_token")
			break
		}
		if saveErr := h.saveWeixinToken(statusResp.BotToken, statusResp.IlinkBotID); saveErr != nil {
			h.setWeixinFlowError(flowID, fmt.Sprintf("failed to save token: %v", saveErr))
			logger.ErrorCF("weixin", "failed to save token", map[string]any{"error": saveErr.Error()})
			break
		}
		h.setWeixinFlowConfirmed(flowID, statusResp.IlinkBotID)
		logger.InfoCF("weixin", "QR login confirmed, token saved", map[string]any{
			"flow_id":    flowID,
			"account_id": statusResp.IlinkBotID,
		})

	case weixinStatusExpired:
		h.updateWeixinFlowStatus(flowID, weixinStatusExpired)

	default:
		// unknown status, keep as-is
	}

	flow, _ = h.getWeixinFlow(flowID)
	w.Header().Set("Content-Type", "application/json")
	resp := weixinFlowResponse{
		FlowID:    flow.ID,
		Status:    flow.Status,
		AccountID: flow.AccountID,
		Error:     flow.Error,
	}
	if flow.Status == weixinStatusWait || flow.Status == weixinStatusScanned {
		resp.QRDataURI = flow.QRDataURI
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// saveWeixinToken writes the token and account ID into the config file.
func (h *Handler) saveWeixinToken(token, accountID string) error {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg.Channels.Weixin.SetToken(token)
	if accountID != "" {
		cfg.Channels.Weixin.AccountID = accountID
	}
	return config.SaveConfig(h.configPath, cfg)
}

// generateQRDataURI encodes content as a QR code PNG and returns a data URI.
func generateQRDataURI(content string) (string, error) {
	code, err := qr.Encode(content, qr.L)
	if err != nil {
		return "", fmt.Errorf("qr encode: %w", err)
	}
	pngBytes := code.PNG()
	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	return "data:image/png;base64," + encoded, nil
}

func newWeixinFlowID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("wx_%d", time.Now().UnixNano())
	}
	return "wx_" + hex.EncodeToString(buf)
}

func (h *Handler) storeWeixinFlow(flow *weixinFlow) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	h.gcWeixinFlowsLocked(time.Now())
	h.weixinFlows[flow.ID] = flow
}

func (h *Handler) getWeixinFlow(flowID string) (*weixinFlow, bool) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	h.gcWeixinFlowsLocked(time.Now())
	flow, ok := h.weixinFlows[flowID]
	if !ok {
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (h *Handler) updateWeixinFlowStatus(flowID, status string) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	if flow, ok := h.weixinFlows[flowID]; ok {
		flow.Status = status
		flow.UpdatedAt = time.Now()
	}
}

func (h *Handler) setWeixinFlowConfirmed(flowID, accountID string) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	if flow, ok := h.weixinFlows[flowID]; ok {
		flow.Status = weixinStatusConfirmed
		flow.AccountID = accountID
		flow.UpdatedAt = time.Now()
	}
}

func (h *Handler) setWeixinFlowError(flowID, errMsg string) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	if flow, ok := h.weixinFlows[flowID]; ok {
		flow.Status = weixinStatusError
		flow.Error = errMsg
		flow.UpdatedAt = time.Now()
	}
}

func (h *Handler) gcWeixinFlowsLocked(now time.Time) {
	for id, flow := range h.weixinFlows {
		if flow.Status == weixinStatusWait || flow.Status == weixinStatusScanned {
			if !flow.ExpiresAt.IsZero() && now.After(flow.ExpiresAt) {
				flow.Status = weixinStatusExpired
				flow.UpdatedAt = now
			}
		}
		if flow.Status != weixinStatusWait &&
			flow.Status != weixinStatusScanned &&
			now.Sub(flow.UpdatedAt) > weixinFlowGCAge {
			delete(h.weixinFlows, id)
		}
	}
}
