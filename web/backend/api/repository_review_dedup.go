package api

import (
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
)

var repositoryReviewDeduplicatedFindingCollectionSchema = mustCollectionQuerySchema(
	[]collectionquery.FieldSchema{
		{Name: "id", Type: collectionquery.TypeString, Sortable: true},
		{Name: "repository", Type: collectionquery.TypeString, Sortable: true},
		{Name: "title", Type: collectionquery.TypeString, Sortable: true},
		{Name: "path", Type: collectionquery.TypeString, Sortable: true},
		{Name: "symbol", Type: collectionquery.TypeString, Sortable: true},
		{
			Name: "severity", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"critical", "high", "medium", "low"},
		},
		{
			Name: "status", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"open", "dismissed", "posted"},
		},
		{
			Name: "run_status", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{
				"pending", "processing", "failed", "associated_new", "associated_existing", "needs_review",
			},
		},
		{
			Name: "association", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"unassociated", "new", "existing", "needs_review"},
		},
		{Name: "contributors", Type: collectionquery.TypeString, Sortable: true},
		{Name: "sources", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "mapped", Type: collectionquery.TypeBoolean, Sortable: true},
		{Name: "created", Type: collectionquery.TypeTimestamp, Sortable: true},
		{Name: "updated", Type: collectionquery.TypeTimestamp, Sortable: true},
	},
	[]collectionquery.SortField{
		{Field: "severity", Direction: collectionquery.Descending},
		{Field: "updated", Direction: collectionquery.Descending},
	},
)

var repositoryReviewRawFindingCollectionSchema = mustCollectionQuerySchema(
	[]collectionquery.FieldSchema{
		{Name: "id", Type: collectionquery.TypeString, Sortable: true},
		{Name: "path", Type: collectionquery.TypeString, Sortable: true},
		{
			Name: "severity", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"critical", "high", "medium", "low"},
		},
		{Name: "title", Type: collectionquery.TypeString, Sortable: true},
		{Name: "symbol", Type: collectionquery.TypeString, Sortable: true},
		{Name: "model", Type: collectionquery.TypeString, Sortable: true},
		{Name: "reviewer", Type: collectionquery.TypeString, Sortable: true},
		{
			Name: "deduplication_state", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"pending", "running", "failed", "completed"},
		},
		{
			Name: "disposition", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"undecided", "new", "duplicate"},
		},
		{Name: "finding", Type: collectionquery.TypeString, Sortable: true},
		{Name: "created", Type: collectionquery.TypeTimestamp, Sortable: true},
		{Name: "updated", Type: collectionquery.TypeTimestamp, Sortable: true},
	},
	[]collectionquery.SortField{{Field: "created", Direction: collectionquery.Descending}},
)

// repositoryReviewDeduplicatedFindingSummary keeps diagnosis evidence,
// history, and source identities on their dedicated detail routes.
type repositoryReviewDeduplicatedFindingSummary struct {
	ID                  string                           `json:"id"`
	Repository          string                           `json:"repository"`
	Path                string                           `json:"path"`
	Line                *int                             `json:"line,omitempty"`
	Severity            string                           `json:"severity"`
	Title               string                           `json:"title"`
	Symbol              string                           `json:"symbol,omitempty"`
	Status              repoaudit.FindingStatus          `json:"status"`
	RunFindingStatus    repositoryReviewRunFindingStatus `json:"run_finding_status"`
	Association         string                           `json:"association"`
	RepositoryFindingID string                           `json:"repository_finding_id,omitempty"`
	Contributors        []string                         `json:"contributors"`
	RawSourceCount      int                              `json:"raw_source_count"`
	CreatedAt           time.Time                        `json:"created_at"`
	UpdatedAt           time.Time                        `json:"updated_at"`
}

type repositoryReviewRawFindingSummary struct {
	ID                    string                                 `json:"id"`
	CampaignID            string                                 `json:"campaign_id"`
	Path                  string                                 `json:"path"`
	Line                  *int                                   `json:"line,omitempty"`
	Severity              string                                 `json:"severity"`
	Title                 string                                 `json:"title"`
	Symbol                string                                 `json:"symbol,omitempty"`
	Model                 string                                 `json:"model"`
	ModelAlias            string                                 `json:"model_alias,omitempty"`
	Account               string                                 `json:"account,omitempty"`
	Reviewer              string                                 `json:"reviewer,omitempty"`
	DeduplicationState    repoaudit.RawFindingDeduplicationState `json:"deduplication_state"`
	Disposition           repoaudit.RawFindingDisposition        `json:"disposition"`
	DeduplicatedFindingID string                                 `json:"deduplicated_finding_id,omitempty"`
	Failure               *repoaudit.DeduplicationFailure        `json:"failure,omitempty"`
	CreatedAt             time.Time                              `json:"created_at"`
	UpdatedAt             time.Time                              `json:"updated_at"`
}

