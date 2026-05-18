interface ClipboardTextareaLike {
  value: string
  style: {
    position: string
    left: string
  }
  setAttribute(name: string, value: string): void
  select(): void
}

interface ClipboardBodyLike {
  appendChild(node: ClipboardTextareaLike): void
  removeChild(node: ClipboardTextareaLike): void
}

interface ClipboardDocumentLike {
  body: ClipboardBodyLike
  createElement(tagName: "textarea"): ClipboardTextareaLike
  execCommand(command: "copy"): boolean
}

interface ClipboardNavigatorLike {
  clipboard?: {
    writeText(text: string): Promise<void>
  }
}

export interface ClipboardEnvironment {
  document?: ClipboardDocumentLike
  navigator?: ClipboardNavigatorLike
}

function getDefaultClipboardEnvironment(): ClipboardEnvironment {
  return {
    document:
      typeof document === "undefined"
        ? undefined
        : (document as unknown as ClipboardDocumentLike),
    navigator:
      typeof navigator === "undefined"
        ? undefined
        : (navigator as unknown as ClipboardNavigatorLike),
  }
}

export async function copyText(
  text: string,
  environment: ClipboardEnvironment = getDefaultClipboardEnvironment(),
): Promise<boolean> {
  try {
    if (environment.navigator?.clipboard?.writeText) {
      await environment.navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // HTTP or restricted environments can reject Clipboard API writes.
  }

  if (!environment.document) {
    return false
  }

  const textArea = environment.document.createElement("textarea")
  textArea.value = text
  textArea.setAttribute("readonly", "")
  textArea.style.position = "fixed"
  textArea.style.left = "-9999px"
  environment.document.body.appendChild(textArea)
  textArea.select()

  try {
    return environment.document.execCommand("copy")
  } finally {
    environment.document.body.removeChild(textArea)
  }
}
