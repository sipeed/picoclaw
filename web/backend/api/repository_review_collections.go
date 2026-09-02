package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
)

var repositoryReviewRunFindingCollectionSchema = mustCollectionQuerySchema(
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
		{Name: "created", Type: collectionquery.TypeTimestamp, Sortable: true},
		{Name: "updated", Type: collectionquery.TypeTimestamp, Sortable: true},
	},
	[]collectionquery.SortField{
		{Field: "severity", Direction: collectionquery.Descending},
		{Field: "updated", Direction: collectionquery.Descending},
	},
)

var repositoryReviewRepositoryFindingCollectionSchema = mustCollectionQuerySchema(
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
			Name: "match", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"new", "known", "provisional"},
		},
		{
			Name: "lifecycle", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"open", "resolution_pending", "resolved", "regressed", "dismissed"},
		},
		{
			Name: "issue", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"none", "draft", "open", "closed", "unknown"},
		},
		{
			Name: "validation", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{
				"not_requested", "pending", "running", "confirmed", "not_fixed", "inconclusive", "failed",
			},
		},
		{Name: "occurrences", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "commits", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "created", Type: collectionquery.TypeTimestamp, Sortable: true},
		{Name: "updated", Type: collectionquery.TypeTimestamp, Sortable: true},
	},
	[]collectionquery.SortField{
		{Field: "severity", Direction: collectionquery.Descending},
		{Field: "updated", Direction: collectionquery.Descending},
	},
)

var repositoryReviewIssueCollectionSchema = mustCollectionQuerySchema(
	[]collectionquery.FieldSchema{
		{Name: "id", Type: collectionquery.TypeString, Sortable: true},
		{Name: "repository", Type: collectionquery.TypeString, Sortable: true},
		{Name: "title", Type: collectionquery.TypeString, Sortable: true},
		{Name: "generation", Type: collectionquery.TypeString, Sortable: true},
		{
			Name: "state", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"generating", "failed", "editing", "publishing", "posted", "unknown"},
		},
		{
			Name: "origin", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"ai_generated", "linked", "discovered", "legacy"},
		},
		{Name: "canonical", Type: collectionquery.TypeBoolean, Sortable: true},
		{Name: "publishable", Type: collectionquery.TypeBoolean, Sortable: true},
		{Name: "findings", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "created", Type: collectionquery.TypeTimestamp, Sortable: true},
		{Name: "updated", Type: collectionquery.TypeTimestamp, Sortable: true},
	},
	[]collectionquery.SortField{{Field: "updated", Direction: collectionquery.Descending}},
)

// repositoryReviewRunFindingSummary intentionally omits observations, evidence,
// validation details, match hints, and fix estimates. Those remain available on
// the finding detail route.
type repositoryReviewRunFindingSummary struct {
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
	CreatedAt           time.Time                        `json:"created_at"`
	UpdatedAt           time.Time                        `json:"updated_at"`
}

type repositoryReviewRepositoryFindingIssueSummary struct {
	URL        string                                `json:"url,omitempty"`
	State      repoaudit.RepositoryFindingIssueState `json:"state"`
	SnapshotAt time.Time                             `json:"snapshot_at,omitempty"`
	Conflict   bool                                  `json:"conflict,omitempty"`
}

// repositoryReviewRepositoryFindingCollectionSummary carries only the latest
// location and aggregate counts. Occurrence IDs, commit lists, duplicate
// evidence, match hints, fix estimates, and resolution history are detail-only.
type repositoryReviewRepositoryFindingCollectionSummary struct {
	ID                string                                        `json:"id"`
	Repository        string                                        `json:"repository"`
	CanonicalTitle    string                                        `json:"canonical_title"`
	CanonicalSeverity string                                        `json:"canonical_severity"`
	Path              string                                        `json:"path,omitempty"`
	Symbol            string                                        `json:"symbol,omitempty"`
	MatchState        repoaudit.RepositoryMatchState                `json:"match_state"`
	Lifecycle         repoaudit.RepositoryFindingLifecycle          `json:"lifecycle"`
	Issue             repositoryReviewRepositoryFindingIssueSummary `json:"issue"`
	ValidationState   repoaudit.RepositoryFindingValidationState    `json:"validation_state"`
	OccurrenceCount   int                                           `json:"occurrence_count"`
	FoundCommitCount  int                                           `json:"found_commit_count"`
	CreatedAt         time.Time                                     `json:"created_at"`
	UpdatedAt         time.Time                                     `json:"updated_at"`
}

