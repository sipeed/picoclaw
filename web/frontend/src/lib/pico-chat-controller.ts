import { getDefaultStore } from "jotai"
import { toast } from "sonner"

import { getPicoToken } from "@/api/pico"
import { getSessionHistory } from "@/api/sessions"
import i18n from "@/i18n"
import {
  clearStoredSessionId,
  generateSessionId,
  normalizeUnixTimestamp,
  readStoredSessionId,
} from "@/lib/pico-chat-state"
import { type ChatMessage, getChatState, updateChatStore } from "@/store/chat"
import { gatewayAtom } from "@/store/gateway"

interface PicoMessage {
  type: string
  id?: string
  session_id?: string
  timestamp?: number | string
  payload?: Record<string, unknown>
}

const store = getDefaultStore()

let wsRef: WebSocket | null = null
let isConnecting = false
let msgIdCounter = 0
let activeSessionIdRef = getChatState().activeSessionId
let initialized = false
let unsubscribeGateway: (() => void) | null = null
let hydratePromise: Promise<void> | null = null
let connectionGeneration = 0

async function loadSessionMessages(sessionId: string): Promise<ChatMessage[]> {
  const detail = await getSessionHistory(sessionId)
  const fallbackTime = detail.updated

  return detail.messages.map((message, index) => ({
    id: `hist-${index}-${Date.now()}`,
    role: message.role,
    content: message.content,
    timestamp: fallbackTime,
  }))
}

