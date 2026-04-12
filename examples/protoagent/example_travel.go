package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/protoagent"
)

// Este exemplo demonstra como usar o ProtoAgent para gerar artefatos
// a partir de requisitos de uma plataforma de experiências de viagem
func main() {
	// Carregar requisitos do arquivo JSON
	reqs, err := loadRequirements("examples/protoagent/travel-experience-platform.json")
	if err != nil {
		fmt.Printf("Erro ao carregar requisitos: %v\n", err)
		os.Exit(1)
	}

	// Configurar o engine do ProtoAgent
	config := protoagent.EngineConfig{
		OutputDir: "./output/travel",
		Workspace: "./workspace",
		EnableOPA: true,
		EnableAI:  true, // Habilitar IA para recursos de curadoria e enriquecimento
		DryRun:    false,
		Verbose:   true,
	}

	engine := protoagent.NewEngine(config)

	// Processar requisitos e gerar artefatos
	ctx := context.Background()
	artifacts, err := engine.ProcessRequirements(ctx, reqs)
	if err != nil {
		fmt.Printf("Erro ao processar requisitos: %v\n", err)
		os.Exit(1)
	}

	// Exibir resumo dos artefatos gerados
	printArtifactsSummary(artifacts)

	// Salvar artefatos em arquivos
	if err := saveArtifacts(artifacts); err != nil {
		fmt.Printf("Erro ao salvar artefatos: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Artefatos gerados com sucesso!")
}

// loadRequirements carrega requisitos de um arquivo JSON
func loadRequirements(filename string) (*protoagent.RequirementsDocument, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo: %w", err)
	}

	var reqs protoagent.RequirementsDocument
	if err := json.Unmarshal(data, &reqs); err != nil {
		return nil, fmt.Errorf("falha ao parsear JSON: %w", err)
	}

	return &reqs, nil
}

