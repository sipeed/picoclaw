# Saída Esperada dos Exemplos ProtoAgent

Este documento descreve a saída esperada ao executar os exemplos do ProtoAgent.

## 📦 Exemplo A: Sistema de Fidelização de Cafeteria

### Comando de Execução
```bash
go run examples/protoagent/example_cafeteria.go
```

### Saída Esperada no Terminal

```
📦 Resumo dos Artefatos Gerados:
==================================================

🤖 Agente: Cafeteria Loyalty System
   Descrição: Sistema de fidelização de cafeteria onde clientes acumulam 'grãos'...
   Tools: [api_client message_handler database_tool]

💾 Schemas de Banco de Dados: 3
   - Customer (sql)
     Tabela: customers (6 colunas)
       - id: uuid [PK] [NOT NULL]
       - created_at: timestamp [DEFAULT CURRENT_TIMESTAMP]
       - updated_at: timestamp
       - name: varchar(255) [NOT NULL]
       - phone: varchar(255) [NOT NULL]
       - email: varchar(255)
   - GrainTransaction (sql)
     Tabela: graintransactions (8 colunas)
       - id: uuid [PK]
       - customer_id: uuid [NOT NULL]
       - amount: integer [NOT NULL]
       - type: varchar(50) [NOT NULL]
       - balance_after: integer [NOT NULL]
       - attendant_id: uuid [NOT NULL]
       - created_at: timestamp
   - Product (sql)
     Tabela: products (7 colunas)
       - id: uuid [PK]
       - name: varchar(255) [NOT NULL]
       - grains_cost: integer [NOT NULL]
       - quantity: integer [NOT NULL]
       - category: varchar(100)
       - active: boolean [DEFAULT true]

🖥️ Interfaces: 2
   - API (api)
     Endpoints: 7
       [POST] /api/v1/managecustomer - Gerenciar cadastro de clientes...
       [POST] /api/v1/addgrains - Adicionar grãos à conta do cliente...
       [POST] /api/v1/redeemgrains - Consumir grãos para resgatar produtos...
       [POST] /api/v1/consultbalance - Consultar saldo de grãos...
       [POST] /api/v1/manageproducts - Gerenciar catálogo de produtos...
       [POST] /api/v1/telegrambalancequery - Permitir consulta via Telegram...
       [POST] /api/v1/transactionhistory - Listar histórico de transações...
   - Web UI (web)
     Telas: 7
       📱 ManageCustomer (/managecustomer)
         Componentes: 4
       📱 AddGrains (/addgrains)
         Componentes: 4
       ...

📱 Canais de Comunicação: 1
   - telegram (telegram) - Habilitado: true
     Configuração:
       token: ${TELEGRAM_BOT_TOKEN}

🔐 Políticas OPA: 3
   - rbac_policy (authz.rbac)
     Descrição: Role-Based Access Control policy
     Código Rego (15 linhas):
       package authz.rbac
       
       # Auto-generated RBAC policy from requirements
       
       # Default deny
       default allow = false
       ... (10 linhas restantes)
   - authentication_policy (authz.custom)
     Descrição: Todos os atendentes devem ser autenticados...
   - data_access_policy (authz.data_access)
     Descrição: Data access control based on classification levels

🎯 Skills: 2
   - addgrains_skill
     Descrição: Adicionar grãos à conta do cliente baseado na compra
     Triggers: [AddGrains]
   - redeemgrains_skill
     Descrição: Consumir grãos para resgatar produtos
     Triggers: [RedeemGrains]

🔧 Tools: 4
   - api_tool (custom)
     Descrição: Tool for api interactions
     Configuração:
       type: http
       base_url: ${API_BASE_URL}
   - message_handler_tool (custom)
     Descrição: Tool for messaging interactions
   - database_tool (custom)
     Descrição: Tool for database interactions
     Configuração:
       type: database
       driver: postgres
       dsn: ${DATABASE_URL}
   - webhook_handler_tool (custom)
     Descrição: Tool for webhook interactions

✅ Validação: true
   Sugestões: 1
     💡 Consider testing OPA policies with: opa eval -i input.json -d policy.rego

📄 AGENT.json salvo
📄 Schema Customer salvo (JSON + SQL)
📄 Schema GrainTransaction salvo (JSON + SQL)
📄 Schema Product salvo (JSON + SQL)
📄 Política rbac_policy salva (JSON)
📄 Código Rego rbac_policy salvo
📄 Política authentication_policy salva (JSON)
📄 Código Rego authentication_policy salvo
📄 Política data_access_policy salva (JSON)
📄 Código Rego data_access_policy salvo
📄 channels.json salvo
📄 skills.json salvo
📄 Códigos das skills salvos
📄 tools.json salvo
📄 validation_report.json salvo

✅ Artefatos gerados com sucesso!
```

