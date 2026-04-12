// Package main provides the CLI for protoagent.
// This CLI tool allows users to generate agent artifacts from requirements via command line.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/protoagent"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate":
		runGenerate(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "version":
		printVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ProtoAgent CLI - Generate agent artifacts from requirements

Usage:
  protoagent-cli <command> [options]

Commands:
  generate    Generate artifacts from requirements file
  validate    Validate a requirements file
  version     Show version information
  help        Show this help message

Generate Options:
  protoagent-cli generate <requirements.json|yaml> [options]
    -o, --output <dir>      Output directory (default: ./output)
    -w, --workspace <dir>   Workspace directory (default: .)
    --opa                   Enable OPA policy generation
    --ai                    Enable AI-assisted generation
    --dry-run               Preview without writing files
    -v, --verbose           Verbose output

Validate Options:
  protoagent-cli validate <requirements.json|yaml>

Examples:
  protoagent-cli generate requirements.json -o ./output --opa
  protoagent-cli validate requirements.json
  protoagent-cli generate travel-experience-platform.json --verbose
`)
}

func printVersion() {
	fmt.Println("protoagent-cli version 0.1.0")
}

func runGenerate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: Requirements file is required")
		fmt.Fprintln(os.Stderr, "Usage: protoagent-cli generate <requirements.json|yaml> [options]")
		os.Exit(1)
	}

	reqFile := args[0]
	outputDir := "./output"
	workspace := "."
	enableOPA := false
	enableAI := false
	dryRun := false
	verbose := false

	// Parse arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 < len(args) {
				outputDir = args[i+1]
				i++
			}
		case "-w", "--workspace":
			if i+1 < len(args) {
				workspace = args[i+1]
				i++
			}
		case "--opa":
			enableOPA = true
		case "--ai":
			enableAI = true
		case "--dry-run":
			dryRun = true
		case "-v", "--verbose":
			verbose = true
		}
	}

	if verbose {
		fmt.Printf("📄 Reading requirements from: %s\n", reqFile)
	}

	// Load requirements
	reqs, err := loadRequirements(reqFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading requirements: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("📋 Loaded %d functional requirements and %d non-functional requirements\n",
			len(reqs.FunctionalRequirements), len(reqs.NonFunctionalRequirements))
	}

	// Configure engine
	config := protoagent.EngineConfig{
		OutputDir: outputDir,
		Workspace: workspace,
		EnableOPA: enableOPA,
		EnableAI:  enableAI,
		DryRun:    dryRun,
		Verbose:   verbose,
	}

	engine := protoagent.NewEngine(config)

	if verbose {
		fmt.Println("🚀 Processing requirements...")
	}

	// Process requirements
	ctx := context.Background()
	artifacts, err := engine.ProcessRequirements(ctx, reqs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing requirements: %v\n", err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("🔍 Dry run mode - no files written")
		printArtifactsSummary(artifacts)
		return
	}

	// Save artifacts
	if err := saveArtifacts(artifacts, outputDir, verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving artifacts: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		printArtifactsSummary(artifacts)
	}

	fmt.Println("\n✅ Artifacts generated successfully!")
}

func runValidate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: Requirements file is required")
		fmt.Fprintln(os.Stderr, "Usage: protoagent-cli validate <requirements.json|yaml>")
		os.Exit(1)
	}

	reqFile := args[0]

	// Load requirements
	reqs, err := loadRequirements(reqFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading requirements: %v\n", err)
		os.Exit(1)
	}

	// Create a minimal engine for validation
	config := protoagent.EngineConfig{
		DryRun:  true,
		Verbose: true,
	}
	engine := protoagent.NewEngine(config)

	ctx := context.Background()
	_, err = engine.ProcessRequirements(ctx, reqs)

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Requirements validation passed!")
}

func loadRequirements(filename string) (*protoagent.RequirementsDocument, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var reqs protoagent.RequirementsDocument

	// Try JSON first
	if err := json.Unmarshal(data, &reqs); err == nil {
		return &reqs, nil
	}

	// Try YAML if JSON fails
	// Note: YAML support would require adding gopkg.in/yaml.v3 dependency
	return nil, fmt.Errorf("failed to parse requirements file (JSON format expected)")
}

func saveArtifacts(artifacts *protoagent.GeneratedArtifacts, outputDir string, verbose bool) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save AGENT.md
	if artifacts.AgentConfig != nil {
		agentJSON, _ := json.MarshalIndent(artifacts.AgentConfig, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "AGENT.json"), agentJSON, 0644); err != nil {
			return fmt.Errorf("failed to save AGENT.json: %w", err)
		}

		agentMD := fmt.Sprintf("# %s Agent\n\n%s\n", artifacts.AgentConfig.Name, artifacts.AgentConfig.Body)
		if err := os.WriteFile(filepath.Join(outputDir, "AGENT.md"), []byte(agentMD), 0644); err != nil {
			return fmt.Errorf("failed to save AGENT.md: %w", err)
		}

		if verbose {
			fmt.Println("📄 AGENT.json and AGENT.md saved")
		}
	}

	// Save database schemas
	for i, schema := range artifacts.DatabaseSchemas {
		schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
		filename := filepath.Join(outputDir, fmt.Sprintf("schema_%d_%s.json", i, sanitizeName(schema.Name)))
		if err := os.WriteFile(filename, schemaJSON, 0644); err != nil {
			return fmt.Errorf("failed to save schema: %w", err)
		}

		// Generate SQL DDL for SQL schemas
		if schema.Type == "sql" && len(schema.Tables) > 0 {
			sqlDDL := generateSQLDDL(schema)
			sqlFilename := filepath.Join(outputDir, fmt.Sprintf("schema_%d_%s.sql", i, sanitizeName(schema.Name)))
			if err := os.WriteFile(sqlFilename, []byte(sqlDDL), 0644); err != nil {
				return fmt.Errorf("failed to save SQL: %w", err)
			}
			if verbose {
				fmt.Printf("📄 Schema %s saved (JSON + SQL)\n", schema.Name)
			}
		} else if verbose {
			fmt.Printf("📄 Schema %s saved\n", schema.Name)
		}
	}

	// Save OPA policies
	for i, policy := range artifacts.Policies {
		policyJSON, _ := json.MarshalIndent(policy, "", "  ")
		filename := filepath.Join(outputDir, fmt.Sprintf("policy_%d_%s.rego.json", i, sanitizeName(policy.Name)))
		if err := os.WriteFile(filename, policyJSON, 0644); err != nil {
			return fmt.Errorf("failed to save policy: %w", err)
		}

		// Save pure Rego code
		regoFilename := filepath.Join(outputDir, fmt.Sprintf("policy_%d_%s.rego", i, sanitizeName(policy.Name)))
		if err := os.WriteFile(regoFilename, []byte(policy.Rego), 0644); err != nil {
			return fmt.Errorf("failed to save rego: %w", err)
		}

		if verbose {
			fmt.Printf("📄 Policy %s saved (JSON + Rego)\n", policy.Name)
		}
	}

	// Save interfaces
	if len(artifacts.Interfaces) > 0 {
		interfacesJSON, _ := json.MarshalIndent(artifacts.Interfaces, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "interfaces.json"), interfacesJSON, 0644); err != nil {
			return fmt.Errorf("failed to save interfaces: %w", err)
		}
		if verbose {
			fmt.Println("📄 interfaces.json saved")
		}
	}

	// Save channels
	if len(artifacts.Channels) > 0 {
		channelsJSON, _ := json.MarshalIndent(artifacts.Channels, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "channels.json"), channelsJSON, 0644); err != nil {
			return fmt.Errorf("failed to save channels: %w", err)
		}
		if verbose {
			fmt.Println("📄 channels.json saved")
		}
	}

	// Save skills
	if len(artifacts.Skills) > 0 {
		skillsJSON, _ := json.MarshalIndent(artifacts.Skills, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "skills.json"), skillsJSON, 0644); err != nil {
			return fmt.Errorf("failed to save skills: %w", err)
		}

		// Save each skill's code
		for i, skill := range artifacts.Skills {
			skillFile := filepath.Join(outputDir, fmt.Sprintf("skill_%d_%s.go", i, sanitizeName(skill.Name)))
			if err := os.WriteFile(skillFile, []byte(skill.Code), 0644); err != nil {
				return fmt.Errorf("failed to save skill code: %w", err)
			}
		}

		if verbose {
			fmt.Println("📄 skills.json and skill codes saved")
		}
	}

	// Save tools
	if len(artifacts.Tools) > 0 {
		toolsJSON, _ := json.MarshalIndent(artifacts.Tools, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "tools.json"), toolsJSON, 0644); err != nil {
			return fmt.Errorf("failed to save tools: %w", err)
		}
		if verbose {
			fmt.Println("📄 tools.json saved")
		}
	}

	// Save MCP configuration
	if artifacts.MCPConfig != nil && len(artifacts.MCPConfig.Servers) > 0 {
		mcpJSON, _ := json.MarshalIndent(artifacts.MCPConfig, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "mcp_config.json"), mcpJSON, 0644); err != nil {
			return fmt.Errorf("failed to save mcp_config: %w", err)
		}
		if verbose {
			fmt.Println("📄 mcp_config.json saved")
		}
	}

	// Save validation report
	if artifacts.ValidationReport != nil {
		reportJSON, _ := json.MarshalIndent(artifacts.ValidationReport, "", "  ")
		if err := os.WriteFile(filepath.Join(outputDir, "validation_report.json"), reportJSON, 0644); err != nil {
			return fmt.Errorf("failed to save validation report: %w", err)
		}
		if verbose {
			fmt.Println("📄 validation_report.json saved")
		}
	}

	return nil
}

func printArtifactsSummary(artifacts *protoagent.GeneratedArtifacts) {
	fmt.Println("\n📦 Generated Artifacts Summary:")
	fmt.Println(strings.Repeat("=", 50))

	if artifacts.AgentConfig != nil {
		fmt.Printf("🤖 Agent: %s\n", artifacts.AgentConfig.Name)
		fmt.Printf("   Description: %s\n", artifacts.AgentConfig.Description)
		fmt.Printf("   Tools: %v\n", artifacts.AgentConfig.Tools)
		fmt.Printf("   Skills: %v\n", artifacts.AgentConfig.Skills)
	}

	if len(artifacts.DatabaseSchemas) > 0 {
		fmt.Printf("\n💾 Database Schemas: %d\n", len(artifacts.DatabaseSchemas))
		for _, schema := range artifacts.DatabaseSchemas {
			fmt.Printf("   - %s (%s)\n", schema.Name, schema.Type)
			if len(schema.Tables) > 0 {
				for _, table := range schema.Tables {
					fmt.Printf("     Table: %s (%d columns)\n", table.Name, len(table.Columns))
				}
			}
		}
	}

	if len(artifacts.Interfaces) > 0 {
		fmt.Printf("\n🖥️ Interfaces: %d\n", len(artifacts.Interfaces))
		for _, iface := range artifacts.Interfaces {
			fmt.Printf("   - %s (%s)\n", iface.Name, iface.Type)
			if iface.Type == "api" && len(iface.Endpoints) > 0 {
				fmt.Printf("     Endpoints: %d\n", len(iface.Endpoints))
			}
			if iface.Type == "web" && len(iface.Screens) > 0 {
				fmt.Printf("     Screens: %d\n", len(iface.Screens))
			}
		}
	}

	if len(artifacts.Channels) > 0 {
		fmt.Printf("\n📱 Communication Channels: %d\n", len(artifacts.Channels))
		for _, channel := range artifacts.Channels {
			fmt.Printf("   - %s (%s) - Enabled: %v\n", channel.Name, channel.Type, channel.Enabled)
		}
	}

	if len(artifacts.Policies) > 0 {
		fmt.Printf("\n🔐 OPA Policies: %d\n", len(artifacts.Policies))
		for _, policy := range artifacts.Policies {
			fmt.Printf("   - %s (%s)\n", policy.Name, policy.Package)
		}
	}

	if len(artifacts.Skills) > 0 {
		fmt.Printf("\n🎯 Skills: %d\n", len(artifacts.Skills))
		for _, skill := range artifacts.Skills {
			fmt.Printf("   - %s\n", skill.Name)
		}
	}

	if len(artifacts.Tools) > 0 {
		fmt.Printf("\n🔧 Tools: %d\n", len(artifacts.Tools))
		for _, tool := range artifacts.Tools {
			fmt.Printf("   - %s (%s)\n", tool.Name, tool.Type)
		}
	}

	if artifacts.MCPConfig != nil && len(artifacts.MCPConfig.Servers) > 0 {
		fmt.Printf("\n🔌 MCP Servers: %d\n", len(artifacts.MCPConfig.Servers))
		for _, server := range artifacts.MCPConfig.Servers {
			fmt.Printf("   - %s (%s)\n", server.Name, server.Type)
		}
	}

	if artifacts.ValidationReport != nil {
		fmt.Printf("\n✅ Validation: %v\n", artifacts.ValidationReport.Valid)
		if len(artifacts.ValidationReport.Errors) > 0 {
			fmt.Printf("   ❌ Errors: %d\n", len(artifacts.ValidationReport.Errors))
		}
		if len(artifacts.ValidationReport.Warnings) > 0 {
			fmt.Printf("   ⚠️  Warnings: %d\n", len(artifacts.ValidationReport.Warnings))
		}
		if len(artifacts.ValidationReport.Suggestions) > 0 {
			fmt.Printf("   💡 Suggestions: %d\n", len(artifacts.ValidationReport.Suggestions))
		}
	}
}

func generateSQLDDL(schema protoagent.DatabaseSchema) string {
	var ddl strings.Builder
	ddl.WriteString(fmt.Sprintf("-- Schema: %s\n", schema.Name))
	ddl.WriteString(fmt.Sprintf("-- Type: %s\n\n", schema.Type))

	for _, table := range schema.Tables {
		ddl.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table.Name))

		columns := make([]string, 0, len(table.Columns))
		for _, col := range table.Columns {
			colDef := fmt.Sprintf("  %s %s", col.Name, col.Type)
			if col.PrimaryKey {
				colDef += " PRIMARY KEY"
			}
			if !col.Nullable {
				colDef += " NOT NULL"
			}
			if col.Unique {
				colDef += " UNIQUE"
			}
			if col.Default != "" {
				colDef += fmt.Sprintf(" DEFAULT %s", col.Default)
			}
			columns = append(columns, colDef)
		}

		ddl.WriteString(strings.Join(columns, ",\n"))
		ddl.WriteString("\n);\n\n")

		// Create indexes
		for _, idx := range table.Indexes {
			ddl.WriteString(fmt.Sprintf("CREATE INDEX ON %s (%s);\n", table.Name, idx))
		}
	}

	return ddl.String()
}

func sanitizeName(name string) string {
	// Replace invalid filename characters with underscores
	result := strings.ReplaceAll(name, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ToLower(result)
	return result
}

var _ = time.Now // Avoid unused import error
