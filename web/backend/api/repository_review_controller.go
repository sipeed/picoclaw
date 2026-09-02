package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	osexec "os/exec"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/gitworkspace"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

var (
	errRepositoryReviewAutomationBusy    = errors.New("repository review automation is already active")
	errRepositoryReviewSafeStop          = errors.New("repository review stopped at a safe checkpoint")
	errRepositoryReviewInvalidTransition = errors.New("repository review action is not valid for the current status")
	errRepositoryReviewCommitSelection   = errors.New("repository review commit selection is required")
	errRepositoryReviewPauseSettled      = errors.New("repository review is already stopped")
	errRepositoryReviewProfileActive     = errors.New(
		"repository review profile is assigned to an active repository review",
	)
	repositoryReviewCommandContext = osexec.CommandContext
	repositoryReviewReadAll        = io.ReadAll
	repositoryReviewParseWorkflow  = workflows.Parse
)

const (
	repositoryReviewControllerInterval = 5 * time.Second
	repositoryReviewQuotaProbeTimeout  = 30 * time.Second
	repositoryReviewMaximumFiles       = 100_000
	// Repository reviews reserve five minutes beyond the fixed assignment
	// deadline for planning, durable checkpoints, and reservation cleanup.
	repositoryReviewWorkflowCleanupReserve = 5 * time.Minute
)

type repositoryReviewActiveRun struct {
	runID        string
	pauseReason  repoaudit.RepositoryReviewPauseReason
	pauseDetail  string
	store        repoaudit.Store
	config       *config.Config
	reservations map[int]repositoryReviewTaskReservation
	guardMu      *sync.Mutex
}

type repositoryReviewTaskReservation struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CostUSD          float64
	CostKnown        bool
}

type repositoryReviewAutomationUpdater func(
	context.Context,
	repoaudit.Store,
	string,
	int64,
	func(*repoaudit.RepositoryReviewAutomation) error,
) (repoaudit.RepositoryReviewAutomation, error)

type repositoryReviewCommitResolver func(
	context.Context,
	*config.Config,
	repoaudit.RepositoryReviewAutomation,
	string,
) (string, error)

type repositoryReviewDefaultBranchResolver func(
	context.Context,
	*config.Config,
	repoaudit.RepositoryReviewAutomation,
) (string, error)

func updateRepositoryReviewAutomation(
	ctx context.Context,
	store repoaudit.Store,
	id string,
	expectedVersion int64,
	mutate func(*repoaudit.RepositoryReviewAutomation) error,
) (repoaudit.RepositoryReviewAutomation, error) {
	return store.UpdateAutomation(ctx, id, expectedVersion, mutate)
}

var reconcileRepositoryReviewDeduplicationJobs = func(
	store repoaudit.Store,
	ctx context.Context,
) (int, error) {
	return store.ReconcileDeduplicationJobs(ctx)
}

var repositoryReviewCampaignWorkflowRuntime = func(
	controller *repositoryReviewController,
	ctx context.Context,
	cfg *config.Config,
) (*config.Config, *workflows.FileRunStore, *workflows.Executor, error) {
	return controller.handler.workflowRuntimeFromConfigWithoutPrune(ctx, cfg)
}

var applyRepositoryReviewPause = applyRepositoryReviewPauseTransition

type repositoryReviewController struct {
	handler *Handler
	ctx     context.Context
	cancel  context.CancelFunc

	startOnce                 sync.Once
	stopOnce                  sync.Once
	releaseOnce               sync.Once
	wg                        sync.WaitGroup
	admissionWG               sync.WaitGroup
	lifecycleMu               sync.Mutex
	stopped                   bool
	startErr                  error
	releaseLease              func()
	leasedStore               repoaudit.Store
	leasedConfig              *config.Config
	mu                        sync.Mutex
	mappingMu                 sync.Mutex
	deduplicationMu           sync.Mutex
	historicalDeduplicationMu sync.Mutex
	validationMu              sync.Mutex
	active                    map[string]*repositoryReviewActiveRun
	now                       func() time.Time
	probe                     func(context.Context) (codexAccountLimitsResponse, error)
	update                    repositoryReviewAutomationUpdater
	resolveCommit             repositoryReviewCommitResolver
	resolveDefaultBranch      repositoryReviewDefaultBranchResolver
	recoverCampaign           func(
		context.Context,
		repoaudit.Store,
		string,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.RepositoryReviewModelProfile,
	) (repoaudit.RepositoryReviewAutomation, error)
	stopTimeout   time.Duration
	monitorEvery  time.Duration
	progressEvery time.Duration
	runBatch      func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.AgentUsageObserver,
	) (*workflows.RunResult, error)
}

func newRepositoryReviewController(handler *Handler) *repositoryReviewController {
	ctx, cancel := context.WithCancel(context.Background())
	controller := &repositoryReviewController{
		handler:              handler,
		ctx:                  ctx,
		cancel:               cancel,
		active:               make(map[string]*repositoryReviewActiveRun),
		now:                  time.Now,
		probe:                loadCodexAccountLimits,
		update:               updateRepositoryReviewAutomation,
		resolveCommit:        resolveRepositoryReviewAutomationCommit,
		resolveDefaultBranch: resolveRepositoryReviewAdvertisedDefaultBranch,
		stopTimeout:          10 * time.Second,
		monitorEvery:         repositoryReviewControllerInterval,
		progressEvery:        time.Second,
	}
	controller.recoverCampaign = controller.recoverLegacyRepositoryReviewCampaign
	return controller
}

func (h *Handler) repositoryReviewControllerInstance() *repositoryReviewController {
	if h == nil {
		return nil
	}
	h.repositoryReviewControllerMu.Lock()
	defer h.repositoryReviewControllerMu.Unlock()
	if h.repositoryReviewController == nil {
		h.repositoryReviewController = newRepositoryReviewController(h)
	}
	return h.repositoryReviewController
}

// StartRepositoryReviewController starts the durable quota/recovery monitor.
// It is safe to call more than once.
func (h *Handler) StartRepositoryReviewController() {
	if controller := h.repositoryReviewControllerInstance(); controller != nil {
		if err := controller.Start(); err != nil {
			logger.ErrorC("repository-review", "Repository review controller did not start: "+err.Error())
		}
	}
}

func (h *Handler) stopRepositoryReviewController() {
	if h == nil {
		return
	}
	h.repositoryReviewControllerMu.Lock()
	controller := h.repositoryReviewController
	h.repositoryReviewControllerMu.Unlock()
	if controller != nil {
		controller.Stop()
	}
}

func (c *repositoryReviewController) Start() error {
	if c == nil {
		return errors.New("repository review controller is unavailable")
	}
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	if c.stopped {
		return context.Canceled
	}
	c.startOnce.Do(func() {
		store, cfg, err := c.store()
		if err == nil {
			c.releaseLease, err = store.LockAutomationController()
		}
		if err != nil {
			c.startErr = err
			c.cancel()
			return
		}
		c.leasedStore = store
		c.leasedConfig = cfg
		if _, err = store.ReconcilePurgeIntents(c.ctx); err != nil {
			c.startErr = fmt.Errorf("reconcile repository review purges: %w", err)
			if c.releaseLease != nil {
				c.releaseLease()
				c.releaseLease = nil
			}
			c.cancel()
			return
		}
		if _, err = store.ReconcileJobs(c.ctx); err != nil {
			c.startErr = fmt.Errorf("reconcile repository finding jobs: %w", err)
			if c.releaseLease != nil {
				c.releaseLease()
				c.releaseLease = nil
			}
			c.cancel()
			return
		}
		if _, err = reconcileRepositoryReviewDeduplicationJobs(store, c.ctx); err != nil {
			c.startErr = fmt.Errorf("reconcile finding deduplication jobs: %w", err)
			if c.releaseLease != nil {
				c.releaseLease()
				c.releaseLease = nil
			}
			c.cancel()
			return
		}
		c.wg.Add(1)
		go c.monitor()
	})
	return c.startErr
}

func (c *repositoryReviewController) admitBackgroundWorker(workerMu *sync.Mutex) bool {
	if c == nil || workerMu == nil || !workerMu.TryLock() {
		return false
	}
	if !c.registerBackgroundWorker() {
		workerMu.Unlock()
		return false
	}
	return true
}

func (c *repositoryReviewController) registerBackgroundWorker() bool {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	if c.stopped || c.ctx.Err() != nil {
		return false
	}
	// Stop holds lifecycleMu while it closes admission, so every Add either
	// happens before Stop begins waiting or is rejected after shutdown starts.
	c.wg.Add(1)
	return true
}

func (c *repositoryReviewController) Stop() {
	if c == nil {
		return
	}
	c.lifecycleMu.Lock()
	c.stopOnce.Do(func() {
		c.stopped = true
		c.cancel()
	})
	c.lifecycleMu.Unlock()
	done := make(chan struct{})
	go func() {
		c.admissionWG.Wait()
		c.wg.Wait()
		c.releaseOnce.Do(func() {
			if c.releaseLease != nil {
				c.releaseLease()
			}
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(c.stopTimeout):
		logger.WarnC("repository-review", "Timed out waiting for repository review controller shutdown")
	}
}

func (c *repositoryReviewController) store() (repoaudit.Store, *config.Config, error) {
	if c == nil || c.handler == nil {
		return repoaudit.Store{}, nil, errors.New("repository review controller is unavailable")
	}
	cfg, err := config.LoadConfig(c.handler.configPath)
	if err != nil {
		return repoaudit.Store{}, nil, err
	}
	return repoaudit.NewStore(cfg.WorkspacePath()), cfg, nil
}

func resolveRepositoryReviewAutomationCommit(
	ctx context.Context,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
	revision string,
) (string, error) {
	if cfg == nil {
		return "", errors.New("repository review configuration is unavailable")
	}
	manager, err := gitworkspace.NewManager(gitworkspace.Options{
		RootDir:             cfg.GitWorkspaceRootPath(),
		MaxTotalSizeBytes:   cfg.GitWorkspaces.EffectiveMaxTotalSizeBytes(),
		IgnoredCleanupDelay: cfg.GitWorkspaces.EffectiveIgnoredCleanupDelay(),
		DropDelay:           cfg.GitWorkspaces.EffectiveDropDelay(),
	})
	if err != nil {
		return "", fmt.Errorf("initialize repository review commit resolver: %w", err)
	}
	ref := strings.TrimSpace(revision)
	if ref == "" {
		ref = strings.TrimSpace(automation.Ref)
	}
	sessionKey := "repository-review-commit/" + automation.ID + "/" + workflows.NewRunID()
	workspace, err := manager.Acquire(ctx, gitworkspace.AcquireRequest{
		Repository: automation.Repository,
		Ref:        ref,
		Fresh:      true,
		SessionKey: sessionKey,
		AgentID:    "repository-review-controller",
	})
	if err != nil {
		return "", fmt.Errorf("resolve repository review commit: %w", err)
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, releaseErr := manager.ReleaseSession(releaseCtx, gitworkspace.ReleaseRequest{
			SessionKey: sessionKey,
			AgentID:    "repository-review-controller",
		}); releaseErr != nil {
			logger.WarnCF("repository-review", "Failed to release commit-resolution workspace", map[string]any{
				"automation_id": automation.ID,
				"error":         releaseErr.Error(),
			})
		}
	}()
	output, err := repositoryReviewGitOutput(
		ctx,
		workspace.Path,
		128,
		"git",
		"rev-parse",
		"--verify",
		"--end-of-options",
		"HEAD^{commit}",
	)
	if err != nil {
		return "", fmt.Errorf("resolve repository review commit ID: %w", err)
	}
	commit := strings.ToLower(strings.TrimSpace(string(output)))
	if !repositoryReviewValidCommitSHA(commit) {
		return "", errors.New("repository review resolved a noncanonical commit")
	}
	return commit, nil
}

func resolveRepositoryReviewAdvertisedDefaultBranch(
	ctx context.Context,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
) (string, error) {
	if cfg == nil {
		return "", errors.New("repository review configuration is unavailable")
	}
	manager, err := gitworkspace.NewManager(gitworkspace.Options{
		RootDir:             cfg.GitWorkspaceRootPath(),
		MaxTotalSizeBytes:   cfg.GitWorkspaces.EffectiveMaxTotalSizeBytes(),
		IgnoredCleanupDelay: cfg.GitWorkspaces.EffectiveIgnoredCleanupDelay(),
		DropDelay:           cfg.GitWorkspaces.EffectiveDropDelay(),
	})
	if err != nil {
		return "", err
	}
	sessionKey := "repository-review-default-branch/" + automation.ID + "/" + workflows.NewRunID()
	workspace, err := manager.Acquire(ctx, gitworkspace.AcquireRequest{
		Repository: automation.Repository, Ref: "", Fresh: true,
		SessionKey: sessionKey, AgentID: "repository-review-controller",
	})
	if err != nil {
		return "", err
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = manager.ReleaseSession(releaseCtx, gitworkspace.ReleaseRequest{
			SessionKey: sessionKey, AgentID: "repository-review-controller",
		})
	}()
	for _, ref := range []string{"refs/remotes/origin/HEAD", "HEAD"} {
		output, outputErr := repositoryReviewGitOutput(
			ctx, workspace.Path, 512, "git", "symbolic-ref", "--short", ref,
		)
		if outputErr != nil {
			continue
		}
		branch := strings.TrimSpace(string(output))
		branch = strings.TrimPrefix(branch, "origin/")
		if normalized, normalizeErr := repoaudit.NormalizeRepositoryReviewBranch(branch); normalizeErr == nil &&
			normalized != "" {
			return normalized, nil
		}
	}
	return "", errors.New("repository advertised default branch is unavailable")
}

func repositoryReviewGitOutput(
	ctx context.Context,
	directory string,
	maximumBytes int64,
	name string,
	arguments ...string,
) ([]byte, error) {
	command := repositoryReviewCommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = repositoryReviewGitEnvironment()
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	output, readErr := repositoryReviewReadAll(io.LimitReader(stdout, maximumBytes+1))
	_, drainErr := io.Copy(io.Discard, stdout)
	waitErr := command.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if drainErr != nil {
		return nil, drainErr
	}
	if waitErr != nil {
		return nil, waitErr
	}
	if int64(len(output)) > maximumBytes {
		return output[:maximumBytes], errors.New("git command output is too large")
	}
	return output, nil
}

func repositoryReviewGitEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(name) {
		case "LC_ALL", "GIT_PAGER", "GIT_LITERAL_PATHSPECS":
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "LC_ALL=C", "GIT_PAGER=cat", "GIT_LITERAL_PATHSPECS=1")
}

func repositoryReviewRememberedCommit(
	automation repoaudit.RepositoryReviewAutomation,
) string {
	commit := strings.ToLower(strings.TrimSpace(automation.ResolvedCommitSHA))
	if repositoryReviewValidCommitSHA(commit) {
		return commit
	}
	commit = strings.ToLower(strings.TrimSpace(automation.ScopePlan.CommitSHA))
	if repositoryReviewValidCommitSHA(commit) {
		return commit
	}
	return ""
}

func repositoryReviewValidCommitSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func repositoryReviewValidCommitSelection(value string) bool {
	return repositoryReviewValidCommitSHA(strings.ToLower(strings.TrimSpace(value)))
}

func repositoryReviewAutomationCanResume(
	status repoaudit.RepositoryReviewAutomationStatus,
) bool {
	return status == repoaudit.RepositoryReviewAutomationPaused ||
		status == repoaudit.RepositoryReviewAutomationFailed
}

func (c *repositoryReviewController) startAutomation(
	ctx context.Context,
	id string,
	expectedVersion int64,
	resetBudget bool,
	action string,
) (repoaudit.RepositoryReviewAutomation, error) {
	return c.startAutomationAtCommit(
		ctx,
		id,
		expectedVersion,
		resetBudget,
		action,
		"",
	)
}

func (c *repositoryReviewController) repositoryReviewCommitOptions(
	ctx context.Context,
	id string,
) (repoaudit.RepositoryReviewAutomation, string, string, error) {
	if c == nil || c.resolveCommit == nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", errors.New(
			"repository review commit resolver is unavailable",
		)
	}
	if err := c.Start(); err != nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", err
	}
	c.lifecycleMu.Lock()
	if c.stopped || c.ctx.Err() != nil {
		c.lifecycleMu.Unlock()
		return repoaudit.RepositoryReviewAutomation{}, "", "", context.Canceled
	}
	c.admissionWG.Add(1)
	c.lifecycleMu.Unlock()
	defer c.admissionWG.Done()
	optionsCtx, cancelOptions := context.WithCancel(c.ctx)
	stopCallerCancellation := context.AfterFunc(ctx, cancelOptions)
	defer func() {
		stopCallerCancellation()
		cancelOptions()
	}()
	ctx = optionsCtx
	store := c.leasedStore
	cfg, err := c.currentLeasedConfiguration()
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", err
	}
	automation, found, err := store.GetAutomation(ctx, id)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", err
	}
	if !found {
		return repoaudit.RepositoryReviewAutomation{}, "", "", os.ErrNotExist
	}
	if !repositoryReviewAutomationCanResume(automation.Status) {
		return repoaudit.RepositoryReviewAutomation{}, "", "", errRepositoryReviewInvalidTransition
	}
	remembered := repositoryReviewRememberedCommit(automation)
	resolutionAutomation, err := repositoryReviewAutomationResolutionTarget(automation)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", err
	}
	latest, err := c.resolveCommit(ctx, cfg, resolutionAutomation, "")
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", err
	}
	latest = strings.ToLower(strings.TrimSpace(latest))
	if !repositoryReviewValidCommitSHA(latest) {
		return repoaudit.RepositoryReviewAutomation{}, "", "", errors.New(
			"repository review resolved a noncanonical latest commit",
		)
	}
	if remembered == "" {
		remembered = latest
	}
	current, currentFound, err := store.GetAutomation(ctx, id)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, "", "", err
	}
	if !currentFound {
		return repoaudit.RepositoryReviewAutomation{}, "", "", os.ErrNotExist
	}
	if current.Version != automation.Version ||
		!repositoryReviewAutomationCanResume(current.Status) ||
		repositoryReviewRememberedCommit(current) != repositoryReviewRememberedCommit(automation) {
		return repoaudit.RepositoryReviewAutomation{}, "", "", repoaudit.ErrConflict
	}
	return current, remembered, latest, nil
}

