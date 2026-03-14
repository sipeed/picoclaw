// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"html/template"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	defaultChannelQueueSize = 16
	defaultRateLimit        = 10 // default 10 msg/s
	maxRetries              = 3
	rateLimitDelay          = 1 * time.Second
	baseBackoff             = 500 * time.Millisecond
	maxBackoff              = 8 * time.Second

	janitorInterval = 10 * time.Second
	typingStopTTL   = 5 * time.Minute
	placeholderTTL  = 10 * time.Minute
	previewHost     = "0.0.0.0"
)

// typingEntry wraps a typing stop function with a creation timestamp for TTL eviction.
type typingEntry struct {
	stop      func()
	createdAt time.Time
}

// reactionEntry wraps a reaction undo function with a creation timestamp for TTL eviction.
type reactionEntry struct {
	undo      func()
	createdAt time.Time
}

// placeholderEntry wraps a placeholder ID with a creation timestamp for TTL eviction.
type placeholderEntry struct {
	id        string
	createdAt time.Time
}

// channelRateConfig maps channel name to per-second rate limit.
var channelRateConfig = map[string]float64{
	"telegram": 20,
	"discord":  1,
	"slack":    1,
	"matrix":   2,
	"line":     10,
	"irc":      2,
}

var rootDashboardTemplate = template.Must(template.New("root_dashboard").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>PicoClaw Dashboard</title>
  <style>
    :root {
      --bg: radial-gradient(1200px 600px at 10% -10%, #243b55 0%, #0f2027 35%, #0b0f14 100%);
      --card: rgba(255,255,255,0.06);
      --card-border: rgba(255,255,255,0.12);
      --text: #eaf2ff;
      --muted: #9fb4d1;
      --ok: #1fd29b;
      --warn: #ffd166;
      --accent: #65b7ff;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
    }
    .wrap { max-width: 1080px; margin: 0 auto; padding: 28px 18px 36px; }
    .hero {
      display: flex; justify-content: space-between; align-items: flex-start; gap: 20px;
      margin-bottom: 18px;
    }
    .title { font-size: 30px; font-weight: 700; letter-spacing: 0.4px; margin: 0; }
    .sub { margin: 6px 0 0; color: var(--muted); font-size: 14px; }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 12px;
      margin-bottom: 14px;
    }
    .card {
      background: var(--card);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 14px;
      backdrop-filter: blur(8px);
    }
    .k { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: 0.6px; }
    .v { font-size: 18px; margin-top: 6px; font-weight: 650; }
    .list { margin: 0; padding-left: 18px; line-height: 1.6; }
    .row { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 8px; }
    .chip {
      border: 1px solid var(--card-border);
      border-radius: 999px;
      padding: 6px 10px;
      font-size: 12px;
      color: var(--muted);
      background: rgba(255,255,255,0.04);
    }
    .ok { color: var(--ok); }
    .warn { color: var(--warn); }
    a { color: var(--accent); text-decoration: none; }
    a:hover { text-decoration: underline; }
    .footer { margin-top: 20px; color: var(--muted); font-size: 12px; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1 class="title">PicoClaw Dashboard</h1>
        <p class="sub">Live runtime overview for channels and health endpoints</p>
      </div>
      <div class="chip">Generated {{.GeneratedAt}}</div>
    </div>

    <div class="grid">
      <div class="card">
        <div class="k">Service</div>
        <div class="v">picoclaw <span class="ok">online</span></div>
      </div>
      <div class="card">
        <div class="k">Gateway</div>
        <div class="v">{{.Address}}</div>
      </div>
      <div class="card">
        <div class="k">Channels Enabled</div>
        <div class="v">{{len .Channels}}</div>
      </div>
      <div class="card">
        <div class="k">Health</div>
        <div class="v"><a href="/health">/health</a> · <a href="/ready">/ready</a></div>
      </div>
    </div>

    <div class="card" style="margin-bottom:12px;">
      <div class="k">Channel List</div>
      {{if .Channels}}
      <div class="row">
        {{range .Channels}}<span class="chip">{{.}}</span>{{end}}
      </div>
      {{else}}
      <p class="warn">No channels currently registered.</p>
      {{end}}
    </div>

    <div class="card">
      <div class="k">Webhook Routes</div>
      {{if .Webhooks}}
      <ul class="list">
        {{range .Webhooks}}
        <li><strong>{{.Name}}</strong>: <a href="{{.Path}}">{{.Path}}</a></li>
        {{end}}
      </ul>
      {{else}}
      <p class="warn">No webhook routes registered.</p>
      {{end}}
    </div>

    <p class="footer">Tip: this page auto-refreshes health status every 15s using /health and /ready.</p>
  </div>
  <script>
    async function ping(path) {
      try {
        const r = await fetch(path, { cache: "no-store" });
        return r.ok;
      } catch (_) { return false; }
    }
    async function run() {
      const h = await ping("/health");
      const r = await ping("/ready");
      const title = document.querySelector(".title");
      if (title) {
        title.textContent = h && r ? "PicoClaw Dashboard" : "PicoClaw Dashboard (degraded)";
      }
      setTimeout(run, 15000);
    }
    run();
  </script>
</body>
</html>`))

type dashboardWebhook struct {
	Name string
	Path string
}

type dashboardViewData struct {
	Address     string
	GeneratedAt string
	Channels    []string
	Webhooks    []dashboardWebhook
}

type channelWorker struct {
	ch         Channel
	queue      chan bus.OutboundMessage
	mediaQueue chan bus.OutboundMediaMessage
	done       chan struct{}
	mediaDone  chan struct{}
	limiter    *rate.Limiter
}

type previewMount struct {
	Root      string
	Entry     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Manager struct {
	channels              map[string]Channel
	workers               map[string]*channelWorker
	bus                   *bus.MessageBus
	config                *config.Config
	mediaStore            media.MediaStore
	dispatchTask          *asyncTask
	mux                   *http.ServeMux
	httpServer            *http.Server
	previewServer         *http.Server
	previewAddr           string
	previews              map[string]previewMount
	controlPlaneDiagnoser func(context.Context, string) (string, error)
	mu                    sync.RWMutex
	placeholders          sync.Map // "channel:chatID" → placeholderID (string)
	typingStops           sync.Map // "channel:chatID" → func()
	reactionUndos         sync.Map // "channel:chatID" → reactionEntry
}

type asyncTask struct {
	cancel context.CancelFunc
}

// RecordPlaceholder registers a placeholder message for later editing.
// Implements PlaceholderRecorder.
func (m *Manager) RecordPlaceholder(channel, chatID, placeholderID string) {
	key := channel + ":" + chatID
	m.placeholders.Store(key, placeholderEntry{id: placeholderID, createdAt: time.Now()})
}

// RecordTypingStop registers a typing stop function for later invocation.
// Implements PlaceholderRecorder.
func (m *Manager) RecordTypingStop(channel, chatID string, stop func()) {
	key := channel + ":" + chatID
	m.typingStops.Store(key, typingEntry{stop: stop, createdAt: time.Now()})
}

// RecordReactionUndo registers a reaction undo function for later invocation.
// Implements PlaceholderRecorder.
func (m *Manager) RecordReactionUndo(channel, chatID string, undo func()) {
	key := channel + ":" + chatID
	m.reactionUndos.Store(key, reactionEntry{undo: undo, createdAt: time.Now()})
}

// preSend handles typing stop, reaction undo, and placeholder editing before sending a message.
// Returns true if the message was edited into a placeholder (skip Send).
func (m *Manager) preSend(ctx context.Context, name string, msg bus.OutboundMessage, ch Channel) bool {
	key := name + ":" + msg.ChatID

	// 1. Stop typing
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if entry, ok := v.(typingEntry); ok {
			entry.stop() // idempotent, safe
		}
	}

	// 2. Undo reaction
	if v, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
		if entry, ok := v.(reactionEntry); ok {
			entry.undo() // idempotent, safe
		}
	}

	// 3. Try editing placeholder
	if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if editor, ok := ch.(MessageEditor); ok {
				if err := editor.EditMessage(ctx, msg.ChatID, entry.id, msg.Content); err == nil {
					return true // edited successfully, skip Send
				}
				// edit failed → fall through to normal Send
			}
		}
	}

	return false
}

func NewManager(cfg *config.Config, messageBus *bus.MessageBus, store media.MediaStore) (*Manager, error) {
	m := &Manager{
		channels:   make(map[string]Channel),
		workers:    make(map[string]*channelWorker),
		bus:        messageBus,
		config:     cfg,
		mediaStore: store,
		previews:   make(map[string]previewMount),
	}

	if err := m.initChannels(); err != nil {
		return nil, err
	}

	return m, nil
}

// SetControlPlaneDiagnoser injects a callback used by the control-plane UI when
// an operator requests an LLM-backed diagnosis from the backend.
func (m *Manager) SetControlPlaneDiagnoser(fn func(context.Context, string) (string, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.controlPlaneDiagnoser = fn
}

// initChannel is a helper that looks up a factory by name and creates the channel.
func (m *Manager) initChannel(name, displayName string) {
	f, ok := getFactory(name)
	if !ok {
		logger.WarnCF("channels", "Factory not registered", map[string]any{
			"channel": displayName,
		})
		return
	}
	logger.DebugCF("channels", "Attempting to initialize channel", map[string]any{
		"channel": displayName,
	})
	ch, err := f(m.config, m.bus)
	if err != nil {
		logger.ErrorCF("channels", "Failed to initialize channel", map[string]any{
			"channel": displayName,
			"error":   err.Error(),
		})
	} else {
		// Inject MediaStore if channel supports it
		if m.mediaStore != nil {
			if setter, ok := ch.(interface{ SetMediaStore(s media.MediaStore) }); ok {
				setter.SetMediaStore(m.mediaStore)
			}
		}
		// Inject PlaceholderRecorder if channel supports it
		if setter, ok := ch.(interface{ SetPlaceholderRecorder(r PlaceholderRecorder) }); ok {
			setter.SetPlaceholderRecorder(m)
		}
		// Inject owner reference so BaseChannel.HandleMessage can auto-trigger typing/reaction
		if setter, ok := ch.(interface{ SetOwner(ch Channel) }); ok {
			setter.SetOwner(ch)
		}
		m.channels[name] = ch
		logger.InfoCF("channels", "Channel enabled successfully", map[string]any{
			"channel": displayName,
		})
	}
}

func (m *Manager) initChannels() error {
	logger.InfoC("channels", "Initializing channel manager")

	if m.config.Channels.Telegram.Enabled && m.config.Channels.Telegram.Token != "" {
		m.initChannel("telegram", "Telegram")
	}

	if m.config.Channels.WhatsApp.Enabled {
		waCfg := m.config.Channels.WhatsApp
		if waCfg.UseNative {
			m.initChannel("whatsapp_native", "WhatsApp Native")
		} else if waCfg.BridgeURL != "" {
			m.initChannel("whatsapp", "WhatsApp")
		}
	}

	if m.config.Channels.Feishu.Enabled {
		m.initChannel("feishu", "Feishu")
	}

	if m.config.Channels.Discord.Enabled && m.config.Channels.Discord.Token != "" {
		m.initChannel("discord", "Discord")
	}

	if m.config.Channels.MaixCam.Enabled {
		m.initChannel("maixcam", "MaixCam")
	}

	if m.config.Channels.QQ.Enabled {
		m.initChannel("qq", "QQ")
	}

	if m.config.Channels.DingTalk.Enabled && m.config.Channels.DingTalk.ClientID != "" {
		m.initChannel("dingtalk", "DingTalk")
	}

	if m.config.Channels.Slack.Enabled && m.config.Channels.Slack.BotToken != "" {
		m.initChannel("slack", "Slack")
	}

	if m.config.Channels.Matrix.Enabled &&
		m.config.Channels.Matrix.Homeserver != "" &&
		m.config.Channels.Matrix.UserID != "" &&
		m.config.Channels.Matrix.AccessToken != "" {
		m.initChannel("matrix", "Matrix")
	}

	if m.config.Channels.LINE.Enabled && m.config.Channels.LINE.ChannelAccessToken != "" {
		m.initChannel("line", "LINE")
	}

	if m.config.Channels.OneBot.Enabled && m.config.Channels.OneBot.WSUrl != "" {
		m.initChannel("onebot", "OneBot")
	}

	if m.config.Channels.WeCom.Enabled && m.config.Channels.WeCom.Token != "" {
		m.initChannel("wecom", "WeCom")
	}

	if m.config.Channels.WeComAIBot.Enabled && m.config.Channels.WeComAIBot.Token != "" {
		m.initChannel("wecom_aibot", "WeCom AI Bot")
	}

	if m.config.Channels.WeComApp.Enabled && m.config.Channels.WeComApp.CorpID != "" {
		m.initChannel("wecom_app", "WeCom App")
	}

	if m.config.Channels.Pico.Enabled && m.config.Channels.Pico.Token != "" {
		m.initChannel("pico", "Pico")
	}

	if m.config.Channels.IRC.Enabled && m.config.Channels.IRC.Server != "" {
		m.initChannel("irc", "IRC")
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]any{
		"enabled_channels": len(m.channels),
	})

	return nil
}

// SetupHTTPServer creates a shared HTTP server with the given listen address.
// It registers health endpoints from the health server and discovers channels
// that implement WebhookHandler and/or HealthChecker to register their handlers.
func (m *Manager) SetupHTTPServer(addr string, healthServer *health.Server) {
	m.mux = http.NewServeMux()

	// Register the control-plane dashboard and API (root endpoint).
	m.registerControlPlaneRoutes(addr)
	m.setupPreviewServer(addr)

	// Keep the original lightweight dashboard available as fallback.
	m.mux.HandleFunc("/legacy-dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/legacy-dashboard" {
			http.NotFound(w, r)
			return
		}

		data := dashboardViewData{
			Address:     addr,
			GeneratedAt: time.Now().Format(time.RFC3339),
			Channels:    make([]string, 0, len(m.channels)),
			Webhooks:    make([]dashboardWebhook, 0, len(m.channels)),
		}

		m.mu.RLock()
		for name, ch := range m.channels {
			data.Channels = append(data.Channels, name)
			if wh, ok := ch.(WebhookHandler); ok {
				data.Webhooks = append(data.Webhooks, dashboardWebhook{
					Name: name,
					Path: wh.WebhookPath(),
				})
			}
		}
		m.mu.RUnlock()

		sort.Strings(data.Channels)
		sort.Slice(data.Webhooks, func(i, j int) bool { return data.Webhooks[i].Name < data.Webhooks[j].Name })

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := rootDashboardTemplate.Execute(w, data); err != nil {
			logger.ErrorCF("channels", "legacy dashboard render failed", map[string]any{"error": err.Error()})
			http.Error(w, "legacy dashboard render failed", http.StatusInternalServerError)
			return
		}
	})

	// Register health endpoints
	if healthServer != nil {
		healthServer.RegisterOnMux(m.mux)
	}

	// Discover and register webhook handlers and health checkers
	for name, ch := range m.channels {
		if wh, ok := ch.(WebhookHandler); ok {
			m.mux.Handle(wh.WebhookPath(), wh)
			logger.InfoCF("channels", "Webhook handler registered", map[string]any{
				"channel": name,
				"path":    wh.WebhookPath(),
			})
		}
		if hc, ok := ch.(HealthChecker); ok {
			m.mux.HandleFunc(hc.HealthPath(), hc.HealthHandler)
			logger.InfoCF("channels", "Health endpoint registered", map[string]any{
				"channel": name,
				"path":    hc.HealthPath(),
			})
		}
	}

	m.httpServer = &http.Server{
		Addr:         addr,
		Handler:      m.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

func (m *Manager) setupPreviewServer(addr string) {
	previewAddr := derivePreviewAddr(addr)
	previewMux := http.NewServeMux()
	previewMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("PicoClaw preview server\n"))
	})
	previewMux.HandleFunc("/preview/", m.handlePreviewRequest)
	m.previewAddr = previewAddr
	m.previewServer = &http.Server{
		Addr:         previewAddr,
		Handler:      previewMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

func derivePreviewAddr(addr string) string {
	_, port := splitHostPort(addr)
	if port <= 0 {
		port = 3001
	}
	return net.JoinHostPort(previewHost, fmt.Sprintf("%d", port+1))
}

func splitHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return "", 0
	}
	port, err := net.LookupPort("tcp", portStr)
	if err != nil {
		return host, 0
	}
	return host, port
}

func sanitizePreviewSlug(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if prevDash {
			continue
		}
		b.WriteByte('-')
		prevDash = true
	}
	return strings.Trim(b.String(), "-")
}

func defaultPreviewSlug(root string) string {
	base := sanitizePreviewSlug(filepath.Base(strings.TrimSpace(root)))
	if base == "" {
		base = "preview"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(root))
	return fmt.Sprintf("%s-%08x", base, h.Sum32())
}

func (m *Manager) PublishPreview(root, entry, slug string) (string, string, string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", "", "", fmt.Errorf("preview root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve preview root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", "", "", fmt.Errorf("preview root not found: %w", err)
	}
	if !info.IsDir() {
		return "", "", "", fmt.Errorf("preview root must be a directory")
	}

	entry = strings.TrimSpace(strings.TrimLeft(filepath.ToSlash(entry), "/"))
	if entry != "" {
		if strings.HasPrefix(entry, "../") || entry == ".." {
			return "", "", "", fmt.Errorf("preview entry escapes hosted directory")
		}
		entryPath := filepath.Join(absRoot, filepath.FromSlash(entry))
		entryInfo, statErr := os.Stat(entryPath)
		if statErr != nil {
			return "", "", "", fmt.Errorf("preview entry not found: %w", statErr)
		}
		if entryInfo.IsDir() {
			entry = strings.TrimSuffix(entry, "/") + "/index.html"
		}
	} else if _, statErr := os.Stat(filepath.Join(absRoot, "index.html")); statErr == nil {
		entry = "index.html"
	}

	slug = sanitizePreviewSlug(slug)
	if slug == "" {
		slug = defaultPreviewSlug(absRoot)
	}

	now := time.Now()
	m.mu.Lock()
	createdAt := now
	if existing, ok := m.previews[slug]; ok {
		createdAt = existing.CreatedAt
	}
	m.previews = map[string]previewMount{
		slug: {Root: absRoot, Entry: entry, CreatedAt: createdAt, UpdatedAt: now},
	}
	m.mu.Unlock()

	tailscaleBase, localBase := m.previewBases()
	return slug, buildPreviewURL(tailscaleBase, slug, entry), buildPreviewURL(localBase, slug, entry), nil
}

func (m *Manager) previewBases() (string, string) {
	_, port := splitHostPort(m.previewAddr)
	if port <= 0 {
		port = 3002
	}
	localBase := fmt.Sprintf("http://127.0.0.1:%d", port)
	tailscaleIP := detectTailscaleIPv4()
	if tailscaleIP == "" {
		return "", localBase
	}
	return fmt.Sprintf("http://%s:%d", tailscaleIP, port), localBase
}

func detectTailscaleIPv4() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tailscale", "ip", "-4")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func buildPreviewURL(base, slug, entry string) string {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(slug) == "" {
		return ""
	}
	base = strings.TrimRight(base, "/")
	urlPath := "/preview/" + slug + "/"
	if entry != "" {
		urlPath += strings.TrimLeft(filepath.ToSlash(entry), "/")
	}
	return base + urlPath
}

func (m *Manager) handlePreviewRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	trimmed := strings.TrimPrefix(path.Clean(r.URL.Path), "/preview/")
	if trimmed == "." || trimmed == "" || strings.HasPrefix(trimmed, "../") {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(trimmed, "/", 2)
	slug := parts[0]
	relPath := ""
	if len(parts) == 2 {
		relPath = strings.TrimPrefix(parts[1], "/")
	}

	m.mu.RLock()
	mount, ok := m.previews[slug]
	m.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	if relPath == "" {
		relPath = mount.Entry
	}
	if relPath == "" {
		relPath = "."
	}
	relPath = strings.TrimPrefix(path.Clean("/"+relPath), "/")
	if strings.HasPrefix(relPath, "../") || relPath == ".." {
		http.NotFound(w, r)
		return
	}

	target := filepath.Join(mount.Root, filepath.FromSlash(relPath))
	if !isPreviewPathWithin(target, mount.Root) {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(target); err == nil {
		if info.IsDir() {
			indexPath := filepath.Join(target, "index.html")
			if _, statErr := os.Stat(indexPath); statErr == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
		} else {
			http.ServeFile(w, r, target)
			return
		}
	}

	if mount.Entry != "" && path.Ext(relPath) == "" {
		fallback := filepath.Join(mount.Root, filepath.FromSlash(mount.Entry))
		if isPreviewPathWithin(fallback, mount.Root) {
			if _, err := os.Stat(fallback); err == nil {
				http.ServeFile(w, r, fallback)
				return
			}
		}
	}

	http.NotFound(w, r)
}

func isPreviewPathWithin(candidate, root string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
		return errors.New("no channels enabled")
	}

	logger.InfoC("channels", "Starting all channels")

	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Starting channel", map[string]any{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
			continue
		}
		// Lazily create worker only after channel starts successfully
		w := newChannelWorker(name, channel)
		m.workers[name] = w
		go m.runWorker(dispatchCtx, name, w)
		go m.runMediaWorker(dispatchCtx, name, w)
	}

	// Start the dispatcher that reads from the bus and routes to workers
	go m.dispatchOutbound(dispatchCtx)
	go m.dispatchOutboundMedia(dispatchCtx)

	// Start the TTL janitor that cleans up stale typing/placeholder entries
	go m.runTTLJanitor(dispatchCtx)

	// Start shared HTTP server if configured
	if m.httpServer != nil {
		go func() {
			logger.InfoCF("channels", "Shared HTTP server listening", map[string]any{
				"addr": m.httpServer.Addr,
			})
			if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.ErrorCF("channels", "Shared HTTP server error", map[string]any{
					"error": err.Error(),
				})
			}
		}()
	}

	if m.previewServer != nil {
		go func() {
			logger.InfoCF("channels", "Preview server listening", map[string]any{
				"addr": m.previewServer.Addr,
			})
			if err := m.previewServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.ErrorCF("channels", "Preview server error", map[string]any{
					"error": err.Error(),
				})
			}
		}()
	}

	logger.InfoC("channels", "All channels started")
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	// Shutdown shared HTTP server first
	if m.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := m.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("channels", "Shared HTTP server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
		m.httpServer = nil
	}

	if m.previewServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := m.previewServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("channels", "Preview server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
		m.previewServer = nil
	}

	// Cancel dispatcher
	if m.dispatchTask != nil {
		m.dispatchTask.cancel()
		m.dispatchTask = nil
	}

	// Close all worker queues and wait for them to drain
	for _, w := range m.workers {
		if w != nil {
			close(w.queue)
		}
	}
	for _, w := range m.workers {
		if w != nil {
			<-w.done
		}
	}
	// Close all media worker queues and wait for them to drain
	for _, w := range m.workers {
		if w != nil {
			close(w.mediaQueue)
		}
	}
	for _, w := range m.workers {
		if w != nil {
			<-w.mediaDone
		}
	}

	// Stop all channels
	for name, channel := range m.channels {
		logger.InfoCF("channels", "Stopping channel", map[string]any{
			"channel": name,
		})
		if err := channel.Stop(ctx); err != nil {
			logger.ErrorCF("channels", "Error stopping channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels stopped")
	return nil
}

// newChannelWorker creates a channelWorker with a rate limiter configured
// for the given channel name.
func newChannelWorker(name string, ch Channel) *channelWorker {
	rateVal := float64(defaultRateLimit)
	if r, ok := channelRateConfig[name]; ok {
		rateVal = r
	}
	burst := int(math.Max(1, math.Ceil(rateVal/2)))

	return &channelWorker{
		ch:         ch,
		queue:      make(chan bus.OutboundMessage, defaultChannelQueueSize),
		mediaQueue: make(chan bus.OutboundMediaMessage, defaultChannelQueueSize),
		done:       make(chan struct{}),
		mediaDone:  make(chan struct{}),
		limiter:    rate.NewLimiter(rate.Limit(rateVal), burst),
	}
}

// runWorker processes outbound messages for a single channel, splitting
// messages that exceed the channel's maximum message length.
func (m *Manager) runWorker(ctx context.Context, name string, w *channelWorker) {
	defer close(w.done)
	for {
		select {
		case msg, ok := <-w.queue:
			if !ok {
				return
			}
			maxLen := 0
			if mlp, ok := w.ch.(MessageLengthProvider); ok {
				maxLen = mlp.MaxMessageLength()
			}
			if maxLen > 0 && len([]rune(msg.Content)) > maxLen {
				chunks := SplitMessage(msg.Content, maxLen)
				for _, chunk := range chunks {
					chunkMsg := msg
					chunkMsg.Content = chunk
					m.sendWithRetry(ctx, name, w, chunkMsg)
				}
			} else {
				m.sendWithRetry(ctx, name, w, msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

// sendWithRetry sends a message through the channel with rate limiting and
// retry logic. It classifies errors to determine the retry strategy:
//   - ErrNotRunning / ErrSendFailed: permanent, no retry
//   - ErrRateLimit: fixed delay retry
//   - ErrTemporary / unknown: exponential backoff retry
func (m *Manager) sendWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	// Rate limit: wait for token
	if err := w.limiter.Wait(ctx); err != nil {
		// ctx canceled, shutting down
		return
	}

	// Pre-send: stop typing and try to edit placeholder
	if m.preSend(ctx, name, msg, w.ch) {
		return // placeholder was edited successfully, skip Send
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = w.ch.Send(ctx, msg)
		if lastErr == nil {
			return
		}

		// Permanent failures — don't retry
		if errors.Is(lastErr, ErrNotRunning) || errors.Is(lastErr, ErrSendFailed) {
			break
		}

		// Last attempt exhausted — don't sleep
		if attempt == maxRetries {
			break
		}

		// Rate limit error — fixed delay
		if errors.Is(lastErr, ErrRateLimit) {
			select {
			case <-time.After(rateLimitDelay):
				continue
			case <-ctx.Done():
				return
			}
		}

		// ErrTemporary or unknown error — exponential backoff
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
	}

	// All retries exhausted or permanent failure
	logger.ErrorCF("channels", "Send failed", map[string]any{
		"channel": name,
		"chat_id": msg.ChatID,
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
}

func dispatchLoop[M any](
	ctx context.Context,
	m *Manager,
	subscribe func(context.Context) (M, bool),
	getChannel func(M) string,
	enqueue func(context.Context, *channelWorker, M) bool,
	startMsg, stopMsg, unknownMsg, noWorkerMsg string,
) {
	logger.InfoC("channels", startMsg)

	for {
		msg, ok := subscribe(ctx)
		if !ok {
			logger.InfoC("channels", stopMsg)
			return
		}

		channel := getChannel(msg)

		// Silently skip internal channels
		if constants.IsInternalChannel(channel) {
			continue
		}

		m.mu.RLock()
		_, exists := m.channels[channel]
		w, wExists := m.workers[channel]
		m.mu.RUnlock()

		if !exists {
			logger.WarnCF("channels", unknownMsg, map[string]any{"channel": channel})
			continue
		}

		if wExists && w != nil {
			if !enqueue(ctx, w, msg) {
				return
			}
		} else if exists {
			logger.WarnCF("channels", noWorkerMsg, map[string]any{"channel": channel})
		}
	}
}

func (m *Manager) dispatchOutbound(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.SubscribeOutbound,
		func(msg bus.OutboundMessage) string { return msg.Channel },
		func(ctx context.Context, w *channelWorker, msg bus.OutboundMessage) bool {
			select {
			case w.queue <- msg:
				return true
			case <-ctx.Done():
				return false
			}
		},
		"Outbound dispatcher started",
		"Outbound dispatcher stopped",
		"Unknown channel for outbound message",
		"Channel has no active worker, skipping message",
	)
}

func (m *Manager) dispatchOutboundMedia(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.SubscribeOutboundMedia,
		func(msg bus.OutboundMediaMessage) string { return msg.Channel },
		func(ctx context.Context, w *channelWorker, msg bus.OutboundMediaMessage) bool {
			select {
			case w.mediaQueue <- msg:
				return true
			case <-ctx.Done():
				return false
			}
		},
		"Outbound media dispatcher started",
		"Outbound media dispatcher stopped",
		"Unknown channel for outbound media message",
		"Channel has no active worker, skipping media message",
	)
}

// runMediaWorker processes outbound media messages for a single channel.
func (m *Manager) runMediaWorker(ctx context.Context, name string, w *channelWorker) {
	defer close(w.mediaDone)
	for {
		select {
		case msg, ok := <-w.mediaQueue:
			if !ok {
				return
			}
			m.sendMediaWithRetry(ctx, name, w, msg)
		case <-ctx.Done():
			return
		}
	}
}

// sendMediaWithRetry sends a media message through the channel with rate limiting and
// retry logic. If the channel does not implement MediaSender, it silently skips.
func (m *Manager) sendMediaWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMediaMessage) {
	ms, ok := w.ch.(MediaSender)
	if !ok {
		logger.DebugCF("channels", "Channel does not support MediaSender, skipping media", map[string]any{
			"channel": name,
		})
		return
	}

	// Rate limit: wait for token
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = ms.SendMedia(ctx, msg)
		if lastErr == nil {
			return
		}

		// Permanent failures — don't retry
		if errors.Is(lastErr, ErrNotRunning) || errors.Is(lastErr, ErrSendFailed) {
			break
		}

		// Last attempt exhausted — don't sleep
		if attempt == maxRetries {
			break
		}

		// Rate limit error — fixed delay
		if errors.Is(lastErr, ErrRateLimit) {
			select {
			case <-time.After(rateLimitDelay):
				continue
			case <-ctx.Done():
				return
			}
		}

		// ErrTemporary or unknown error — exponential backoff
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
	}

	// All retries exhausted or permanent failure
	logger.ErrorCF("channels", "SendMedia failed", map[string]any{
		"channel": name,
		"chat_id": msg.ChatID,
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
}

// runTTLJanitor periodically scans the typingStops and placeholders maps
// and evicts entries that have exceeded their TTL. This prevents memory
// accumulation when outbound paths fail to trigger preSend (e.g. LLM errors).
func (m *Manager) runTTLJanitor(ctx context.Context) {
	ticker := time.NewTicker(janitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			m.typingStops.Range(func(key, value any) bool {
				if entry, ok := value.(typingEntry); ok {
					if now.Sub(entry.createdAt) > typingStopTTL {
						if _, loaded := m.typingStops.LoadAndDelete(key); loaded {
							entry.stop() // idempotent, safe
						}
					}
				}
				return true
			})
			m.reactionUndos.Range(func(key, value any) bool {
				if entry, ok := value.(reactionEntry); ok {
					if now.Sub(entry.createdAt) > typingStopTTL {
						if _, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
							entry.undo() // idempotent, safe
						}
					}
				}
				return true
			})
			m.placeholders.Range(func(key, value any) bool {
				if entry, ok := value.(placeholderEntry); ok {
					if now.Sub(entry.createdAt) > placeholderTTL {
						m.placeholders.Delete(key)
					}
				}
				return true
			})
		}
	}
}

func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channel, ok := m.channels[name]
	return channel, ok
}

func (m *Manager) GetStatus() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]any)
	for name, channel := range m.channels {
		status[name] = map[string]any{
			"enabled": true,
			"running": channel.IsRunning(),
		}
	}
	return status
}

func (m *Manager) GetEnabledChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

func (m *Manager) RegisterChannel(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
}

func (m *Manager) UnregisterChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w, ok := m.workers[name]; ok && w != nil {
		close(w.queue)
		<-w.done
		close(w.mediaQueue)
		<-w.mediaDone
	}
	delete(m.workers, name)
	delete(m.channels, name)
}

func (m *Manager) SendToChannel(ctx context.Context, channelName, chatID, content string) error {
	m.mu.RLock()
	_, exists := m.channels[channelName]
	w, wExists := m.workers[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	msg := bus.OutboundMessage{
		Channel: channelName,
		ChatID:  chatID,
		Content: content,
	}

	if wExists && w != nil {
		select {
		case w.queue <- msg:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Fallback: direct send (should not happen)
	channel, _ := m.channels[channelName]
	return channel.Send(ctx, msg)
}
