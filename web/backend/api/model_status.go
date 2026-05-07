package api

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	modelProbeTimeout             = 800 * time.Millisecond
	modelProbeSuccessBaseInterval = 2 * time.Second
	modelProbeSuccessMaxInterval  = 60 * time.Second
	modelProbeFailureBaseInterval = 1 * time.Second
	modelProbeFailureMaxInterval  = 30 * time.Second
	modelProbeBackoffMaxShift     = 8
	modelProbeCacheMaxEntries     = 1024
	modelProbeCacheEntryTTL       = 30 * time.Minute
	modelProbeCacheTrimToEntries  = modelProbeCacheMaxEntries * 8 / 10
	modelProbeTTLGCInterval       = 1 * time.Minute
)

const (
	modelStatusAvailable    = "available"
	modelStatusUnconfigured = "unconfigured"
	modelStatusUnreachable  = "unreachable"
)

type modelConfigurationSummary struct {
	Available bool
	Status    string
}

var (
	probeTCPServiceFunc            = probeTCPService
	probeOllamaModelFunc           = probeOllamaModel
	probeOpenAICompatibleModelFunc = probeOpenAICompatibleModel
	probeCommandAvailableFunc      = probeCommandAvailable
	modelProbeNowFunc              = time.Now
	modelProbeState                = newModelProbeCacheState()
)

type modelProbeCacheState struct {
	mu          sync.RWMutex
	cache       map[string]*modelProbeCacheEntry
	group       singleflight.Group
	nextTTLGCAt time.Time
}

type modelProbeCacheEntry struct {
	lastResult    bool
	hasResult     bool
	successStreak int
	failureStreak int
	nextProbeAt   time.Time
	updatedAt     time.Time
}

func newModelProbeCacheState() *modelProbeCacheState {
	return &modelProbeCacheState{cache: map[string]*modelProbeCacheEntry{}}
}

func resetModelProbeCache() {
	modelProbeState.resetForTest()
}

func (s *modelProbeCacheState) resetForTest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = map[string]*modelProbeCacheEntry{}
	s.nextTTLGCAt = time.Time{}
}

func hasModelConfiguration(m *config.ModelConfig) bool {
	protocol := modelProtocol(m)
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	apiKey := strings.TrimSpace(m.APIKey())

	if authMethod == "oauth" || authMethod == "token" {
		if configured, checked := hasStoredOAuthCredential(m); checked {
			return configured
		}
	}

	if authMethod == "" && providerUsesImplicitOAuth(protocol) {
		if configured, checked := hasStoredOAuthCredential(m); checked {
			return configured
		}
	}

	if providerUsesAmbientCredentials(protocol) {
		return true
	}

	if requiresRuntimeProbe(m) {
		return true
	}

	return apiKey != ""
}

func hasStoredOAuthCredential(m *config.ModelConfig) (bool, bool) {
	provider, ok := oauthProviderForModel(m)
	if !ok {
		return false, false
	}
	cred, err := oauthGetCredential(provider)
	if err != nil || cred == nil {
		return false, true
	}
	return strings.TrimSpace(cred.AccessToken) != "" || strings.TrimSpace(cred.RefreshToken) != "", true
}

func providerUsesImplicitOAuth(protocol string) bool {
	switch protocol {
	case "antigravity", "google-antigravity":
		return true
	default:
		return false
	}
}

func providerUsesAmbientCredentials(protocol string) bool {
	switch protocol {
	case "bedrock":
		// Bedrock relies on the AWS SDK credential chain instead of an explicit
		// API key stored in ModelConfig. We cannot reliably preflight every AWS
		// credential source here, so avoid misclassifying valid environments as
		// "unconfigured" and defer concrete credential failures to runtime.
		return true
	default:
		return false
	}
}

func modelConfigurationStatus(m *config.ModelConfig) modelConfigurationSummary {
	if !hasModelConfiguration(m) {
		return modelConfigurationSummary{Available: false, Status: modelStatusUnconfigured}
	}
	if requiresRuntimeProbe(m) {
		if probeLocalModelAvailability(m) {
			return modelConfigurationSummary{Available: true, Status: modelStatusAvailable}
		}
		return modelConfigurationSummary{Available: false, Status: modelStatusUnreachable}
	}
	return modelConfigurationSummary{Available: true, Status: modelStatusAvailable}
}

