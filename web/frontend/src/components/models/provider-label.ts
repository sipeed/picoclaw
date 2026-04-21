const PROVIDER_LABELS: Record<string, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  azure: "Azure OpenAI",
  gemini: "Google Gemini",
  deepseek: "DeepSeek",
  "qwen-portal": "Qwen (阿里云)",
  "qwen-intl": "Qwen International",
  moonshot: "Moonshot (月之暗面)",
  groq: "Groq",
  openrouter: "OpenRouter",
  nvidia: "NVIDIA",
  cerebras: "Cerebras",
  volcengine: "Volcengine (火山引擎)",
  shengsuanyun: "ShengsuanYun (神算云)",
  antigravity: "Google Code Assist",
  "github-copilot": "GitHub Copilot",
  ollama: "Ollama (local)",
  lmstudio: "LM Studio (local)",
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

export function getProviderKey(provider?: string): string {
  const normalized = provider?.trim().toLowerCase()
  if (!normalized) return "openai"
  return PROVIDER_ALIASES[normalized] ?? normalized
}

export function getProviderLabel(provider?: string): string {
  const prefix = getProviderKey(provider)
  return PROVIDER_LABELS[prefix] ?? prefix
}
