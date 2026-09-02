package repoaudit

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestRepositoryReviewPurgeEligibility(t *testing.T) {
	automation := validAutomationForTest("rra_purge_eligibility", "purge")
	automation.Status = RepositoryReviewAutomationCompleted
	state := RepositoryState{
		Version:              7,
		RawFindings:          make([]RawReviewFinding, 2),
		DeduplicatedFindings: make([]DeduplicatedReviewFinding, 1),
		RepositoryFindings: []RepositoryFinding{{
			Issue: RepositoryFindingIssueAssociation{
				URL: "https://github.com/owner/repo/issues/1",
				ConflictURLs: []string{
					"https://github.com/owner/repo/issues/1",
					"https://github.com/owner/repo/issues/2",
					"https://github.com/owner/repo/issues/3",
				},
			},
		}},
		IssueDrafts: make([]IssueDraft, 3),
	}
	eligible := EvaluateRepositoryReviewPurge(automation, state, true)
	if !eligible.CanPurge || !eligible.CanRemove || len(eligible.Blockers) != 0 ||
		eligible.Summary.RepositoryVersion != 7 || eligible.Summary.RawFindings != 2 ||
		eligible.Summary.DeduplicatedFindings != 1 || eligible.Summary.RepositoryFindings != 1 ||
		eligible.Summary.IssuePreviews != 3 || eligible.Summary.ExternalIssueAssociations != 3 {
		t.Fatalf("eligible purge = %#v", eligible)
	}

	state.DeduplicationJobs = []DeduplicationJob{{State: DeduplicationJobRunning}}
	state.MappingJobs = []RepositoryMappingJob{{State: RepositoryMappingRunning}}
	state.ValidationJobs = []RepositoryValidationJob{{State: RepositoryValidationRunning}}
	state.IssueDrafts = []IssueDraft{{State: IssueDraftGenerating}, {State: IssueDraftPublishing}}
	state.HistoricalDeduplication = HistoricalDeduplicationReplay{
		Required: true, Status: HistoricalDeduplicationReplaying,
	}
	automation.Status = RepositoryReviewAutomationRunning
	blocked := EvaluateRepositoryReviewPurge(automation, state, true)
	wantCodes := []RepositoryReviewPurgeBlockerCode{
		RepositoryReviewPurgeBlockerReviewActive,
		RepositoryReviewPurgeBlockerFindingProcessingActive,
		RepositoryReviewPurgeBlockerResolutionCheckActive,
		RepositoryReviewPurgeBlockerIssueGenerationActive,
		RepositoryReviewPurgeBlockerPublicationActive,
		RepositoryReviewPurgeBlockerHistoricalConsolidationActive,
	}
	if blocked.CanPurge || blocked.CanRemove || len(blocked.Blockers) != len(wantCodes) {
		t.Fatalf("blocked purge = %#v", blocked)
	}
	for index, code := range wantCodes {
		if blocked.Blockers[index].Code != code || blocked.Blockers[index].Count < 1 ||
			blocked.Blockers[index].Message == "" {
			t.Fatalf("blocker[%d] = %#v", index, blocked.Blockers[index])
		}
	}

	automation.Status = RepositoryReviewAutomationIdle
	missing := EvaluateRepositoryReviewPurge(automation, RepositoryState{}, false)
	if missing.CanPurge || !missing.CanRemove || len(missing.Blockers) != 0 {
		t.Fatalf("missing history eligibility = %#v", missing)
	}
	aggregated := evaluateRepositoryReviewPurgeInventory(
		automation,
		7,
		[]RepositoryState{
			{Repository: "owner/repo", Version: 7, RawFindings: make([]RawReviewFinding, 2)},
			{
				Repository: "https://github.com/Owner/Repo.git", Version: 3,
				RawFindings:     make([]RawReviewFinding, 4),
				ActiveReviewRun: &RepositoryReviewActiveRun{ID: "wr_secondary"},
			},
		},
	)
	if aggregated.CanPurge || aggregated.Summary.RawFindings != 6 ||
		aggregated.Summary.LedgerFence == "" || len(aggregated.Blockers) != 1 ||
		aggregated.Blockers[0].Code != RepositoryReviewPurgeBlockerReviewActive {
		t.Fatalf("aggregated eligibility = %#v", aggregated)
	}
}

