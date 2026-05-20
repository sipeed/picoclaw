import type { ModelProviderOption } from "@/api/models"

export interface ProviderCatalogEntry {
  key: string
  label: string
  iconSlug?: string
  domain?: string
  priority: number
  isLocal: boolean
  defaultApiBase?: string
  requiresApiKey: boolean
  createAllowed: boolean
  defaultModelAllowed: boolean
  supportsFetch: boolean
  defaultAuthMethod?: string
  authMethodLocked?: boolean
  emptyApiKeyAllowed?: boolean
  commonModels: string[]
  aliases: string[]
}

interface ProviderPresentationFallback {
  label: string
  iconSlug?: string
  domain?: string
  aliases?: string[]
}

const PROVIDER_PRESENTATION_FALLBACKS: Record<
  string,
  ProviderPresentationFallback
> = {
  openai: { label: "OpenAI", iconSlug: "openai", domain: "openai.com", aliases: ["gpt"] },
  anthropic: { label: "Anthropic", iconSlug: "anthropic", domain: "anthropic.com", aliases: ["claude"] },
  "anthropic-messages": { label: "Anthropic Messages", iconSlug: "anthropic", domain: "anthropic.com" },
  gemini: { label: "Google Gemini", iconSlug: "googlegemini", domain: "gemini.google.com", aliases: ["google"] },
  deepseek: { label: "DeepSeek", iconSlug: "deepseek", domain: "deepseek.com" },
  openrouter: { label: "OpenRouter", iconSlug: "openrouter", domain: "openrouter.ai" },
  "qwen-portal": { label: "Qwen", iconSlug: "alibabacloud", domain: "qwenlm.ai", aliases: ["qwen"] },
  "qwen-intl": {
    label: "Qwen International",
    iconSlug: "alibabacloud",
    domain: "alibabacloud.com",
    aliases: ["qwen-international", "dashscope-intl"],
  },
  "qwen-us": { label: "Qwen US", iconSlug: "alibabacloud", domain: "alibabacloud.com", aliases: ["dashscope-us"] },
  moonshot: { label: "Moonshot", domain: "moonshot.ai" },
  volcengine: { label: "Volcengine", iconSlug: "bytedance", domain: "volcengine.com" },
  zhipu: { label: "Zhipu AI", iconSlug: "zhipu", domain: "zhipuai.cn", aliases: ["glm"] },
  groq: { label: "Groq", iconSlug: "groq", domain: "groq.com" },
  mistral: { label: "Mistral AI", iconSlug: "mistralai", domain: "mistral.ai" },
  nvidia: { label: "NVIDIA", iconSlug: "nvidia", domain: "nvidia.com" },
  cerebras: { label: "Cerebras", iconSlug: "cerebras", domain: "cerebras.ai" },
  azure: { label: "Azure OpenAI", iconSlug: "microsoftazure", domain: "azure.com", aliases: ["azure-openai"] },
  bedrock: { label: "AWS Bedrock", iconSlug: "amazonwebservices", domain: "aws.amazon.com" },
  "github-copilot": { label: "GitHub Copilot", iconSlug: "githubcopilot", domain: "github.com", aliases: ["copilot"] },
  antigravity: { label: "Google Code Assist", domain: "antigravity.google", aliases: ["google-antigravity"] },
  "claude-cli": { label: "Claude CLI", iconSlug: "anthropic", domain: "anthropic.com", aliases: ["claudecli"] },
  "codex-cli": { label: "Codex CLI", iconSlug: "openai", domain: "openai.com", aliases: ["codexcli"] },
  ollama: { label: "Ollama", iconSlug: "ollama", domain: "ollama.com" },
  vllm: { label: "VLLM", domain: "vllm.ai" },
  lmstudio: { label: "LM Studio", domain: "lmstudio.ai" },
  elevenlabs: { label: "ElevenLabs ASR", iconSlug: "elevenlabs", domain: "elevenlabs.io" },
  venice: { label: "Venice AI", iconSlug: "venice", domain: "venice.ai" },
  shengsuanyun: { label: "ShengsuanYun", domain: "shengsuanyun.com" },
  siliconflow: { label: "SiliconFlow", domain: "siliconflow.cn" },
  vivgrid: { label: "Vivgrid", domain: "vivgrid.com" },
  minimax: { label: "MiniMax", domain: "minimaxi.com" },
  longcat: { label: "LongCat", domain: "longcat.chat" },
  modelscope: { label: "ModelScope", domain: "modelscope.cn" },
  mimo: { label: "Xiaomi MiMo", iconSlug: "xiaomi", domain: "xiaomi.com" },
  avian: { label: "Avian", domain: "avian.io" },
  zai: { label: "Z.ai", domain: "z.ai", aliases: ["z.ai", "z-ai"] },
  "alibaba-coding": {
    label: "Alibaba Coding Plan",
    iconSlug: "alibabacloud",
    domain: "alibabacloud.com",
    aliases: ["coding-plan", "qwen-coding"],
  },
  "alibaba-coding-anthropic": {
    label: "Alibaba Coding Plan (Anthropic)",
    iconSlug: "alibabacloud",
    domain: "alibabacloud.com",
    aliases: ["coding-plan-anthropic"],
  },
  novita: { label: "Novita AI", domain: "novita.ai" },
  litellm: { label: "LiteLLM", domain: "litellm.ai" },
}