func (c *repositoryReviewController) startAutomationAtCommit(
	ctx context.Context,
	id string,
	expectedVersion int64,
	resetBudget bool,
	action string,
	commitSelection string,
) (repoaudit.RepositoryReviewAutomation, error) {
	if err := c.Start(); err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	c.lifecycleMu.Lock()
	if c.stopped || c.ctx.Err() != nil {
		c.lifecycleMu.Unlock()
		return repoaudit.RepositoryReviewAutomation{}, context.Canceled
	}
	c.admissionWG.Add(1)
	c.lifecycleMu.Unlock()
	defer c.admissionWG.Done()
	admissionCtx, cancelAdmission := context.WithCancel(c.ctx)
	stopCallerCancellation := context.AfterFunc(ctx, cancelAdmission)
	defer func() {
		stopCallerCancellation()
		cancelAdmission()
	}()
	ctx = admissionCtx
	store := c.leasedStore
	cfg, err := c.currentLeasedConfiguration()
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	automation, found, err := store.GetAutomation(ctx, id)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if !found {
		return repoaudit.RepositoryReviewAutomation{}, errors.New("repository review automation not found")
	}
	if automation.Version != expectedVersion {
		return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
	}
	rememberedAtAdmission := repositoryReviewRememberedCommit(automation)
	switch action {
	case "start":
		if automation.Status != repoaudit.RepositoryReviewAutomationIdle {
			return repoaudit.RepositoryReviewAutomation{}, errRepositoryReviewInvalidTransition
		}
	case "resume":
		if !repositoryReviewAutomationCanResume(automation.Status) {
			return repoaudit.RepositoryReviewAutomation{}, errRepositoryReviewInvalidTransition
		}
	case "restart":
		if automation.Status != repoaudit.RepositoryReviewAutomationPaused &&
			automation.Status != repoaudit.RepositoryReviewAutomationCompleted &&
			automation.Status != repoaudit.RepositoryReviewAutomationFailed {
			return repoaudit.RepositoryReviewAutomation{}, errRepositoryReviewInvalidTransition
		}
	default:
		return repoaudit.RepositoryReviewAutomation{}, errRepositoryReviewInvalidTransition
	}
	restart := action == "restart"
	c.mu.Lock()
	_, locallyActive := c.active[id]
	c.mu.Unlock()
	if locallyActive || automation.Status == repoaudit.RepositoryReviewAutomationRunning ||
		automation.Status == repoaudit.RepositoryReviewAutomationStopping {
		return repoaudit.RepositoryReviewAutomation{}, errRepositoryReviewAutomationBusy
	}
	automation, err = c.normalizeRepositoryReviewAutomationAdmission(ctx, store, automation)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	automation, err = c.materializeLatestRepositoryReviewProfile(ctx, store, automation)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	effectiveAccount := repositoryReviewEffectiveAccountRef(cfg, automation.AccountRef)
	if validationErr := validateSelectableAccountRef(cfg, effectiveAccount); validationErr != nil {
		return repoaudit.RepositoryReviewAutomation{}, fmt.Errorf(
			"%w: account_ref: %v", repoaudit.ErrInvalidAutomation, validationErr,
		)
	}
	for _, model := range repositoryReviewExecutionModels(automation) {
		if !repositoryReviewAliasAvailableForAccount(cfg, model, effectiveAccount) {
			return repoaudit.RepositoryReviewAutomation{}, fmt.Errorf(
				"%w: reviewer model %q is unavailable on account %q",
				repoaudit.ErrInvalidAutomation, model, effectiveAccount,
			)
		}
	}
	if writerErr := validateRepositoryReviewIssueWriterAlias(
		cfg, effectiveAccount, automation.IssueWriterModel,
	); writerErr != nil {
		return repoaudit.RepositoryReviewAutomation{}, fmt.Errorf(
			"%w: %v", repoaudit.ErrInvalidAutomation, writerErr,
		)
	}
	commitAutomation := automation
	if repositoryReviewRememberedCommit(commitAutomation) == "" && rememberedAtAdmission != "" {
		commitAutomation.ResolvedCommitSHA = rememberedAtAdmission
	}
	resolvedCommit, err := c.resolveRepositoryReviewAdmissionCommit(
		ctx,
		cfg,
		commitAutomation,
		action,
		commitSelection,
	)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	advertisedDefaultBranch := strings.TrimSpace(automation.AdvertisedDefaultBranch)
	usesDefaultCommitResolver := c.resolveCommit != nil &&
		reflect.ValueOf(c.resolveCommit).Pointer() ==
			reflect.ValueOf(repositoryReviewCommitResolver(resolveRepositoryReviewAutomationCommit)).Pointer()
	if c.resolveDefaultBranch != nil && usesDefaultCommitResolver {
		resolvedDefault, defaultErr := c.resolveDefaultBranch(ctx, cfg, automation)
		if defaultErr != nil {
			return repoaudit.RepositoryReviewAutomation{}, fmt.Errorf(
				"resolve repository advertised default branch: %w", defaultErr,
			)
		}
		advertisedDefaultBranch = resolvedDefault
	}
	resolvedTargetBranch := strings.TrimSpace(automation.Ref)
	if resolvedTargetBranch == "" {
		resolvedTargetBranch = advertisedDefaultBranch
	}
	targetIsDefault := automation.Ref == "" ||
		advertisedDefaultBranch != "" && resolvedTargetBranch == advertisedDefaultBranch
	expectedVersion = automation.Version
	priced := automation
	if pricingErr := repositoryReviewRefreshAccountingSnapshot(cfg, &priced); pricingErr != nil {
		return repoaudit.RepositoryReviewAutomation{}, pricingErr
	}
	automation, err = c.update(
		ctx,
		store,
		id,
		expectedVersion,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			candidate.ModelPrices = maps.Clone(priced.ModelPrices)
			candidate.EffectiveAccountRef = effectiveAccount
			return nil
		},
	)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	expectedVersion = automation.Version

	if resetBudget {
		automation, err = c.update(
			ctx,
			store,
			id,
			expectedVersion,
			func(candidate *repoaudit.RepositoryReviewAutomation) error {
				candidate.Usage = repoaudit.RepositoryReviewTokenUsage{}
				candidate.EstimatedCostUSD = 0
				return nil
			},
		)
		if err != nil {
			return repoaudit.RepositoryReviewAutomation{}, err
		}
		expectedVersion = automation.Version
	}
	if _, _, interruptErr := store.InterruptAbandonedRepositoryReviewRun(
		ctx, repoaudit.CanonicalRepositoryIdentity(automation.Repository),
	); interruptErr != nil {
		return repoaudit.RepositoryReviewAutomation{}, interruptErr
	}
	commitChanged := repositoryReviewRememberedCommit(automation) != "" &&
		repositoryReviewRememberedCommit(automation) != resolvedCommit
	legacyContinuation := automation.CampaignID == "" &&
		(action == "resume" && !automation.StartedAt.IsZero() ||
			action == "start" && strings.EqualFold(
				strings.TrimSpace(automation.Progress.Stage), "next batch queued",
			))
	restartPrepared := restart && automation.CampaignID != "" &&
		automation.ResolvedCommitSHA == resolvedCommit && automation.ActiveRunID == "" &&
		automation.StartedAt.IsZero() && automation.Progress.CompletedBatches == 0 &&
		automation.Progress.ReviewedFiles == 0 && automation.Progress.InspectedFiles == 0
	newCampaign := restart && !restartPrepared || commitChanged ||
		automation.CampaignID == "" && !legacyContinuation
	recoverLegacyCampaign := !newCampaign &&
		repositoryReviewShouldRecoverLegacyCampaign(automation, action)
	if recoverLegacyCampaign && c.recoverCampaign == nil {
		return repoaudit.RepositoryReviewAutomation{}, errors.New(
			"legacy repository review campaign recovery is unavailable",
		)
	}
	campaignID := automation.CampaignID
	if recoverLegacyCampaign {
		recoveryProfile, profileErr := resolveRepositoryReviewCampaignProfile(
			ctx, c.handler.configPath, cfg, automation,
		)
		if profileErr != nil {
			return repoaudit.RepositoryReviewAutomation{}, profileErr
		}
		recovered, recoveryErr := c.recoverCampaign(
			ctx, store, cfg.WorkspacePath(), automation, resolvedCommit, recoveryProfile,
		)
		if recoveryErr != nil {
			return repoaudit.RepositoryReviewAutomation{}, recoveryErr
		}
		if recovered.ID != automation.ID || recovered.Version <= automation.Version ||
			recovered.CampaignID == "" || recovered.CampaignRecoveryPending ||
			recovered.ActiveRunID != "" || recovered.ScopeSelection == nil ||
			recovered.ScopePlan.Hash == "" || recovered.ScopePlan.CommitSHA != resolvedCommit {
			return repoaudit.RepositoryReviewAutomation{}, errors.New(
				"legacy repository review campaign recovery returned invalid installed authority",
			)
		}
		automation = recovered
		expectedVersion = automation.Version
		campaignID = automation.CampaignID
	}
	if newCampaign {
		campaignID = repoaudit.NewRepositoryReviewCampaignID()
	}
	automation, err = c.update(
		ctx,
		store,
		id,
		expectedVersion,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			if newCampaign {
				resetRepositoryReviewCampaignProgress(candidate)
			}
			candidate.CampaignID = campaignID
			candidate.CampaignRecoveryPending = false
			candidate.ResolvedCommitSHA = resolvedCommit
			candidate.ResolvedTargetBranch = resolvedTargetBranch
			candidate.AdvertisedDefaultBranch = advertisedDefaultBranch
			candidate.TargetIsDefault = targetIsDefault
			return nil
		},
	)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	expectedVersion = automation.Version
	if campaignID != "" {
		ledgerRepository := repoaudit.CanonicalRepositoryIdentity(automation.Repository)
		ledgerState, _, ledgerErr := store.Get(ledgerRepository)
		if ledgerErr != nil {
			return repoaudit.RepositoryReviewAutomation{}, ledgerErr
		}
		var deduplicationSnapshot repoaudit.RepositoryReviewDeduplicationSnapshot
		if current := ledgerState.CurrentCampaign; current != nil &&
			current.ID == campaignID && current.DeduplicationSnapshot != nil {
			// A campaign freezes its provider revision and policy. Configuration
			// changes between continuation batches apply only to a new campaign.
			deduplicationSnapshot = *current.DeduplicationSnapshot
		} else {
			var snapshotErr error
			deduplicationSnapshot, snapshotErr = c.repositoryReviewDeduplicationSnapshot(automation)
			if snapshotErr != nil {
				return repoaudit.RepositoryReviewAutomation{}, snapshotErr
			}
		}
		expectedCampaignID := ""
		exact := true
		if ledgerState.CurrentCampaign != nil {
			expectedCampaignID = ledgerState.CurrentCampaign.ID
			if expectedCampaignID == campaignID {
				exact = ledgerState.CurrentCampaign.Exact
			}
		}
		if _, beginErr := store.BeginCampaign(ctx, repoaudit.BeginCampaignRequest{
			Repository: ledgerRepository, CampaignID: campaignID,
			ExpectedCampaignID: expectedCampaignID, CommitSHA: resolvedCommit,
			ExpectedReviewVersion: ledgerState.ReviewVersion, Exact: exact,
			DeduplicationSnapshot: &deduplicationSnapshot,
		}); beginErr != nil {
			_, _ = c.updateLatest(ctx, store, id, func(candidate *repoaudit.RepositoryReviewAutomation) error {
				if candidate.CampaignID != campaignID || candidate.ActiveRunID != "" {
					return nil
				}
				candidate.Status = repoaudit.RepositoryReviewAutomationFailed
				candidate.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
				candidate.PauseDetail = repositoryReviewBoundedDetail(
					"Campaign authorization failed before workflow admission: " + beginErr.Error(),
				)
				candidate.Progress.Stage = "failed"
				return nil
			})
			return repoaudit.RepositoryReviewAutomation{}, beginErr
		}
	}
	runID := workflows.NewRunID()
	now := c.clock()
	c.lifecycleMu.Lock()
	if c.stopped || c.ctx.Err() != nil {
		c.lifecycleMu.Unlock()
		return repoaudit.RepositoryReviewAutomation{}, context.Canceled
	}
	c.mu.Lock()
	if _, exists := c.active[id]; exists {
		c.mu.Unlock()
		c.lifecycleMu.Unlock()
		return repoaudit.RepositoryReviewAutomation{}, errRepositoryReviewAutomationBusy
	}
	c.active[id] = &repositoryReviewActiveRun{
		runID: runID, store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation),
		guardMu:      &sync.Mutex{},
	}
	c.mu.Unlock()
	updated, err := c.update(
		ctx,
		store,
		id,
		expectedVersion,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			if candidate.CampaignID != campaignID || candidate.ResolvedCommitSHA != resolvedCommit {
				return repoaudit.ErrConflict
			}
			candidate.Status = repoaudit.RepositoryReviewAutomationRunning
			candidate.PauseReason = ""
			candidate.PauseDetail = ""
			candidate.RequestedPauseReason = ""
			candidate.RequestedPauseDetail = ""
			candidate.ActiveRunID = runID
			candidate.RunIDs = append(candidate.RunIDs, runID)
			candidate.AccountLimitSnapshots = nil
			candidate.CompletedAt = time.Time{}
			if candidate.StartedAt.IsZero() {
				candidate.StartedAt = now
			}
			candidate.Progress.Stage = "queued"
			candidate.Progress.TotalBatches = max(
				candidate.Progress.TotalBatches,
				candidate.Progress.CompletedBatches+1,
			)
			return nil
		},
	)
	if err != nil {
		c.removeActive(id, runID)
		c.lifecycleMu.Unlock()
		return repoaudit.RepositoryReviewAutomation{}, err
	}

	c.wg.Add(1)
	go c.executeAutomation(id, runID)
	c.lifecycleMu.Unlock()
	return updated, nil
}

func resolveRepositoryReviewCampaignProfile(
	ctx context.Context,
	configPath string,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
) (workflows.RepositoryReviewModelProfile, error) {
	runners := workflowRuntimeRunnersForConfig(configPath, cfg)
	resolver, ok := runners.Agents.(workflows.RepositoryReviewProfileResolver)
	if !ok {
		return workflows.RepositoryReviewModelProfile{}, errors.New(
			"repository review campaign recovery requires a profile-aware runtime",
		)
	}
	if closer, closeOK := runners.Agents.(interface{ Close() error }); closeOK {
		defer closer.Close()
	}
	return resolver.ResolveRepositoryReviewProfile(
		ctx, "main",
		repositoryReviewEffectiveAccountRef(cfg, automation.EffectiveAccountRef),
		repositoryReviewExecutionModels(automation),
	)
}

