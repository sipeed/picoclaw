package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/protoagent"
)

// Este exemplo demonstra como usar o ProtoAgent para gerar artefatos
// a partir de requisitos de um sistema de fidelização de cafeteria
func main() {
	// Carregar requisitos do arquivo JSON
	reqs, err := loadRequirements("examples/protoagent/cafeteria-loyalty-system.json")
	if err != nil {
		fmt.Printf("Erro ao carregar requisitos: %v\n", err)
		os.Exit(1)
	}

	// Configurar o engine do ProtoAgent
	config := protoagent.EngineConfig{
		OutputDir: "./output/cafeteria",
		Workspace: "./workspace",
		EnableOPA: true,
		EnableAI:  false,
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
	}

	if len(artifacts.DatabaseSchemas) > 0 {
		fmt.Printf("\n💾 Schemas de Banco de Dados: %d\n", len(artifacts.DatabaseSchemas))
		for _, schema := range artifacts.DatabaseSchemas {
			fmt.Printf("   - %s (%s)\n", schema.Name, schema.Type)
			if len(schema.Tables) > 0 {
				for _, table := range schema.Tables {
					fmt.Printf("     Tabela: %s (%d colunas)\n", table.Name, len(table.Columns))
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
				fmt.Printf("     Telas: %d\n", len(iface.Screens))
			}
		}
	}

	if len(artifacts.Channels) > 0 {
		fmt.Printf("\n📱 Canais de Comunicação: %d\n", len(artifacts.Channels))
		for _, channel := range artifacts.Channels {
			fmt.Printf("   - %s (%s) - Habilitado: %v\n", channel.Name, channel.Type, channel.Enabled)
			if len(channel.Commands) > 0 {
				fmt.Printf("     Comandos: %d\n", len(channel.Commands))
			}
		}
	}

	if len(artifacts.Policies) > 0 {
		fmt.Printf("\n🔐 Políticas OPA: %d\n", len(artifacts.Policies))
		for _, policy := range artifacts.Policies {
			fmt.Printf("   - %s (%s)\n", policy.Name, policy.Package)
			fmt.Printf("     Descrição: %s\n", policy.Description)
		}
	}

	if len(artifacts.Skills) > 0 {
		fmt.Printf("\n🎯 Skills: %d\n", len(artifacts.Skills))
		for _, skill := range artifacts.Skills {
			fmt.Printf("   - %s\n", skill.Name)
			fmt.Printf("     Descrição: %s\n", skill.Description)
		}
	}

	if len(artifacts.Tools) > 0 {
		fmt.Printf("\n🔧 Tools: %d\n", len(artifacts.Tools))
		for _, tool := range artifacts.Tools {
			fmt.Printf("   - %s (%s)\n", tool.Name, tool.Type)
		}
	}

	if artifacts.MCPConfig != nil && len(artifacts.MCPConfig.Servers) > 0 {
		fmt.Printf("\n🔌 Servidores MCP: %d\n", len(artifacts.MCPConfig.Servers))
		for _, server := range artifacts.MCPConfig.Servers {
			fmt.Printf("   - %s (%s)\n", server.Name, server.Type)
		}
	}

	if artifacts.ValidationReport != nil {
		fmt.Printf("\n✅ Validação: %v\n", artifacts.ValidationReport.Valid)
		if len(artifacts.ValidationReport.Errors) > 0 {
			fmt.Printf("   Erros: %d\n", len(artifacts.ValidationReport.Errors))
		}
		if len(artifacts.ValidationReport.Warnings) > 0 {
			fmt.Printf("   Alertas: %d\n", len(artifacts.ValidationReport.Warnings))
		}
		if len(artifacts.ValidationReport.Suggestions) > 0 {
			fmt.Printf("   Sugestões: %d\n", len(artifacts.ValidationReport.Suggestions))
			for _, sug := range artifacts.ValidationReport.Suggestions {
				fmt.Printf("     💡 %s\n", sug)
			}
		}
	}
}

// saveArtifacts salva os artefatos em arquivos
func saveArtifacts(artifacts *protoagent.GeneratedArtifacts) error {
	// Criar diretório de saída
	if err := os.MkdirAll("./output/cafeteria", 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório: %w", err)
	}

	// Salvar AGENT.md
	if artifacts.AgentConfig != nil {
		agentJSON, _ := json.MarshalIndent(artifacts.AgentConfig, "", "  ")
		if err := os.WriteFile("./output/cafeteria/AGENT.json", agentJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar AGENT.json: %w", err)
		}
		fmt.Println("\n📄 AGENT.json salvo")
	}

	// Salvar schemas de banco de dados
	for i, schema := range artifacts.DatabaseSchemas {
		schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
		filename := fmt.Sprintf("./output/cafeteria/schema_%d_%s.json", i, schema.Name)
		if err := os.WriteFile(filename, schemaJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar schema: %w", err)
		}
		fmt.Printf("📄 Schema %s salvo\n", schema.Name)
	}

	// Salvar políticas OPA
	for i, policy := range artifacts.Policies {
		policyJSON, _ := json.MarshalIndent(policy, "", "  ")
		filename := fmt.Sprintf("./output/cafeteria/policy_%d_%s.rego.json", i, policy.Name)
		if err := os.WriteFile(filename, policyJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar política: %w", err)
		}
		fmt.Printf("📄 Política %s salva\n", policy.Name)

		// Salvar também o código Rego puro
		regoFilename := fmt.Sprintf("./output/cafeteria/policy_%d_%s.rego", i, policy.Name)
		if err := os.WriteFile(regoFilename, []byte(policy.Rego), 0644); err != nil {
			return fmt.Errorf("falha ao salvar rego: %w", err)
		}
		fmt.Printf("📄 Código Rego %s salvo\n", policy.Name)
	}

	// Salvar channels
	if len(artifacts.Channels) > 0 {
		channelsJSON, _ := json.MarshalIndent(artifacts.Channels, "", "  ")
		if err := os.WriteFile("./output/cafeteria/channels.json", channelsJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar channels: %w", err)
		}
		fmt.Println("📄 channels.json salvo")
	}

	// Salvar relatório de validação
	if artifacts.ValidationReport != nil {
		reportJSON, _ := json.MarshalIndent(artifacts.ValidationReport, "", "  ")
		if err := os.WriteFile("./output/cafeteria/validation_report.json", reportJSON, 0644); err != nil {
			return fmt.Errorf("falha ao salvar validation report: %w", err)
		}
		fmt.Println("📄 validation_report.json salvo")
	}

	return nil
}