// Frontend still needs the same trim/lower normalization as the backend
// NormalizeProvider before it can look up canonical IDs in provider_options.
// This helper does not define provider semantics; aliases and canonical IDs
// still come entirely from the backend payload.
function normalizeProvider(provider?: string): string {
  return provider?.trim().toLowerCase() || ""
}

function buildPresentationAliasMap(): Record<string, string> {
  const aliases: Record<string, string> = {}
  for (const [key, fallback] of Object.entries(PROVIDER_PRESENTATION_FALLBACKS)) {
    aliases[normalizeProvider(key)] = key
    for (const alias of fallback.aliases || []) {
      const normalized = normalizeProvider(alias)
      if (normalized) {
        aliases[normalized] = key
      }
    }
  }
  return aliases
}

const providerPresentationAliasMap = buildPresentationAliasMap()

function toCatalogEntry(option: ModelProviderOption): ProviderCatalogEntry {
  const fallback = PROVIDER_PRESENTATION_FALLBACKS[option.id]
  const defaultApiBase = option.default_api_base || undefined
  return {
    key: option.id,
    label: option.display_name || fallback?.label || option.id,
    iconSlug: option.icon_slug || fallback?.iconSlug || undefined,
    domain: option.domain || fallback?.domain || undefined,
    priority: option.priority ?? 0,
    isLocal: option.local === true,
    defaultApiBase,
    requiresApiKey: !option.empty_api_key_allowed,
    createAllowed: option.create_allowed,
    defaultModelAllowed: option.default_model_allowed,
    supportsFetch: option.supports_fetch === true,
    defaultAuthMethod: option.default_auth_method || undefined,
    authMethodLocked: option.auth_method_locked,
    emptyApiKeyAllowed: option.empty_api_key_allowed,
    commonModels: option.common_models || [],
    aliases: Array.from(new Set([...(option.aliases || []), ...(fallback?.aliases || [])])),
  }
}

function buildAliasMap(
  backendOptions?: ModelProviderOption[],
): Record<string, string> {
  const aliases: Record<string, string> = {}
  for (const option of backendOptions || []) {
    const key = normalizeProvider(option.id)
    if (!key) continue
    aliases[key] = option.id
    for (const alias of option.aliases || []) {
      const normalized = normalizeProvider(alias)
      if (normalized) {
        aliases[normalized] = option.id
      }
    }
  }
  return aliases
}

export function getProviderAliasMap(
  backendOptions?: ModelProviderOption[],
): Record<string, string> {
  return buildAliasMap(backendOptions)
}

