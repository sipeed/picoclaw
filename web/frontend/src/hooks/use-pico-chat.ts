import dayjs from "dayjs"
import { useAtomValue } from "jotai"

import {
  newChatSession,
  sendChatMessage,
  switchChatSession,
} from "@/features/chat/controller"
import { chatAtom } from "@/store/chat"

const UNIX_MS_THRESHOLD = 1e12

function normalizeUnixTimestamp(timestamp: number): number {
  return timestamp < UNIX_MS_THRESHOLD ? timestamp * 1000 : timestamp
}

function parseTimestamp(dateRaw: number | string | Date) {
  if (typeof dateRaw === "number") {
    return dayjs(normalizeUnixTimestamp(dateRaw))
  }

  if (typeof dateRaw === "string") {
    const trimmed = dateRaw.trim()
    if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
      const numeric = Number(trimmed)
      if (Number.isFinite(numeric)) {
        return dayjs(normalizeUnixTimestamp(numeric))
      }
    }
    return dayjs(trimmed)
  }

  return dayjs(dateRaw)
}

export function formatMessageTime(dateRaw: number | string | Date): string {
  const date = parseTimestamp(dateRaw)
  if (!date.isValid()) {
    return ""
  }
  const now = dayjs()

  const isToday = date.isSame(now, "day")
  const isThisYear = date.isSame(now, "year")

  if (isToday) {
    return date.format("LT")
  }

  if (isThisYear) {
    return date.format("MMM D LT")
  }

  return date.format("ll LT")
}

export function usePicoChat() {
  const { messages, connectionState, isTyping, activeSessionId, contextUsage } =
    useAtomValue(chatAtom)

  return {
    messages,
    connectionState,
    isTyping,
    activeSessionId,
    contextUsage,
    sendMessage: sendChatMessage,
    switchSession: switchChatSession,
    newChat: newChatSession,
  }
}