function handlePicoMessage(message: PicoMessage) {
  const payload = message.payload || {}

  switch (message.type) {
    case "message.create": {
      const content = (payload.content as string) || ""
      const messageId = (payload.message_id as string) || `pico-${Date.now()}`
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
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: prev.messages.map((msg) =>
          msg.id === messageId ? { ...msg, content } : msg,
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

    case "error":
      console.error("Pico error:", payload)
      updateChatStore({ isTyping: false })
      break

    case "pong":
      break

    default:
      console.log("Unknown pico message type:", message.type)
  }
}

function setActiveSessionId(sessionId: string) {
  activeSessionIdRef = sessionId
  updateChatStore({ activeSessionId: sessionId })
}

export async function connectChat() {
  if (store.get(gatewayAtom).status !== "running") {
    return
  }

  if (
    isConnecting ||
    (wsRef &&
      (wsRef.readyState === WebSocket.OPEN ||
        wsRef.readyState === WebSocket.CONNECTING))
  ) {
    return
  }

  const generation = connectionGeneration + 1
  connectionGeneration = generation
  isConnecting = true
  updateChatStore({ connectionState: "connecting" })

  try {
    const { token, ws_url } = await getPicoToken()

    if (generation !== connectionGeneration) {
      return
    }

    if (!token) {
      console.error("No pico token available")
      updateChatStore({ connectionState: "error" })
      isConnecting = false
      return
    }

    let finalWsUrl = ws_url
    try {
      const parsedUrl = new URL(ws_url)
      const isLocalHost =
        parsedUrl.hostname === "localhost" ||
        parsedUrl.hostname === "127.0.0.1" ||
        parsedUrl.hostname === "0.0.0.0"
      const isBrowserLocal =
        window.location.hostname === "localhost" ||
        window.location.hostname === "127.0.0.1"

      if (isLocalHost && !isBrowserLocal) {
        parsedUrl.hostname = window.location.hostname
        finalWsUrl = parsedUrl.toString()
      }
    } catch (error) {
      console.warn("Could not parse ws_url:", error)
    }

    const url = `${finalWsUrl}?session_id=${encodeURIComponent(activeSessionIdRef)}`
    // Send token as a subprotocol so it doesn't end up in the URL.
    const socket = new WebSocket(url, [`token.${token}`])

    if (generation !== connectionGeneration) {
      socket.close()
      return
    }

    socket.onopen = () => {
      if (wsRef !== socket) {
        return
      }
      updateChatStore({ connectionState: "connected" })
      isConnecting = false
    }

    socket.onmessage = (event) => {
      try {
        const message: PicoMessage = JSON.parse(event.data)
        handlePicoMessage(message)
      } catch {
        console.warn("Non-JSON message from pico:", event.data)
      }
    }

    socket.onclose = () => {
      if (wsRef !== socket) {
        return
      }
      wsRef = null
      isConnecting = false
      updateChatStore({
        connectionState: "disconnected",
        isTyping: false,
      })
    }

    socket.onerror = () => {
      if (wsRef !== socket) {
        return
      }
      isConnecting = false
      updateChatStore({ connectionState: "error" })
    }

    wsRef = socket
  } catch (error) {
    if (generation !== connectionGeneration) {
      return
    }
    console.error("Failed to connect to pico:", error)
    updateChatStore({ connectionState: "error" })
    isConnecting = false
  }
}

export function disconnectChat() {
  connectionGeneration += 1

  const socket = wsRef
  wsRef = null
  isConnecting = false

  if (socket) {
    socket.close()
  }

  updateChatStore({
    connectionState: "disconnected",
    isTyping: false,
  })
}

export async function hydrateActiveSession() {
  if (hydratePromise) {
    return hydratePromise
  }

  const state = getChatState()
  const storedSessionId = readStoredSessionId()

  if (
    !storedSessionId ||
    state.hasHydratedActiveSession ||
    state.messages.length > 0 ||
    storedSessionId !== state.activeSessionId
  ) {
    if (!state.hasHydratedActiveSession) {
      updateChatStore({ hasHydratedActiveSession: true })
    }
    return
  }

  hydratePromise = loadSessionMessages(storedSessionId)
    .then((historyMessages) => {
      const currentState = getChatState()
      if (currentState.activeSessionId !== storedSessionId) {
        return
      }

      if (currentState.messages.length > 0) {
        updateChatStore({ hasHydratedActiveSession: true })
        return
      }

      updateChatStore({
        messages: historyMessages,
        isTyping: false,
        hasHydratedActiveSession: true,
      })
    })
    .catch((error) => {
      console.error("Failed to restore last session history:", error)

      const currentState = getChatState()
      if (currentState.activeSessionId !== storedSessionId) {
        return
      }

      if (currentState.messages.length > 0) {
        updateChatStore({ hasHydratedActiveSession: true })
        return
      }

      clearStoredSessionId()
      updateChatStore({
        messages: [],
        isTyping: false,
        hasHydratedActiveSession: true,
      })
    })
    .finally(() => {
      hydratePromise = null
    })

  return hydratePromise
}

export function sendChatMessage(content: string) {
  if (!wsRef || wsRef.readyState !== WebSocket.OPEN) {
    console.warn("WebSocket not connected")
    return
  }

  const id = `msg-${++msgIdCounter}-${Date.now()}`

  updateChatStore((prev) => ({
    messages: [
      ...prev.messages,
      { id, role: "user", content, timestamp: Date.now() },
    ],
    isTyping: true,
  }))

  wsRef.send(
    JSON.stringify({
      type: "message.send",
      id,
      payload: { content },
    }),
  )
}

export async function switchChatSession(sessionId: string) {
  if (sessionId === activeSessionIdRef) {
    return
  }

  try {
    const historyMessages = await loadSessionMessages(sessionId)

    disconnectChat()
    setActiveSessionId(sessionId)
    updateChatStore({
      messages: historyMessages,
      isTyping: false,
      hasHydratedActiveSession: true,
    })

    if (store.get(gatewayAtom).status === "running") {
      await connectChat()
    }
  } catch (error) {
    console.error("Failed to load session history:", error)
    toast.error(i18n.t("chat.historyOpenFailed"))
  }
}

export async function newChatSession() {
  if (getChatState().messages.length === 0) {
    return
  }

  disconnectChat()
  setActiveSessionId(generateSessionId())
  updateChatStore({
    messages: [],
    isTyping: false,
    hasHydratedActiveSession: true,
  })

  if (store.get(gatewayAtom).status === "running") {
    await connectChat()
  }
}

export function initializeChatStore() {
  if (initialized) {
    return
  }

  initialized = true
  activeSessionIdRef = getChatState().activeSessionId

  const syncConnectionWithGateway = () => {
    if (store.get(gatewayAtom).status === "running") {
      void connectChat()
      return
    }

    disconnectChat()
  }

  unsubscribeGateway = store.sub(gatewayAtom, syncConnectionWithGateway)

  if (!readStoredSessionId()) {
    updateChatStore({ hasHydratedActiveSession: true })
  }

  syncConnectionWithGateway()
}

export function teardownChatStore() {
  unsubscribeGateway?.()
  unsubscribeGateway = null
  initialized = false
  disconnectChat()
}
