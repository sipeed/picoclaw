package skills

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/utils"
)

// mockRegistry is a test double implementing SkillRegistry.
type mockRegistry struct {
	name          string
	searchResults []SearchResult
	searchErr     error
	meta          *SkillMeta
	metaErr       error
	installResult *InstallResult
	installErr    error
}

func (m *mockRegistry) Name() string { return m.name }

func (m *mockRegistry) Search(_ context.Context, _ string, _ int) ([]SearchResult, error) {
	return m.searchResults, m.searchErr
}

func (m *mockRegistry) GetSkillMeta(_ context.Context, _ string) (*SkillMeta, error) {
	return m.meta, m.metaErr
}

func (m *mockRegistry) DownloadAndInstall(_ context.Context, _, _, _ string) (*InstallResult, error) {
	return m.installResult, m.installErr
}

func TestRegistryManagerSearchAllSingle(t *testing.T) {
	mgr := NewRegistryManager()
	mgr.AddRegistry(&mockRegistry{
		name: "test",
		searchResults: []SearchResult{
			{Slug: "skill-a", Score: 0.9, RegistryName: "test"},
			{Slug: "skill-b", Score: 0.5, RegistryName: "test"},
		},
	})

	results, err := mgr.SearchAll(context.Background(), "test query", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "skill-a", results[0].Slug)
}

func TestRegistryManagerSearchAllMultiple(t *testing.T) {
	mgr := NewRegistryManager()
	mgr.AddRegistry(&mockRegistry{
		name: "alpha",
		searchResults: []SearchResult{
			{Slug: "skill-a", Score: 0.8, RegistryName: "alpha"},
		},
	})
	mgr.AddRegistry(&mockRegistry{
		name: "beta",
		searchResults: []SearchResult{
			{Slug: "skill-b", Score: 0.95, RegistryName: "beta"},
		},
	})

	results, err := mgr.SearchAll(context.Background(), "test query", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	// Should be sorted by score descending
	assert.Equal(t, "skill-b", results[0].Slug)
	assert.Equal(t, "skill-a", results[1].Slug)
}

func TestRegistryManagerSearchAllOneFailsGracefully(t *testing.T) {
	mgr := NewRegistryManager()
	mgr.AddRegistry(&mockRegistry{
		name:      "failing",
		searchErr: fmt.Errorf("network error"),
	})
	mgr.AddRegistry(&mockRegistry{
		name: "working",
		searchResults: []SearchResult{
			{Slug: "skill-a", Score: 0.8, RegistryName: "working"},
		},
	})

	results, err := mgr.SearchAll(context.Background(), "test query", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "skill-a", results[0].Slug)
}

func TestRegistryManagerSearchAllAllFail(t *testing.T) {
	mgr := NewRegistryManager()
	mgr.AddRegistry(&mockRegistry{
		name:      "fail-1",
		searchErr: fmt.Errorf("error 1"),
	})

	_, err := mgr.SearchAll(context.Background(), "test query", 10)
	assert.Error(t, err)
}

func TestRegistryManagerSearchAllNoRegistries(t *testing.T) {
	mgr := NewRegistryManager()
	_, err := mgr.SearchAll(context.Background(), "test query", 10)
	assert.Error(t, err)
}

func TestRegistryManagerGetRegistry(t *testing.T) {
	mgr := NewRegistryManager()
	mock := &mockRegistry{name: "clawhub"}
	mgr.AddRegistry(mock)

	got := mgr.GetRegistry("clawhub")
	assert.NotNil(t, got)
	assert.Equal(t, "clawhub", got.Name())

	got = mgr.GetRegistry("nonexistent")
	assert.Nil(t, got)
}

func TestRegistryManagerSearchAllRespectLimit(t *testing.T) {
	mgr := NewRegistryManager()
	results := make([]SearchResult, 20)
	for i := range results {
		results[i] = SearchResult{Slug: fmt.Sprintf("skill-%d", i), Score: float64(20 - i)}
	}
	mgr.AddRegistry(&mockRegistry{
		name:          "test",
		searchResults: results,
	})

	got, err := mgr.SearchAll(context.Background(), "test", 5)
	assert.NoError(t, err)
	assert.Len(t, got, 5)
	// Top scores first
	assert.Equal(t, "skill-0", got[0].Slug)
}

func TestRegistryManagerSearchAllTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(5 * time.Millisecond) // Let context expire.

	mgr := NewRegistryManager()
	mgr.AddRegistry(&mockRegistry{
		name:      "slow",
		searchErr: fmt.Errorf("context deadline exceeded"),
	})

	_, err := mgr.SearchAll(ctx, "test", 5)
	assert.Error(t, err)
}

func TestSortByScoreDesc(t *testing.T) {
	results := []SearchResult{
		{Slug: "c", Score: 0.3},
		{Slug: "a", Score: 0.9},
		{Slug: "b", Score: 0.5},
	}
	sortByScoreDesc(results)
	assert.Equal(t, "a", results[0].Slug)
	assert.Equal(t, "b", results[1].Slug)
	assert.Equal(t, "c", results[2].Slug)
}

func TestIsSafeSlug(t *testing.T) {
	assert.NoError(t, utils.ValidateSkillIdentifier("github"))
	assert.NoError(t, utils.ValidateSkillIdentifier("docker-compose"))
	assert.Error(t, utils.ValidateSkillIdentifier(""))
	assert.Error(t, utils.ValidateSkillIdentifier("../etc/passwd"))
	assert.Error(t, utils.ValidateSkillIdentifier("path/traversal"))
	assert.Error(t, utils.ValidateSkillIdentifier("path\\traversal"))
}