func TestPurgeAutomationHistoryResetsRuntimeAndDeletesOnlyReviewLedger(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_purge_history", "purge")
	state := createPurgeTestLedger(t, store, automation.Repository)
	automation, err := store.UpdateAutomation(
		context.Background(), automation.ID, automation.Version,
		func(candidate *RepositoryReviewAutomation) error {
			candidate.Status = RepositoryReviewAutomationCompleted
			candidate.EffectiveAccountRef = "account-a"
			candidate.ResolvedCommitSHA = strings.Repeat("a", 40)
			candidate.ResolvedTargetBranch = "main"
			candidate.AdvertisedDefaultBranch = "main"
			candidate.TargetIsDefault = true
			candidate.RunIDs = []string{"wr_retained"}
			candidate.Usage = RepositoryReviewTokenUsage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5}
			candidate.EstimatedCostUSD = 1.25
			candidate.Progress = RepositoryReviewProgress{Stage: "complete", ReviewedFiles: 1}
			candidate.ModelStats = map[string]RepositoryReviewModelStats{"review-a": {Requests: 1}}
			candidate.AccountLimitSnapshots = []RepositoryReviewAccountLimitSnapshot{{
				AccountID: "account-a", Window: "weekly", CheckedAt: automationTestNow,
			}}
			candidate.StartedAt = automationTestNow.Add(-time.Hour)
			candidate.CompletedAt = automationTestNow
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	genericRunPath := store.root + ".workflow-run-canary"
	if err := os.WriteFile(genericRunPath, []byte("retained"), 0o600); err != nil {
		t.Fatal(err)
	}

	updated, eligibility, err := store.PurgeAutomationHistory(
		context.Background(), automation.ID, automation.Version, state.Version,
		purgeTestFence(state), automation.Repository,
	)
	if err != nil || !eligibility.CanPurge || !repositoryReviewAutomationHistoryReset(updated) ||
		updated.Version != automation.Version+1 || updated.ProfileID != automation.ProfileID ||
		updated.Repository != automation.Repository || updated.Ref != automation.Ref {
		t.Fatalf("purged automation=%#v eligibility=%#v err=%v", updated, eligibility, err)
	}
	if _, found, getErr := store.Get(automation.Repository); getErr != nil || found {
		t.Fatalf("ledger found=%v err=%v", found, getErr)
	}
	loaded, found, getErr := store.GetAutomation(context.Background(), automation.ID)
	if getErr != nil || !found || loaded.Version != updated.Version ||
		!repositoryReviewAutomationHistoryReset(loaded) {
		t.Fatalf("loaded reset automation=%#v found=%v err=%v", loaded, found, getErr)
	}
	if data, readErr := os.ReadFile(genericRunPath); readErr != nil || string(data) != "retained" {
		t.Fatalf("generic workflow canary data=%q err=%v", data, readErr)
	}
}

func TestDeleteAutomationAndHistoryRequiresFencesAndLeavesExternalSystemsUntouched(t *testing.T) {
	store := newAutomationTestStore(t)
	input := validAutomationForTest("rra_delete_history", "delete")
	input.Repository = "https://github.com/owner/repo.git"
	automation, err := store.CreateAutomation(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	ledgerRepository := CanonicalRepositoryIdentity(automation.Repository)
	state := createPurgeTestLedger(t, store, ledgerRepository)

	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), automation.ID, automation.Version, state.Version,
		purgeTestFence(state), "owner/other",
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("confirmation mismatch error = %v", err)
	}
	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), automation.ID, automation.Version, state.Version+1,
		purgeTestFence(state), automation.Repository,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("repository fence error = %v", err)
	}

	eligibility, err := store.DeleteAutomationAndHistory(
		context.Background(), automation.ID, automation.Version, state.Version,
		purgeTestFence(state), automation.Repository,
	)
	if err != nil || !eligibility.CanRemove || eligibility.Summary.ExternalIssueAssociations != 0 {
		t.Fatalf("delete eligibility=%#v err=%v", eligibility, err)
	}
	if _, found, getErr := store.GetAutomation(context.Background(), automation.ID); getErr != nil || found {
		t.Fatalf("automation found=%v err=%v", found, getErr)
	}
	if _, found, getErr := store.Get(ledgerRepository); getErr != nil || found {
		t.Fatalf("ledger found=%v err=%v", found, getErr)
	}
}

