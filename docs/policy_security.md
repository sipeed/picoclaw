# Política de Segurança com Open Policy Agent (OPA)

## Visão Geral

O PicoClaw agora suporta avaliação de políticas de segurança baseadas em regras configuráveis. Este sistema permite que você defina quais ações, ferramentas e intenções o agente pode executar, tornando-o mais previsível e seguro contra injeção de código e outros ataques.

## Arquitetura

O sistema de políticas funciona em três níveis:

1. **Identificação da Intenção**: Após o LLM identificar a intenção do usuário
2. **Avaliação do Plano de Ação**: Antes de executar qualquer ação planejada
3. **Avaliação de Chamadas de Ferramentas**: Antes de cada chamada de ferramenta individual

## Estrutura do Arquivo de Política

O arquivo de política `.policy.yml` deve estar localizado no mesmo diretório que o `config.json` (geralmente `~/.picoclaw/`).

### Exemplo Básico

```yaml
# ~/.picoclaw/.policy.yml
enabled: true
timeout: 5
default_allow: false

# Ferramentas permitidas
allowed_tools:
  - "web_search"
  - "web_fetch"
  - "message"

# Ferramentas negadas
denied_tools:
  - "bash"
  - "shell"
  - "exec"

# Intenções permitidas
allowed_intents:
  - "search"
  - "fetch"
  - "communicate"

# Ferramentas que requerem aprovação
require_approval:
  - "spawn"
  - "install_skill"
```

## Configuração

### Opções Principais

| Campo | Tipo | Descrição | Padrão |
|-------|------|-----------|--------|
| `enabled` | boolean | Habilita ou desabilita a avaliação de políticas | `false` |
| `timeout` | int | Timeout em segundos para avaliação | `5` |
| `default_allow` | boolean | Comportamento padrão quando nenhuma regra corresponde | `true` |

### Listas de Controle

#### allowed_tools
Lista branca de ferramentas que podem ser usadas. Se especificada, apenas essas ferramentas serão permitidas.

```yaml
allowed_tools:
  - "web_search"
  - "web_fetch"
  - "message"
  - "send_file"
```

#### denied_tools
Lista negra de ferramentas que são explicitamente proibidas.

```yaml
denied_tools:
  - "bash"
  - "shell"
  - "exec"
  - "system"
  - "rm"
```

#### allowed_intents / denied_intents
Controle baseado em intenções identificadas pelo LLM.

```yaml
allowed_intents:
  - "search"
  - "fetch"
  - "read_file"

denied_intents:
  - "execute_code"
  - "modify_system"
  - "delete_files"
```

#### require_approval
Ferramentas que requerem aprovação explícita do usuário antes da execução.

```yaml
require_approval:
  - "spawn"
  - "subagent"
  - "install_skill"
  - "mcp"
```

### Padrões de Argumentos

Você pode definir padrões regex para detectar operações perigosas nos argumentos das ferramentas:

```yaml
argument_patterns:
  - tool: "bash"
    argument: "command"
    pattern: "^(rm|sudo|chmod|chown|dd|mkfs)"
    action: "deny"
    reason: "Comandos destrutivos não são permitidos"
  
  - tool: "bash"
    argument: "command"
    pattern: "(wget|curl).*\\|.*sh"
    action: "deny"
    reason: "Piping de scripts remotos para shell não é permitido"
  
  - tool: "send_file"
    argument: "path"
    pattern: "^(/etc/|/root/|\\.ssh/)"
    action: "deny"
    reason: "Acesso a diretórios sensíveis não é permitido"
```

### Regras Customizadas

Regras permitem lógica mais complexa com condições:

```yaml
rules:
  - id: "block-dangerous-shells"
    description: "Bloqueia comandos shell perigosos"
    tools:
      - "bash"
      - "shell"
    action: "deny"
    priority: 100
  
  - id: "allow-safe-search"
    description: "Permite operações de busca web"
    tools:
      - "web_search"
      - "web_fetch"
    action: "allow"
    priority: 50
  
  - id: "require-approval-for-spawn"
    description: "Requer aprovação para criar subagentes"
    tools:
      - "spawn"
      - "subagent"
    action: "require_approval"
    priority: 75
```

