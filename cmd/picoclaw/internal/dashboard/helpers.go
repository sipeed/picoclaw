package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func runDashboard(host string, port int, openBrowser bool) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	url := fmt.Sprintf("http://%s", addr)
	if host == "0.0.0.0" {
		url = fmt.Sprintf("http://localhost:%d", port)
	}

	mux := http.NewServeMux()

	// API Handlers
	mux.HandleFunc("/api/config", configHandler)

	// Static Assets
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := embeddedFiles.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Printf("🚀 PicoClaw Dashboard starting on %s\n", url)

	if openBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openInBrowser(url)
		}()
	}

	return server.ListenAndServe()
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	configPath := internal.GetConfigPath()

	switch r.Method {
	case http.MethodGet:
		cfg, err := internal.LoadConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)

	case http.MethodPost:
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := config.SaveConfig(configPath, &cfg); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func openInBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		logger.WarnCF("dashboard", "Failed to open browser", map[string]any{"error": err.Error(), "url": url})
	}
}
