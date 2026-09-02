package repoaudit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type purgePhaseFixture struct {
	store      Store
	automation RepositoryReviewAutomation
	state      RepositoryState
	intent     repositoryReviewPurgeIntent
}

func newPurgePhaseFixture(
	t *testing.T,
	phase repositoryReviewPurgePhase,
	resetAutomation bool,
) purgePhaseFixture {
	t.Helper()
	store := newAutomationTestStore(t)
	id := "rra_phase_fixture_" + strings.ReplaceAll(string(phase), "_", "")
	automation := createAutomationForTest(t, store, id, string(phase))
	state := createPurgeTestLedger(t, store, automation.Repository)
	intent := purgeTestIntent(automation, state)
	intent.Phase = phase
	if resetAutomation {
		reset := cloneAutomation(automation)
		resetRepositoryReviewAutomationHistory(&reset)
		reset.Version++
		reset.UpdatedAt = automationTestNow
		if err := store.saveAutomation(reset); err != nil {
			t.Fatal(err)
		}
	}
	return purgePhaseFixture{store: store, automation: automation, state: state, intent: intent}
}

func TestRepositoryReviewPurgePhaseFailureCoverage(t *testing.T) {
	t.Run("prepared blocked", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		running := cloneAutomation(fixture.automation)
		running.Status = RepositoryReviewAutomationRunning
		running.ActiveRunID = "wr_phase_running"
		running.RunIDs = []string{running.ActiveRunID}
		running.Version++
		running.UpdatedAt = automationTestNow
		if err := fixture.store.saveAutomation(running); err != nil {
			t.Fatal(err)
		}
		fixture.intent.ExpectedAutomationVersion = running.Version
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrRepositoryReviewPurgeBlocked) {
			t.Fatalf("blocked prepared phase error = %v", err)
		}
	})

	t.Run("prepared phase save failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		if err := os.Mkdir(fixture.store.purgeAutomationIntentPath(fixture.automation.ID), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("prepared phase ignored marker write failure")
		}
	})

	t.Run("automation committing apply failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationCommitting, false)
		drifted := cloneAutomation(fixture.automation)
		drifted.Version += 2
		drifted.UpdatedAt = automationTestNow
		if err := fixture.store.saveAutomation(drifted); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("automation phase drift error = %v", err)
		}
	})

	t.Run("automation committing phase save failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationCommitting, false)
		if err := os.Mkdir(fixture.store.purgeAutomationIntentPath(fixture.automation.ID), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("automation phase ignored marker write failure")
		}
	})

	t.Run("automation applied verification failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationApplied, false)
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("unapplied automation error = %v", err)
		}
	})

	t.Run("automation applied ledger load failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationApplied, true)
		sentinel := errors.New("injected purge ledger load failure")
		fixture.store.loadForTest = func(string) (RepositoryState, error) {
			return RepositoryState{}, sentinel
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, sentinel) {
			t.Fatalf("ledger load error = %v", err)
		}
	})

	t.Run("automation applied ledger drift", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationApplied, true)
		fixture.state.Version++
		if err := fixture.store.save(&fixture.state); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("automation-applied ledger drift error = %v", err)
		}
	})

	t.Run("automation applied phase save failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeAutomationApplied, true)
		if err := os.Mkdir(fixture.store.purgeAutomationIntentPath(fixture.automation.ID), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("automation-applied phase ignored marker write failure")
		}
	})

	t.Run("ledger committing verification failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerCommitting, false)
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("ledger phase automation error = %v", err)
		}
	})

	t.Run("ledger committing version drift", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerCommitting, true)
		fixture.state.Version++
		if err := fixture.store.save(&fixture.state); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("ledger phase version error = %v", err)
		}
	})

	t.Run("ledger committing delete failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerCommitting, true)
		summaryPath := strings.TrimSuffix(fixture.store.path(fixture.state.Repository), ".json") + ".summary.json"
		if err := os.Remove(summaryPath); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err := os.Mkdir(summaryPath, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(summaryPath, "block"), []byte("block"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("ledger phase ignored delete failure")
		}
	})

	t.Run("ledger committing injected delete failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerCommitting, true)
		sentinel := errors.New("injected ledger deletion failure")
		original := repositoryReviewPurgeRemoveLedger
		repositoryReviewPurgeRemoveLedger = func(
			Store,
			[]repositoryReviewPurgeLedgerTarget,
		) error {
			return sentinel
		}
		t.Cleanup(func() { repositoryReviewPurgeRemoveLedger = original })
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, sentinel) {
			t.Fatalf("injected ledger delete error = %v", err)
		}
	})

	t.Run("ledger committing phase save failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerCommitting, true)
		if err := os.Mkdir(fixture.store.purgeAutomationIntentPath(fixture.automation.ID), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("ledger phase ignored marker write failure")
		}
	})

	t.Run("ledger removed still exists", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerRemoved, true)
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, ErrConflict) {
			t.Fatalf("ledger-removed retained state error = %v", err)
		}
	})

	t.Run("ledger removed load failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerRemoved, true)
		sentinel := errors.New("injected removed-ledger load failure")
		fixture.store.loadForTest = func(string) (RepositoryState, error) {
			return RepositoryState{}, sentinel
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); !errors.Is(err, sentinel) {
			t.Fatalf("removed-ledger load error = %v", err)
		}
	})

	t.Run("ledger removed intent cleanup failure", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerRemoved, true)
		if err := fixture.store.removeRepositoryReviewLedger(fixture.state.Repository); err != nil {
			t.Fatal(err)
		}
		fence := fixture.store.purgeRepositoryFencePath(fixture.state.Repository)
		if err := os.Mkdir(fence, 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("ledger-removed phase ignored intent cleanup failure")
		}
	})

	t.Run("ledger removed reset configuration missing", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgeLedgerRemoved, true)
		if err := fixture.store.removeRepositoryReviewLedger(fixture.state.Repository); err != nil {
			t.Fatal(err)
		}
		if err := removeRepositoryReviewRegularFile(
			fixture.store.automationPath(fixture.automation.ID),
		); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.store.applyPurgeIntent(fixture.intent); err == nil {
			t.Fatal("ledger-removed reset accepted missing configuration")
		}
	})
}

