package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
)

// createBenchmarkSkills creates a set of skills for benchmarking.
func createBenchmarkSkills(basePath string, count int) error {
	for i := 0; i < count; i++ {
		skillName := fmt.Sprintf("benchmark-skill-%d", i)
		skillDir := filepath.Join(basePath, "skills", skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return err
		}

		skillFile := filepath.Join(skillDir, "SKILL.md")
		content := fmt.Sprintf(`---
name: %s
description: This is benchmark skill number %d with some description text
---

# %s

This is the content of benchmark skill %d.
It contains multiple lines to simulate real skill content.
The purpose is to test token usage in different context building strategies.
`, skillName, i, skillName, i)

		if err := os.WriteFile(skillFile, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// BenchmarkContextBuilder_FullStrategy measures performance of Full strategy.
func BenchmarkContextBuilder_FullStrategy(b *testing.B) {
	workspace := b.TempDir()

	// Create 10 skills for realistic benchmark
	if err := createBenchmarkSkills(workspace, 10); err != nil {
		b.Fatal(err)
	}

	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "Hello, I need help with my code"},
		{Role: "assistant", Content: "Sure, I'd be happy to help! What do you need?"},
		{Role: "user", Content: "Can you review this function?"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyFull,
		IncludeMemory:  true,
		IncludeRuntime: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		messages := cb.BuildMessagesWithOptions(history, "", "Please review the code", nil, "telegram", "chat123", opts)
		if len(messages) == 0 {
			b.Fatal("no messages returned")
		}
	}
}

// BenchmarkContextBuilder_LiteStrategy measures performance of Lite strategy.
func BenchmarkContextBuilder_LiteStrategy(b *testing.B) {
	workspace := b.TempDir()

	// Create 10 skills
	if err := createBenchmarkSkills(workspace, 10); err != nil {
		b.Fatal(err)
	}

	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "Quick question"},
		{Role: "assistant", Content: "Yes?"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyLite,
		IncludeMemory:  false,
		IncludeRuntime: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		messages := cb.BuildMessagesWithOptions(history, "", "What time is it?", nil, "telegram", "chat123", opts)
		if len(messages) == 0 {
			b.Fatal("no messages returned")
		}
	}
}

// BenchmarkContextBuilder_CustomStrategy measures performance of Custom strategy.
func BenchmarkContextBuilder_CustomStrategy(b *testing.B) {
	workspace := b.TempDir()

	// Create 10 skills
	if err := createBenchmarkSkills(workspace, 10); err != nil {
		b.Fatal(err)
	}

	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "Help me with task X"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyCustom,
		IncludeSkills:  []string{"benchmark-skill-0", "benchmark-skill-1"},
		IncludeMemory:  true,
		IncludeRuntime: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		messages := cb.BuildMessagesWithOptions(history, "", "Do task X", nil, "telegram", "chat123", opts)
		if len(messages) == 0 {
			b.Fatal("no messages returned")
		}
	}
}

// BenchmarkContextBuilder_TokenUsageComparison compares token usage across strategies.
func BenchmarkContextBuilder_TokenUsageComparison(b *testing.B) {
	workspace := b.TempDir()

	// Create 20 skills for comprehensive comparison
	if err := createBenchmarkSkills(workspace, 20); err != nil {
		b.Fatal(err)
	}

	cb := NewContextBuilder(workspace)

	history := []providers.Message{
		{Role: "user", Content: "I need assistance"},
		{Role: "assistant", Content: "How can I help?"},
	}

	strategies := []ContextStrategy{
		ContextStrategyFull,
		ContextStrategyLite,
		ContextStrategyCustom,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, strategy := range strategies {
			opts := ContextBuildOptions{
				Strategy:       strategy,
				IncludeMemory:  true,
				IncludeRuntime: true,
				IncludeSkills:  []string{"benchmark-skill-0"}, // Custom strategy filter
			}

			messages := cb.BuildMessagesWithOptions(history, "", "Test message", nil, "telegram", "chat123", opts)
			if len(messages) == 0 {
				b.Fatal("no messages returned")
			}

			// Record token count (approximate by character count)
			totalChars := 0
			for _, msg := range messages {
				totalChars += len(msg.Content)
			}
			b.ReportMetric(float64(totalChars), "chars")
		}
	}
}

