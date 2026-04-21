import { atom, getDefaultStore } from "jotai"

import {
  getInitialActiveSessionId,
  writeStoredSessionId,
} from "@/features/chat/state"

export interface ChatAttachment {
  type: "image"
  url: string
  filename?: string
}

export interface ChatOptionItem {
  label: string
  value: string
  description?: string
}

export interface ChatActionItem {
  type?: string
  action?: string
  label: string
  value?: string
  url?: string
  variant?: "default" | "outline" | "secondary" | "ghost"
}

export interface ChatFieldItem {
  label: string
  value: string
}

export interface ChatListItem {
  label?: string
  text: string
}

export interface ChatTableBlock {
  type: "table"
  headers?: string[]
  rows: string[][]
}

export interface ChatTextBlock {
  type: "text" | "markdown"
  text: string
}

export interface ChatFieldsBlock {
  type: "fields"
  fields: ChatFieldItem[]
}

export interface ChatBadgeBlock {
  type: "badge"
  label: string
  status?: string
}

export interface ChatActionsBlock {
  type: "actions"
  actions: ChatActionItem[]
}

export interface ChatListBlock {
  type: "list"
  items: ChatListItem[]
}

export interface ChatImageBlock {
  type: "image"
  url: string
  alt?: string
}

export interface ChatDividerBlock {
  type: "divider"
}

export interface ChatJsonBlock {
  type: "json"
  data: unknown
}

export interface ChatUnknownBlock {
  type: "unknown"
  blockType: string
  raw: Record<string, unknown>
}

export type ChatCardBlock =
  | ChatTextBlock
  | ChatFieldsBlock
  | ChatBadgeBlock
  | ChatActionsBlock
  | ChatListBlock
  | ChatTableBlock
  | ChatImageBlock
  | ChatDividerBlock
  | ChatJsonBlock
  | ChatUnknownBlock

export interface ChatStructuredOptions {
  type: "options"
  options: ChatOptionItem[]
  mode?: "single" | "multiple"
  allowCustom?: boolean
  customPlaceholder?: string
  submitLabel?: string
  raw?: Record<string, unknown>
}

export interface ChatStructuredCard {
  type: "card"
  kind?: string
  version?: string
  title?: string
  blocks?: ChatCardBlock[]
  actions?: ChatActionItem[]
  raw?: Record<string, unknown>
}

export interface ChatFormField {
  name: string
  label: string
  fieldType?: string
  value?: string
  required?: boolean
  placeholder?: string
}

export interface ChatStructuredForm {
  type: "form"
  kind?: string
  version?: string
  title?: string
  content?: string
  fields?: ChatFormField[]
  actions?: ChatActionItem[]
  raw?: Record<string, unknown>
}

export interface ChatProgressStep {
  label: string
  status?: string
  detail?: string
}

export interface ChatStructuredProgress {
  type: "progress"
  kind?: string
  version?: string
  title?: string
  content?: string
  status?: string
  steps?: ChatProgressStep[]
  raw?: Record<string, unknown>
}

export interface ChatStructuredAlert {
  type: "alert"
  kind?: string
  version?: string
  title?: string
  level?: string
  content?: string
  actions?: ChatActionItem[]
  raw?: Record<string, unknown>
}

export type ChatTodoStatus = "not-started" | "in-progress" | "completed"

export interface ChatTodoItem {
  id?: string
  title: string
  status?: ChatTodoStatus
  detail?: string
}

export interface ChatStructuredTodo {
  type: "todo"
  kind?: string
  version?: string
  title?: string
  content?: string
  items?: ChatTodoItem[]
  raw?: Record<string, unknown>
}

export interface ChatStructuredUnknown {
  type: "unknown"
  kind?: string
  raw: Record<string, unknown>
}

export type ChatStructuredContent =
  | ChatStructuredOptions
  | ChatStructuredCard
  | ChatStructuredForm
  | ChatStructuredProgress
  | ChatStructuredAlert
  | ChatStructuredTodo
  | ChatStructuredUnknown

export type ChatStructuredValue = ChatStructuredContent | ChatStructuredContent[]

export type AssistantMessageKind = "normal" | "thought"

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: number | string
  kind?: AssistantMessageKind
  attachments?: ChatAttachment[]
  structured?: ChatStructuredValue
}

export interface ContextUsage {
  used_tokens: number
  total_tokens: number
  compress_at_tokens: number
  used_percent: number
}

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error"

export type ChatInteractionMode = "agent" | "ask" | "plan"

export interface ChatStoreState {
  messages: ChatMessage[]
  connectionState: ConnectionState
  isTyping: boolean
  activeSessionId: string
  hasHydratedActiveSession: boolean
  contextUsage?: ContextUsage
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

export const showThoughtsAtom = atom<boolean>(true)

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
