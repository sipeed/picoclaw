import { getDefaultStore } from "jotai"
import { toast } from "sonner"

import { getPicoToken } from "@/api/pico"
import {
  loadSessionMessages,
  mergeHistoryMessages,
} from "@/features/chat/history"
import { type PicoMessage, handlePicoMessage } from "@/features/chat/protocol"
import {
  clearStoredSessionId,
  generateSessionId,
  readStoredSessionId,
} from "@/features/chat/state"
import {
  invalidateSocket,
  isCurrentSocket,
} from "@/features/chat/websocket"
import i18n from "@/i18n"
import {
  type ChatAttachment,
  getChatState,
  updateChatStore,
} from "@/store/chat"
import { type GatewayState, gatewayAtom } from "@/store/gateway"

const store = getDefaultStore()

let wsRef: WebSocket | null = null
let isConnecting = false
let msgIdCounter = 0
let activeSessionIdRef = getChatState().activeSessionId
let initialized = false
let unsubscribeGateway: (() => void) | null = null
let hydratePromise: Promise<void> | null = null
let connectionGeneration = 0
let reconnectTimer: number | null = null
let reconnectAttempts = 0
let shouldMaintainConnection = false

function clearReconnectTimer() {
  if (reconnectTimer !== null) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function shouldReconnectFor(generation: number, sessionId: string): boolean {
  return (
    shouldMaintainConnection &&
    generation === connectionGeneration &&
    sessionId === activeSessionIdRef &&
    store.get(gatewayAtom).status === "running"
  )
}

function scheduleReconnect(generation: number, sessionId: string) {
  if (!shouldReconnectFor(generation, sessionId) || reconnectTimer !== null) {
    return
  }

  const delay = Math.min(1000 * 2 ** reconnectAttempts, 5000)
  reconnectAttempts += 1
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = null
    if (!shouldReconnectFor(generation, sessionId)) {
      return
    }
    void connectChat()
  }, delay)
}

function needsActiveSessionHydration(): boolean {
  const state = getChatState()
  const storedSessionId = readStoredSessionId()

  return Boolean(
    storedSessionId &&
    storedSessionId === state.activeSessionId &&
    !state.hasHydratedActiveSession,
  )
}

function setActiveSessionId(sessionId: string) {
  activeSessionIdRef = sessionId
  updateChatStore({ activeSessionId: sessionId })
}

function disconnectChatInternal({
  clearDesiredConnection,
}: {
  clearDesiredConnection: boolean
}) {
  connectionGeneration += 1
  clearReconnectTimer()

  if (clearDesiredConnection) {
    shouldMaintainConnection = false
  }

  const socket = wsRef
  wsRef = null
  isConnecting = false

  invalidateSocket(socket)

  updateChatStore({
    connectionState: "disconnected",
    isTyping: false,
  })
}