func repositoryReviewShouldRecoverLegacyCampaign(
	automation repoaudit.RepositoryReviewAutomation,
	action string,
) bool {
	resume := action == "resume"
	automaticHandoff := action == "start" && strings.EqualFold(
		strings.TrimSpace(automation.Progress.Stage), "next batch queued",
	)
	return (resume || automaticHandoff) && (automation.CampaignRecoveryPending ||
		automation.CampaignID == "" && !automation.StartedAt.IsZero())
}

func (c *repositoryReviewController) resolveRepositoryReviewAdmissionCommit(
	ctx context.Context,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
	action string,
	selection string,
) (string, error) {
	if c == nil || c.resolveCommit == nil {
		return "", errors.New("repository review commit resolver is unavailable")
	}
	selection = strings.TrimSpace(selection)
	if selection != "" && !repositoryReviewValidCommitSelection(selection) {
		return "", fmt.Errorf(
			"%w: commit_sha must be a full 40 or 64 character hexadecimal commit ID",
			repoaudit.ErrInvalidAutomation,
		)
	}
	remembered := repositoryReviewRememberedCommit(automation)
	if action == "start" && selection == "" && remembered != "" &&
		strings.EqualFold(strings.TrimSpace(automation.Progress.Stage), "next batch queued") {
		return remembered, nil
	}
	if action == "resume" && selection == "" {
		latest, err := c.resolveCommit(ctx, cfg, automation, "")
		if err != nil {
			return "", err
		}
		latest = strings.ToLower(strings.TrimSpace(latest))
		if !repositoryReviewValidCommitSHA(latest) {
			return "", errors.New("repository review resolved a noncanonical latest commit")
		}
		if remembered != "" && remembered != latest {
			return "", fmt.Errorf(
				"%w: remembered commit %s differs from latest commit %s",
				errRepositoryReviewCommitSelection,
				remembered,
				latest,
			)
		}
		if remembered != "" {
			return remembered, nil
		}
		return latest, nil
	}
	resolved, err := c.resolveCommit(ctx, cfg, automation, selection)
	if err != nil {
		if selection != "" {
			return "", fmt.Errorf(
				"%w: commit_sha could not be resolved in this repository: %v",
				repoaudit.ErrInvalidAutomation,
				err,
			)
		}
		return "", err
	}
	resolved = strings.ToLower(strings.TrimSpace(resolved))
	if !repositoryReviewValidCommitSHA(resolved) {
		return "", errors.New("repository review resolved a noncanonical commit")
	}
	return resolved, nil
}

func (c *repositoryReviewController) normalizeRepositoryReviewAutomationAdmission(
	ctx context.Context,
	store repoaudit.Store,
	automation repoaudit.RepositoryReviewAutomation,
) (repoaudit.RepositoryReviewAutomation, error) {
	normalized, err := repositoryReviewAutomationResolutionTarget(automation)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if automation.Repository == normalized.Repository &&
		automation.Ref == normalized.Ref &&
		automation.Target == "all" {
		return automation, nil
	}
	updated, err := c.update(
		ctx,
		store,
		automation.ID,
		automation.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			candidate.Repository = normalized.Repository
			candidate.Ref = normalized.Ref
			candidate.Target = "all"
			resetRepositoryReviewExecutionCampaign(candidate)
			return nil
		},
	)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	return updated, nil
}

func repositoryReviewAutomationResolutionTarget(
	automation repoaudit.RepositoryReviewAutomation,
) (repoaudit.RepositoryReviewAutomation, error) {
	repository, err := normalizeRepositoryReviewAutomationRepository(automation.Repository)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	branch := automation.Ref
	if automation.ProfileID == "" && strings.EqualFold(branch, "HEAD") {
		branch = ""
	} else {
		branch, err = repoaudit.NormalizeRepositoryReviewBranch(branch)
		if err != nil {
			return repoaudit.RepositoryReviewAutomation{}, err
		}
	}
	automation.Repository = repository
	automation.Ref = branch
	automation.Target = "all"
	return automation, nil
}

func (c *repositoryReviewController) materializeLatestRepositoryReviewProfile(
	ctx context.Context,
	store repoaudit.Store,
	automation repoaudit.RepositoryReviewAutomation,
) (repoaudit.RepositoryReviewAutomation, error) {
	if strings.TrimSpace(automation.ProfileID) == "" {
		return automation, nil
	}
	profile, found, err := store.GetProfile(ctx, automation.ProfileID)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if !found {
		return repoaudit.RepositoryReviewAutomation{}, fmt.Errorf(
			"repository review profile %q not found", automation.ProfileID,
		)
	}
	materialized, err := repoaudit.MaterializeRepositoryReviewAutomation(profile, automation)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	materialized.Name = repositoryReviewAssignedAutomationName(materialized.Repository, profile.Name)
	if repositoryReviewProfileSnapshotMatches(automation, materialized) {
		return automation, nil
	}
	cfg, err := config.LoadConfig(c.handler.configPath)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if pricingErr := repositoryReviewRefreshAccountingSnapshot(cfg, &materialized); pricingErr != nil {
		return repoaudit.RepositoryReviewAutomation{}, pricingErr
	}
	materializedPrices := maps.Clone(materialized.ModelPrices)
	updated, err := c.update(
		ctx,
		store,
		automation.ID,
		automation.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			materialized, materializeErr := repoaudit.MaterializeRepositoryReviewAutomation(
				profile, *candidate,
			)
			if materializeErr != nil {
				return materializeErr
			}
			materialized.Name = repositoryReviewAssignedAutomationName(
				materialized.Repository, profile.Name,
			)
			materialized.ModelPrices = maps.Clone(materializedPrices)
			resetRepositoryReviewExecutionCampaign(&materialized)
			*candidate = materialized
			return nil
		},
	)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	return updated, nil
}

func repositoryReviewProfileSnapshotMatches(
	automation repoaudit.RepositoryReviewAutomation,
	materialized repoaudit.RepositoryReviewAutomation,
) bool {
	return automation.ProfileID == materialized.ProfileID &&
		automation.ProfileVersion == materialized.ProfileVersion &&
		automation.AccountRef == materialized.AccountRef &&
		automation.Name == materialized.Name &&
		automation.Target == "all" &&
		automation.ReviewFocus == materialized.ReviewFocus &&
		reflect.DeepEqual(automation.ScopePolicy, materialized.ScopePolicy) &&
		reflect.DeepEqual(automation.ReviewerModels, materialized.ReviewerModels) &&
		automation.DeduplicationModel == materialized.DeduplicationModel &&
		automation.DeduplicationSimilarityThreshold == materialized.DeduplicationSimilarityThreshold &&
		automation.DeduplicationCandidateLimit == materialized.DeduplicationCandidateLimit &&
		automation.IssueWriterModel == materialized.IssueWriterModel &&
		!automation.CompareModels &&
		automation.Force == materialized.Force &&
		automation.AutoContinue == materialized.AutoContinue &&
		automation.MaxFilesPerRun == materialized.MaxFilesPerRun &&
		automation.MaxContentBytes == materialized.MaxContentBytes &&
		automation.MaxParallelChildren == materialized.MaxParallelChildren &&
		automation.AssignmentTimeoutSeconds == materialized.AssignmentTimeoutSeconds &&
		automation.EstimatedOutputTokens == materialized.EstimatedOutputTokens &&
		reflect.DeepEqual(automation.BudgetPolicy, materialized.BudgetPolicy)
}

func (c *repositoryReviewController) ensureRepositoryReviewCampaign(
	ctx context.Context,
	store repoaudit.Store,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
	resolvedCommit string,
	action string,
) (repoaudit.RepositoryReviewAutomation, error) {
	resolvedCommit = strings.ToLower(strings.TrimSpace(resolvedCommit))
	if !repositoryReviewValidCommitSHA(resolvedCommit) {
		return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrInvalidAutomation
	}
	// Complete or retry the legacy adapter before admitting any new assignment.
	// Ambiguous historical evidence intentionally falls through to a fresh empty
	// catalog on the same campaign continuation; it is never guessed into bits.
	stateSnapshot, stateFound, stateErr := store.ResolveRepositoryState(
		automation.Repository, automation.RunIDs,
	)
	if stateErr != nil {
		return repoaudit.RepositoryReviewAutomation{}, stateErr
	}
	retainedCampaignRun := false
	configuredRunIDs := make(map[string]struct{}, len(automation.RunIDs))
	for _, runID := range automation.RunIDs {
		configuredRunIDs[runID] = struct{}{}
	}
	for _, run := range stateSnapshot.Runs {
		if _, configured := configuredRunIDs[run.ID]; configured &&
			(run.CampaignID == "" || run.CampaignID == automation.CampaignID) {
			retainedCampaignRun = true
			break
		}
	}
	legacyCatalogMissing := automation.CampaignID != "" && stateFound && retainedCampaignRun &&
		stateSnapshot.CurrentCampaign != nil &&
		stateSnapshot.CurrentCampaign.ID == automation.CampaignID &&
		len(stateSnapshot.CurrentCampaign.AssignmentCatalog) == 0
	shouldRecover := automation.CampaignRecoveryPending || legacyCatalogMissing ||
		action == "resume" && automation.CampaignID == "" && len(automation.RunIDs) > 0
	if shouldRecover {
		if !stateFound {
			if automation.CampaignRecoveryPending {
				return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
			}
			shouldRecover = false
		}
	}
	if shouldRecover {
		resolvedProfile, err := c.resolveRepositoryReviewCampaignProfile(ctx, cfg, automation)
		if err != nil {
			return repoaudit.RepositoryReviewAutomation{}, err
		}
		if c.recoverCampaign == nil {
			return repoaudit.RepositoryReviewAutomation{}, errors.New(
				"legacy repository review campaign recovery is unavailable",
			)
		}
		legacy := automation
		legacy.ResolvedCommitSHA = resolvedCommit
		recovered, recoverErr := c.recoverCampaign(
			ctx, store, cfg.WorkspacePath(), legacy, resolvedCommit, resolvedProfile,
		)
		if recoverErr == nil {
			return recovered, nil
		}
		if automation.CampaignRecoveryPending ||
			(!errors.Is(recoverErr, repoaudit.ErrConflict) &&
				!errors.Is(recoverErr, os.ErrNotExist)) {
			return repoaudit.RepositoryReviewAutomation{}, recoverErr
		}
	}
	ledgerRepository := repoaudit.CanonicalRepositoryIdentity(automation.Repository)
	state, _, err := store.Get(ledgerRepository)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if automation.CampaignID != "" && automation.ResolvedCommitSHA == resolvedCommit {
		if state.CurrentCampaign == nil || state.CurrentCampaign.ID != automation.CampaignID ||
			state.CurrentCampaign.CommitSHA != resolvedCommit {
			return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
		}
		return automation, nil
	}
	expectedCampaignID := ""
	if state.CurrentCampaign != nil {
		expectedCampaignID = state.CurrentCampaign.ID
	}
	campaignID := repoaudit.NewRepositoryReviewCampaignID()
	deduplicationSnapshot, err := c.repositoryReviewDeduplicationSnapshot(automation)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if _, err = store.BeginCampaign(ctx, repoaudit.BeginCampaignRequest{
		Repository: ledgerRepository, CampaignID: campaignID,
		ExpectedCampaignID: expectedCampaignID, CommitSHA: resolvedCommit,
		ExpectedReviewVersion: state.ReviewVersion, Exact: false,
		DeduplicationSnapshot: &deduplicationSnapshot,
	}); err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	return store.UpdateAutomation(
		ctx,
		automation.ID,
		automation.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			newCampaign := candidate.CampaignID != campaignID
			if candidate.ResolvedCommitSHA != resolvedCommit {
				candidate.ScopePlan = repoaudit.RepositoryReviewScopePlan{}
				candidate.ScopeSelection = nil
			}
			if newCampaign {
				candidate.Progress = repoaudit.RepositoryReviewProgress{}
				candidate.StartedAt = c.clock()
				candidate.CompletedAt = time.Time{}
				candidate.ModelCoverageSketches = make(map[string]string)
				for alias, stats := range candidate.ModelStats {
					stats.Findings = 0
					stats.ReviewedFiles = 0
					candidate.ModelStats[alias] = stats
				}
			}
			candidate.ResolvedCommitSHA = resolvedCommit
			candidate.CampaignID = campaignID
			candidate.CampaignRecoveryPending = false
			return nil
		},
	)
}

func (c *repositoryReviewController) repositoryReviewDeduplicationSnapshot(
	automation repoaudit.RepositoryReviewAutomation,
) (repoaudit.RepositoryReviewDeduplicationSnapshot, error) {
	if c != nil && c.handler != nil {
		revision, err := config.ConfigRevision(c.handler.configPath)
		if err != nil {
			return repoaudit.RepositoryReviewDeduplicationSnapshot{}, fmt.Errorf(
				"capture repository review account/model revision: %w", err,
			)
		}
		automation.AccountModelRevision = revision
	}
	return repoaudit.RepositoryReviewDeduplicationSnapshotFromAutomation(automation)
}

func (c *repositoryReviewController) resolveRepositoryReviewCampaignProfile(
	ctx context.Context,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
) (workflows.RepositoryReviewModelProfile, error) {
	if c == nil || c.handler == nil || cfg == nil {
		return workflows.RepositoryReviewModelProfile{}, errors.New(
			"repository review model profile resolver is unavailable",
		)
	}
	_, _, executor, err := repositoryReviewCampaignWorkflowRuntime(c, ctx, cfg)
	if err != nil {
		return workflows.RepositoryReviewModelProfile{}, err
	}
	defer closeWorkflowRuntime(executor)
	resolver, ok := executor.Agents.(workflows.RepositoryReviewProfileResolver)
	if !ok || resolver == nil {
		return workflows.RepositoryReviewModelProfile{}, errors.New(
			"repository review model profile resolver is unavailable",
		)
	}
	return resolver.ResolveRepositoryReviewProfile(
		ctx,
		"main",
		repositoryReviewEffectiveAccountRef(cfg, automation.EffectiveAccountRef),
		automation.ReviewerModels,
	)
}

func resetRepositoryReviewExecutionCampaign(automation *repoaudit.RepositoryReviewAutomation) {
	if automation == nil {
		return
	}
	automation.ScopePlan = repoaudit.RepositoryReviewScopePlan{}
	automation.ScopeSelection = nil
	automation.CampaignID = ""
	automation.CampaignRecoveryPending = false
	automation.ResolvedCommitSHA = ""
	automation.ResolvedTargetBranch = ""
	automation.AdvertisedDefaultBranch = ""
	automation.TargetIsDefault = automation.Ref == ""
	automation.RequestedPauseReason = ""
	automation.RequestedPauseDetail = ""
	automation.ActiveRunID = ""
	automation.EffectiveAccountRef = ""
	automation.Usage = repoaudit.RepositoryReviewTokenUsage{}
	automation.EstimatedCostUSD = 0
	automation.Progress = repoaudit.RepositoryReviewProgress{}
	automation.ModelStats = make(map[string]repoaudit.RepositoryReviewModelStats)
	automation.ModelCoverageSketches = make(map[string]string)
	automation.AccountLimitSnapshots = nil
	automation.StartedAt = time.Time{}
	automation.CompletedAt = time.Time{}
}

func resetRepositoryReviewCampaignProgress(automation *repoaudit.RepositoryReviewAutomation) {
	if automation == nil {
		return
	}
	automation.ScopePlan = repoaudit.RepositoryReviewScopePlan{}
	automation.ScopeSelection = nil
	automation.RequestedPauseReason = ""
	automation.RequestedPauseDetail = ""
	automation.ActiveRunID = ""
	automation.Usage = repoaudit.RepositoryReviewTokenUsage{}
	automation.EstimatedCostUSD = 0
	automation.Progress = repoaudit.RepositoryReviewProgress{}
	automation.ModelStats = make(map[string]repoaudit.RepositoryReviewModelStats)
	automation.ModelCoverageSketches = make(map[string]string)
	automation.AccountLimitSnapshots = nil
	automation.StartedAt = time.Time{}
	automation.CompletedAt = time.Time{}
}

func (c *repositoryReviewController) currentLeasedConfiguration() (*config.Config, error) {
	if c == nil || c.handler == nil || c.leasedConfig == nil {
		return nil, errors.New("repository review controller lease is unavailable")
	}
	cfg, err := config.LoadConfig(c.handler.configPath)
	if err != nil {
		return nil, err
	}
	if cfg.WorkspacePath() != c.leasedConfig.WorkspacePath() {
		return nil, errors.New("repository review workspace changed; restart the launcher before controlling reviews")
	}
	return cfg, nil
}

