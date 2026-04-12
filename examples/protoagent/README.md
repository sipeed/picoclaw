# Exemplos de Uso do ProtoAgent

Este diretório contém exemplos funcionais de como utilizar o ProtoAgent para prototipar diferentes tipos de sistemas.

## 📋 Visão Geral

O ProtoAgent transforma requisitos funcionais e não-funcionais em artefatos prontos para uso:
- Configurações de agente (AGENT.md)
- Schemas de banco de dados (SQL/NoSQL)
- Interfaces (API, Web UI)
- Canais de comunicação (Telegram, Discord, etc.)
- Políticas OPA (Open Policy Agent)
- Skills personalizadas
- Tools de integração
- Configuração MCP

## 🎯 Exemplos Disponíveis

### a) Sistema de Fidelização de Cafeteria

**Arquivo:** `cafeteria-loyalty-system.json`

Um sistema onde clientes acumulam "grãos" baseados no consumo e podem trocar por produtos.

#### Características Principais:
- **Canal Telegram**: Clientes consultam saldo via bot
- **Atendentes**: Gerenciam contas, adicionam/consumem grãos
- **Catálogo de Produtos**: Resgate com grãos acumulados
- **Histórico de Transações**: Auditoria completa

#### Requisitos Funcionais:
1. `ManageCustomer` - Cadastro de clientes
2. `AddGrains` - Adicionar grãos por compra
3. `RedeemGrains` - Resgatar produtos
4. `ConsultBalance` - Consultar saldo
5. `ManageProducts` - Gerenciar catálogo
6. `TelegramBalanceQuery` - Consulta via Telegram
7. `TransactionHistory` - Histórico de transações

#### Requisitos Não-Funcionais:
- Autenticação JWT para atendentes
- Resposta em < 2 segundos para consultas
- Registro imutável de transações
- Suporte a 50 usuários concorrentes

#### Papéis e Permissões:
- **admin**: Todas as operações
- **attendant**: Ler, escrever, adicionar/consumir grãos
- **customer**: Apenas consultar próprio saldo

#### Como Executar:

```bash
cd /workspace
go run examples/protoagent/example_cafeteria.go
```

#### Artefatos Gerados:
- `output/cafeteria/AGENT.json` - Configuração do agente
- `output/cafeteria/schema_*.json` - Schemas de banco de dados
- `output/cafeteria/schema_*.sql` - DDL SQL
- `output/cafeteria/policy_*.rego` - Políticas OPA
- `output/cafeteria/channels.json` - Configuração Telegram
- `output/cafeteria/validation_report.json` - Relatório de validação

---

### b) Plataforma de Experiências de Viagem

**Arquivo:** `travel-experience-platform.json`

Plataforma onde viajantes compartilham relatos com mídias. O sistema atua como editor, filtrando dados pessoais, enriquecendo conteúdo e publicando em redes sociais.

#### Características Principais:
- **Submissão de Relatos**: Viajantes enviam histórias com fotos/vídeos
- **Filtragem de PII**: Detecção automática de dados pessoais
- **Enriquecimento com IA**: Solicita informações adicionais aos autores
- **Geração de Artigos**: Cria conteúdo formatado para redes sociais
- **Curadoria**: Aprovação, rejeição ou solicitação de atualizações
- **Publicação Automática**: Integração com APIs de redes sociais
- **Monitoramento de Engajamento**: Analytics das publicações

#### Requisitos Funcionais:
1. `SubmitTravelStory` - Submeter relato com mídias
2. `FilterPersonalData` - Filtrar informações pessoais (PII)
3. `EnrichStoryContent` - Coletar informações adicionais
4. `GenerateSocialMediaArticle` - Criar artigos para redes sociais
5. `CurateContent` - Curadoria (aprovar/rejeitar/atualizar)
6. `PublishToSocialMedia` - Publicar em Instagram, Facebook, etc.
7. `MonitorEngagement` - Acompanhar likes, shares, comments
8. `ManageTravelerProfile` - Perfis de viajantes
9. `ExternalAPIIntegration` - Integrar com APIs externas (clima, mapas)
10. `UpdateStory` - Atualizar relatos existentes

#### Requisitos Não-Funcionais:
- **Privacidade**: Detecção e remoção automática de PII (GDPR compliant)
- **Moderação**: Conteúdo revisado antes de publicação
- **Performance**: Processamento em até 30 segundos
- **Persistência**: Backup diário, histórico de versões (5 anos)
- **Escalabilidade**: Armazenamento cloud com CDN
- **Disponibilidade**: Fallback para APIs indisponíveis

#### Papéis e Permissões:
- **admin**: Todas as operações
- **curator**: Aprovar, rejeitar, publicar conteúdo
- **traveler**: Criar/editar próprios relatos
- **viewer**: Apenas leitura

#### Recursos de IA:
- Detecção de PII (informações pessoais identificáveis)
- Enriquecimento de conteúdo
- Geração automática de artigos
- Assistência na curadoria