// BenchmarkContextBuilder_SystemPromptSize measures system prompt size for each strategy.
func BenchmarkContextBuilder_SystemPromptSize(b *testing.B) {
	workspace := b.TempDir()

	// Create varying number of skills
	skillCounts := []int{5, 10, 20, 50}

	for _, count := range skillCounts {
		b.Run(fmt.Sprintf("%d_skills", count), func(b *testing.B) {
			// Clean workspace
			os.RemoveAll(workspace)
			os.MkdirAll(workspace, 0o755)

			if err := createBenchmarkSkills(workspace, count); err != nil {
				b.Fatal(err)
			}

			cb := NewContextBuilder(workspace)

			opts := ContextBuildOptions{
				Strategy:       ContextStrategyFull,
				IncludeMemory:  false,
				IncludeRuntime: false,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				messages := cb.BuildMessagesWithOptions([]providers.Message{}, "", "test", nil, "telegram", "chat", opts)
				if len(messages) == 0 || len(messages[0].Content) == 0 {
					b.Fatal("empty system prompt")
				}

				b.ReportMetric(float64(len(messages[0].Content)), "system_prompt_chars")
			}
		})
	}
}

// BenchmarkContextBuilder_CachePerformance tests cache hit performance.
func BenchmarkContextBuilder_CachePerformance(b *testing.B) {
	workspace := b.TempDir()

	if err := createBenchmarkSkills(workspace, 15); err != nil {
		b.Fatal(err)
	}

	cb := NewContextBuilder(workspace)

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyFull,
		IncludeMemory:  false,
		IncludeRuntime: false,
	}

	// First call builds cache
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		messages := cb.BuildMessagesWithOptions([]providers.Message{}, "", "test", nil, "telegram", "chat", opts)
		if len(messages) == 0 {
			b.Fatal("no messages")
		}
	}
}

// BenchmarkContextBuilder_WithSkillRecommender tests performance with skill recommender.
func BenchmarkContextBuilder_WithSkillRecommender(b *testing.B) {
	workspace := b.TempDir()

	if err := createBenchmarkSkills(workspace, 10); err != nil {
		b.Fatal(err)
	}

	cb := NewContextBuilder(workspace)

	// Note: We're not setting up actual recommender here to avoid LLM dependency
	// In production, the recommender would add some overhead but improve relevance

	history := []providers.Message{
		{Role: "user", Content: "I want to create a poll"},
	}

	opts := ContextBuildOptions{
		Strategy:       ContextStrategyFull,
		IncludeMemory:  true,
		IncludeRuntime: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		messages := cb.BuildMessagesWithOptions(history, "", "Create a poll for me", nil, "telegram", "chat123", opts)
		if len(messages) == 0 {
			b.Fatal("no messages")
		}
	}
}

// BenchmarkSkillsLoader_BuildSkillsSummaryFiltered benchmarks filtered skill loading.
func BenchmarkSkillsLoader_BuildSkillsSummaryFiltered(b *testing.B) {
	workspace := b.TempDir()
	globalSkills := b.TempDir()
	builtinSkills := b.TempDir()

	// Create 50 skills
	if err := createBenchmarkSkills(workspace, 50); err != nil {
		b.Fatal(err)
	}

	loader := skills.NewSkillsLoader(workspace, globalSkills, builtinSkills)

	filterSizes := []int{1, 5, 10, 25, 50}

	for _, filterSize := range filterSizes {
		b.Run(fmt.Sprintf("filter_%d_skills", filterSize), func(b *testing.B) {
			filter := make([]string, filterSize)
			for i := 0; i < filterSize; i++ {
				filter[i] = fmt.Sprintf("benchmark-skill-%d", i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				summary := loader.BuildSkillsSummaryFiltered(filter)
				if summary == "" && filterSize > 0 {
					b.Fatal("empty summary")
				}
			}
		})
	}
}