func (c *repositoryReviewController) pauseAutomation(
	ctx context.Context,
	id string,
	expectedVersion int64,
) (repoaudit.RepositoryReviewAutomation, error) {
	return c.pauseAutomationForRun(ctx, id, expectedVersion, "")
}

func (c *repositoryReviewController) pauseAutomationForRun(
	ctx context.Context,
	id string,
	expectedVersion int64,
	expectedRunID string,
) (repoaudit.RepositoryReviewAutomation, error) {
	if err := c.Start(); err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	c.lifecycleMu.Lock()
	if c.stopped || c.ctx.Err() != nil {
		c.lifecycleMu.Unlock()
		return repoaudit.RepositoryReviewAutomation{}, context.Canceled
	}
	c.admissionWG.Add(1)
	c.lifecycleMu.Unlock()
	defer c.admissionWG.Done()
	pauseCtx, cancelPause := context.WithCancel(c.ctx)
	stopCallerCancellation := context.AfterFunc(ctx, cancelPause)
	defer func() {
		stopCallerCancellation()
		cancelPause()
	}()
	ctx = pauseCtx
	active, locallyActive := c.activeRunSnapshot(id, "")
	store := active.store
	if !locallyActive {
		store = c.leasedStore
	}
	current, found, err := store.GetAutomation(ctx, id)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if !found {
		return repoaudit.RepositoryReviewAutomation{}, os.ErrNotExist
	}
	if expectedVersion < 1 || expectedVersion > current.Version {
		return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
	}
	expectedRunID = strings.TrimSpace(expectedRunID)
	if expectedRunID == "" {
		if expectedVersion != current.Version {
			return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
		}
	} else if len(expectedRunID) > 1024 || !repositoryReviewPauseRunMatches(current, expectedRunID) {
		return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
	}
	if current.Status == repoaudit.RepositoryReviewAutomationStopping ||
		current.Status == repoaudit.RepositoryReviewAutomationPaused ||
		current.Status == repoaudit.RepositoryReviewAutomationCompleted ||
		current.Status == repoaudit.RepositoryReviewAutomationFailed {
		return current, nil
	}
	latchedRunID := current.ActiveRunID
	updated, err := c.updateLatest(ctx, store, id, func(candidate *repoaudit.RepositoryReviewAutomation) error {
		return applyRepositoryReviewPause(candidate, expectedVersion, expectedRunID)
	})
	if errors.Is(err, errRepositoryReviewPauseSettled) {
		return loadSettledRepositoryReviewPause(ctx, store, id)
	}
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	c.latchManualPause(id, latchedRunID)
	return updated, nil
}

func loadSettledRepositoryReviewPause(
	ctx context.Context,
	store repoaudit.Store,
	id string,
) (repoaudit.RepositoryReviewAutomation, error) {
	settled, found, err := store.GetAutomation(ctx, id)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if !found {
		return repoaudit.RepositoryReviewAutomation{}, os.ErrNotExist
	}
	return settled, nil
}

func applyRepositoryReviewPauseTransition(
	candidate *repoaudit.RepositoryReviewAutomation,
	expectedVersion int64,
	expectedRunID string,
) error {
	if candidate == nil {
		return errRepositoryReviewInvalidTransition
	}
	if expectedRunID == "" && candidate.Version != expectedVersion {
		return repoaudit.ErrConflict
	}
	if expectedRunID != "" && !repositoryReviewPauseRunMatches(*candidate, expectedRunID) {
		return repoaudit.ErrConflict
	}
	switch candidate.Status {
	case repoaudit.RepositoryReviewAutomationRunning:
		candidate.Status = repoaudit.RepositoryReviewAutomationStopping
		candidate.RequestedPauseReason = repoaudit.RepositoryReviewPauseManual
		candidate.RequestedPauseDetail = "Paused manually after the current safe checkpoint."
		candidate.Progress.Stage = "stopping after current batch"
	case repoaudit.RepositoryReviewAutomationIdle:
		if !candidate.AutoContinue ||
			!strings.EqualFold(strings.TrimSpace(candidate.Progress.Stage), "next batch queued") {
			return errRepositoryReviewInvalidTransition
		}
		candidate.Status = repoaudit.RepositoryReviewAutomationPaused
		candidate.PauseReason = repoaudit.RepositoryReviewPauseManual
		candidate.PauseDetail = "Paused before the next review batch started."
		candidate.Progress.Stage = "paused"
	case repoaudit.RepositoryReviewAutomationStopping,
		repoaudit.RepositoryReviewAutomationPaused,
		repoaudit.RepositoryReviewAutomationCompleted,
		repoaudit.RepositoryReviewAutomationFailed:
		return errRepositoryReviewPauseSettled
	default:
		return errRepositoryReviewInvalidTransition
	}
	return nil
}

func (c *repositoryReviewController) latchManualPause(id, runID string) {
	if c == nil || strings.TrimSpace(runID) == "" {
		return
	}
	c.mu.Lock()
	if active := c.active[id]; active != nil && active.runID == runID {
		active.pauseReason = repoaudit.RepositoryReviewPauseManual
		active.pauseDetail = "Paused manually after the current safe checkpoint."
	}
	c.mu.Unlock()
}

func repositoryReviewPauseRunMatches(
	automation repoaudit.RepositoryReviewAutomation,
	expectedRunID string,
) bool {
	expectedRunID = strings.TrimSpace(expectedRunID)
	if expectedRunID == "" {
		return false
	}
	if automation.ActiveRunID != "" {
		return automation.ActiveRunID == expectedRunID
	}
	return len(automation.RunIDs) > 0 && automation.RunIDs[len(automation.RunIDs)-1] == expectedRunID
}

func (c *repositoryReviewController) executeAutomation(id, runID string) {
	defer c.wg.Done()
	runCtx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	activeSnapshot, ok := c.activeRunSnapshot(id, runID)
	if !ok {
		c.finishAutomationRun(
			id, runID, nil,
			errors.New("repository review active run is unavailable"),
			false, nil,
		)
		return
	}
	store, cfg := activeSnapshot.store, activeSnapshot.config
	automation, found, err := store.GetAutomation(runCtx, id)
	if err != nil || !found || automation.ActiveRunID != runID {
		if err == nil {
			err = errors.New("repository review automation disappeared before execution")
		}
		c.finishAutomationRun(id, runID, nil, err, false, nil)
		return
	}
	priceIndex := repositoryReviewAccountingIndex(nil, automation)
	observeUsage := func(usage workflows.AgentUsage) error {
		return c.recordUsage(id, runID, usage, priceIndex)
	}
	if c.runBatch != nil {
		result, runErr := c.runBatch(runCtx, automation, runID, observeUsage)
		runErr = repositoryReviewJoinCommitError(automation, result, runErr)
		_, remainingFound := repositoryReviewRemainingFiles(result, nil)
		checkpointed := runErr == nil && result != nil &&
			result.Status == workflows.RunStatusSucceeded && remainingFound
		c.finishAutomationRun(id, runID, result, runErr, checkpointed, nil)
		return
	}

	_, workflowStore, executor, err := c.handler.workflowRuntimeFromConfig(runCtx, cfg)
	if err != nil {
		c.finishAutomationRun(id, runID, nil, err, false, nil)
		return
	}
	defer closeWorkflowRuntime(executor)
	workflow, err := repositoryReviewParseWorkflow([]byte(workflows.RepositoryBugFinderWorkflowYAML))
	if err != nil {
		c.finishAutomationRun(id, runID, nil, err, false, nil)
		return
	}
	executor.DefaultTimeout = repositoryReviewEffectiveWorkflowTimeoutForAssignment(
		executor.DefaultTimeout, automation.AssignmentTimeoutSeconds,
	)
	priceIndex = repositoryReviewAccountingIndex(cfg, automation)
	executor.AgentUsageObserver = repositoryReviewAgentUsageObserver(runID, observeUsage)
	executor.AgentCallAdmission = repositoryReviewAgentCallAdmissionObserver(
		runID,
		func() error { return c.admitProviderCall(id, runID) },
	)
	executor.ManagedChildActivityObserver = c.repositoryReviewManagedChildObserver(id, runID)
	executor.StepActivityObserver = c.repositoryReviewStepObserver(id, runID, store, workflowStore)

	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		c.monitorWorkflowProgress(runCtx, store, workflowStore, id, runID)
	}()
	scopePolicyJSON, _ := json.Marshal(automation.ScopePolicy)
	scopePlanned, scopeSelectionInput, scopePlanInput := repositoryReviewWorkflowScopeInputs(automation)
	result, runErr := executor.Run(runCtx, workflows.RunRequest{
		RunID:       runID,
		Workflow:    workflow,
		WorkflowRef: workflows.RepositoryBugFinderWorkflowRef,
		Inputs: map[string]any{
			"repository":                 automation.Repository,
			"automation_id":              automation.ID,
			"account_ref":                repositoryReviewEffectiveAccountRef(cfg, automation.EffectiveAccountRef),
			"ref":                        repositoryReviewExecutionRef(automation),
			"target_branch":              automation.ResolvedTargetBranch,
			"advertised_default_branch":  automation.AdvertisedDefaultBranch,
			"target_is_default":          automation.TargetIsDefault,
			"target":                     automation.Target,
			"review_focus":               automation.ReviewFocus,
			"review_models":              strings.Join(repositoryReviewExecutionModels(automation), ","),
			"planner_model":              repositoryReviewPlannerModel(automation),
			"scope_policy":               string(scopePolicyJSON),
			"scope_planned":              scopePlanned,
			"scope_selection":            scopeSelectionInput,
			"scope_plan":                 scopePlanInput,
			"force":                      automation.Force,
			"max_content_bytes":          automation.MaxContentBytes,
			"max_files_per_run":          automation.MaxFilesPerRun,
			"max_parallel_children":      automation.MaxParallelChildren,
			"assignment_timeout_seconds": automation.AssignmentTimeoutSeconds,
			"campaign_id":                automation.CampaignID,
			"estimated_output_tokens":    automation.EstimatedOutputTokens,
		},
	})
	runErr = errors.Join(runErr, repositoryReviewValidateExecutionCommit(automation, result))
	checkpointed := false
	var persistedRun *workflows.Run
	if persisted, persistedErr := workflowStore.GetRun(context.Background(), runID); persistedErr == nil {
		persistedRun = persisted
		runErr = errors.Join(runErr, repositoryReviewValidatePersistedScope(automation, persisted))
		c.recordManagedChildOutcomes(id, runID, persisted, priceIndex)
		checkpointed = runErr == nil && repositoryReviewRunCheckpointed(persisted, result)
	}
	cancel()
	<-monitorDone
	c.finishAutomationRun(id, runID, result, runErr, checkpointed, persistedRun)
}

func repositoryReviewEffectiveWorkflowTimeout(configured time.Duration) time.Duration {
	return repositoryReviewEffectiveWorkflowTimeoutForAssignment(
		configured, repoaudit.DefaultRepositoryReviewAssignmentTimeoutSeconds,
	)
}

func repositoryReviewEffectiveWorkflowTimeoutForAssignment(
	configured time.Duration,
	assignmentTimeoutSeconds int,
) time.Duration {
	if assignmentTimeoutSeconds <= 0 {
		assignmentTimeoutSeconds = repoaudit.DefaultRepositoryReviewAssignmentTimeoutSeconds
	}
	return max(
		configured,
		time.Duration(assignmentTimeoutSeconds)*time.Second+repositoryReviewWorkflowCleanupReserve,
	)
}

func repositoryReviewWorkflowObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var object map[string]any
	if json.Unmarshal(encoded, &object) != nil || object == nil {
		return map[string]any{}
	}
	return object
}

func repositoryReviewWorkflowScopeInputs(
	automation repoaudit.RepositoryReviewAutomation,
) (bool, map[string]any, map[string]any) {
	if automation.ScopeSelection == nil {
		return false, map[string]any{}, map[string]any{}
	}
	return true,
		repositoryReviewWorkflowObject(automation.ScopeSelection),
		repositoryReviewWorkflowObject(automation.ScopePlan)
}

func repositoryReviewJoinCommitError(
	automation repoaudit.RepositoryReviewAutomation,
	result *workflows.RunResult,
	runErr error,
) error {
	return errors.Join(
		runErr,
		repositoryReviewValidateExecutionCommit(automation, result),
		repositoryReviewValidateExecutionScope(automation, result),
	)
}

func repositoryReviewValidateExecutionScope(
	automation repoaudit.RepositoryReviewAutomation,
	result *workflows.RunResult,
) error {
	if automation.ScopeSelection == nil {
		return nil
	}
	if result == nil || result.Outputs == nil {
		return errors.New("repository review omitted its frozen scope result")
	}
	return repositoryReviewValidateScopeOutputs(automation, result.Outputs)
}

func repositoryReviewValidatePersistedScope(
	automation repoaudit.RepositoryReviewAutomation,
	run *workflows.Run,
) error {
	if automation.ScopeSelection == nil {
		return nil
	}
	scope := repositoryReviewRunStep(run, "scope")
	if scope.Status != workflows.RunStatusSucceeded || strings.TrimSpace(scope.Error) != "" {
		return errors.New("repository review omitted its frozen scope result")
	}
	return repositoryReviewValidateScopeOutputs(automation, scope.Outputs)
}

func repositoryReviewValidateScopeOutputs(
	automation repoaudit.RepositoryReviewAutomation,
	outputs map[string]any,
) error {
	rawSelection, selectionFound := repositoryReviewOutputValue(
		outputs, "scopeSelection", "scope_selection",
	)
	rawPlan, planFound := repositoryReviewOutputValue(outputs, "scopePlan", "scope_plan")
	if !selectionFound || !planFound {
		return errors.New("repository review omitted its frozen scope result")
	}
	var selection repoaudit.RepositoryReviewScopeSelection
	var plan repoaudit.RepositoryReviewScopePlan
	if !repositoryReviewDecodeValue(rawSelection, &selection) ||
		!repositoryReviewDecodeValue(rawPlan, &plan) {
		return errors.New("repository review returned an invalid frozen scope result")
	}
	if !repositoryReviewScopeSelectionsEqual(selection, *automation.ScopeSelection) ||
		!repositoryReviewScopePlansEqual(plan, automation.ScopePlan) {
		return errors.New("repository review changed its frozen campaign scope")
	}
	return nil
}

func (c *repositoryReviewController) repositoryReviewStepObserver(
	id string,
	runID string,
	store repoaudit.Store,
	workflowStore *workflows.FileRunStore,
) workflows.StepActivityObserver {
	return func(event workflows.StepActivityEvent) error {
		if event.RunID != runID || event.StepID != "scope_files" {
			return nil
		}
		if workflowStore == nil {
			return errors.New("repository review workflow store is unavailable")
		}
		run, err := workflowStore.GetRun(context.Background(), runID)
		if err != nil {
			return fmt.Errorf("load repository review frozen scope: %w", err)
		}
		return c.persistRepositoryReviewFrozenScope(store, id, runID, run)
	}
}

func (c *repositoryReviewController) persistRepositoryReviewFrozenScope(
	store repoaudit.Store,
	id string,
	runID string,
	run *workflows.Run,
) error {
	scope := repositoryReviewRunStep(run, "scope")
	if scope.Status != workflows.RunStatusSucceeded || strings.TrimSpace(scope.Error) != "" {
		return errors.New("repository review scope was not durably validated")
	}
	rawSelection, selectionFound := repositoryReviewOutputValue(
		scope.Outputs, "scopeSelection", "scope_selection",
	)
	rawPlan, planFound := repositoryReviewOutputValue(scope.Outputs, "scopePlan", "scope_plan")
	var selection repoaudit.RepositoryReviewScopeSelection
	var plan repoaudit.RepositoryReviewScopePlan
	if !selectionFound || !planFound ||
		!repositoryReviewDecodeValue(rawSelection, &selection) ||
		!repositoryReviewDecodeValue(rawPlan, &plan) {
		return errors.New("repository review scope step omitted its durable selection")
	}
	_, err := c.updateLatest(
		context.Background(),
		store,
		id,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			if candidate.ActiveRunID != runID {
				return errRepositoryReviewSafeStop
			}
			if candidate.ScopeSelection != nil {
				if !repositoryReviewScopeSelectionsEqual(*candidate.ScopeSelection, selection) ||
					!repositoryReviewScopePlansEqual(candidate.ScopePlan, plan) {
					return errors.New("repository review changed its frozen campaign scope")
				}
				return nil
			}
			if remembered := repositoryReviewRememberedCommit(*candidate); remembered != "" &&
				plan.CommitSHA != remembered {
				return errors.New("repository review scope plan does not match its admitted commit")
			}
			candidate.ScopeSelection = &selection
			candidate.ScopePlan = plan
			return nil
		},
	)
	return err
}

