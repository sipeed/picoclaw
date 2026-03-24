import type { PicoMessage } from "@/features/chat/protocol"

/**
 * Reads a fetch() response body as text/event-stream and invokes onMessage for each data payload.
 * Comment lines (:) are ignored. Supports simple multi-line data: fields (joined with newlines).
 */
export async function readPicoSSEStream(
  body: ReadableStream<Uint8Array>,
  signal: AbortSignal,
  onMessage: (data: string) => void,
): Promise<void> {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buf = ""

  while (!signal.aborted) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    buf += decoder.decode(value, { stream: true })

    for (;;) {
      const sep = buf.indexOf("\n\n")
      if (sep < 0) {
        break
      }
      const raw = buf.slice(0, sep)
      buf = buf.slice(sep + 2)

      const lines = raw.split("\n")
      const dataParts: string[] = []
      for (const line of lines) {
        if (line.startsWith(":")) {
          continue
        }
        if (line.startsWith("data:")) {
          dataParts.push(line.slice(5).trimStart())
        }
      }
      const data = dataParts.join("\n").trimEnd()
      if (data.length > 0) {
        onMessage(data)
      }
    }
  }
}

export function parsePicoSSEData(data: string): PicoMessage | null {
  try {
    return JSON.parse(data) as PicoMessage
  } catch {
    console.warn("Non-JSON SSE data from pico:", data)
    return null
  }
}
