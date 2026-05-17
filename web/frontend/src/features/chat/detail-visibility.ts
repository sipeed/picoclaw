export type AssistantDetailVisibility =
  | "none"
  | "thought"
  | "tool_calls"
  | "all"

export type AssistantDetailMessageKind =
  | "normal"
  | "thought"
  | "tool_calls"
  | undefined

interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

interface AssistantDetailVisibilityDecision {
  value: AssistantDetailVisibility
  newValueAction: "keep" | "write" | "remove"
  removeLegacyValue: boolean
}

export const ASSISTANT_DETAIL_VISIBILITY_STORAGE_KEY =
  "picoclaw:chat-assistant-detail-visibility"
export const LEGACY_SHOW_ASSISTANT_DETAILS_STORAGE_KEY =
  "picoclaw:chat-show-thoughts"
export const DEFAULT_ASSISTANT_DETAIL_VISIBILITY: AssistantDetailVisibility =
  "all"

function getSafeLocalStorage(): StorageLike | undefined {
  try {
    return globalThis.localStorage
  } catch {
    return undefined
  }
}

function serializeAssistantDetailVisibility(
  value: AssistantDetailVisibility,
): string {
  return JSON.stringify(value)
}

function parseStoredValue(rawValue: string | null): unknown {
  if (rawValue === null) {
    return undefined
  }

  try {
    return JSON.parse(rawValue)
  } catch {
    return rawValue.trim()
  }
}

function parseAssistantDetailVisibility(
  rawValue: unknown,
): AssistantDetailVisibility | undefined {
  if (typeof rawValue !== "string") {
    return undefined
  }

  const normalized = rawValue.trim().toLowerCase()
  if (
    normalized === "none" ||
    normalized === "thought" ||
    normalized === "tool_calls" ||
    normalized === "all"
  ) {
    return normalized
  }

  return undefined
}

function parseLegacyShowAssistantDetails(
  rawValue: unknown,
): boolean | undefined {
  if (typeof rawValue === "boolean") {
    return rawValue
  }

  if (typeof rawValue !== "string") {
    return undefined
  }

  const normalized = rawValue.trim().toLowerCase()
  if (normalized === "true") {
    return true
  }
  if (normalized === "false") {
    return false
  }

  return undefined
}

export function resolveAssistantDetailVisibilityPreference(
  storedValue: string | null,
  legacyStoredValue: string | null,
): AssistantDetailVisibilityDecision {
  const nextValue = parseAssistantDetailVisibility(
    parseStoredValue(storedValue),
  )
  if (nextValue) {
    return {
      value: nextValue,
      newValueAction:
        storedValue === serializeAssistantDetailVisibility(nextValue)
          ? "keep"
          : "write",
      removeLegacyValue: legacyStoredValue !== null,
    }
  }

  const legacyValue = parseLegacyShowAssistantDetails(
    parseStoredValue(legacyStoredValue),
  )
  if (legacyValue !== undefined) {
    return {
      value: legacyValue ? "all" : "none",
      newValueAction: "write",
      removeLegacyValue: legacyStoredValue !== null,
    }
  }

  return {
    value: DEFAULT_ASSISTANT_DETAIL_VISIBILITY,
    newValueAction: storedValue !== null ? "remove" : "keep",
    removeLegacyValue: legacyStoredValue !== null,
  }
}

export function syncAssistantDetailVisibilityStorage(
  storage?: StorageLike,
): AssistantDetailVisibility {
  const resolvedStorage = storage ?? getSafeLocalStorage()
  if (!resolvedStorage) {
    return DEFAULT_ASSISTANT_DETAIL_VISIBILITY
  }

  let decision: AssistantDetailVisibilityDecision
  try {
    decision = resolveAssistantDetailVisibilityPreference(
      resolvedStorage.getItem(ASSISTANT_DETAIL_VISIBILITY_STORAGE_KEY),
      resolvedStorage.getItem(LEGACY_SHOW_ASSISTANT_DETAILS_STORAGE_KEY),
    )
  } catch {
    return DEFAULT_ASSISTANT_DETAIL_VISIBILITY
  }

  if (decision.newValueAction === "write") {
    try {
      resolvedStorage.setItem(
        ASSISTANT_DETAIL_VISIBILITY_STORAGE_KEY,
        serializeAssistantDetailVisibility(decision.value),
      )
    } catch {
      // Ignore migration write failures and keep the parsed preference value.
    }
  } else if (decision.newValueAction === "remove") {
    try {
      resolvedStorage.removeItem(ASSISTANT_DETAIL_VISIBILITY_STORAGE_KEY)
    } catch {
      // Ignore cleanup failures and keep the parsed preference value.
    }
  }

  if (decision.removeLegacyValue) {
    try {
      resolvedStorage.removeItem(LEGACY_SHOW_ASSISTANT_DETAILS_STORAGE_KEY)
    } catch {
      // Ignore cleanup failures and keep the parsed preference value.
    }
  }

  return decision.value
}

export const assistantDetailVisibilityStorage = {
  getItem(): AssistantDetailVisibility {
    return syncAssistantDetailVisibilityStorage()
  },
  setItem(key: string, newValue: AssistantDetailVisibility) {
    const storage = getSafeLocalStorage()
    if (!storage) {
      return
    }

    try {
      storage.setItem(key, serializeAssistantDetailVisibility(newValue))
      storage.removeItem(LEGACY_SHOW_ASSISTANT_DETAILS_STORAGE_KEY)
    } catch {
      // Ignore storage write failures and keep the in-memory atom state.
    }
  },
  removeItem(key: string) {
    const storage = getSafeLocalStorage()
    if (!storage) {
      return
    }

    try {
      storage.removeItem(key)
    } catch {
      // Ignore storage write failures and keep the in-memory atom state.
    }
  },
  subscribe(key: string, callback: (value: AssistantDetailVisibility) => void) {
    if (
      typeof window === "undefined" ||
      typeof window.addEventListener !== "function"
    ) {
      return undefined
    }

    const handleStorage = (event: StorageEvent) => {
      const storage = getSafeLocalStorage()
      if (
        !storage ||
        event.storageArea !== storage ||
        (event.key !== key &&
          event.key !== LEGACY_SHOW_ASSISTANT_DETAILS_STORAGE_KEY)
      ) {
        return
      }

      callback(syncAssistantDetailVisibilityStorage(storage))
    }

    window.addEventListener("storage", handleStorage)
    return () => window.removeEventListener("storage", handleStorage)
  },
}

export function shouldShowAssistantMessage(
  visibility: AssistantDetailVisibility,
  kind: AssistantDetailMessageKind,
): boolean {
  if (kind !== "thought" && kind !== "tool_calls") {
    return true
  }

  if (visibility === "all") {
    return true
  }

  if (visibility === "none") {
    return false
  }

  return visibility === kind
}