func repositoryReviewScopeSelectionsEqual(
	left repoaudit.RepositoryReviewScopeSelection,
	right repoaudit.RepositoryReviewScopeSelection,
) bool {
	return slices.Equal(left.IncludePrefixes, right.IncludePrefixes) &&
		slices.Equal(left.ExcludePrefixes, right.ExcludePrefixes) &&
		slices.Equal(left.CandidateIDs, right.CandidateIDs) &&
		slices.Equal(left.HotpathCandidateIDs, right.HotpathCandidateIDs)
}

func repositoryReviewScopePlansEqual(
	left repoaudit.RepositoryReviewScopePlan,
	right repoaudit.RepositoryReviewScopePlan,
) bool {
	return left.CommitSHA == right.CommitSHA && left.PolicyHash == right.PolicyHash &&
		left.Hash == right.Hash && left.Summary == right.Summary &&
		left.Rationale == right.Rationale && left.Counts == right.Counts &&
		slices.Equal(left.Warnings, right.Warnings)
}

func repositoryReviewOutputValue(outputs map[string]any, names ...string) (any, bool) {
	for _, name := range names {
		if value, exists := outputs[name]; exists && value != nil {
			return value, true
		}
	}
	return nil, false
}

func repositoryReviewDecodeValue(value any, destination any) bool {
	encoded, err := json.Marshal(value)
	return err == nil && json.Unmarshal(encoded, destination) == nil
}

func (c *repositoryReviewController) repositoryReviewManagedChildObserver(
	id string,
	runID string,
) workflows.ManagedChildActivityEventObserver {
	return func(event workflows.ManagedChildActivityEvent) error {
		if event.RunID != runID || event.StepID != "review" {
			return nil
		}
		return c.observeRepositoryReviewTask(id, runID, event.ManagedChildActivity)
	}
}

func repositoryReviewWorkflowRef(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return ""
	}
	return "refs/heads/" + branch
}

func repositoryReviewExecutionRef(automation repoaudit.RepositoryReviewAutomation) string {
	if commit := repositoryReviewRememberedCommit(automation); commit != "" {
		return commit
	}
	return repositoryReviewWorkflowRef(automation.Ref)
}

func repositoryReviewValidateExecutionCommit(
	automation repoaudit.RepositoryReviewAutomation,
	result *workflows.RunResult,
) error {
	expected := repositoryReviewRememberedCommit(automation)
	if expected == "" || result == nil || result.Outputs == nil {
		return nil
	}
	actual := strings.ToLower(strings.TrimSpace(fmt.Sprint(result.Outputs["commit"])))
	if actual == "" || actual == "<nil>" {
		return nil
	}
	if actual != expected {
		return fmt.Errorf(
			"repository review workflow resolved commit %s, want remembered commit %s",
			actual,
			expected,
		)
	}
	return nil
}

func repositoryReviewAgentUsageObserver(
	runID string,
	observe workflows.AgentUsageObserver,
) workflows.AgentUsageEventObserver {
	return func(event workflows.AgentUsageEvent) error {
		if event.RunID != runID {
			return nil
		}
		return observe(event.Usage)
	}
}

func repositoryReviewAgentCallAdmissionObserver(
	runID string,
	admit workflows.AgentCallAdmission,
) workflows.AgentCallAdmissionEventObserver {
	return func(event workflows.AgentCallAdmissionEvent) error {
		if event.RunID != runID {
			return nil
		}
		return admit()
	}
}

func repositoryReviewExecutionModels(automation repoaudit.RepositoryReviewAutomation) []string {
	if automation.CompareModels || len(automation.ReviewerModels) <= 1 {
		return append([]string(nil), automation.ReviewerModels...)
	}
	return append([]string(nil), automation.ReviewerModels[0])
}

func repositoryReviewPlannerModel(automation repoaudit.RepositoryReviewAutomation) string {
	models := repositoryReviewExecutionModels(automation)
	if len(models) == 0 {
		return ""
	}
	return models[0]
}

type repositoryReviewAccountingModel struct {
	alias string
	price repoaudit.RepositoryReviewModelPrice
	known bool
}

func repositoryReviewAccountingIndex(
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
) map[string]repositoryReviewAccountingModel {
	index := make(map[string]repositoryReviewAccountingModel)
	conservative := repositoryReviewAccountingModel{}
	aliases := make(map[string]config.ModelAliasConfig)
	if cfg != nil {
		for _, alias := range cfg.ModelAliases {
			aliases[alias.Name] = alias
		}
	}
	for _, aliasName := range repositoryReviewExecutionModels(automation) {
		price, known := automation.ModelPrices[aliasName]
		entry := repositoryReviewAccountingModel{alias: aliasName, price: price, known: known}
		if known {
			conservative.known = true
			conservative.price.InputPricePer1M = max(
				conservative.price.InputPricePer1M, price.InputPricePer1M,
			)
			conservative.price.OutputPricePer1M = max(
				conservative.price.OutputPricePer1M, price.OutputPricePer1M,
			)
		}
		index[aliasName] = entry
		if alias, exists := aliases[aliasName]; exists {
			for _, model := range append([]string{alias.Model}, mapStringValues(alias.AccountOverrides)...) {
				model = strings.TrimSpace(model)
				if model == "" {
					continue
				}
				index[model] = entry
				if _, concrete, ok := strings.Cut(model, "/"); ok && concrete != "" {
					index[concrete] = entry
				}
			}
		}
	}
	if models := repositoryReviewExecutionModels(automation); len(models) == 1 {
		index[""] = index[models[0]]
	}
	if conservative.known {
		index["*"] = conservative
	}
	return index
}

func mapStringValues(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, values[key])
	}
	return out
}

func (c *repositoryReviewController) recordUsage(
	id string,
	runID string,
	usage workflows.AgentUsage,
	priceIndex map[string]repositoryReviewAccountingModel,
) error {
	activeSnapshot, ok := c.activeRunSnapshot(id, runID)
	if !ok {
		return c.latchAccountingFailure(id, runID, errors.New("repository review active store is unavailable"))
	}
	store := activeSnapshot.store
	accounting, known := priceIndex[strings.TrimSpace(usage.Reviewer)]
	if !known {
		accounting, known = priceIndex[strings.TrimSpace(usage.Model)]
	}
	if !known {
		accounting, known = priceIndex[""]
	}
	if !known {
		accounting, known = priceIndex["*"]
	}
	priceKnown := accounting.known &&
		(accounting.price.InputPricePer1M > 0 || accounting.price.OutputPricePer1M > 0)
	promptTokens := max(0, usage.PromptTokens)
	completionTokens := max(0, usage.CompletionTokens)
	totalTokens := max(max(0, usage.TotalTokens), promptTokens+completionTokens)
	cachedTokens := min(max(0, usage.CachedTokens), promptTokens)
	cost := 0.0
	if priceKnown {
		cost = (float64(promptTokens)*accounting.price.InputPricePer1M +
			float64(completionTokens)*accounting.price.OutputPricePer1M) / 1_000_000
	}
	updated, updateErr := c.updateLatest(
		context.Background(),
		store,
		id,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			if candidate.ActiveRunID != runID {
				return nil
			}
			candidate.Usage.PromptTokens += int64(promptTokens)
			candidate.Usage.CompletionTokens += int64(completionTokens)
			candidate.Usage.CachedTokens += int64(cachedTokens)
			candidate.Usage.TotalTokens += int64(totalTokens)
			candidate.EstimatedCostUSD += cost
			if known && accounting.alias != "" {
				stats := candidate.ModelStats[accounting.alias]
				stats.Tokens.PromptTokens += int64(promptTokens)
				stats.Tokens.CompletionTokens += int64(completionTokens)
				stats.Tokens.CachedTokens += int64(cachedTokens)
				stats.Tokens.TotalTokens += int64(totalTokens)
				stats.EstimatedCostUSD += cost
				stats.Requests++
				stats.LatencyMillis += max(int64(0), usage.LatencyMillis)
				candidate.ModelStats[accounting.alias] = stats
			}
			return nil
		},
	)
	if updateErr != nil {
		logger.WarnCF("repository-review", "Failed to persist repository review usage", map[string]any{
			"automation_id": id, "run_id": runID, "error": updateErr.Error(),
		})
		return c.latchAccountingFailure(id, runID, updateErr)
	}
	_ = updated
	return nil
}

func (c *repositoryReviewController) latchAccountingFailure(
	id string,
	runID string,
	accountingErr error,
) error {
	detail := "Usage accounting failed closed; no additional review work will be admitted."
	if accountingErr != nil {
		detail += " " + accountingErr.Error()
	}
	detail = repositoryReviewBoundedDetail(detail)
	c.mu.Lock()
	if active := c.active[id]; active != nil && active.runID == runID {
		active.pauseReason = repoaudit.RepositoryReviewPauseRunFailed
		active.pauseDetail = detail
	}
	c.mu.Unlock()
	return errors.Join(errRepositoryReviewSafeStop, accountingErr)
}

type repositoryReviewChildOutcome struct {
	failures      int64
	reviewedPaths []string
}

func (c *repositoryReviewController) recordManagedChildOutcomes(
	id string,
	runID string,
	run *workflows.Run,
	accountingIndex map[string]repositoryReviewAccountingModel,
) {
	if run == nil {
		return
	}
	review := repositoryReviewRunStep(run, "review")
	children := repositoryReviewAnySlice(review.Outputs["managed_children"])
	if len(children) == 0 {
		return
	}
	outcomes := make(map[string]repositoryReviewChildOutcome)
	for _, raw := range children {
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		modelMeta, _ := child["model"].(map[string]any)
		model := ""
		if requested, exists := modelMeta["requested"]; exists && requested != nil {
			model = strings.TrimSpace(fmt.Sprint(requested))
		}
		if model == "" || model == "<nil>" {
			model = strings.TrimSpace(fmt.Sprint(modelMeta["selected"]))
		}
		accounting, found := accountingIndex[model]
		if !found {
			accounting, found = accountingIndex[""]
		}
		if !found || accounting.alias == "" {
			continue
		}
		admitted, _ := child["admitted"].(bool)
		if !admitted {
			continue
		}
		outcome := outcomes[accounting.alias]
		valid, _ := child["valid"].(bool)
		runError := ""
		if rawError, exists := child["run_error"]; exists && rawError != nil {
			runError = strings.TrimSpace(fmt.Sprint(rawError))
		}
		if !valid || runError != "" {
			outcome.failures++
			outcomes[accounting.alias] = outcome
			continue
		}
		scopePaths := make(map[string]struct{})
		for _, scopeRaw := range repositoryReviewAnySlice(child["scope"]) {
			if scope, ok := scopeRaw.(map[string]any); ok {
				if path := strings.TrimSpace(fmt.Sprint(scope["path"])); path != "" && path != "<nil>" {
					scopePaths[path] = struct{}{}
				}
			}
		}
		acknowledged := make(map[string]struct{})
		if structured, ok := child["structured"].(map[string]any); ok {
			reviewed := repositoryReviewAnySlice(structured["reviewedFiles"])
			if len(reviewed) == 0 {
				reviewed = repositoryReviewAnySlice(structured["reviewed_files"])
			}
			for _, rawPath := range reviewed {
				path := strings.TrimSpace(fmt.Sprint(rawPath))
				if _, assigned := scopePaths[path]; assigned {
					acknowledged[path] = struct{}{}
				}
			}
		}
		for path := range acknowledged {
			outcome.reviewedPaths = append(outcome.reviewedPaths, path)
		}
		outcomes[accounting.alias] = outcome
	}
	if len(outcomes) == 0 {
		return
	}
	activeSnapshot, ok := c.activeRunSnapshot(id, runID)
	if !ok {
		return
	}
	store := activeSnapshot.store
	_, _ = c.updateLatest(context.Background(), store, id, func(candidate *repoaudit.RepositoryReviewAutomation) error {
		if candidate.ActiveRunID != runID {
			return nil
		}
		for alias, outcome := range outcomes {
			stats := candidate.ModelStats[alias]
			stats.Failures += outcome.failures
			stats.Requests = max(stats.Requests, stats.Failures)
			candidate.ModelStats[alias] = stats
			addRepositoryReviewModelPaths(candidate, alias, outcome.reviewedPaths)
		}
		return nil
	})
}

func (c *repositoryReviewController) admitProviderCall(id, runID string) error {
	if c == nil {
		return errRepositoryReviewSafeStop
	}
	if err := c.ctx.Err(); err != nil {
		return errors.Join(errRepositoryReviewSafeStop, err)
	}
	c.mu.Lock()
	active := c.active[id]
	if active == nil || active.runID != runID {
		c.mu.Unlock()
		return fmt.Errorf("%w: repository review is no longer active", errRepositoryReviewSafeStop)
	}
	reason, detail := active.pauseReason, active.pauseDetail
	c.mu.Unlock()
	if reason != "" && reason != repoaudit.RepositoryReviewPauseGuardExpression {
		return fmt.Errorf("%w: %s", errRepositoryReviewSafeStop, detail)
	}
	return nil
}

func (c *repositoryReviewController) observeRepositoryReviewTask(
	id string,
	runID string,
	activity workflows.ManagedChildActivity,
) error {
	if activity.Phase == workflows.ManagedChildCompleted {
		c.mu.Lock()
		if active := c.active[id]; active != nil && active.runID == runID {
			delete(active.reservations, activity.Index)
		}
		c.mu.Unlock()
		return nil
	}
	if activity.Phase != workflows.ManagedChildStarted {
		return nil
	}
	activeSnapshot, ok := c.activeRunSnapshot(id, runID)
	if !ok {
		return fmt.Errorf("%w: repository review is no longer active", errRepositoryReviewSafeStop)
	}
	if activeSnapshot.pauseReason != "" {
		return fmt.Errorf("%w: %s", errRepositoryReviewSafeStop, activeSnapshot.pauseDetail)
	}
	guardMu := activeSnapshot.guardMu
	if guardMu == nil {
		c.mu.Lock()
		if active := c.active[id]; active != nil && active.runID == runID {
			if active.guardMu == nil {
				active.guardMu = &sync.Mutex{}
			}
			guardMu = active.guardMu
		}
		c.mu.Unlock()
	}
	if guardMu == nil {
		return fmt.Errorf("%w: task admission guard is unavailable", errRepositoryReviewSafeStop)
	}
	guardMu.Lock()
	defer guardMu.Unlock()
	activeSnapshot, ok = c.activeRunSnapshot(id, runID)
	if !ok || activeSnapshot.pauseReason != "" {
		detail := "repository review stopped before task admission"
		if ok && activeSnapshot.pauseDetail != "" {
			detail = activeSnapshot.pauseDetail
		}
		return fmt.Errorf("%w: %s", errRepositoryReviewSafeStop, detail)
	}
	automation, found, err := activeSnapshot.store.GetAutomation(c.ctx, id)
	if err != nil || !found || automation.ActiveRunID != runID {
		if err == nil {
			err = errors.New("repository review automation is unavailable at task admission")
		}
		return errors.Join(errRepositoryReviewSafeStop, err)
	}
	expression := strings.TrimSpace(automation.BudgetPolicy.GuardExpression)
	if expression == "" {
		return nil
	}

	reservation := repositoryReviewGuardReservation(automation, activity)

	environment := repoaudit.RepositoryReviewGuardEnvironment{
		SpentTokens:   automation.Usage,
		SpendTotalUSD: automation.EstimatedCostUSD,
		CostKnown:     repositoryReviewAutomationPriceKnown(automation),
	}
	c.mu.Lock()
	if active := c.active[id]; active != nil && active.runID == runID {
		for _, pending := range active.reservations {
			addRepositoryReviewGuardReservation(&environment, pending)
		}
	}
	c.mu.Unlock()
	addRepositoryReviewGuardReservation(&environment, reservation)

	if repoaudit.RepositoryReviewGuardUsesAccountLimits(expression) {
		snapshots, known, probeErr := c.repositoryReviewGuardAccountLimits(
			c.ctx, activeSnapshot.config, automation,
		)
		environment.AccountLimitSnapshots = snapshots
		environment.AccountLimitsKnown = known && probeErr == nil
		if len(snapshots) > 0 {
			_, _ = c.updateLatest(
				context.Background(), activeSnapshot.store, id,
				func(candidate *repoaudit.RepositoryReviewAutomation) error {
					if candidate.ActiveRunID == runID {
						candidate.AccountLimitSnapshots = snapshots
					}
					return nil
				},
			)
		}
	}

	allowed, evaluateErr := repoaudit.EvaluateRepositoryReviewGuardExpression(expression, environment)
	if evaluateErr != nil || !allowed {
		detail := "Task admission guard evaluated to false."
		if evaluateErr != nil {
			detail = "Task admission guard could not produce true: " + evaluateErr.Error()
		}
		detail = repositoryReviewBoundedDetail(detail)
		c.requestSafeStop(id, runID, repoaudit.RepositoryReviewPauseGuardExpression, detail)
		return fmt.Errorf("%w: %s", errRepositoryReviewSafeStop, detail)
	}

	c.mu.Lock()
	active := c.active[id]
	if active == nil || active.runID != runID || active.pauseReason != "" {
		detail := "repository review stopped before task dispatch"
		if active != nil && active.pauseDetail != "" {
			detail = active.pauseDetail
		}
		c.mu.Unlock()
		return fmt.Errorf("%w: %s", errRepositoryReviewSafeStop, detail)
	}
	active.reservations[activity.Index] = reservation
	c.mu.Unlock()
	return nil
}

