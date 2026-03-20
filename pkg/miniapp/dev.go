package miniapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// validateLocalhostURL parses and validates that a URL targets localhost.
func validateLocalhostURL(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	host := u.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return nil, fmt.Errorf("only localhost targets are allowed, got %q", host)
	}
	return u, nil
}

// RegisterDevTarget registers a new dev server target. Only localhost targets are allowed.

// RegisterDevTarget registers a new dev server target. Only localhost targets are allowed.
func (h *Handler) RegisterDevTarget(name, target string) (string, error) {
	if _, err := validateLocalhostURL(target); err != nil {
		return "", err
	}

	h.devMu.Lock()
	defer h.devMu.Unlock()

	h.devNextID++
	id := strconv.Itoa(h.devNextID)

	h.devTargets[id] = &DevTarget{ID: id, Name: name, Target: target}
	h.notifyStateChanged()
	return id, nil
}

// UnregisterDevTarget removes a registered target. If it was active, the proxy is disabled.

// UnregisterDevTarget removes a registered target. If it was active, the proxy is disabled.
func (h *Handler) UnregisterDevTarget(id string) error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if _, ok := h.devTargets[id]; !ok {
		return fmt.Errorf("target %q not found", id)
	}
	delete(h.devTargets, id)

	if h.devActiveID == id {
		h.clearActiveDevTargetLocked()
	}
	h.notifyStateChanged()
	return nil
}

// ActivateDevTarget sets the reverse proxy to the registered target with the given ID.

// ActivateDevTarget sets the reverse proxy to the registered target with the given ID.
func (h *Handler) ActivateDevTarget(id string) error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	dt, ok := h.devTargets[id]
	if !ok {
		return fmt.Errorf("target %q not found", id)
	}

	u, proxy, err := buildDevProxy(dt)
	if err != nil {
		return err
	}

	h.setActiveDevTargetLocked(id, u, proxy)
	h.notifyStateChanged()
	return nil
}

// DeactivateDevTarget disables the reverse proxy without removing registrations.

// DeactivateDevTarget disables the reverse proxy without removing registrations.
func (h *Handler) DeactivateDevTarget() error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	h.clearActiveDevTargetLocked()
	h.notifyStateChanged()
	return nil
}

// GetDevTarget returns the current dev proxy target URL, or empty string if disabled.

// GetDevTarget returns the current dev proxy target URL, or empty string if disabled.
func (h *Handler) GetDevTarget() string {
	h.devMu.RLock()
	defer h.devMu.RUnlock()
	if h.devTarget == nil {
		return ""
	}
	return h.devTarget.String()
}

// ListDevTargets returns all registered dev targets.

// ListDevTargets returns all registered dev targets.
func (h *Handler) ListDevTargets() []DevTarget {
	h.devMu.RLock()
	defer h.devMu.RUnlock()
	return h.sortedDevTargetsLocked()
}

// devProxyScript is the JavaScript injected into HTML responses from the dev proxy.
// It rewrites fetch() and XMLHttpRequest.open() so that absolute paths like
// "/api/items" are prefixed with "/miniapp/dev", matching the reverse proxy mount.
// It also captures console.log/warn/error/info and forwards them to the server.

// devProxyScript is the JavaScript injected into HTML responses from the dev proxy.
// It rewrites fetch() and XMLHttpRequest.open() so that absolute paths like
// "/api/items" are prefixed with "/miniapp/dev", matching the reverse proxy mount.
// It also captures console.log/warn/error/info and forwards them to the server.
const devProxyScript = `<script data-dev-proxy>
(function(){
  var B='/miniapp/dev';
  function rw(u){
    if(typeof u==='string'&&u.startsWith('/')&&!u.startsWith('//')&&!u.startsWith(B))return B+u;
    return u;
  }
  var _f=window.fetch;
  window.fetch=function(r,i){
    if(typeof r==='string')r=rw(r);
    else if(r instanceof Request)r=new Request(rw(r.url),r);
    return _f.call(this,r,i);
  };
  var _o=XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open=function(m,u){
    arguments[1]=rw(u);
    return _o.apply(this,arguments);
  };
  // Console capture: batch POST to /miniapp/dev/console
  var _cl=console.log,_cw=console.warn,_ce=console.error,_ci=console.info;
  var _buf=[],_timer=null;
  function _flush(){
    _timer=null;
    if(!_buf.length)return;
    var batch=_buf.splice(0,20);
    var payload=JSON.stringify(batch);
    try{
      if(navigator.sendBeacon&&navigator.sendBeacon('/miniapp/dev/console',new Blob([payload],{type:'application/json'})))return;
    }catch(e){}
    try{fetch('/miniapp/dev/console',{method:'POST',headers:{'Content-Type':'application/json'},body:payload,keepalive:true});}catch(e){}
  }
  function _cap(level,args){
    var msg=Array.prototype.map.call(args,function(a){
      try{return typeof a==='object'?JSON.stringify(a):String(a);}catch(e){return String(a);}
    }).join(' ');
    if(msg.length>1024)msg=msg.substring(0,1024);
    _buf.push({level:level,message:msg,timestamp:new Date().toISOString()});
    if(_buf.length>=20){if(_timer){clearTimeout(_timer);_timer=null;}_flush();}
    else if(!_timer){_timer=setTimeout(_flush,500);}
  }
  console.log=function(){_cap('log',arguments);_cl.apply(console,arguments);};
  console.warn=function(){_cap('warn',arguments);_cw.apply(console,arguments);};
  console.error=function(){_cap('error',arguments);_ce.apply(console,arguments);};
  console.info=function(){_cap('info',arguments);_ci.apply(console,arguments);};
  window.onerror=function(m,s,l,c,e){_cap('error',[m,'at',s+':'+l+':'+c]);};
  window.onunhandledrejection=function(e){_cap('error',['Unhandled rejection:',e.reason]);};
})();
</script>`

