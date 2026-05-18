/**
 * Unified provider registry — single source of truth for all provider metadata.
 * All consumer files (provider-label, provider-icon, models-page, add/edit sheets)
 * should derive their data from this registry.
 */

import type { ModelProviderOption } from "@/api/models"

export interface ProviderDefinition {
  key: string
  label: string
  labelZh?: string
  iconSlug?: string
  domain?: string
  defaultApiBase?: string
  requiresApiKey: boolean
  isLocal: boolean
  priority: number
  commonModels?: string[]
  aliases?: string[]
  /** Whether this provider supports the OpenAI-compatible /models listing endpoint. */
  supportsFetch?: boolean
}

export const PROVIDERS: ProviderDefinition[] = [
  {
    key: "openai",
    label: "OpenAI",
    iconSlug: "openai",
    domain: "openai.com",
    defaultApiBase: "https://api.openai.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 100,
    commonModels: ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1", "o3-mini"],
    aliases: ["gpt"],
    supportsFetch: true,
  },
  {
    key: "anthropic",
    label: "Anthropic",
    iconSlug: "anthropic",
    domain: "anthropic.com",
    defaultApiBase: "https://api.anthropic.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 95,
    commonModels: [
      "claude-sonnet-4-20250514",
      "claude-haiku-4-20250414",
      "claude-3-5-sonnet-20241022",
    ],
    aliases: ["claude"],
  },
  {
    key: "gemini",
    label: "Google Gemini",
    iconSlug: "googlegemini",
    domain: "gemini.google.com",
    defaultApiBase: "https://generativelanguage.googleapis.com/v1beta",
    requiresApiKey: true,
    isLocal: false,
    priority: 90,
    commonModels: ["gemini-2.0-flash", "gemini-2.5-pro", "gemini-1.5-flash"],
    aliases: ["google"],
  },
  {
    key: "deepseek",
    label: "DeepSeek",
    iconSlug: "deepseek",
    domain: "deepseek.com",
    defaultApiBase: "https://api.deepseek.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 85,
    commonModels: ["deepseek-chat", "deepseek-reasoner"],
    supportsFetch: true,
  },
  {
    key: "openrouter",
    label: "OpenRouter",
    iconSlug: "openrouter",
    domain: "openrouter.ai",
    defaultApiBase: "https://openrouter.ai/api/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 80,
    commonModels: [
      "openai/gpt-4o",
      "anthropic/claude-sonnet-4",
      "google/gemini-2.0-flash",
    ],
    supportsFetch: true,
  },
  {
    key: "qwen-portal",
    label: "Qwen",
    labelZh: "Qwen (阿里云)",
    iconSlug: "alibabacloud",
    domain: "qwenlm.ai",
    defaultApiBase: "https://dashscope.aliyuncs.com/compatible-mode/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 75,
    commonModels: ["qwen-max", "qwen-plus", "qwen-turbo"],
    aliases: ["qwen"],
    supportsFetch: true,
  },
  {
    key: "qwen-intl",
    label: "Qwen International",
    iconSlug: "alibabacloud",
    domain: "alibabacloud.com",
    defaultApiBase: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 74,
    commonModels: ["qwen-max", "qwen-plus", "qwen-turbo"],
    aliases: ["qwen-international", "dashscope-intl"],
    supportsFetch: true,
  },
  {
    key: "moonshot",
    label: "Moonshot",
    labelZh: "Moonshot (月之暗面)",
    domain: "moonshot.ai",
    defaultApiBase: "https://api.moonshot.cn/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 70,
    commonModels: ["moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"],
    supportsFetch: true,
  },
  {
    key: "volcengine",
    label: "Volcengine",
    labelZh: "Volcengine (火山引擎)",
    iconSlug: "bytedance",
    domain: "volcengine.com",
    defaultApiBase: "https://ark.cn-beijing.volces.com/api/v3",
    requiresApiKey: true,
    isLocal: false,
    priority: 69,
    commonModels: ["doubao-1.5-pro", "doubao-1.5-lite"],
    supportsFetch: true,
  },
  {
    key: "zhipu",
    label: "Zhipu AI",
    labelZh: "Zhipu AI (智谱)",
    iconSlug: "zhipu",
    domain: "zhipuai.cn",
    defaultApiBase: "https://open.bigmodel.cn/api/paas/v4",
    requiresApiKey: true,
    isLocal: false,
    priority: 68,
    commonModels: ["glm-4-plus", "glm-4-flash"],
    supportsFetch: true,
  },
  {
    key: "groq",
    label: "Groq",
    iconSlug: "groq",
    domain: "groq.com",
    defaultApiBase: "https://api.groq.com/openai/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 65,
    commonModels: ["llama-3.3-70b-versatile", "mixtral-8x7b-32768"],
    supportsFetch: true,
  },
  {
    key: "mistral",
    label: "Mistral AI",
    iconSlug: "mistralai",
    domain: "mistral.ai",
    defaultApiBase: "https://api.mistral.ai/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 64,
    commonModels: ["mistral-large-latest", "mistral-small-latest"],
    supportsFetch: true,
  },
  {
    key: "nvidia",
    label: "NVIDIA",
    iconSlug: "nvidia",
    domain: "nvidia.com",
    defaultApiBase: "https://integrate.api.nvidia.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 63,
    commonModels: ["meta/llama-3.1-405b-instruct"],
    supportsFetch: true,
  },
  {
    key: "cerebras",
    label: "Cerebras",
    iconSlug: "cerebras",
    domain: "cerebras.ai",
    defaultApiBase: "https://api.cerebras.ai/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 62,
    commonModels: ["llama3.1-8b", "llama3.1-70b"],
    supportsFetch: true,
  },
  {
    key: "azure",
    label: "Azure OpenAI",
    iconSlug: "microsoftazure",
    domain: "azure.com",
    requiresApiKey: true,
    isLocal: false,
    priority: 61,
    commonModels: ["gpt-4o", "gpt-4o-mini"],
  },
  {
    key: "github-copilot",
    label: "GitHub Copilot",
    iconSlug: "githubcopilot",
    domain: "github.com",
    requiresApiKey: false,
    isLocal: true,
    priority: 55,
  },
  {
    key: "antigravity",
    label: "Google Code Assist",
    domain: "antigravity.google",
    requiresApiKey: false,
    isLocal: false,
    priority: 54,
  },
  {
    key: "ollama",
    label: "Ollama",
    labelZh: "Ollama (本地)",
    iconSlug: "ollama",
    domain: "ollama.com",
    defaultApiBase: "http://localhost:11434/v1",
    requiresApiKey: false,
    isLocal: true,
    priority: 50,
    commonModels: ["llama3", "mistral", "codellama", "qwen2.5"],
    supportsFetch: true,
  },
  {
    key: "vllm",
    label: "VLLM",
    labelZh: "VLLM (本地)",
    domain: "vllm.ai",
    defaultApiBase: "http://localhost:8000/v1",
    requiresApiKey: false,
    isLocal: true,
    priority: 49,
    supportsFetch: true,
  },
  {
    key: "lmstudio",
    label: "LM Studio",
    labelZh: "LM Studio (本地)",
    domain: "lmstudio.ai",
    defaultApiBase: "http://localhost:1234/v1",
    requiresApiKey: false,
    isLocal: true,
    priority: 48,
    supportsFetch: true,
  },
  {
    key: "venice",
    label: "Venice AI",
    iconSlug: "venice",
    domain: "venice.ai",
    defaultApiBase: "https://api.venice.ai/api/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 45,
    supportsFetch: true,
  },
  {
    key: "shengsuanyun",
    label: "ShengsuanYun",
    labelZh: "ShengsuanYun (神算云)",
    domain: "shengsuanyun.com",
    defaultApiBase: "https://router.shengsuanyun.com/api/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 44,
    supportsFetch: true,
  },
  {
    key: "siliconflow",
    label: "SiliconFlow",
    labelZh: "硅基流动",
    domain: "siliconflow.cn",
    defaultApiBase: "https://api.siliconflow.cn/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 43.5,
    supportsFetch: true,
  },
  {
    key: "vivgrid",
    label: "Vivgrid",
    domain: "vivgrid.com",
    defaultApiBase: "https://api.vivgrid.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 43,
    supportsFetch: true,
  },
  {
    key: "minimax",
    label: "MiniMax",
    domain: "minimaxi.com",
    defaultApiBase: "https://api.minimaxi.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 42,
    supportsFetch: true,
  },
  {
    key: "longcat",
    label: "LongCat",
    domain: "longcat.chat",
    defaultApiBase: "https://api.longcat.chat/openai",
    requiresApiKey: true,
    isLocal: false,
    priority: 41,
    supportsFetch: true,
  },
  {
    key: "modelscope",
    label: "ModelScope",
    labelZh: "ModelScope (魔搭社区)",
    domain: "modelscope.cn",
    defaultApiBase: "https://api-inference.modelscope.cn/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 40,
    supportsFetch: true,
  },
  {
    key: "mimo",
    label: "Xiaomi MiMo",
    iconSlug: "xiaomi",
    domain: "xiaomi.com",
    defaultApiBase: "https://api.xiaomimimo.com/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 39,
    supportsFetch: true,
  },
  {
    key: "avian",
    label: "Avian",
    domain: "avian.io",
    defaultApiBase: "https://api.avian.io/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 38,
    supportsFetch: true,
  },
  {
    key: "zai",
    label: "Z.ai",
    domain: "z.ai",
    defaultApiBase: "https://api.z.ai/api/coding/paas/v4",
    requiresApiKey: true,
    isLocal: false,
    priority: 37,
    aliases: ["z.ai", "z-ai"],
    supportsFetch: true,
  },
  {
    key: "novita",
    label: "Novita AI",
    domain: "novita.ai",
    defaultApiBase: "https://api.novita.ai/openai",
    requiresApiKey: true,
    isLocal: false,
    priority: 36,
    supportsFetch: true,
  },
  {
    key: "litellm",
    label: "LiteLLM",
    domain: "litellm.ai",
    defaultApiBase: "http://localhost:4000/v1",
    requiresApiKey: true,
    isLocal: false,
    priority: 35,
    supportsFetch: true,
  },
]

