package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/repoaudit"
)

func TestRepositoryReviewPurgeHandlerFailureCoverage(t *testing.T) {
	validBody, err := json.Marshal(map[string]any{
		"expected_version":            1,
		"expected_repository_version": 0,
		"expected_ledger_fence":       "rplf_test",
		"confirm_repository":          "owner/repo",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("cross site", func(t *testing.T) {
		handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/rra_missing/purge-history",
			bytes.NewReader(validBody),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Sec-Fetch-Site", "cross-site")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("cross-site purge status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("malformed", func(t *testing.T) {
		handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/rra_missing/purge-history",
			strings.NewReader(`{`),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("malformed purge status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("nil handler", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/purge", bytes.NewReader(validBody))
		request.SetPathValue("automation_id", "rra_missing")
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		(*Handler)(nil).handlePurgeRepositoryReviewAutomationHistory(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("nil-handler purge status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("nil handler delete", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodDelete, "/delete", bytes.NewReader(validBody))
		request.SetPathValue("automation_id", "rra_missing")
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		(*Handler)(nil).handleDeleteRepositoryReviewAutomation(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("nil-handler delete status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("controller start", func(t *testing.T) {
		badConfig := t.TempDir()
		handler := NewHandler(badConfig)
		mux := http.NewServeMux()
		handler.RegisterRoutes(mux)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/rra_missing/purge-history",
			bytes.NewReader(validBody),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("unavailable-controller purge status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("history absent", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store := repoaudit.NewStore(workspace)
		automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
		if err != nil {
			t.Fatal(err)
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/purge-history",
			map[string]any{
				"expected_version":            automation.Version,
				"expected_repository_version": 0,
				"expected_ledger_fence":       repositoryReviewPurgeFenceForTest(t, workspace, automation),
				"confirm_repository":          automation.Repository,
			},
		)
		if response.Code != http.StatusNotFound ||
			!strings.Contains(response.Body.String(), "repository_review_history_not_found") {
			t.Fatalf("absent-history purge status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("active review", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store := repoaudit.NewStore(workspace)
		input := testRepositoryReviewAutomation()
		input.Status = repoaudit.RepositoryReviewAutomationRunning
		input.ActiveRunID = "wr_purge_active"
		input.RunIDs = []string{input.ActiveRunID}
		automation, err := store.CreateAutomation(t.Context(), input)
		if err != nil {
			t.Fatal(err)
		}
		controller := handler.repositoryReviewControllerInstance()
		controller.mu.Lock()
		controller.active[automation.ID] = &repositoryReviewActiveRun{
			runID: automation.ActiveRunID, store: store,
		}
		controller.mu.Unlock()
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/purge-history",
			map[string]any{
				"expected_version":            automation.Version,
				"expected_repository_version": 0,
				"expected_ledger_fence":       repositoryReviewPurgeFenceForTest(t, workspace, automation),
				"confirm_repository":          automation.Repository,
			},
		)
		if response.Code != http.StatusConflict ||
			!strings.Contains(response.Body.String(), "repository_review_purge_blocked") {
			t.Fatalf("active purge status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func TestRepositoryReviewControllerFailsClosedOnUnsafePurgeIntent(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	root := filepath.Join(workspace, "repository_reviews")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "purge_automation_rra_unsafe.json"),
		[]byte(`{}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	controller := handler.repositoryReviewControllerInstance()
	if err := controller.Start(); err == nil ||
		!strings.Contains(err.Error(), "reconcile repository review purges") {
		t.Fatalf("unsafe purge controller start error = %v", err)
	}
}