func TestRepositoryReviewPurgeUsesAuthoritativeLegacyLedgerIdentity(t *testing.T) {
	for _, test := range []struct {
		name       string
		repository string
		ledger     string
		runID      string
	}{
		{
			name: "original URL fallback", repository: "https://github.com/Owner/Repo.git",
			ledger: "https://github.com/Owner/Repo.git",
		},
		{
			name: "retained run fallback", repository: "owner/missing",
			ledger: "legacy/location", runID: "wr_legacy_purge",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newAutomationTestStore(t)
			id := "rra_legacy_" + strings.ToLower(strings.ReplaceAll(test.name, " ", "_"))
			input := validAutomationForTest(id, test.name)
			input.Repository = test.repository
			if test.runID != "" {
				input.RunIDs = []string{test.runID}
			}
			automation, err := store.CreateAutomation(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			state := createPurgeTestLedger(t, store, test.ledger)
			if test.runID != "" {
				state.Runs = []ReviewRun{{ID: test.runID}}
				state.Version++
				if err := store.save(&state); err != nil {
					t.Fatal(err)
				}
			}
			eligibility, err := store.DeleteAutomationAndHistory(
				context.Background(), automation.ID, automation.Version,
				state.Version, purgeTestFence(state), automation.Repository,
			)
			if err != nil || !eligibility.CanRemove {
				t.Fatalf("delete eligibility=%#v err=%v", eligibility, err)
			}
			if _, found, err := store.Get(test.ledger); err != nil || found {
				t.Fatalf("authoritative ledger found=%v err=%v", found, err)
			}
		})
	}
}

func TestRepositoryReviewPurgeDeletesAllConfiguredIdentityLedgersWithCompositeFence(t *testing.T) {
	store := newAutomationTestStore(t)
	input := validAutomationForTest("rra_alias_inventory", "aliases")
	input.Repository = "https://github.com/Owner/Repo.git"
	automation, err := store.CreateAutomation(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	canonical := createPurgeTestLedger(t, store, CanonicalRepositoryIdentity(automation.Repository))
	legacy := createPurgeTestLedger(t, store, automation.Repository)
	eligibility, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation)
	if err != nil || !eligibility.CanRemove || eligibility.Summary.RepositoryVersion != canonical.Version ||
		eligibility.Summary.LedgerFence != purgeTestFence(canonical, legacy) {
		t.Fatalf("alias eligibility=%#v err=%v", eligibility, err)
	}

	legacy.Version++
	legacy.UpdatedAt = automationTestNow
	if err := store.save(&legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), automation.ID, automation.Version, canonical.Version,
		eligibility.Summary.LedgerFence, automation.Repository,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale composite fence error = %v", err)
	}
	eligibility, err = store.RepositoryReviewPurgeEligibilityForAutomation(automation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), automation.ID, automation.Version, canonical.Version,
		eligibility.Summary.LedgerFence, automation.Repository,
	); err != nil {
		t.Fatal(err)
	}
	for _, repository := range []string{canonical.Repository, legacy.Repository} {
		if _, found, err := store.Get(repository); err != nil || found {
			t.Fatalf("alias ledger %q found=%v err=%v", repository, found, err)
		}
	}
}

