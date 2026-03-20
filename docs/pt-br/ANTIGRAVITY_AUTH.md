> Voltar ao [README](../../README.pt-br.md)

# Guia de Autenticação e Integração do Antigravity

## Visão Geral

**Antigravity** (Google Cloud Code Assist) é um provedor de modelos de IA apoiado pelo Google que oferece acesso a modelos como Claude Opus 4.6 e Gemini através da infraestrutura de nuvem do Google. Este documento fornece um guia completo sobre como a autenticação funciona, como buscar modelos e como implementar um novo provedor no PicoClaw.

---

## Índice

1. [Fluxo de Autenticação](#fluxo-de-autenticação)
2. [Detalhes da Implementação OAuth](#detalhes-da-implementação-oauth)
3. [Gerenciamento de Tokens](#gerenciamento-de-tokens)
4. [Busca da Lista de Modelos](#busca-da-lista-de-modelos)
5. [Rastreamento de Uso](#rastreamento-de-uso)
6. [Estrutura do Plugin do Provedor](#estrutura-do-plugin-do-provedor)
7. [Requisitos de Integração](#requisitos-de-integração)
8. [Endpoints da API](#endpoints-da-api)
9. [Configuração](#configuração)
10. [Criando um Novo Provedor no PicoClaw](#criando-um-novo-provedor-no-picoclaw)

---

## Fluxo de Autenticação

### 1. OAuth 2.0 com PKCE

O Antigravity utiliza **OAuth 2.0 com PKCE (Proof Key for Code Exchange)** para autenticação segura:

```
┌─────────────┐                                    ┌─────────────────┐
│   Client    │ ───(1) Generate PKCE Pair────────> │                 │
│             │ ───(2) Open Auth URL─────────────> │  Google OAuth   │
│             │                                    │    Server       │
│             │ <──(3) Redirect with Code───────── │                 │
│             │                                    └─────────────────┘
│             │ ───(4) Exchange Code for Tokens──> │   Token URL     │
│             │                                    │                 │
│             │ <──(5) Access + Refresh Tokens──── │                 │
└─────────────┘                                    └─────────────────┘
```

### 2. Etapas Detalhadas

#### Etapa 1: Gerar Parâmetros PKCE
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### Etapa 2: Construir a URL de Autorização
```typescript
const AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth";
const REDIRECT_URI = "http://localhost:51121/oauth-callback";

function buildAuthUrl(params: { challenge: string; state: string }): string {
  const url = new URL(AUTH_URL);
  url.searchParams.set("client_id", CLIENT_ID);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("redirect_uri", REDIRECT_URI);
  url.searchParams.set("scope", SCOPES.join(" "));
  url.searchParams.set("code_challenge", params.challenge);
  url.searchParams.set("code_challenge_method", "S256");
  url.searchParams.set("state", params.state);
  url.searchParams.set("access_type", "offline");
  url.searchParams.set("prompt", "consent");
  return url.toString();
}
```

**Escopos Necessários:**
```typescript
const SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];
```

#### Etapa 3: Tratar o Callback OAuth

**Modo Automático (Desenvolvimento Local):**
- Iniciar um servidor HTTP local na porta 51121
- Aguardar o redirecionamento do Google
- Extrair o código de autorização dos parâmetros da query

**Modo Manual (Remoto/Sem Interface Gráfica):**
- Exibir a URL de autorização para o usuário
- O usuário completa a autenticação no navegador
- O usuário cola a URL de redirecionamento completa no terminal
- Analisar o código da URL colada

#### Etapa 4: Trocar o Código por Tokens
```typescript
const TOKEN_URL = "https://oauth2.googleapis.com/token";

async function exchangeCode(params: {
  code: string;
  verifier: string;
}): Promise<{ access: string; refresh: string; expires: number }> {
  const response = await fetch(TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
      code: params.code,
      grant_type: "authorization_code",
      redirect_uri: REDIRECT_URI,
      code_verifier: params.verifier,
    }),
  });

  const data = await response.json();
  
  return {
    access: data.access_token,
    refresh: data.refresh_token,
    expires: Date.now() + data.expires_in * 1000 - 5 * 60 * 1000, // 5 min buffer
  };
}
```

#### Etapa 5: Buscar Dados Adicionais do Usuário

**E-mail do Usuário:**
```typescript
async function fetchUserEmail(accessToken: string): Promise<string | undefined> {
  const response = await fetch(
    "https://www.googleapis.com/oauth2/v1/userinfo?alt=json",
    { headers: { Authorization: `Bearer ${accessToken}` } }
  );
  const data = await response.json();
  return data.email;
}
```

**ID do Projeto (Necessário para chamadas de API):**
```typescript
async function fetchProjectId(accessToken: string): Promise<string> {
  const headers = {
    Authorization: `Bearer ${accessToken}`,
    "Content-Type": "application/json",
    "User-Agent": "google-api-nodejs-client/9.15.1",
    "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
    "Client-Metadata": JSON.stringify({
      ideType: "IDE_UNSPECIFIED",
      platform: "PLATFORM_UNSPECIFIED",
      pluginType: "GEMINI",
    }),
  };

  const response = await fetch(
    "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist",
    {
      method: "POST",
      headers,
      body: JSON.stringify({
        metadata: {
          ideType: "IDE_UNSPECIFIED",
          platform: "PLATFORM_UNSPECIFIED",
          pluginType: "GEMINI",
        },
      }),
    }
  );

  const data = await response.json();
  return data.cloudaicompanionProject || "rising-fact-p41fc"; // Valor padrão de fallback
}
```

---

## Detalhes da Implementação OAuth

### Credenciais do Cliente

**Importante:** Estas são codificadas em base64 no código-fonte para sincronização com pi-ai:

```typescript
const decode = (s: string) => Buffer.from(s, "base64").toString();

const CLIENT_ID = decode(
  "MTA3MTAwNjA2MDU5MS10bWhzc2luMmgyMWxjcmUyMzV2dG9sb2poNGc0MDNlcC5hcHBzLmdvb2dsZXVzZXJjb250ZW50LmNvbQ=="
);
const CLIENT_SECRET = decode("R09DU1BYLUs1OEZXUjQ4NkxkTEoxbUxCOHNYQzR6NnFEQWY=");
```

### Modos do Fluxo OAuth

1. **Fluxo Automático** (máquinas locais com navegador):
   - Abre o navegador automaticamente
   - O servidor de callback local captura o redirecionamento
   - Nenhuma interação do usuário necessária após a autenticação inicial

2. **Fluxo Manual** (remoto/sem interface/WSL2):
   - URL exibida para copiar e colar manualmente
   - O usuário completa a autenticação em um navegador externo
   - O usuário cola a URL de redirecionamento completa de volta

```typescript
function shouldUseManualOAuthFlow(isRemote: boolean): boolean {
  return isRemote || isWSL2Sync();
}
```

---

## Gerenciamento de Tokens

### Estrutura do Perfil de Autenticação

```typescript
type OAuthCredential = {
  type: "oauth";
  provider: "google-antigravity";
  access: string;           // Token de acesso
  refresh: string;          // Token de atualização
  expires: number;          // Timestamp de expiração (ms desde epoch)
  email?: string;           // E-mail do usuário
  projectId?: string;       // ID do projeto Google Cloud
};
```

### Atualização de Tokens

A credencial inclui um token de atualização que pode ser usado para obter novos tokens de acesso quando o atual expira. A expiração é definida com um buffer de 5 minutos para evitar condições de corrida.

---

## Busca da Lista de Modelos

### Buscar Modelos Disponíveis

```typescript
const BASE_URL = "https://cloudcode-pa.googleapis.com";

async function fetchAvailableModels(
  accessToken: string,
  projectId: string
): Promise<Model[]> {
  const headers = {
    Authorization: `Bearer ${accessToken}`,
    "Content-Type": "application/json",
    "User-Agent": "antigravity",
    "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
  };

  const response = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers,
      body: JSON.stringify({ project: projectId }),
    }
  );

  const data = await response.json();
  
  // Retorna modelos com informações de cota
  return Object.entries(data.models).map(([modelId, modelInfo]) => ({
    id: modelId,
    displayName: modelInfo.displayName,
    quotaInfo: {
      remainingFraction: modelInfo.quotaInfo?.remainingFraction,
      resetTime: modelInfo.quotaInfo?.resetTime,
      isExhausted: modelInfo.quotaInfo?.isExhausted,
    },
  }));
}
```

### Formato da Resposta

```typescript
type FetchAvailableModelsResponse = {
  models?: Record<string, {
    displayName?: string;
    quotaInfo?: {
      remainingFraction?: number | string;
      resetTime?: string;      // Timestamp ISO 8601
      isExhausted?: boolean;
    };
  }>;
};
```

---

## Rastreamento de Uso

### Buscar Dados de Uso

```typescript
export async function fetchAntigravityUsage(
  token: string,
  timeoutMs: number
): Promise<ProviderUsageSnapshot> {
  // 1. Buscar créditos e informações do plano
  const loadCodeAssistRes = await fetch(
    `${BASE_URL}/v1internal:loadCodeAssist`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        metadata: {
          ideType: "ANTIGRAVITY",
          platform: "PLATFORM_UNSPECIFIED",
          pluginType: "GEMINI",
        },
      }),
    }
  );

  // Extrair informações de créditos
  const { availablePromptCredits, planInfo, currentTier } = data;
  
  // 2. Buscar cotas dos modelos
  const modelsRes = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ project: projectId }),
    }
  );

  // Construir janelas de uso
  return {
    provider: "google-antigravity",
    displayName: "Google Antigravity",
    windows: [
      { label: "Credits", usedPercent: calculateUsedPercent(available, monthly) },
      // Cotas individuais dos modelos...
    ],
    plan: currentTier?.name || planType,
  };
}
```

### Estrutura da Resposta de Uso

```typescript
type ProviderUsageSnapshot = {
  provider: "google-antigravity";
  displayName: string;
  windows: UsageWindow[];
  plan?: string;
  error?: string;
};

type UsageWindow = {
  label: string;           // "Credits" ou ID do modelo
  usedPercent: number;     // 0-100
  resetAt?: number;        // Timestamp de quando a cota é redefinida
};
```

---

## Estrutura do Plugin do Provedor

### Definição do Plugin

```typescript
const antigravityPlugin = {
  id: "google-antigravity-auth",
  name: "Google Antigravity Auth",
  description: "OAuth flow for Google Antigravity (Cloud Code Assist)",
  configSchema: emptyPluginConfigSchema(),
  
  register(api: PicoClawPluginApi) {
    api.registerProvider({
      id: "google-antigravity",
      label: "Google Antigravity",
      docsPath: "/providers/models",
      aliases: ["antigravity"],
      
      auth: [
        {
          id: "oauth",
          label: "Google OAuth",
          hint: "PKCE + localhost callback",
          kind: "oauth",
          run: async (ctx: ProviderAuthContext) => {
            // Implementação OAuth aqui
          },
        },
      ],
    });
  },
};
```

### ProviderAuthContext

```typescript
type ProviderAuthContext = {
  config: PicoClawConfig;
  agentDir?: string;
  workspaceDir?: string;
  prompter: WizardPrompter;      // Prompts/notificações da UI
  runtime: RuntimeEnv;           // Logging, etc.
  isRemote: boolean;             // Se está executando remotamente
  openUrl: (url: string) => Promise<void>;  // Abridor de navegador
  oauth: {
    createVpsAwareHandlers: Function;
  };
};
```

### ProviderAuthResult

```typescript
type ProviderAuthResult = {
  profiles: Array<{
    profileId: string;
    credential: AuthProfileCredential;
  }>;
  configPatch?: Partial<PicoClawConfig>;
  defaultModel?: string;
  notes?: string[];
};
```

---

## Requisitos de Integração

### 1. Ambiente/Dependências Necessários

- Go ≥ 1.25
- Base de código do PicoClaw (`pkg/providers/` e `pkg/auth/`)
- Pacotes da biblioteca padrão `crypto` e `net/http`

### 2. Cabeçalhos Necessários para Chamadas de API

```typescript
const REQUIRED_HEADERS = {
  "Authorization": `Bearer ${accessToken}`,
  "Content-Type": "application/json",
  "User-Agent": "antigravity",  // ou "google-api-nodejs-client/9.15.1"
  "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
};

// Para chamadas loadCodeAssist, incluir também:
const CLIENT_METADATA = {
  ideType: "ANTIGRAVITY",  // ou "IDE_UNSPECIFIED"
  platform: "PLATFORM_UNSPECIFIED",
  pluginType: "GEMINI",
};
```

### 3. Sanitização de Schemas de Modelos

O Antigravity usa modelos compatíveis com Gemini, então os schemas de ferramentas devem ser sanitizados:

```typescript
const GOOGLE_SCHEMA_UNSUPPORTED_KEYWORDS = new Set([
  "patternProperties",
  "additionalProperties",
  "$schema",
  "$id",
  "$ref",
  "$defs",
  "definitions",
  "examples",
  "minLength",
  "maxLength",
  "minimum",
  "maximum",
  "multipleOf",
  "pattern",
  "format",
  "minItems",
  "maxItems",
  "uniqueItems",
  "minProperties",
  "maxProperties",
]);

// Limpar schema antes de enviar
function cleanToolSchemaForGemini(schema: Record<string, unknown>): unknown {
  // Remover palavras-chave não suportadas
  // Garantir que o nível superior tenha type: "object"
  // Achatar uniões anyOf/oneOf
}
```

### 4. Tratamento de Blocos de Pensamento (Modelos Claude)

Para modelos Claude via Antigravity, os blocos de pensamento requerem tratamento especial:

```typescript
const ANTIGRAVITY_SIGNATURE_RE = /^[A-Za-z0-9+/]+={0,2}$/;

export function sanitizeAntigravityThinkingBlocks(
  messages: AgentMessage[]
): AgentMessage[] {
  // Validar assinaturas de pensamento
  // Normalizar campos de assinatura
  // Descartar blocos de pensamento não assinados
}
```

---

## Endpoints da API

### Endpoints de Autenticação

| Endpoint | Método | Finalidade |
|----------|--------|-----------|
| `https://accounts.google.com/o/oauth2/v2/auth` | GET | Autorização OAuth |
| `https://oauth2.googleapis.com/token` | POST | Troca de tokens |
| `https://www.googleapis.com/oauth2/v1/userinfo` | GET | Informações do usuário (e-mail) |

### Endpoints do Cloud Code Assist

| Endpoint | Método | Finalidade |
|----------|--------|-----------|
| `https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist` | POST | Carregar informações do projeto, créditos, plano |
| `https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels` | POST | Listar modelos disponíveis com cotas |
| `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse` | POST | Endpoint de streaming de chat |

**Formato de Requisição da API (Chat):**
O endpoint `v1internal:streamGenerateContent` espera um envelope encapsulando a requisição Gemini padrão:

```json
{
  "project": "your-project-id",
  "model": "model-id",
  "request": {
    "contents": [...],
    "systemInstruction": {...},
    "generationConfig": {...},
    "tools": [...]
  },
  "requestType": "agent",
  "userAgent": "antigravity",
  "requestId": "agent-timestamp-random"
}
```

**Formato de Resposta da API (SSE):**
Cada mensagem SSE (`data: {...}`) é encapsulada em um campo `response`:

```json
{
  "response": {
    "candidates": [...],
    "usageMetadata": {...},
    "modelVersion": "...",
    "responseId": "..."
  },
  "traceId": "...",
  "metadata": {}
}
```

---

## Configuração

### Configuração do config.json

```json
{
  "model_list": [
    {
      "model_name": "gemini-flash",
      "model": "antigravity/gemini-3-flash",
      "auth_method": "oauth"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gemini-flash"
    }
  }
}
```

### Armazenamento do Perfil de Autenticação

Os perfis de autenticação são armazenados em `~/.picoclaw/auth.json`:

```json
{
  "credentials": {
    "google-antigravity": {
      "access_token": "ya29...",
      "refresh_token": "1//...",
      "expires_at": "2026-01-01T00:00:00Z",
      "provider": "google-antigravity",
      "auth_method": "oauth",
      "email": "user@example.com",
      "project_id": "my-project-id"
    }
  }
}
```

---

## Criando um Novo Provedor no PicoClaw

Os provedores do PicoClaw são implementados como pacotes Go em `pkg/providers/`. Para adicionar um novo provedor:

### Implementação Passo a Passo

#### 1. Criar o Arquivo do Provedor

Crie um novo arquivo Go em `pkg/providers/`:

```
pkg/providers/
└── your_provider.go
```

#### 2. Implementar a Interface Provider

Seu provedor deve implementar a interface `Provider` definida em `pkg/providers/types.go`:

```go
package providers

type YourProvider struct {
    apiKey  string
    apiBase string
}

func NewYourProvider(apiKey, apiBase, proxy string) *YourProvider {
    if apiBase == "" {
        apiBase = "https://api.your-provider.com/v1"
    }
    return &YourProvider{apiKey: apiKey, apiBase: apiBase}
}

func (p *YourProvider) Chat(ctx context.Context, messages []Message, tools []Tool, cb StreamCallback) error {
    // Implementar conclusão de chat com streaming
}
```

#### 3. Registrar na Factory

Adicione seu provedor ao switch de protocolo em `pkg/providers/factory.go`:

```go
case "your-provider":
    return NewYourProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
```

#### 4. Adicionar Configuração Padrão (Opcional)

Adicione uma entrada padrão em `pkg/config/defaults.go`:

```go
{
    ModelName: "your-model",
    Model:     "your-provider/model-name",
    APIKey:    "",
},
```

#### 5. Adicionar Suporte de Autenticação (Opcional)

Se seu provedor requer OAuth ou autenticação especial, adicione um caso em `cmd/picoclaw/internal/auth/helpers.go`:

```go
case "your-provider":
    authLoginYourProvider()
```

#### 6. Configurar via `config.json`

```json
{
  "model_list": [
    {
      "model_name": "your-model",
      "model": "your-provider/model-name",
      "api_key": "your-api-key",
      "api_base": "https://api.your-provider.com/v1"
    }
  ]
}
```

---

## Testando Sua Implementação

### Comandos CLI

```bash
# Autenticar com um provedor
picoclaw auth login --provider your-provider

# Listar modelos (para Antigravity)
picoclaw auth models

# Iniciar o gateway
picoclaw gateway

# Executar um agente com um modelo específico
picoclaw agent -m "Hello" --model your-model
```

### Variáveis de Ambiente para Testes

```bash
# Substituir o modelo padrão
export PICOCLAW_AGENTS_DEFAULTS_MODEL=your-model

# Substituir configurações do provedor
export PICOCLAW_MODEL_LIST='[{"model_name":"your-model","model":"your-provider/model-name","api_key":"..."}]'
```

---

## Referências

- **Arquivos Fonte:**
  - `pkg/providers/antigravity_provider.go` - Implementação do provedor Antigravity
  - `pkg/auth/oauth.go` - Implementação do fluxo OAuth
  - `pkg/auth/store.go` - Armazenamento de credenciais de autenticação (`~/.picoclaw/auth.json`)
  - `pkg/providers/factory.go` - Factory de provedores e roteamento de protocolo
  - `pkg/providers/types.go` - Definições da interface do provedor
  - `cmd/picoclaw/internal/auth/helpers.go` - Comandos CLI de autenticação

- **Documentação:**
  - `docs/ANTIGRAVITY_USAGE.md` - Guia de uso do Antigravity
  - `docs/migration/model-list-migration.md` - Guia de migração

---

## Observações

1. **Projeto Google Cloud:** O Antigravity requer que o Gemini for Google Cloud esteja habilitado no seu projeto Google Cloud
2. **Cotas:** Usa cotas do projeto Google Cloud (sem cobrança separada)
3. **Acesso a Modelos:** Os modelos disponíveis dependem da configuração do seu projeto Google Cloud
4. **Blocos de Pensamento:** Modelos Claude via Antigravity requerem tratamento especial de blocos de pensamento com assinaturas
5. **Sanitização de Schemas:** Os schemas de ferramentas devem ser sanitizados para remover palavras-chave JSON Schema não suportadas

---

---

## Tratamento de Erros Comuns

### 1. Limitação de Taxa (HTTP 429)

O Antigravity retorna um erro 429 quando as cotas do projeto/modelo estão esgotadas. A resposta de erro frequentemente contém um `quotaResetDelay` no campo `details`.

**Exemplo de Erro 429:**
```json
{
  "error": {
    "code": 429,
    "message": "You have exhausted your capacity on this model. Your quota will reset after 4h30m28s.",
    "status": "RESOURCE_EXHAUSTED",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.ErrorInfo",
        "metadata": {
          "quotaResetDelay": "4h30m28.060903746s"
        }
      }
    ]
  }
}
```

### 2. Respostas Vazias (Modelos Restritos)

Alguns modelos podem aparecer na lista de modelos disponíveis, mas retornar uma resposta vazia (200 OK mas stream SSE vazio). Isso geralmente acontece com modelos em preview ou restritos que o projeto atual não tem permissão para usar.

**Tratamento:** Tratar respostas vazias como erros informando ao usuário que o modelo pode estar restrito ou inválido para seu projeto.

---

## Solução de Problemas

### "Token expired" (token expirado)
- Atualizar tokens OAuth: `picoclaw auth login --provider antigravity`

### "Gemini for Google Cloud is not enabled" (Gemini for Google Cloud não está habilitado)
- Habilitar a API no seu Google Cloud Console

### "Project not found" (projeto não encontrado)
- Verificar se seu projeto Google Cloud tem as APIs necessárias habilitadas
- Verificar se o ID do projeto foi obtido corretamente durante a autenticação

### Modelos não aparecem na lista
- Verificar se a autenticação OAuth foi concluída com sucesso
- Verificar o armazenamento do perfil de autenticação: `~/.picoclaw/auth.json`
- Executar novamente `picoclaw auth login --provider antigravity`
