package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openclaw/go_node/config"
	"github.com/openclaw/go_node/gateway"
)

// FileSaveParams holds params for file.save (from picoclaw nodes tool)
type FileSaveParams struct {
	Base64   string `json:"base64"`
	Format   string `json:"format"`
	Filename string `json:"filename"`
}

// FileSaveHandler handles file.save command: save base64-encoded image to local disk
type FileSaveHandler struct {
	cfg config.ExecConfig
}

// NewFileSaveHandler creates a handler that saves files under exec.workDir/saved
func NewFileSaveHandler(cfg config.ExecConfig) *FileSaveHandler {
	return &FileSaveHandler{cfg: cfg}
}

// Handle decodes base64 image and saves to local directory
func (h *FileSaveHandler) Handle(req gateway.InvokeRequest) gateway.InvokeResult {
	var params FileSaveParams
	paramsJSON := strings.TrimSpace(req.ParamsJSON)
	if paramsJSON == "" {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "paramsJSON required",
			},
		}
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "invalid paramsJSON: " + err.Error(),
			},
		}
	}

	if params.Base64 == "" {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "base64 required",
			},
		}
	}

	data, err := base64.StdEncoding.DecodeString(params.Base64)
	if err != nil {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "base64 decode failed: " + err.Error(),
			},
		}
	}

	filename := strings.TrimSpace(params.Filename)
	if filename == "" {
		ext := strings.TrimSpace(strings.ToLower(params.Format))
		if ext == "" {
			ext = "bin"
		}
		filename = "file." + ext
	}

	// Sanitize filename: remove path traversal and dangerous chars
	filename = filepath.Base(filename)
	if filename == "" || filename == "." {
		filename = "file.bin"
	}

	saveDir := h.saveDir()
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "UNAVAILABLE",
				Message: "mkdir failed: " + err.Error(),
			},
		}
	}

	outPath := filepath.Join(saveDir, filename)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "UNAVAILABLE",
				Message: "write file failed: " + err.Error(),
			},
		}
	}

	absPath, _ := filepath.Abs(outPath)
	log.Printf("go_node: file.save path=%s bytes=%d", absPath, len(data))

	payload, _ := json.Marshal(map[string]any{
		"path":   absPath,
		"bytes":  len(data),
		"format": params.Format,
	})
	return gateway.InvokeResult{
		OK:          true,
		PayloadJSON: string(payload),
	}
}

func (h *FileSaveHandler) saveDir() string {
	workDir := strings.TrimSpace(h.cfg.WorkDir)
	if workDir == "" {
		return filepath.Join(os.TempDir(), "go_node_saved")
	}
	return filepath.Join(workDir, "saved")
}
