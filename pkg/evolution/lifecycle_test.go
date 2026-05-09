package evolution_test

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func TestStore_SaveAndLoadProfile(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	profile := evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    root,
		CurrentVersion: "v2",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "weather lookup helper",
		LastUsedAt:     time.Unix(1700000000, 0).UTC(),
		UseCount:       3,
		RetentionScore: 0.8,
		VersionHistory: []evolution.SkillVersionEntry{
			{
				Version:   "v1",
				Action:    "create",
				Timestamp: time.Unix(1699990000, 0).UTC(),
				Summary:   "initial learned version",
			},
		},
	}

	if err := store.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	loaded, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if loaded.SkillName != "weather" {
		t.Fatalf("SkillName = %q, want weather", loaded.SkillName)
	}
	if loaded.Status != evolution.SkillStatusActive {
		t.Fatalf("Status = %q, want %q", loaded.Status, evolution.SkillStatusActive)
	}
	if len(loaded.VersionHistory) != 1 {
		t.Fatalf("len(VersionHistory) = %d, want 1", len(loaded.VersionHistory))
	}
}

func TestNextLifecycleState_ActiveToCold(t *testing.T) {
	now := time.Now().UTC()
	profile := evolution.SkillProfile{
		SkillName:      "release-flow",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		LastUsedAt:     now.AddDate(0, -6, 0),
		RetentionScore: 0.1,
	}

	got := evolution.NextLifecycleState(profile, now)
	if got != evolution.SkillStatusCold {
		t.Fatalf("NextLifecycleState = %q, want %q", got, evolution.SkillStatusCold)
	}
}

func TestNextLifecycleState_ManualSkillStaysActive(t *testing.T) {
	now := time.Now().UTC()
	profile := evolution.SkillProfile{
		SkillName:      "manual-weather",
		Status:         evolution.SkillStatusActive,
		Origin:         "manual",
		LastUsedAt:     now.AddDate(-1, 0, 0),
		RetentionScore: 0,
	}

	got := evolution.NextLifecycleState(profile, now)
	if got != evolution.SkillStatusActive {
		t.Fatalf("NextLifecycleState = %q, want %q", got, evolution.SkillStatusActive)
	}
}

func TestStore_SaveProfileRejectsInvalidSkillName(t *testing.T) {
	store := evolution.NewStore(evolution.NewPaths(t.TempDir(), ""))

	err := store.SaveProfile(evolution.SkillProfile{SkillName: "../escape"})
	if err == nil {
		t.Fatal("expected SaveProfile to reject invalid skill name")
	}
}

func TestStore_LoadProfileRejectsInvalidSkillName(t *testing.T) {
	store := evolution.NewStore(evolution.NewPaths(t.TempDir(), ""))

	_, err := store.LoadProfile("/tmp/escape")
	if err == nil {
		t.Fatal("expected LoadProfile to reject invalid skill name")
	}
}

func TestStore_SharedStateProfilesRemainIsolatedPerWorkspace(t *testing.T) {
	sharedState := t.TempDir()
	workspaceA := t.TempDir()
	workspaceB := t.TempDir()

	storeA := evolution.NewStore(evolution.NewPaths(workspaceA, sharedState))
	storeB := evolution.NewStore(evolution.NewPaths(workspaceB, sharedState))

	profileA := evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspaceA,
		CurrentVersion: "v-a",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "workspace A weather helper",
		LastUsedAt:     time.Unix(1700000000, 0).UTC(),
		UseCount:       2,
		RetentionScore: 0.6,
	}
	profileB := evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspaceB,
		CurrentVersion: "v-b",
		Status:         evolution.SkillStatusCold,
		Origin:         "manual",
		HumanSummary:   "workspace B weather helper",
		LastUsedAt:     time.Unix(1700000500, 0).UTC(),
		UseCount:       9,
		RetentionScore: 0.2,
	}

	if err := storeA.SaveProfile(profileA); err != nil {
		t.Fatalf("storeA.SaveProfile: %v", err)
	}
	if err := storeB.SaveProfile(profileB); err != nil {
		t.Fatalf("storeB.SaveProfile: %v", err)
	}

	loadedA, err := storeA.LoadProfile("weather")
	if err != nil {
		t.Fatalf("storeA.LoadProfile: %v", err)
	}
	if loadedA.WorkspaceID != workspaceA {
		t.Fatalf("storeA workspace = %q, want %q", loadedA.WorkspaceID, workspaceA)
	}
	if loadedA.CurrentVersion != "v-a" {
		t.Fatalf("storeA CurrentVersion = %q, want v-a", loadedA.CurrentVersion)
	}

	loadedB, err := storeB.LoadProfile("weather")
	if err != nil {
		t.Fatalf("storeB.LoadProfile: %v", err)
	}
	if loadedB.WorkspaceID != workspaceB {
		t.Fatalf("storeB workspace = %q, want %q", loadedB.WorkspaceID, workspaceB)
	}
	if loadedB.CurrentVersion != "v-b" {
		t.Fatalf("storeB CurrentVersion = %q, want v-b", loadedB.CurrentVersion)
	}

	allProfiles, err := storeA.LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}
	if len(allProfiles) != 2 {
		t.Fatalf("len(LoadProfiles()) = %d, want 2", len(allProfiles))
	}
}

func TestStore_LoadProfileDoesNotBorrowAnotherWorkspaceProfile(t *testing.T) {
	sharedState := t.TempDir()
	workspaceA := t.TempDir()
	workspaceB := t.TempDir()

	storeA := evolution.NewStore(evolution.NewPaths(workspaceA, sharedState))
	storeB := evolution.NewStore(evolution.NewPaths(workspaceB, sharedState))

	if err := storeA.SaveProfile(evolution.SkillProfile{
		SkillName:      "weather",
		WorkspaceID:    workspaceA,
		CurrentVersion: "v-a",
		Status:         evolution.SkillStatusActive,
		Origin:         "evolved",
		HumanSummary:   "workspace A weather helper",
		LastUsedAt:     time.Unix(1700000000, 0).UTC(),
		UseCount:       4,
		RetentionScore: 0.8,
	}); err != nil {
		t.Fatalf("storeA.SaveProfile: %v", err)
	}

	_, err := storeB.LoadProfile("weather")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("storeB.LoadProfile should not borrow workspace A profile, got err=%v", err)
	}
}

func TestStore_UpdateProfileIsAtomicPerWorkspaceSkill(t *testing.T) {
	root := t.TempDir()
	store := evolution.NewStore(evolution.NewPaths(root, ""))

	const workers = 64
	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.UpdateProfile(root, "weather", func(profile *evolution.SkillProfile, exists bool) error {
				if !exists {
					*profile = evolution.SkillProfile{
						SkillName:      "weather",
						WorkspaceID:    root,
						Status:         evolution.SkillStatusActive,
						Origin:         "manual",
						HumanSummary:   "weather",
						RetentionScore: 0.2,
					}
				}
				profile.UseCount++
				return nil
			})
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
	}

	profile, err := store.LoadProfile("weather")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.UseCount != workers {
		t.Fatalf("UseCount = %d, want %d", profile.UseCount, workers)
	}
}
