package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/miniapp"
)

// mockDevTargetManager implements miniapp.DevTargetManager for testing.
type mockDevTargetManager struct {
	targets  map[string]*miniapp.DevTarget
	nextID   int
	activeID string
	active   string // active target URL
	regErr   error
}

func newMockManager() *mockDevTargetManager {
	return &mockDevTargetManager{targets: make(map[string]*miniapp.DevTarget)}
}

func (m *mockDevTargetManager) RegisterDevTarget(name, target string) (string, error) {
	if m.regErr != nil {
		return "", m.regErr
	}
	m.nextID++
	id := fmt.Sprintf("%d", m.nextID)
	m.targets[id] = &miniapp.DevTarget{ID: id, Name: name, Target: target}
	return id, nil
}

func (m *mockDevTargetManager) UnregisterDevTarget(id string) error {
	if _, ok := m.targets[id]; !ok {
		return fmt.Errorf("target %q not found", id)
	}
	delete(m.targets, id)
	if m.activeID == id {
		m.activeID = ""
		m.active = ""
	}
	return nil
}

func (m *mockDevTargetManager) ActivateDevTarget(id string) error {
	dt, ok := m.targets[id]
	if !ok {
		return fmt.Errorf("target %q not found", id)
	}
	m.activeID = id
	m.active = dt.Target
	return nil
}

func (m *mockDevTargetManager) DeactivateDevTarget() error {
	m.activeID = ""
	m.active = ""
	return nil
}

func (m *mockDevTargetManager) GetDevTarget() string {
	return m.active
}

func (m *mockDevTargetManager) ListDevTargets() []miniapp.DevTarget {
	out := make([]miniapp.DevTarget, 0, len(m.targets))
	for _, dt := range m.targets {
		out = append(out, *dt)
	}
	return out
}

func TestDevPreviewTool_Start(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:3000",
		"name":   "frontend",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if len(mgr.targets) != 1 {
		t.Errorf("expected 1 registered target, got %d", len(mgr.targets))
	}
	if mgr.active != "http://localhost:3000" {
		t.Errorf("expected active target http://localhost:3000, got %q", mgr.active)
	}
	if !strings.Contains(result.ForLLM, "started") {
		t.Errorf("expected result to contain 'started', got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "frontend") {
		t.Errorf("expected result to contain 'frontend', got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_StartAutoName(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:3000",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	// Auto-generated name should be "localhost:3000"
	for _, dt := range mgr.targets {
		if dt.Name != "localhost:3000" {
			t.Errorf("expected auto-name 'localhost:3000', got %q", dt.Name)
		}
	}
}

func TestDevPreviewTool_StartMissingTarget(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
	})

	if !result.IsError {
		t.Error("expected error for missing target")
	}
}

func TestDevPreviewTool_StartError(t *testing.T) {
	mgr := newMockManager()
	mgr.regErr = fmt.Errorf("only localhost")
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://example.com:3000",
	})

	if !result.IsError {
		t.Error("expected error for registration failure")
	}
}

func TestDevPreviewTool_Stop(t *testing.T) {
	mgr := newMockManager()
	mgr.active = "http://localhost:3000"
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "stop",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if mgr.active != "" {
		t.Errorf("expected empty active target after stop, got %q", mgr.active)
	}
}

func TestDevPreviewTool_Unregister(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	// Register a target first
	id, _ := mgr.RegisterDevTarget("frontend", "http://localhost:3000")

	result := tool.Execute(context.Background(), map[string]any{
		"action": "unregister",
		"id":     id,
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if len(mgr.targets) != 0 {
		t.Errorf("expected 0 targets after unregister, got %d", len(mgr.targets))
	}
}

func TestDevPreviewTool_UnregisterMissingID(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "unregister",
	})

	if !result.IsError {
		t.Error("expected error for missing id")
	}
}

func TestDevPreviewTool_UnregisterNotFound(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "unregister",
		"id":     "999",
	})

	if !result.IsError {
		t.Error("expected error for non-existent target")
	}
}

func TestDevPreviewTool_Status(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	mgr.RegisterDevTarget("api", "http://localhost:8080")
	mgr.RegisterDevTarget("frontend", "http://localhost:3000")
	mgr.active = "http://localhost:8080"

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "active") {
		t.Errorf("expected 'active' in result, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "http://localhost:8080") {
		t.Errorf("expected target URL in result, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "api") {
		t.Errorf("expected 'api' in result, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "frontend") {
		t.Errorf("expected 'frontend' in result, got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_StatusInactive(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "not active") {
		t.Errorf("expected 'not active' in result, got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_UnknownAction(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "restart",
	})

	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestDevPreviewTool_MissingAction(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestDevPreviewTool_NameAndSchema(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	if tool.Name() != "dev_preview" {
		t.Errorf("expected name dev_preview, got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}
}

// ── Additional edge-case tests ──

func TestDevPreviewTool_StartMultipleTargets(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	r1 := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:8080",
		"name":   "api",
	})
	r2 := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:3000",
		"name":   "frontend",
	})

	if r1.IsError || r2.IsError {
		t.Fatalf("expected both starts to succeed, got err1=%v err2=%v", r1.IsError, r2.IsError)
	}
	if len(mgr.targets) != 2 {
		t.Errorf("expected 2 registered targets, got %d", len(mgr.targets))
	}
	// The second start should make the frontend active
	if mgr.active != "http://localhost:3000" {
		t.Errorf("expected last started target to be active, got %q", mgr.active)
	}
}

func TestDevPreviewTool_StopPreservesRegistrations(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:3000",
		"name":   "frontend",
	})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "stop",
	})

	if result.IsError {
		t.Fatalf("stop failed: %s", result.ForLLM)
	}
	// Registration should still be there
	if len(mgr.targets) != 1 {
		t.Errorf("expected 1 registered target after stop, got %d", len(mgr.targets))
	}
	// But active should be cleared
	if mgr.active != "" {
		t.Errorf("expected inactive after stop, got %q", mgr.active)
	}
}