### Arquivos Gerados em `output/cafeteria/`

```
output/cafeteria/
├── AGENT.json                          # Configuração do agente em JSON
├── AGENT.md                            # Configuração do agente em Markdown
├── schema_0_customer.json              # Schema do cliente em JSON
├── schema_0_customer.sql               # DDL SQL da tabela customers
├── schema_1_graintransaction.json      # Schema de transações em JSON
├── schema_1_graintransaction.sql       # DDL SQL da tabela graintransactions
├── schema_2_product.json               # Schema de produtos em JSON
├── schema_2_product.sql                # DDL SQL da tabela products
├── policy_0_rbac_policy.rego.json      # Política RBAC em JSON
├── policy_0_rbac_policy.rego           # Política RBAC em Rego puro
├── policy_1_authentication_policy.rego.json
├── policy_1_authentication_policy.rego
├── policy_2_data_access_policy.rego.json
├── policy_2_data_access_policy.rego
├── channels.json                       # Configuração dos canais
├── skills.json                         # Definições das skills
├── skill_0_addgrains_skill.go          # Código Go da skill AddGrains
├── skill_1_redeemgrains_skill.go       # Código Go da skill RedeemGrains
├── tools.json                          # Definições das tools
└── validation_report.json              # Relatório de validação
```

### Exemplo de Código Rego Gerado (policy_0_rbac_policy.rego)

```rego
package authz.rbac

# Auto-generated RBAC policy from requirements

# Default deny
default allow = false

# Role definitions
roles := {
  "admin",
  "attendant",
  "customer",
}

# Permission definitions
permissions := {
  "read",
  "write",
  "delete",
  "redeem",
  "add_grains",
}

# Role-permission mapping
role_permissions := {
  "admin": {"read", "write", "delete", "admin"},
  "attendant": {"read", "write", "add_grains", "redeem"},
  "customer": {"read"}
}

# Allow if user has required permission
allow {
  some role in input.user.roles
  some perm in role_permissions[role]
  perm == input.permission
}

# Admin bypass
allow {
  some role in input.user.roles
  role == "admin"
}
```

### Exemplo de SQL Gerado (schema_0_customer.sql)

```sql
-- Schema: Customer
-- Type: sql

CREATE TABLE IF NOT EXISTS customers (
  id uuid PRIMARY KEY,
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp,
  name varchar(255) NOT NULL,
  phone varchar(255) NOT NULL,
  email varchar(255)
);

CREATE TABLE IF NOT EXISTS graintransactions (
  id uuid PRIMARY KEY,
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp,
  customer_id uuid NOT NULL,
  amount integer NOT NULL,
  type varchar(50) NOT NULL,
  balance_after integer NOT NULL,
  attendant_id uuid NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
  id uuid PRIMARY KEY,
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp,
  name varchar(255) NOT NULL,
  grains_cost integer NOT NULL,
  quantity integer NOT NULL,
  category varchar(100),
  active boolean DEFAULT true
);
```

---

## 📦 Exemplo B: Plataforma de Experiências de Viagem

### Comando de Execução
```bash
go run examples/protoagent/example_travel.go
```

### Saída Esperada no Terminal