func repositoryReviewGuardReservation(
	automation repoaudit.RepositoryReviewAutomation,
	activity workflows.ManagedChildActivity,
) repositoryReviewTaskReservation {
	reservation := repositoryReviewTaskReservation{
		PromptTokens:     int64(max(0, activity.EstimatedPromptTokens)),
		CompletionTokens: int64(max(0, activity.EstimatedOutputTokens)),
	}
	reservation.TotalTokens = reservation.PromptTokens + reservation.CompletionTokens
	alias := strings.TrimSpace(activity.ModelAlias)
	price, found := automation.ModelPrices[alias]
	if !found {
		if models := repositoryReviewExecutionModels(automation); len(models) == 1 {
			price, found = automation.ModelPrices[models[0]]
		}
	}
	reservation.CostKnown = found && (price.InputPricePer1M > 0 || price.OutputPricePer1M > 0)
	if reservation.CostKnown {
		reservation.CostUSD = (float64(reservation.PromptTokens)*price.InputPricePer1M +
			float64(reservation.CompletionTokens)*price.OutputPricePer1M) / 1_000_000
	}
	return reservation
}

func addRepositoryReviewGuardReservation(
	environment *repoaudit.RepositoryReviewGuardEnvironment,
	reservation repositoryReviewTaskReservation,
) {
	if environment == nil {
		return
	}
	environment.SpentTokens.PromptTokens += reservation.PromptTokens
	environment.SpentTokens.CompletionTokens += reservation.CompletionTokens
	environment.SpentTokens.TotalTokens += reservation.TotalTokens
	environment.SpendTotalUSD += reservation.CostUSD
	environment.CostKnown = environment.CostKnown && reservation.CostKnown
}

func repositoryReviewAutomationPriceKnown(automation repoaudit.RepositoryReviewAutomation) bool {
	models := repositoryReviewExecutionModels(automation)
	if len(models) == 0 {
		return false
	}
	for _, model := range models {
		price, ok := automation.ModelPrices[model]
		if !ok || price.InputPricePer1M <= 0 && price.OutputPricePer1M <= 0 {
			return false
		}
	}
	return true
}

func repositoryReviewAnySlice(value any) []any {
	switch values := value.(type) {
	case []any:
		return values
	case []map[string]any:
		out := make([]any, len(values))
		for index := range values {
			out[index] = values[index]
		}
		return out
	default:
		return nil
	}
}

func (c *repositoryReviewController) requestSafeStop(
	id, runID string,
	reason repoaudit.RepositoryReviewPauseReason,
	detail string,
) {
	c.mu.Lock()
	active := c.active[id]
	if active != nil && active.runID == runID && active.pauseReason == "" {
		active.pauseReason = reason
		active.pauseDetail = repositoryReviewBoundedDetail(detail)
	}
	c.mu.Unlock()
	activeSnapshot, ok := c.activeRunSnapshot(id, runID)
	if !ok {
		return
	}
	store := activeSnapshot.store
	_, _ = c.updateLatest(context.Background(), store, id, func(candidate *repoaudit.RepositoryReviewAutomation) error {
		if candidate.ActiveRunID == runID && candidate.Status == repoaudit.RepositoryReviewAutomationRunning {
			candidate.Status = repoaudit.RepositoryReviewAutomationStopping
			candidate.RequestedPauseReason = reason
			candidate.RequestedPauseDetail = repositoryReviewBoundedDetail(detail)
			candidate.Progress.Stage = "stopping after current batch"
		}
		return nil
	})
}

func (c *repositoryReviewController) monitorWorkflowProgress(
	ctx context.Context,
	automationStore repoaudit.Store,
	workflowStore *workflows.FileRunStore,
	automationID string,
	runID string,
) {
	ticker := time.NewTicker(c.progressEvery)
	defer ticker.Stop()
	lastStage := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run, err := workflowStore.GetRun(ctx, runID)
			if err != nil || run == nil {
				continue
			}
			stage := repositoryReviewWorkflowStage(run)
			if stage == "" || stage == lastStage {
				continue
			}
			lastStage = stage
			_, _ = c.updateLatest(
				context.Background(),
				automationStore,
				automationID,
				func(candidate *repoaudit.RepositoryReviewAutomation) error {
					if candidate.ActiveRunID == runID &&
						candidate.Status == repoaudit.RepositoryReviewAutomationRunning {
						candidate.Progress.Stage = stage
					}
					return nil
				},
			)
		}
	}
}

func repositoryReviewWorkflowStage(run *workflows.Run) string {
	if run == nil {
		return ""
	}
	labels := map[string]string{
		"checkout":           "Acquiring repository snapshot",
		"inventory":          "Inventorying tracked files",
		"scope_catalog":      "Classifying target code",
		"release_structure":  "Releasing checkout before scope planning",
		"plan_scope":         "AI planning target scope",
		"scope_checkout":     "Reacquiring the exact commit",
		"scope_inventory":    "Validating target inventory",
		"full_scope_catalog": "Rebuilding complete target scope",
		"scope":              "Validating AI target scope",
		"scope_files":        "Binding exact target files",
		"plan":               "Planning changed files",
		"freeze":             "Freezing immutable evidence",
		"begin_assignments":  "Reserving review assignments",
		"release":            "Releasing checkout",
		"review":             "Reviewing bounded file batch",
		"record":             "Checkpointing findings",
		"result":             "Finalizing batch",
	}
	order := []string{
		"checkout", "inventory", "scope_catalog", "release_structure", "plan_scope",
		"scope_checkout", "scope_inventory", "full_scope_catalog", "scope", "scope_files",
		"plan", "freeze", "begin_assignments", "release", "review", "record", "result",
	}
	for index := len(order) - 1; index >= 0; index-- {
		step := repositoryReviewRunStep(run, order[index])
		if step.Status == workflows.RunStatusRunning {
			return labels[order[index]]
		}
	}
	for index := len(order) - 1; index >= 0; index-- {
		step := repositoryReviewRunStep(run, order[index])
		if step.Status == workflows.RunStatusSucceeded {
			return labels[order[index]]
		}
	}
	return "queued"
}

func repositoryReviewRunStep(run *workflows.Run, stepID string) workflows.StepExecution {
	if run == nil {
		return workflows.StepExecution{}
	}
	if step, exists := run.Steps["find_bugs/"+stepID]; exists {
		return step
	}
	if step, exists := run.Steps[stepID]; exists {
		return step
	}
	return workflows.StepExecution{}
}

func repositoryReviewRunCheckpointed(
	run *workflows.Run,
	result *workflows.RunResult,
) bool {
	if run == nil || result == nil || result.Status != workflows.RunStatusSucceeded {
		return false
	}
	record := repositoryReviewRunStep(run, "record")
	if record.Status == workflows.RunStatusSucceeded {
		if strings.TrimSpace(record.Error) != "" {
			return false
		}
		durableRemaining, durableRemainingFound := repositoryReviewDurableRecordRemainingFiles(run)
		remaining, remainingFound := repositoryReviewRemainingFiles(result, run)
		return durableRemainingFound && remainingFound && remaining == durableRemaining
	}
	plan := repositoryReviewRunStep(run, "plan")
	resultStep := repositoryReviewRunStep(run, "result")
	pending, pendingFound := repositoryReviewOutputNonnegativeInt(
		plan.Outputs, "pendingCount", "pending_count",
	)
	durableRemaining, durableRemainingFound := repositoryReviewDurableResultRemainingFiles(run)
	remaining, remainingFound := repositoryReviewRemainingFiles(result, run)
	return plan.Status == workflows.RunStatusSucceeded && strings.TrimSpace(plan.Error) == "" &&
		pendingFound && pending == 0 &&
		resultStep.Status == workflows.RunStatusSucceeded && strings.TrimSpace(resultStep.Error) == "" &&
		durableRemainingFound && durableRemaining == 0 && remainingFound && remaining == 0
}

func repositoryReviewRemainingFiles(
	result *workflows.RunResult,
	run *workflows.Run,
) (int, bool) {
	topLevel, topLevelFound, topLevelConflict := repositoryReviewTopLevelRemainingFilesDetailed(result)
	if topLevelConflict {
		return 0, false
	}
	record, recordFound := repositoryReviewDurableRecordRemainingFiles(run)
	persistedResult, persistedResultFound := repositoryReviewDurableResultRemainingFiles(run)
	remaining := 0
	found := false
	for _, source := range []struct {
		value int
		found bool
	}{
		{value: topLevel, found: topLevelFound},
		{value: record, found: recordFound},
		{value: persistedResult, found: persistedResultFound},
	} {
		if !source.found {
			continue
		}
		if found && remaining != source.value {
			return 0, false
		}
		remaining = source.value
		found = true
	}
	return remaining, found
}

func repositoryReviewTopLevelRemainingFilesDetailed(
	result *workflows.RunResult,
) (int, bool, bool) {
	if result == nil {
		return 0, false, false
	}
	return repositoryReviewOutputNonnegativeIntDetailed(
		result.Outputs, "remainingFiles", "remaining_files",
	)
}

func repositoryReviewDurableRecordRemainingFiles(run *workflows.Run) (int, bool) {
	record := repositoryReviewRunStep(run, "record")
	if record.Status != workflows.RunStatusSucceeded || strings.TrimSpace(record.Error) != "" {
		return 0, false
	}
	recordedRun, ok := record.Outputs["run"].(map[string]any)
	if !ok {
		return 0, false
	}
	return repositoryReviewOutputNonnegativeInt(recordedRun, "remaining_files")
}

func repositoryReviewDurableRecordFileCounts(run *workflows.Run) (int, int, bool) {
	record := repositoryReviewRunStep(run, "record")
	if record.Status != workflows.RunStatusSucceeded || strings.TrimSpace(record.Error) != "" {
		return 0, 0, false
	}
	recordedRun, ok := record.Outputs["run"].(map[string]any)
	if !ok {
		return 0, 0, false
	}
	reviewed, reviewedFound := repositoryReviewOutputNonnegativeInt(
		recordedRun,
		"reviewed_files",
	)
	unsupported, unsupportedFound := repositoryReviewOutputNonnegativeInt(
		recordedRun,
		"unsupported_files",
	)
	if !reviewedFound || !unsupportedFound {
		return 0, 0, false
	}
	return reviewed, unsupported, true
}

func repositoryReviewDurableResultRemainingFiles(run *workflows.Run) (int, bool) {
	result := repositoryReviewRunStep(run, "result")
	if result.Status != workflows.RunStatusSucceeded || strings.TrimSpace(result.Error) != "" {
		return 0, false
	}
	recordedRun, ok := result.Outputs["run"].(map[string]any)
	if !ok {
		return 0, false
	}
	return repositoryReviewOutputNonnegativeInt(recordedRun, "remaining_files")
}

func repositoryReviewOutputNonnegativeInt(
	values map[string]any,
	keys ...string,
) (int, bool) {
	parsed, found, conflict := repositoryReviewOutputNonnegativeIntDetailed(values, keys...)
	return parsed, found && !conflict
}

func repositoryReviewOutputNonnegativeIntDetailed(
	values map[string]any,
	keys ...string,
) (int, bool, bool) {
	parsed := 0
	found := false
	for _, key := range keys {
		value, exists := values[key]
		if !exists {
			continue
		}
		if candidate, ok := repositoryReviewNonnegativeInt(value); ok {
			if found && candidate != parsed {
				return 0, true, true
			}
			parsed = candidate
			found = true
		}
	}
	return parsed, found, false
}

func repositoryReviewNonnegativeInt(value any) (int, bool) {
	maximum := uint64(repositoryReviewMaximumFiles)
	fromSigned := func(number int64) (int, bool) {
		if number < 0 || uint64(number) > maximum {
			return 0, false
		}
		return int(number), true
	}
	fromUnsigned := func(number uint64) (int, bool) {
		if number > maximum {
			return 0, false
		}
		return int(number), true
	}
	fromFloat := func(number float64) (int, bool) {
		if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 ||
			math.Trunc(number) != number || number > float64(maximum) {
			return 0, false
		}
		converted := int(number)
		return converted, true
	}

	switch number := value.(type) {
	case int:
		if number < 0 || uint64(number) > maximum {
			return 0, false
		}
		return number, true
	case int8:
		return fromSigned(int64(number))
	case int16:
		return fromSigned(int64(number))
	case int32:
		return fromSigned(int64(number))
	case int64:
		return fromSigned(number)
	case uint:
		return fromUnsigned(uint64(number))
	case uint8:
		return fromUnsigned(uint64(number))
	case uint16:
		return fromUnsigned(uint64(number))
	case uint32:
		return fromUnsigned(uint64(number))
	case uint64:
		return fromUnsigned(number)
	case float32:
		return fromFloat(float64(number))
	case float64:
		return fromFloat(number)
	case json.Number:
		parsed, err := number.Int64()
		if err != nil {
			return 0, false
		}
		return fromSigned(parsed)
	default:
		return 0, false
	}
}