export function getCanonicalProviderKey(
  provider?: string,
  backendOptions?: ModelProviderOption[],
): string {
  const normalized = normalizeProvider(provider)
  if (!normalized) return ""
  return (
    getProviderAliasMap(backendOptions)[normalized] ??
    providerPresentationAliasMap[normalized] ??
    normalized
  )
}

export function getKnownProviderKeys(
  backendOptions?: ModelProviderOption[],
): Set<string> {
  return new Set(getProviderCatalog(backendOptions).map((p) => p.key))
}

export function getProviderCatalog(
  backendOptions?: ModelProviderOption[],
): ProviderCatalogEntry[] {
  if (!backendOptions || backendOptions.length === 0) {
    return []
  }

  return [...backendOptions]
    .map(toCatalogEntry)
    .sort((a, b) => b.priority - a.priority)
}

export function getProviderCatalogMap(
  backendOptions?: ModelProviderOption[],
): Map<string, ProviderCatalogEntry> {
  return new Map(getProviderCatalog(backendOptions).map((p) => [p.key, p]))
}

export function getProviderCatalogEntry(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): ProviderCatalogEntry | undefined {
  const key = getCanonicalProviderKey(provider, backendOptions)
  if (!key) return undefined
  const catalogEntry = getProviderCatalogMap(backendOptions).get(key)
  if (catalogEntry) {
    return catalogEntry
  }

  const fallback = PROVIDER_PRESENTATION_FALLBACKS[key]
  if (!fallback) {
    return undefined
  }

  return {
    key,
    label: fallback.label,
    iconSlug: fallback.iconSlug,
    domain: fallback.domain,
    priority: 0,
    isLocal: false,
    requiresApiKey: true,
    createAllowed: false,
    defaultModelAllowed: false,
    supportsFetch: false,
    commonModels: [],
    aliases: fallback.aliases || [],
  }
}

export function getProviderDefaultAPIBase(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): string {
  return getProviderCatalogEntry(provider, backendOptions)?.defaultApiBase ?? ""
}

export function getProviderDefaultAuthMethod(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): string {
  return getProviderCatalogEntry(provider, backendOptions)?.defaultAuthMethod ?? ""
}

export function isProviderAuthMethodLocked(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): boolean {
  return getProviderCatalogEntry(provider, backendOptions)?.authMethodLocked === true
}

export function providerSupportsFetch(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): boolean {
  const key = getCanonicalProviderKey(provider, backendOptions)
  if (!key) return false
  return getProviderCatalogMap(backendOptions).get(key)?.supportsFetch === true
}

/**
 * Find the closest known provider key by edit distance.
 * Returns the key if distance <= 2, otherwise undefined.
 */
export function findClosestProvider(
  input: string,
  backendOptions?: ModelProviderOption[],
): string | undefined {
  const lower = input.toLowerCase()
  let best: string | undefined
  let bestDist = 3

  for (const key of getKnownProviderKeys(backendOptions)) {
    const dist = editDistance(lower, key)
    if (dist < bestDist) {
      bestDist = dist
      best = key
    }
  }

  for (const alias of Object.keys(getProviderAliasMap(backendOptions))) {
    const dist = editDistance(lower, alias)
    if (dist < bestDist) {
      bestDist = dist
      best = getProviderAliasMap(backendOptions)[alias]
    }
  }
  return best
}

function editDistance(a: string, b: string): number {
  const m = a.length
  const n = b.length
  const dp: number[][] = Array.from({ length: m + 1 }, () =>
    new Array(n + 1).fill(0),
  )
  for (let i = 0; i <= m; i++) dp[i][0] = i
  for (let j = 0; j <= n; j++) dp[0][j] = j
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] =
        a[i - 1] === b[j - 1]
          ? dp[i - 1][j - 1]
          : 1 + Math.min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
    }
  }
  return dp[m][n]
}
