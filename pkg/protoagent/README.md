# ProtoAgent - Behavior Prototyping Tool

ProtoAgent é uma ferramenta de prototipagem de comportamentos que transforma requisitos funcionais e não-funcionais em configurações de agentes, schemas de banco de dados, interfaces, canais de comunicação e políticas de segurança.

## Visão Geral

O ProtoAgent estende o PicoClaw para permitir que você descreva o comportamento desejado de um agente através de requisitos estruturados, e automaticamente gera:

- **Configurações de Agente** (AGENT.md)
- **Schemas de Banco de Dados** (SQL/NoSQL)
- **Interfaces** (API, Web UI)
- **Canais de Comunicação** (Telegram, Discord, Slack, Webhooks)
- **Políticas OPA** (Open Policy Agent para controle de acesso)
- **Skills** (Habilidades personalizadas)
- **Tools** (Ferramentas de integração)
- **Configuração MCP** (Model Context Protocol)

## Estrutura do Pacote

```
pkg/protoagent/
├── types.go           # Definições de tipos e estruturas de dados
├── engine.go          # Motor principal de processamento
├── generators.go      # Geradores de artefatos
└── policies.go        # Gerador de políticas OPA

cmd/protoagent-cli/
└── main.go            # CLI para uso por linha de comando
```

## Separação do ProtoAgent

O código do protoagente foi completamente separado do restante do agente:

- **Backend (pkg/protoagent/)**: Contém toda a lógica de processamento de requisitos e geração de artefatos
  - `types.go`: Definições de tipos e estruturas de dados
  - `engine.go`: Motor principal de processamento
  - `generators.go`: Geradores de artefatos (interfaces, schemas, channels, skills, tools)
  - `policies.go`: Gerador de políticas OPA

- **Frontend (CLI)**: Interface de linha de comando para interação com o protoagente
  - `cmd/protoagent-cli/main.go`: CLI completa com comandos generate, validate, version e help

## CLI de Linha de Comando

O ProtoAgent possui uma CLI dedicada para uso via terminal:

### Instalação

```bash
go build -o protoagent-cli ./cmd/protoagent-cli
```

### Uso

```bash
# Gerar artefatos a partir de requisitos
protoagent-cli generate requirements.json -o ./output --opa --verbose

# Validar arquivo de requisitos
protoagent-cli validate requirements.json

# Ver versão
protoagent-cli version

# Ajuda
protoagent-cli help
```

### Comandos

- `generate`: Gera todos os artefatos a partir de um arquivo de requisitos JSON
  - Opções: `-o/--output`, `-w/--workspace`, `--opa`, `--ai`, `--dry-run`, `-v/--verbose`

- `validate`: Valida um arquivo de requisitos sem gerar artefatos

- `version`: Mostra informações de versão

- `help`: Mostra ajuda detalhada

## Tipos de Requisitos

### Requisitos Funcionais

Descrevem **o que** o sistema deve fazer:

```yaml
functionalRequirements:
  - id: FR001
    type: action
    name: CreateUser
    description: Create a new user account
    inputs:
      - name: username
        type: string
        required: true
      - name: email
        type: string
        required: true
    interactionMethods:
      - api
      - ui
```

### Requisitos Não-Funcionais

Descrevem **restrições e atributos de qualidade**:

```yaml
nonFunctionalRequirements:
  - id: NFR001
    category: security
    name: Authentication
    description: All API calls must be authenticated
    constraints:
      auth_type: jwt
      token_expiry: 24h
```

## Métodos de Interação

- `api` - Integração via API REST/GraphQL
- `mcp` - Model Context Protocol
- `ui` - Interface de usuário (web/cli)
- `messaging` - Aplicativos de mensagem (Telegram, Discord, etc.)
- `webhook` - Webhooks para integrações
- `cli` - Interface de linha de comando
- `database` - Acesso direto ao banco de dados
- `file` - Operações com arquivos
- `eventbus` - Barramento de eventos

## Exemplo de Uso

```go
package main

import (
    "context"
    "github.com/sipeed/picoclaw/pkg/protoagent"
)

func main() {
    // Configurar o engine
    config := protoagent.EngineConfig{
        OutputDir: "./output",
        Workspace: "./workspace",
        EnableOPA: true,
        EnableAI:  false,
        DryRun:    false,
    }
    
    engine := protoagent.NewEngine(config)
    
    // Definir requisitos
    reqs := &protoagent.RequirementsDocument{
        Version:     "1.0.0",
        Name:        "Customer Support Bot",
        Description: "Automated customer support assistant",
        FunctionalRequirements: []protoagent.FunctionalRequirement{
            {
                ID:          "FR001",
                Type:        "action",
                Name:        "HandleTicket",
                Description: "Process customer support tickets",
                Inputs: []protoagent.ParameterDef{
                    {Name: "ticket_id", Type: "string", Required: true},
                    {Name: "message", Type: "string", Required: true},
                },
                InteractionMethods: []protoagent.InteractionMethod{
                    protoagent.InteractionMessaging,
                    protoagent.InteractionAPI,
                },
            },
        },
        SecurityRequirements: []protoagent.SecurityRequirement{
            {
                Roles:          []string{"admin", "agent", "customer"},
                Permissions:    []string{"read", "write", "resolve"},
                SecurityControls: []string{"authentication", "authorization"},
            },
        },
    }
    
    // Processar requisitos e gerar artefatos
    ctx := context.Background()
    artifacts, err := engine.ProcessRequirements(ctx, reqs)
    if err != nil {
        panic(err)
    }
    
    // Usar artefatos gerados
    if artifacts.AgentConfig != nil {
        // Salvar AGENT.md
    }
    
    for _, policy := range artifacts.Policies {
        // Salvar políticas OPA
    }
}
```

## Políticas OPA

O ProtoAgent gera automaticamente políticas Rego para Open Policy Agent baseadas nos requisitos de segurança:

### RBAC (Role-Based Access Control)

```rego
package authz.rbac

default allow = false

roles := {"admin", "user", "viewer"}

role_permissions := {
  "admin": {"read", "write", "delete", "admin"},
  "user": {"read", "write"},
  "viewer": {"read"}
}

allow {
  some role in input.user.roles
  some perm in role_permissions[role]
  perm == input.permission
}
```

### Controle de Acesso a Dados

```rego
package authz.data_access

default allow = false

allow {
  input.data_classification == "public"
}

allow {
  input.data_classification == "confidential"
  input.user.clearance_level >= 2
}
```

## Workflow de Desenvolvimento

1. **Definir Requisitos**: Crie um documento YAML/JSON com requisitos funcionais e não-funcionais
2. **Processar**: Execute o ProtoAgent para gerar artefatos
3. **Revisar**: Analise os artefatos gerados
4. **Customizar**: Ajuste conforme necessário
5. **Implantar**: Use os artefatos no seu workspace PicoClaw

## Integração com PicoClaw

Os artefatos gerados pelo ProtoAgent são compatíveis com a estrutura do PicoClaw:

- `AGENT.md` → Configuração do agente
- `skills/` → Habilidades personalizadas
- `workspace/memory/` → Esquemas de memória
- Políticas OPA → Controle de acesso

## Próximos Passos

- [ ] Suporte a provedores de IA para geração assistida
- [ ] Validação de políticas OPA com OPA CLI
- [ ] Templates customizáveis por domínio
- [ ] Export para Docker Compose/Kubernetes
- [ ] Interface web para definição de requisitos

## Licença

Mesma licença do PicoClaw original.