// printArtifactsSummary exibe um resumo dos artefatos gerados
func printArtifactsSummary(artifacts *protoagent.GeneratedArtifacts) {
	fmt.Println("\n📦 Resumo dos Artefatos Gerados:")
	fmt.Println("=" + string(make([]byte, 50)))

	if artifacts.AgentConfig != nil {
		fmt.Printf("🤖 Agente: %s\n", artifacts.AgentConfig.Name)
		fmt.Printf("   Descrição: %s\n", artifacts.AgentConfig.Description)
		fmt.Printf("   Tools: %v\n", artifacts.AgentConfig.Tools)
		fmt.Printf("   Skills: %v\n", artifacts.AgentConfig.Skills)
	}

	if len(artifacts.DatabaseSchemas) > 0 {
		fmt.Printf("\n💾 Schemas de Banco de Dados: %d\n", len(artifacts.DatabaseSchemas))
		for _, schema := range artifacts.DatabaseSchemas {
			fmt.Printf("   - %s (%s)\n", schema.Name, schema.Type)
			if len(schema.Tables) > 0 {
				for _, table := range schema.Tables {
					fmt.Printf("     Tabela: %s (%d colunas)\n", table.Name, len(table.Columns))
					for _, col := range table.Columns {
						fmt.Printf("       - %s: %s", col.Name, col.Type)
						if col.PrimaryKey {
							fmt.Print(" [PK]")
						}
						if !col.Nullable {
							fmt.Print(" [NOT NULL]")
						}
						fmt.Println()
					}
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
				for _, ep := range iface.Endpoints {
					fmt.Printf("       [%s] %s - %s\n", ep.Method, ep.Path, ep.Description)
				}
			}
			if iface.Type == "web" && len(iface.Screens) > 0 {
				fmt.Printf("     Telas: %d\n", len(iface.Screens))
				for _, screen := range iface.Screens {
					fmt.Printf("       📱 %s (%s)\n", screen.Name, screen.Route)
					if len(screen.Components) > 0 {
						fmt.Printf("         Componentes: %d\n", len(screen.Components))
					}
				}
			}
		}
	}

	if len(artifacts.Channels) > 0 {
		fmt.Printf("\n📱 Canais de Comunicação: %d\n", len(artifacts.Channels))
		for _, channel := range artifacts.Channels {
			fmt.Printf("   - %s (%s) - Habilitado: %v\n", channel.Name, channel.Type, channel.Enabled)
			if len(channel.Config) > 0 {
				fmt.Printf("     Configuração:\n")
				for k, v := range channel.Config {
					fmt.Printf("       %s: %s\n", k, v)
				}
			}
		}
	}

	if len(artifacts.Policies) > 0 {
		fmt.Printf("\n🔐 Políticas OPA: %d\n", len(artifacts.Policies))
		for _, policy := range artifacts.Policies {
			fmt.Printf("   - %s (%s)\n", policy.Name, policy.Package)
			fmt.Printf("     Descrição: %s\n", policy.Description)
			
			// Mostrar preview do código Rego
			regoLines := splitLines(policy.Rego)
			if len(regoLines) > 0 {
				fmt.Printf("     Código Rego (%d linhas):\n", len(regoLines))
				previewLen := len(regoLines)
				if previewLen > 5 {
					previewLen = 5
				}
				for i := 0; i < previewLen; i++ {
					fmt.Printf("       %s\n", regoLines[i])
				}
				if len(regoLines) > 5 {
					fmt.Printf("       ... (%d linhas restantes)\n", len(regoLines)-5)
				}
			}
		}
	}

	if len(artifacts.Skills) > 0 {
		fmt.Printf("\n🎯 Skills: %d\n", len(artifacts.Skills))
		for _, skill := range artifacts.Skills {
			fmt.Printf("   - %s\n", skill.Name)
			fmt.Printf("     Descrição: %s\n", skill.Description)
			if len(skill.Triggers) > 0 {
				fmt.Printf("     Triggers: %v\n", skill.Triggers)
			}
			if len(skill.Dependencies) > 0 {
				fmt.Printf("     Dependências: %v\n", skill.Dependencies)
			}
		}
	}

	if len(artifacts.Tools) > 0 {
		fmt.Printf("\n🔧 Tools: %d\n", len(artifacts.Tools))
		for _, tool := range artifacts.Tools {
			fmt.Printf("   - %s (%s)\n", tool.Name, tool.Type)
			fmt.Printf("     Descrição: %s\n", tool.Description)
			if len(tool.Config) > 0 {
				fmt.Printf("     Configuração:\n")
				for k, v := range tool.Config {
					fmt.Printf("       %s: %s\n", k, v)
				}
			}
		}
	}

	if artifacts.MCPConfig != nil && len(artifacts.MCPConfig.Servers) > 0 {
		fmt.Printf("\n🔌 Servidores MCP: %d\n", len(artifacts.MCPConfig.Servers))
		for _, server := range artifacts.MCPConfig.Servers {
			fmt.Printf("   - %s (%s)\n", server.Name, server.Type)
			if server.Command != "" {
				fmt.Printf("     Comando: %s %v\n", server.Command, server.Args)
			}
			if server.URL != "" {
				fmt.Printf("     URL: %s\n", server.URL)
			}
		}
	}

	if artifacts.ValidationReport != nil {
		fmt.Printf("\n✅ Validação: %v\n", artifacts.ValidationReport.Valid)
		if len(artifacts.ValidationReport.Errors) > 0 {
			fmt.Printf("   ❌ Erros: %d\n", len(artifacts.ValidationReport.Errors))
			for _, err := range artifacts.ValidationReport.Errors {
				fmt.Printf("     - %s: %s\n", err.Field, err.Message)
			}
		}
		if len(artifacts.ValidationReport.Warnings) > 0 {
			fmt.Printf("   ⚠️  Alertas: %d\n", len(artifacts.ValidationReport.Warnings))
			for _, warn := range artifacts.ValidationReport.Warnings {
				fmt.Printf("     - %s: %s\n", warn.Field, warn.Message)
			}
		}
		if len(artifacts.ValidationReport.Suggestions) > 0 {
			fmt.Printf("   💡 Sugestões: %d\n", len(artifacts.ValidationReport.Suggestions))
			for _, sug := range artifacts.ValidationReport.Suggestions {
				fmt.Printf("     - %s\n", sug)
			}
		}
	}
}

// splitLines divide uma string em linhas
func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// saveArtifacts salva os artefatos em arquivos
func saveArtifacts(artifacts *protoagent.GeneratedArtifacts) error {
	// Criar diretório de saída
	if err := os.MkdirAll("./output/travel", 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório: %w", err)
	}

	// Salvar AGENT.md
	if artifacts.AgentConfig != nil {
		agentJSON, _ := json.MarshalIndent(artifacts.AgentConfig, "", "  ")
		if err := os.WriteFile("./output/travel/AGENT.json", agentJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar AGENT.json: %w", err)
		}
		
		// Também salvar como Markdown
		agentMD := fmt.Sprintf("# %s Agent\n\n%s\n", artifacts.AgentConfig.Name, artifacts.AgentConfig.Body)
		if err := os.WriteFile("./output/travel/AGENT.md", []byte(agentMD), 0644); err != nil {
			return fmt.Errorf("falha ao salvar AGENT.md: %w", err)
		}
		fmt.Println("\n📄 AGENT.json e AGENT.md salvos")
	}

	// Salvar schemas de banco de dados
	for i, schema := range artifacts.DatabaseSchemas {
		schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
		filename := fmt.Sprintf("./output/travel/schema_%d_%s.json", i, schema.Name)
		if err := os.WriteFile(filename, schemaJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar schema: %w", err)
		}
		
		// Gerar SQL DDL
		if schema.Type == "sql" && len(schema.Tables) > 0 {
			sqlDDL := generateSQLDDL(schema)
			sqlFilename := fmt.Sprintf("./output/travel/schema_%d_%s.sql", i, schema.Name)
			if err := os.WriteFile(sqlFilename, []byte(sqlDDL), 0644); err != nil {
				return fmt.Errorf("falha ao salvar SQL: %w", err)
			}
			fmt.Printf("📄 Schema %s salvo (JSON + SQL)\n", schema.Name)
		} else {
			fmt.Printf("📄 Schema %s salvo\n", schema.Name)
		}
	}

	// Salvar políticas OPA
	for i, policy := range artifacts.Policies {
		policyJSON, _ := json.MarshalIndent(policy, "", "  ")
		filename := fmt.Sprintf("./output/travel/policy_%d_%s.rego.json", i, policy.Name)
		if err := os.WriteFile(filename, policyJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar política: %w", err)
		}
		fmt.Printf("📄 Política %s salva (JSON)\n", policy.Name)

		// Salvar também o código Rego puro
		regoFilename := fmt.Sprintf("./output/travel/policy_%d_%s.rego", i, policy.Name)
		if err := os.WriteFile(regoFilename, []byte(policy.Rego), 0644); err != nil {
			return fmt.Errorf("falha ao salvar rego: %w", err)
		}
		fmt.Printf("📄 Código Rego %s salvo\n", policy.Name)
	}

	// Salvar interfaces
	if len(artifacts.Interfaces) > 0 {
		interfacesJSON, _ := json.MarshalIndent(artifacts.Interfaces, "", "  ")
		if err := os.WriteFile("./output/travel/interfaces.json", interfacesJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar interfaces: %w", err)
		}
		fmt.Println("📄 interfaces.json salvo")
	}

	// Salvar channels
	if len(artifacts.Channels) > 0 {
		channelsJSON, _ := json.MarshalIndent(artifacts.Channels, "", "  ")
		if err := os.WriteFile("./output/travel/channels.json", channelsJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar channels: %w", err)
		}
		fmt.Println("📄 channels.json salvo")
	}

	// Salvar skills
	if len(artifacts.Skills) > 0 {
		skillsJSON, _ := json.MarshalIndent(artifacts.Skills, "", "  ")
		if err := os.WriteFile("./output/travel/skills.json", skillsJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar skills: %w", err)
		}
		fmt.Println("📄 skills.json salvo")
		
		// Salvar código de cada skill
		for i, skill := range artifacts.Skills {
			skillFile := fmt.Sprintf("./output/travel/skill_%d_%s.go", i, skill.Name)
			if err := os.WriteFile(skillFile, []byte(skill.Code), 0644); err != nil {
				return fmt.Errorf("falha ao salvar código da skill: %w", err)
			}
		}
		fmt.Println("📄 Códigos das skills salvos")
	}

	// Salvar tools
	if len(artifacts.Tools) > 0 {
		toolsJSON, _ := json.MarshalIndent(artifacts.Tools, "", "  ")
		if err := os.WriteFile("./output/travel/tools.json", toolsJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar tools: %w", err)
		}
		fmt.Println("📄 tools.json salvo")
	}

	// Salvar configuração MCP
	if artifacts.MCPConfig != nil {
		mcpJSON, _ := json.MarshalIndent(artifacts.MCPConfig, "", "  ")
		if err := os.WriteFile("./output/travel/mcp_config.json", mcpJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar mcp_config: %w", err)
		}
		fmt.Println("📄 mcp_config.json salvo")
	}

	// Salvar relatório de validação
	if artifacts.ValidationReport != nil {
		reportJSON, _ := json.MarshalIndent(artifacts.ValidationReport, "", "  ")
		if err := os.WriteFile("./output/travel/validation_report.json", reportJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar validation report: %w", err)
		}
		fmt.Println("📄 validation_report.json salvo")
	}

	return nil
}

// generateSQLDDL gera DDL SQL a partir de um schema
func generateSQLDDL(schema protoagent.DatabaseSchema) string {
	var ddl string
	ddl += fmt.Sprintf("-- Schema: %s\n", schema.Name)
	ddl += fmt.Sprintf("-- Type: %s\n\n", schema.Type)
	
	for _, table := range schema.Tables {
		ddl += fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table.Name)
		
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
		
		ddl += joinStrings(columns, ",\n")
		ddl += "\n);\n\n"
		
		// Criar índices
		for _, idx := range table.Indexes {
			ddl += fmt.Sprintf("CREATE INDEX ON %s (%s);\n", table.Name, idx)
		}
	}
	
	return ddl
}

// joinStrings junta strings com um separador
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