// ── Derived data for consumers ───────────────────────────────────────────────

export const PROVIDER_MAP = new Map(PROVIDERS.map((p) => [p.key, p]))

export const PROVIDER_LABELS: Record<string, string> = Object.fromEntries(
  PROVIDERS.map((p) => [p.key, p.labelZh || p.label]),
)

export const PROVIDER_ALIASES: Record<string, string> = Object.fromEntries(
  PROVIDERS.flatMap((p) => (p.aliases || []).map((a) => [a, p.key])),
)

export const KNOWN_PROVIDER_KEYS = new Set(PROVIDERS.map((p) => p.key))

export const FETCHABLE_PROVIDER_KEYS = new Set(
  PROVIDERS.filter((p) => p.supportsFetch).map((p) => p.key),
)

export const PROVIDER_ICON_SLUGS: Record<string, string> = Object.fromEntries(
  PROVIDERS.filter((p) => p.iconSlug).map((p) => [p.key, p.iconSlug!]),
)

export const PROVIDER_DOMAINS: Record<string, string> = Object.fromEntries(
  PROVIDERS.filter((p) => p.domain).map((p) => [p.key, p.domain!]),
)

export const PROVIDER_PRIORITY: Record<string, number> = Object.fromEntries(
  PROVIDERS.map((p) => [p.key, p.priority]),
)