func requiresRuntimeProbe(m *config.ModelConfig) bool {
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if authMethod == "local" {
		return true
	}

	protocol := modelProtocol(m)

	switch protocol {
	case "claude-cli", "claudecli", "codex-cli", "codexcli", "github-copilot", "copilot":
		return true
	}

	if providers.IsEmptyAPIKeyAllowedForProtocol(protocol) {
		apiBase := strings.TrimSpace(m.APIBase)
		return apiBase == "" || hasLocalAPIBase(apiBase)
	}

	if hasLocalAPIBase(m.APIBase) {
		return true
	}

	return false
}

func probeLocalModelAvailability(m *config.ModelConfig) bool {
	cacheKey := modelProbeCacheKey(m)
	return modelProbeState.probe(cacheKey, func() bool {
		return runLocalModelProbe(m)
	})
}

func (s *modelProbeCacheState) probe(cacheKey string, probeFunc func() bool) bool {
	now := modelProbeNowFunc()
	if cachedResult, ok := s.getCachedResult(cacheKey, now); ok {
		return cachedResult
	}

	v, _, _ := s.group.Do(cacheKey, func() (any, error) {
		now = modelProbeNowFunc()
		if cachedResult, ok := s.getCachedResult(cacheKey, now); ok {
			return cachedResult, nil
		}

		result := probeFunc()
		s.setCachedResult(cacheKey, result, now)
		return result, nil
	})

	result, _ := v.(bool)
	return result
}

func runLocalModelProbe(m *config.ModelConfig) bool {
	apiBase := modelProbeAPIBase(m)
	protocol, modelID := splitModel(m)
	switch protocol {
	case "ollama":
		return probeOllamaModelFunc(apiBase, modelID)
	case "vllm", "lmstudio":
		return probeOpenAICompatibleModelFunc(apiBase, modelID, m.APIKey())
	case "github-copilot", "copilot":
		return probeTCPServiceFunc(apiBase)
	case "claude-cli", "claudecli":
		return probeCommandAvailableFunc("claude")
	case "codex-cli", "codexcli":
		return probeCommandAvailableFunc("codex")
	default:
		if hasLocalAPIBase(apiBase) {
			return probeOpenAICompatibleModelFunc(apiBase, modelID, m.APIKey())
		}
		return false
	}
}

func probeCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func modelProbeCacheKey(m *config.ModelConfig) string {
	protocol, modelID := splitModel(m)

	apiBaseRaw := modelProbeAPIBase(m)
	apiBase := strings.ToLower(strings.TrimRight(strings.TrimSpace(apiBaseRaw), "/"))
	apiKeyFingerprint := modelProbeAPIKeyFingerprint(m.APIKey())

	var b strings.Builder
	b.Grow(len(protocol) + len(modelID) + len(apiBase) + len(apiKeyFingerprint) + 8)
	b.WriteString(protocol)
	b.WriteByte('|')
	b.WriteString(modelID)
	b.WriteByte('|')
	b.WriteString(apiBase)
	b.WriteByte('|')
	b.WriteString(apiKeyFingerprint)

	return b.String()
}

func modelProbeAPIKeyFingerprint(raw string) string {
	apiKey := strings.TrimSpace(raw)
	if apiKey == "" {
		return "none"
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(apiKey))
	return strconv.FormatUint(h.Sum64(), 36)
}

func (s *modelProbeCacheState) getCachedResult(cacheKey string, now time.Time) (bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[cacheKey]
	if !ok || !entry.hasResult {
		return false, false
	}
	if now.Before(entry.nextProbeAt) {
		return entry.lastResult, true
	}
	return false, false
}

