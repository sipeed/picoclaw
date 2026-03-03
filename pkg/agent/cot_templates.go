// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// CotTemplate represents a Chain-of-Thought prompting template.
type CotTemplate struct {
	ID          string // Short identifier (e.g. "analytical", "code")
	Name        string // Human-readable name
	Description string // One-line description for the pre-LLM to choose from
	Prompt      string // The actual CoT instruction injected into the system prompt
}

// --- Built-in CoT Templates -------------------------------------------------

var builtinCotTemplates = []CotTemplate{
	{
		ID:          "direct",
		Name:        "Direct Answer",
		Description: "Simple, direct response — no special reasoning needed",
		Prompt:      "", // No CoT injection for simple answers
	},
	{
		ID:          "analytical",
		Name:        "Analytical Reasoning",
		Description: "Complex questions requiring step-by-step logical analysis",
		Prompt: `## Thinking Strategy: Analytical Reasoning

Before answering, follow this reasoning process:
1. **Clarify** — Restate the core question in your own words.
2. **Decompose** — Break it into sub-problems or key aspects.
3. **Analyse** — Work through each sub-problem with evidence/logic.
4. **Synthesise** — Combine findings into a coherent answer.
5. **Verify** — Check for logical gaps or contradictions.`,
	},
	{
		ID:          "code",
		Name:        "Code Analysis",
		Description: "Writing, reviewing, or understanding code",
		Prompt: `## Thinking Strategy: Code Analysis

Before writing or analysing code:
1. **Requirements** — What exactly needs to be done?
2. **Inputs/Outputs** — Define the interface: what goes in, what comes out.
3. **Edge Cases** — Consider boundary conditions, errors, empty inputs, concurrency.
4. **Approach** — Choose the algorithm/pattern, justify the choice.
5. **Implement** — Write clean, well-commented code.
6. **Test** — Mentally trace through with sample inputs to verify correctness.`,
	},
	{
		ID:          "debug",
		Name:        "Debugging",
		Description: "Finding and fixing bugs, errors, or unexpected behaviour",
		Prompt: `## Thinking Strategy: Debugging

Follow a systematic debugging approach:
1. **Reproduce** — Understand the exact symptoms and conditions.
2. **Hypothesise** — List 2-3 most likely root causes.
3. **Narrow Down** — For each hypothesis, describe what evidence would confirm/deny it.
4. **Root Cause** — Identify the actual root cause with evidence.
5. **Fix** — Propose the minimal, targeted fix.
6. **Verify** — Confirm the fix resolves the issue without side effects.`,
	},
	{
		ID:          "creative",
		Name:        "Creative Thinking",
		Description: "Brainstorming, creative writing, idea generation",
		Prompt: `## Thinking Strategy: Creative Exploration

Use divergent-convergent thinking:
1. **Diverge** — Generate multiple distinct ideas or approaches without judgment.
2. **Explore** — Expand on the most promising 2-3 ideas.
3. **Combine** — Look for unexpected connections between ideas.
4. **Converge** — Select the best approach and refine it.
5. **Polish** — Add detail, nuance, and completeness.`,
	},
	{
		ID:          "task",
		Name:        "Task Planning",
		Description: "Multi-step tasks, planning, project work",
		Prompt: `## Thinking Strategy: Task Planning

Plan before executing:
1. **Goal** — What is the desired end state?
2. **Current State** — What exists now? What resources are available?
3. **Steps** — Break into ordered, actionable steps.
4. **Dependencies** — Identify which steps depend on others.
5. **Risks** — What could go wrong? How to mitigate?
6. **Execute** — Carry out steps, adapting as needed.`,
	},
	{
		ID:          "explain",
		Name:        "Explain / Teach",
		Description: "Teaching concepts, explaining how things work",
		Prompt: `## Thinking Strategy: Educational Explanation

Structure your explanation for clarity:
1. **Big Picture** — Start with a one-sentence summary of the concept.
2. **Analogy** — Relate to something familiar if possible.
3. **Core Mechanism** — Explain how it works step by step.
4. **Example** — Provide a concrete example or demonstration.
5. **Gotchas** — Mention common misconceptions or pitfalls.`,
	},
	{
		ID:          "compare",
		Name:        "Comparison / Decision",
		Description: "Comparing options, making decisions, trade-off analysis",
		Prompt: `## Thinking Strategy: Comparison Analysis

Structure your analysis:
1. **Criteria** — Define what matters most for this decision.
2. **Options** — List all viable options.
3. **Trade-offs** — For each option, list pros and cons against the criteria.
4. **Recommendation** — State the best choice with clear reasoning.
5. **Caveats** — Note when the recommendation might not apply.`,
	},
}

