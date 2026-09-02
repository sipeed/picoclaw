package repoaudit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRepositoryReviewPurgeCompositeBoundaryCoverage(t *testing.T) {
	store := newAutomationTestStore(t)
	if _, _, err := store.PurgeAutomationHistory(
		context.Background(), "rra_missing_fence", 1, 0, " ", "owner/repo",
	); !errors.Is(err, ErrInvalidAutomation) {
		t.Fatalf("blank purge ledger fence error = %v", err)
	}
	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), "rra_missing_fence", 1, 0, "", "owner/repo",
	); !errors.Is(err, ErrInvalidAutomation) {
		t.Fatalf("blank removal ledger fence error = %v", err)
	}

	automation := validAutomationForTest("rra_empty_active_id", "empty active id")
	eligibility := EvaluateRepositoryReviewPurge(automation, RepositoryState{
		Repository:      automation.Repository,
		Version:         1,
		ActiveReviewRun: &RepositoryReviewActiveRun{},
	}, true)
	if eligibility.CanRemove || len(eligibility.Blockers) != 1 ||
		eligibility.Blockers[0].Code != RepositoryReviewPurgeBlockerReviewActive {
		t.Fatalf("empty-ID active run eligibility = %#v", eligibility)
	}
	if message := repositoryReviewPurgeBlockerMessage(RepositoryReviewPurgeBlockerRetentionUnavailable); message == "" {
		t.Fatal("retention-unavailable blocker has no message")
	}
}

func TestRepositoryReviewPurgeEligibilitySnapshotFailures(t *testing.T) {
	t.Run("lock failure", func(t *testing.T) {
		store := NewStore(t.TempDir())
		if err := os.Mkdir(store.root+".lock", 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := store.RepositoryReviewPurgeEligibilityForAutomation(
			validAutomationForTest("rra_eligibility_lock", "lock"),
		)
		if err == nil {
			t.Fatal("eligibility ignored lock failure")
		}
	})

	t.Run("automation load failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_eligibility_load", "load")
		if err := os.WriteFile(store.automationPath(automation.ID), []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation); err == nil {
			t.Fatal("eligibility accepted corrupt automation")
		}
	})

	t.Run("stale snapshot", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_eligibility_stale", "stale")
		automation.Version++
		if _, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation); !errors.Is(err, ErrConflict) {
			t.Fatalf("stale eligibility error = %v", err)
		}
	})

	t.Run("active purge fence", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_eligibility_fenced", "fenced")
		state := createPurgeTestLedger(t, store, automation.Repository)
		if err := store.savePurgeIntent(purgeTestIntent(automation, state)); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
			t.Fatalf("fenced eligibility error = %v", err)
		}
	})

	t.Run("corrupt purge fence", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_eligibility_bad_fence", "bad fence")
		if err := os.WriteFile(store.purgeRepositoryFencePath(automation.Repository), []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation); err == nil {
			t.Fatal("eligibility accepted corrupt purge fence")
		}
	})

	t.Run("inventory load failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_eligibility_inventory", "inventory")
		sentinel := errors.New("injected eligibility inventory failure")
		store.loadForTest = func(string) (RepositoryState, error) {
			return RepositoryState{}, sentinel
		}
		if _, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation); !errors.Is(err, sentinel) {
			t.Fatalf("inventory eligibility error = %v", err)
		}
	})
}

