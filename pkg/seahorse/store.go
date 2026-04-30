package seahorse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store provides SQLite storage for seahorse.
type Store struct {
	db *sql.DB
}

// CreateSummaryInput holds parameters for creating a summary.
type CreateSummaryInput struct {
	ConversationID       int64
	Kind                 SummaryKind
	Depth                int
	Content              string
	TokenCount           int
	EarliestAt           *time.Time
	LatestAt             *time.Time
	DescendantCount      int
	DescendantTokenCount int
	SourceMessageTokens  int
	Model                string
	ParentIDs            []string // For condensed: child summary IDs being condensed
}

// --- Conversation Operations ---

// GetOrCreateConversation returns the conversation for a sessionKey, creating if needed.
func (s *Store) GetOrCreateConversation(ctx context.Context, sessionKey string) (*Conversation, error) {
	// Try to get first
	conv, err := s.GetConversationBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	// Create
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO conversations (session_key) VALUES (?)",
		sessionKey,
	)
	if err != nil {
		// Race: another goroutine may have inserted
		if isUniqueViolation(err) {
			return s.GetConversationBySessionKey(ctx, sessionKey)
		}
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	id, _ := result.LastInsertId()
	return &Conversation{
		ConversationID: id,
		SessionKey:     sessionKey,
	}, nil
}

// GetConversationBySessionKey retrieves a conversation by session key.
func (s *Store) GetConversationBySessionKey(ctx context.Context, sessionKey string) (*Conversation, error) {
	var conv Conversation
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		"SELECT conversation_id, session_key, created_at, updated_at FROM conversations WHERE session_key = ?",
		sessionKey,
	).Scan(&conv.ConversationID, &conv.SessionKey, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation by session key: %w", err)
	}
	conv.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	conv.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return &conv, nil
}

// GetSessionStatus returns status for a specific session.
func (s *Store) GetSessionStatus(ctx context.Context, sessionKey string) (*SessionStatus, error) {
	conv, err := s.GetConversationBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, nil
	}

	msgCount, _ := s.GetMessageCount(ctx, conv.ConversationID)
	sumCount, _ := s.getSummaryCount(ctx, conv.ConversationID)
	tokenCount, _ := s.GetContextTokenCount(ctx, conv.ConversationID)

	oldest, newest, _ := s.getMessageTimeRange(ctx, conv.ConversationID)

	return &SessionStatus{
		SessionKey:     conv.SessionKey,
		ConversationID: conv.ConversationID,
		Messages:       msgCount,
		TotalTokens:    tokenCount,
		Summaries:      sumCount,
		OldestAt:       oldest,
		NewestAt:       newest,
	}, nil
}

// GetAllSessionStatuses returns status for all sessions.
func (s *Store) GetAllSessionStatuses(ctx context.Context) ([]SessionStatus, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT session_key FROM conversations")
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var statuses []SessionStatus
	for rows.Next() {
		var sessionKey string
		if err := rows.Scan(&sessionKey); err != nil {
			continue
		}
		status, err := s.GetSessionStatus(ctx, sessionKey)
		if err != nil {
			continue
		}
		if status != nil {
			statuses = append(statuses, *status)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return statuses, nil
}

func (s *Store) getSummaryCount(ctx context.Context, convID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM summaries WHERE conversation_id = ?",
		convID,
	).Scan(&count)
	return count, err
}

func (s *Store) getMessageTimeRange(ctx context.Context, convID int64) (time.Time, time.Time, error) {
	var minTime, maxTime string
	err := s.db.QueryRowContext(ctx,
		"SELECT MIN(created_at), MAX(created_at) FROM messages WHERE conversation_id = ?",
		convID,
	).Scan(&minTime, &maxTime)
	if err != nil || minTime == "" {
		return time.Time{}, time.Time{}, err
	}
	oldest, _ := time.Parse("2006-01-02 15:04:05", minTime)
	newest, _ := time.Parse("2006-01-02 15:04:05", maxTime)
	return oldest, newest, nil
}

// --- Message Operations ---

// AddMessage appends a message to a conversation.
func (s *Store) AddMessage(ctx context.Context, convID int64, role, content string, tokenCount int) (*Message, error) {
	return s.AddMessageWithReasoning(ctx, convID, role, content, "", tokenCount)
}

// AddMessageWithReasoning appends a message with reasoning content to a conversation.
func (s *Store) AddMessageWithReasoning(
	ctx context.Context,
	convID int64,
	role, content, reasoningContent string,
	tokenCount int,
) (*Message, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (conversation_id, role, content, reasoning_content, token_count) VALUES (?, ?, ?, ?, ?)",
		convID, role, content, reasoningContent, tokenCount,
	)
	if err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}
	id, _ := result.LastInsertId()
	return &Message{
		ID:               id,
		ConversationID:   convID,
		Role:             role,
		Content:          content,
		ReasoningContent: reasoningContent,
		TokenCount:       tokenCount,
	}, nil
}

// partsToReadableContent derives a readable text summary from message parts.
// This ensures FTS5 indexing and summary formatting can access tool call information.
func partsToReadableContent(parts []MessagePart) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString("\n")
		}
		switch p.Type {
		case "text":
			b.WriteString(p.Text)
		case "tool_use":
			fmt.Fprintf(&b, "[tool_use: %s, args: %s]", p.Name, p.Arguments)
		case "tool_result":
			fmt.Fprintf(&b, "[tool_result for %s: %s]", p.ToolCallID, p.Text)
		case "media":
			fmt.Fprintf(&b, "[media: %s (%s)]", p.MediaURI, p.MimeType)
		default:
			if p.Text != "" {
				b.WriteString(p.Text)
			}
		}
	}
	return b.String()
}

