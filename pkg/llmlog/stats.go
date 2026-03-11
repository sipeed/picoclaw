package llmlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TokenStats represents token usage statistics for a model
type TokenStats struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CallCount        int
	ErrorCount       int
}

// StatsResult represents the result of a stats query
type StatsResult struct {
	ByModel  map[string]*TokenStats
	Total    *TokenStats
	StartDay string
	EndDay   string
}

// CalculateStats calculates token usage statistics from log files
// If days is 0, it calculates all available logs
// If modelFilter is empty, it includes all models
// Supports filename formats: "2006-01-02.jsonl" and "llmcall_2006-01-02.jsonl"
func CalculateStats(logDir string, days int, modelFilter string) (*StatsResult, error) {
	result := &StatsResult{
		ByModel: make(map[string]*TokenStats),
		Total: &TokenStats{
			Model: "Total",
		},
	}

	now := time.Now()
	var startDate time.Time
	if days > 0 {
		startDate = now.AddDate(0, 0, -days)
		result.StartDay = startDate.Format("2006-01-02")
	} else {
		startDate = time.Time{} // zero time means all
	}
	result.EndDay = now.Format("2006-01-02")

	// Find all log files
	files, err := filepath.Glob(filepath.Join(logDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		// Parse date from filename
		// Supported formats: "2006-01-02.jsonl" and "llmcall_2006-01-02.jsonl"
		filename := filepath.Base(file)
		dateStr := strings.TrimSuffix(filename, ".jsonl")
		
		// Handle "llmcall_" prefix
		if strings.HasPrefix(dateStr, "llmcall_") {
			dateStr = strings.TrimPrefix(dateStr, "llmcall_")
		}
		
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // skip files with invalid names
		}

		// Check if file is within date range
		if !startDate.IsZero() && fileDate.Before(startDate) {
			continue
		}

		// Process file
		if err := processLogFile(file, modelFilter, result); err != nil {
			// Log error but continue processing other files
			continue
		}
	}

	// Sort models by total tokens for consistent output
	return result, nil
}

func processLogFile(filePath string, modelFilter string, result *StatsResult) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var record CallRecord
		if err := decoder.Decode(&record); err != nil {
			continue // skip malformed records
		}

		// Filter by model if specified
		if modelFilter != "" && record.Model != modelFilter {
			continue
		}

		// Get or create stats for this model
		stats, exists := result.ByModel[record.Model]
		if !exists {
			stats = &TokenStats{Model: record.Model}
			result.ByModel[record.Model] = stats
		}

		// Update stats
		stats.PromptTokens += record.PromptTokens
		stats.CompletionTokens += record.CompletionTokens
		stats.TotalTokens += record.TotalTokens
		stats.CallCount++
		if record.Error != "" {
			stats.ErrorCount++
		}

		// Update totals
		result.Total.PromptTokens += record.PromptTokens
		result.Total.CompletionTokens += record.CompletionTokens
		result.Total.TotalTokens += record.TotalTokens
		result.Total.CallCount++
		if record.Error != "" {
			result.Total.ErrorCount++
		}
	}

	return nil
}

// FormatStatsTable formats the stats result as a table string
func FormatStatsTable(result *StatsResult) string {
	if len(result.ByModel) == 0 {
		return "📊 no stats data found"
	}

	var sb strings.Builder

	if result.StartDay != "" {
		sb.WriteString(fmt.Sprintf("📊 Token Count (%s ~ %s)\n\n", result.StartDay, result.EndDay))
	} else {
		sb.WriteString(fmt.Sprintf("📊 Token Count (All，Until %s)\n\n", result.EndDay))
	}

	// Sort models by total tokens descending
	models := make([]string, 0, len(result.ByModel))
	for model := range result.ByModel {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		return result.ByModel[models[i]].TotalTokens > result.ByModel[models[j]].TotalTokens
	})

	// Table header
	sb.WriteString("┌──────────────────────┬──────────────┬─────────────────┬──────────────┬──────────┐\n")
	sb.WriteString("│ Model                │ Prompt Tokens│Completion Tokens│ Total Tokens │Call Count│\n")
	sb.WriteString("├──────────────────────┼──────────────┼─────────────────┼──────────────┼──────────┤\n")

	// Table rows
	for _, model := range models {
		stats := result.ByModel[model]
		// Truncate model name if too long
		displayModel := model
		if len(displayModel) > 20 {
			displayModel = displayModel[:17] + "..."
		}
		sb.WriteString(fmt.Sprintf("│ %-20s │ %12d │ %15d │ %12d │ %8d │\n",
			displayModel, stats.PromptTokens, stats.CompletionTokens, stats.TotalTokens, stats.CallCount))
	}

	// Total row
	sb.WriteString("├──────────────────────┼──────────────┼─────────────────┼──────────────┼──────────┤\n")
	sb.WriteString(fmt.Sprintf("│ %-20s │ %12d │ %15d │ %12d │ %8d │\n",
		"Total", result.Total.PromptTokens, result.Total.CompletionTokens, result.Total.TotalTokens, result.Total.CallCount))
	sb.WriteString("└──────────────────────┴──────────────┴─────────────────┴──────────────┴──────────┘\n")

	return sb.String()
}

// FormatSimpleStats formats stats for a single model (used in runtime /stats command)
func FormatSimpleStats(stats *TokenStats) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 Token Stats (Current Session)\n"))
	sb.WriteString(fmt.Sprintf("Model: %s\n", stats.Model))
	sb.WriteString("┌─────────────────────┬───────────┐\n")
	sb.WriteString("│ Metric              │ Value     │\n")
	sb.WriteString("├─────────────────────┼───────────┤\n")
	sb.WriteString(fmt.Sprintf("│ %-19s │ %9d │\n", "Prompt Tokens", stats.PromptTokens))
	sb.WriteString(fmt.Sprintf("│ %-19s │ %9d │\n", "Completion Tokens", stats.CompletionTokens))
	sb.WriteString(fmt.Sprintf("│ %-19s │ %9d │\n", "Total Tokens", stats.TotalTokens))
	sb.WriteString(fmt.Sprintf("│ %-19s │ %9d │\n", "Call Count", stats.CallCount))
	sb.WriteString("└─────────────────────┴───────────┘\n")

	return sb.String()
}