func TestPurgeAutomationHistoryDeletesAllConfiguredIdentityLedgers(t *testing.T) {
	store := newAutomationTestStore(t)
	input := validAutomationForTest("rra_alias_history", "alias history")
	input.Repository = "https://github.com/Owner/Repo.git"
	automation, err := store.CreateAutomation(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	canonical := createPurgeTestLedger(t, store, CanonicalRepositoryIdentity(automation.Repository))
	legacy := createPurgeTestLedger(t, store, automation.Repository)
	eligibility, err := store.RepositoryReviewPurgeEligibilityForAutomation(automation)
	if err != nil {
		t.Fatal(err)
	}

	updated, applied, err := store.PurgeAutomationHistory(
		context.Background(), automation.ID, automation.Version, canonical.Version,
		eligibility.Summary.LedgerFence, automation.Repository,
	)
	if err != nil || !applied.CanPurge || !repositoryReviewAutomationHistoryReset(updated) {
		t.Fatalf("purged automation=%#v eligibility=%#v err=%v", updated, applied, err)
	}
	if updated.Version != automation.Version+1 || updated.Repository != automation.Repository {
		t.Fatalf("retained configuration=%#v", updated)
	}
	for _, repository := range []string{canonical.Repository, legacy.Repository} {
		if _, found, err := store.Get(repository); err != nil || found {
			t.Fatalf("alias ledger %q found=%v err=%v", repository, found, err)
		}
	}
	if _, found, err := store.ResolveRepositoryState(updated.Repository, updated.RunIDs); err != nil || found {
		t.Fatalf("reset automation resolved deleted history found=%v err=%v", found, err)
	}
}

func TestRepositoryReviewPurgeRecoversPartialMultiLedgerDeletion(t *testing.T) {
	store := newAutomationTestStore(t)
	input := validAutomationForTest("rra_alias_partial_recovery", "alias recovery")
	input.Repository = "https://github.com/Owner/Repo.git"
	automation, err := store.CreateAutomation(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	canonical := createPurgeTestLedger(t, store, CanonicalRepositoryIdentity(automation.Repository))
	legacy := createPurgeTestLedger(t, store, automation.Repository)
	targets := []repositoryReviewPurgeLedgerTarget{
		{Repository: canonical.Repository, Version: canonical.Version},
		{Repository: legacy.Repository, Version: legacy.Version},
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Repository < targets[j].Repository })
	intent := repositoryReviewPurgeIntent{
		SchemaVersion:             repositoryReviewPurgeIntentSchemaVersion,
		Mode:                      repositoryReviewPurgeReset,
		Phase:                     repositoryReviewPurgeLedgerCommitting,
		AutomationID:              automation.ID,
		ConfiguredRepository:      automation.Repository,
		Repository:                canonical.Repository,
		LedgerTargets:             targets,
		ExpectedAutomationVersion: automation.Version,
		ExpectedRepositoryVersion: canonical.Version,
		CreatedAt:                 automationTestNow,
	}
	reset := cloneAutomation(automation)
	resetRepositoryReviewAutomationHistory(&reset)
	reset.Version++
	reset.UpdatedAt = automationTestNow
	if err := store.saveAutomation(reset); err != nil {
		t.Fatal(err)
	}
	if err := store.savePurgeIntent(intent); err != nil {
		t.Fatal(err)
	}
	if err := store.removeRepositoryReviewLedger(targets[0].Repository); err != nil {
		t.Fatal(err)
	}

	if count, err := store.ReconcilePurgeIntents(context.Background()); err != nil || count != 1 {
		t.Fatalf("reconcile count=%d err=%v", count, err)
	}
	for _, target := range targets {
		if _, found, err := store.Get(target.Repository); err != nil || found {
			t.Fatalf("recovered ledger %q found=%v err=%v", target.Repository, found, err)
		}
	}
	loaded, found, err := store.GetAutomation(context.Background(), automation.ID)
	if err != nil || !found || !repositoryReviewAutomationHistoryReset(loaded) {
		t.Fatalf("recovered automation=%#v found=%v err=%v", loaded, found, err)
	}
}

func TestRepositoryReviewPurgeIntentFencesAutomationLedgerAndReassignment(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_fenced_purge", "fenced")
	state := createPurgeTestLedger(t, store, automation.Repository)
	intent := repositoryReviewPurgeIntent{
		SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
		Mode:          repositoryReviewPurgeReset, Phase: repositoryReviewPurgePrepared,
		AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
		Repository: state.Repository, ExpectedAutomationVersion: automation.Version,
		LedgerTargets:             []repositoryReviewPurgeLedgerTarget{{Repository: state.Repository, Version: state.Version}},
		ExpectedRepositoryVersion: state.Version, CreatedAt: automationTestNow,
	}
	if err := store.savePurgeIntent(intent); err != nil {
		t.Fatal(err)
	}
	second := NewStore(filepath.Dir(store.root))
	if _, _, err := second.GetAutomation(context.Background(), automation.ID); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
		t.Fatalf("automation fence error = %v", err)
	}
	if _, _, err := second.Get(automation.Repository); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
		t.Fatalf("ledger fence error = %v", err)
	}
	if _, err := second.List(); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
		t.Fatalf("ledger list fence error = %v", err)
	}
	if _, err := second.ListSummaries(); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
		t.Fatalf("summary list fence error = %v", err)
	}
	replacement := validAutomationForTest("rra_fenced_replacement", "replacement")
	replacement.Repository = automation.Repository
	if _, err := second.CreateAutomation(context.Background(), replacement); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
		t.Fatalf("reassignment fence error = %v", err)
	}
	if count, err := store.ReconcilePurgeIntents(context.Background()); err != nil || count != 1 {
		t.Fatalf("reconcile count=%d err=%v", count, err)
	}
}

