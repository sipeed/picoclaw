package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
	"github.com/KarakuriAgent/clawdroid/pkg/logger"
)

// handleGetSchema returns the configuration schema.
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	schema := BuildSchema(config.DefaultConfig())
	writeJSON(w, http.StatusOK, schema)
}

// handleGetConfig returns the current configuration.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.cfg.RLock()
	data, err := json.Marshal(s.cfg)
	s.cfg.RUnlock()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to marshal config")
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to process config")
		return
	}

	writeJSON(w, http.StatusOK, raw)
}

// handlePutConfig updates the configuration.
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	s.cfg.RLock()
	currentData, err := json.Marshal(s.cfg)
	s.cfg.RUnlock()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to read current config")
		return
	}

	var incoming map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	mergedData, err := json.Marshal(incoming)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to prepare config")
		return
	}

	// Deep copy current config, then apply incoming partial update
	var newCfg config.Config
	if err := json.Unmarshal(currentData, &newCfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to copy config")
		return
	}
	if err := json.Unmarshal(mergedData, &newCfg); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}

	if err := config.SaveConfig(s.configPath, &newCfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"restart": true,
	})

	// Trigger restart after response is sent
	if s.onRestart != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.onRestart()
		}()
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.ErrorCF("gateway", "failed to encode JSON response", map[string]interface{}{"error": err.Error()})
	}
}
