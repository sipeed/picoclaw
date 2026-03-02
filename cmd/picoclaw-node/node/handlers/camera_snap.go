package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openclaw/go_node/config"
	"github.com/openclaw/go_node/gateway"
)

// CameraSnapHandler handles camera.snap command: mock by reading a fixed image file and returning base64
type CameraSnapHandler struct {
	cfg config.ExecConfig
}

// NewCameraSnapHandler creates a mock handler that reads from a fixed path
func NewCameraSnapHandler(cfg config.ExecConfig) *CameraSnapHandler {
	return &CameraSnapHandler{cfg: cfg}
}

// Handle reads the mock image from fixed path, base64-encodes, and returns camera.snap payload
func (h *CameraSnapHandler) Handle(req gateway.InvokeRequest) gateway.InvokeResult {
	path := h.mockPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "UNAVAILABLE",
				Message: fmt.Sprintf("camera.snap mock: read file %s: %v", path, err),
			},
		}
	}

	width, height := 0, 0
	if cfg, _, err := image.DecodeConfig(strings.NewReader(string(data))); err == nil {
		width, height = cfg.Width, cfg.Height
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "jpg"
	}
	if ext == "jpeg" {
		ext = "jpg"
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	log.Printf("go_node: camera.snap mock path=%s bytes=%d width=%d height=%d", path, len(data), width, height)

	out := map[string]any{
		"format": ext,
		"base64": b64,
		"width":  width,
		"height": height,
	}
	payload, _ := json.Marshal(out)
	return gateway.InvokeResult{
		OK:          true,
		PayloadJSON: string(payload),
	}
}

func (h *CameraSnapHandler) mockPath() string {
	if p := strings.TrimSpace(h.cfg.CameraMockPath); p != "" {
		return p
	}
	workDir := strings.TrimSpace(h.cfg.WorkDir)
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	if workDir == "" {
		return "camera_mock.jpg"
	}
	return filepath.Join(workDir, "camera_mock.jpg")
}