#### Campos da Regra

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `id` | string | Identificador único da regra |
| `description` | string | Descrição da regra |
| `condition` | string | Condição opcional (ex: `tool.name == bash`) |
| `tools` | []string | Lista de ferramentas que a regra afeta |
| `intents` | []string | Lista de intenções que a regra afeta |
| `action` | string | Ação: `allow`, `deny`, ou `require_approval` |
| `priority` | int | Prioridade da regra (maior = primeiro) |

### Condições Suportadas

As condições suportam os seguintes operadores:

- `==` - Igualdade
- `!=` - Diferença
- `contains` - Contém substring
- `starts_with` - Começa com
- `ends_with` - Termina com
- `>`, `<`, `>=`, `<=` - Comparação numérica

Exemplos:

```yaml
condition: "tool.name == bash"
condition: "intent.confidence > 0.8"
condition: "intent.type contains execute"
```

## Integração com o Agente

### No Código Go

```go
import "github.com/sipeed/picoclaw/pkg/policy"

// Criar avaliador de políticas
evaluator, err := policy.NewEvaluator(cfg, configPath)
if err != nil {
    logger.Error("Failed to create policy evaluator", err)
}

// Avaliar intenção
intent := policy.Intent{
    Type: "execute_code",
    Description: "User wants to run a shell command",
    Confidence: 0.95,
}

result, err := evaluator.EvaluateIntent(ctx, intent)
if err != nil {
    // Erro na avaliação
}

if !result.Allowed {
    // Bloquear ação
    logger.Warn("Action blocked by policy", result.Reason)
    return fmt.Errorf("action not allowed: %s", result.Reason)
}

// Avaliar chamada de ferramenta
toolCall := policy.ToolCall{
    Name: "bash",
    Arguments: map[string]interface{}{
        "command": "rm -rf /",
    },
}

result, err = evaluator.EvaluateToolCall(ctx, toolCall)
if !result.Allowed {
    // Bloquear chamada de ferramenta
    return fmt.Errorf("tool call not allowed: %s", result.Reason)
}

// Avaliar plano de ação
plan := policy.ActionPlan{
    Actions: []policy.Action{
        {
            Type: "tool_call",
            Tool: "web_search",
            Arguments: map[string]interface{}{
                "query": "weather",
            },
        },
    },
}

result, err = evaluator.EvaluateActionPlan(ctx, plan)
if !result.Allowed {
    // Bloquear plano inteiro
    return fmt.Errorf("action plan not allowed: %s", result.Reason)
}
```

### Pontos de Integração no AgentLoop

Os pontos recomendados para integração são:

1. **Após identificação da intenção** (no início do processamento da mensagem)
2. **Antes de executar o plano de ações** (após o LLM retornar tool_calls)
3. **Antes de cada chamada de ferramenta** (no hook BeforeTool)

Exemplo de integração no hook BeforeTool:

```go
func (al *AgentLoop) processToolCall(ctx context.Context, tc providers.ToolCall) error {
    // Converter para formato de política
    toolCall := policy.ToolCall{
        Name:      tc.Function.Name,
        Arguments: tc.Function.Arguments,
        Channel:   ts.opts.Channel,
        ChatID:    ts.opts.ChatID,
        SenderID:  ts.opts.SenderID,
    }
    
    // Avaliar política
    if al.policyEvaluator != nil {
        result, err := al.policyEvaluator.EvaluateToolCall(ctx, toolCall)
        if err != nil {
            logger.WarnCF("policy", "Policy evaluation failed", map[string]any{
                "error": err.Error(),
            })
        } else if !result.Allowed {
            // Verificar se requer aprovação
            if requiresApproval, ok := result.Data["requires_approval"].(bool); ok && requiresApproval {
                // Solicitar aprovação do usuário
                approved := al.requestUserApproval(tc)
                if !approved {
                    return fmt.Errorf("tool call not approved by user")
                }
            } else {
                // Bloquear completamente
                return fmt.Errorf("tool call blocked by policy: %s", result.Reason)
            }
        }
    }
    
    // Continuar com execução normal...
}
```

## Recarregamento de Políticas

As políticas podem ser recarregadas sem reiniciar o agente:

```go
// Recarregar políticas do disco
err := evaluator.Reload()
if err != nil {
    logger.Error("Failed to reload policies", err)
}

// Ou atualizar configuração programaticamente
newConfig := policy.Config{
    Enabled: true,
    DefaultAllow: false,
    DeniedTools: []string{"bash", "shell"},
}
err = evaluator.UpdateConfig(newConfig)
```

