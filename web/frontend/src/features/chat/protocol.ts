import { toast } from "sonner"

import { normalizeUnixTimestamp } from "@/features/chat/state"
import { type AssistantMessageKind, updateChatStore } from "@/store/chat"

export interface PicoMessage {
  type: string
  id?: string
  session_id?: string
  timestamp?: number | string
  payload?: Record<string, unknown>
}

function parseAssistantMessageKind(
  payload: Record<string, unknown>,
): AssistantMessageKind {
  return payload.thought === true ? "thought" : "normal"
}

function hasAssistantKindPayload(payload: Record<string, unknown>): boolean {
  return typeof payload.thought === "boolean"
}

export function handlePicoMessage(
  message: PicoMessage,
  expectedSessionId: string,
) {
  if (message.session_id && message.session_id !== expectedSessionId) {
    return
  }

  const payload = message.payload || {}

  switch (message.type) {
    case "message.create": {
      const content = (payload.content as string) || ""
      const messageId = (payload.message_id as string) || `pico-${Date.now()}`
      const kind = parseAssistantMessageKind(payload)
      const timestamp =
        message.timestamp !== undefined &&
        Number.isFinite(Number(message.timestamp))
          ? normalizeUnixTimestamp(Number(message.timestamp))
          : Date.now()

      updateChatStore((prev) => ({
        messages: [
          ...prev.messages,
          {
            id: messageId,
            role: "assistant",
            content,
            kind,
            timestamp,
          },
        ],
        isTyping: false,
      }))
      break
    }

    case "message.update": {
      const content = (payload.content as string) || ""
      const messageId = payload.message_id as string
      const hasKind = hasAssistantKindPayload(payload)
      const kind = parseAssistantMessageKind(payload)
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: prev.messages.map((msg) =>
          msg.id === messageId
            ? {
                ...msg,
                content,
                ...(hasKind ? { kind } : {}),
              }
            : msg,
        ),
      }))
      break
    }

    case "typing.start":
      updateChatStore({ isTyping: true })
      break

    case "typing.stop":
      updateChatStore({ isTyping: false })
      break

    case "error": {
      const requestId =
        typeof payload.request_id === "string" ? payload.request_id : ""
      const errorMessage =
        typeof payload.message === "string" ? payload.message : ""

      console.error("Pico error:", payload)
      if (errorMessage) {
        toast.error(errorMessage)
      }
      updateChatStore((prev) => ({
        messages: requestId
          ? prev.messages.filter((msg) => msg.id !== requestId)
          : prev.messages,
        isTyping: false,
      }))
      break
    }

    case "pong":
      break

    default:
      console.log("Unknown pico message type:", message.type)
  }
}
