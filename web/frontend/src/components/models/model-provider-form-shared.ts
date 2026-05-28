import type { ModelProviderOption } from "@/api/models"

import { getProviderDefaultAPIBase } from "./provider-registry"

export function normalizeApiBase(value: string): string {
  return value.trim().replace(/\/+$/, "")
}

export function getEffectiveAPIBase(
  provider: string,
  apiBase: string,
  providerOptions?: ModelProviderOption[],
): string {
  return normalizeApiBase(
    apiBase || getProviderDefaultAPIBase(provider, providerOptions),
  )
}

export function getSubmittedAPIBase(apiBase: string): string | undefined {
  return normalizeApiBase(apiBase) || undefined
}