// --- CoT Template Registry --------------------------------------------------

// CotRegistry manages the available CoT templates.
// It loads built-in templates and supports user-defined ones from workspace.
type CotRegistry struct {
	mu        sync.RWMutex
	templates map[string]CotTemplate
}

// NewCotRegistry creates a registry with built-in templates and optionally
// loads user-defined templates from the workspace/cot_templates/ directory.
func NewCotRegistry(workspace string) *CotRegistry {
	r := &CotRegistry{
		templates: make(map[string]CotTemplate, len(builtinCotTemplates)),
	}

	// Register built-in templates.
	for _, t := range builtinCotTemplates {
		r.templates[t.ID] = t
	}

	// Load user-defined templates from workspace.
	r.loadUserTemplates(workspace)

	return r
}

// Get returns a template by ID (case-insensitive). Returns the "direct"
// template if not found.
func (r *CotRegistry) Get(id string) CotTemplate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id = strings.ToLower(strings.TrimSpace(id))
	if t, ok := r.templates[id]; ok {
		return t
	}
	return r.templates["direct"]
}

// ListForPrompt returns a formatted list of available template IDs and
// descriptions, suitable for quick reference.
func (r *CotRegistry) ListForPrompt() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	for _, t := range builtinCotTemplates {
		fmt.Fprintf(&sb, "- %s: %s\n", t.ID, t.Description)
	}

	// Append user-defined templates.
	for id, t := range r.templates {
		isBuiltin := false
		for _, bt := range builtinCotTemplates {
			if bt.ID == id {
				isBuiltin = true
				break
			}
		}
		if !isBuiltin {
			fmt.Fprintf(&sb, "- %s: %s\n", t.ID, t.Description)
		}
	}

	return sb.String()
}

// ListExamplesForPrompt returns full template examples for the pre-LLM to
// use as inspiration when generating custom CoT prompts.
// Shows 3-4 diverse examples with their full prompt content.
func (r *CotRegistry) ListExamplesForPrompt() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Select a diverse set of examples (not all — keep prompt concise).
	exampleIDs := []string{"analytical", "code", "debug", "task"}

	var sb strings.Builder
	for _, id := range exampleIDs {
		t, ok := r.templates[id]
		if !ok || t.Prompt == "" {
			continue
		}
		fmt.Fprintf(&sb, "### Example: %s (%s)\n%s\n\n", t.Name, t.Description, t.Prompt)
	}

	// Append any user-defined templates as additional examples.
	for id, t := range r.templates {
		isBuiltin := false
		for _, bt := range builtinCotTemplates {
			if bt.ID == id {
				isBuiltin = true
				break
			}
		}
		if !isBuiltin && t.Prompt != "" {
			fmt.Fprintf(&sb, "### Example: %s (%s)\n%s\n\n", t.Name, t.Description, t.Prompt)
		}
	}

	return sb.String()
}

// loadUserTemplates scans workspace/cot_templates/ for .md files.
// Each file becomes a template with ID = filename (without .md).
// File format:
//
//	Line 1: description (one line)
//	Line 2: ---
//	Line 3+: prompt content
func (r *CotRegistry) loadUserTemplates(workspace string) {
	dir := filepath.Join(workspace, "cot_templates")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // Directory doesn't exist — that's fine.
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".md")
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" {
			continue
		}

		content := string(data)
		description := id
		prompt := content

		// Parse optional description header.
		if idx := strings.Index(content, "\n---\n"); idx > 0 {
			description = strings.TrimSpace(content[:idx])
			prompt = strings.TrimSpace(content[idx+5:])
		}

		r.mu.Lock()
		r.templates[id] = CotTemplate{
			ID:          id,
			Name:        id,
			Description: description,
			Prompt:      prompt,
		}
		r.mu.Unlock()

		logger.DebugCF("cot", "Loaded user CoT template",
			map[string]any{"id": id, "description": description})
	}
}
