package health

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

// Mux returns the underlying ServeMux so additional routes can be registered.
func (s *Server) Mux() *http.ServeMux {
	return s.server.Handler.(*http.ServeMux)
}

// StartTLS starts the server with TLS using the provided certificate and key files.
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS cert: %w", err)
	}
	s.server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return s.server.ListenAndServeTLS("", "")
}
