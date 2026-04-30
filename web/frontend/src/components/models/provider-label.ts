import type { ModelProviderOption } from "@/api/models"

const PROVIDER_LABELS: Record<string, string> = {
  openai: "OpenAI",
  bedrock: "AWS Bedrock",
  elevenlabs: "ElevenLabs ASR",
  anthropic: "Anthropic",
  "anthropic-messages": "Anthropic Messages",
  azure: "Azure OpenAI",
  gemini: "Google Gemini",
  deepseek: "DeepSeek",
  "coding-plan": "Alibaba Coding Plan",
  "coding-plan-anthropic": "Alibaba Coding Plan (Anthropic)",
  "qwen-portal": "Qwen (阿里云)",
  "qwen-intl": "Qwen International",
  "qwen-us": "Qwen US",
  moonshot: "Moonshot (月之暗面)",
  groq: "Groq",
  openrouter: "OpenRouter",
  nvidia: "NVIDIA",
  cerebras: "Cerebras",
  volcengine: "Volcengine (火山引擎)",
  shengsuanyun: "ShengsuanYun (神算云)",
  antigravity: "Google Code Assist",
  "github-copilot": "GitHub Copilot",
  "claude-cli": "Claude CLI (local)",
  "codex-cli": "Codex CLI (local)",
  ollama: "Ollama (local)",
  lmstudio: "LM Studio (local)",
  litellm: "LiteLLM",
  mistral: "Mistral AI",
  avian: "Avian",
  vllm: "VLLM (local)",
  zhipu: "Zhipu AI (智谱)",
  zai: "Z.ai",
  mimo: "Xiaomi MiMo",
  venice: "Venice AI",
  vivgrid: "Vivgrid",
  minimax: "MiniMax",
  longcat: "LongCat",
  modelscope: "ModelScope (魔搭社区)",
  novita: "Novita AI",
}

const PROVIDER_ALIASES: Record<string, string> = {
  qwen: "qwen-portal",
  "qwen-international": "qwen-intl",
  "dashscope-intl": "qwen-intl",
  "z.ai": "zai",
  "z-ai": "zai",
  google: "gemini",
  "google-antigravity": "antigravity",
}

export const PROVIDER_PRIORITY: Record<string, number> = {
  volcengine: 0,
  openai: 1,
  gemini: 2,
  anthropic: 3,
  bedrock: 4,
  elevenlabs: 5,
  "anthropic-messages": 6,
  zhipu: 7,
  deepseek: 8,
  openrouter: 9,
  "qwen-portal": 10,
  "qwen-intl": 11,
  "qwen-us": 12,
  moonshot: 13,
  groq: 14,
  "coding-plan": 15,
  "coding-plan-anthropic": 16,
  "github-copilot": 17,
  antigravity: 18,
  nvidia: 19,
  cerebras: 20,
  shengsuanyun: 21,
  venice: 22,
  vivgrid: 23,
  minimax: 24,
  longcat: 25,
  modelscope: 26,
  mistral: 27,
  avian: 28,
  novita: 29,
  azure: 30,
  litellm: 31,
  ollama: 32,
  vllm: 33,
  lmstudio: 34,
  "claude-cli": 35,
  "codex-cli": 36,
  zai: 37,
  mimo: 38,
}

export function getProviderKey(provider?: string): string {
  const normalized = provider?.trim().toLowerCase()
  if (!normalized) return "openai"
  return PROVIDER_ALIASES[normalized] ?? normalized
}

export function getProviderLabel(provider?: string): string {
  const prefix = getProviderKey(provider)
  return PROVIDER_LABELS[prefix] ?? prefix
}

export function findProviderOption(
  provider: string | undefined,
  options: ModelProviderOption[],
): ModelProviderOption | undefined {
  const providerKey = getProviderKey(provider)
  return options.find((option) => option.id === providerKey)
}

export function getProviderDefaultAPIBase(
  provider: string | undefined,
  options: ModelProviderOption[],
): string {
  return findProviderOption(provider, options)?.default_api_base ?? ""
}

export function getSortedProviderOptions(
  options: ModelProviderOption[],
): ModelProviderOption[] {
  return [...options].sort((a, b) => {
    const aPriority = PROVIDER_PRIORITY[a.id] ?? Number.MAX_SAFE_INTEGER
    const bPriority = PROVIDER_PRIORITY[b.id] ?? Number.MAX_SAFE_INTEGER
    if (aPriority !== bPriority) {
      return aPriority - bPriority
    }
    return getProviderLabel(a.id).localeCompare(getProviderLabel(b.id))
  })
}

export function getProviderDefaultAuthMethod(
  provider: string | undefined,
  options: ModelProviderOption[],
): string {
  return findProviderOption(provider, options)?.default_auth_method ?? ""
}

export function isProviderAuthMethodLocked(
  provider: string | undefined,
  options: ModelProviderOption[],
): boolean {
  return findProviderOption(provider, options)?.auth_method_locked === true
}
