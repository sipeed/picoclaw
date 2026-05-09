package seahorse

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tokenizer"
)

// SummaryKind distinguishes leaf summaries (from raw messages) vs condensed
// summaries (from other summaries).
type SummaryKind string

const (
	SummaryKindLeaf      SummaryKind = "leaf"
	SummaryKindCondensed SummaryKind = "condensed"
)

// Message represents a single chat message with role and content.
type Message struct {
	ID               int64         `json:"id"`
	ConversationID   int64         `json:"conversationId"`
	Role             string        `json:"role"`
	Content          string        `json:"content"`
	ReasoningContent string        `json:"reasoningContent,omitempty"`
	TokenCount       int           `json:"tokenCount"`
	CreatedAt        time.Time     `json:"createdAt"`
	Parts            []MessagePart `json:"parts,omitempty"`
}

// MessagePart holds structured content (tool calls, media, etc.)
type MessagePart struct {
	ID         int64  `json:"id"`
	MessageID  int64  `json:"messageId"`
	Type       string `json:"type"` // "text", "tool_use", "tool_result", "media"
	Text       string `json:"text"`
	Name       string `json:"name"`
	Arguments  string `json:"arguments"`
	ToolCallID string `json:"toolCallId"`
	MediaURI   string `json:"mediaUri"`
	MimeType   string `json:"mimeType"`
}

// Summary represents a compressed representation of messages or other summaries.
type Summary struct {
	SummaryID               string      `json:"summaryId"`
	ConversationID          int64       `json:"conversationId"`
	Kind                    SummaryKind `json:"kind"`
	Depth                   int         `json:"depth"`
	Content                 string      `json:"content"`
	TokenCount              int         `json:"tokenCount"`
	EarliestAt              *time.Time  `json:"earliestAt,omitempty"`
	LatestAt                *time.Time  `json:"latestAt,omitempty"`
	DescendantCount         int         `json:"descendantCount"`
	DescendantTokenCount    int         `json:"descendantTokenCount"`
	SourceMessageTokenCount int         `json:"sourceMessageTokenCount"`
	Model                   string      `json:"model"`
	CreatedAt               time.Time   `json:"createdAt"`
}

// SummaryNode is a Summary with graph relationships for tree traversal.
type SummaryNode struct {
	Summary
	Children []string `json:"children"` // Child summary IDs
	Expanded bool     `json:"expanded"` // UI state for expansion
}

// Conversation represents a session's conversation with metadata.
type Conversation struct {
	ConversationID int64     `json:"conversationId"`
	SessionKey     string    `json:"sessionKey"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// SessionStatus contains status information for a session.
type SessionStatus struct {
	SessionKey     string    `json:"sessionKey"`
	ConversationID int64     `json:"conversationId"`
	Messages       int       `json:"messages"`
	TotalTokens    int       `json:"totalTokens"`
	Summaries      int       `json:"summaries"`
	OldestAt       time.Time `json:"oldestAt"`
	NewestAt       time.Time `json:"newestAt"`
}

// ContextItem represents one item in the assembled context window.
type ContextItem struct {
	ConversationID int64     `json:"conversationId"`
	Ordinal        int       `json:"ordinal"`
	ItemType       string    `json:"itemType"` // "summary" or "message"
	SummaryID      string    `json:"summaryId,omitempty"`
	MessageID      int64     `json:"messageId,omitempty"`
	TokenCount     int       `json:"tokenCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

// SummarySubtreeNode is a node in a summary DAG subtree.
type SummarySubtreeNode struct {
	SummaryID     string `json:"summaryId"`
	DepthFromRoot int    `json:"depthFromRoot"`
}

// SearchInput controls summary search.
type SearchInput struct {
	Pattern          string     `json:"pattern"`
	Mode             string     `json:"mode"`            // "like" (LIKE search) or "full_text" (FTS5, default)
	Scope            string     `json:"scope,omitempty"` // "messages", "summaries", "both"
	Role             string     `json:"role,omitempty"`  // "user", "assistant", or "" (all)
	Since            *time.Time `json:"since,omitempty"`
	Before           *time.Time `json:"before,omitempty"`
	Limit            int        `json:"limit,omitempty"`
	ConversationID   int64      `json:"conversationId,omitempty"`
	AllConversations bool       `json:"allConversations,omitempty"`
}

// SearchResult is a search match.
type SearchResult struct {
	SummaryID      string      `json:"summaryId,omitempty"`
	MessageID      int64       `json:"messageId,omitempty"`
	ConversationID int64       `json:"conversationId"`
	Kind           SummaryKind `json:"kind,omitempty"`
	Depth          int         `json:"depth,omitempty"`
	Role           string      `json:"role,omitempty"`
	Content        string      `json:"content,omitempty"` // Full content for summaries
	Snippet        string      `json:"snippet"`
	CreatedAt      time.Time   `json:"createdAt"`
	Rank           float64     `json:"rank,omitempty"`
	TotalCount     int         `json:"totalCount,omitempty"` // Total matching rows (from window function)
}

// ResearchGraphNode represents a node in the research knowledge graph
type ResearchGraphNode struct {
	Name string  `json:"name"`
	Abbr string  `json:"abbr"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// ResearchGraphStore manages research graph nodes
type ResearchGraphStore interface {
	ListNodes() ([]ResearchGraphNode, error)
	UpdateNode(node ResearchGraphNode) error
}

// EstimateMessageTokens estimates token count for a full message using the
// shared tokenizer package for consistency with agent.context_budget.
func EstimateMessageTokens(msg Message) int {
	pm := providers.Message{
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
	}

	// Convert MessageParts to ToolCalls / ToolCallID / Media
	for _, part := range msg.Parts {
		switch part.Type {
		case "tool_use":
			pm.ToolCalls = append(pm.ToolCalls, providers.ToolCall{
				ID:   part.ToolCallID,
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      part.Name,
					Arguments: part.Arguments,
				},
			})
		case "tool_result":
			pm.ToolCallID = part.ToolCallID
		case "media":
			pm.Media = append(pm.Media, part.MediaURI)
		}
	}

	return tokenizer.EstimateMessageTokens(pm)
}
