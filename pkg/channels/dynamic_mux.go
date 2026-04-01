package channels

import (
	"net/http"
	"strings"
	"sync"
)

// dynamicServeMux is an http.Handler that supports dynamic registration
// and unregistration of handlers without recreating the server.
type dynamicServeMux struct {
	mu       sync.RWMutex
	handlers map[string]http.Handler
}

func newDynamicServeMux() *dynamicServeMux {
	return &dynamicServeMux{
		handlers: make(map[string]http.Handler),
	}
}

// Handle registers the handler for the given pattern.
func (dm *dynamicServeMux) Handle(pattern string, handler http.Handler) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.handlers[pattern] = handler
}

// HandleFunc registers the handler function for the given pattern.
func (dm *dynamicServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	dm.Handle(pattern, http.HandlerFunc(handler))
}

// Unhandle removes the handler for the given pattern.
func (dm *dynamicServeMux) Unhandle(pattern string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	delete(dm.handlers, pattern)
}

// ServeHTTP dispatches the request to the handler whose pattern best matches
// the request URL path. It supports both exact path matches and subtree
// (trailing-slash) prefix matches, choosing the longest prefix on collision.
func (dm *dynamicServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	path := r.URL.Path

	// Exact match first.
	if h, ok := dm.handlers[path]; ok {
		h.ServeHTTP(w, r)
		return
	}

	// Longest subtree prefix match (patterns ending with "/").
	var bestLen int
	var bestHandler http.Handler
	for pattern, handler := range dm.handlers {
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) {
			if len(pattern) > bestLen {
				bestLen = len(pattern)
				bestHandler = handler
			}
		}
	}

	if bestHandler != nil {
		bestHandler.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}
