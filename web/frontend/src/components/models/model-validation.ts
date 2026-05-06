/**
 * Real-time model field validation utilities.
 * All checks are pure frontend, no network required.
 *
 * Messages use i18n keys with interpolation params — callers must
 * translate them via t(key, params).
 */

import {
  findClosestProvider,
  KNOWN_PROVIDER_KEYS,
  PROVIDER_ALIASES,
} from "./provider-registry"

export type ValidationLevel = "error" | "warning" | "success"

export interface FieldValidation {
  level: ValidationLevel
  messageKey: string
  messageParams?: Record<string, string>
  fix?: string
}

/**
 * Validate a model identifier string with optional provider context.
 * Returns validation result with optional one-click fix suggestion.
 */
export function validateModelField(
  input: string,
  selectedProvider?: string,
): FieldValidation {
  const trimmed = input.trim()
  if (!trimmed) return { level: "success", messageKey: "" }

  // Hard errors
  if (/\s/.test(trimmed)) {
    return {
      level: "error",
      messageKey: "models.validation.whitespace",
      fix: trimmed.replace(/\s+/g, "/"),
    }
  }
  if (trimmed.startsWith("/")) {
    return {
      level: "error",
      messageKey: "models.validation.leadingSlash",
      fix: trimmed.replace(/^\/+/, ""),
    }
  }
  if (trimmed.includes("//")) {
    return {
      level: "error",
      messageKey: "models.validation.consecutiveSlash",
      fix: trimmed.replace(/\/+/g, "/"),
    }
  }

  const slashIdx = trimmed.indexOf("/")
  if (slashIdx === -1) {
    // No provider prefix — when a provider is already selected,
    // the model ID is provider-local and needs no prefix.
    if (selectedProvider) {
      return {
        level: "success",
        messageKey: "models.validation.parsed",
        messageParams: { provider: selectedProvider, model: trimmed },
      }
    }
    return {
      level: "warning",
      messageKey: "models.validation.defaultToOpenAI",
      fix: `openai/${trimmed}`,
    }
  }

  const provider = trimmed.slice(0, slashIdx)
  const model = trimmed.slice(slashIdx + 1)
  if (!model) {
    return { level: "error", messageKey: "models.validation.emptyModel" }
  }

  if (!KNOWN_PROVIDER_KEYS.has(provider)) {
    // Check aliases
    const alias = PROVIDER_ALIASES[provider]
    if (alias) {
      return {
        level: "warning",
        messageKey: "models.validation.shouldUse",
        messageParams: { provider, alias },
        fix: `${alias}/${model}`,
      }
    }
    // Typo check
    const closest = findClosestProvider(provider)
    if (closest) {
      return {
        level: "warning",
        messageKey: "models.validation.didYouMean",
        messageParams: { closest },
        fix: `${closest}/${model}`,
      }
    }
    return {
      level: "warning",
      messageKey: "models.validation.unknownProvider",
      messageParams: { provider },
    }
  }

  return {
    level: "success",
    messageKey: "models.validation.parsed",
    messageParams: { provider, model },
  }
}