export async function connectChat() {
  if (
    store.get(gatewayAtom).status !== "running" ||
    needsActiveSessionHydration()
  ) {
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
  clearReconnectTimer()
  updateChatStore({ connectionState: "connecting" })

  try {
    const { token } = await getPicoToken()
    const sessionId = activeSessionIdRef

    if (generation !== connectionGeneration) {
      isConnecting = false
      return
    }

    if (!token) {
      console.error("No pico token available")
      updateChatStore({ connectionState: "error" })
      isConnecting = false
      scheduleReconnect(generation, sessionId)
      return
    }

    const wsScheme = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsUrl = `${wsScheme}//${window.location.host}/pico/ws`
    const url = `${wsUrl}?session_id=${encodeURIComponent(sessionId)}`
    const socket = new WebSocket(url, [`token.${token}`])

    if (generation !== connectionGeneration) {
      isConnecting = false
      invalidateSocket(socket)
      return
    }

    socket.onopen = () => {
      if (
        !isCurrentSocket({
          socket,
          currentSocket: wsRef,
          generation,
          currentGeneration: connectionGeneration,
          sessionId,
          currentSessionId: activeSessionIdRef,
        })
      ) {
        return
      }
      updateChatStore({ connectionState: "connected" })
      isConnecting = false
      reconnectAttempts = 0
    }

    socket.onmessage = (event) => {
      if (
        !isCurrentSocket({
          socket,
          currentSocket: wsRef,
          generation,
          currentGeneration: connectionGeneration,
          sessionId,
          currentSessionId: activeSessionIdRef,
        })
      ) {
        return
      }

      try {
        const message = JSON.parse(event.data) as PicoMessage
        handlePicoMessage(message, sessionId)
      } catch {
        console.warn("Non-JSON message from pico:", event.data)
      }
    }

    socket.onclose = () => {
      if (
        !isCurrentSocket({
          socket,
          currentSocket: wsRef,
          generation,
          currentGeneration: connectionGeneration,
          sessionId,
          currentSessionId: activeSessionIdRef,
        })
      ) {
        return
      }
      wsRef = null
      isConnecting = false
      updateChatStore({
        connectionState: "disconnected",
        isTyping: false,
      })
      scheduleReconnect(generation, sessionId)
    }

    socket.onerror = () => {
      if (
        !isCurrentSocket({
          socket,
          currentSocket: wsRef,
          generation,
          currentGeneration: connectionGeneration,
          sessionId,
          currentSessionId: activeSessionIdRef,
        })
      ) {
        return
      }
      isConnecting = false
      updateChatStore({ connectionState: "error" })
      scheduleReconnect(generation, sessionId)
    }

    wsRef = socket
  } catch (error) {
    if (generation !== connectionGeneration) {
      isConnecting = false
      return
    }
    console.error("Failed to connect to pico:", error)
    updateChatStore({ connectionState: "error" })
    isConnecting = false
    scheduleReconnect(generation, activeSessionIdRef)
  }
}

export function disconnectChat() {
  disconnectChatInternal({ clearDesiredConnection: true })
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
        updateChatStore({
          messages: mergeHistoryMessages(
            historyMessages,
            currentState.messages,
          ),
          hasHydratedActiveSession: true,
        })
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

interface SendChatMessageInput {
  content: string
  attachments?: ChatAttachment[]
}

export function sendChatMessage({
  content,
  attachments = [],
}: SendChatMessageInput) {
  if (!wsRef || wsRef.readyState !== WebSocket.OPEN) {
    console.warn("WebSocket not connected")
    return false
  }

  const normalizedContent = content.trim()
  const normalizedAttachments = attachments
    .filter((attachment) => attachment.type === "image" && attachment.url)
    .map((attachment) => ({ ...attachment }))

  if (!normalizedContent && normalizedAttachments.length === 0) {
    return false
  }

  const socket = wsRef
  const id = `msg-${++msgIdCounter}-${Date.now()}`

  updateChatStore((prev) => ({
    messages: [
      ...prev.messages,
      {
        id,
        role: "user",
        content: normalizedContent,
        attachments:
          normalizedAttachments.length > 0 ? normalizedAttachments : undefined,
        timestamp: Date.now(),
      },
    ],
    isTyping: true,
  }))

  try {
    socket.send(
      JSON.stringify({
        type: "message.send",
        id,
        payload: {
          content: normalizedContent,
          media: normalizedAttachments.map((attachment) => attachment.url),
        },
      }),
    )
    return true
  } catch (error) {
    console.error("Failed to send pico message:", error)
    updateChatStore((prev) => ({
      messages: prev.messages.filter((message) => message.id !== id),
      isTyping: false,
    }))
    return false
  }
}

export async function switchChatSession(sessionId: string) {
  if (sessionId === activeSessionIdRef) {
    return
  }

  try {
    const historyMessages = await loadSessionMessages(sessionId)

    disconnectChatInternal({ clearDesiredConnection: false })
    setActiveSessionId(sessionId)
    updateChatStore({
      messages: historyMessages,
      isTyping: false,
      hasHydratedActiveSession: true,
    })

    if (store.get(gatewayAtom).status === "running") {
      shouldMaintainConnection = true
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

  disconnectChatInternal({ clearDesiredConnection: false })
  setActiveSessionId(generateSessionId())
  updateChatStore({
    messages: [],
    isTyping: false,
    hasHydratedActiveSession: true,
  })

  if (store.get(gatewayAtom).status === "running") {
    shouldMaintainConnection = true
    await connectChat()
  }
}

export function initializeChatStore() {
  if (initialized) {
    return
  }

  initialized = true
  activeSessionIdRef = getChatState().activeSessionId
  let lastGatewayStatus: GatewayState | null = null

  const syncConnectionWithGateway = (force: boolean = false) => {
    const gatewayStatus = store.get(gatewayAtom).status
    if (!force && gatewayStatus === lastGatewayStatus) {
      return
    }
    lastGatewayStatus = gatewayStatus

    if (gatewayStatus === "running") {
      shouldMaintainConnection = true
      if (needsActiveSessionHydration()) {
        return
      }
      void connectChat()
      return
    }

    if (gatewayStatus === "stopped" || gatewayStatus === "error") {
      disconnectChatInternal({ clearDesiredConnection: true })
    }
  }

  unsubscribeGateway = store.sub(gatewayAtom, syncConnectionWithGateway)

  if (!readStoredSessionId()) {
    updateChatStore({ hasHydratedActiveSession: true })
    syncConnectionWithGateway(true)
    return
  }

  void hydrateActiveSession().finally(() => {
    if (!initialized) {
      return
    }
    syncConnectionWithGateway(true)
  })
}

export function teardownChatStore() {
  unsubscribeGateway?.()
  unsubscribeGateway = null
  initialized = false
  disconnectChat()
}