func TestRepositoryReviewPurgeFenceCatalogReadFailures(t *testing.T) {
	store := newAutomationTestStore(t)
	state := createPurgeTestLedger(t, store, "owner/corrupt-fence-catalog")
	if err := os.WriteFile(store.purgeRepositoryFencePath(state.Repository), []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListSummaries(); err == nil {
		t.Fatal("summary catalog accepted a corrupt purge fence")
	}
	if _, err := store.List(); err == nil {
		t.Fatal("state catalog accepted a corrupt purge fence")
	}
}

func TestRepositoryReviewCreateAutomationSaveFailureCoverage(t *testing.T) {
	store := newAutomationTestStore(t)
	input := validAutomationForTest("rra_save_failure_coverage", "save failure")
	store.now = func() time.Time {
		if err := os.MkdirAll(store.root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(store.automationPath(input.ID), 0o700); err != nil {
			t.Fatal(err)
		}
		return automationTestNow
	}
	if _, err := store.CreateAutomation(context.Background(), input); err == nil {
		t.Fatal("automation creation ignored its persistence failure")
	} else if !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("automation creation failed before persistence: %v", err)
	}
}

func TestRepositoryReviewPurgeCompositeIntentValidation(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_composite_intent", "composite intent")
	state := createPurgeTestLedger(t, store, automation.Repository)
	valid := purgeTestIntent(automation, state)

	tests := []struct {
		name   string
		mutate func(*repositoryReviewPurgeIntent)
	}{
		{
			name: "version without targets",
			mutate: func(intent *repositoryReviewPurgeIntent) {
				intent.LedgerTargets = nil
			},
		},
		{
			name: "duplicate target",
			mutate: func(intent *repositoryReviewPurgeIntent) {
				intent.LedgerTargets = append(intent.LedgerTargets, intent.LedgerTargets[0])
			},
		},
		{
			name: "missing primary target",
			mutate: func(intent *repositoryReviewPurgeIntent) {
				intent.LedgerTargets = []repositoryReviewPurgeLedgerTarget{{
					Repository: "owner/other", Version: intent.ExpectedRepositoryVersion,
				}}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intent := valid
			intent.LedgerTargets = append([]repositoryReviewPurgeLedgerTarget(nil), valid.LedgerTargets...)
			test.mutate(&intent)
			if err := validateRepositoryReviewPurgeIntent(intent); !errors.Is(err, ErrInvalidAutomation) {
				t.Fatalf("invalid composite intent error = %v", err)
			}
		})
	}
}

func TestRepositoryReviewPurgePreparedInventoryReadFailure(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_prepared_inventory", "prepared inventory")
	state := createPurgeTestLedger(t, store, automation.Repository)
	intent := purgeTestIntent(automation, state)
	sentinel := errors.New("injected second inventory read failure")
	loads := 0
	store.loadForTest = func(string) (RepositoryState, error) {
		loads++
		if loads > 1 {
			return RepositoryState{}, sentinel
		}
		return state, nil
	}
	if _, err := store.applyPurgeIntent(intent); !errors.Is(err, sentinel) {
		t.Fatalf("prepared inventory error = %v", err)
	}
}

func TestRepositoryReviewPurgeMultiLedgerRemovalStopsOnFailure(t *testing.T) {
	store := newAutomationTestStore(t)
	first := createPurgeTestLedger(t, store, "owner/aaa")
	second := createPurgeTestLedger(t, store, "owner/zzz")
	secondSummary := strings.TrimSuffix(store.path(second.Repository), ".json") + ".summary.json"
	if err := os.Remove(secondSummary); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if err := os.Mkdir(secondSummary, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondSummary, "block"), []byte("block"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := store.removeRepositoryReviewLedgers([]repositoryReviewPurgeLedgerTarget{
		{Repository: first.Repository, Version: first.Version},
		{Repository: second.Repository, Version: second.Version},
	})
	if err == nil {
		t.Fatal("multi-ledger removal ignored later target failure")
	}
	if _, statErr := os.Stat(store.path(first.Repository)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("first ledger stat error = %v", statErr)
	}
	if _, statErr := os.Stat(store.path(second.Repository)); statErr != nil {
		t.Fatalf("failed ledger stat error = %v", statErr)
	}
}

func TestRepositoryReviewPurgeIntentOwnershipFailureCoverage(t *testing.T) {
	t.Run("save rejects corrupt existing marker", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_save_corrupt_owner", "corrupt owner")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := purgeTestIntent(automation, state)
		if err := os.WriteFile(store.purgeAutomationIntentPath(automation.ID), []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := store.savePurgeIntent(intent); err == nil {
			t.Fatal("intent save replaced a corrupt ownership marker")
		}
	})

	t.Run("remove rejects different owner", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_remove_other_owner", "other owner")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := purgeTestIntent(automation, state)
		if err := store.savePurgeIntent(intent); err != nil {
			t.Fatal(err)
		}
		other := intent
		other.CreatedAt = other.CreatedAt.Add(time.Second)
		if err := store.removePurgeIntent(other); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
			t.Fatalf("different-owner cleanup error = %v", err)
		}
		if _, found, err := store.loadPurgeIntentForAutomation(automation.ID); err != nil || !found {
			t.Fatalf("owned intent found=%v err=%v", found, err)
		}
	})

	t.Run("remove preserves markers on IO failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_remove_io_failure", "remove IO")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := purgeTestIntent(automation, state)
		if err := store.savePurgeIntent(intent); err != nil {
			t.Fatal(err)
		}
		paths := store.purgeIntentPaths(intent)
		sentinel := errors.New("injected owned marker removal failure")
		original := repositoryReviewPurgeLstat
		calls := 0
		repositoryReviewPurgeLstat = func(path string) (os.FileInfo, error) {
			calls++
			if calls == len(paths)+1 {
				return nil, sentinel
			}
			return original(path)
		}
		t.Cleanup(func() { repositoryReviewPurgeLstat = original })
		if err := store.removePurgeIntent(intent); !errors.Is(err, sentinel) {
			t.Fatalf("owned marker removal error = %v", err)
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("marker %q stat error = %v", path, err)
			}
		}
	})

	if order := repositoryReviewPurgePhaseOrder(repositoryReviewPurgePhase("unknown")); order != 0 {
		t.Fatalf("unknown purge phase order = %d", order)
	}
}
