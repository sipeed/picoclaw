import { PROVIDER_ALIASES, PROVIDER_LABELS } from "./provider-registry"

export function getProviderKey(provider?: string): string {
  const normalized = provider?.trim().toLowerCase()
  if (!normalized) return "openai"
  return PROVIDER_ALIASES[normalized] ?? normalized
}

export function getProviderLabel(provider?: string): string {
  const prefix = getProviderKey(provider)
  return PROVIDER_LABELS[prefix] ?? prefix
}

export { PROVIDER_LABELS, PROVIDER_ALIASES }