func TestRepositoryReviewPurgeRejectsOverlappingIntentOwner(t *testing.T) {
	store := newAutomationTestStore(t)
	first := createAutomationForTest(t, store, "rra_overlap_first", "first")
	second := cloneAutomation(first)
	second.ID = "rra_overlap_second"
	second.Name = "second"
	if err := store.saveAutomation(second); err != nil {
		t.Fatal(err)
	}
	state := createPurgeTestLedger(t, store, first.Repository)
	firstIntent := purgeTestIntent(first, state)
	if err := store.savePurgeIntent(firstIntent); err != nil {
		t.Fatal(err)
	}

	if _, err := store.DeleteAutomationAndHistory(
		context.Background(), second.ID, second.Version, state.Version,
		purgeTestFence(state), second.Repository,
	); !errors.Is(err, ErrRepositoryReviewPurgeInProgress) {
		t.Fatalf("overlapping purge error = %v", err)
	}
	owner, found, err := store.loadPurgeFence(state.Repository)
	if err != nil || !found || owner.AutomationID != first.ID {
		t.Fatalf("shared fence owner=%#v found=%v err=%v", owner, found, err)
	}
	if _, found, err := store.loadPurgeIntentForAutomation(second.ID); err != nil || found {
		t.Fatalf("second primary intent found=%v err=%v", found, err)
	}
}

func TestRepositoryReviewPurgeIntentRejectsPhaseRegression(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_phase_regression", "phase regression")
	state := createPurgeTestLedger(t, store, automation.Repository)
	prepared := purgeTestIntent(automation, state)
	if err := store.savePurgeIntent(prepared); err != nil {
		t.Fatal(err)
	}
	advanced := prepared
	advanced.Phase = repositoryReviewPurgeAutomationCommitting
	if err := store.savePurgeIntent(advanced); err != nil {
		t.Fatal(err)
	}
	if err := store.savePurgeIntent(prepared); !errors.Is(err, ErrConflict) {
		t.Fatalf("phase regression error = %v", err)
	}
	loaded, found, err := store.loadPurgeIntentForAutomation(automation.ID)
	if err != nil || !found || loaded.Phase != advanced.Phase {
		t.Fatalf("retained intent=%#v found=%v err=%v", loaded, found, err)
	}
}

