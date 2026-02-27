package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
	"github.com/KarakuriAgent/clawdroid/pkg/logger"
)

// Server is the Gateway HTTP server that exposes the Config API.
type Server struct {
	cfg        *config.Config
	configPath string
	server     *http.Server
	onRestart  func()
}

// NewServer creates a new Gateway HTTP server.
func NewServer(cfg *config.Config, configPath string, onRestart func()) *Server {
	return &Server{
		cfg:        cfg,
		configPath: configPath,
		onRestart:  onRestart,
	}
}

// Start begins listening for HTTP requests on the configured host:port.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/config/schema", s.authMiddleware(s.handleGetSchema))
	mux.HandleFunc("GET /api/config", s.authMiddleware(s.handleGetConfig))
	mux.HandleFunc("PUT /api/config", s.authMiddleware(s.handlePutConfig))

	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Gateway.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logger.InfoCF("gateway", "HTTP server starting", map[string]interface{}{"addr": addr})
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("gateway", "HTTP server error", map[string]interface{}{"error": err.Error()})
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
