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

// handleGetConfig returns the current configuration with secrets masked.
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

	maskSecrets(raw)
	writeJSON(w, http.StatusOK, raw)
}

// handlePutConfig updates the configuration.
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	// Read current config as a map for secret preservation
	s.cfg.RLock()
	currentData, err := json.Marshal(s.cfg)
	s.cfg.RUnlock()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to read current config")
		return
	}

	var currentMap map[string]interface{}
	if err := json.Unmarshal(currentData, &currentMap); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to process current config")
		return
	}

	// Decode request body into a map to inspect raw values
	var incoming map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Restore masked secrets: replace "****" with current values
	restoreSecrets(incoming, currentMap)

	// Marshal merged map, then unmarshal onto a deep copy of current config (partial update)
	mergedData, err := json.Marshal(incoming)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to prepare config")
		return
	}

	// Deep copy current config via JSON round-trip to avoid shared map/slice references
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

// maskSecrets replaces non-empty secret values with "****".
func maskSecrets(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			maskSecrets(val)
		case string:
			if secretKeys[k] && val != "" {
				m[k] = "****"
			}
		}
	}
}

// restoreSecrets replaces "****" values in incoming with the corresponding current values.
func restoreSecrets(incoming, current map[string]interface{}) {
	for k, v := range incoming {
		switch val := v.(type) {
		case map[string]interface{}:
			if curSub, ok := current[k].(map[string]interface{}); ok {
				restoreSecrets(val, curSub)
			}
		case string:
			if secretKeys[k] && val == "****" {
				if curVal, ok := current[k]; ok {
					incoming[k] = curVal
				}
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.ErrorCF("gateway", "failed to encode JSON response", map[string]interface{}{"error": err.Error()})
	}
}