// AddMessageWithParts adds a message with structured parts.
func (s *Store) AddMessageWithParts(
	ctx context.Context,
	convID int64,
	role string,
	parts []MessagePart,
	tokenCount int,
) (*Message, error) {
	return s.AddMessageWithPartsAndReasoning(ctx, convID, role, parts, "", tokenCount)
}

// AddMessageWithPartsAndReasoning adds a message with structured parts and reasoning content.
func (s *Store) AddMessageWithPartsAndReasoning(
	ctx context.Context,
	convID int64,
	role string,
	parts []MessagePart,
	reasoningContent string,
	tokenCount int,
) (*Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Derive readable content from Parts for FTS5 indexing and summary formatting
	readableContent := partsToReadableContent(parts)

	result, err := tx.ExecContext(ctx,
		"INSERT INTO messages (conversation_id, role, content, reasoning_content, token_count) VALUES (?, ?, ?, ?, ?)",
		convID, role, readableContent, reasoningContent, tokenCount,
	)
	if err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}
	msgID, _ := result.LastInsertId()

	for i, p := range parts {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO message_parts (message_id, type, text, name, arguments, tool_call_id, media_uri, mime_type, ordinal)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			msgID,
			p.Type,
			p.Text,
			p.Name,
			p.Arguments,
			p.ToolCallID,
			p.MediaURI,
			p.MimeType,
			i,
		)
		if err != nil {
			return nil, fmt.Errorf("add message part %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Return message with parts
	msg := &Message{
		ID:               msgID,
		ConversationID:   convID,
		Role:             role,
		ReasoningContent: reasoningContent,
		TokenCount:       tokenCount,
		Parts:            make([]MessagePart, len(parts)),
	}
	for i, p := range parts {
		p.MessageID = msgID
		msg.Parts[i] = p
	}
	return msg, nil
}

// GetMessages retrieves messages for a conversation.
func (s *Store) GetMessages(ctx context.Context, convID int64, limit int, beforeID int64) ([]Message, error) {
	query := "SELECT message_id, conversation_id, role, content, reasoning_content, token_count, created_at FROM messages WHERE conversation_id = ?"
	args := []any{convID}
	if beforeID > 0 {
		query += " AND message_id < ?"
		args = append(args, beforeID)
	}
	query += " ORDER BY message_id ASC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		var createdAt string
		if err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&msg.ReasoningContent,
			&msg.TokenCount,
			&createdAt,
		); err != nil {
			return nil, err
		}
		msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load parts for all messages
	for i := range msgs {
		parts, err := s.loadMessageParts(ctx, msgs[i].ID)
		if err != nil {
			return nil, err
		}
		msgs[i].Parts = parts
	}

	return msgs, nil
}

// GetMessageCount returns total message count for a conversation.
func (s *Store) GetMessageCount(ctx context.Context, convID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT count(*) FROM messages WHERE conversation_id = ?", convID,
	).Scan(&count)
	return count, err
}

// GetMessageByID retrieves a single message by ID.
func (s *Store) GetMessageByID(ctx context.Context, messageID int64) (*Message, error) {
	var msg Message
	var createdAt string
	err := s.db.QueryRowContext(
		ctx,
		"SELECT message_id, conversation_id, role, content, reasoning_content, token_count, created_at FROM messages WHERE message_id = ?",
		messageID,
	).Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.ReasoningContent, &msg.TokenCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message %d not found", messageID)
	}
	if err != nil {
		return nil, err
	}
	msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	msg.Parts, _ = s.loadMessageParts(ctx, msg.ID)
	return &msg, nil
}