```
📦 Resumo dos Artefatos Gerados:
==================================================

🤖 Agente: Travel Experience Platform
   Descrição: Plataforma de experiências de viagem onde viajantes compartilham...
   Tools: [api_client webhook_handler file_tool]
   Skills: [filterpersonaldata_skill enrichstorycontent_skill curatecontent_skill]

💾 Schemas de Banco de Dados: 4
   - TravelStory (nosql)
     Coleção: travel_stories
       Schema: {title: string, content: text, status: string, ...}
   - TravelerProfile (nosql)
     Coleção: traveler_profiles
   - MediaAsset (nosql)
     Coleção: media_assets
   - SocialMediaPublication (nosql)
     Coleção: social_publications

🖥️ Interfaces: 2
   - API (api)
     Endpoints: 10
       [POST] /api/v1/submittavelstory - Submeter relato com mídias
       [POST] /api/v1/filterpersonaldata - Filtrar informações pessoais (PII)
       [POST] /api/v1/enrichstorycontent - Coletar informações adicionais
       [POST] /api/v1/generatesocialmediaarticle - Criar artigos para redes sociais
       [POST] /api/v1/curatecontent - Curadoria de conteúdo
       [POST] /api/v1/publishtosocialmedia - Publicar em redes sociais
       [POST] /api/v1/monitorengagement - Monitorar engajamento
       [POST] /api/v1/managetravelerprofile - Gerenciar perfil de viajante
       [POST] /api/v1/externalapiintegration - Integrar APIs externas
       [POST] /api/v1/updatestory - Atualizar relatos existentes
   - Web UI (web)
     Telas: 10
       📱 SubmitTravelStory (/submittavelstory)
         Componentes: 7
       📱 CurateContent (/curatecontent)
         Componentes: 4
       ...

📱 Canais de Comunicação: 2
   - webhook (webhook) - Habilitado: true
     Configuração:
       path: /webhook
       secret: ${WEBHOOK_SECRET}
   - messaging (messaging) - Habilitado: true
     Configuração:
       provider: ${MESSAGING_PROVIDER}

🔐 Políticas OPA: 4
   - rbac_policy (authz.rbac)
     Descrição: Role-Based Access Control policy
   - privacy_protection_policy (authz.custom)
     Descrição: Sistema deve detectar e remover automaticamente PII
   - content_moderation_policy (authz.custom)
     Descrição: Conteúdo deve passar por moderação antes de publicação
   - data_access_policy (authz.data_access)
     Descrição: Data access control based on classification levels

🎯 Skills: 5
   - filterpersonaldata_skill
     Descrição: Filtrar automaticamente informações pessoais dos relatos
     Triggers: [FilterPersonalData]
     Dependências: [pii_detection_lib]
   - enrichstorycontent_skill
     Descrição: Coletar informações adicionais do autor
     Triggers: [EnrichStoryContent]
   - curatecontent_skill
     Descrição: Realizar curadoria do conteúdo
     Triggers: [CurateContent]
   - generatesocialmediaarticle_skill
     Descrição: Elaborar artigos formatados para redes sociais
     Triggers: [GenerateSocialMediaArticle]
   - monitorengagement_skill
     Descrição: Monitorar engajamento das publicações
     Triggers: [MonitorEngagement]

🔧 Tools: 5
   - api_tool (custom)
   - webhook_handler_tool (custom)
   - file_tool (custom)
   - external_api_tool (custom)
     Configuração:
       type: http
       providers: instagram,facebook,twitter,linkedin
   - ai_tool (custom)
     Configuração:
       type: ai
       capabilities: pii_detection,content_generation

🔌 Servidores MCP: 1
   - default (stdio)
     Comando: mcp-server --config ${MCP_CONFIG_PATH}

✅ Validação: true
   Sugestões: 2
     💡 Enable AI features for content enrichment and curation
     💡 Configure external API credentials for social media platforms

📄 AGENT.json e AGENT.md salvos
📄 Schema TravelStory salvo
📄 Schema TravelerProfile salvo
📄 Schema MediaAsset salvo
📄 Schema SocialMediaPublication salvo
📄 Política rbac_policy salva (JSON)
📄 Código Rego rbac_policy salvo
... (outras políticas)
📄 interfaces.json salvo
📄 channels.json salvo
📄 skills.json salvo
📄 Códigos das skills salvos
📄 tools.json salvo
📄 mcp_config.json salvo
📄 validation_report.json salvo

✅ Artefatos gerados com sucesso!
```

### Arquivos Gerados em `output/travel/`