export const PROVIDER_API_BASES: Record<string, string> = Object.fromEntries(
  PROVIDERS.filter((p) => p.defaultApiBase).map((p) => [
    p.key,
    p.defaultApiBase!,
  ]),
)

/**
 * Find the closest known provider key by edit distance.
 * Returns the key if distance <= 2, otherwise undefined.
 */
export function findClosestProvider(input: string): string | undefined {
  const lower = input.toLowerCase()
  let best: string | undefined
  let bestDist = 3 // only accept distance <= 2

  for (const key of KNOWN_PROVIDER_KEYS) {
    const dist = editDistance(lower, key)
    if (dist < bestDist) {
      bestDist = dist
      best = key
    }
  }
  // Also check aliases
  for (const alias of Object.keys(PROVIDER_ALIASES)) {
    const dist = editDistance(lower, alias)
    if (dist < bestDist) {
      bestDist = dist
      best = PROVIDER_ALIASES[alias]
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

// ── Backend options merge ────────────────────────────────────────────────────

export interface MergedProvider extends ProviderDefinition {
  createAllowed: boolean
  defaultModelAllowed: boolean
  defaultAuthMethod?: string
  authMethodLocked?: boolean
}

/**
 * Merge the frontend PROVIDERS registry with backend provider_options.
 * Frontend provides presentation data (labels, icons, priority, etc.).
 * Backend provides authoritative availability and policy fields.
 */
export function mergeWithBackendOptions(
  backendOptions: ModelProviderOption[],
): MergedProvider[] {
  const backendMap = new Map(backendOptions.map((o) => [o.id, o]))
  const merged: MergedProvider[] = []

  // Start with frontend providers, enriched with backend policy
  for (const p of PROVIDERS) {
    const backend = backendMap.get(p.key)
    merged.push({
      ...p,
      createAllowed: backend?.create_allowed ?? false,
      defaultModelAllowed: backend?.default_model_allowed ?? false,
      defaultAuthMethod: backend?.default_auth_method,
      authMethodLocked: backend?.auth_method_locked,
    })
    if (backend) backendMap.delete(p.key)
  }

  // Add providers only known to the backend
  for (const [key, backend] of backendMap) {
    merged.push({
      key,
      label: key,
      requiresApiKey: !backend.empty_api_key_allowed,
      isLocal: backend.empty_api_key_allowed,
      priority: 0,
      createAllowed: backend.create_allowed,
      defaultModelAllowed: backend.default_model_allowed,
      defaultAuthMethod: backend.default_auth_method,
      authMethodLocked: backend.auth_method_locked,
      defaultApiBase: backend.default_api_base || undefined,
    })
  }

  return merged.sort((a, b) => b.priority - a.priority)
}
