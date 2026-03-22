> Retour au [README](../../README.fr.md)

# Guide d'authentification et d'intégration Antigravity

## Aperçu

**Antigravity** (Google Cloud Code Assist) est un fournisseur de modèles IA soutenu par Google qui offre l'accès à des modèles tels que Claude Opus 4.6 et Gemini via l'infrastructure cloud de Google. Ce document fournit un guide complet sur le fonctionnement de l'authentification, la récupération des modèles et l'implémentation d'un nouveau fournisseur dans PicoClaw.

---

## Table des matières

1. [Flux d'authentification](#flux-dauthentification)
2. [Détails de l'implémentation OAuth](#détails-de-limplémentation-oauth)
3. [Gestion des jetons](#gestion-des-jetons)
4. [Récupération de la liste des modèles](#récupération-de-la-liste-des-modèles)
5. [Suivi de l'utilisation](#suivi-de-lutilisation)
6. [Structure du plugin fournisseur](#structure-du-plugin-fournisseur)
7. [Exigences d'intégration](#exigences-dintégration)
8. [Points de terminaison API](#points-de-terminaison-api)
9. [Configuration](#configuration)
10. [Créer un nouveau fournisseur dans PicoClaw](#créer-un-nouveau-fournisseur-dans-picoclaw)

---

## Flux d'authentification

### 1. OAuth 2.0 avec PKCE

Antigravity utilise **OAuth 2.0 avec PKCE (Proof Key for Code Exchange)** pour une authentification sécurisée :

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

### 2. Étapes détaillées

#### Étape 1 : Générer les paramètres PKCE
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### Étape 2 : Construire l'URL d'autorisation
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

**Portées requises :**
```typescript
const SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];
```

#### Étape 3 : Gérer le callback OAuth

**Mode automatique (développement local) :**
- Démarrer un serveur HTTP local sur le port 51121
- Attendre la redirection de Google
- Extraire le code d'autorisation des paramètres de requête

**Mode manuel (distant/sans interface graphique) :**
- Afficher l'URL d'autorisation à l'utilisateur
- L'utilisateur complète l'authentification dans son navigateur
- L'utilisateur colle l'URL de redirection complète dans le terminal
- Analyser le code depuis l'URL collée

#### Étape 4 : Échanger le code contre des jetons
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

#### Étape 5 : Récupérer les données utilisateur supplémentaires

**E-mail de l'utilisateur :**
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

**ID du projet (requis pour les appels API) :**
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
  return data.cloudaicompanionProject || "rising-fact-p41fc"; // Valeur par défaut
}
```

---

## Détails de l'implémentation OAuth

### Identifiants client

**Important :** Ceux-ci sont encodés en base64 dans le code source pour la synchronisation avec pi-ai :

```typescript
const decode = (s: string) => Buffer.from(s, "base64").toString();

const CLIENT_ID = decode(
  "MTA3MTAwNjA2MDU5MS10bWhzc2luMmgyMWxjcmUyMzV2dG9sb2poNGc0MDNlcC5hcHBzLmdvb2dsZXVzZXJjb250ZW50LmNvbQ=="
);
const CLIENT_SECRET = decode("R09DU1BYLUs1OEZXUjQ4NkxkTEoxbUxCOHNYQzR6NnFEQWY=");
```

### Modes de flux OAuth

1. **Flux automatique** (machines locales avec navigateur) :
   - Ouvre le navigateur automatiquement
   - Le serveur de callback local capture la redirection
   - Aucune interaction utilisateur requise après l'authentification initiale

2. **Flux manuel** (distant/sans interface/WSL2) :
   - URL affichée pour copier-coller manuellement
   - L'utilisateur complète l'authentification dans un navigateur externe
   - L'utilisateur colle l'URL de redirection complète

```typescript
function shouldUseManualOAuthFlow(isRemote: boolean): boolean {
  return isRemote || isWSL2Sync();
}
```

---

## Gestion des jetons

### Structure du profil d'authentification

```typescript
type OAuthCredential = {
  type: "oauth";
  provider: "google-antigravity";
  access: string;           // Jeton d'accès
  refresh: string;          // Jeton de rafraîchissement
  expires: number;          // Horodatage d'expiration (ms depuis epoch)
  email?: string;           // E-mail de l'utilisateur
  projectId?: string;       // ID du projet Google Cloud
};
```

### Rafraîchissement des jetons

Les identifiants incluent un jeton de rafraîchissement qui peut être utilisé pour obtenir de nouveaux jetons d'accès lorsque le jeton actuel expire. L'expiration est définie avec un tampon de 5 minutes pour éviter les conditions de concurrence.

---

## Récupération de la liste des modèles

### Récupérer les modèles disponibles

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
  
  // Retourne les modèles avec les informations de quota
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

### Format de réponse

```typescript
type FetchAvailableModelsResponse = {
  models?: Record<string, {
    displayName?: string;
    quotaInfo?: {
      remainingFraction?: number | string;
      resetTime?: string;      // Horodatage ISO 8601
      isExhausted?: boolean;
    };
  }>;
};
```

---

## Suivi de l'utilisation

### Récupérer les données d'utilisation

```typescript
export async function fetchAntigravityUsage(
  token: string,
  timeoutMs: number
): Promise<ProviderUsageSnapshot> {
  // 1. Récupérer les crédits et les informations du plan
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

  // Extraire les informations de crédits
  const { availablePromptCredits, planInfo, currentTier } = data;
  
  // 2. Récupérer les quotas des modèles
  const modelsRes = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ project: projectId }),
    }
  );

  // Construire les fenêtres d'utilisation
  return {
    provider: "google-antigravity",
    displayName: "Google Antigravity",
    windows: [
      { label: "Credits", usedPercent: calculateUsedPercent(available, monthly) },
      // Quotas individuels des modèles...
    ],
    plan: currentTier?.name || planType,
  };
}
```

### Structure de la réponse d'utilisation

```typescript
type ProviderUsageSnapshot = {
  provider: "google-antigravity";
  displayName: string;
  windows: UsageWindow[];
  plan?: string;
  error?: string;
};

type UsageWindow = {
  label: string;           // "Credits" ou ID du modèle
  usedPercent: number;     // 0-100
  resetAt?: number;        // Horodatage de réinitialisation du quota
};
```

---

## Structure du plugin fournisseur

### Définition du plugin

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
            // Implémentation OAuth ici
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
  prompter: WizardPrompter;      // Invites/notifications UI
  runtime: RuntimeEnv;           // Journalisation, etc.
  isRemote: boolean;             // Exécution à distance ou non
  openUrl: (url: string) => Promise<void>;  // Ouverture du navigateur
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

## Exigences d'intégration

### 1. Environnement/dépendances requis

- Go ≥ 1.25
- Base de code PicoClaw (`pkg/providers/` et `pkg/auth/`)
- Packages de la bibliothèque standard `crypto` et `net/http`

### 2. En-têtes requis pour les appels API

```typescript
const REQUIRED_HEADERS = {
  "Authorization": `Bearer ${accessToken}`,
  "Content-Type": "application/json",
  "User-Agent": "antigravity",  // ou "google-api-nodejs-client/9.15.1"
  "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
};

// Pour les appels loadCodeAssist, inclure également :
const CLIENT_METADATA = {
  ideType: "ANTIGRAVITY",  // ou "IDE_UNSPECIFIED"
  platform: "PLATFORM_UNSPECIFIED",
  pluginType: "GEMINI",
};
```

### 3. Assainissement des schémas de modèles

Antigravity utilise des modèles compatibles Gemini, les schémas d'outils doivent donc être assainis :

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

// Nettoyer le schéma avant l'envoi
function cleanToolSchemaForGemini(schema: Record<string, unknown>): unknown {
  // Supprimer les mots-clés non supportés
  // S'assurer que le niveau supérieur a type: "object"
  // Aplatir les unions anyOf/oneOf
}
```

### 4. Gestion des blocs de réflexion (modèles Claude)

Pour les modèles Claude via Antigravity, les blocs de réflexion nécessitent un traitement spécial :

```typescript
const ANTIGRAVITY_SIGNATURE_RE = /^[A-Za-z0-9+/]+={0,2}$/;

export function sanitizeAntigravityThinkingBlocks(
  messages: AgentMessage[]
): AgentMessage[] {
  // Valider les signatures de réflexion
  // Normaliser les champs de signature
  // Rejeter les blocs de réflexion non signés
}
```

---

## Points de terminaison API

### Points de terminaison d'authentification

| Point de terminaison | Méthode | Objectif |
|---------------------|---------|----------|
| `https://accounts.google.com/o/oauth2/v2/auth` | GET | Autorisation OAuth |
| `https://oauth2.googleapis.com/token` | POST | Échange de jetons |
| `https://www.googleapis.com/oauth2/v1/userinfo` | GET | Informations utilisateur (e-mail) |

### Points de terminaison Cloud Code Assist

| Point de terminaison | Méthode | Objectif |
|---------------------|---------|----------|
| `https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist` | POST | Charger les infos du projet, crédits, plan |
| `https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels` | POST | Lister les modèles disponibles avec quotas |
| `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse` | POST | Point de terminaison de streaming de chat |

**Format de requête API (chat) :**
Le point de terminaison `v1internal:streamGenerateContent` attend une enveloppe encapsulant la requête Gemini standard :

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

**Format de réponse API (SSE) :**
Chaque message SSE (`data: {...}`) est encapsulé dans un champ `response` :

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

## Configuration

### Configuration config.json

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

### Stockage du profil d'authentification

Les profils d'authentification sont stockés dans `~/.picoclaw/auth.json` :

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

## Créer un nouveau fournisseur dans PicoClaw

Les fournisseurs PicoClaw sont implémentés en tant que packages Go sous `pkg/providers/`. Pour ajouter un nouveau fournisseur :

### Implémentation étape par étape

#### 1. Créer le fichier du fournisseur

Créez un nouveau fichier Go dans `pkg/providers/` :

```
pkg/providers/
└── your_provider.go
```

#### 2. Implémenter l'interface Provider

Votre fournisseur doit implémenter l'interface `Provider` définie dans `pkg/providers/types.go` :

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
    // Implémenter la complétion de chat avec streaming
}
```

#### 3. Enregistrer dans la factory

Ajoutez votre fournisseur au switch de protocole dans `pkg/providers/factory.go` :

```go
case "your-provider":
    return NewYourProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
```

#### 4. Ajouter la configuration par défaut (optionnel)

Ajoutez une entrée par défaut dans `pkg/config/defaults.go` :

```go
{
    ModelName: "your-model",
    Model:     "your-provider/model-name",
    APIKey:    "",
},
```

#### 5. Ajouter le support d'authentification (optionnel)

Si votre fournisseur nécessite OAuth ou une authentification spéciale, ajoutez un cas dans `cmd/picoclaw/internal/auth/helpers.go` :

```go
case "your-provider":
    authLoginYourProvider()
```

#### 6. Configurer via `config.json`

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

## Tester votre implémentation

### Commandes CLI

```bash
# S'authentifier avec un fournisseur
picoclaw auth login --provider your-provider

# Lister les modèles (pour Antigravity)
picoclaw auth models

# Démarrer la passerelle
picoclaw gateway

# Exécuter un agent avec un modèle spécifique
picoclaw agent -m "Hello" --model your-model
```

### Variables d'environnement pour les tests

```bash
# Remplacer le modèle par défaut
export PICOCLAW_AGENTS_DEFAULTS_MODEL=your-model

# Remplacer les paramètres du fournisseur
export PICOCLAW_MODEL_LIST='[{"model_name":"your-model","model":"your-provider/model-name","api_key":"..."}]'
```

---

## Références

- **Fichiers source :**
  - `pkg/providers/antigravity_provider.go` - Implémentation du fournisseur Antigravity
  - `pkg/auth/oauth.go` - Implémentation du flux OAuth
  - `pkg/auth/store.go` - Stockage des identifiants d'authentification (`~/.picoclaw/auth.json`)
  - `pkg/providers/factory.go` - Factory des fournisseurs et routage de protocole
  - `pkg/providers/types.go` - Définitions de l'interface fournisseur
  - `cmd/picoclaw/internal/auth/helpers.go` - Commandes CLI d'authentification

- **Documentation :**
  - `docs/ANTIGRAVITY_USAGE.md` - Guide d'utilisation d'Antigravity
  - `docs/migration/model-list-migration.md` - Guide de migration

---

## Notes

1. **Projet Google Cloud :** Antigravity nécessite que Gemini for Google Cloud soit activé sur votre projet Google Cloud
2. **Quotas :** Utilise les quotas du projet Google Cloud (pas de facturation séparée)
3. **Accès aux modèles :** Les modèles disponibles dépendent de la configuration de votre projet Google Cloud
4. **Blocs de réflexion :** Les modèles Claude via Antigravity nécessitent un traitement spécial des blocs de réflexion avec signatures
5. **Assainissement des schémas :** Les schémas d'outils doivent être assainis pour supprimer les mots-clés JSON Schema non supportés

---

---

## Gestion des erreurs courantes

### 1. Limitation de débit (HTTP 429)

Antigravity retourne une erreur 429 lorsque les quotas du projet/modèle sont épuisés. La réponse d'erreur contient souvent un `quotaResetDelay` dans le champ `details`.

**Exemple d'erreur 429 :**
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

### 2. Réponses vides (modèles restreints)

Certains modèles peuvent apparaître dans la liste des modèles disponibles mais retourner une réponse vide (200 OK mais flux SSE vide). Cela se produit généralement pour les modèles en préversion ou restreints que le projet actuel n'a pas la permission d'utiliser.

**Traitement :** Traiter les réponses vides comme des erreurs informant l'utilisateur que le modèle pourrait être restreint ou invalide pour son projet.

---

## Dépannage

### "Token expired" (jeton expiré)
- Rafraîchir les jetons OAuth : `picoclaw auth login --provider antigravity`

### "Gemini for Google Cloud is not enabled" (Gemini for Google Cloud n'est pas activé)
- Activer l'API dans votre Google Cloud Console

### "Project not found" (projet non trouvé)
- Vérifier que votre projet Google Cloud a les API nécessaires activées
- Vérifier que l'ID du projet est correctement récupéré lors de l'authentification

### Les modèles n'apparaissent pas dans la liste
- Vérifier que l'authentification OAuth s'est terminée avec succès
- Vérifier le stockage du profil d'authentification : `~/.picoclaw/auth.json`
- Relancer `picoclaw auth login --provider antigravity`
