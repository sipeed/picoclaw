const PROVIDER_LABELS: Record<string, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  gemini: "Google Gemini",
  deepseek: "DeepSeek",
  qwen: "Qwen (阿里云)",
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
  mistral: "Mistral AI",
  avian: "Avian",
  vllm: "VLLM (local)",
  zhipu: "Zhipu AI (智谱)",
  mimo: "Xiaomi MiMo",
}

export function getProviderKey(model: string): string {
  return model.split("/")[0]
}

export function getProviderLabel(model: string): string {
  const prefix = getProviderKey(model)
  const labels: Record<string, string> = {
    ...PROVIDER_LABELS,
  }
  return labels[prefix] ?? prefix
}
