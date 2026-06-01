package toolshared

import (
	"encoding/json"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	HandledToolLLMNote   = "The requested output has already been delivered to the user in the current chat. Do not call send_file or any other delivery tool again. If you reply, provide only a brief confirmation."
	ArtifactPathsLLMNote = "Use `send_file` with one of these paths to send it to the user, or use file/exec tools to save it inside the workspace if requested."
)

type AsyncDeliveryMode string

const (
	AsyncDeliveryUserOnly      AsyncDeliveryMode = "user_only"
	AsyncDeliveryParentOnly    AsyncDeliveryMode = "parent_only"
	AsyncDeliveryUserAndParent AsyncDeliveryMode = "user_and_parent"
)

// ToolResult represents the structured return value from tool execution.
// It provides clear semantics for different types of results and supports
// async operations, user-facing messages, and error handling.
type ToolResult struct {
	// ForLLM is the content sent to the LLM for context.
	// Required for all results.
	ForLLM string `json:"for_llm"`

	// ForUser is the content sent directly to the user.
	// If empty, no user message is sent.
	// Silent=true overrides this field.
	ForUser string `json:"for_user,omitempty"`

	// Silent suppresses sending any message to the user.
	// When true, ForUser is ignored even if set.
	Silent bool `json:"silent"`

	// IsError indicates whether the tool execution failed.
	// When true, the result should be treated as an error.
	IsError bool `json:"is_error"`

	// Async indicates whether the tool is running asynchronously.
	// When true, the tool will complete later and notify via callback.
	Async bool `json:"async"`

	// AsyncDelivery controls how the final async result should be routed when
	// the background work completes.
	//
	// Empty means "use runtime default behavior".
	// Supported values:
	//   - user_only
	//   - parent_only
	//   - user_and_parent
	AsyncDelivery AsyncDeliveryMode `json:"async_delivery,omitempty"`

	// AsyncTaskID links an async result back to a durable task registry record
	// when the tool runtime has one, for example "subagent-42".
	AsyncTaskID string `json:"async_task_id,omitempty"`

	// Err is the underlying error (not JSON serialized).
	// Used for internal error handling and logging.
	Err error `json:"-"`

	// Media contains media store refs produced by this tool.
	// When non-empty, the agent will publish these as OutboundMediaMessage.
	Media []string `json:"media,omitempty"`

	// Completion carries a structured child-run result for parent agents.
	// It is populated by sub-turns/delegation/spawn handoffs so the parent can
	// see both text and media refs without scraping prose.
	// Deprecated for new consumers: use Deliverable for produced outputs and
	// keep Completion only as a legacy child-run handoff adapter.
	Completion *CompletionResult `json:"completion,omitempty"`

	// Deliverable describes the actual artifact/result produced by the tool,
	// independent from LLM context or user-facing phrasing. Use this for durable
	// task state and follow-up ownership; Completion is kept as the legacy
	// child-run handoff shape and is mirrored into Deliverable when possible.
	Deliverable *DeliverableResult `json:"deliverable,omitempty"`

	// Messages holds the ephemeral session history after execution.
	// Only populated by SubTurn executions; used by evaluator_optimizer
	// to carry stateful worker context across evaluation iterations.
	Messages []providers.Message `json:"-"`

	// ArtifactTags exposes local artifact paths back to the LLM in a structured
	// form, e.g. "[file:/tmp/example.png]". This is used when a tool produced a
	// reusable local artifact but did not deliver it to the user yet.
	ArtifactTags []string `json:"artifact_tags,omitempty"`

	// ResponseHandled indicates that this tool execution already satisfied the
	// user's request at the channel/output level, so the agent loop can stop
	// without a follow-up assistant response.
	ResponseHandled bool `json:"response_handled,omitempty"`
}

// CompletionResult is the structured handoff payload used when one agent run
// completes work for another. It intentionally describes data/artifacts only;
// the caller still owns any user-facing delivery decision.
type CompletionResult struct {
	Text  string            `json:"text,omitempty"`
	Media []CompletionMedia `json:"media,omitempty"`
}