func TestRepositoryReviewPurgeStoreFailureCoverage(t *testing.T) {
	t.Run("lock failure", func(t *testing.T) {
		workspace := t.TempDir()
		store := NewStore(workspace)
		if err := os.Mkdir(store.root+".lock", 0o700); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.PurgeAutomationHistory(
			context.Background(), "rra_lock_failure", 1, 0, purgeTestFence(), "owner/repo",
		); err == nil {
			t.Fatal("purge ignored lock failure")
		}
	})

	t.Run("reconcile lock failure", func(t *testing.T) {
		workspace := t.TempDir()
		store := NewStore(workspace)
		if err := os.Mkdir(store.root+".lock", 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReconcilePurgeIntents(context.Background()); err == nil {
			t.Fatal("reconcile ignored lock failure")
		}
	})

	t.Run("post-lock cancellation", func(t *testing.T) {
		store := newAutomationTestStore(t)
		ctx := &repositoryReviewCancelAfterFirstContext{first: make(chan struct{})}
		ctx.canceled.Store(true)
		if _, _, err := store.PurgeAutomationHistory(
			ctx, "rra_cancel_after_lock", 1, 0, purgeTestFence(), "owner/repo",
		); !errors.Is(err, context.Canceled) {
			t.Fatalf("post-lock cancellation error = %v", err)
		}
	})

	t.Run("corrupt automation", func(t *testing.T) {
		store := newAutomationTestStore(t)
		if err := os.MkdirAll(store.root, 0o700); err != nil {
			t.Fatal(err)
		}
		id := "rra_corrupt_purge"
		if err := os.WriteFile(store.automationPath(id), []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.PurgeAutomationHistory(
			context.Background(), id, 1, 0, purgeTestFence(), "owner/repo",
		); err == nil {
			t.Fatal("purge accepted corrupt automation")
		}
	})

	t.Run("scope resolution failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_resolve_failure", "resolve")
		sentinel := errors.New("injected identity resolution failure")
		store.loadForTest = func(string) (RepositoryState, error) {
			return RepositoryState{}, sentinel
		}
		if _, _, err := store.PurgeAutomationHistory(
			context.Background(), automation.ID, automation.Version, 0,
			purgeTestFence(), automation.Repository,
		); !errors.Is(err, sentinel) {
			t.Fatalf("scope resolution error = %v", err)
		}
	})

	t.Run("initial marker write failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_marker_failure", "marker")
		state := createPurgeTestLedger(t, store, automation.Repository)
		if err := os.Mkdir(store.purgeAutomationIntentPath(automation.ID), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.PurgeAutomationHistory(
			context.Background(), automation.ID, automation.Version, state.Version,
			purgeTestFence(state), automation.Repository,
		); err == nil {
			t.Fatal("purge ignored initial marker write failure")
		}
	})

	t.Run("initial repository fence write failure", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_fence_failure", "fence")
		state := createPurgeTestLedger(t, store, automation.Repository)
		if err := os.Mkdir(store.purgeRepositoryFencePath(state.Repository), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.PurgeAutomationHistory(
			context.Background(), automation.ID, automation.Version, state.Version,
			purgeTestFence(state), automation.Repository,
		); err == nil {
			t.Fatal("purge ignored repository-fence write failure")
		}
	})

	t.Run("reconcile unsafe root", func(t *testing.T) {
		workspace := t.TempDir()
		store := NewStore(workspace)
		if err := os.WriteFile(store.root, []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReconcilePurgeIntents(context.Background()); err == nil {
			t.Fatal("reconcile accepted unsafe root")
		}
	})
}

