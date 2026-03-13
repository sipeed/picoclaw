import { atom, getDefaultStore } from "jotai"

import {
  getInitialActiveSessionId,
  writeStoredSessionId,
} from "@/lib/pico-chat-state"

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: number | string
}

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error"

export interface ChatStoreState {
  messages: ChatMessage[]
  connectionState: ConnectionState
  isTyping: boolean
  activeSessionId: string
  hasHydratedActiveSession: boolean
}

type ChatStorePatch = Partial<ChatStoreState>

const DEFAULT_CHAT_STATE: ChatStoreState = {
  messages: [],
  connectionState: "disconnected",
  isTyping: false,
  activeSessionId: getInitialActiveSessionId(),
  hasHydratedActiveSession: false,
}

export const chatAtom = atom<ChatStoreState>(DEFAULT_CHAT_STATE)

const store = getDefaultStore()

export function getChatState() {
  return store.get(chatAtom)
}

export function updateChatStore(
  patch:
    | ChatStorePatch
    | ((prev: ChatStoreState) => ChatStorePatch | ChatStoreState),
) {
  store.set(chatAtom, (prev) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    const next = { ...prev, ...nextPatch }

    if (next.activeSessionId !== prev.activeSessionId) {
      writeStoredSessionId(next.activeSessionId)
    }

    return next
  })
}
