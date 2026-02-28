package gateway

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
	"github.com/KarakuriAgent/clawdroid/pkg/logger"
)

type gatewaySetupRequest struct {
	Port   int    `json:"port"`
	APIKey string `json:"api_key"`
}

type setupInitRequest struct {
	Gateway gatewaySetupRequest `json:"gateway"`
}

// handleSetupInit creates config.json from scratch when it does not yet exist.
// POST /api/setup/init — no authentication required.
func (s *Server) handleSetupInit(w http.ResponseWriter, r *http.Request) {
	// Only allowed when config.json does not exist
	if _, err := os.Stat(s.configPath); err == nil {
		writeJSONError(w, http.StatusConflict, "config already exists")
		return
	}

	var req setupInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	cfg := config.DefaultConfig()

	if req.Gateway.Port > 0 {
		cfg.Gateway.Port = req.Gateway.Port
	}
	cfg.Gateway.APIKey = req.Gateway.APIKey

	if err := config.SaveConfig(s.configPath, cfg); err != nil {
		logger.ErrorCF("gateway", "Failed to save initial config", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	logger.InfoC("gateway", "Initial config created via setup wizard")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleSetupComplete merges additional settings into an existing config.json.
// PUT /api/setup/complete — authentication required.
func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	// Read current config from disk (includes init step's gateway settings)
	// rather than s.cfg which may still be DefaultConfig in setup mode.
	currentData, err := os.ReadFile(s.configPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to read config file")
		return
	}

	var incoming map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// The "agents_extra" key allows patching agents.defaults without overwriting
	// the entire agents block. Deep-merge it into the "agents" key so that
	// e.g. agents_extra.defaults.max_tokens merges alongside agents.defaults.workspace.
	if extra, ok := incoming["agents_extra"]; ok {
		agents, _ := incoming["agents"].(map[string]interface{})
		if agents == nil {
			agents = make(map[string]interface{})
		}
		if extraMap, ok := extra.(map[string]interface{}); ok {
			for k, v := range extraMap {
				// If both sides have a map for this key, merge them deeply
				existingMap, existingOk := agents[k].(map[string]interface{})
				newMap, newOk := v.(map[string]interface{})
				if existingOk && newOk {
					for nk, nv := range newMap {
						existingMap[nk] = nv
					}
				} else {
					agents[k] = v
				}
			}
		}
		incoming["agents"] = agents
		delete(incoming, "agents_extra")
	}

	mergedData, err := json.Marshal(incoming)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to prepare config")
		return
	}

	var newCfg config.Config
	if err := json.Unmarshal(currentData, &newCfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to copy config")
		return
	}
	if err := json.Unmarshal(mergedData, &newCfg); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}

	s.cfg.Lock()
	err = config.SaveConfigLocked(s.configPath, &newCfg)
	if err == nil {
		s.cfg.CopyFrom(&newCfg)
	}
	s.cfg.Unlock()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	logger.InfoC("gateway", "Setup wizard completed, config updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	if s.onRestart != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.onRestart()
		}()
	}
}
