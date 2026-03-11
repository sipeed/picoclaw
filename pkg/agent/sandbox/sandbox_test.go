package sandbox

import (
	"context"
	"testing"
)

func TestContextHelpers(t *testing.T) {
	// Test SessionKey
	ctx := context.Background()
	if got := SessionKeyFromContext(ctx); got != "" {
		t.Fatalf("expected empty session key, got %q", got)
	}

	ctx = WithSessionKey(ctx, "session-123")
	if got := SessionKeyFromContext(ctx); got != "session-123" {
		t.Fatalf("expected session-123, got %q", got)
	}

	// Test nil contexts
	if got := SessionKeyFromContext(nil); got != "" { //nolint:staticcheck
		t.Fatalf("SessionKeyFromContext(nil) = %q, want empty", got)
	}
	if got := FromContext(nil); got != nil { //nolint:staticcheck
		t.Fatalf("FromContext(nil) = %v, want nil", got)
	}
	if got := managerFromContext(nil); got != nil { //nolint:staticcheck
		t.Fatalf("managerFromContext(nil) = %v, want nil", got)
	}

	// Test Sandbox context
	mockSb := &unavailableSandboxManager{}
	ctx = WithSandbox(context.Background(), mockSb)
	if got := FromContext(ctx); got != mockSb {
		t.Fatalf("expected to retrieve mock sandbox from context")
	}

	// Test Manager context resolving
	mockMgr := NewUnavailableSandboxManager(nil)
	ctx = WithManager(context.Background(), mockMgr)

	if got := managerFromContext(ctx); got != mockMgr {
		t.Fatalf("expected to retrieve mock manager from context")
	}

	// FromContext with Manager only should attempt to Resolve (which returns error/nil here)
	if got := FromContext(ctx); got != nil {
		t.Fatalf("expected nil from FromContext when Resolve fails, got %v", got)
	}
}