func (s *modelProbeCacheState) setCachedResult(cacheKey string, result bool, now time.Time) {
	s.mu.Lock()

	entry, ok := s.cache[cacheKey]
	if !ok {
		entry = &modelProbeCacheEntry{}
		s.cache[cacheKey] = entry
	}

	entry.lastResult = result
	entry.hasResult = true
	entry.updatedAt = now

	var delay time.Duration
	if result {
		entry.successStreak++
		entry.failureStreak = 0
		delay = modelProbeBackoffDelay(
			modelProbeSuccessBaseInterval,
			modelProbeSuccessMaxInterval,
			entry.successStreak,
		)
	} else {
		entry.failureStreak++
		entry.successStreak = 0
		delay = modelProbeBackoffDelay(
			modelProbeFailureBaseInterval,
			modelProbeFailureMaxInterval,
			entry.failureStreak,
		)
	}

	entry.nextProbeAt = now.Add(delay)

	shouldRunTTLGC := modelProbeCacheEntryTTL > 0 && (s.nextTTLGCAt.IsZero() || !now.Before(s.nextTTLGCAt))
	if shouldRunTTLGC {
		s.nextTTLGCAt = now.Add(modelProbeTTLGCInterval)
	}
	shouldRunSizeGC := len(s.cache) > modelProbeCacheMaxEntries
	s.mu.Unlock()

	if shouldRunTTLGC || shouldRunSizeGC {
		s.gc(now, shouldRunTTLGC)
	}
}

func (s *modelProbeCacheState) gc(now time.Time, runTTL bool) {
	type evictionCandidate struct {
		key       string
		updatedAt time.Time
	}

	var expireBefore time.Time
	if runTTL && modelProbeCacheEntryTTL > 0 {
		expireBefore = now.Add(-modelProbeCacheEntryTTL)
	}

	s.mu.RLock()
	cacheLen := len(s.cache)
	if cacheLen == 0 {
		s.mu.RUnlock()
		return
	}

	expiredKeys := make([]string, 0)
	if !expireBefore.IsZero() {
		expiredKeys = make([]string, 0, min(cacheLen/8+1, 64))
		for key, entry := range s.cache {
			if entry.updatedAt.Before(expireBefore) {
				expiredKeys = append(expiredKeys, key)
			}
		}
	}

	effectiveLen := cacheLen - len(expiredKeys)
	removeCount := max(effectiveLen-modelProbeCacheTrimToEntries, 0)

	candidates := make([]evictionCandidate, 0)
	if removeCount > 0 {
		candidates = make([]evictionCandidate, 0, effectiveLen)
		for key, entry := range s.cache {
			if !expireBefore.IsZero() && entry.updatedAt.Before(expireBefore) {
				continue
			}
			candidates = append(candidates, evictionCandidate{key: key, updatedAt: entry.updatedAt})
		}
	}
	s.mu.RUnlock()

	if len(expiredKeys) == 0 && len(candidates) == 0 {
		return
	}

	toEvict := map[string]time.Time{}
	for i := 0; i < removeCount && len(candidates) > 0; i++ {
		oldest := 0
		for j := 1; j < len(candidates); j++ {
			if candidates[j].updatedAt.Before(candidates[oldest].updatedAt) {
				oldest = j
			}
		}
		victim := candidates[oldest]
		toEvict[victim.key] = victim.updatedAt
		candidates[oldest] = candidates[len(candidates)-1]
		candidates = candidates[:len(candidates)-1]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !expireBefore.IsZero() {
		for _, key := range expiredKeys {
			entry, ok := s.cache[key]
			if ok && entry.updatedAt.Before(expireBefore) {
				delete(s.cache, key)
			}
		}
	}

	for key, victimUpdatedAt := range toEvict {
		entry, ok := s.cache[key]
		if ok && !entry.updatedAt.After(victimUpdatedAt) {
			delete(s.cache, key)
		}
	}
}

func modelProbeBackoffDelay(base, maxDelay time.Duration, streak int) time.Duration {
	if streak <= 0 {
		streak = 1
	}

	shift := min(streak-1, modelProbeBackoffMaxShift)

	delay := base * time.Duration(1<<shift)
	if maxDelay > 0 && (delay > maxDelay || delay < 0) {
		return maxDelay
	}
	if delay <= 0 {
		return base
	}
	return delay
}

func modelProbeAPIBase(m *config.ModelConfig) string {
	if apiBase := strings.TrimSpace(m.APIBase); apiBase != "" {
		return normalizeModelProbeAPIBase(apiBase)
	}

	protocol := modelProtocol(m)
	if providers.IsEmptyAPIKeyAllowedForProtocol(protocol) {
		return providers.DefaultAPIBaseForProtocol(protocol)
	}

	switch protocol {
	case "github-copilot", "copilot":
		return "localhost:4321"
	default:
		return ""
	}
}

func normalizeModelProbeAPIBase(raw string) string {
	u, err := parseAPIBase(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}

	switch strings.ToLower(u.Hostname()) {
	case "0.0.0.0":
		u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	case "::":
		u.Host = net.JoinHostPort("::1", u.Port())
	default:
		return strings.TrimSpace(raw)
	}

	if u.Port() == "" {
		u.Host = u.Hostname()
	}

	return u.String()
}

func oauthProviderForModel(m *config.ModelConfig) (string, bool) {
	switch modelProtocol(m) {
	case "openai":
		return oauthProviderOpenAI, true
	case "anthropic":
		return oauthProviderAnthropic, true
	case "antigravity", "google-antigravity":
		return oauthProviderGoogleAntigravity, true
	default:
		return "", false
	}
}

func modelProtocol(m *config.ModelConfig) string {
	protocol, _ := splitModel(m)
	return protocol
}

func splitModel(m *config.ModelConfig) (protocol, modelID string) {
	protocol, modelID = providers.ExtractProtocol(m)
	return strings.ToLower(strings.TrimSpace(protocol)), strings.ToLower(strings.TrimSpace(modelID))
}

func hasLocalAPIBase(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		u, err = url.Parse("//" + raw)
		if err != nil {
			return false
		}
	}

	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	default:
		return false
	}
}

