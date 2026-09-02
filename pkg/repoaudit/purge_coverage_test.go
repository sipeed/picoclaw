package repoaudit

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryReviewPurgeRequestBoundaryCoverage(t *testing.T) {
	store := newAutomationTestStore(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := store.PurgeAutomationHistory(canceled, "rra_missing", 1, 0, purgeTestFence(), "owner/repo"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled purge error = %v", err)
	}
	if _, _, err := store.PurgeAutomationHistory(context.Background(), "bad", 0, -1, purgeTestFence(), ""); !errors.Is(err, ErrInvalidAutomation) {
		t.Fatalf("invalid purge error = %v", err)
	}
	if _, _, err := store.PurgeAutomationHistory(
		context.Background(), "rra_missing", 1, 0, purgeTestFence(), "owner/repo",
	); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing purge error = %v", err)
	}

	empty := createAutomationForTest(t, store, "rra_empty_purge", "empty")
	if _, _, err := store.PurgeAutomationHistory(
		context.Background(), empty.ID, empty.Version, 0, purgeTestFence(), empty.Repository,
	); !errors.Is(err, ErrRepositoryReviewHistoryAbsent) {
		t.Fatalf("empty purge error = %v", err)
	}

	runningInput := validAutomationForTest("rra_running_remove", "running")
	runningInput.Status = RepositoryReviewAutomationRunning
	runningInput.ActiveRunID = "wr_running_remove"
	runningInput.RunIDs = []string{runningInput.ActiveRunID}
	running, err := store.CreateAutomation(context.Background(), runningInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), running.ID, running.Version, 0, purgeTestFence(), running.Repository,
	); !errors.Is(err, ErrRepositoryReviewPurgeBlocked) {
		t.Fatalf("running remove error = %v", err)
	}

	withIntent := createAutomationForTest(t, store, "rra_existing_intent", "intent")
	state := createPurgeTestLedger(t, store, withIntent.Repository)
	intent := purgeTestIntent(withIntent, state)
	if err := store.savePurgeIntent(intent); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PurgeAutomationHistory(
		context.Background(), withIntent.ID, withIntent.Version, state.Version,
		purgeTestFence(state), withIntent.Repository,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("existing intent replay error = %v", err)
	}

	unboundStore := newAutomationTestStore(t)
	unbound := createAutomationForTest(t, unboundStore, "rra_existing_bad_intent", "bad intent")
	assigned := createPurgeTestLedger(t, unboundStore, unbound.Repository)
	other := createPurgeTestLedger(t, unboundStore, "other/existing-intent")
	badIntent := purgeTestIntent(unbound, other)
	if err := unboundStore.savePurgeIntent(badIntent); err != nil {
		t.Fatal(err)
	}
	if _, _, err := unboundStore.PurgeAutomationHistory(
		context.Background(), unbound.ID, unbound.Version, assigned.Version,
		purgeTestFence(assigned), unbound.Repository,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("invalid existing intent apply error = %v", err)
	}
}