func TestRepositoryReviewPurgeInjectedIOCoverage(t *testing.T) {
	t.Run("reconcile read directory", func(t *testing.T) {
		store := newAutomationTestStore(t)
		sentinel := errors.New("injected purge readdir failure")
		original := repositoryReviewPurgeReadDir
		repositoryReviewPurgeReadDir = func(string) ([]os.DirEntry, error) { return nil, sentinel }
		t.Cleanup(func() { repositoryReviewPurgeReadDir = original })
		if _, err := store.ReconcilePurgeIntents(context.Background()); !errors.Is(err, sentinel) {
			t.Fatalf("reconcile readdir error = %v", err)
		}
	})

	t.Run("intent catalog limit", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		if err := fixture.store.savePurgeIntent(fixture.intent); err != nil {
			t.Fatal(err)
		}
		original := repositoryReviewPurgeIntentLimit
		repositoryReviewPurgeIntentLimit = 0
		t.Cleanup(func() { repositoryReviewPurgeIntentLimit = original })
		if _, err := fixture.store.ReconcilePurgeIntents(context.Background()); err == nil {
			t.Fatal("reconcile accepted oversized intent catalog")
		}
	})

	t.Run("reconcile cancellation inside catalog", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		if err := fixture.store.savePurgeIntent(fixture.intent); err != nil {
			t.Fatal(err)
		}
		ctx := &repositoryReviewCancelAfterFirstContext{first: make(chan struct{})}
		ctx.canceled.Store(true)
		if _, err := fixture.store.ReconcilePurgeIntents(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("catalog cancellation error = %v", err)
		}
	})

	t.Run("intent disappears during reconcile", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		if err := fixture.store.savePurgeIntent(fixture.intent); err != nil {
			t.Fatal(err)
		}
		primary := fixture.store.purgeAutomationIntentPath(fixture.automation.ID)
		original := repositoryReviewPurgeLstat
		repositoryReviewPurgeLstat = func(path string) (os.FileInfo, error) {
			if path == primary {
				return nil, os.ErrNotExist
			}
			return os.Lstat(path)
		}
		t.Cleanup(func() { repositoryReviewPurgeLstat = original })
		if count, err := fixture.store.ReconcilePurgeIntents(context.Background()); err != nil || count != 0 {
			t.Fatalf("disappeared intent count=%d err=%v", count, err)
		}
	})

	t.Run("intent stat", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		sentinel := errors.New("injected purge lstat failure")
		original := repositoryReviewPurgeLstat
		repositoryReviewPurgeLstat = func(string) (os.FileInfo, error) { return nil, sentinel }
		t.Cleanup(func() { repositoryReviewPurgeLstat = original })
		if err := fixture.store.savePurgeIntent(fixture.intent); !errors.Is(err, sentinel) {
			t.Fatalf("intent stat error = %v", err)
		}
	})

	t.Run("intent write", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		sentinel := errors.New("injected purge write failure")
		original := repositoryReviewPurgeWriteFileAtomic
		repositoryReviewPurgeWriteFileAtomic = func(string, []byte, os.FileMode) error { return sentinel }
		t.Cleanup(func() { repositoryReviewPurgeWriteFileAtomic = original })
		if err := fixture.store.savePurgeIntent(fixture.intent); !errors.Is(err, sentinel) {
			t.Fatalf("intent write error = %v", err)
		}
	})

	t.Run("intent read", func(t *testing.T) {
		fixture := newPurgePhaseFixture(t, repositoryReviewPurgePrepared, false)
		if err := fixture.store.savePurgeIntent(fixture.intent); err != nil {
			t.Fatal(err)
		}
		sentinel := errors.New("injected purge read failure")
		original := repositoryReviewPurgeReadFile
		repositoryReviewPurgeReadFile = func(string) ([]byte, error) { return nil, sentinel }
		t.Cleanup(func() { repositoryReviewPurgeReadFile = original })
		if _, _, err := fixture.store.loadPurgeIntentForAutomation(fixture.automation.ID); !errors.Is(err, sentinel) {
			t.Fatalf("intent read error = %v", err)
		}
	})

	t.Run("remove stat", func(t *testing.T) {
		sentinel := errors.New("injected purge remove stat failure")
		original := repositoryReviewPurgeLstat
		repositoryReviewPurgeLstat = func(string) (os.FileInfo, error) { return nil, sentinel }
		t.Cleanup(func() { repositoryReviewPurgeLstat = original })
		if err := removeRepositoryReviewRegularFile("ignored"); !errors.Is(err, sentinel) {
			t.Fatalf("remove stat error = %v", err)
		}
	})
}