```
output/travel/
├── AGENT.json
├── AGENT.md
├── schema_0_travelstory.json
├── schema_1_travelerprofile.json
├── schema_2_mediaasset.json
├── schema_3_socialpublication.json
├── policy_0_rbac_policy.rego.json
├── policy_0_rbac_policy.rego
├── policy_1_privacy_protection_policy.rego.json
├── policy_1_privacy_protection_policy.rego
├── policy_2_content_moderation_policy.rego.json
├── policy_2_content_moderation_policy.rego
├── policy_3_data_access_policy.rego.json
├── policy_3_data_access_policy.rego
├── interfaces.json
├── channels.json
├── skills.json
├── skill_0_filterpersonaldata_skill.go
├── skill_1_enrichstorycontent_skill.go
├── skill_2_curatecontent_skill.go
├── skill_3_generatesocialmediaarticle_skill.go
├── skill_4_monitorengagement_skill.go
├── tools.json
├── mcp_config.json
└── validation_report.json
```

### Exemplo de Política de Privacidade (policy_1_privacy_protection_policy.rego)

```rego
package authz.custom

# Policy: PrivacyProtection
# Description: Sistema deve detectar e remover automaticamente informações pessoais identificáveis (PII)

default allow = false

# PII detection enabled
pii_detection_enabled {
  input.pii_detection == "true"
}

# Auto redaction enabled
auto_redaction_enabled {
  input.auto_redaction == "true"
}

# GDPR compliance check
gdpr_compliant {
  input.gdpr_compliance == "true"
  input.consent_obtained == true
}

allow {
  pii_detection_enabled
  auto_redaction_enabled
  gdpr_compliant
}
```

### Exemplo de Skill Gerada (skill_0_filterpersonaldata_skill.go)

```go
// Auto-generated skill for: FilterPersonalData
package skills

import (
	"context"
	"regexp"
)

// FilterPersonalDataSkill filters personally identifiable information from content
func FilterPersonalDataSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Preconditions: Relato deve estar em revisão
	
	content, ok := params["story_content"].(string)
	if !ok {
		return nil, fmt.Errorf("story_content is required")
	}
	
	// Detect PII patterns
	patterns := map[string]*regexp.Regexp{
		"email": regexp.MustCompile(`[\w\.-]+@[\w\.-]+\.\w+`),
		"phone": regexp.MustCompile(`\+?\d[\d\s\-\(\)]{8,}\d`),
		"cpf":   regexp.MustCompile(`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`),
	}
	
	detectedPII := make([]map[string]string, 0)
	filteredContent := content
	
	for piiType, pattern := range patterns {
		matches := pattern.FindAllString(content, -1)
		for _, match := range matches {
			detectedPII = append(detectedPII, map[string]string{
				"type":  piiType,
				"value": match,
			})
			// Redact PII
			filteredContent = regexp.MustCompile(regexp.QuoteMeta(match)).
				ReplaceAllString(filteredContent, "[REDACTED]")
		}
	}
	
	result := map[string]interface{}{
		"filtered_content": filteredContent,
		"detected_pii":     detectedPII,
		"confidence_score": float64(len(detectedPII)) / float64(len(content)) * 100,
	}
	
	return result, nil
}
```

---

## 🔍 Como Testar as Políticas OPA Geradas

### Instalando OPA

```bash
# Linux/Mac
curl -L -o opa https://openpolicyagent.org/downloads/latest/opa_linux_amd64_static
chmod +x opa
sudo mv opa /usr/local/bin/
```

### Testando Política RBAC

Crie um arquivo `input.json`:

```json
{
  "user": {
    "id": "user123",
    "roles": ["attendant"]
  },
  "permission": "add_grains",
  "resource": "grain_transaction"
}
```

Execute:

```bash
cd output/cafeteria
opa eval -i input.json -d policy_0_rbac_policy.rego "data.authz.rbac.allow"
# Resultado esperado: true
```

Teste com permissão não autorizada:

```json
{
  "user": {
    "id": "customer456",
    "roles": ["customer"]
  },
  "permission": "add_grains"
}
```

```bash
opa eval -i input.json -d policy_0_rbac_policy.rego "data.authz.rbac.allow"
# Resultado esperado: false
```

---

## 🚀 Próximos Passos

1. **Revisar Artefatos**: Analise os arquivos gerados
2. **Customizar**: Ajuste conforme necessidades específicas
3. **Configurar Credenciais**:
   - Token do Telegram
   - Connection string do banco de dados
   - Chaves de API para redes sociais
4. **Implementar Skills**: Complete a lógica de negócio nas skills geradas
5. **Testar Localmente**: Execute o agente em modo de desenvolvimento
6. **Implantar**: Copie artefatos para produção

Para mais detalhes, consulte `examples/protoagent/README.md`.
