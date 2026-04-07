package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/logger"
)

var (
	flagData   string
	flagOut    string
	flagMode   string
	flagBudget int
)

func main() {
	// Suppress seahorse INFO logs during benchmark
	logger.SetLevel(logger.WARN)

	rootCmd := &cobra.Command{
		Use:   "membench",
		Short: "Memory benchmark tool for picoclaw",
	}

	ingestCmd := &cobra.Command{
		Use:   "ingest",
		Short: "Load LOCOMO data into storage backends",
		RunE:  runIngest,
	}
	ingestCmd.Flags().StringVar(&flagData, "data", "", "LOCOMO dataset directory (required)")
	ingestCmd.Flags().StringVar(&flagOut, "out", "./bench-out", "output working directory")
	ingestCmd.Flags().StringVar(&flagMode, "mode", "all", "modes to ingest: legacy, seahorse, or all")

	evalCmd := &cobra.Command{
		Use:   "eval",
		Short: "Run QA evaluation against ingested data",
		RunE:  runEval,
	}
	evalCmd.Flags().StringVar(&flagData, "data", "", "LOCOMO dataset directory (required)")
	evalCmd.Flags().StringVar(&flagOut, "out", "./bench-out", "output working directory")
	evalCmd.Flags().StringVar(&flagMode, "mode", "all", "modes to evaluate: legacy, seahorse, or all")
	evalCmd.Flags().IntVar(&flagBudget, "budget", 4000, "token budget for retrieval")

	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Output comparison results from evaluation",
		RunE:  runReport,
	}
	reportCmd.Flags().StringVar(&flagOut, "out", "./bench-out", "output working directory")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Convenience: eval + report (ingestion is done inline)",
		RunE:  runAll,
	}
	runCmd.Flags().StringVar(&flagData, "data", "", "LOCOMO dataset directory (required)")
	runCmd.Flags().StringVar(&flagOut, "out", "./bench-out", "output working directory")
	runCmd.Flags().StringVar(&flagMode, "mode", "all", "modes to run: legacy, seahorse, or all")
	runCmd.Flags().IntVar(&flagBudget, "budget", 4000, "token budget for retrieval")

	rootCmd.AddCommand(ingestCmd, evalCmd, reportCmd, runCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func modesFromFlag() []string {
	switch strings.ToLower(flagMode) {
	case "all":
		return []string{"legacy", "seahorse"}
	default:
		return []string{strings.ToLower(flagMode)}
	}
}

func runIngest(cmd *cobra.Command, args []string) error {
	if flagData == "" {
		return fmt.Errorf("--data is required")
	}
	modes := modesFromFlag()
	if len(modes) == 0 {
		return nil
	}

	ctx := context.Background()
	samples, err := LoadDataset(flagData)
	if err != nil {
		return fmt.Errorf("load dataset: %w", err)
	}
	log.Printf("Loaded %d samples from %s", len(samples), flagData)

	for _, mode := range modes {
		switch mode {
		case "legacy":
			legacy := NewLegacyStore()
			for i := range samples {
				legacy.IngestSample(&samples[i])
			}
			log.Printf("legacy: ingested %d samples", len(samples))
		case "seahorse":
			dbPath := filepath.Join(flagOut, "seahorse.db")
			if err := os.MkdirAll(flagOut, 0o755); err != nil {
				return fmt.Errorf("create out dir: %w", err)
			}
			_, err := IngestSeahorse(ctx, samples, dbPath)
			if err != nil {
				return fmt.Errorf("ingest seahorse: %w", err)
			}
		}
	}
	return nil
}

func runEval(cmd *cobra.Command, args []string) error {
	if flagData == "" {
		return fmt.Errorf("--data is required")
	}
	modes := modesFromFlag()
	if len(modes) == 0 {
		return nil
	}

	ctx := context.Background()
	samples, err := LoadDataset(flagData)
	if err != nil {
		return fmt.Errorf("load dataset: %w", err)
	}
	log.Printf("Loaded %d samples", len(samples))

	var allResults []EvalResult

	for _, mode := range modes {
		switch mode {
		case "legacy":
			legacy := NewLegacyStore()
			for i := range samples {
				legacy.IngestSample(&samples[i])
			}
			results := EvalLegacy(ctx, samples, legacy, flagBudget)
			allResults = append(allResults, results...)
			log.Printf("legacy: evaluated %d samples", len(results))
		case "seahorse":
			dbPath := filepath.Join(flagOut, "seahorse.db")
			ir, err := IngestSeahorse(ctx, samples, dbPath)
			if err != nil {
				return fmt.Errorf("ingest seahorse: %w", err)
			}
			results := EvalSeahorse(ctx, samples, ir, flagBudget)
			allResults = append(allResults, results...)
			log.Printf("seahorse: evaluated %d samples", len(results))
		}
	}

	if err := SaveResults(allResults, flagOut); err != nil {
		return fmt.Errorf("save results: %w", err)
	}
	if err := SaveAggregated(allResults, flagOut); err != nil {
		return fmt.Errorf("save aggregated: %w", err)
	}

	PrintComparison(allResults, nil)
	return nil
}

func runReport(cmd *cobra.Command, args []string) error {
	entries, err := os.ReadDir(flagOut)
	if err != nil {
		return fmt.Errorf("read out dir: %w", err)
	}

	var allResults []EvalResult
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "eval_") && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(flagOut, entry.Name())
			var r EvalResult
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("WARN: read %s: %v", path, err)
				continue
			}
			if err := json.Unmarshal(data, &r); err != nil {
				log.Printf("WARN: parse %s: %v", path, err)
				continue
			}
			allResults = append(allResults, r)
		}
	}

	if len(allResults) == 0 {
		return fmt.Errorf("no eval results found in %s", flagOut)
	}

	PrintComparison(allResults, nil)
	return nil
}

func runAll(cmd *cobra.Command, args []string) error {
	return runEval(cmd, args)
}
