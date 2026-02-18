package security

import (
	"testing"
)

func TestIsApproveKeyword(t *testing.T) {
	approve := []string{"approve", "yes", "allow", "ok", "y"}
	for _, w := range approve {
		if !isApproveKeyword(w) {
			t.Errorf("expected %q to be an approve keyword", w)
		}
	}
	notApprove := []string{"deny", "no", "hello", ""}
	for _, w := range notApprove {
		if isApproveKeyword(w) {
			t.Errorf("expected %q to NOT be an approve keyword", w)
		}
	}
}

func TestIsApproveKeywordCJK(t *testing.T) {
	approve := []string{"批准", "允许", "通过", "是", "承認", "許可", "はい"}
	for _, w := range approve {
		if !isApproveKeywordCJK(w) {
			t.Errorf("expected %q to be a CJK approve keyword", w)
		}
	}
	notApprove := []string{"拒绝", "否决", "hello", ""}
	for _, w := range notApprove {
		if isApproveKeywordCJK(w) {
			t.Errorf("expected %q to NOT be a CJK approve keyword", w)
		}
	}
}

func TestIsDenyKeyword(t *testing.T) {
	deny := []string{"deny", "no", "reject", "block", "n"}
	for _, w := range deny {
		if !isDenyKeyword(w) {
			t.Errorf("expected %q to be a deny keyword", w)
		}
	}
	notDeny := []string{"approve", "yes", "hello", ""}
	for _, w := range notDeny {
		if isDenyKeyword(w) {
			t.Errorf("expected %q to NOT be a deny keyword", w)
		}
	}
}

func TestIsDenyKeywordCJK(t *testing.T) {
	deny := []string{"拒绝", "否决", "不", "拒否", "いいえ"}
	for _, w := range deny {
		if !isDenyKeywordCJK(w) {
			t.Errorf("expected %q to be a CJK deny keyword", w)
		}
	}
	notDeny := []string{"批准", "允许", "hello", ""}
	for _, w := range notDeny {
		if isDenyKeywordCJK(w) {
			t.Errorf("expected %q to NOT be a CJK deny keyword", w)
		}
	}
}

func TestFormatApprovalMessage(t *testing.T) {
	msg := formatApprovalMessage(Violation{
		Category: "exec_guard",
		Tool:     "exec",
		Action:   "rm -rf /tmp",
		Reason:   "dangerous pattern detected",
		RuleName: `\brm\s+-[rf]`,
	}, 300)

	// Check essential fields are present
	checks := []string{
		"Approval Required",
		"exec_guard",
		"exec",
		"rm -rf /tmp",
		"dangerous pattern",
		"300 seconds",
		"approve",
		"deny",
		"批准",
		"拒绝",
	}
	for _, c := range checks {
		if !containsSubstring(msg, c) {
			t.Errorf("approval message missing %q:\n%s", c, msg)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