func probeTCPService(raw string) bool {
	hostPort, err := hostPortFromAPIBase(raw)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), modelProbeTimeout)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func probeOllamaModel(apiBase, modelID string) bool {
	root, err := apiRootFromAPIBase(apiBase)
	if err != nil {
		return false
	}

	var resp struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := getJSON(root+"/api/tags", &resp, ""); err != nil {
		return false
	}

	for _, model := range resp.Models {
		if ollamaModelMatches(model.Name, modelID) || ollamaModelMatches(model.Model, modelID) {
			return true
		}
	}
	return false
}

func probeOpenAICompatibleModel(apiBase, modelID, apiKey string) bool {
	if strings.TrimSpace(apiBase) == "" {
		return false
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := getJSON(strings.TrimRight(strings.TrimSpace(apiBase), "/")+"/models", &resp, apiKey); err != nil {
		return false
	}

	for _, model := range resp.Data {
		if strings.EqualFold(strings.TrimSpace(model.ID), modelID) {
			return true
		}
	}
	return false
}

func getJSON(rawURL string, out any, apiKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), modelProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func apiRootFromAPIBase(raw string) (string, error) {
	u, err := parseAPIBase(raw)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String(), nil
}

func hostPortFromAPIBase(raw string) (string, error) {
	u, err := parseAPIBase(raw)
	if err != nil {
		return "", err
	}

	if port := u.Port(); port != "" {
		return u.Host, nil
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return net.JoinHostPort(u.Hostname(), "443"), nil
	default:
		return net.JoinHostPort(u.Hostname(), "80"), nil
	}
}

func parseAPIBase(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty api base")
	}

	u, err := url.Parse(raw)
	if err == nil && u.Hostname() != "" {
		return u, nil
	}

	u, err = url.Parse("//" + raw)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid api base %q", raw)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u, nil
}

func ollamaModelMatches(candidate, want string) bool {
	candidate = strings.TrimSpace(candidate)
	want = strings.TrimSpace(want)
	if candidate == "" || want == "" {
		return false
	}

	candidateBase, candidateTag := splitOllamaModel(candidate)
	wantBase, wantTag := splitOllamaModel(want)
	if candidateBase == "" || wantBase == "" {
		return false
	}

	if candidateTag == "" {
		candidateTag = "latest"
	}
	if wantTag == "" {
		wantTag = "latest"
	}

	return strings.EqualFold(candidateBase, wantBase) && strings.EqualFold(candidateTag, wantTag)
}

func splitOllamaModel(raw string) (base, tag string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	base, tag, _ = strings.Cut(raw, ":")
	return strings.TrimSpace(base), strings.TrimSpace(tag)
}
