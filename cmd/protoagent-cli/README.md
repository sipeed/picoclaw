# ProtoAgent CLI

Interface de linha de comando para o ProtoAgent - ferramenta de prototipagem de comportamentos.

## Estrutura

```
cmd/protoagent-cli/
└── main.go            # CLI completa com comandos generate, validate, version e help
```

## Comandos

### generate

Gera artefatos a partir de um arquivo de requisitos JSON.

```bash
protoagent-cli generate requirements.json [opções]
```

**Opções:**
- `-o, --output <dir>` - Diretório de saída (padrão: ./output)
- `-w, --workspace <dir>` - Diretório do workspace (padrão: .)
- `--opa` - Habilitar geração de políticas OPA
- `--ai` - Habilitar geração assistida por IA
- `--dry-run` - Preview sem escrever arquivos
- `-v, --verbose` - Output detalhado

### validate

Valida um arquivo de requisitos.

```bash
protoagent-cli validate requirements.json
```

### version

Mostra informações de versão.

```bash
protoagent-cli version
```

### help

Mostra ajuda detalhada.

```bash
protoagent-cli help
```

## Exemplos

```bash
# Gerar artefatos com políticas OPA
protoagent-cli generate travel-experience-platform.json -o ./output --opa --verbose

# Validar requisitos
protoagent-cli validate cafeteria-loyalty-system.json

# Dry run (preview)
protoagent-cli generate requirements.json --dry-run --verbose
```

## Artefatos Gerados

O CLI gera os seguintes arquivos no diretório de saída:

- `AGENT.json` / `AGENT.md` - Configuração do agente
- `schema_*.json` / `schema_*.sql` - Schemas de banco de dados
- `policy_*.rego.json` / `policy_*.rego` - Políticas OPA
- `interfaces.json` - Definições de interfaces
- `channels.json` - Configurações de canais
- `skills.json` / `skill_*.go` - Skills geradas
- `tools.json` - Tools configuradas
- `mcp_config.json` - Configuração MCP
- `validation_report.json` - Relatório de validação

## Requisitos

- Go 1.21+ (para suporte a log/slog e slices)
- Arquivo de requisitos em formato JSON

## Build

```bash
go build -o protoagent-cli ./cmd/protoagent-cli
```
