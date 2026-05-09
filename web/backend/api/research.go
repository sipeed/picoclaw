package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func (h *Handler) registerResearchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/research/agents", h.handleListResearchAgents)
	mux.HandleFunc("PUT /api/research/agents/{id}/toggle", h.handleToggleResearchAgent)
	mux.HandleFunc("GET /api/research/graph", h.handleListResearchGraph)
	mux.HandleFunc("PUT /api/research/graph/nodes", h.handleUpdateResearchGraph)
	mux.HandleFunc("GET /api/research/reports", h.handleListResearchReports)
	mux.HandleFunc("PUT /api/research/reports", h.handleUpdateResearchReport)
	mux.HandleFunc("GET /api/research/config", h.handleGetResearchConfig)
	mux.HandleFunc("PUT /api/research/config", h.handleUpdateResearchConfig)
	mux.HandleFunc("GET /api/research/export", h.handleResearchExport)
	mux.HandleFunc("GET /ws/research", h.handleResearchWsProxy)
}

// researchHTTPProxy creates a reverse proxy to the gateway for research HTTP endpoints
func (h *Handler) researchHTTPProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			fmt.Printf("Failed to proxy research request to gateway: %v\n", err)
			// Return fallback data
			h.serveResearchFallback(w, r)
		},
	}
}

// serveResearchFallback returns fallback data when gateway is unavailable
func (h *Handler) serveResearchFallback(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.Contains(path, "/agents"):
		agents := []map[string]interface{}{
			{"id": "1", "name": "Literature Analyzer", "active": false, "progress": 0, "ram": "2GB", "type": "literature-analyzer"},
			{"id": "2", "name": "Data Extractor", "active": false, "progress": 0, "ram": "4GB", "type": "data-extractor"},
			{"id": "3", "name": "Fact Validator", "active": false, "progress": 0, "ram": "1GB", "type": "fact-validator"},
			{"id": "4", "name": "Synthesizer", "active": false, "progress": 0, "ram": "3GB", "type": "synthesizer"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"agents": agents})

	case strings.Contains(path, "/graph"):
		nodes := []map[string]interface{}{
			{"name": "Research Topic", "abbr": "RT", "x": 400.0, "y": 300.0},
			{"name": "Literature", "abbr": "Lit", "x": 200.0, "y": 150.0},
			{"name": "Data Sources", "abbr": "DS", "x": 600.0, "y": 150.0},
			{"name": "Analysis", "abbr": "An", "x": 400.0, "y": 450.0},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"nodes": nodes})

	case strings.Contains(path, "/reports"):
		reports := []map[string]interface{}{
			{"id": "1", "title": "Initial Research Report", "pages": 0, "words": 0, "status": "in-progress", "progress": 0},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"reports": reports})

	case strings.Contains(path, "/config"):
		config := map[string]interface{}{
			"type":             "comprehensive",
			"depth":            "deep",
			"restrict_to_graph": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleListResearchAgents proxies to gateway or returns fallback
func (h *Handler) handleListResearchAgents(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleToggleResearchAgent(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleListResearchGraph(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleUpdateResearchGraph(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleListResearchReports(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleUpdateResearchReport(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleGetResearchConfig(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleUpdateResearchConfig(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

func (h *Handler) handleResearchExport(w http.ResponseWriter, r *http.Request) {
	h.researchHTTPProxy().ServeHTTP(w, r)
}

// handleResearchWsProxy proxies WebSocket to gateway
func (h *Handler) handleResearchWsProxy(w http.ResponseWriter, r *http.Request) {
	gatewayURL := h.gatewayProxyURL()
	wsURL := &url.URL{
		Scheme: "ws",
		Host:   gatewayURL.Host,
		Path:   "/ws/research",
	}

	// Use the same pattern as pico WebSocket proxy
	wsProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(wsURL)
			pr.Out.Header.Del("Upgrade")
			pr.Out.Header.Del("Connection")
			pr.Out.Header.Set("Upgrade", "websocket")
			pr.Out.Header.Set("Connection", "upgrade")
		},
		ModifyResponse: func(r *http.Response) error {
			r.Header.Del("Upgrade")
			r.Header.Del("Connection")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Connection", "upgrade")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			fmt.Printf("Failed to proxy research WebSocket: %v\n", err)
			http.Error(w, "Gateway unavailable for WebSocket", http.StatusBadGateway)
		},
	}

	wsProxy.ServeHTTP(w, r)
}