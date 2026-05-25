export interface MarkdownNode {
  type?: string
  value?: string
  tagName?: string
  properties?: Record<string, unknown>
  children?: MarkdownNode[]
}

function toClassNameTokens(className: unknown): string[] {
  if (typeof className === "string") {
    return className.split(/\s+/).filter(Boolean)
  }

  if (Array.isArray(className)) {
    return className.filter(
      (token): token is string => typeof token === "string" && token.length > 0,
    )
  }

  return []
}

function findFirstDescendantByTagName(
  node: MarkdownNode | undefined,
  tagName: string,
): MarkdownNode | undefined {
  if (!node) {
    return undefined
  }

  if (node.tagName === tagName) {
    return node
  }

  if (!Array.isArray(node.children)) {
    return undefined
  }

  for (const child of node.children) {
    const match = findFirstDescendantByTagName(child, tagName)
    if (match) {
      return match
    }
  }

  return undefined
}

export function extractTextFromMarkdownNode(
  node: MarkdownNode | undefined,
): string {
  if (!node) {
    return ""
  }

  if (node.type === "text") {
    return typeof node.value === "string" ? node.value : ""
  }

  if (!Array.isArray(node.children)) {
    return ""
  }

  return node.children.map(extractTextFromMarkdownNode).join("")
}

export function extractCodeBlockLanguage(className: unknown): string | null {
  const languageToken = toClassNameTokens(className).find(
    (token) => token.startsWith("language-") && token.length > "language-".length,
  )

  return languageToken ? languageToken.slice("language-".length) : null
}

export function extractCodeBlockFromPreNode(node: MarkdownNode | undefined): {
  code: string
  language: string | null
} {
  const codeNode = findFirstDescendantByTagName(node, "code")

  return {
    code: extractTextFromMarkdownNode(codeNode ?? node),
    language: extractCodeBlockLanguage(codeNode?.properties?.className),
  }
}