#### Como Executar:

```bash
cd /workspace
go run examples/protoagent/example_travel.go
```

#### Artefatos Gerados:
- `output/travel/AGENT.json` + `AGENT.md` - Configuração do agente
- `output/travel/schema_*.json` + `.sql` - Schemas de banco de dados
- `output/travel/interfaces.json` - Definições de API e UI
- `output/travel/policy_*.rego` - Políticas OPA (RBAC, PII, moderação)
- `output/travel/channels.json` - Canais de comunicação
- `output/travel/skills.json` + `.go` - Skills personalizadas
- `output/travel/tools.json` - Ferramentas de integração
- `output/travel/mcp_config.json` - Configuração MCP
- `output/travel/validation_report.json` - Relatório de validação

---

## 🔧 Estrutura dos Arquivos de Requisitos

Os arquivos JSON seguem esta estrutura:

```json
{
  "version": "1.0.0",
  "name": "Nome do Sistema",
  "description": "Descrição detalhada",
  
  "functionalRequirements": [
    {
      "id": "FR001",
      "type": "action|operation|resource|actor",
      "name": "NomeDaOperacao",
      "description": "Descrição do que faz",
      "inputs": [...],
      "outputs": [...],
      "preconditions": [...],
      "postconditions": [...],
      "interactionMethods": ["api", "ui", "messaging", "webhook", "mcp"]
    }
  ],
  
  "nonFunctionalRequirements": [
    {
      "id": "NFR001",
      "category": "security|performance|reliability|scalability",
      "name": "NomeRequisito",
      "description": "Descrição",
      "constraints": {...},
      "metrics": [...]
    }
  ],
  
  "securityRequirements": [
    {
      "roles": ["admin", "user"],
      "permissions": ["read", "write", "delete"],
      "authorizations": ["role_based"],
      "securityControls": ["authentication", "authorization"],
      "dataClassification": "internal"
    }
  ],
  
  "performanceRequirements": [...],
  "metadata": {...}
}
```

## 🚀 Métodos de Interação Suportados

- `api` - Integração via API REST/GraphQL
- `mcp` - Model Context Protocol
- `ui` - Interface de usuário (web/cli)
- `messaging` - Aplicativos de mensagem (Telegram, Discord, Slack)
- `webhook` - Webhooks para integrações
- `cli` - Interface de linha de comando
- `database` - Acesso direto ao banco de dados
- `file` - Operações com arquivos
- `eventbus` - Barramento de eventos

## 📊 Tipos de Artefatos Gerados

### 1. Configuração de Agente (AGENT.md)
Define o comportamento, tools, skills e instruções do agente.

### 2. Schemas de Banco de Dados
- **SQL**: Tabelas com colunas, tipos, chaves primárias, índices
- **NoSQL**: Collections com schemas JSON
- **Migrações**: Scripts de criação/atualização

### 3. Interfaces
- **API**: Endpoints REST com métodos, paths, autenticação
- **Web UI**: Telas, rotas, componentes, ações

### 4. Canais de Comunicação
- **Telegram**: Bot com comandos e handlers
- **Discord/Slack**: Integrações similares
- **Webhooks**: Endpoints para recebimento de eventos

### 5. Políticas OPA (Open Policy Agent)
- **RBAC**: Controle de acesso baseado em papéis
- **Autorização**: Políticas customizadas
- **Acesso a Dados**: Classificação e controle de sensibilidade

### 6. Skills
Código Go personalizado para operações específicas do domínio.

### 7. Tools
Configurações para ferramentas de integração (HTTP, database, filesystem).

### 8. Configuração MCP
Servidores Model Context Protocol para contexto adicional.

## 🛠️ Workflow de Desenvolvimento

1. **Definir Requisitos**: Crie um arquivo JSON descrevendo funcionalidades e restrições
2. **Executar ProtoAgent**: Rode o exemplo correspondente
3. **Revisar Artefatos**: Analise os arquivos gerados em `output/`
4. **Customizar**: Ajuste conforme necessidades específicas
5. **Implantar**: Copie artefatos para o workspace do PicoClaw

## 📝 Próximos Passos Sugeridos

Após gerar os artefatos:

1. **Configurar Credenciais**:
   - Token do Telegram em `channels.json`
   - URLs de APIs externas
   - Connection string do banco de dados

2. **Implementar Skills**:
   - Complete o código das skills geradas
   - Adicione lógica de negócio específica

3. **Testar Políticas OPA**:
   ```bash
   opa eval -i input.json -d policy.rego "authz.rbac.allow"
   ```

4. **Integrar com PicoClaw**:
   - Copie `AGENT.md` para `workspace/`
   - Coloque skills em `workspace/skills/`
   - Configure channels no config do PicoClaw

## 📞 Suporte

Para mais informações, consulte:
- `pkg/protoagent/README.md` - Documentação completa do ProtoAgent
- `docs/configuration.md` - Configuração do PicoClaw
- `examples/` - Outros exemplos de uso
