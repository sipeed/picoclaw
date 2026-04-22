import { toast } from "sonner"

import { normalizeUnixTimestamp } from "@/features/chat/state"
import {
  type AssistantMessageKind,
  type ChatAttachment,
  type ChatMessage,
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

function parseAssistantMessageKind(
  payload: Record<string, unknown>,
): AssistantMessageKind {
  return payload.thought === true ? "thought" : "normal"
}

function hasAssistantKindPayload(payload: Record<string, unknown>): boolean {
  return typeof payload.thought === "boolean"
}

function parseAttachments(
  payload: Record<string, unknown>,
): ChatAttachment[] | undefined {
  const raw = payload.attachments
  if (!Array.isArray(raw)) {
    return undefined
  }

  const attachments: ChatAttachment[] = []
  for (const item of raw) {
    if (!item || typeof item !== "object") {
      continue
    }

    const attachment = item as Record<string, unknown>
    const url = typeof attachment.url === "string" ? attachment.url : ""
    if (!url) {
      continue
    }

    const type =
      attachment.type === "audio" ||
      attachment.type === "video" ||
      attachment.type === "file" ||
      attachment.type === "image"
        ? attachment.type
        : "file"

    const filename =
      typeof attachment.filename === "string" ? attachment.filename : undefined
    const contentType =
      typeof attachment.content_type === "string"
        ? attachment.content_type
        : undefined

    attachments.push({
      type,
      url,
      ...(filename ? { filename } : {}),
      ...(contentType ? { contentType } : {}),
    })
  }

  return attachments.length > 0 ? attachments : undefined
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

function isToolFeedbackMessage(message: ChatMessage): boolean {
  if (message.role !== "assistant") {
    return false
  }

  const firstLine = message.content.split("\n", 1)[0]?.trim() ?? ""
  return /^🔧\s+`[^`]+`/.test(firstLine)
}

function findToolFeedbackMessageIndex(messages: ChatMessage[]): number {
  let lastUserIndex = -1
  for (let i = messages.length - 1; i >= 0; i -= 1) {
    if (messages[i].role === "user") {
      lastUserIndex = i
      break
    }
  }

  for (let i = messages.length - 1; i >= 0; i -= 1) {
    if (i <= lastUserIndex) {
      break
    }
    if (isToolFeedbackMessage(messages[i])) {
      return i
    }
  }
  return -1
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
    case "message.create":
    case "media.create": {
      const content = (payload.content as string) || ""
      const messageId = (payload.message_id as string) || `pico-${Date.now()}`
      const kind = parseAssistantMessageKind(payload)
      const attachments = parseAttachments(payload)
      const contextUsage = parseContextUsage(payload)
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
            attachments,
            timestamp,
          },
        ],
        isTyping: false,
        ...(contextUsage ? { contextUsage } : {}),
      }))
      break
    }

    case "message.update": {
      const content = (payload.content as string) || ""
      const messageId = payload.message_id as string
      const hasKind = hasAssistantKindPayload(payload)
      const kind = parseAssistantMessageKind(payload)
      const attachments = parseAttachments(payload)
      const contextUsage = parseContextUsage(payload)
      const timestamp =
        message.timestamp !== undefined &&
        Number.isFinite(Number(message.timestamp))
          ? normalizeUnixTimestamp(Number(message.timestamp))
          : Date.now()
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: (() => {
          let found = false
          const messages = prev.messages.map((msg) => {
            if (msg.id !== messageId) {
              return msg
            }
            found = true
            return {
              ...msg,
              id: messageId,
              content,
              ...(hasKind ? { kind } : {}),
              ...(attachments ? { attachments } : {}),
            }
          })
          if (found) {
            return messages
          }

          const fallbackIndex = findToolFeedbackMessageIndex(messages)
          if (fallbackIndex >= 0) {
            return messages.map((msg, index) =>
              index === fallbackIndex
                ? {
                    ...msg,
                    id: messageId,
                    content,
                    ...(hasKind ? { kind } : {}),
                    ...(attachments ? { attachments } : {}),
                  }
                : msg,
            )
          }

          return [
            ...messages,
            {
              id: messageId,
              role: "assistant" as const,
              content,
              ...(hasKind ? { kind } : {}),
              ...(attachments ? { attachments } : {}),
              timestamp,
            },
          ]
        })(),
        ...(contextUsage ? { contextUsage } : {}),
      }))
      break
    }

    case "message.delete": {
      const messageId = payload.message_id as string
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: (() => {
          const exactMessages = prev.messages.filter((msg) => msg.id !== messageId)
          if (exactMessages.length !== prev.messages.length) {
            return exactMessages
          }

          const fallbackIndex = findToolFeedbackMessageIndex(prev.messages)
          if (fallbackIndex < 0) {
            return prev.messages
          }

          return prev.messages.filter((_, index) => index !== fallbackIndex)
        })(),
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
