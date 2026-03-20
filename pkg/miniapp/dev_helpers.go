package miniapp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func (h *Handler) notifyStateChanged() {
	if h.notifier != nil {
		h.notifier.Notify()
	}
}

func (h *Handler) clearActiveDevTargetLocked() {
	h.devActiveID = ""
	h.devTarget = nil
	h.devProxy = nil
}

func (h *Handler) setActiveDevTargetLocked(id string, target *url.URL, proxy *httputil.ReverseProxy) {
	h.devActiveID = id
	h.devTarget = target
	h.devProxy = proxy
}

func (h *Handler) sortedDevTargetsLocked() []DevTarget {
	targets := make([]DevTarget, 0, len(h.devTargets))
	for _, dt := range h.devTargets {
		targets = append(targets, *dt)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	return targets
}

func buildDevProxy(target *DevTarget) (*url.URL, *httputil.ReverseProxy, error) {
	u, err := url.Parse(target.Target)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Fix IPv6: resolve "localhost" to 127.0.0.1 to avoid connection refused on systems
	// where localhost resolves to [::1] but the dev server only listens on IPv4.
	if u.Hostname() == "localhost" {
		u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Prevent browser/WebView from caching dev proxy responses (CSS, JS, etc.)
		resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		resp.Header.Del("ETag")
		resp.Header.Del("Last-Modified")

		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		modified := injectDevProxyScript(body)
		resp.Body = io.NopCloser(bytes.NewReader(modified))
		resp.ContentLength = int64(len(modified))
		resp.Header.Set("Content-Length", strconv.Itoa(len(modified)))
		resp.Header.Del("Content-Encoding")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><style>
body{background:#1c1c1e;color:#fff;font-family:-apple-system,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
.box{text-align:center;padding:32px}
h2{margin:0 0 12px;font-size:20px;font-weight:600}
p{color:#8e8e93;font-size:14px;margin:0}
</style></head><body><div class="box"><h2>Cannot connect</h2><p>%s</p><p style="margin-top:8px;font-size:12px">Target: %s</p></div></body></html>`,
			escapeHTMLString(err.Error()), escapeHTMLString(target.Target))
	}

	return u, proxy, nil
}