func (c *repositoryReviewController) finishAutomationRun(
	id, runID string,
	result *workflows.RunResult,
	runErr error,
	checkpointed bool,
	persistedRun *workflows.Run,
) {
	activeSnapshot, activeFound := c.activeRunSnapshot(id, runID)
	if !activeFound {
		c.removeActive(id, runID)
		return
	}
	store := activeSnapshot.store
	c.mu.Lock()
	active := c.active[id]
	pauseReason := repoaudit.RepositoryReviewPauseReason("")
	pauseDetail := ""
	if active != nil && active.runID == runID {
		pauseReason, pauseDetail = active.pauseReason, active.pauseDetail
	}
	c.mu.Unlock()
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, workflows.ErrRunCanceled) {
		if pauseReason == "" {
			pauseReason = repoaudit.RepositoryReviewPauseServiceRestart
			pauseDetail = "The launcher stopped while this batch was running. Resume continues from durable checkpoints."
		}
	}
	current, currentFound, _ := store.GetAutomation(context.Background(), id)
	outcome := repositoryReviewOutcome{}
	if currentFound {
		// Final record normally clears the run. On timeout, cancellation, or a
		// failed final envelope this releases only unfinished reservations; every
		// child checkpoint already committed remains durable.
		_, _ = store.InterruptRepositoryReviewRun(
			context.Background(), repoaudit.CanonicalRepositoryIdentity(current.Repository), runID,
		)
		outcome = loadRepositoryReviewOutcome(store, current)
		if state, found, resolveErr := store.ResolveRepositoryState(
			current.Repository, current.RunIDs,
		); resolveErr == nil && found {
			if snapshot, snapshotErr := repositoryMappingSnapshot(
				context.Background(), store, activeSnapshot.config, current,
			); snapshotErr == nil {
				campaign := repositoryReviewCurrentFindings(current, state)
				ids := make([]string, 0, len(campaign))
				for _, finding := range campaign {
					if finding.RepositoryFindingID == "" {
						ids = append(ids, finding.ID)
					}
				}
				if len(ids) > 0 {
					_, _ = store.SnapshotMappingJobs(state.Repository, ids, snapshot)
				}
			}
		}
		if pauseReason == "" && current.RequestedPauseReason != "" {
			pauseReason = current.RequestedPauseReason
			pauseDetail = current.RequestedPauseDetail
		}
	}

	updated, err := c.updateLatest(
		context.Background(),
		store,
		id,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			if candidate.ActiveRunID != runID {
				return nil
			}
			previousProgress := candidate.Progress
			fileProgress := false
			candidate.ActiveRunID = ""
			candidate.RequestedPauseReason = ""
			candidate.RequestedPauseDetail = ""
			candidate.Progress.Stage = ""
			if checkpointed {
				candidate.Progress.CompletedBatches++
				candidate.Progress.TotalBatches = max(
					candidate.Progress.TotalBatches,
					candidate.Progress.CompletedBatches,
				)
				applyRepositoryReviewRunProgress(candidate, result, persistedRun)
				applyRepositoryReviewOutcome(candidate, outcome)
				fileProgress = repositoryReviewFileProgressMade(
					previousProgress,
					candidate.Progress,
					persistedRun,
					outcome,
					c.runBatch != nil && persistedRun == nil,
				)
			}
			if runErr == nil && result != nil && result.Status == workflows.RunStatusSucceeded && !checkpointed {
				candidate.Status = repoaudit.RepositoryReviewAutomationFailed
				candidate.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
				candidate.PauseDetail = "The workflow ended without a verified durable repository review checkpoint."
				candidate.Progress.Stage = "failed"
				return nil
			}
			if pauseReason == repoaudit.RepositoryReviewPauseRunFailed {
				candidate.Status = repoaudit.RepositoryReviewAutomationFailed
				candidate.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
				candidate.PauseDetail = repositoryReviewBoundedDetail(pauseDetail)
				candidate.Progress.Stage = "failed"
				return nil
			}

			finalPauseReason, finalPauseDetail, shouldPause := repositoryReviewFinalPause(
				pauseReason, pauseDetail, candidate.Status,
			)
			if shouldPause {
				candidate.Status = repoaudit.RepositoryReviewAutomationPaused
				candidate.PauseReason = finalPauseReason
				candidate.PauseDetail = finalPauseDetail
				candidate.Progress.Stage = "paused"
				return nil
			}
			if runErr != nil || result == nil || result.Status == workflows.RunStatusFailed {
				candidate.Status = repoaudit.RepositoryReviewAutomationFailed
				candidate.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
				candidate.PauseDetail = repositoryReviewRunError(runErr, result)
				candidate.Progress.Stage = "failed"
				return nil //nolint:nilerr // Persist the run failure as durable automation state.
			}
			if candidate.Progress.RemainingFiles <= 0 {
				candidate.Status = repoaudit.RepositoryReviewAutomationCompleted
				candidate.PauseReason = ""
				candidate.PauseDetail = ""
				candidate.Progress.Stage = "complete"
				candidate.CompletedAt = c.clock()
				return nil
			}
			if checkpointed && !fileProgress {
				candidate.Status = repoaudit.RepositoryReviewAutomationPaused
				candidate.PauseReason = repoaudit.RepositoryReviewPauseNoProgress
				candidate.PauseDetail = repositoryReviewBoundedDetail(
					"Automatic continuation stopped after a verified batch resolved zero files. Resume to retry the remaining files.",
				)
				candidate.Progress.Stage = "paused"
				return nil
			}
			if candidate.AutoContinue {
				candidate.Status = repoaudit.RepositoryReviewAutomationIdle
				candidate.PauseReason = ""
				candidate.PauseDetail = ""
				candidate.Progress.Stage = "next batch queued"
				return nil
			}
			candidate.Status = repoaudit.RepositoryReviewAutomationPaused
			candidate.PauseReason = repoaudit.RepositoryReviewPauseManual
			candidate.PauseDetail = "The bounded batch completed. Resume to review the remaining files."
			candidate.Progress.Stage = "paused"
			return nil
		},
	)
	c.removeActive(id, runID)
	if err != nil {
		logger.WarnCF("repository-review", "Failed to finalize repository review automation", map[string]any{
			"automation_id": id, "run_id": runID, "error": err.Error(),
		})
		return
	}
	if updated.Status == repoaudit.RepositoryReviewAutomationIdle && updated.AutoContinue {
		_, startErr := c.startAutomation(context.Background(), id, updated.Version, false, "start")
		if startErr != nil {
			logger.WarnCF("repository-review", "Failed to start next repository review batch", map[string]any{
				"automation_id": id, "error": startErr.Error(),
			})
		}
	}
}

func repositoryReviewFileProgressMade(
	before repoaudit.RepositoryReviewProgress,
	after repoaudit.RepositoryReviewProgress,
	persistedRun *workflows.Run,
	outcome repositoryReviewOutcome,
	allowProjectedCounts bool,
) bool {
	if before.RemainingFiles > 0 && after.RemainingFiles < before.RemainingFiles {
		return true
	}
	reviewed, unsupported, durableCountsFound := repositoryReviewDurableRecordFileCounts(persistedRun)
	if durableCountsFound && (reviewed > 0 || unsupported > 0) {
		return true
	}
	if outcome.found && (outcome.coverageExact || !outcome.coverageAvailable) &&
		(outcome.reviewedFiles > before.ReviewedFiles ||
			outcome.unsupportedFiles > before.UnsupportedFiles) {
		return true
	}
	// runBatch is a controller test seam with no persisted workflow run. Keep
	// its projected counters useful without letting production workflow outputs
	// qualify as durable file progress.
	return allowProjectedCounts && (after.ReviewedFiles > before.ReviewedFiles ||
		after.UnsupportedFiles > before.UnsupportedFiles)
}

func repositoryReviewFinalPause(
	reason repoaudit.RepositoryReviewPauseReason,
	detail string,
	status repoaudit.RepositoryReviewAutomationStatus,
) (repoaudit.RepositoryReviewPauseReason, string, bool) {
	if reason == "" && status != repoaudit.RepositoryReviewAutomationStopping {
		return "", "", false
	}
	if reason == "" {
		return repoaudit.RepositoryReviewPauseManual, "Paused after the current safe checkpoint.", true
	}
	return reason, detail, true
}

func (c *repositoryReviewController) removeActive(id, runID string) {
	c.mu.Lock()
	if active := c.active[id]; active != nil && active.runID == runID {
		delete(c.active, id)
	}
	c.mu.Unlock()
}

func (c *repositoryReviewController) activeRunSnapshot(
	id string,
	runID string,
) (repositoryReviewActiveRun, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	active := c.active[id]
	if active == nil || runID != "" && active.runID != runID {
		return repositoryReviewActiveRun{}, false
	}
	return *active, true
}

func applyRepositoryReviewRunProgress(
	automation *repoaudit.RepositoryReviewAutomation,
	result *workflows.RunResult,
	persistedRun *workflows.Run,
) {
	if automation == nil || result == nil {
		return
	}
	if automation.ScopeSelection == nil {
		rawPlan, planFound := repositoryReviewOutputValue(result.Outputs, "scopePlan", "scope_plan")
		rawSelection, selectionFound := repositoryReviewOutputValue(
			result.Outputs, "scopeSelection", "scope_selection",
		)
		var scopePlan repoaudit.RepositoryReviewScopePlan
		var scopeSelection repoaudit.RepositoryReviewScopeSelection
		if planFound && selectionFound &&
			repositoryReviewDecodeValue(rawPlan, &scopePlan) &&
			repositoryReviewDecodeValue(rawSelection, &scopeSelection) &&
			scopePlan.CommitSHA != "" && scopePlan.Hash != "" {
			automation.ScopePlan = scopePlan
			automation.ScopeSelection = &scopeSelection
		}
	}
	if remaining, ok := repositoryReviewRemainingFiles(result, persistedRun); ok {
		automation.Progress.RemainingFiles = remaining
	}
	reviewed := 0
	if persistedRun == nil {
		reviewed = repositoryReviewInt(result.Outputs["reviewedFiles"])
		if reviewed == 0 {
			reviewed = repositoryReviewInt(result.Outputs["reviewed_files"])
		}
	} else if durableReviewed, _, ok := repositoryReviewDurableRecordFileCounts(persistedRun); ok {
		reviewed = durableReviewed
	}
	if reviewed > 0 {
		automation.Progress.ReviewedFiles += reviewed
	}
	if automation.MaxFilesPerRun > 0 {
		remainingBatches := int(
			math.Ceil(float64(automation.Progress.RemainingFiles) / float64(automation.MaxFilesPerRun)),
		)
		automation.Progress.TotalBatches = max(
			automation.Progress.TotalBatches,
			automation.Progress.CompletedBatches+remainingBatches,
		)
	}
}

type repositoryReviewOutcome struct {
	found                  bool
	coverageAvailable      bool
	coverageExact          bool
	selectedFiles          int
	inspectedFiles         int
	reviewedFiles          int
	remainingFiles         int
	unsupportedFiles       int
	rawFindings            int
	deduplicatedFindings   int
	findings               int
	findingAggregates      int
	pendingFindingMappings int
	modelFindings          map[string]int
	modelPaths             map[string][]string
}

func loadRepositoryReviewOutcome(
	store repoaudit.Store,
	automation repoaudit.RepositoryReviewAutomation,
) repositoryReviewOutcome {
	state, found, err := store.ResolveRepositoryState(automation.Repository, automation.RunIDs)
	if err != nil || !found {
		return repositoryReviewOutcome{}
	}
	return loadRepositoryReviewOutcomeFromResolvedState(state, automation)
}

func loadRepositoryReviewOutcomeFromResolvedState(
	state repoaudit.RepositoryState,
	automation repoaudit.RepositoryReviewAutomation,
) repositoryReviewOutcome {
	if automation.CampaignID != "" {
		metrics := repoaudit.CurrentCampaignMetrics(
			state, automation.CampaignID, automation.RunIDs, automation.StartedAt,
		)
		if metrics.CoverageAvailable || !automation.CampaignRecoveryPending {
			return loadRepositoryReviewCampaignOutcome(state, automation)
		}
	}
	configuredRuns := make(map[string]struct{}, len(automation.RunIDs))
	for _, runID := range automation.RunIDs {
		configuredRuns[runID] = struct{}{}
	}
	campaignRuns := make(map[string]struct{})
	findingIDs := make(map[string]struct{})
	unsupportedPaths := make(map[string]struct{})
	for _, run := range state.Runs {
		if automation.CampaignID != "" {
			if run.CampaignID != automation.CampaignID {
				continue
			}
		} else {
			if _, selected := configuredRuns[run.ID]; !selected ||
				!automation.StartedAt.IsZero() && run.CompletedAt.Before(automation.StartedAt) {
				continue
			}
		}
		campaignRuns[run.ID] = struct{}{}
		for _, findingID := range run.FindingIDs {
			findingIDs[findingID] = struct{}{}
		}
		for _, path := range run.UnsupportedPaths {
			unsupportedPaths[path] = struct{}{}
		}
	}
	for _, finding := range repoaudit.CurrentCampaignFindingsByID(
		state, repositoryReviewSelectionCampaignID(automation),
		automation.RunIDs, automation.StartedAt,
	) {
		findingIDs[finding.ID] = struct{}{}
	}
	if len(campaignRuns) == 0 && automation.CampaignID == "" {
		return repositoryReviewOutcome{}
	}
	reviewedPaths := make(map[string]struct{})
	if automation.CampaignID != "" && state.CurrentCampaign != nil &&
		state.CurrentCampaign.ID == automation.CampaignID {
		for pathValue, coverage := range state.CurrentCampaign.Paths {
			if coverage.Completed {
				reviewedPaths[pathValue] = struct{}{}
			}
			if coverage.Unsupported {
				unsupportedPaths[pathValue] = struct{}{}
			}
		}
	} else {
		for path, file := range state.Files {
			if _, selected := campaignRuns[file.RunID]; selected {
				reviewedPaths[path] = struct{}{}
			}
		}
	}
	selectedContexts := make(map[string]repoaudit.FindingContext)
	for _, findingContext := range state.Contexts {
		_, selectedRun := campaignRuns[findingContext.RunID]
		if automation.CampaignID != "" && findingContext.CampaignID == automation.CampaignID ||
			automation.CampaignID == "" && selectedRun {
			selectedContexts[findingContext.ID] = findingContext
		}
	}
	currentRaw := repoaudit.CurrentCampaignRawFindings(
		state, repositoryReviewSelectionCampaignID(automation),
		automation.RunIDs, automation.StartedAt,
	)
	currentDeduplicated := repositoryReviewCurrentDeduplicatedFindings(automation, state)
	outcome := repositoryReviewOutcome{
		found: true, reviewedFiles: len(reviewedPaths),
		unsupportedFiles: len(unsupportedPaths), rawFindings: len(currentRaw),
		deduplicatedFindings: len(currentDeduplicated), findings: len(currentDeduplicated),
		modelFindings: make(map[string]int), modelPaths: make(map[string][]string),
	}
	aggregates := make(map[string]struct{})
	for _, finding := range currentDeduplicated {
		if finding.RepositoryFindingID == "" {
			outcome.pendingFindingMappings++
			continue
		}
		aggregates[finding.RepositoryFindingID] = struct{}{}
	}
	outcome.findingAggregates = len(aggregates)
	for _, alias := range automation.ReviewerModels {
		modelFindingIDs := make(map[string]struct{})
		files := make(map[string]struct{})
		for _, finding := range state.Findings {
			if _, selected := findingIDs[finding.ID]; !selected {
				continue
			}
			for _, observation := range finding.Observations {
				if contextRecord, selected := selectedContexts[observation.ContextID]; selected &&
					(repositoryReviewObservationMatchesAlias(observation, alias) ||
						repositoryReviewContextMatchesAlias(contextRecord, alias)) {
					modelFindingIDs[finding.ID] = struct{}{}
				}
			}
		}
		for _, findingContext := range selectedContexts {
			if !repositoryReviewContextMatchesAlias(findingContext, alias) {
				continue
			}
			for _, file := range findingContext.Files {
				files[file.Path] = struct{}{}
			}
		}
		outcome.modelFindings[alias] = len(modelFindingIDs)
		for path := range files {
			outcome.modelPaths[alias] = append(outcome.modelPaths[alias], path)
		}
	}
	return outcome
}

func loadRepositoryReviewCampaignOutcome(
	state repoaudit.RepositoryState,
	automation repoaudit.RepositoryReviewAutomation,
) repositoryReviewOutcome {
	metrics := repoaudit.CurrentCampaignMetrics(
		state, repositoryReviewSelectionCampaignID(automation),
		automation.RunIDs, automation.StartedAt,
	)
	findings := repoaudit.CurrentCampaignFindingsByID(
		state, automation.CampaignID, automation.RunIDs, automation.StartedAt,
	)
	currentRaw := repoaudit.CurrentCampaignRawFindings(
		state, repositoryReviewSelectionCampaignID(automation),
		automation.RunIDs, automation.StartedAt,
	)
	currentDeduplicated := repositoryReviewCurrentDeduplicatedFindings(automation, state)
	findingAggregates, pendingFindingMappings := repositoryReviewDeduplicatedAssociationCounts(
		currentDeduplicated,
	)
	outcome := repositoryReviewOutcome{
		found:             metrics.CoverageAvailable || len(findings) > 0,
		coverageAvailable: metrics.CoverageAvailable,
		coverageExact:     metrics.CoverageExact, selectedFiles: metrics.SelectedFiles,
		inspectedFiles: metrics.InspectedFiles, reviewedFiles: metrics.CompletedFiles,
		remainingFiles: metrics.RemainingFiles, unsupportedFiles: metrics.UnsupportedFiles,
		rawFindings: len(currentRaw), deduplicatedFindings: len(currentDeduplicated),
		findings: len(currentDeduplicated), findingAggregates: findingAggregates,
		pendingFindingMappings: pendingFindingMappings,
		modelFindings:          make(map[string]int), modelPaths: make(map[string][]string),
	}
	selectedContexts := make(map[string]repoaudit.FindingContext)
	for _, contextRecord := range state.Contexts {
		if contextRecord.CampaignID == automation.CampaignID {
			selectedContexts[contextRecord.ID] = contextRecord
		}
	}
	for _, alias := range automation.ReviewerModels {
		modelFindingIDs := make(map[string]struct{})
		paths := make(map[string]struct{})
		for _, finding := range findings {
			for _, observation := range finding.Observations {
				if contextRecord, selected := selectedContexts[observation.ContextID]; selected &&
					(repositoryReviewObservationMatchesAlias(observation, alias) ||
						repositoryReviewContextMatchesAlias(contextRecord, alias)) {
					modelFindingIDs[finding.ID] = struct{}{}
				}
			}
		}
		for _, contextRecord := range selectedContexts {
			if !repositoryReviewContextMatchesAlias(contextRecord, alias) {
				continue
			}
			for _, file := range contextRecord.Files {
				paths[file.Path] = struct{}{}
			}
		}
		outcome.modelFindings[alias] = len(modelFindingIDs)
		for pathValue := range paths {
			outcome.modelPaths[alias] = append(outcome.modelPaths[alias], pathValue)
		}
	}
	return outcome
}

