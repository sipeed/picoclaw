export function maskedSecretPlaceholder(value: unknown, fallback = ""): string {
  const secret = typeof value === "string" ? value.trim() : ""
  if (!secret) {
    return fallback
  }

  // ensure at least 40% of the characters are masked for secrets of length 4 or more
  if (secret.length <= 6) {
    const first = secret[0]
    const last = secret[secret.length - 1]
    return `${first}***${last}`
  }

  if (secret.length <= 12) {
    const firstTwo = secret.slice(0, 2)
    const lastTwo = secret.slice(-2)
    return `${firstTwo}****${lastTwo}`
  }

  const prefix = secret.slice(0, 3)
  const suffix = secret.slice(-4)
  return `${prefix}*****${suffix}`
}