// injectDevProxyScript inserts the dev proxy rewrite script into an HTML document.
// Insertion priority: before </head>, after <body...>, or prepend to document.

// injectDevProxyScript inserts the dev proxy rewrite script into an HTML document.
// Insertion priority: before </head>, after <body...>, or prepend to document.
func injectDevProxyScript(html []byte) []byte {
	script := []byte(devProxyScript)

	// Priority 1: before </head>
	if idx := bytes.Index(bytes.ToLower(html), []byte("</head>")); idx >= 0 {
		out := make([]byte, 0, len(html)+len(script))
		out = append(out, html[:idx]...)
		out = append(out, script...)
		out = append(out, html[idx:]...)
		return out
	}

	// Priority 2: after <body ...>
	lower := bytes.ToLower(html)
	if idx := bytes.Index(lower, []byte("<body")); idx >= 0 {
		// Find the closing '>' of the <body> tag
		closeIdx := bytes.IndexByte(lower[idx:], '>')
		if closeIdx >= 0 {
			insertAt := idx + closeIdx + 1
			out := make([]byte, 0, len(html)+len(script))
			out = append(out, html[:insertAt]...)
			out = append(out, script...)
			out = append(out, html[insertAt:]...)
			return out
		}
	}

	// Priority 3: prepend
	out := make([]byte, 0, len(html)+len(script))
	out = append(out, script...)
	out = append(out, html...)
	return out
}

// escapeHTMLString escapes HTML special characters in a string.

// escapeHTMLString escapes HTML special characters in a string.
func escapeHTMLString(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// RegisterRoutes registers Mini App routes on the given mux.

func (h *Handler) apiDev(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, h.devStatus())
	case http.MethodPost:
		var req struct {
			Action string `json:"action"`
			ID     string `json:"id"`
		}
		if err := decodeJSONBody(r, 4096, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		switch req.Action {
		case "activate":
			if req.ID == "" {
				writeJSONError(w, http.StatusBadRequest, "id is required")
				return
			}
			if err := h.ActivateDevTarget(req.ID); err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
		case "deactivate":
			if err := h.DeactivateDevTarget(); err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
		case "unregister":
			if req.ID == "" {
				writeJSONError(w, http.StatusBadRequest, "id is required")
				return
			}
			if err := h.UnregisterDevTarget(req.ID); err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
		default:
			writeJSONError(w, http.StatusBadRequest, "unknown action")
			return
		}
		writeJSON(w, h.devStatus())
	default:
		writeMethodNotAllowed(w)
	}
}

func (h *Handler) serveDevProxy(w http.ResponseWriter, r *http.Request) {
	h.devMu.RLock()
	proxy := h.devProxy
	h.devMu.RUnlock()

	if proxy == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "dev proxy not configured")
		return
	}

	// Strip /miniapp/dev prefix so /miniapp/dev/foo → /foo
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/miniapp/dev")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	proxy.ServeHTTP(w, r)
}

// extractUserFromInitData parses user.id from the initData query string.
// initData contains a "user" param with JSON like {"id":123456,...}.

func (h *Handler) devStatus() map[string]any {
	h.devMu.RLock()
	defer h.devMu.RUnlock()

	active := h.devTarget != nil
	target := ""
	if h.devTarget != nil {
		target = h.devTargets[h.devActiveID].Target // original URL before IPv6 rewrite
	}

	return map[string]any{
		"active":    active,
		"active_id": h.devActiveID,
		"target":    target,
		"targets":   h.sortedDevTargetsLocked(),
	}
}

// apiDevConsole receives console output from dev preview iframes.
func (h *Handler) apiDevConsole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	// Only accept console posts when dev proxy is active
	if h.GetDevTarget() == "" {
		writeJSONError(w, http.StatusNotFound, "not available")
		return
	}

	// Simple rate limit: max 10 requests per second
	now := time.Now().Unix()
	h.consoleMu.Lock()
	if h.consoleReqSec != now {
		h.consoleReqSec = now
		h.consoleReqCount = 0
	}
	h.consoleReqCount++
	over := h.consoleReqCount > 10
	h.consoleMu.Unlock()
	if over {
		writeJSONError(w, http.StatusTooManyRequests, "rate limit")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 32*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	var entries []struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Cap at 20 entries per batch
	if len(entries) > 20 {
		entries = entries[:20]
	}

	for _, e := range entries {
		msg := e.Message
		if len(msg) > 1024 {
			msg = msg[:1024]
		}
		switch e.Level {
		case "warn":
			logger.WarnC("dev-console", msg)
		case "error":
			logger.ErrorC("dev-console", msg)
		default:
			logger.InfoC("dev-console", msg)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// wsLogs serves a WebSocket endpoint that streams log entries in real time.