func TestRepositoryReviewPurgeIntentRejectsUnsafeOrUnboundState(t *testing.T) {
	t.Run("public permissions", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_public_intent", "public")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := purgeTestIntent(automation, state)
		if err := store.savePurgeIntent(intent); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(store.purgeAutomationIntentPath(automation.ID), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.GetAutomation(context.Background(), automation.ID); err == nil {
			t.Fatal("automation read ignored unsafe purge marker")
		}
		if _, err := store.ReconcilePurgeIntents(context.Background()); err == nil {
			t.Fatal("public purge intent was accepted")
		}
		if _, found, err := store.loadAutomationIgnoringPurge(automation.ID); err != nil || !found {
			t.Fatalf("unsafe marker changed automation found=%v err=%v", found, err)
		}
		if state, err := store.loadIgnoringPurge(automation.Repository); err != nil || state.Version == 0 {
			t.Fatalf("unsafe marker changed ledger=%#v err=%v", state, err)
		}
	})

	t.Run("unknown JSON field", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_unknown_intent", "unknown")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := purgeTestIntent(automation, state)
		data, err := json.Marshal(intent)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data[:len(data)-1], []byte(`,"unexpected":true}`)...)
		if err := os.MkdirAll(store.root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(store.purgeAutomationIntentPath(automation.ID), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReconcilePurgeIntents(context.Background()); err == nil {
			t.Fatal("unknown purge intent field was accepted")
		}
	})

	t.Run("unrelated ledger", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_unbound_intent", "unbound")
		assigned := createPurgeTestLedger(t, store, automation.Repository)
		unrelated := createPurgeTestLedger(t, store, "other/repository")
		intent := purgeTestIntent(automation, unrelated)
		if err := store.savePurgeIntent(intent); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReconcilePurgeIntents(context.Background()); !errors.Is(err, ErrConflict) {
			t.Fatalf("unbound intent error = %v", err)
		}
		if retained, err := store.loadIgnoringPurge(assigned.Repository); err != nil || retained.Version == 0 {
			t.Fatalf("assigned ledger changed=%#v err=%v", retained, err)
		}
		if retained, err := store.loadIgnoringPurge(unrelated.Repository); err != nil || retained.Version == 0 {
			t.Fatalf("unrelated ledger changed=%#v err=%v", retained, err)
		}
	})
}

func TestRepositoryReviewPurgeBlocksRunningEffectsWithoutMutation(t *testing.T) {
	store := newAutomationTestStore(t)
	automation := createAutomationForTest(t, store, "rra_blocked_purge", "blocked")
	state := createPurgeTestLedger(t, store, automation.Repository)
	state.IssueDrafts = []IssueDraft{{State: IssueDraftPublishing}}
	store.loadForTest = func(repository string) (RepositoryState, error) {
		if repository != automation.Repository {
			return RepositoryState{}, errors.New("unexpected repository")
		}
		return state, nil
	}

	_, eligibility, err := store.PurgeAutomationHistory(
		context.Background(), automation.ID, automation.Version, state.Version,
		purgeTestFence(state), automation.Repository,
	)
	if !errors.Is(err, ErrRepositoryReviewPurgeBlocked) || eligibility.CanPurge ||
		len(eligibility.Blockers) != 1 || eligibility.Blockers[0].Code != RepositoryReviewPurgeBlockerPublicationActive {
		t.Fatalf("blocked eligibility=%#v err=%v", eligibility, err)
	}
	loaded, found, getErr := store.GetAutomation(context.Background(), automation.ID)
	if getErr != nil || !found || loaded.Version != automation.Version {
		t.Fatalf("blocked automation changed=%#v found=%v err=%v", loaded, found, getErr)
	}
	retained, found, getErr := store.Get(automation.Repository)
	if getErr != nil || !found || retained.Version != state.Version {
		t.Fatalf("blocked ledger changed=%#v found=%v err=%v", retained, found, getErr)
	}
}

