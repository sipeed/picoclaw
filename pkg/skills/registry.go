package skills

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	defaultMaxConcurrentSearches = 2
)

// SearchResult represents a single result from a skill registry search.
type SearchResult struct {
	Score        float64 `json:"score"`
	Slug         string  `json:"slug"`
	DisplayName  string  `json:"display_name"`
	Summary      string  `json:"summary"`
	Version      string  `json:"version"`
	RegistryName string  `json:"registry_name"`
}

// SkillMeta holds metadata about a skill from a registry.
type SkillMeta struct {
	Slug             string `json:"slug"`
	DisplayName      string `json:"display_name"`
	Summary          string `json:"summary"`
	LatestVersion    string `json:"latest_version"`
	IsMalwareBlocked bool   `json:"is_malware_blocked"`
	IsSuspicious     bool   `json:"is_suspicious"`
	RegistryName     string `json:"registry_name"`
}

// InstallResult is returned by DownloadAndInstall to carry metadata
// back to the caller for moderation and user messaging.
type InstallResult struct {
	Version          string
	IsMalwareBlocked bool
	IsSuspicious     bool
	Summary          string
}

// SkillRegistry is the interface that all skill registries must implement.
// Each registry represents a different source of skills (e.g., clawhub.ai)
type SkillRegistry interface {
	// Name returns the unique name of this registry (e.g., "clawhub").
	Name() string
	// Search searches the registry for skills matching the query.
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	// GetSkillMeta retrieves metadata for a specific skill by slug.
	GetSkillMeta(ctx context.Context, slug string) (*SkillMeta, error)
	// DownloadAndInstall fetches metadata, resolves the version, downloads and
	// installs the skill to targetDir. Returns an InstallResult with metadata
	// for the caller to use for moderation and user messaging.
	DownloadAndInstall(ctx context.Context, slug, version, targetDir string) (*InstallResult, error)
}

// RegistryConfig holds configuration for all skill registries.
// This is the input to NewRegistryManagerFromConfig.
type RegistryConfig struct {
	ClawHub               ClawHubConfig
	MaxConcurrentSearches int
}

// ClawHubConfig configures the ClawHub registry.
type ClawHubConfig struct {
	Enabled         bool
	BaseURL         string
	AuthToken       string
	SearchPath      string // e.g. "/api/v1/search"
	SkillsPath      string // e.g. "/api/v1/skills"
	DownloadPath    string // e.g. "/api/v1/download"
	Timeout         int    // seconds, 0 = default (30s)
	MaxZipSize      int    // bytes, 0 = default (50MB)
	MaxResponseSize int    // bytes, 0 = default (2MB)
}

// RegistryManager coordinates multiple skill registries.
// It fans out search requests and routes installs to the correct registry.
type RegistryManager struct {
	registries    []SkillRegistry
	maxConcurrent int
	mu            sync.RWMutex
}

// NewRegistryManager creates an empty RegistryManager.
func NewRegistryManager() *RegistryManager {
	return &RegistryManager{
		registries:    make([]SkillRegistry, 0),
		maxConcurrent: defaultMaxConcurrentSearches,
	}
}

// NewRegistryManagerFromConfig builds a RegistryManager from config,
// instantiating only the enabled registries.
func NewRegistryManagerFromConfig(cfg RegistryConfig) *RegistryManager {
	rm := NewRegistryManager()
	if cfg.MaxConcurrentSearches > 0 {
		rm.maxConcurrent = cfg.MaxConcurrentSearches
	}
	if cfg.ClawHub.Enabled {
		rm.AddRegistry(NewClawHubRegistry(cfg.ClawHub))
	}
	return rm
}

// AddRegistry adds a registry to the manager.
func (rm *RegistryManager) AddRegistry(r SkillRegistry) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.registries = append(rm.registries, r)
}

// GetRegistry returns a registry by name, or nil if not found.
func (rm *RegistryManager) GetRegistry(name string) SkillRegistry {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	for _, r := range rm.registries {
		if r.Name() == name {
			return r
		}
	}
	return nil
}

// SearchAll fans out the query to all registries concurrently
// and merges results sorted by score descending.
func (rm *RegistryManager) SearchAll(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	rm.mu.RLock()
	regs := make([]SkillRegistry, len(rm.registries))
	copy(regs, rm.registries)
	rm.mu.RUnlock()

	if len(regs) == 0 {
		return nil, fmt.Errorf("no registries configured")
	}

	type regResult struct {
		results []SearchResult
		err     error
	}

	// Semaphore: limit concurrency.
	sem := make(chan struct{}, rm.maxConcurrent)
	resultsCh := make(chan regResult, len(regs))

	var wg sync.WaitGroup
	for _, reg := range regs {
		wg.Add(1)
		go func(r SkillRegistry) {
			defer wg.Done()

			// Acquire semaphore slot.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				resultsCh <- regResult{err: ctx.Err()}
				return
			}

			searchCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()

			results, err := r.Search(searchCtx, query, limit)
			if err != nil {
				slog.Warn("registry search failed", "registry", r.Name(), "error", err)
				resultsCh <- regResult{err: err}
				return
			}
			resultsCh <- regResult{results: results}
		}(reg)
	}

	// Close results channel after all goroutines complete.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var merged []SearchResult
	var lastErr error

	var anyRegistrySucceeded bool
	for rr := range resultsCh {
		if rr.err != nil {
			lastErr = rr.err
			continue
		}
		anyRegistrySucceeded = true
		merged = append(merged, rr.results...)
	}

	// If all registries failed, return the last error.
	if !anyRegistrySucceeded && lastErr != nil {
		return nil, fmt.Errorf("all registries failed: %w", lastErr)
	}

	// Sort by score descending.
	sortByScoreDesc(merged)

	// Clamp to limit.
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	return merged, nil
}

// sortByScoreDesc sorts SearchResults by Score in descending order (insertion sort — small slices).
func sortByScoreDesc(results []SearchResult) {
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		for j >= 0 && results[j].Score < key.Score {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}
