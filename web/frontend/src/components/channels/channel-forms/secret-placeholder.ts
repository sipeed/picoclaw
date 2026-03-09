export function maskedSecretPlaceholder(value: unknown, fallback = ""): string {
  const secret = typeof value === "string" ? value.trim() : ""
  if (!secret) {
    return fallback
  }

  const prefix = secret.slice(0, Math.min(4, secret.length))
  const suffix = secret.slice(-Math.min(3, secret.length))
  return `${prefix}***${suffix}`
}
