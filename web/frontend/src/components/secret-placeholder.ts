export function maskedSecretPlaceholder(value: unknown, fallback = ""): string {
  const secret = typeof value === "string" ? value.trim() : ""
  if (!secret) {
    return fallback
  }

  if (secret.length < 7) {
    const first = secret[0]
    const last = secret[secret.length - 1]
    return `${first}***${last}`
  }

  const prefix = secret.slice(0, Math.min(3, secret.length))
  const suffix = secret.slice(-Math.min(4, secret.length))
  return `${prefix}***${suffix}`
}