func TestRepositoryReviewPurgeRecoveryBoundaryCoverage(t *testing.T) {
	missing := NewStore(t.TempDir())
	if count, err := missing.ReconcilePurgeIntents(context.Background()); err != nil || count != 0 {
		t.Fatalf("missing-root reconcile count=%d err=%v", count, err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := missing.ReconcilePurgeIntents(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reconcile error = %v", err)
	}

	store := newAutomationTestStore(t)
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		t.Fatal(err)
	}
	invalidPath := filepath.Join(store.root, "purge_automation_invalid.json")
	if err := os.WriteFile(invalidPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcilePurgeIntents(context.Background()); err == nil {
		t.Fatal("invalid recovery marker was accepted")
	}

	invalid := repositoryReviewPurgeIntent{}
	if _, err := store.applyPurgeIntent(invalid); err == nil {
		t.Fatal("invalid purge intent was applied")
	}
	if err := store.savePurgeIntent(invalid); err == nil {
		t.Fatal("invalid purge intent was saved")
	}
	if _, _, err := store.loadPurgeIntentForAutomation("bad"); !errors.Is(err, ErrInvalidAutomation) {
		t.Fatalf("invalid automation marker lookup error = %v", err)
	}
	if _, _, err := store.loadPurgeFence(""); !errors.Is(err, ErrInvalidAutomation) {
		t.Fatalf("invalid repository marker lookup error = %v", err)
	}
	if _, found, err := store.loadPurgeIntentForAutomation("rra_absent_marker"); err != nil || found {
		t.Fatalf("absent marker found=%v err=%v", found, err)
	}
}

func TestRepositoryReviewPurgePhaseVerificationCoverage(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_phase_coverage", "phase")
	state := createPurgeTestLedger(t, store, automation.Repository)
	intent := purgeTestIntent(automation, state)

	drifted := intent
	drifted.ExpectedAutomationVersion++
	if _, _, _, err := store.validatePreparedPurgeIntent(drifted); !errors.Is(err, ErrConflict) {
		t.Fatalf("automation drift error = %v", err)
	}
	drifted = intent
	drifted.ExpectedRepositoryVersion++
	if _, _, _, err := store.validatePreparedPurgeIntent(drifted); !errors.Is(err, ErrConflict) {
		t.Fatalf("ledger drift error = %v", err)
	}

	emptyInput := validAutomationForTest("rra_phase_empty", "empty phase")
	emptyInput.Repository = "owner/empty"
	missingLedger, err := store.CreateAutomation(context.Background(), emptyInput)
	if err != nil {
		t.Fatal(err)
	}
	emptyIntent := repositoryReviewPurgeIntent{
		SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
		Mode:          repositoryReviewPurgeRemove, Phase: repositoryReviewPurgePrepared,
		AutomationID: missingLedger.ID, ConfiguredRepository: missingLedger.Repository,
		Repository:                CanonicalRepositoryIdentity(missingLedger.Repository),
		ExpectedAutomationVersion: missingLedger.Version, CreatedAt: automationTestNow,
	}
	if _, _, found, err := store.validatePreparedPurgeIntent(emptyIntent); err != nil || found {
		t.Fatalf("empty prepared intent found=%v err=%v", found, err)
	}
	emptyIntent.ExpectedRepositoryVersion = 1
	if _, _, _, err := store.validatePreparedPurgeIntent(emptyIntent); !errors.Is(err, ErrConflict) {
		t.Fatalf("empty prepared version error = %v", err)
	}

	if err := store.verifyPurgeAutomationApplied(intent); !errors.Is(err, ErrConflict) {
		t.Fatalf("unapplied reset verification error = %v", err)
	}
	removeIntent := intent
	removeIntent.Mode = repositoryReviewPurgeRemove
	if err := store.verifyPurgeAutomationApplied(removeIntent); !errors.Is(err, ErrConflict) {
		t.Fatalf("unapplied removal verification error = %v", err)
	}

	wrongVersion := cloneAutomation(automation)
	wrongVersion.Version += 2
	if err := store.saveAutomation(wrongVersion); err != nil {
		t.Fatal(err)
	}
	if err := store.applyPurgeAutomationPhase(intent); !errors.Is(err, ErrConflict) {
		t.Fatalf("invalid reset version error = %v", err)
	}

	resetRepositoryReviewAutomationHistory(nil)
	if repositoryReviewPurgeBlockerMessage(RepositoryReviewPurgeBlockerCode("unknown")) == "" {
		t.Fatal("unknown blocker had no safe fallback")
	}
}

func TestRepositoryReviewPurgeHelperErrorCoverage(t *testing.T) {
	t.Run("prepared automation load", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_prepared_load", "prepared load")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := purgeTestIntent(automation, state)
		if err := os.WriteFile(store.automationPath(automation.ID), []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := store.validatePreparedPurgeIntent(intent); err == nil {
			t.Fatal("prepared validation accepted corrupt automation")
		}
		if err := store.applyPurgeAutomationPhase(intent); err == nil {
			t.Fatal("automation phase accepted corrupt automation")
		}
		if err := store.verifyPurgeAutomationApplied(intent); err == nil {
			t.Fatal("automation verification accepted corrupt automation")
		}
	})

	t.Run("prepared resolver load", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_prepared_resolve", "prepared resolve")
		intent := repositoryReviewPurgeIntent{
			SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
			Mode:          repositoryReviewPurgeRemove, Phase: repositoryReviewPurgePrepared,
			AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
			Repository: automation.Repository, ExpectedAutomationVersion: automation.Version,
			CreatedAt: automationTestNow,
		}
		sentinel := errors.New("prepared resolver failure")
		store.loadForTest = func(string) (RepositoryState, error) { return RepositoryState{}, sentinel }
		if _, _, _, err := store.validatePreparedPurgeIntent(intent); !errors.Is(err, sentinel) {
			t.Fatalf("prepared resolver error = %v", err)
		}
	})

	t.Run("remove assignment mismatch", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationCommitting, false)
		fixture.intent.Mode = repositoryReviewPurgeRemove
		fixture.intent.ConfiguredRepository = "owner/other"
		if err := fixture.store.applyPurgeAutomationPhase(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("remove assignment mismatch error = %v", err)
		}
	})

	t.Run("reset assignment missing", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationCommitting, false)
		if err := removeRepositoryReviewRegularFile(
			fixture.store.automationPath(fixture.automation.ID),
		); err != nil {
			t.Fatal(err)
		}
		if err := fixture.store.applyPurgeAutomationPhase(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("missing reset assignment error = %v", err)
		}
	})

	t.Run("summary removal failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		state := createPurgeTestLedger(t, store, "owner/remove-failure")
		summaryPath := strings.TrimSuffix(store.path(state.Repository), ".json") + ".summary.json"
		if err := os.Remove(summaryPath); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err := os.Mkdir(summaryPath, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(summaryPath, "block"), []byte("block"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := store.removeRepositoryReviewLedger(state.Repository); err == nil {
			t.Fatal("ledger removal accepted non-removable summary")
		}
	})

	t.Run("intent root failure", func(t *testing.T) {
		workspace := t.TempDir()
		store := NewStore(workspace)
		automation := validAutomationForTest("rra_intent_root", "intent root")
		automation.SchemaVersion = RepositoryReviewAutomationSchemaVersion
		automation.Version = 1
		automation.CreatedAt = automationTestNow
		automation.UpdatedAt = automationTestNow
		intent := repositoryReviewPurgeIntent{
			SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
			Mode:          repositoryReviewPurgeRemove, Phase: repositoryReviewPurgePrepared,
			AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
			Repository: automation.Repository, ExpectedAutomationVersion: 1,
			CreatedAt: automationTestNow,
		}
		if err := os.WriteFile(store.root, []byte("not a root"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := store.savePurgeIntent(intent); err == nil {
			t.Fatal("intent save accepted unsafe root")
		}
	})

	t.Run("run fallback unmatched and ambiguous", func(t *testing.T) {
		store := newAutomationTestStore(t)
		input := validAutomationForTest("rra_resolver_edges", "resolver edges")
		input.Repository = "owner/absent"
		input.RunIDs = []string{"wr_resolver_edge"}
		automation, err := store.CreateAutomation(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		unmatched := createPurgeTestLedger(t, store, "legacy/unmatched")
		unmatched.Runs = []ReviewRun{{ID: "wr_other"}}
		unmatched.Version++
		if err := store.save(&unmatched); err != nil {
			t.Fatal(err)
		}
		if _, found, err := store.resolveRepositoryStateIgnoringPurge(automation); err != nil || found {
			t.Fatalf("unmatched fallback found=%v err=%v", found, err)
		}
		for _, repository := range []string{"legacy/first", "legacy/second"} {
			state := createPurgeTestLedger(t, store, repository)
			state.Runs = []ReviewRun{{ID: "wr_resolver_edge"}}
			state.Version++
			if err := store.save(&state); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := store.resolveRepositoryStateIgnoringPurge(automation); err == nil ||
			!strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("ambiguous fallback error = %v", err)
		}
	})

	t.Run("run fallback catalog failure", func(t *testing.T) {
		workspace := t.TempDir()
		store := NewStore(workspace)
		store.loadForTest = func(repository string) (RepositoryState, error) {
			return RepositoryState{Repository: repository}, nil
		}
		if err := os.WriteFile(store.root, []byte("not a catalog"), 0o600); err != nil {
			t.Fatal(err)
		}
		automation := validAutomationForTest("rra_catalog_failure", "catalog failure")
		automation.Repository = "owner/absent"
		automation.RunIDs = []string{"wr_catalog_failure"}
		if _, _, err := store.resolveRepositoryStateIgnoringPurge(automation); err == nil {
			t.Fatal("run fallback ignored catalog failure")
		}
	})
}

func TestRepositoryReviewPurgeFileBoundaryCoverage(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	if err := removeRepositoryReviewRegularFile(missing); err != nil {
		t.Fatalf("remove missing error = %v", err)
	}
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := removeRepositoryReviewRegularFile(directory); err == nil {
		t.Fatal("purge accepted a directory")
	}
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err == nil {
		if err := removeRepositoryReviewRegularFile(link); err == nil {
			t.Fatal("purge accepted a symlink")
		}
	}

	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_trailing_marker", "trailing")
	state := createPurgeTestLedger(t, store, automation.Repository)
	intent := purgeTestIntent(automation, state)
	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := store.purgeAutomationIntentPath(automation.ID)
	if err := os.WriteFile(path, append(data, []byte(` {}`)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.loadPurgeIntentForAutomation(automation.ID); err == nil {
		t.Fatal("purge marker accepted trailing JSON")
	}

	badPath := filepath.Join(store.root, "purge_automation_other.json")
	if err := os.WriteFile(badPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.loadPurgeIntentPath(badPath); err == nil ||
		!strings.Contains(err.Error(), "path mismatch") {
		t.Fatalf("marker path mismatch error = %v", err)
	}
}
