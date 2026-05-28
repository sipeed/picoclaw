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

// Frontend still needs the same trim/lower normalization as the backend
// NormalizeProvider before it can look up canonical IDs in provider_options.
// This helper does not define provider semantics; aliases and canonical IDs
// still come entirely from the backend payload.
function normalizeProvider(provider?: string): string {
  return provider?.trim().toLowerCase() || ""
}

function toCatalogEntry(option: ModelProviderOption): ProviderCatalogEntry {
  const defaultApiBase = option.default_api_base || undefined
  return {
    key: option.id,
    label: option.display_name || option.id,
    iconSlug: option.icon_slug || undefined,
    domain: option.domain || undefined,
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
    aliases: option.aliases || [],
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
  return getProviderAliasMap(backendOptions)[normalized] ?? normalized
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
  return getProviderCatalogMap(backendOptions).get(key)
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
