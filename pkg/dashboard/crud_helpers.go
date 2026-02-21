package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
)

var configMu sync.Mutex

func saveConfig(configPath string, cfg *config.Config) error {
	configMu.Lock()
	defer configMu.Unlock()
	return config.SaveConfig(configPath, cfg)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func hxRedirect(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Redirect", url)
	w.WriteHeader(http.StatusNoContent)
}
