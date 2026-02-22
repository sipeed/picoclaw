// ABOUTME: CLI subcommand for querying and managing the semantic memory store.
// ABOUTME: Provides recall, remember, list, and clear operations from the command line.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/memory"
)

func memoryCmd() {
	if len(os.Args) < 3 {
		memoryHelp()
		return
	}

	subcommand := os.Args[2]

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	vectorDir := filepath.Join(workspace, "memory", "vectors")

	// "clear" doesn't need a store connection
	if subcommand == "clear" {
		confirmed := false
		for _, arg := range os.Args[3:] {
			if arg == "--yes" || arg == "-y" {
				confirmed = true
			}
		}
		memoryClearCmd(vectorDir, confirmed)
		return
	}

	store, err := memory.NewSemanticStore(
		vectorDir,
		cfg.Tools.Memory.OllamaURL,
		cfg.Tools.Memory.EmbeddingModel,
	)
	if err != nil {
		fmt.Printf("Error: could not open memory store: %v\n", err)
		os.Exit(1)
	}
	if !store.IsAvailable() {
		fmt.Println("Error: semantic memory is not available (is Ollama running?)")
		os.Exit(1)
	}

	switch subcommand {
	case "recall", "search":
		query, topK := parseRecallArgs(os.Args[3:])
		if query == "" {
			fmt.Println("Usage: picoclaw memory recall <query> [--top N]")
			os.Exit(1)
		}
		memoryRecallCmd(store, query, topK)
	case "remember", "add":
		content, category, tags := parseRememberArgs(os.Args[3:])
		if content == "" {
			fmt.Println("Usage: picoclaw memory remember <content> [--category TYPE] [--tags a,b,c]")
			os.Exit(1)
		}
		memoryRememberCmd(store, content, category, tags)
	case "list", "count":
		memoryListCmd(store)
	default:
		fmt.Printf("Unknown memory command: %s\n", subcommand)
		memoryHelp()
	}
}

func parseRecallArgs(args []string) (query string, topK int) {
	topK = 5
	var queryParts []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--top", "-n":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &topK)
				i++
			}
		default:
			queryParts = append(queryParts, args[i])
		}
	}

	query = strings.Join(queryParts, " ")
	return
}

func parseRememberArgs(args []string) (content, category string, tags []string) {
	category = "fact"
	var contentParts []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--category", "-c":
			if i+1 < len(args) {
				category = args[i+1]
				i++
			}
		case "--tags", "-t":
			if i+1 < len(args) {
				for _, t := range strings.Split(args[i+1], ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
				i++
			}
		default:
			contentParts = append(contentParts, args[i])
		}
	}

	content = strings.Join(contentParts, " ")
	return
}

func memoryRecallCmd(store *memory.SemanticStore, query string, topK int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := store.Recall(ctx, query, topK)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No memories found.")
		return
	}

	fmt.Printf("Found %d memories:\n\n", len(results))
	for i, r := range results {
		fmt.Printf("  %d. [%.0f%% match] [%s] %s\n", i+1, r.Similarity*100, r.Category, r.Content)
		if len(r.Tags) > 0 {
			fmt.Printf("     Tags: %s\n", strings.Join(r.Tags, ", "))
		}
		if !r.Timestamp.IsZero() {
			fmt.Printf("     Stored: %s\n", r.Timestamp.Format("2006-01-02 15:04"))
		}
	}
}

func memoryRememberCmd(store *memory.SemanticStore, content, category string, tags []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entry := memory.MemoryEntry{
		Content:  content,
		Category: category,
		Tags:     tags,
		Source:   "cli",
	}

	if err := store.Remember(ctx, entry); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Stored: %s\n", content)
}

func memoryListCmd(store *memory.SemanticStore) {
	count := store.Count()
	fmt.Printf("Memory store: %d entries\n", count)
	if count > 0 {
		fmt.Println("\nUse 'picoclaw memory recall <query>' to search.")
	}
}

func memoryClearCmd(vectorDir string, confirmed bool) {
	if _, err := os.Stat(vectorDir); os.IsNotExist(err) {
		fmt.Println("Memory store is already empty.")
		return
	}

	if !confirmed {
		fmt.Print("Clear all memories? This cannot be undone. [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	if err := os.RemoveAll(vectorDir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Memory store cleared.")
}

func memoryHelp() {
	fmt.Println("Usage: picoclaw memory <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  recall <query>     Search memories by semantic similarity")
	fmt.Println("  remember <text>    Store a memory")
	fmt.Println("  list               Show memory count")
	fmt.Println("  clear              Delete all memories")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  recall --top N           Number of results (default 5)")
	fmt.Println("  remember --category TYPE Category: fact, preference, decision, context, other")
	fmt.Println("  remember --tags a,b,c    Comma-separated tags")
	fmt.Println("  clear --yes              Skip confirmation prompt")
}