// UpdateMessageReasoningContent updates reasoning_content for an existing message.
func (s *Store) UpdateMessageReasoningContent(ctx context.Context, messageID int64, reasoningContent string) error {
	result, err := s.db.ExecContext(
		ctx,
		"UPDATE messages SET reasoning_content = ? WHERE message_id = ?",
		reasoningContent,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("update message reasoning_content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update message reasoning_content rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("message %d not found", messageID)
	}
	return nil
}

func (s *Store) loadMessageParts(ctx context.Context, msgID int64) ([]MessagePart, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT part_id, message_id, type, text, name, arguments, tool_call_id, media_uri, mime_type
		 FROM message_parts WHERE message_id = ? ORDER BY ordinal`,
		msgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []MessagePart
	for rows.Next() {
		var p MessagePart
		if err := rows.Scan(&p.ID, &p.MessageID, &p.Type, &p.Text, &p.Name, &p.Arguments,
			&p.ToolCallID, &p.MediaURI, &p.MimeType); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return parts, nil
}

// --- Summary Operations ---

// CreateSummary creates a new summary and indexes it in FTS5.
func (s *Store) CreateSummary(ctx context.Context, input CreateSummaryInput) (*Summary, error) {
	// Generate summary ID
	now := time.Now().UTC()
	summaryID := generateSummaryID(input.Content, now)

	var earliestAt, latestAt sql.NullString
	if input.EarliestAt != nil {
		earliestAt = sql.NullString{String: input.EarliestAt.Format(time.RFC3339), Valid: true}
	}
	if input.LatestAt != nil {
		latestAt = sql.NullString{String: input.LatestAt.Format(time.RFC3339), Valid: true}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO summaries (summary_id, conversation_id, kind, depth, content, token_count,
			earliest_at, latest_at, descendant_count, descendant_token_count,
			source_message_token_count, model)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		summaryID, input.ConversationID, string(input.Kind), input.Depth,
		input.Content, input.TokenCount,
		earliestAt, latestAt,
		input.DescendantCount, input.DescendantTokenCount,
		input.SourceMessageTokens, input.Model,
	)
	if err != nil {
		return nil, fmt.Errorf("insert summary: %w", err)
	}

	// FTS trigger will fire automatically for summaries table insert

	// Link parent summaries (DAG edges) for condensed summaries
	for _, parentID := range input.ParentIDs {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO summary_parents (summary_id, parent_summary_id) VALUES (?, ?)",
			summaryID, parentID,
		)
		if err != nil {
			return nil, fmt.Errorf("link parent %s: %w", parentID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &Summary{
		SummaryID:               summaryID,
		ConversationID:          input.ConversationID,
		Kind:                    input.Kind,
		Depth:                   input.Depth,
		Content:                 input.Content,
		TokenCount:              input.TokenCount,
		EarliestAt:              input.EarliestAt,
		LatestAt:                input.LatestAt,
		DescendantCount:         input.DescendantCount,
		DescendantTokenCount:    input.DescendantTokenCount,
		SourceMessageTokenCount: input.SourceMessageTokens,
		Model:                   input.Model,
		CreatedAt:               now,
	}, nil
}

// GetSummary retrieves a summary by ID.
func (s *Store) GetSummary(ctx context.Context, summaryID string) (*Summary, error) {
	return s.scanSummary(ctx, "WHERE summary_id = ?", summaryID)
}

// GetSummariesByConversation retrieves all summaries for a conversation.
func (s *Store) GetSummariesByConversation(ctx context.Context, convID int64) ([]Summary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT summary_id, conversation_id, kind, depth, content, token_count,
			earliest_at, latest_at, descendant_count, descendant_token_count,
			source_message_token_count, model, created_at
		 FROM summaries WHERE conversation_id = ? ORDER BY created_at`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanSummaries(rows)
}

// GetSummaryChildren retrieves child summary IDs (summaries that list this summary as parent).
func (s *Store) GetSummaryChildren(ctx context.Context, summaryID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT summary_id FROM summary_parents WHERE parent_summary_id = ?",
		summaryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// GetSummaryParents retrieves parent summaries (full objects) for a summary.
func (s *Store) GetSummaryParents(ctx context.Context, summaryID string) ([]Summary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.summary_id, s.conversation_id, s.kind, s.depth, s.content, s.token_count,
			s.earliest_at, s.latest_at, s.descendant_count, s.descendant_token_count,
			s.source_message_token_count, s.model, s.created_at
		 FROM summary_parents sp
		 JOIN summaries s ON s.summary_id = sp.parent_summary_id
		 WHERE sp.summary_id = ?`,
		summaryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanSummaries(rows)
}

// LinkSummaryToMessages links a leaf summary to its source messages.
func (s *Store) LinkSummaryToMessages(ctx context.Context, summaryID string, messageIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, msgID := range messageIDs {
		_, err = tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO summary_messages (summary_id, message_id, ordinal) VALUES (?, ?, ?)",
			summaryID, msgID, i,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetSummarySourceMessages retrieves source messages for a summary.
func (s *Store) GetSummarySourceMessages(ctx context.Context, summaryID string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.message_id, m.conversation_id, m.role, m.content, m.reasoning_content, m.token_count, m.created_at
		 FROM summary_messages sm
		 JOIN messages m ON m.message_id = sm.message_id
		 WHERE sm.summary_id = ?
		 ORDER BY sm.ordinal`,
		summaryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		var createdAt string
		if err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&msg.ReasoningContent,
			&msg.TokenCount,
			&createdAt,
		); err != nil {
			return nil, err
		}
		msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

// GetRootSummaries retrieves root summaries (not children of any other summary).
func (s *Store) GetRootSummaries(ctx context.Context, convID int64) ([]Summary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.summary_id, s.conversation_id, s.kind, s.depth, s.content, s.token_count,
			s.earliest_at, s.latest_at, s.descendant_count, s.descendant_token_count,
			s.source_message_token_count, s.model, s.created_at
		 FROM summaries s
		 WHERE s.conversation_id = ?
		 AND s.summary_id NOT IN (SELECT sp.parent_summary_id FROM summary_parents sp)
		 ORDER BY s.created_at`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanSummaries(rows)
}

// --- Context Item Operations ---

// GetContextItems retrieves context items for a conversation, ordered by ordinal.
func (s *Store) GetContextItems(ctx context.Context, convID int64) ([]ContextItem, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT ordinal, item_type, summary_id, message_id, token_count, created_at FROM context_items WHERE conversation_id = ? ORDER BY ordinal",
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ContextItem
	for rows.Next() {
		var item ContextItem
		var summaryID sql.NullString
		var messageID sql.NullInt64
		var createdAt sql.NullString
		if err := rows.Scan(
			&item.Ordinal,
			&item.ItemType,
			&summaryID,
			&messageID,
			&item.TokenCount,
			&createdAt,
		); err != nil {
			return nil, err
		}
		item.ConversationID = convID
		if summaryID.Valid {
			item.SummaryID = summaryID.String
		}
		if messageID.Valid {
			item.MessageID = messageID.Int64
		}
		if createdAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", createdAt.String)
			item.CreatedAt = t
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertContextItems replaces all context items for a conversation.
func (s *Store) UpsertContextItems(ctx context.Context, convID int64, items []ContextItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "DELETE FROM context_items WHERE conversation_id = ?", convID)
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO context_items (conversation_id, ordinal, item_type, summary_id, message_id, token_count)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			convID, item.Ordinal, item.ItemType,
			nullString(item.SummaryID), nullInt64(item.MessageID),
			item.TokenCount,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ClearContextItems removes all context items for a conversation.
func (s *Store) ClearContextItems(ctx context.Context, convID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM context_items WHERE conversation_id = ?", convID)
	return err
}

// DeleteMessagesAfterID deletes all messages with ID > afterID for a conversation.
// Also clears related context_items, message_parts, summary_messages, and FTS entries.
// Uses transaction to ensure atomicity of the delete cascade.
func (s *Store) DeleteMessagesAfterID(ctx context.Context, convID int64, afterID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get message IDs to delete for cleaning up related tables
	rows, err := tx.QueryContext(ctx,
		"SELECT message_id FROM messages WHERE conversation_id = ? AND message_id > ?", convID, afterID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var msgIDs []int64
	for rows.Next() {
		var id int64
		if scanErr := rows.Scan(&id); scanErr != nil {
			return scanErr
		}
		msgIDs = append(msgIDs, id)
	}
	if rows.Err() != nil {
		return rows.Err()
	}

	// Delete context_items referencing these messages
	for _, msgID := range msgIDs {
		if _, err := tx.ExecContext(ctx, "DELETE FROM context_items WHERE message_id = ?", msgID); err != nil {
			return err
		}
	}

	// Delete from message_parts and summary_messages
	// Note: messages_fts is handled automatically by trigger, no manual delete needed
	for _, msgID := range msgIDs {
		if _, err := tx.ExecContext(ctx, "DELETE FROM message_parts WHERE message_id = ?", msgID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM summary_messages WHERE message_id = ?", msgID); err != nil {
			return err
		}
	}

	// Delete messages
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM messages WHERE conversation_id = ? AND message_id > ?", convID, afterID); err != nil {
		return err
	}

	return tx.Commit()
}

// ClearConversation removes all data for a conversation from all tables.
// Deletes context_items, summary_messages, summary_parents (via subquery), summaries,
// message_parts, and messages. FTS entries are handled automatically by triggers.
// Uses a transaction for atomicity.
func (s *Store) ClearConversation(ctx context.Context, convID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete in child→parent order. FTS tables (messages_fts, summaries_fts) are
	// kept in sync by DELETE triggers, so we just delete from the parent tables.

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM context_items WHERE conversation_id = ?", convID); err != nil {
		return fmt.Errorf("context_items: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM summary_messages WHERE summary_id IN (
			SELECT summary_id FROM summaries WHERE conversation_id = ?
		)`, convID); err != nil {
		return fmt.Errorf("summary_messages: %w", err)
	}
	// Note: summary_parents has no convID column; delete via subquery on summaries
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM summary_parents WHERE summary_id IN (
			SELECT summary_id FROM summaries WHERE conversation_id = ?
		) OR parent_summary_id IN (
			SELECT summary_id FROM summaries WHERE conversation_id = ?
		)`, convID, convID); err != nil {
		return fmt.Errorf("summary_parents: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM summaries WHERE conversation_id = ?", convID); err != nil {
		return fmt.Errorf("summaries: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM message_parts WHERE message_id IN (
			SELECT message_id FROM messages WHERE conversation_id = ?
		)`, convID); err != nil {
		return fmt.Errorf("message_parts: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM messages WHERE conversation_id = ?", convID); err != nil {
		return fmt.Errorf("messages: %w", err)
	}

	return tx.Commit()
}

// AppendContextMessage appends a single message to context_items at next ordinal.
func (s *Store) AppendContextMessage(ctx context.Context, convID int64, messageID int64) error {
	return s.appendContextItems(ctx, convID, []ContextItem{
		{ItemType: "message", MessageID: messageID},
	})
}

// AppendContextMessages bulk-appends messages to context_items.
func (s *Store) AppendContextMessages(ctx context.Context, convID int64, messageIDs []int64) error {
	items := make([]ContextItem, len(messageIDs))
	for i, id := range messageIDs {
		items[i] = ContextItem{ItemType: "message", MessageID: id}
	}
	return s.appendContextItems(ctx, convID, items)
}

// AppendContextSummary appends a summary to context_items at next ordinal.
func (s *Store) AppendContextSummary(ctx context.Context, convID int64, summaryID string) error {
	return s.appendContextItems(ctx, convID, []ContextItem{
		{ItemType: "summary", SummaryID: summaryID},
	})
}

func (s *Store) appendContextItems(ctx context.Context, convID int64, items []ContextItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	maxOrd, err := s.GetMaxOrdinalTx(ctx, tx, convID)
	if err != nil {
		return err
	}

	ordinal := maxOrd + OrdinalStep
	for _, item := range items {
		item.ConversationID = convID
		item.Ordinal = ordinal

		// Resolve token count if not set
		tokenCount := item.TokenCount
		if tokenCount == 0 {
			tokenCount = s.resolveItemTokenCountTx(ctx, tx, item)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO context_items (conversation_id, ordinal, item_type, summary_id, message_id, token_count)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			convID, ordinal, item.ItemType,
			nullString(item.SummaryID), nullInt64(item.MessageID),
			tokenCount,
		)
		if err != nil {
			return err
		}
		ordinal += OrdinalStep
	}
	return tx.Commit()
}

// resolveItemTokenCountTx looks up token count within a transaction.
func (s *Store) resolveItemTokenCountTx(ctx context.Context, tx *sql.Tx, item ContextItem) int {
	if item.ItemType == "message" && item.MessageID > 0 {
		var tc int
		err := tx.QueryRowContext(ctx,
			"SELECT token_count FROM messages WHERE message_id = ?", item.MessageID,
		).Scan(&tc)
		if err == nil {
			return tc
		}
	}
	if item.ItemType == "summary" && item.SummaryID != "" {
		var tc int
		err := tx.QueryRowContext(ctx,
			"SELECT token_count FROM summaries WHERE summary_id = ?", item.SummaryID,
		).Scan(&tc)
		if err == nil {
			return tc
		}
	}
	return 0
}

// ReplaceContextRangeWithSummary atomically replaces a range of context items with a summary.
// If ordinal gap is exhausted, triggers resequencing (spec lines 1204-1209).
func (s *Store) ReplaceContextRangeWithSummary(
	ctx context.Context,
	convID int64,
	startOrdinal, endOrdinal int,
	summaryID string,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete the range
	_, err = tx.ExecContext(ctx,
		"DELETE FROM context_items WHERE conversation_id = ? AND ordinal >= ? AND ordinal <= ?",
		convID, startOrdinal, endOrdinal,
	)
	if err != nil {
		return err
	}

	// Insert summary at midpoint of replaced range
	midpoint := (startOrdinal + endOrdinal) / 2

	// Check if midpoint conflicts with existing ordinal
	var conflict bool
	var existingOrd int
	err = tx.QueryRowContext(ctx,
		"SELECT ordinal FROM context_items WHERE conversation_id = ? AND ordinal = ?",
		convID, midpoint,
	).Scan(&existingOrd)
	if err == nil {
		conflict = true
	}

	if conflict {
		// Gap exhausted, need resequence (spec lines 1204-1209)
		err = s.resequenceContextItemsTx(ctx, tx, convID, summaryID)
		if err != nil {
			return fmt.Errorf("resequence: %w", err)
		}
	} else {
		// Normal insert at midpoint with token_count from summary
		_, err = tx.ExecContext(ctx,
			`INSERT INTO context_items (conversation_id, ordinal, item_type, summary_id, token_count)
			 SELECT ?, ?, 'summary', ?, token_count FROM summaries WHERE summary_id = ?`,
			convID, midpoint, summaryID, summaryID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ReplaceContextItemsWithSummary replaces specific context items (by summary_id) with a new summary.
// Use this when candidates are not contiguous in ordinal space to avoid deleting non-candidate items.
func (s *Store) ReplaceContextItemsWithSummary(
	ctx context.Context,
	convID int64,
	summaryIDs []string,
	newSummaryID string,
) error {
	if len(summaryIDs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find the ordinals of items to delete and calculate midpoint
	placeholders := make([]string, len(summaryIDs))
	args := make([]any, len(summaryIDs)+1)
	args[0] = convID
	for i, sid := range summaryIDs {
		placeholders[i] = "?"
		args[i+1] = sid
	}

	query := fmt.Sprintf(
		"SELECT ordinal FROM context_items WHERE conversation_id = ? AND summary_id IN (%s) ORDER BY ordinal",
		strings.Join(placeholders, ","),
	)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ordinals []int
	for rows.Next() {
		var ord int
		if scanErr := rows.Scan(&ord); scanErr != nil {
			return scanErr
		}
		ordinals = append(ordinals, ord)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	if len(ordinals) == 0 {
		return nil
	}

	midpoint := (ordinals[0] + ordinals[len(ordinals)-1]) / 2

	// Delete the specific items by summary_id
	deleteQuery := fmt.Sprintf(
		"DELETE FROM context_items WHERE conversation_id = ? AND summary_id IN (%s)",
		strings.Join(placeholders, ","),
	)
	_, err = tx.ExecContext(ctx, deleteQuery, args...)
	if err != nil {
		return err
	}

	// Check if midpoint conflicts with existing ordinal
	var conflict bool
	var existingOrd int
	err = tx.QueryRowContext(ctx,
		"SELECT ordinal FROM context_items WHERE conversation_id = ? AND ordinal = ?",
		convID, midpoint,
	).Scan(&existingOrd)
	if err == nil {
		conflict = true
	}

	if conflict {
		// Gap exhausted, need resequence
		err = s.resequenceContextItemsTx(ctx, tx, convID, newSummaryID)
		if err != nil {
			return fmt.Errorf("resequence: %w", err)
		}
	} else {
		// Normal insert at midpoint
		_, err = tx.ExecContext(ctx,
			`INSERT INTO context_items (conversation_id, ordinal, item_type, summary_id, token_count)
			 SELECT ?, ?, 'summary', ?, token_count FROM summaries WHERE summary_id = ?`,
			convID, midpoint, newSummaryID, newSummaryID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// resequenceContextItemsTx renumbers context_items with fresh OrdinalStep gaps.
// Uses temp negative ordinals to avoid PRIMARY KEY constraint violations (spec lines 1240-1247).
func (s *Store) resequenceContextItemsTx(ctx context.Context, tx *sql.Tx, convID int64, newSummaryID string) error {
	// Get all remaining items sorted by current ordinal
	rows, err := tx.QueryContext(
		ctx,
		"SELECT ordinal, item_type, summary_id, message_id, token_count FROM context_items WHERE conversation_id = ? ORDER BY ordinal",
		convID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		ordinal    int
		itemType   string
		summaryID  string
		messageID  int64
		tokenCount int
	}
	var items []item
	for rows.Next() {
		var i item
		var sid sql.NullString
		var mid sql.NullInt64
		var scanErr error
		if scanErr = rows.Scan(&i.ordinal, &i.itemType, &sid, &mid, &i.tokenCount); scanErr != nil {
			return scanErr
		}
		if sid.Valid {
			i.summaryID = sid.String
		}
		if mid.Valid {
			i.messageID = mid.Int64
		}
		items = append(items, i)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return rowsErr
	}

	// Step 1: Move all items to temp negative ordinals
	tempOrd := -1
	for _, i := range items {
		_, execErr := tx.ExecContext(ctx,
			"UPDATE context_items SET ordinal = ? WHERE conversation_id = ? AND ordinal = ?",
			tempOrd, convID, i.ordinal,
		)
		if execErr != nil {
			return execErr
		}
		tempOrd--
	}

	// Step 2: Insert new summary at the end with positive ordinal
	// Include token_count from summaries table
	newOrd := (len(items) + 1) * OrdinalStep
	_, err = tx.ExecContext(ctx,
		`INSERT INTO context_items (conversation_id, ordinal, item_type, summary_id, token_count)
		 SELECT ?, ?, 'summary', ?, token_count FROM summaries WHERE summary_id = ?`,
		convID, newOrd, newSummaryID, newSummaryID,
	)
	if err != nil {
		return err
	}

	// Step 3: Update each temp item to its final positive ordinal
	// Use specific temp ordinal matching (not ordinal < 0) to avoid updating all items
	finalOrd := OrdinalStep
	tempOrd = -1 // Reset to first temp ordinal (already declared in Step 1)
	for range items {
		_, execErr := tx.ExecContext(ctx,
			"UPDATE context_items SET ordinal = ? WHERE conversation_id = ? AND ordinal = ?",
			finalOrd, convID, tempOrd,
		)
		if execErr != nil {
			return execErr
		}
		finalOrd += OrdinalStep
		tempOrd--
	}

	return nil
}

// GetContextTokenCount returns total token count for all items in context.
func (s *Store) GetContextTokenCount(ctx context.Context, convID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(token_count), 0) FROM context_items WHERE conversation_id = ?",
		convID,
	).Scan(&count)
	return count, err
}

// GetMaxOrdinal returns the highest ordinal in context_items for a conversation.
func (s *Store) GetMaxOrdinal(ctx context.Context, convID int64) (int, error) {
	var maxOrd sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		"SELECT MAX(ordinal) FROM context_items WHERE conversation_id = ?",
		convID,
	).Scan(&maxOrd)
	if err != nil {
		return 0, err
	}
	if !maxOrd.Valid {
		return 0, nil
	}
	return int(maxOrd.Int64), nil
}

// GetMaxOrdinalTx returns the highest ordinal within a transaction.
func (s *Store) GetMaxOrdinalTx(ctx context.Context, tx *sql.Tx, convID int64) (int, error) {
	var maxOrd sql.NullInt64
	err := tx.QueryRowContext(ctx,
		"SELECT MAX(ordinal) FROM context_items WHERE conversation_id = ?",
		convID,
	).Scan(&maxOrd)
	if err != nil {
		return 0, err
	}
	if !maxOrd.Valid {
		return 0, nil
	}
	return int(maxOrd.Int64), nil
}

// GetDistinctDepthsInContext returns distinct depth levels of summaries currently in context.
// maxOrdinalExclusive filters out summaries with ordinal >= this value (0 = no filter).
func (s *Store) GetDistinctDepthsInContext(ctx context.Context, convID int64, maxOrdinalExclusive int) ([]int, error) {
	query := `SELECT DISTINCT s.depth
		FROM context_items ci
		JOIN summaries s ON s.summary_id = ci.summary_id
		WHERE ci.conversation_id = ? AND ci.item_type = 'summary'`
	args := []any{convID}

	if maxOrdinalExclusive > 0 {
		query += " AND ci.ordinal < ?"
		args = append(args, maxOrdinalExclusive)
	}

	query += " ORDER BY s.depth"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get distinct depths: %w", err)
	}
	defer rows.Close()

	var depths []int
	for rows.Next() {
		var d int
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		depths = append(depths, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return depths, nil
}

// GetSummarySubtree returns all summaries in the subtree rooted at summaryID,
// including summaryID itself. Uses a recursive CTE to traverse the DAG.
func (s *Store) GetSummarySubtree(ctx context.Context, summaryID string) ([]SummarySubtreeNode, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT summary_id, 0 AS depth_from_root
			FROM summaries
			WHERE summary_id = ?
			UNION ALL
			SELECT sp.parent_summary_id, st.depth_from_root + 1
			FROM summary_parents sp
			JOIN subtree st ON sp.summary_id = st.summary_id
		)
		SELECT summary_id, depth_from_root FROM subtree`,
		summaryID,
	)
	if err != nil {
		return nil, fmt.Errorf("get summary subtree: %w", err)
	}
	defer rows.Close()

	var nodes []SummarySubtreeNode
	for rows.Next() {
		var n SummarySubtreeNode
		if err := rows.Scan(&n.SummaryID, &n.DepthFromRoot); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

// --- Search Operations ---

// SearchSummaries performs full-text search on summaries.
func (s *Store) SearchSummaries(ctx context.Context, input SearchInput) ([]SearchResult, error) {
	// "like" → LIKE search, anything else (including "full_text" or empty) → FTS5
	if input.Mode == "like" {
		return s.searchSummariesLike(ctx, input)
	}
	return s.searchSummariesFTS(ctx, input)
}

func (s *Store) searchSummariesFTS(ctx context.Context, input SearchInput) ([]SearchResult, error) {
	sanitized := SanitizeFTS5Query(input.Pattern)
	if sanitized == "" {
		return nil, nil
	}

	// Build WHERE clause for filters (used in both count and data queries)
	whereClauses := []string{"summaries_fts MATCH ?"}
	args := []any{sanitized}

	if input.ConversationID > 0 && !input.AllConversations {
		whereClauses = append(whereClauses, "s.conversation_id = ?")
		args = append(args, input.ConversationID)
	}

	if input.Since != nil {
		whereClauses = append(whereClauses, "s.created_at >= ?")
		args = append(args, input.Since.Format("2006-01-02 15:04:05"))
	}
	if input.Before != nil {
		whereClauses = append(whereClauses, "s.created_at < ?")
		args = append(args, input.Before.Format("2006-01-02 15:04:05"))
	}

	whereStr := strings.Join(whereClauses, " AND ")

	// First, get total count (bm25 conflicts with window functions in FTS5)
	countQuery := `SELECT COUNT(*) FROM summaries_fts fts
		JOIN summaries s ON s.summary_id = fts.summary_id
		WHERE ` + whereStr
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Then, get actual results with bm25 ranking
	dataQuery := `SELECT s.summary_id, s.conversation_id, s.kind, s.content, s.created_at, bm25(summaries_fts) as rank
		FROM summaries_fts fts
		JOIN summaries s ON s.summary_id = fts.summary_id
		WHERE ` + whereStr + ` ORDER BY rank`

	dataArgs := append([]any{}, args...) // copy args
	if input.Limit > 0 {
		dataQuery += " LIMIT ?"
		dataArgs = append(dataArgs, input.Limit)
	}

	rows, err := s.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := s.scanSearchResults(rows, true)
	if err != nil {
		return nil, err
	}

	// Set total count on all results
	for i := range results {
		results[i].TotalCount = totalCount
	}
	return results, nil
}

// buildLikeQuery appends conversation/time filters and limit to a LIKE query.
// Note: role filtering is NOT applied here since summaries don't have role column.
// Use buildMessagesLikeQuery for message searches that need role filtering.
func buildLikeQuery(query string, args []any, input SearchInput) (string, []any) {
	if input.ConversationID > 0 && !input.AllConversations {
		query += " AND conversation_id = ?"
		args = append(args, input.ConversationID)
	}
	if input.Since != nil {
		query += " AND created_at >= ?"
		args = append(args, input.Since.Format("2006-01-02 15:04:05"))
	}
	if input.Before != nil {
		query += " AND created_at < ?"
		args = append(args, input.Before.Format("2006-01-02 15:04:05"))
	}
	// Order by newest first for LIKE mode
	query += " ORDER BY created_at DESC"
	if input.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, input.Limit)
	}
	return query, args
}

// buildMessagesLikeQuery is like buildLikeQuery but adds role filtering for messages.
func buildMessagesLikeQuery(query string, args []any, input SearchInput) (string, []any) {
	if input.Role != "" {
		query += " AND role = ?"
		args = append(args, input.Role)
	}
	return buildLikeQuery(query, args, input)
}

func (s *Store) searchSummariesLike(ctx context.Context, input SearchInput) ([]SearchResult, error) {
	query := `SELECT summary_id, conversation_id, kind, content, created_at, COUNT(*) OVER() as total_count
		FROM summaries WHERE content LIKE ?`
	args := []any{"%" + input.Pattern + "%"}
	query, args = buildLikeQuery(query, args, input)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanSearchResults(rows, false)
}

func (s *Store) scanSearchResults(rows *sql.Rows, withRank bool) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var createdAt string
		var kind string
		if withRank {
			// FTS5 mode: no TotalCount in query (set by caller after COUNT)
			if err := rows.Scan(&r.SummaryID, &r.ConversationID, &kind, &r.Content, &createdAt, &r.Rank); err != nil {
				return nil, err
			}
		} else {
			// LIKE mode: TotalCount from window function
			if err := rows.Scan(&r.SummaryID, &r.ConversationID, &kind,
				&r.Content, &createdAt, &r.TotalCount); err != nil {
				return nil, err
			}
		}
		r.Kind = SummaryKind(kind)
		r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		results = append(results, r)
	}
	return results, nil
}

// SearchMessages performs full-text or regex search on messages.
func (s *Store) SearchMessages(ctx context.Context, input SearchInput) ([]SearchResult, error) {
	// Try FTS5 first for full-text mode
	if input.Mode == "" || input.Mode == "full_text" {
		results, err := s.searchMessagesFTS(ctx, input)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		// Fall through to LIKE
	}

	return s.searchMessagesLike(ctx, input)
}

func (s *Store) searchMessagesFTS(ctx context.Context, input SearchInput) ([]SearchResult, error) {
	sanitized := SanitizeFTS5Query(input.Pattern)
	if sanitized == "" {
		return nil, nil
	}

	// Build WHERE clause for filters (used in both count and data queries)
	whereClauses := []string{"messages_fts MATCH ?"}
	args := []any{sanitized}

	if input.ConversationID > 0 && !input.AllConversations {
		whereClauses = append(whereClauses, "m.conversation_id = ?")
		args = append(args, input.ConversationID)
	}

	if input.Role != "" {
		whereClauses = append(whereClauses, "m.role = ?")
		args = append(args, input.Role)
	}

	if input.Since != nil {
		whereClauses = append(whereClauses, "m.created_at >= ?")
		args = append(args, input.Since.Format("2006-01-02 15:04:05"))
	}
	if input.Before != nil {
		whereClauses = append(whereClauses, "m.created_at < ?")
		args = append(args, input.Before.Format("2006-01-02 15:04:05"))
	}

	whereStr := strings.Join(whereClauses, " AND ")

	// First, get total count (bm25 conflicts with window functions in FTS5)
	countQuery := `SELECT COUNT(*) FROM messages_fts f
		JOIN messages m ON f.message_id = m.message_id
		WHERE ` + whereStr
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Then, get actual results with bm25 ranking
	dataQuery := `SELECT m.message_id, m.conversation_id, m.role, m.content, m.created_at, bm25(messages_fts) as rank
		FROM messages_fts f
		JOIN messages m ON f.message_id = m.message_id
		WHERE ` + whereStr + ` ORDER BY rank`

	dataArgs := append([]any{}, args...) // copy args
	if input.Limit > 0 {
		dataQuery += " LIMIT ?"
		dataArgs = append(dataArgs, input.Limit)
	}

	rows, err := s.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := s.scanMessageSearchResults(rows, true)
	if err != nil {
		return nil, err
	}

	// Set total count on all results
	for i := range results {
		results[i].TotalCount = totalCount
	}
	return results, nil
}

func (s *Store) searchMessagesLike(ctx context.Context, input SearchInput) ([]SearchResult, error) {
	query := `SELECT message_id, conversation_id, role, content, created_at, COUNT(*) OVER() as total_count
		FROM messages WHERE content LIKE ?`
	args := []any{"%" + input.Pattern + "%"}
	query, args = buildMessagesLikeQuery(query, args, input)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMessageSearchResults(rows, false)
}

func (s *Store) scanMessageSearchResults(rows *sql.Rows, withRank bool) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var createdAt string
		var content string
		if withRank {
			// FTS5 mode: no TotalCount in query (set by caller after COUNT)
			if err := rows.Scan(&r.MessageID, &r.ConversationID, &r.Role, &content, &createdAt, &r.Rank); err != nil {
				return nil, err
			}
		} else {
			// LIKE mode: TotalCount from window function
			if err := rows.Scan(&r.MessageID, &r.ConversationID, &r.Role, &content,
				&createdAt, &r.TotalCount); err != nil {
				return nil, err
			}
		}
		r.Snippet = content
		r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// --- Helpers ---

func (s *Store) scanSummary(ctx context.Context, where string, args ...any) (*Summary, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT summary_id, conversation_id, kind, depth, content, token_count,
			earliest_at, latest_at, descendant_count, descendant_token_count,
			source_message_token_count, model, created_at
		 FROM summaries `+where, args...,
	)
	var sum Summary
	var kind, createdAt string
	var earliestAt, latestAt sql.NullString
	err := row.Scan(
		&sum.SummaryID, &sum.ConversationID, &kind, &sum.Depth, &sum.Content, &sum.TokenCount,
		&earliestAt, &latestAt, &sum.DescendantCount, &sum.DescendantTokenCount,
		&sum.SourceMessageTokenCount, &sum.Model, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("summary not found")
	}
	if err != nil {
		return nil, err
	}
	sum.Kind = SummaryKind(kind)
	sum.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	if earliestAt.Valid {
		t, _ := time.Parse(time.RFC3339, earliestAt.String)
		sum.EarliestAt = &t
	}
	if latestAt.Valid {
		t, _ := time.Parse(time.RFC3339, latestAt.String)
		sum.LatestAt = &t
	}
	return &sum, nil
}

func (s *Store) scanSummaries(rows *sql.Rows) ([]Summary, error) {
	var summaries []Summary
	for rows.Next() {
		var sum Summary
		var kind, createdAt string
		var earliestAt, latestAt sql.NullString
		err := rows.Scan(
			&sum.SummaryID, &sum.ConversationID, &kind, &sum.Depth, &sum.Content, &sum.TokenCount,
			&earliestAt, &latestAt, &sum.DescendantCount, &sum.DescendantTokenCount,
			&sum.SourceMessageTokenCount, &sum.Model, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		sum.Kind = SummaryKind(kind)
		sum.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		if earliestAt.Valid {
			t, _ := time.Parse(time.RFC3339, earliestAt.String)
			sum.EarliestAt = &t
		}
		if latestAt.Valid {
			t, _ := time.Parse(time.RFC3339, latestAt.String)
			sum.LatestAt = &t
		}
		summaries = append(summaries, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return summaries, nil
}

func generateSummaryID(content string, t time.Time) string {
	return fmt.Sprintf("sum_%x", t.UnixNano())
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") ||
		contains(err.Error(), "constraint failed"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt64(n int64) sql.NullInt64 {
	return sql.NullInt64{Int64: n, Valid: n != 0}
}