func repositoryReviewObservationMatchesAlias(
	observation repoaudit.FindingObservation,
	alias string,
) bool {
	if observation.ModelAlias != "" {
		return observation.ModelAlias == alias
	}
	return observation.Model == alias || observation.Reviewer == alias
}

func repositoryReviewContextMatchesAlias(
	contextRecord repoaudit.FindingContext,
	alias string,
) bool {
	if contextRecord.ModelAlias != "" {
		return contextRecord.ModelAlias == alias
	}
	return contextRecord.Model == alias || contextRecord.Reviewer == alias
}

func applyRepositoryReviewLiveMetrics(
	automation *repoaudit.RepositoryReviewAutomation,
	state repoaudit.RepositoryState,
) {
	if automation == nil {
		return
	}
	defer applyRepositoryReviewCurrentFindingProgress(automation, state)
	if automation.CampaignID != "" {
		metrics := repoaudit.CurrentCampaignMetrics(
			state, automation.CampaignID, automation.RunIDs, automation.StartedAt,
		)
		if metrics.CoverageAvailable {
			applyRepositoryReviewOutcome(
				automation, loadRepositoryReviewCampaignOutcome(state, *automation),
			)
			return
		}
		if !automation.CampaignRecoveryPending {
			automation.Progress.CoverageAvailable = false
			automation.Progress.CoverageExact = false
			automation.Progress.SelectedFiles = 0
			automation.Progress.InspectedFiles = 0
			automation.Progress.RawFindings = 0
			automation.Progress.DeduplicatedFindings = 0
			automation.Progress.Findings = 0
			automation.Progress.FindingAggregates = 0
			automation.Progress.PendingFindingMappings = 0
			return
		}
	}
	automation.Progress.CoverageAvailable = false
	automation.Progress.CoverageExact = false
	currentDeduplicated := repositoryReviewCurrentDeduplicatedFindings(*automation, state)
	automation.Progress.FindingAggregates, automation.Progress.PendingFindingMappings = repositoryReviewDeduplicatedAssociationCounts(
		currentDeduplicated,
	)
}

func applyRepositoryReviewCurrentFindingProgress(
	automation *repoaudit.RepositoryReviewAutomation,
	state repoaudit.RepositoryState,
) {
	if automation == nil {
		return
	}
	rawFindings := repoaudit.CurrentCampaignRawFindings(
		state, repositoryReviewSelectionCampaignID(*automation),
		automation.RunIDs, automation.StartedAt,
	)
	deduplicatedFindings := repositoryReviewCurrentDeduplicatedFindings(*automation, state)
	automation.Progress.RawFindings = len(rawFindings)
	automation.Progress.DeduplicatedFindings = len(deduplicatedFindings)
	automation.Progress.Findings = len(deduplicatedFindings)
}

func repositoryReviewDeduplicatedAssociationCounts(
	findings []repoaudit.DeduplicatedReviewFinding,
) (int, int) {
	aggregates := make(map[string]struct{})
	pending := 0
	for _, finding := range findings {
		if finding.RepositoryFindingID == "" {
			pending++
			continue
		}
		aggregates[finding.RepositoryFindingID] = struct{}{}
	}
	return len(aggregates), pending
}

func repositoryReviewCurrentFindings(
	automation repoaudit.RepositoryReviewAutomation,
	state repoaudit.RepositoryState,
) []repoaudit.Finding {
	if automation.CampaignID != "" {
		metrics := repoaudit.CurrentCampaignMetrics(
			state, automation.CampaignID, automation.RunIDs, automation.StartedAt,
		)
		if metrics.CoverageAvailable || !automation.CampaignRecoveryPending {
			return repoaudit.CurrentCampaignFindingsByID(
				state, automation.CampaignID, automation.RunIDs, automation.StartedAt,
			)
		}
	}
	return repoaudit.CurrentCampaignFindings(state, automation.RunIDs, automation.StartedAt)
}

func applyRepositoryReviewOutcome(
	automation *repoaudit.RepositoryReviewAutomation,
	outcome repositoryReviewOutcome,
) {
	if automation == nil || !outcome.found {
		return
	}
	automation.Progress.CoverageAvailable = outcome.coverageAvailable
	automation.Progress.CoverageExact = outcome.coverageExact
	automation.Progress.SelectedFiles = outcome.selectedFiles
	automation.Progress.InspectedFiles = outcome.inspectedFiles
	if outcome.coverageExact {
		automation.Progress.ReviewedFiles = outcome.reviewedFiles
		automation.Progress.RemainingFiles = outcome.remainingFiles
		automation.Progress.UnsupportedFiles = outcome.unsupportedFiles
	} else if !outcome.coverageAvailable {
		automation.Progress.ReviewedFiles = max(automation.Progress.ReviewedFiles, outcome.reviewedFiles)
		automation.Progress.UnsupportedFiles = max(automation.Progress.UnsupportedFiles, outcome.unsupportedFiles)
	}
	deduplicatedFindings := outcome.deduplicatedFindings
	if deduplicatedFindings == 0 && outcome.findings > 0 {
		deduplicatedFindings = outcome.findings
	}
	automation.Progress.RawFindings = outcome.rawFindings
	automation.Progress.DeduplicatedFindings = deduplicatedFindings
	automation.Progress.Findings = deduplicatedFindings
	automation.Progress.FindingAggregates = outcome.findingAggregates
	automation.Progress.PendingFindingMappings = outcome.pendingFindingMappings
	for _, alias := range automation.ReviewerModels {
		stats := automation.ModelStats[alias]
		stats.Findings = max(stats.Findings, outcome.modelFindings[alias])
		automation.ModelStats[alias] = stats
		addRepositoryReviewModelPaths(automation, alias, outcome.modelPaths[alias])
	}
}

func addRepositoryReviewModelPaths(
	automation *repoaudit.RepositoryReviewAutomation,
	alias string,
	paths []string,
) {
	if automation == nil || alias == "" || len(paths) == 0 {
		return
	}
	const sketchBytes = 8 << 10
	raw := make([]byte, sketchBytes)
	if encoded := automation.ModelCoverageSketches[alias]; encoded != "" {
		if decoded, err := base64.RawStdEncoding.DecodeString(encoded); err == nil && len(decoded) == sketchBytes {
			copy(raw, decoded)
		}
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		digest := sha256.Sum256([]byte(path))
		bucket := int(digest[0])<<8 | int(digest[1])
		raw[bucket/8] |= byte(1 << uint(bucket%8))
	}
	if automation.ModelCoverageSketches == nil {
		automation.ModelCoverageSketches = make(map[string]string)
	}
	automation.ModelCoverageSketches[alias] = base64.RawStdEncoding.EncodeToString(raw)
	setBits := 0
	for _, value := range raw {
		for current := value; current != 0; current &= current - 1 {
			setBits++
		}
	}
	totalBits := float64(sketchBytes * 8)
	zeroBits := totalBits - float64(setBits)
	estimate := 100_000
	if zeroBits > 0 {
		estimate = min(100_000, int(math.Round(-totalBits*math.Log(zeroBits/totalBits))))
	}
	stats := automation.ModelStats[alias]
	stats.ReviewedFiles = max(stats.ReviewedFiles, estimate)
	automation.ModelStats[alias] = stats
}

func repositoryReviewRunError(runErr error, result *workflows.RunResult) string {
	if runErr != nil {
		return repositoryReviewBoundedDetail(runErr.Error())
	}
	if result != nil && strings.TrimSpace(result.Error) != "" {
		return repositoryReviewBoundedDetail(result.Error)
	}
	return "The repository review batch failed."
}

func repositoryReviewBoundedDetail(value string) string {
	const maximumBytes = 4096
	value = strings.TrimSpace(value)
	if len(value) <= maximumBytes {
		return value
	}
	end := maximumBytes - len("...")
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end] + "..."
}

func repositoryReviewInt(value any) int {
	switch number := value.(type) {
	case int:
		return number
	case int64:
		return int(number)
	case float64:
		return int(number)
	case float32:
		return int(number)
	case string:
		var parsed int
		_, _ = fmt.Sscanf(strings.TrimSpace(number), "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

func (c *repositoryReviewController) updateLatest(
	ctx context.Context,
	store repoaudit.Store,
	id string,
	mutate func(*repoaudit.RepositoryReviewAutomation) error,
) (repoaudit.RepositoryReviewAutomation, error) {
	var lastErr error
	for attempt := 0; attempt < 12; attempt++ {
		current, found, err := store.GetAutomation(ctx, id)
		if err != nil {
			return repoaudit.RepositoryReviewAutomation{}, err
		}
		if !found {
			return repoaudit.RepositoryReviewAutomation{}, errors.New("repository review automation not found")
		}
		updated, err := store.UpdateAutomation(ctx, id, current.Version, mutate)
		if !errors.Is(err, repoaudit.ErrConflict) {
			return updated, err
		}
		lastErr = err
	}
	return repoaudit.RepositoryReviewAutomation{}, lastErr
}

func (c *repositoryReviewController) clock() time.Time {
	if c == nil || c.now == nil {
		return time.Now().UTC()
	}
	return c.now().UTC()
}

func (c *repositoryReviewController) repositoryReviewGuardAccountLimits(
	ctx context.Context,
	cfg *config.Config,
	automation repoaudit.RepositoryReviewAutomation,
) ([]repoaudit.RepositoryReviewAccountLimitSnapshot, bool, error) {
	probe := c.probe
	if probe == nil {
		probe = loadCodexAccountLimits
	}
	probeCtx, cancel := context.WithTimeout(ctx, repositoryReviewQuotaProbeTimeout)
	defer cancel()
	response, probeErr := probe(probeCtx)
	now := c.clock()
	refs := repositoryReviewAccountRefsForSelection(cfg, automation.AccountRef)
	if len(refs) == 0 {
		return nil, false, errors.New("selected review account is unavailable")
	}
	byID := make(map[string]codexAccountLimitAccount, len(response.Accounts))
	for _, account := range response.Accounts {
		byID[strings.ToLower(strings.TrimSpace(account.ID))] = account
	}
	snapshots := make([]repoaudit.RepositoryReviewAccountLimitSnapshot, 0)
	complete := probeErr == nil && strings.TrimSpace(response.Error) == ""
	for _, ref := range refs {
		telemetryIDs := repositoryReviewTelemetryIDsForAccountRef(cfg, ref)
		var telemetry codexAccountLimitAccount
		matched := false
		for _, telemetryID := range telemetryIDs {
			if candidate, exists := byID[strings.ToLower(strings.TrimSpace(telemetryID))]; exists {
				telemetry, matched = candidate, true
				break
			}
		}
		if !matched || len(telemetry.Entries) == 0 {
			complete = false
			detail := "account limit telemetry is unavailable"
			if matched {
				detail = firstRepositoryReviewLimitDetail(
					telemetry.LimitsError, telemetry.LimitsStatus, telemetry.CredentialStatus,
				)
			}
			snapshots = append(snapshots, repoaudit.RepositoryReviewAccountLimitSnapshot{
				AccountID: ref, Window: "unknown", CheckedAt: now, Detail: detail,
			})
			continue
		}
		for _, entry := range telemetry.Entries {
			window := normalizeRepositoryReviewWindow(entry.Window)
			snapshot := repoaudit.RepositoryReviewAccountLimitSnapshot{
				AccountID: ref, Name: strings.TrimSpace(entry.Name), Window: window,
				CheckedAt: now, Detail: strings.TrimSpace(entry.Status),
			}
			status := strings.ToLower(strings.TrimSpace(entry.Status))
			exhausted := status == "limit_reached" || status == "exhausted" ||
				status == "blocked" || status == "quota_exhausted"
			if exhausted {
				remaining := 0.0
				snapshot.RemainingPercent = &remaining
			} else if entry.UsedPercent != nil {
				remaining := math.Max(0, math.Min(100, 100-float64(*entry.UsedPercent)))
				snapshot.RemainingPercent = &remaining
			} else {
				complete = false
			}
			if reset, ok := parseRepositoryReviewReset(entry.RefreshesAt); ok {
				snapshot.ResetsAt = reset
			}
			snapshots = append(snapshots, snapshot)
		}
	}
	if probeErr != nil {
		return snapshots, false, probeErr
	}
	if detail := strings.TrimSpace(response.Error); detail != "" {
		return snapshots, false, errors.New(detail)
	}
	return snapshots, complete, nil
}

func repositoryReviewTelemetryIDsForAccountRef(cfg *config.Config, accountRef string) []string {
	accountRef = strings.TrimSpace(accountRef)
	ids := make([]string, 0, 2)
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || slices.Contains(ids, value) {
			return
		}
		ids = append(ids, value)
	}
	if credentialID, ok := config.AccountRouterCredentialAccountID(accountRef); ok {
		add(credentialID)
		return ids
	}
	if cfg != nil {
		if account, err := cfg.GetEnabledModelConfig(accountRef); err == nil && account != nil {
			add(account.CredentialID)
		}
	}
	add(accountRef)
	return ids
}

func firstRepositoryReviewLimitDetail(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "unavailable"
}

func normalizeRepositoryReviewWindow(window string) string {
	window = strings.ToLower(strings.TrimSpace(window))
	switch {
	case window == "":
		return "unknown"
	case strings.Contains(window, "week") || window == "7d":
		return "weekly"
	case strings.Contains(window, "day") || window == "24h":
		return "daily"
	default:
		return window
	}
}

func parseRepositoryReviewReset(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), true
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05 MST", value, time.Local)
	return parsed.UTC(), err == nil
}

func (c *repositoryReviewController) monitor() {
	defer c.wg.Done()
	c.reconcile()
	ticker := time.NewTicker(c.monitorEvery)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.reconcile()
		}
	}
}

func (c *repositoryReviewController) reconcile() {
	store, cfg := c.leasedStore, c.leasedConfig
	if cfg == nil {
		return
	}
	automations, err := store.ListAutomations(c.ctx)
	if err != nil {
		return
	}
	for _, automation := range automations {
		if c.ctx.Err() != nil {
			return
		}
		c.mu.Lock()
		_, active := c.active[automation.ID]
		c.mu.Unlock()
		if (automation.Status == repoaudit.RepositoryReviewAutomationRunning ||
			automation.Status == repoaudit.RepositoryReviewAutomationStopping) && !active {
			_, _ = store.InterruptRepositoryReviewRun(
				context.Background(), repoaudit.CanonicalRepositoryIdentity(automation.Repository),
				automation.ActiveRunID,
			)
			if strings.TrimSpace(cfg.WorkspacePath()) != "" {
				workflowStore := workflows.NewFileRunStore(cfg.WorkspacePath())
				_, _ = workflowStore.CancelRun(context.Background(), automation.ActiveRunID, "launcher restarted")
			}
			requestedReason := automation.RequestedPauseReason
			requestedDetail := automation.RequestedPauseDetail
			if requestedReason == "" {
				requestedReason = repoaudit.RepositoryReviewPauseServiceRestart
				requestedDetail = "The launcher restarted. Resume continues from durable review checkpoints."
			}
			paused, pauseErr := store.UpdateAutomation(
				context.Background(),
				automation.ID,
				automation.Version,
				func(candidate *repoaudit.RepositoryReviewAutomation) error {
					candidate.Status = repoaudit.RepositoryReviewAutomationPaused
					candidate.ActiveRunID = ""
					candidate.PauseReason = requestedReason
					candidate.PauseDetail = requestedDetail
					candidate.RequestedPauseReason = ""
					candidate.RequestedPauseDetail = ""
					candidate.Progress.Stage = "paused"
					return nil
				},
			)
			if pauseErr == nil {
				if commit := repositoryReviewRememberedCommit(paused); commit != "" {
					_, _ = c.ensureRepositoryReviewCampaign(
						context.Background(), store, cfg, paused, commit, "resume",
					)
				}
			}
		}
	}
	c.startHistoricalFindingDeduplication(automations)
	c.startRepositoryFindingDeduplication()
	c.startRepositoryFindingMapping(automations)
	c.startRepositoryFindingValidation(automations)
}
