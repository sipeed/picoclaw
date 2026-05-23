import {
  Children,
  cloneElement,
  Fragment,
  isValidElement,
  type ReactNode,
} from "react"

export interface MarkdownNode {
  type?: string
  value?: string
  tagName?: string
  properties?: Record<string, unknown>
  children?: MarkdownNode[]
}

export function toClassNameTokens(className: unknown): string[] {
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

export function stripSingleTrailingLineBreak(value: string): string {
  return value.replace(/\r?\n$/, "")
}

export function extractCodeBlockFromPreNode(node: MarkdownNode | undefined): {
  code: string
  language: string | null
} {
  const codeNode = findFirstDescendantByTagName(node, "code")

  return {
    code: stripSingleTrailingLineBreak(extractTextFromMarkdownNode(codeNode ?? node)),
    language: extractCodeBlockLanguage(codeNode?.properties?.className),
  }
}

export function extractCodeBlockRenderState(children: ReactNode): {
  renderedContent: ReactNode
  className: string | undefined
} {
  const childNodes = Children.toArray(children)
  const codeChild = childNodes.find(
    (child) =>
      isValidElement<{ children?: ReactNode; className?: unknown }>(child) &&
      typeof child.type === "string" &&
      child.type === "code",
  )

  if (
    isValidElement<{ children?: ReactNode; className?: unknown }>(codeChild)
  ) {
    const classNameTokens = toClassNameTokens(codeChild.props.className)
    return {
      renderedContent: codeChild.props.children,
      className:
        classNameTokens.length > 0 ? classNameTokens.join(" ") : undefined,
    }
  }

  return {
    renderedContent: children,
    className: undefined,
  }
}

function mergeNodeLineGroups(
  currentLines: Node[][],
  nextLines: Node[][],
): Node[][] {
  if (nextLines.length === 0) {
    return currentLines
  }

  const mergedLines = currentLines.map((line) => [...line])
  mergedLines[mergedLines.length - 1].push(...nextLines[0])

  for (const line of nextLines.slice(1)) {
    mergedLines.push([...line])
  }

  return mergedLines
}

function splitDomNodeIntoLines(node: Node, ownerDocument: Document): Node[][] {
  if (node.nodeType === Node.TEXT_NODE) {
    return (node.textContent ?? "").split("\n").map((line) =>
      line.length > 0 ? [ownerDocument.createTextNode(line)] : [],
    )
  }

  if (node.nodeType !== Node.ELEMENT_NODE) {
    return [[]]
  }

  const element = node as Element
  if (element.tagName.toLowerCase() === "br") {
    return [
      [],
      [],
    ]
  }

  const childLines = splitHighlightedHtmlIntoNodeLines(
    Array.from(element.childNodes),
    ownerDocument,
  )

  return childLines.map((lineChildren) => {
    const clonedElement = element.cloneNode(false)
    for (const child of lineChildren) {
      clonedElement.appendChild(child)
    }

    return [clonedElement]
  })
}

function splitHighlightedHtmlIntoNodeLines(
  nodes: Node[],
  ownerDocument: Document,
): Node[][] {
  let lines: Node[][] = [[]]

  for (const node of nodes) {
    lines = mergeNodeLineGroups(
      lines,
      splitDomNodeIntoLines(node, ownerDocument),
    )
  }

  return lines
}

export function splitCodeIntoLines(code: string): string[] {
  return code.split("\n")
}

export function splitHighlightedHtmlIntoLines(highlightedHtml: string): string[] {
  if (typeof document === "undefined") {
    return splitCodeIntoLines(highlightedHtml)
  }

  const container = document.createElement("div")
  container.innerHTML = highlightedHtml

  return splitHighlightedHtmlIntoNodeLines(
    Array.from(container.childNodes),
    document,
  ).map((lineNodes) => {
    const lineContainer = document.createElement("div")
    for (const node of lineNodes) {
      lineContainer.appendChild(node)
    }

    return lineContainer.innerHTML
  })
}

export function trimTrailingEmptyStringLine(lines: string[]): string[] {
  if (lines.length > 1 && lines[lines.length - 1] === "") {
    return lines.slice(0, -1)
  }

  return lines
}

function isEmptyRenderedCodeNode(node: ReactNode): boolean {
  if (node === null || node === undefined || typeof node === "boolean") {
    return true
  }

  if (typeof node === "string" || typeof node === "number") {
    return String(node).length === 0
  }

  if (Array.isArray(node)) {
    return node.every(isEmptyRenderedCodeNode)
  }

  if (!isValidElement<{ children?: ReactNode }>(node)) {
    return false
  }

  return Children.toArray(node.props.children).every(isEmptyRenderedCodeNode)
}

export function trimTrailingEmptyRenderedCodeLine(
  lines: ReactNode[][],
): ReactNode[][] {
  if (
    lines.length > 1 &&
    lines[lines.length - 1].every(isEmptyRenderedCodeNode)
  ) {
    return lines.slice(0, -1)
  }

  return lines
}

function mergeReactLineGroups(
  currentLines: ReactNode[][],
  nextLines: ReactNode[][],
): ReactNode[][] {
  if (nextLines.length === 0) {
    return currentLines
  }

  const mergedLines = currentLines.map((line) => [...line])
  mergedLines[mergedLines.length - 1].push(...nextLines[0])

  for (const line of nextLines.slice(1)) {
    mergedLines.push([...line])
  }

  return mergedLines
}

function splitTextNodeIntoLines(value: string | number): ReactNode[][] {
  return String(value).split("\n").map((line) => (line.length > 0 ? [line] : []))
}

function splitReactNodeIntoLines(node: ReactNode): ReactNode[][] {
  if (node === null || node === undefined || typeof node === "boolean") {
    return [[]]
  }

  if (typeof node === "string" || typeof node === "number") {
    return splitTextNodeIntoLines(node)
  }

  if (Array.isArray(node)) {
    return splitRenderedCodeContentIntoLines(node)
  }

  if (!isValidElement<{ children?: ReactNode }>(node)) {
    return [[node]]
  }

  if (node.type === Fragment) {
    return splitRenderedCodeContentIntoLines(Children.toArray(node.props.children))
  }

  if (typeof node.type === "string" && node.type === "br") {
    return [
      [],
      [],
    ]
  }

  const childLines = splitRenderedCodeContentIntoLines(
    Children.toArray(node.props.children),
  )

  return childLines.map((lineChildren, lineIndex) => [
    cloneElement(
      node,
      {
        key: `${node.key ?? "code-line"}-${lineIndex}`,
      },
      ...lineChildren,
    ),
  ])
}

export function splitRenderedCodeContentIntoLines(
  content: ReactNode,
): ReactNode[][] {
  const contentNodes = Array.isArray(content) ? content : [content]
  let lines: ReactNode[][] = [[]]

  for (const node of contentNodes) {
    lines = mergeReactLineGroups(lines, splitReactNodeIntoLines(node))
  }

  return lines
}
