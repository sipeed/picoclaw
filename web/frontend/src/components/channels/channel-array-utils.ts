const ALLOW_FROM_HIDDEN_CHARS_RE =
  /\u200b|\u200c|\u200d|\u200e|\u200f|\u202a|\u202b|\u202c|\u202d|\u202e|\u2060|\u2061|\u2062|\u2063|\u2064|\u2066|\u2067|\u2068|\u2069|\ufeff/g

function normalizeStringListItems(
  items: string[],
  options: { stripHiddenChars?: boolean } = {},
): string[] {
  const result: string[] = []
  const seen = new Set<string>()

  for (const item of items) {
    const normalized = options.stripHiddenChars
      ? item.replace(ALLOW_FROM_HIDDEN_CHARS_RE, "")
      : item
    const trimmed = normalized.trim()
    if (trimmed.length === 0 || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    result.push(trimmed)
  }

  return result
}

function splitStringList(
  raw: string,
  separators: RegExp,
  options: { stripHiddenChars?: boolean } = {},
): string[] {
  if (raw.trim() === "") {
    return []
  }
  return normalizeStringListItems(raw.split(separators), options)
}

export function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.filter((item): item is string => typeof item === "string")
}

export function parseAllowFromInput(raw: string): string[] {
  return splitStringList(raw, /[,\uFF0C、;；\n\r\t]+/, {
    stripHiddenChars: true,
  })
}

export function parseConservativeStringListInput(raw: string): string[] {
  return splitStringList(raw, /[,\uFF0C\n\r\t]+/)
}

export function normalizeAllowFromValues(value: unknown): string[] {
  return normalizeStringListItems(asStringArray(value), {
    stripHiddenChars: true,
  })
}

export function mergeUniqueStringItems(
  currentItems: string[],
  nextItems: string[],
): string[] {
  return normalizeStringListItems([...currentItems, ...nextItems])
}

export function serializeStringArrayForSubmit(value: unknown): unknown {
  if (!Array.isArray(value)) {
    return value
  }
  return normalizeStringListItems(asStringArray(value)).join("\n")
}