// repositoryReviewIssueCollectionSummary keeps issue bodies and generation
// instructions on the issue detail route.
type repositoryReviewIssueCollectionSummary struct {
	ID              string                              `json:"id"`
	Repository      string                              `json:"repository"`
	FindingCount    int                                 `json:"finding_count"`
	Origin          repoaudit.IssueDraftOrigin          `json:"origin"`
	GenerationID    string                              `json:"generation_id,omitempty"`
	Canonical       bool                                `json:"canonical"`
	Publishable     bool                                `json:"publishable"`
	PublishBlockers []repoaudit.IssuePublicationBlocker `json:"publish_blockers"`
	Title           string                              `json:"title"`
	State           repoaudit.IssueDraftState           `json:"state"`
	Version         int64                               `json:"version"`
	CreatedAt       time.Time                           `json:"created_at"`
	UpdatedAt       time.Time                           `json:"updated_at"`
}

func (h *Handler) handleListRepositoryReviewRunFindingsCollection(w http.ResponseWriter, r *http.Request) {
	listRequest, ok := parseCollectionListRequest(w, r, repositoryReviewRunFindingCollectionSchema)
	if !ok {
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	findings := []repoaudit.Finding{}
	if ledger.Found {
		findings = repositoryReviewReportFindings(ledger.Automation, ledger.State, "current")
	}
	statusIndex := newRepositoryReviewRunFindingStatusIndex(ledger.State)
	summaries := make([]repositoryReviewRunFindingSummary, 0, len(findings))
	for _, finding := range findings {
		if !strings.HasPrefix(finding.ID, "rfn_") {
			continue
		}
		summaries = append(summaries, projectRepositoryReviewRunFindingSummary(finding, statusIndex.status(finding)))
	}
	contextID := repositoryReviewCollectionCursorContext("run-findings", ledger.Automation.ID, "current")
	page, pageErr := collectionquery.Paginate(
		summaries,
		listRequest.Query,
		listRequest.Cursor,
		listRequest.Limit,
		listRequest.Now,
		repositoryReviewRunFindingPageOptions(contextID),
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
			repositoryReviewRunFindingCollectionSchema,
			map[collectionquery.Field][]string{
				"repository": repositories, "title": titles, "path": paths,
				"symbol": symbols, "contributors": contributors,
			},
		),
		"capabilities": repositoryReviewGlobalCapabilities(ledger),
	}
	if ledger.Found {
		response["repository"] = repoaudit.Summarize(ledger.State)
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleListRepositoryReviewRepositoryFindingsCollection(w http.ResponseWriter, r *http.Request) {
	listRequest, ok := parseCollectionListRequest(w, r, repositoryReviewRepositoryFindingCollectionSchema)
	if !ok {
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	summaries := make([]repositoryReviewRepositoryFindingCollectionSummary, 0, len(ledger.State.RepositoryFindings))
	if ledger.Found {
		for _, finding := range ledger.State.RepositoryFindings {
			summaries = append(summaries, projectRepositoryReviewRepositoryFindingCollectionSummary(finding))
		}
	}
	contextID := repositoryReviewCollectionCursorContext("repository-findings", ledger.Automation.ID)
	page, pageErr := collectionquery.Paginate(
		summaries,
		listRequest.Query,
		listRequest.Cursor,
		listRequest.Limit,
		listRequest.Now,
		repositoryReviewRepositoryFindingPageOptions(contextID),
	)
	if pageErr != nil {
		writeCollectionPageError(w, pageErr)
		return
	}
	repositories, titles, paths, symbols := []string{}, []string{}, []string{}, []string{}
	for _, finding := range summaries {
		repositories = append(repositories, finding.Repository)
		titles = append(titles, finding.CanonicalTitle)
		paths = append(paths, finding.Path)
		symbols = append(symbols, finding.Symbol)
	}
	response := map[string]any{
		"automation":          projectRepositoryReviewAutomation(ledger.Automation),
		"repository_findings": page.Items,
		"total":               page.Total,
		"next_cursor":         page.NextCursor,
		"canonical_query":     listRequest.Query.Canonical(),
		"query_schema": collectionSchemaWithSuggestions(
			repositoryReviewRepositoryFindingCollectionSchema,
			map[collectionquery.Field][]string{
				"repository": repositories, "title": titles, "path": paths, "symbol": symbols,
			},
		),
		"capabilities": repositoryReviewGlobalCapabilities(ledger),
	}
	if ledger.Found {
		response["repository"] = repoaudit.Summarize(ledger.State)
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func (h *Handler) handleListRepositoryReviewIssuesCollection(w http.ResponseWriter, r *http.Request) {
	generationID, listRequest, ok := parseRepositoryReviewIssueCollectionRequest(w, r)
	if !ok {
		return
	}
	ledger, err := h.repositoryReviewAutomationLedger(r.Context(), r.PathValue("automation_id"))
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	summaries := []repositoryReviewIssueCollectionSummary{}
	if ledger.Found {
		for _, draft := range ledger.State.IssueDrafts {
			if generationID == "" || draft.GenerationID == generationID {
				summaries = append(summaries, projectRepositoryReviewIssueCollectionSummary(ledger.State, draft))
			}
		}
	}
	contextID := repositoryReviewCollectionCursorContext("issues", ledger.Automation.ID, generationID)
	page, pageErr := collectionquery.Paginate(
		summaries,
		listRequest.Query,
		listRequest.Cursor,
		listRequest.Limit,
		listRequest.Now,
		repositoryReviewIssuePageOptions(contextID),
	)
	if pageErr != nil {
		writeCollectionPageError(w, pageErr)
		return
	}
	repositories, titles, generations := []string{}, []string{}, []string{}
	for _, issue := range summaries {
		repositories = append(repositories, issue.Repository)
		titles = append(titles, issue.Title)
		generations = append(generations, issue.GenerationID)
	}
	response := map[string]any{
		"automation":      projectRepositoryReviewAutomation(ledger.Automation),
		"issues":          page.Items,
		"total":           page.Total,
		"next_cursor":     page.NextCursor,
		"canonical_query": listRequest.Query.Canonical(),
		"query_schema": collectionSchemaWithSuggestions(
			repositoryReviewIssueCollectionSchema,
			map[collectionquery.Field][]string{
				"repository": repositories, "title": titles, "generation": generations,
			},
		),
		"capabilities": repositoryReviewGlobalCapabilities(ledger),
	}
	if generationID != "" {
		response["generation_id"] = generationID
	}
	if ledger.Found {
		response["repository"] = repoaudit.Summarize(ledger.State)
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func projectRepositoryReviewRunFindingSummary(
	finding repoaudit.Finding,
	runStatus repositoryReviewRunFindingStatus,
) repositoryReviewRunFindingSummary {
	contributors := repositoryReviewFindingContributors(finding)
	return repositoryReviewRunFindingSummary{
		ID: finding.ID, Repository: finding.Repository,
		Path: finding.File.Path, Line: finding.Line,
		Severity: finding.Severity, Title: finding.Title, Symbol: finding.Symbol,
		Status: finding.Status, RunFindingStatus: runStatus,
		Association:         repositoryReviewRunFindingAssociation(runStatus),
		RepositoryFindingID: finding.RepositoryFindingID,
		Contributors:        contributors,
		CreatedAt:           finding.CreatedAt, UpdatedAt: finding.UpdatedAt,
	}
}

func projectRepositoryReviewRepositoryFindingCollectionSummary(
	finding repoaudit.RepositoryFinding,
) repositoryReviewRepositoryFindingCollectionSummary {
	projected := repositoryReviewRepositoryFindingSummary(finding)
	path, symbol := "", ""
	if len(projected.PathSymbolHistory) > 0 {
		latest := projected.PathSymbolHistory[len(projected.PathSymbolHistory)-1]
		path, symbol = latest.Path, latest.Symbol
	}
	issueState := projected.Issue.State
	if issueState == "" {
		issueState = repoaudit.RepositoryFindingIssueNone
	}
	return repositoryReviewRepositoryFindingCollectionSummary{
		ID: projected.ID, Repository: projected.Repository,
		CanonicalTitle: projected.CanonicalTitle, CanonicalSeverity: projected.CanonicalSeverity,
		Path: path, Symbol: symbol,
		MatchState: projected.MatchState, Lifecycle: projected.Lifecycle,
		Issue: repositoryReviewRepositoryFindingIssueSummary{
			URL: projected.Issue.URL, State: issueState,
			SnapshotAt: projected.Issue.SnapshotAt, Conflict: projected.Issue.Conflict,
		},
		ValidationState: projected.ValidationState,
		OccurrenceCount: projected.OccurrenceCount, FoundCommitCount: projected.FoundCommitCount,
		CreatedAt: projected.CreatedAt, UpdatedAt: projected.UpdatedAt,
	}
}

func projectRepositoryReviewIssueCollectionSummary(
	state repoaudit.RepositoryState,
	draft repoaudit.IssueDraft,
) repositoryReviewIssueCollectionSummary {
	origin := draft.Origin
	if origin == "" {
		origin = repoaudit.IssueDraftOriginLegacy
	}
	eligibility := repoaudit.EvaluateIssuePublication(state, draft)
	return repositoryReviewIssueCollectionSummary{
		ID: draft.ID, Repository: draft.Repository,
		FindingCount: len(draft.FindingIDs),
		Origin:       origin, GenerationID: draft.GenerationID,
		Canonical:       draft.Canonical,
		Publishable:     eligibility.CanPublish,
		PublishBlockers: eligibility.PublishBlockers,
		Title:           draft.Title, State: draft.State,
		Version: draft.Version, CreatedAt: draft.CreatedAt, UpdatedAt: draft.UpdatedAt,
	}
}

func repositoryReviewRunFindingPageOptions(
	contextID string,
) collectionquery.PageOptions[repositoryReviewRunFindingSummary] {
	return collectionquery.PageOptions[repositoryReviewRunFindingSummary]{
		ID: func(finding repositoryReviewRunFindingSummary) (string, error) {
			return repositoryReviewCollectionCursorItemID(contextID, finding.ID)
		},
		ValidateID: repositoryReviewCollectionCursorIDValidator(contextID),
		Resolve: func(
			finding repositoryReviewRunFindingSummary,
			field collectionquery.Field,
			_ time.Time,
		) (collectionquery.FieldValue, bool) {
			return repositoryReviewRunFindingCollectionField(finding, field)
		},
		Compare: repositoryReviewSeverityComparator,
	}
}

func repositoryReviewRepositoryFindingPageOptions(
	contextID string,
) collectionquery.PageOptions[repositoryReviewRepositoryFindingCollectionSummary] {
	return collectionquery.PageOptions[repositoryReviewRepositoryFindingCollectionSummary]{
		ID: func(finding repositoryReviewRepositoryFindingCollectionSummary) (string, error) {
			return repositoryReviewCollectionCursorItemID(contextID, finding.ID)
		},
		ValidateID: repositoryReviewCollectionCursorIDValidator(contextID),
		Resolve: func(
			finding repositoryReviewRepositoryFindingCollectionSummary,
			field collectionquery.Field,
			_ time.Time,
		) (collectionquery.FieldValue, bool) {
			return repositoryReviewRepositoryFindingCollectionField(finding, field)
		},
		Compare: repositoryReviewSeverityComparator,
	}
}

func repositoryReviewIssuePageOptions(
	contextID string,
) collectionquery.PageOptions[repositoryReviewIssueCollectionSummary] {
	return collectionquery.PageOptions[repositoryReviewIssueCollectionSummary]{
		ID: func(issue repositoryReviewIssueCollectionSummary) (string, error) {
			return repositoryReviewCollectionCursorItemID(contextID, issue.ID)
		},
		ValidateID: repositoryReviewCollectionCursorIDValidator(contextID),
		Resolve: func(
			issue repositoryReviewIssueCollectionSummary,
			field collectionquery.Field,
			_ time.Time,
		) (collectionquery.FieldValue, bool) {
			return repositoryReviewIssueCollectionField(issue, field)
		},
	}
}

func repositoryReviewRunFindingCollectionField(
	finding repositoryReviewRunFindingSummary,
	field collectionquery.Field,
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
	case "created":
		return collectionquery.TimestampValue(finding.CreatedAt), true
	case "updated":
		return collectionquery.TimestampValue(finding.UpdatedAt), true
	default:
		return collectionquery.FieldValue{}, false
	}
}

func repositoryReviewRepositoryFindingCollectionField(
	finding repositoryReviewRepositoryFindingCollectionSummary,
	field collectionquery.Field,
) (collectionquery.FieldValue, bool) {
	switch field {
	case "id":
		return collectionquery.StringValue(finding.ID), true
	case "repository":
		return collectionquery.StringValue(finding.Repository), true
	case "title":
		return collectionquery.StringValue(finding.CanonicalTitle), true
	case "path":
		return collectionquery.StringValue(finding.Path), true
	case "symbol":
		return collectionquery.StringValue(finding.Symbol), true
	case "severity":
		return collectionquery.EnumValue(finding.CanonicalSeverity), true
	case "match":
		return collectionquery.EnumValue(string(finding.MatchState)), true
	case "lifecycle":
		return collectionquery.EnumValue(string(finding.Lifecycle)), true
	case "issue":
		return collectionquery.EnumValue(string(finding.Issue.State)), true
	case "validation":
		return collectionquery.EnumValue(string(finding.ValidationState)), true
	case "occurrences":
		return collectionquery.NumberValue(float64(finding.OccurrenceCount)), true
	case "commits":
		return collectionquery.NumberValue(float64(finding.FoundCommitCount)), true
	case "created":
		return collectionquery.TimestampValue(finding.CreatedAt), true
	case "updated":
		return collectionquery.TimestampValue(finding.UpdatedAt), true
	default:
		return collectionquery.FieldValue{}, false
	}
}

func repositoryReviewIssueCollectionField(
	issue repositoryReviewIssueCollectionSummary,
	field collectionquery.Field,
) (collectionquery.FieldValue, bool) {
	switch field {
	case "id":
		return collectionquery.StringValue(issue.ID), true
	case "repository":
		return collectionquery.StringValue(issue.Repository), true
	case "title":
		return collectionquery.StringValue(issue.Title), true
	case "generation":
		return collectionquery.StringValue(issue.GenerationID), true
	case "state":
		return collectionquery.EnumValue(string(issue.State)), true
	case "origin":
		return collectionquery.EnumValue(string(issue.Origin)), true
	case "canonical":
		return collectionquery.BooleanValue(issue.Canonical), true
	case "publishable":
		return collectionquery.BooleanValue(issue.Publishable), true
	case "findings":
		return collectionquery.NumberValue(float64(issue.FindingCount)), true
	case "created":
		return collectionquery.TimestampValue(issue.CreatedAt), true
	case "updated":
		return collectionquery.TimestampValue(issue.UpdatedAt), true
	default:
		return collectionquery.FieldValue{}, false
	}
}

func repositoryReviewSeverityComparator(
	field collectionquery.Field,
	left collectionquery.FieldValue,
	right collectionquery.FieldValue,
) (int, bool) {
	if field != "severity" {
		return 0, false
	}
	return repositoryReviewSeverityRank(left.Text) - repositoryReviewSeverityRank(right.Text), true
}

func repositoryReviewSeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func repositoryReviewRunFindingAssociation(status repositoryReviewRunFindingStatus) string {
	switch status {
	case repositoryReviewRunFindingAssociatedNew:
		return "new"
	case repositoryReviewRunFindingAssociatedExisting:
		return "existing"
	case repositoryReviewRunFindingNeedsReview:
		return "needs_review"
	default:
		return "unassociated"
	}
}

func repositoryReviewFindingContributors(finding repoaudit.Finding) []string {
	seen := make(map[string]struct{}, len(finding.Models)+len(finding.Observations))
	contributors := make([]string, 0, len(seen))
	appendContributor := func(value string) {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			return
		}
		if _, duplicate := seen[key]; duplicate {
			return
		}
		seen[key] = struct{}{}
		contributors = append(contributors, value)
	}
	for _, observation := range finding.Observations {
		if strings.TrimSpace(observation.Reviewer) != "" {
			appendContributor(observation.Reviewer)
		} else {
			appendContributor(observation.Model)
		}
	}
	for _, model := range finding.Models {
		appendContributor(model)
	}
	sort.SliceStable(contributors, func(i, j int) bool {
		return strings.ToLower(contributors[i]) < strings.ToLower(contributors[j])
	})
	return contributors
}

func repositoryReviewCollectionCursorContext(parts ...string) string {
	identity, _ := json.Marshal(parts)
	contextID, _ := encodeCollectionResourceID(
		"repository-review-cursor-context",
		string(identity),
	)
	return contextID
}

func repositoryReviewCollectionCursorItemID(contextID, itemID string) (string, error) {
	if !validCollectionResourceID(contextID) {
		return "", errors.New("invalid repository review collection context")
	}
	encodedItem, err := encodeCollectionResourceID("repository-review-cursor-item", strings.TrimSpace(itemID))
	if err != nil {
		return "", err
	}
	return contextID + "." + encodedItem, nil
}

func repositoryReviewCollectionCursorIDValidator(contextID string) func(string) bool {
	prefix := contextID + "."
	return func(value string) bool {
		if !strings.HasPrefix(value, prefix) {
			return false
		}
		encodedItem := strings.TrimPrefix(value, prefix)
		return validCollectionResourceID(contextID) && validCollectionResourceID(encodedItem)
	}
}

func parseRepositoryReviewIssueCollectionRequest(
	w http.ResponseWriter,
	r *http.Request,
) (string, collectionListRequest, bool) {
	if r == nil || r.URL == nil {
		writeCollectionError(
			w,
			http.StatusBadRequest,
			"invalid_collection_request",
			"Invalid collection request",
			-1,
			nil,
		)
		return "", collectionListRequest{}, false
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeCollectionError(
			w,
			http.StatusBadRequest,
			"invalid_collection_request",
			"Collection query parameters are malformed",
			-1,
			nil,
		)
		return "", collectionListRequest{}, false
	}
	for key, entries := range values {
		if (key != "query" && key != "cursor" && key != "limit" && key != "generation_id") || len(entries) != 1 {
			writeCollectionError(
				w,
				http.StatusBadRequest,
				"invalid_collection_request",
				"Only query, cursor, limit, and generation_id are supported",
				-1,
				nil,
			)
			return "", collectionListRequest{}, false
		}
	}
	generationID := strings.TrimSpace(values.Get("generation_id"))
	if generationID != "" && !repositoryReviewValidGenerationText(generationID, 256, false) {
		writeCollectionError(w, http.StatusBadRequest, "invalid_generation_id", "Generation ID is invalid", -1, nil)
		return "", collectionListRequest{}, false
	}
	collectionRequest := r.Clone(r.Context())
	collectionURL := *r.URL
	values.Del("generation_id")
	collectionURL.RawQuery = values.Encode()
	collectionRequest.URL = &collectionURL
	parsed, ok := parseCollectionListRequest(w, collectionRequest, repositoryReviewIssueCollectionSchema)
	return generationID, parsed, ok
}

func repositoryReviewUsesIssueCollectionRequest(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	return err != nil || values.Has("query") || values.Has("cursor")
}