func TestDevPreviewTool_StatusWithTargetsButInactive(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	mgr.RegisterDevTarget("api", "http://localhost:8080")
	// active remains empty

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("status failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "not active") {
		t.Errorf("expected 'not active' in status, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "api") {
		t.Errorf("expected 'api' listed in status, got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_ResultIsSilent(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	cases := []struct {
		name string
		args map[string]any
	}{
		{"start", map[string]any{"action": "start", "target": "http://localhost:3000"}},
		{"stop", map[string]any{"action": "stop"}},
		{"status", map[string]any{"action": "status"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tc.args)
			if result.IsError {
				t.Fatalf("expected success, got error: %s", result.ForLLM)
			}
			if result.Silent != true {
				t.Errorf("expected SilentResult (IsSilent=true), got IsSilent=%v", result.Silent)
			}
		})
	}
}

func TestDevPreviewTool_ActionTypeNotString(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": 123,
	})
	if !result.IsError {
		t.Error("expected error for non-string action")
	}
}

func TestDevPreviewTool_InferName(t *testing.T) {
	cases := []struct {
		target string
		want   string
	}{
		{"http://localhost:3000", "localhost:3000"},
		{"http://localhost:8080", "localhost:8080"},
		{"http://127.0.0.1:9000", "127.0.0.1:9000"},
		{"http://localhost", "localhost"},
		{"http://[::1]:5000", "::1:5000"},
		{"not-a-url", ""}, // url.Parse succeeds but Hostname() is empty
	}
	for _, tc := range cases {
		got := inferName(tc.target)
		if got != tc.want {
			t.Errorf("inferName(%q) = %q, want %q", tc.target, got, tc.want)
		}
	}
}

func TestDevPreviewTool_StartEmptyName(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	// Explicitly pass empty name — should auto-infer
	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:5000",
		"name":   "",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	for _, dt := range mgr.targets {
		if dt.Name != "localhost:5000" {
			t.Errorf("expected auto-name 'localhost:5000', got %q", dt.Name)
		}
	}
}

func TestDevPreviewTool_UnregisterActiveTarget(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	// Register and activate
	tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:3000",
		"name":   "frontend",
	})

	// Find the registered ID
	var id string
	for k := range mgr.targets {
		id = k
	}

	result := tool.Execute(context.Background(), map[string]any{
		"action": "unregister",
		"id":     id,
	})

	if result.IsError {
		t.Fatalf("unregister failed: %s", result.ForLLM)
	}
	if len(mgr.targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(mgr.targets))
	}
	if mgr.active != "" {
		t.Errorf("expected no active target, got %q", mgr.active)
	}
}

func TestDevPreviewTool_StartTargetEmptyString(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "",
	})

	if !result.IsError {
		t.Error("expected error for empty target string")
	}
}

func TestDevPreviewTool_StatusActiveNoTargets(t *testing.T) {
	// Edge case: active proxy but no registered targets (shouldn't normally happen)
	mgr := newMockManager()
	mgr.active = "http://localhost:9999" // active but targets map is empty
	tool := NewDevPreviewTool(mgr)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "active") {
		t.Errorf("expected 'active' in result, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "http://localhost:9999") {
		t.Errorf("expected target URL in result, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No registered targets") {
		t.Errorf("expected 'No registered targets' in result, got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_StatusOutputFormat(t *testing.T) {
	mgr := newMockManager()
	tool := NewDevPreviewTool(mgr)

	id1, _ := mgr.RegisterDevTarget("api", "http://localhost:8080")
	mgr.RegisterDevTarget("frontend", "http://localhost:3000")
	mgr.ActivateDevTarget(id1)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("status failed: %s", result.ForLLM)
	}
	// Should contain IDs in bracket format
	if !strings.Contains(result.ForLLM, "["+id1+"]") {
		t.Errorf("expected [%s] in output, got %q", id1, result.ForLLM)
	}
	// Should contain the arrow
	if !strings.Contains(result.ForLLM, "→") {
		t.Errorf("expected arrow in output, got %q", result.ForLLM)
	}
	// Should contain "Registered targets:"
	if !strings.Contains(result.ForLLM, "Registered targets:") {
		t.Errorf("expected 'Registered targets:' header, got %q", result.ForLLM)
	}
}
