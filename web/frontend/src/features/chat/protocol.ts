import { toast } from "sonner"

import {
  inferStructuredContentFromText,
  parseStructuredContent,
} from "@/features/chat/structured"
import { normalizeUnixTimestamp } from "@/features/chat/state"
import {
  getChatState,
  type AssistantMessageKind,
  type ContextUsage,
  updateChatStore,
} from "@/store/chat"

export interface PicoMessage {
  type: string
  id?: string
  session_id?: string
  timestamp?: number | string
  payload?: Record<string, unknown>
}

export interface PicoMessageHandleResult {
  shouldRefreshHistory: boolean
}

function parseAssistantMessageKind(
  payload: Record<string, unknown>,
): AssistantMessageKind {
  return payload.thought === true ? "thought" : "normal"
}

function hasAssistantKindPayload(payload: Record<string, unknown>): boolean {
  return typeof payload.thought === "boolean"
}

function parseContextUsage(
  payload: Record<string, unknown>,
): ContextUsage | undefined {
  const raw = payload.context_usage
  if (!raw || typeof raw !== "object") return undefined
  const obj = raw as Record<string, unknown>
  const used = Number(obj.used_tokens)
  const total = Number(obj.total_tokens)
  if (!Number.isFinite(used) || !Number.isFinite(total) || total <= 0)
    return undefined
  return {
    used_tokens: used,
    total_tokens: total,
    compress_at_tokens: Number(obj.compress_at_tokens) || 0,
    used_percent: Number(obj.used_percent) || 0,
  }
}

export function handlePicoMessage(
  message: PicoMessage,
  expectedSessionId: string,
): PicoMessageHandleResult {
  if (message.session_id && message.session_id !== expectedSessionId) {
    return { shouldRefreshHistory: false }
  }

  const payload = message.payload || {}

  switch (message.type) {
    case "message.create": {
      const content = (payload.content as string) || ""
      const messageId = (payload.message_id as string) || `pico-${Date.now()}`
      const kind = parseAssistantMessageKind(payload)
      const contextUsage = parseContextUsage(payload)
      const wasTyping = getChatState().isTyping
      const structured =
        parseStructuredContent(payload.structured) ??
        (kind === "normal" ? inferStructuredContentFromText(content) : undefined)
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
            structured,
            timestamp,
          },
        ],
        isTyping: false,
        ...(contextUsage ? { contextUsage } : {}),
      }))

      return {
        shouldRefreshHistory:
          kind === "normal" &&
          !structured &&
          content.trim().length > 0 &&
          wasTyping,
      }
    }

    case "message.update": {
      const hasContent = typeof payload.content === "string"
      const content = hasContent ? ((payload.content as string) || "") : ""
      const messageId = payload.message_id as string
      const hasKind = hasAssistantKindPayload(payload)
      const kind = parseAssistantMessageKind(payload)
      const structured =
        parseStructuredContent(payload.structured) ??
        (hasContent && kind === "normal"
          ? inferStructuredContentFromText(content)
          : undefined)
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: prev.messages.map((msg) =>
          msg.id === messageId
            ? {
                ...msg,
                ...(hasContent ? { content } : {}),
                ...(hasKind ? { kind } : {}),
                ...(structured ? { structured } : {}),
              }
            : msg,
        ),
      }))
      return { shouldRefreshHistory: false }
    }

    case "typing.start":
      updateChatStore({ isTyping: true })
      return { shouldRefreshHistory: false }

    case "typing.stop":
      updateChatStore({ isTyping: false })
      return { shouldRefreshHistory: true }

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
      return { shouldRefreshHistory: false }
    }

    case "pong":
      return { shouldRefreshHistory: false }

    default:
      console.log("Unknown pico message type:", message.type)
      return { shouldRefreshHistory: false }
  }

  return { shouldRefreshHistory: false }
}
