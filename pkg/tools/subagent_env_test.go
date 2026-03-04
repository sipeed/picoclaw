package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPlanContext(t *testing.T) {
	tmpDir := t.TempDir()

	memDir := filepath.Join(tmpDir, "memory")

	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}

	memContent := `# Active Plan

> Task: Implement authentication

> Status: executing

> Phase: 1



## Context

The project uses JWT tokens for auth.

Database is PostgreSQL.



## Phase 1: Setup

- [x] Add middleware

- [ ] Add JWT validation



## Commands

build: go build ./...

test: go test ./...



## Orchestration

### Delegated

- auth-scout: investigate patterns

### Findings

- Found existing middleware in pkg/auth

`

	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(memContent), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := extractPlanContext(tmpDir)

	if ctx == "" {
		t.Fatal("extractPlanContext returned empty")
	}

	// Should contain the task line.

	if !strings.Contains(ctx, "> Task: Implement authentication") {
		t.Error("missing task line")
	}

	// Should contain Context section.

	if !strings.Contains(ctx, "JWT tokens") {
		t.Error("missing Context section content")
	}

	// Should contain Commands section.

	if !strings.Contains(ctx, "go build") {
		t.Error("missing Commands section content")
	}

	// Should contain Orchestration section.

	if !strings.Contains(ctx, "auth-scout") {
		t.Error("missing Orchestration section content")
	}
}

func TestExtractPlanContext_NoFile(t *testing.T) {
	ctx := extractPlanContext(t.TempDir())

	if ctx != "" {
		t.Errorf("expected empty for missing MEMORY.md, got %q", ctx)
	}
}

func TestExtractSection(t *testing.T) {
	content := `## Context

Some context here.



## Commands

build: go build



## Other

stuff`

	section := extractSection(content, "## Context")

	if !strings.Contains(section, "Some context here.") {
		t.Errorf("Context section = %q, missing content", section)
	}

	if strings.Contains(section, "## Commands") {
		t.Errorf("Context section leaked into next section")
	}

	section = extractSection(content, "## Commands")

	if !strings.Contains(section, "go build") {
		t.Errorf("Commands section = %q, missing content", section)
	}

	section = extractSection(content, "## Nonexistent")

	if section != "" {
		t.Errorf("expected empty for nonexistent section, got %q", section)
	}
}

func TestBuildSubagentSystemPrompt(t *testing.T) {
	// With no workspace/MEMORY.md, should return base prompt unchanged.

	base := "You are a subagent."

	got := buildSubagentSystemPrompt(base, t.TempDir())

	if got != base {
		t.Errorf("expected base prompt unchanged, got %q", got)
	}

	// With MEMORY.md, should append environment context.

	tmpDir := t.TempDir()

	memDir := filepath.Join(tmpDir, "memory")

	os.MkdirAll(memDir, 0o755)

	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(`# Plan

> Task: Test task



## Context

Test context info.

`), 0o644)

	got = buildSubagentSystemPrompt(base, tmpDir)

	if !strings.Contains(got, base) {
		t.Error("result should contain base prompt")
	}

	if !strings.Contains(got, "Environment Context") {
		t.Error("result should contain Environment Context header")
	}

	if !strings.Contains(got, "Test context info") {
		t.Error("result should contain MEMORY.md context")
	}
}