func TestRepositoryReviewPurgeIntentRecoveryIsIdempotent(t *testing.T) {
	t.Run("remove after automation deletion", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_recover_remove", "remove")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := repositoryReviewPurgeIntent{
			SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
			Mode:          repositoryReviewPurgeRemove, Phase: repositoryReviewPurgeAutomationCommitting,
			AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
			Repository: automation.Repository, ExpectedAutomationVersion: automation.Version,
			LedgerTargets:             []repositoryReviewPurgeLedgerTarget{{Repository: state.Repository, Version: state.Version}},
			ExpectedRepositoryVersion: state.Version, CreatedAt: automationTestNow,
		}
		if err := store.savePurgeIntent(intent); err != nil {
			t.Fatal(err)
		}
		if err := removeRepositoryReviewRegularFile(store.automationPath(automation.ID)); err != nil {
			t.Fatal(err)
		}
		if count, err := store.ReconcilePurgeIntents(context.Background()); err != nil || count != 1 {
			t.Fatalf("reconcile count=%d err=%v", count, err)
		}
		if count, err := store.ReconcilePurgeIntents(context.Background()); err != nil || count != 0 {
			t.Fatalf("idempotent reconcile count=%d err=%v", count, err)
		}
		if _, found, err := store.Get(automation.Repository); err != nil || found {
			t.Fatalf("recovered ledger found=%v err=%v", found, err)
		}
	})

	t.Run("reset after automation update", func(t *testing.T) {
		store := newAutomationTestStore(t)
		automation := createAutomationForTest(t, store, "rra_recover_reset", "reset")
		state := createPurgeTestLedger(t, store, automation.Repository)
		intent := repositoryReviewPurgeIntent{
			SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
			Mode:          repositoryReviewPurgeReset, Phase: repositoryReviewPurgeAutomationCommitting,
			AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
			Repository: automation.Repository, ExpectedAutomationVersion: automation.Version,
			LedgerTargets:             []repositoryReviewPurgeLedgerTarget{{Repository: state.Repository, Version: state.Version}},
			ExpectedRepositoryVersion: state.Version, CreatedAt: automationTestNow,
		}
		if err := store.savePurgeIntent(intent); err != nil {
			t.Fatal(err)
		}
		reset := cloneAutomation(automation)
		resetRepositoryReviewAutomationHistory(&reset)
		reset.Version++
		reset.UpdatedAt = automationTestNow
		if err := store.saveAutomation(reset); err != nil {
			t.Fatal(err)
		}
		if count, err := store.ReconcilePurgeIntents(context.Background()); err != nil || count != 1 {
			t.Fatalf("reconcile count=%d err=%v", count, err)
		}
		loaded, found, err := store.GetAutomation(context.Background(), automation.ID)
		if err != nil || !found || !repositoryReviewAutomationHistoryReset(loaded) {
			t.Fatalf("recovered automation=%#v found=%v err=%v", loaded, found, err)
		}
	})

	for _, phase := range []repositoryReviewPurgePhase{
		repositoryReviewPurgeAutomationApplied,
		repositoryReviewPurgeLedgerCommitting,
		repositoryReviewPurgeLedgerRemoved,
	} {
		t.Run(string(phase), func(t *testing.T) {
			store := newAutomationTestStore(t)
			automation := createAutomationForTest(t, store, "rra_recover_"+string(phase), string(phase))
			state := createPurgeTestLedger(t, store, automation.Repository)
			intent := purgeTestIntent(automation, state)
			intent.Phase = phase
			if err := store.savePurgeIntent(intent); err != nil {
				t.Fatal(err)
			}
			reset := cloneAutomation(automation)
			resetRepositoryReviewAutomationHistory(&reset)
			reset.Version++
			reset.UpdatedAt = automationTestNow
			if err := store.saveAutomation(reset); err != nil {
				t.Fatal(err)
			}
			if phase == repositoryReviewPurgeLedgerRemoved {
				if err := store.removeRepositoryReviewLedger(state.Repository); err != nil {
					t.Fatal(err)
				}
			}
			if count, err := store.ReconcilePurgeIntents(context.Background()); err != nil || count != 1 {
				t.Fatalf("reconcile count=%d err=%v", count, err)
			}
			if _, found, err := store.Get(state.Repository); err != nil || found {
				t.Fatalf("recovered ledger found=%v err=%v", found, err)
			}
		})
	}
}

func createPurgeTestLedger(t *testing.T, store Store, repository string) RepositoryState {
	t.Helper()
	state := repositoryReviewCoverageState(repository)
	if err := store.save(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

func purgeTestIntent(
	automation RepositoryReviewAutomation,
	state RepositoryState,
) repositoryReviewPurgeIntent {
	return repositoryReviewPurgeIntent{
		SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
		Mode:          repositoryReviewPurgeReset, Phase: repositoryReviewPurgePrepared,
		AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
		Repository: state.Repository, ExpectedAutomationVersion: automation.Version,
		LedgerTargets:             []repositoryReviewPurgeLedgerTarget{{Repository: state.Repository, Version: state.Version}},
		ExpectedRepositoryVersion: state.Version, CreatedAt: automationTestNow,
	}
}

func purgeTestFence(states ...RepositoryState) string {
	targets := make([]repositoryReviewPurgeLedgerTarget, 0, len(states))
	for _, state := range states {
		if state.Version > 0 {
			targets = append(targets, repositoryReviewPurgeLedgerTarget{
				Repository: state.Repository, Version: state.Version,
			})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Repository < targets[j].Repository })
	return repositoryReviewPurgeLedgerFence(targets)
}