// CompletionMedia describes a media artifact in a child completion handoff.
// Ref is the stable media:// reference used by the runtime to resolve bytes;
// the remaining fields are hints for delivery policy and UI mapping.
type CompletionMedia struct {
	Ref         string `json:"ref"`
	Type        string `json:"type,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// DeliverableResult is the generic output ownership payload for tools and
// child runs. It represents what was produced, not how it should be worded to
// the LLM or user.
type DeliverableResult struct {
	Text      string            `json:"text,omitempty"`
	Artifacts []DeliverableItem `json:"artifacts,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// DeliverableItem describes a concrete produced artifact. Ref may be a
// media:// ref, file path tag, external URL, or other stable runtime reference.
type DeliverableItem struct {
	Ref         string `json:"ref"`
	Kind        string `json:"kind,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Delivered   bool   `json:"delivered,omitempty"`
}

// ContentForLLM returns the normalized textual content to append to the
// conversation after a tool call. Errors fall back to Err when ForLLM is empty.
func (tr *ToolResult) ContentForLLM() string {
	if tr == nil {
		return ""
	}
	content := tr.ForLLM
	if content == "" && tr.Err != nil {
		content = tr.Err.Error()
	}
	if tr.ResponseHandled {
		if content == "" {
			return HandledToolLLMNote
		}
		if !strings.Contains(content, HandledToolLLMNote) {
			content += "\n" + HandledToolLLMNote
		}
	}
	if len(tr.ArtifactTags) > 0 {
		artifactNote := "Local artifact paths: " + strings.Join(tr.ArtifactTags, " ") + "\n" + ArtifactPathsLLMNote
		if content == "" {
			content = artifactNote
		} else if !strings.Contains(content, artifactNote) {
			content += "\n" + artifactNote
		}
	}
	content = appendUniqueLLMNote(content, tr.completionNoteForLLM())
	content = appendUniqueLLMNote(content, tr.deliverableNoteForLLM())
	if content != "" {
		return content
	}
	return ""
}

func appendUniqueLLMNote(content, note string) string {
	if note == "" {
		return content
	}
	if content == "" {
		return note
	}
	if !strings.Contains(content, note) {
		return content + "\n" + note
	}
	return content
}

func (tr *ToolResult) completionNoteForLLM() string {
	if tr == nil || tr.Completion == nil {
		return ""
	}
	type completionForLLM struct {
		Text  string            `json:"text,omitempty"`
		Media []CompletionMedia `json:"media,omitempty"`
	}
	payload := completionForLLM{
		Text:  strings.TrimSpace(tr.Completion.Text),
		Media: append([]CompletionMedia(nil), tr.Completion.Media...),
	}
	if payload.Text == "" && len(payload.Media) == 0 {
		return ""
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return "Structured child completion: " + string(data)
}

func (tr *ToolResult) deliverableNoteForLLM() string {
	if tr == nil || tr.Deliverable == nil {
		return ""
	}
	if tr.Completion != nil {
		// Completion already renders a child-handoff note. The mirrored
		// deliverable is still available to registries, but adding both notes
		// would duplicate the same payload in model context.
		return ""
	}
	payload := tr.effectiveDeliverable()
	if payload == nil {
		return ""
	}
	if strings.TrimSpace(payload.Text) == "" && len(payload.Artifacts) == 0 && len(payload.Metadata) == 0 {
		return ""
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return "Structured deliverable: " + string(data)
}

func (tr *ToolResult) effectiveDeliverable() *DeliverableResult {
	if tr == nil {
		return nil
	}
	if tr.Deliverable != nil {
		return tr.Deliverable
	}
	if tr.Completion == nil {
		return nil
	}
	deliverable := &DeliverableResult{Text: tr.Completion.Text}
	for _, item := range tr.Completion.Media {
		deliverable.Artifacts = append(deliverable.Artifacts, DeliverableItem{
			Ref:         item.Ref,
			Kind:        item.Type,
			Filename:    item.Filename,
			ContentType: item.ContentType,
		})
	}
	if strings.TrimSpace(deliverable.Text) == "" && len(deliverable.Artifacts) == 0 {
		return nil
	}
	return deliverable
}

// NewToolResult creates a basic ToolResult with content for the LLM.
// Use this when you need a simple result with default behavior.
//
// Example:
//
//	result := NewToolResult("File updated successfully")
func NewToolResult(forLLM string) *ToolResult {
	return &ToolResult{
		ForLLM: forLLM,
	}
}

// SilentResult creates a ToolResult that is silent (no user message).
// The content is only sent to the LLM for context.
//
// Use this for operations that should not spam the user, such as:
// - File reads/writes
// - Status updates
// - Background operations
//
// Example:
//
//	result := SilentResult("Config file saved")
func SilentResult(forLLM string) *ToolResult {
	return &ToolResult{
		ForLLM:  forLLM,
		Silent:  true,
		IsError: false,
		Async:   false,
	}
}

// AsyncResult creates a ToolResult for async operations.
// The task will run in the background and complete later.
//
// Use this for long-running operations like:
// - Subagent spawns
// - Background processing
// - External API calls with callbacks
//
// Example:
//
//	result := AsyncResult("Subagent spawned, will report back")
func AsyncResult(forLLM string) *ToolResult {
	return &ToolResult{
		ForLLM:  forLLM,
		Silent:  false,
		IsError: false,
		Async:   true,
	}
}

// ErrorResult creates a ToolResult representing an error.
// Sets IsError=true and includes the error message.
//
// Example:
//
//	result := ErrorResult("Failed to connect to database: connection refused")
func ErrorResult(message string) *ToolResult {
	return &ToolResult{
		ForLLM:  message,
		Silent:  false,
		IsError: true,
		Async:   false,
	}
}

// UserResult creates a ToolResult with content for both LLM and user.
// Both ForLLM and ForUser are set to the same content.
//
// Use this when the user needs to see the result directly:
// - Command execution output
// - Fetched web content
// - Query results
//
// Example:
//
//	result := UserResult("Total files found: 42")
func UserResult(content string) *ToolResult {
	return &ToolResult{
		ForLLM:  content,
		ForUser: content,
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// MediaResult creates a ToolResult with media refs for the user.
// The agent will publish these refs as OutboundMediaMessage.
//
// Example:
//
//	result := MediaResult("Image generated successfully", []string{"media://abc123"})
func MediaResult(forLLM string, mediaRefs []string) *ToolResult {
	result := &ToolResult{
		ForLLM: forLLM,
		Media:  mediaRefs,
	}
	if len(mediaRefs) > 0 {
		result.Deliverable = &DeliverableResult{}
		for _, ref := range mediaRefs {
			result.Deliverable.Artifacts = append(result.Deliverable.Artifacts, DeliverableItem{
				Ref:  ref,
				Kind: "media",
			})
		}
	}
	return result
}

// MarshalJSON implements custom JSON serialization.
// The Err field is excluded from JSON output via the json:"-" tag.
func (tr *ToolResult) MarshalJSON() ([]byte, error) {
	type Alias ToolResult
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(tr),
	})
}

// WithError sets the Err field and returns the result for chaining.
// This preserves the error for logging while keeping it out of JSON.
//
// Example:
//
//	result := ErrorResult("Operation failed").WithError(err)
func (tr *ToolResult) WithError(err error) *ToolResult {
	tr.Err = err
	return tr
}

// WithResponseHandled marks the tool result as already delivered to the user.
func (tr *ToolResult) WithResponseHandled() *ToolResult {
	tr.ResponseHandled = true
	return tr
}

// WithAsyncDelivery sets the async delivery policy for this tool result.
func (tr *ToolResult) WithAsyncDelivery(mode AsyncDeliveryMode) *ToolResult {
	tr.AsyncDelivery = mode
	return tr
}

// WithAsyncTaskID links this result to a durable async task registry record.
func (tr *ToolResult) WithAsyncTaskID(taskID string) *ToolResult {
	tr.AsyncTaskID = strings.TrimSpace(taskID)
	return tr
}

// WithCompletion attaches a structured completion payload to this result.
func (tr *ToolResult) WithCompletion(completion *CompletionResult) *ToolResult {
	tr.Completion = completion
	if tr.Deliverable == nil && completion != nil {
		tr.Deliverable = tr.effectiveDeliverable()
	}
	return tr
}

// WithDeliverable attaches a generic durable output payload to this result.
func (tr *ToolResult) WithDeliverable(deliverable *DeliverableResult) *ToolResult {
	tr.Deliverable = deliverable
	return tr
}