func (h *Handler) handleListRepositoryReviewDeduplicatedFindingsCollection(
	w http.ResponseWriter,
	r *http.Request,
) {
	listRequest, ok := parseCollectionListRequest(
		w, r, repositoryReviewDeduplicatedFindingCollectionSchema,
	)
	if !ok {
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	findings := repositoryReviewCurrentDeduplicatedFindings(ledger.Automation, ledger.State)
	rawFindings := repoaudit.CurrentCampaignRawFindings(
		ledger.State, repositoryReviewSelectionCampaignID(ledger.Automation),
		ledger.Automation.RunIDs, ledger.Automation.StartedAt,
	)
	statusIndex := newRepositoryReviewRunFindingStatusIndex(ledger.State)
	rawByID := make(map[string]repoaudit.RawReviewFinding, len(ledger.State.RawFindings))
	for _, raw := range ledger.State.RawFindings {
		rawByID[raw.ID] = raw
	}
	summaries := make([]repositoryReviewDeduplicatedFindingSummary, 0, len(findings))
	for _, finding := range findings {
		summaries = append(summaries, projectRepositoryReviewDeduplicatedFindingSummary(
			finding, statusIndex, rawByID,
		))
	}
	contextID := repositoryReviewCollectionCursorContext(
		"deduplicated-findings", ledger.Automation.ID,
		repositoryReviewCurrentCampaignCursorKey(ledger.Automation),
	)
	page, pageErr := collectionquery.Paginate(
		summaries,
		listRequest.Query,
		listRequest.Cursor,
		listRequest.Limit,
		listRequest.Now,
		repositoryReviewDeduplicatedFindingPageOptions(contextID),
	)
	if pageErr != nil {
		writeCollectionPageError(w, pageErr)
		return
	}
	repositories, titles, paths, symbols, contributors := []string{}, []string{}, []string{}, []string{}, []string{}
	for _, finding := range summaries {
		repositories = append(repositories, finding.Repository)
		titles = append(titles, finding.Title)
		paths = append(paths, finding.Path)
		symbols = append(symbols, finding.Symbol)
		contributors = append(contributors, finding.Contributors...)
	}
	response := map[string]any{
		"automation":      projectRepositoryReviewAutomation(ledger.Automation),
		"findings":        page.Items,
		"total":           page.Total,
		"next_cursor":     page.NextCursor,
		"canonical_query": listRequest.Query.Canonical(),
		"query_schema": collectionSchemaWithSuggestions(
			repositoryReviewDeduplicatedFindingCollectionSchema,
			map[collectionquery.Field][]string{
				"repository": repositories, "title": titles, "path": paths,
				"symbol": symbols, "contributors": contributors,
			},
		),
		"capabilities":             repositoryReviewGlobalCapabilities(ledger),
		"findings_processing":      repositoryReviewFindingsProcessingCounters(rawFindings),
		"historical_deduplication": ledger.State.HistoricalDeduplication,
	}
	if ledger.Found {
		response["repository"] = repoaudit.Summarize(ledger.State)
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleListRepositoryReviewRawFindingsCollection(
	w http.ResponseWriter,
	r *http.Request,
) {
	listRequest, ok := parseCollectionListRequest(w, r, repositoryReviewRawFindingCollectionSchema)
	if !ok {
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	rawFindings := repoaudit.CurrentCampaignRawFindings(
		ledger.State, repositoryReviewSelectionCampaignID(ledger.Automation),
		ledger.Automation.RunIDs, ledger.Automation.StartedAt,
	)
	summaries := make([]repositoryReviewRawFindingSummary, 0, len(rawFindings))
	for _, raw := range rawFindings {
		summaries = append(summaries, projectRepositoryReviewRawFindingSummary(raw))
	}
	contextID := repositoryReviewCollectionCursorContext(
		"raw-findings", ledger.Automation.ID, repositoryReviewCurrentCampaignCursorKey(ledger.Automation),
	)
	page, pageErr := collectionquery.Paginate(
		summaries,
		listRequest.Query,
		listRequest.Cursor,
		listRequest.Limit,
		listRequest.Now,
		repositoryReviewRawFindingPageOptions(contextID),
	)
	if pageErr != nil {
		writeCollectionPageError(w, pageErr)
		return
	}
	titles, paths, symbols, models, reviewers, findings := []string{}, []string{}, []string{}, []string{}, []string{}, []string{}
	for _, raw := range summaries {
		titles = append(titles, raw.Title)
		paths = append(paths, raw.Path)
		symbols = append(symbols, raw.Symbol)
		models = append(models, raw.Model)
		reviewers = append(reviewers, raw.Reviewer)
		findings = append(findings, raw.DeduplicatedFindingID)
	}
	response := map[string]any{
		"automation":      projectRepositoryReviewAutomation(ledger.Automation),
		"raw_findings":    page.Items,
		"total":           page.Total,
		"next_cursor":     page.NextCursor,
		"canonical_query": listRequest.Query.Canonical(),
		"query_schema": collectionSchemaWithSuggestions(
			repositoryReviewRawFindingCollectionSchema,
			map[collectionquery.Field][]string{
				"title": titles, "path": paths, "symbol": symbols,
				"model": models, "reviewer": reviewers, "finding": findings,
			},
		),
		"capabilities":             repositoryReviewGlobalCapabilities(ledger),
		"findings_processing":      repositoryReviewFindingsProcessingCounters(rawFindings),
		"historical_deduplication": ledger.State.HistoricalDeduplication,
	}
	if ledger.Found {
		response["repository"] = repoaudit.Summarize(ledger.State)
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) writeRepositoryReviewDeduplicatedFindingsPage(
	w http.ResponseWriter,
	ledger repositoryReviewAutomationLedger,
	scope string,
	offset, limit int,
) {
	findings := make([]repoaudit.DeduplicatedReviewFinding, 0, len(ledger.State.DeduplicatedFindings))
	if scope == "all" {
		for _, finding := range ledger.State.DeduplicatedFindings {
			if strings.HasPrefix(finding.ID, "rdf_") {
				findings = append(findings, finding)
			}
		}
	} else {
		findings = repositoryReviewCurrentDeduplicatedFindings(ledger.Automation, ledger.State)
	}
	rawFindings := repoaudit.CurrentCampaignRawFindings(
		ledger.State, repositoryReviewSelectionCampaignID(ledger.Automation),
		ledger.Automation.RunIDs, ledger.Automation.StartedAt,
	)
	total := len(findings)
	offset = min(offset, total)
	end := min(total, offset+limit)
	page := make([]repositoryReviewRunFindingProjection, 0, end-offset)
	for _, finding := range findings[offset:end] {
		if projection, found := repositoryReviewFindingByID(ledger.State, finding.ID); found {
			projection.Observations = nil
			page = append(page, projectRepositoryReviewRunFinding(ledger.State, projection))
		}
	}
	response := map[string]any{
		"automation":               projectRepositoryReviewAutomation(ledger.Automation),
		"repository":               repoaudit.Summarize(ledger.State),
		"findings":                 page,
		"repository_findings":      []repoaudit.RepositoryFinding{},
		"scope":                    scope,
		"offset":                   offset,
		"total":                    total,
		"capabilities":             repositoryReviewGlobalCapabilities(ledger),
		"findings_processing":      repositoryReviewFindingsProcessingCounters(rawFindings),
		"historical_deduplication": ledger.State.HistoricalDeduplication,
	}
	if end < total {
		response["next_offset"] = end
	}
	repositoryOffset := offset
	if repositoryTotal := len(ledger.State.RepositoryFindings); repositoryTotal == 0 {
		repositoryOffset = 0
	} else if repositoryOffset >= repositoryTotal {
		repositoryOffset = ((repositoryTotal - 1) / limit) * limit
	}
	repositoryEnd := min(len(ledger.State.RepositoryFindings), repositoryOffset+limit)
	repositoryPage := make([]repoaudit.RepositoryFinding, 0, repositoryEnd-repositoryOffset)
	for _, finding := range ledger.State.RepositoryFindings[repositoryOffset:repositoryEnd] {
		repositoryPage = append(repositoryPage, repositoryReviewRepositoryFindingSummary(finding))
	}
	response["repository_findings"] = repositoryPage
	response["repository_finding_total"] = len(ledger.State.RepositoryFindings)
	response["repository_finding_offset"] = repositoryOffset
	if repositoryEnd < len(ledger.State.RepositoryFindings) {
		response["next_repository_finding_offset"] = repositoryEnd
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleGetRepositoryReviewDeduplicatedFinding(
	w http.ResponseWriter,
	r *http.Request,
) {
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	findingID := strings.TrimSpace(r.PathValue("finding_id"))
	var finding repoaudit.DeduplicatedReviewFinding
	found := false
	for _, candidate := range repositoryReviewCurrentDeduplicatedFindings(
		ledger.Automation, ledger.State,
	) {
		if candidate.ID == findingID {
			finding, found = candidate, true
			break
		}
	}
	if !found {
		writeRepositoryReviewAutomationError(w, os.ErrNotExist)
		return
	}
	capabilities := repositoryReviewGlobalCapabilities(ledger)
	contexts := []repoaudit.FindingContext{}
	if projection, projectionFound := repositoryReviewFindingByID(ledger.State, finding.ID); projectionFound {
		capabilities = repositoryReviewFindingCapabilities(ledger.State, projection)
	}
	if len(finding.RawSourceIDs) > 0 {
		if raw, rawFound := repositoryReviewRawFindingByID(
			ledger.State, finding.RawSourceIDs[0],
		); rawFound {
			if contextRecord, contextFound := repositoryReviewContextByID(
				ledger.State, raw.ContextID,
			); contextFound {
				contextRecord.CampaignID = ""
				contextRecord.RawDigest = ""
				contexts = append(contexts, contextRecord)
			}
		}
	}
	response := map[string]any{
		"automation":       projectRepositoryReviewAutomation(ledger.Automation),
		"repository":       repoaudit.Summarize(ledger.State),
		"finding":          finding,
		"raw_source_total": len(finding.RawSourceIDs),
		"contexts":         contexts,
		"capabilities":     capabilities,
	}
	if projection, projectionFound := repositoryReviewFindingByID(ledger.State, finding.ID); projectionFound {
		response["finding"] = projectRepositoryReviewRunFinding(ledger.State, projection)
	}
	if finding.RepositoryFindingID != "" {
		if repositoryFinding, exists := repositoryReviewRepositoryFindingByID(
			ledger.State, finding.RepositoryFindingID,
		); exists {
			response["repository_finding"] = repositoryFinding
		}
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleListRepositoryReviewRawSources(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := repositoryReviewRawPage(r)
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	requestedFindingID := strings.TrimSpace(r.PathValue("finding_id"))
	var finding repoaudit.DeduplicatedReviewFinding
	found := false
	for _, candidate := range repositoryReviewCurrentDeduplicatedFindings(
		ledger.Automation, ledger.State,
	) {
		if candidate.ID == requestedFindingID {
			finding, found = candidate, true
			break
		}
	}
	if !found {
		writeRepositoryReviewAutomationError(w, os.ErrNotExist)
		return
	}
	byID := make(map[string]repoaudit.RawReviewFinding, len(ledger.State.RawFindings))
	for _, raw := range ledger.State.RawFindings {
		byID[raw.ID] = raw
	}
	sources := make([]repositoryReviewRawFindingSummary, 0, len(finding.RawSourceIDs))
	for _, sourceID := range finding.RawSourceIDs {
		if raw, exists := byID[sourceID]; exists {
			sources = append(sources, projectRepositoryReviewRawFindingSummary(raw))
		}
	}
	total := len(sources)
	offset = min(offset, total)
	end := min(total, offset+limit)
	response := map[string]any{
		"automation": projectRepositoryReviewAutomation(ledger.Automation),
		"repository": repoaudit.Summarize(ledger.State),
		"finding_id": finding.ID,
		"sources":    append([]repositoryReviewRawFindingSummary(nil), sources[offset:end]...),
		"offset":     offset,
		"total":      total,
	}
	if end < total {
		response["next_offset"] = end
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleGetRepositoryReviewRawSource(w http.ResponseWriter, r *http.Request) {
	ledger, raw, ok := h.repositoryReviewRawSource(w, r)
	if !ok {
		return
	}
	response := map[string]any{
		"automation":               projectRepositoryReviewAutomation(ledger.Automation),
		"repository":               repoaudit.Summarize(ledger.State),
		"source":                   projectRepositoryReviewRawFindingDetail(raw),
		"historical_deduplication": ledger.State.HistoricalDeduplication,
	}
	if contextRecord, found := repositoryReviewContextByID(ledger.State, raw.ContextID); found {
		response["context"] = contextRecord
	}
	if strings.HasPrefix(raw.DeduplicatedFindingID, "rdf_") {
		if finding, found := repositoryReviewDeduplicatedFindingByID(
			ledger.State, raw.DeduplicatedFindingID,
		); found {
			if projection, projectionFound := repositoryReviewFindingByID(
				ledger.State, finding.ID,
			); projectionFound {
				response["finding"] = projectRepositoryReviewRunFinding(ledger.State, projection)
			} else {
				response["finding"] = finding
			}
		}
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleRetryRepositoryReviewRawSource(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil || r.URL == nil ||
		r.URL.RawQuery != "" {
		writeRepositoryReviewError(w, errors.New("invalid raw finding retry request"))
		return
	}
	var request struct{}
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	ledger, raw, ok := h.repositoryReviewRawSource(w, r)
	if !ok {
		return
	}
	persistedRaw, persisted := repositoryReviewRawFindingByID(ledger.State, raw.ID)
	if !persisted || repoaudit.HistoricalDeduplicationRawFinding(persistedRaw) {
		writeRepositoryReviewError(w, errors.Join(
			repoaudit.ErrConflict,
			errors.New("historical raw findings require retrying the whole historical deduplication replay"),
		))
		return
	}
	state, retried, err := ledger.Store.RetryDeduplication(ledger.State.Repository, raw.ID)
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	if controller := h.repositoryReviewControllerInstance(); controller != nil {
		controller.wakeRepositoryFindingDeduplication()
	}
	currentRaw := repoaudit.CurrentCampaignRawFindings(
		state, repositoryReviewSelectionCampaignID(ledger.Automation),
		ledger.Automation.RunIDs, ledger.Automation.StartedAt,
	)
	writeRepositoryReviewJSON(w, http.StatusAccepted, map[string]any{
		"automation":          projectRepositoryReviewAutomation(ledger.Automation),
		"repository":          repoaudit.Summarize(state),
		"source":              projectRepositoryReviewRawFindingDetail(retried),
		"findings_processing": repositoryReviewFindingsProcessingCounters(currentRaw),
	})
}

func (h *Handler) handleGetRepositoryReviewFindingsProcessing(
	w http.ResponseWriter,
	r *http.Request,
) {
	offset, limit, stateFilter, err := repositoryReviewFindingsProcessingPage(r)
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	campaignID := strings.TrimSpace(r.PathValue("campaign_id"))
	if campaignID != "" && (len(campaignID) > 256 || strings.ContainsRune(campaignID, 0)) {
		writeRepositoryReviewError(w, errors.New("a valid campaign is required"))
		return
	}
	campaignRaw := make([]repoaudit.RawReviewFinding, 0)
	if campaignID == "" {
		campaignRaw = repoaudit.CurrentCampaignRawFindings(
			ledger.State, repositoryReviewSelectionCampaignID(ledger.Automation),
			ledger.Automation.RunIDs, ledger.Automation.StartedAt,
		)
	} else {
		for _, raw := range ledger.State.RawFindings {
			if raw.CampaignID == campaignID {
				campaignRaw = append(campaignRaw, raw)
			}
		}
	}
	rawFindings := make([]repoaudit.RawReviewFinding, 0)
	for _, raw := range campaignRaw {
		if stateFilter == "" || string(raw.State) == stateFilter {
			rawFindings = append(rawFindings, raw)
		}
	}
	sort.SliceStable(rawFindings, func(left, right int) bool {
		if rawFindings[left].CreatedAt.Equal(rawFindings[right].CreatedAt) {
			return rawFindings[left].ID < rawFindings[right].ID
		}
		return rawFindings[left].CreatedAt.Before(rawFindings[right].CreatedAt)
	})
	summaries := make([]repositoryReviewRawFindingSummary, 0, len(rawFindings))
	for _, raw := range rawFindings {
		summaries = append(summaries, projectRepositoryReviewRawFindingSummary(raw))
	}
	total := len(summaries)
	offset = min(offset, total)
	end := min(total, offset+limit)
	response := map[string]any{
		"automation":          projectRepositoryReviewAutomation(ledger.Automation),
		"repository":          repoaudit.Summarize(ledger.State),
		"campaign_id":         campaignID,
		"findings_processing": repositoryReviewFindingsProcessingCounters(campaignRaw),
		"raw_findings": append(
			[]repositoryReviewRawFindingSummary(nil), summaries[offset:end]...,
		),
		"offset": offset,
		"total":  total,
	}
	if end < total {
		response["next_offset"] = end
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) repositoryReviewRawSource(
	w http.ResponseWriter,
	r *http.Request,
) (repositoryReviewAutomationLedger, repoaudit.RawReviewFinding, bool) {
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return repositoryReviewAutomationLedger{}, repoaudit.RawReviewFinding{}, false
	}
	requestedID := strings.TrimSpace(r.PathValue("source_id"))
	campaignID := strings.TrimSpace(r.PathValue("campaign_id"))
	var candidates []repoaudit.RawReviewFinding
	if campaignID != "" {
		for _, raw := range ledger.State.RawFindings {
			if raw.CampaignID == campaignID {
				candidates = append(candidates, raw)
			}
		}
	} else {
		candidates = repoaudit.CurrentCampaignRawFindings(
			ledger.State, repositoryReviewSelectionCampaignID(ledger.Automation),
			ledger.Automation.RunIDs, ledger.Automation.StartedAt,
		)
	}
	raw, found := repositoryReviewRawFindingByAlias(candidates, requestedID)
	if !found {
		writeRepositoryReviewAutomationError(w, os.ErrNotExist)
		return repositoryReviewAutomationLedger{}, repoaudit.RawReviewFinding{}, false
	}
	if findingID := strings.TrimSpace(r.PathValue("finding_id")); findingID != "" {
		var finding repoaudit.DeduplicatedReviewFinding
		exists := false
		for _, candidate := range repositoryReviewCurrentDeduplicatedFindings(
			ledger.Automation, ledger.State,
		) {
			if candidate.ID == findingID {
				finding, exists = candidate, true
				break
			}
		}
		if !exists || !containsRepositoryReviewSourceID(finding.RawSourceIDs, raw.ID) {
			writeRepositoryReviewAutomationError(w, os.ErrNotExist)
			return repositoryReviewAutomationLedger{}, repoaudit.RawReviewFinding{}, false
		}
	}
	return ledger, raw, true
}

func repositoryReviewCurrentDeduplicatedFindings(
	automation repoaudit.RepositoryReviewAutomation,
	state repoaudit.RepositoryState,
) []repoaudit.DeduplicatedReviewFinding {
	selected := repoaudit.CurrentCampaignDeduplicatedFindings(
		state, repositoryReviewSelectionCampaignID(automation),
		automation.RunIDs, automation.StartedAt,
	)
	result := make([]repoaudit.DeduplicatedReviewFinding, 0, len(selected))
	for _, finding := range selected {
		if strings.HasPrefix(finding.ID, "rdf_") {
			result = append(result, finding)
		}
	}
	return result
}

func repositoryReviewSelectionCampaignID(
	automation repoaudit.RepositoryReviewAutomation,
) string {
	if automation.CampaignRecoveryPending {
		return ""
	}
	return automation.CampaignID
}

func repositoryReviewCurrentCampaignCursorKey(
	automation repoaudit.RepositoryReviewAutomation,
) string {
	if !automation.StartedAt.IsZero() {
		return automation.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if automation.CampaignID != "" {
		return automation.CampaignID
	}
	return "current"
}

func repositoryReviewDeduplicatedFindingByID(
	state repoaudit.RepositoryState,
	id string,
) (repoaudit.DeduplicatedReviewFinding, bool) {
	id = strings.TrimSpace(id)
	for _, finding := range state.DeduplicatedFindings {
		if finding.ID == id {
			return finding, true
		}
	}
	return repoaudit.DeduplicatedReviewFinding{}, false
}

func repositoryReviewRawFindingByID(
	state repoaudit.RepositoryState,
	id string,
) (repoaudit.RawReviewFinding, bool) {
	id = strings.TrimSpace(id)
	for _, finding := range state.RawFindings {
		if finding.ID == id {
			return finding, true
		}
	}
	return repoaudit.RawReviewFinding{}, false
}

func repositoryReviewRawFindingByAlias(
	findings []repoaudit.RawReviewFinding,
	id string,
) (repoaudit.RawReviewFinding, bool) {
	id = strings.TrimSpace(id)
	for _, finding := range findings {
		if finding.ID == id {
			return finding, true
		}
	}
	if !strings.HasPrefix(id, "rfn_") {
		return repoaudit.RawReviewFinding{}, false
	}
	var selected repoaudit.RawReviewFinding
	found := false
	for _, finding := range findings {
		if finding.LegacyFindingID != id && finding.DeduplicatedFindingID != id {
			continue
		}
		if !found || repositoryReviewRawFindingBefore(finding, selected) {
			selected, found = finding, true
		}
	}
	return selected, found
}

func repositoryReviewRawFindingBefore(
	left, right repoaudit.RawReviewFinding,
) bool {
	if left.InsertionOrdinal != right.InsertionOrdinal {
		return left.InsertionOrdinal < right.InsertionOrdinal
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.Before(right.CreatedAt)
	}
	return left.ID < right.ID
}

func repositoryReviewContextByID(
	state repoaudit.RepositoryState,
	id string,
) (repoaudit.FindingContext, bool) {
	for _, contextRecord := range state.Contexts {
		if contextRecord.ID == id {
			return contextRecord, true
		}
	}
	return repoaudit.FindingContext{}, false
}

func projectRepositoryReviewDeduplicatedFindingSummary(
	finding repoaudit.DeduplicatedReviewFinding,
	statusIndex repositoryReviewRunFindingStatusIndex,
	rawByID map[string]repoaudit.RawReviewFinding,
) repositoryReviewDeduplicatedFindingSummary {
	projection := repoaudit.Finding{
		ID: finding.ID, RepositoryFindingID: finding.RepositoryFindingID,
		RepositoryMatchState: finding.RepositoryMatchState,
	}
	runStatus := statusIndex.status(projection)
	contributors := make([]string, 0)
	for _, rawID := range finding.RawSourceIDs {
		raw, found := rawByID[rawID]
		if !found {
			continue
		}
		model := raw.ModelAlias
		if model == "" {
			model = raw.Model
		}
		contributors = appendUniqueRepositoryReviewContributor(contributors, model)
		contributors = appendUniqueRepositoryReviewContributor(contributors, raw.Reviewer)
	}
	return repositoryReviewDeduplicatedFindingSummary{
		ID: finding.ID, Repository: finding.Repository, Path: finding.File.Path,
		Line: finding.Line, Severity: finding.Severity, Title: finding.Title,
		Symbol: finding.Symbol, Status: finding.Status,
		RunFindingStatus:    runStatus,
		Association:         repositoryReviewRunFindingAssociation(runStatus),
		RepositoryFindingID: finding.RepositoryFindingID,
		Contributors:        contributors,
		RawSourceCount:      len(finding.RawSourceIDs),
		CreatedAt:           finding.CreatedAt, UpdatedAt: finding.UpdatedAt,
	}
}

func appendUniqueRepositoryReviewContributor(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func projectRepositoryReviewRawFindingSummary(
	raw repoaudit.RawReviewFinding,
) repositoryReviewRawFindingSummary {
	parentID := raw.DeduplicatedFindingID
	if !strings.HasPrefix(parentID, "rdf_") {
		parentID = ""
	}
	return repositoryReviewRawFindingSummary{
		ID: raw.ID, CampaignID: raw.CampaignID, Path: raw.File.Path, Line: raw.Line,
		Severity: raw.Severity, Title: raw.Title, Symbol: raw.Symbol,
		Model: raw.Model, ModelAlias: raw.ModelAlias, Account: raw.Account,
		Reviewer: raw.Reviewer, DeduplicationState: raw.State,
		Disposition: raw.Disposition, DeduplicatedFindingID: parentID,
		Failure: raw.Failure, CreatedAt: raw.CreatedAt, UpdatedAt: raw.UpdatedAt,
	}
}

func projectRepositoryReviewRawFindingDetail(
	raw repoaudit.RawReviewFinding,
) repoaudit.RawReviewFinding {
	if !strings.HasPrefix(raw.DeduplicatedFindingID, "rdf_") {
		raw.DeduplicatedFindingID = ""
	}
	return raw
}

func repositoryReviewRawFindingPageOptions(
	contextID string,
) collectionquery.PageOptions[repositoryReviewRawFindingSummary] {
	return collectionquery.PageOptions[repositoryReviewRawFindingSummary]{
		ID: func(finding repositoryReviewRawFindingSummary) (string, error) {
			return repositoryReviewCollectionCursorItemID(contextID, finding.ID)
		},
		ValidateID: repositoryReviewCollectionCursorIDValidator(contextID),
		Resolve: func(
			finding repositoryReviewRawFindingSummary,
			field collectionquery.Field,
			_ time.Time,
		) (collectionquery.FieldValue, bool) {
			switch field {
			case "id":
				return collectionquery.StringValue(finding.ID), true
			case "path":
				return collectionquery.StringValue(finding.Path), true
			case "severity":
				return collectionquery.EnumValue(finding.Severity), true
			case "title":
				return collectionquery.StringValue(finding.Title), true
			case "symbol":
				return collectionquery.StringValue(finding.Symbol), true
			case "model":
				return collectionquery.StringValue(finding.Model), true
			case "reviewer":
				return collectionquery.StringValue(finding.Reviewer), true
			case "deduplication_state":
				return collectionquery.EnumValue(string(finding.DeduplicationState)), true
			case "disposition":
				return collectionquery.EnumValue(string(finding.Disposition)), true
			case "finding":
				return collectionquery.StringValue(finding.DeduplicatedFindingID), true
			case "created":
				return collectionquery.TimestampValue(finding.CreatedAt), true
			case "updated":
				return collectionquery.TimestampValue(finding.UpdatedAt), true
			default:
				return collectionquery.FieldValue{}, false
			}
		},
		Compare: repositoryReviewSeverityComparator,
	}
}

func repositoryReviewDeduplicatedFindingPageOptions(
	contextID string,
) collectionquery.PageOptions[repositoryReviewDeduplicatedFindingSummary] {
	return collectionquery.PageOptions[repositoryReviewDeduplicatedFindingSummary]{
		ID: func(finding repositoryReviewDeduplicatedFindingSummary) (string, error) {
			return repositoryReviewCollectionCursorItemID(contextID, finding.ID)
		},
		ValidateID: repositoryReviewCollectionCursorIDValidator(contextID),
		Resolve: func(
			finding repositoryReviewDeduplicatedFindingSummary,
			field collectionquery.Field,
			_ time.Time,
		) (collectionquery.FieldValue, bool) {
			switch field {
			case "id":
				return collectionquery.StringValue(finding.ID), true
			case "repository":
				return collectionquery.StringValue(finding.Repository), true
			case "title":
				return collectionquery.StringValue(finding.Title), true
			case "path":
				return collectionquery.StringValue(finding.Path), true
			case "symbol":
				return collectionquery.StringValue(finding.Symbol), true
			case "severity":
				return collectionquery.EnumValue(finding.Severity), true
			case "status":
				return collectionquery.EnumValue(string(finding.Status)), true
			case "run_status":
				return collectionquery.EnumValue(string(finding.RunFindingStatus)), true
			case "association":
				return collectionquery.EnumValue(finding.Association), true
			case "contributors":
				return collectionquery.StringValue(strings.Join(finding.Contributors, " ")), true
			case "sources":
				return collectionquery.NumberValue(float64(finding.RawSourceCount)), true
			case "mapped":
				return collectionquery.BooleanValue(finding.RepositoryFindingID != ""), true
			case "created":
				return collectionquery.TimestampValue(finding.CreatedAt), true
			case "updated":
				return collectionquery.TimestampValue(finding.UpdatedAt), true
			default:
				return collectionquery.FieldValue{}, false
			}
		},
		Compare: repositoryReviewSeverityComparator,
	}
}

func repositoryReviewRawPage(r *http.Request) (int, int, error) {
	if r == nil || r.URL == nil {
		return 0, 0, errors.New("invalid raw finding source request")
	}
	query := r.URL.Query()
	for key, values := range query {
		if (key != "offset" && key != "limit") || len(values) != 1 {
			return 0, 0, errors.New("invalid raw finding source request")
		}
	}
	offset, err := repositoryReviewPageInteger(query.Get("offset"), 0, 0)
	if err != nil {
		return 0, 0, err
	}
	limit, err := repositoryReviewPageInteger(query.Get("limit"), 50, 200)
	return offset, limit, err
}

func repositoryReviewFindingsProcessingPage(
	r *http.Request,
) (int, int, string, error) {
	if r == nil || r.URL == nil {
		return 0, 0, "", errors.New("invalid findings processing request")
	}
	query := r.URL.Query()
	for key, values := range query {
		if (key != "offset" && key != "limit" && key != "state") || len(values) != 1 {
			return 0, 0, "", errors.New("invalid findings processing request")
		}
	}
	offset, err := repositoryReviewPageInteger(query.Get("offset"), 0, 0)
	if err != nil {
		return 0, 0, "", err
	}
	limit, err := repositoryReviewPageInteger(query.Get("limit"), 50, 200)
	if err != nil {
		return 0, 0, "", err
	}
	state := strings.TrimSpace(query.Get("state"))
	if state != "" && state != string(repoaudit.RawFindingDeduplicationPending) &&
		state != string(repoaudit.RawFindingDeduplicationRunning) &&
		state != string(repoaudit.RawFindingDeduplicationFailed) &&
		state != string(repoaudit.RawFindingDeduplicationCompleted) {
		return 0, 0, "", errors.New("invalid findings processing state")
	}
	return offset, limit, state, nil
}

func repositoryReviewFindingsProcessingCounters(
	findings []repoaudit.RawReviewFinding,
) repoaudit.FindingsProcessingCounters {
	result := repoaudit.FindingsProcessingCounters{RawTotal: len(findings)}
	for _, finding := range findings {
		if finding.UpdatedAt.After(result.UpdatedAt) {
			result.UpdatedAt = finding.UpdatedAt
		}
		switch finding.State {
		case repoaudit.RawFindingDeduplicationPending:
			result.Pending++
		case repoaudit.RawFindingDeduplicationRunning:
			result.Processing++
		case repoaudit.RawFindingDeduplicationFailed:
			result.Failed++
		case repoaudit.RawFindingDeduplicationCompleted:
			result.Completed++
		}
		switch finding.Disposition {
		case repoaudit.RawFindingDispositionNew:
			result.New++
		case repoaudit.RawFindingDispositionDuplicate:
			result.Duplicates++
		}
	}
	return result
}

func containsRepositoryReviewSourceID(ids []string, wanted string) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}