## Melhores Práticas

### 1. Defense in Depth

Use múltiplas camadas de proteção:

```yaml
# Camada 1: Lista negra de ferramentas perigosas
denied_tools:
  - "bash"
  - "shell"
  - "exec"

# Camada 2: Padrões de argumentos
argument_patterns:
  - tool: ".*"
    argument: ".*"
    pattern: "(rm -rf|chmod 777|sudo)"
    action: "deny"

# Camada 3: Requer aprovação para operações sensíveis
require_approval:
  - "spawn"
  - "install_skill"
```

### 2. Default Deny

Para máxima segurança, use `default_allow: false`:

```yaml
enabled: true
default_allow: false

# Lista branca explícita
allowed_tools:
  - "web_search"
  - "web_fetch"
  - "message"
```

### 3. Logging e Auditoria

Sempre logue decisões de política:

```go
if !result.Allowed {
    logger.InfoCF("policy", "Action blocked", map[string]any{
        "tool": toolCall.Name,
        "reason": result.Reason,
        "channel": toolCall.Channel,
        "chat_id": toolCall.ChatID,
        "sender_id": toolCall.SenderID,
    })
}
```

### 4. Teste suas Políticas

Teste políticas em ambiente controlado antes de produção:

```bash
# Usar modo dry-run (se implementado)
picoclaw --policy-dry-run

# Ou habilitar logging verbose
export PICOCLAW_LOG_LEVEL=debug
```

## Exemplos de Cenários

### Cenário 1: Agente Somente Leitura

Permitir apenas operações de leitura e comunicação:

```yaml
enabled: true
default_allow: false

allowed_tools:
  - "web_search"
  - "web_fetch"
  - "message"
  - "send_file"
  - "load_image"

allowed_intents:
  - "search"
  - "fetch"
  - "communicate"
  - "read_file"
```

### Cenário 2: Ambiente de Desenvolvimento

Permitir mais ferramentas mas com aprovações:

```yaml
enabled: true
default_allow: true

denied_tools:
  - "rm"
  - "delete"
  - "format"

require_approval:
  - "bash"
  - "shell"
  - "spawn"
  - "install_skill"

argument_patterns:
  - tool: "bash"
    argument: "command"
    pattern: "^(rm|sudo|dd|mkfs)"
    action: "deny"
    reason: "Comandos destrutivos requerem aprovação manual"
```

### Cenário 3: Proteção Contra Injeção

Bloquear padrões comuns de injeção:

```yaml
enabled: true
default_allow: true

argument_patterns:
  # Bloquear download e execução de scripts
  - tool: "bash"
    argument: "command"
    pattern: "(wget|curl|fetch).*(\\|.*sh|\\|.*bash|&&.*sh)"
    action: "deny"
    reason: "Download e execução de scripts remotos proibido"
  
  # Bloquear codificação base64 (técnica comum de evasão)
  - tool: "bash"
    argument: "command"
    pattern: "base64.*-d.*\\|"
    action: "deny"
    reason: "Decodificação base64 com pipe proibida"
  
  # Bloquear avaliações dinâmicas
  - tool: "bash"
    argument: "command"
    pattern: "(eval|exec)\\("
    action: "deny"
    reason: "Avaliação dinâmica de código proibida"
```

## Troubleshooting

### Política não está sendo aplicada

1. Verifique se `enabled: true`
2. Confirme que o arquivo `.policy.yml` está no diretório correto
3. Verifique os logs do agente por erros de parsing
4. Use `default_allow: false` para testar se as regras estão funcionando

### Regras não correspondem como esperado

1. Verifique a prioridade das regras (maior número = executado primeiro)
2. Teste padrões regex separadamente
3. Use logging para depurar qual regra está sendo avaliada

### Performance lenta

1. Aumente o timeout se necessário
2. Reduza o número de padrões regex complexos
3. Use listas de controle (allowed/denied) em vez de muitas regras

## Referências

- [Exemplo de Configuração](pkg/policy/policy.example.yml)
- [Documentação de Segurança](docs/security_configuration.md)
- [Hooks do Agente](pkg/agent/hooks.go)
