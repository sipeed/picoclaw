package security

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// ApprovalResult carries the user's decision on a security approval request.
type ApprovalResult struct {
	Approved bool
	Reason   string
}

// requestApproval sends an approval notification via IM and blocks until the
// user responds with an approval/denial keyword or the timeout expires.
func (pe *PolicyEngine) requestApproval(ctx context.Context, v Violation, channel, chatID string) error {
	resultCh := make(chan ApprovalResult, 1)

	// Register an interceptor to capture the approval reply from the same chat
	removeInterceptor := pe.bus.AddInterceptor(func(msg bus.InboundMessage) bool {
		if msg.Channel != channel || msg.ChatID != chatID {
			return false
		}
		content := strings.TrimSpace(msg.Content)
		lower := strings.ToLower(content)
		if isApproveKeyword(lower) || isApproveKeywordCJK(content) {
			resultCh <- ApprovalResult{Approved: true}
			return true
		}
		if isDenyKeyword(lower) || isDenyKeywordCJK(content) {
			resultCh <- ApprovalResult{Approved: false, Reason: "denied by user"}
			return true
		}
		return false // not an approval keyword, pass through
	})
	defer removeInterceptor()

	// Send approval request notification to the user via IM
	pe.bus.PublishOutbound(bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: formatApprovalMessage(v, pe.config.ApprovalTimeout),
	})

	timeout := time.Duration(pe.config.ApprovalTimeout) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}

	select {
	case result := <-resultCh:
		if result.Approved {
			return nil
		}
		return fmt.Errorf("denied by user: %s", result.Reason)
	case <-time.After(timeout):
		return fmt.Errorf("approval timed out after %v", timeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// formatApprovalMessage builds a human-readable approval notification.
func formatApprovalMessage(v Violation, timeoutSec int) string {
	var b strings.Builder
	b.WriteString("⚠️ Security Approval Required / 安全审批请求\n\n")
	b.WriteString(fmt.Sprintf("Category: %s\n", v.Category))
	if v.Tool != "" {
		b.WriteString(fmt.Sprintf("Tool: %s\n", v.Tool))
	}
	if v.Action != "" {
		b.WriteString(fmt.Sprintf("Action: %s\n", v.Action))
	}
	b.WriteString(fmt.Sprintf("Reason: %s\n", v.Reason))
	if v.RuleName != "" {
		b.WriteString(fmt.Sprintf("Rule: %s\n", v.RuleName))
	}
	b.WriteString(fmt.Sprintf("\nReply \"approve\" to allow or \"deny\" to block.\n"))
	b.WriteString(fmt.Sprintf("回复 \"批准\" 允许执行，回复 \"拒绝\" 阻止执行。\n"))
	if timeoutSec > 0 {
		b.WriteString(fmt.Sprintf("Auto-deny in %d seconds.\n", timeoutSec))
	}
	return b.String()
}

// isApproveKeyword checks lowercase ASCII approval keywords.
func isApproveKeyword(lower string) bool {
	switch lower {
	case "approve", "yes", "allow", "ok", "y":
		return true
	}
	return false
}

// isApproveKeywordCJK checks CJK approval keywords (case-sensitive).
func isApproveKeywordCJK(s string) bool {
	switch s {
	case "批准", "允许", "通过", "是", "承認", "許可", "はい":
		return true
	}
	return false
}

// isDenyKeyword checks lowercase ASCII denial keywords.
func isDenyKeyword(lower string) bool {
	switch lower {
	case "deny", "no", "reject", "block", "n":
		return true
	}
	return false
}

// isDenyKeywordCJK checks CJK denial keywords (case-sensitive).
func isDenyKeywordCJK(s string) bool {
	switch s {
	case "拒绝", "否决", "不", "拒否", "いいえ":
		return true
	}
	return false
}
