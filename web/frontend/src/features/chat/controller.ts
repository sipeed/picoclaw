import { getDefaultStore } from "jotai"
import { toast } from "sonner"

import { launcherFetch } from "@/api/http"
import { getPicoToken } from "@/api/pico"
import {
  loadSessionMessages,
  mergeHistoryMessages,
} from "@/features/chat/history"
import { type PicoMessage, handlePicoMessage } from "@/features/chat/protocol"
import { parsePicoSSEData, readPicoSSEStream } from "@/features/chat/sse"
import {
  clearStoredSessionId,
  generateSessionId,
  readStoredSessionId,
} from "@/features/chat/state"
import {
  invalidateSocket,
  isCurrentSocket,
  normalizeWsUrlForBrowser,
} from "@/features/chat/websocket"
import i18n from "@/i18n"
import { getChatState, updateChatStore } from "@/store/chat"
import { type GatewayState, gatewayAtom } from "@/store/gateway"

const store = getDefaultStore()

const WS_CONNECT_TIMEOUT_MS = 8000

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

/** SSE long-poll stream active (fetch returned 200 and body is being read). */
let sseActive = false
let sseAbort: AbortController | null = null
let picoTokenRef: string | null = null
let picoSendUrlRef: string | null = null

function clearReconnectTimer() {
  if (reconnectTimer !== null) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function stopSSE() {
  sseAbort?.abort()
  sseAbort = null
  sseActive = false
  picoSendUrlRef = null
  picoTokenRef = null
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
  stopSSE()

  updateChatStore({
    connectionState: "disconnected",
    isTyping: false,
  })
}

function attachWebSocketHandlers(
  socket: WebSocket,
  generation: number,
  sessionId: string,
) {
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
}

/**
 * Try WebSocket first. Resolves false if connection fails or times out (then caller may use SSE).
 */
function tryOpenWebSocket(
  generation: number,
  sessionId: string,
  token: string,
  wsUrl: string,
): Promise<boolean> {
  return new Promise((resolve) => {
    const finalWsUrl = normalizeWsUrlForBrowser(wsUrl)
    const url = `${finalWsUrl}?session_id=${encodeURIComponent(sessionId)}`
    const socket = new WebSocket(url, [`token.${token}`])

    if (generation !== connectionGeneration) {
      invalidateSocket(socket)
      resolve(false)
      return
    }

    let settled = false
    const finishFail = () => {
      if (settled) {
        return
      }
      settled = true
      window.clearTimeout(timer)
      invalidateSocket(socket)
      try {
        socket.close()
      } catch {
        /* ignore */
      }
      resolve(false)
    }

    const timer = window.setTimeout(() => {
      if (socket.readyState !== WebSocket.OPEN) {
        finishFail()
      }
    }, WS_CONNECT_TIMEOUT_MS)

    socket.onerror = () => {
      if (socket.readyState !== WebSocket.OPEN) {
        finishFail()
      }
    }

    socket.onopen = () => {
      window.clearTimeout(timer)
      if (settled || generation !== connectionGeneration) {
        invalidateSocket(socket)
        resolve(false)
        return
      }
      settled = true
      wsRef = socket
      updateChatStore({ connectionState: "connected" })
      isConnecting = false
      reconnectAttempts = 0
      attachWebSocketHandlers(socket, generation, sessionId)
      resolve(true)
    }
  })
}

async function openSSETransport(
  generation: number,
  sessionId: string,
  token: string,
  eventsUrlRaw: string | undefined,
  sendUrlRaw: string | undefined,
): Promise<void> {
  const eventsPath = eventsUrlRaw?.trim() || "/pico/events"
  const sendPath = sendUrlRaw?.trim() || "/pico/send"

  const eventsURL = new URL(eventsPath, window.location.origin)
  eventsURL.searchParams.set("session_id", sessionId)

  const sendURL = new URL(sendPath, window.location.origin)

  picoTokenRef = token
  picoSendUrlRef = sendURL.toString()

  sseAbort = new AbortController()
  const signal = sseAbort.signal

  let res: Response
  try {
    res = await launcherFetch(eventsURL.toString(), {
      headers: { Authorization: `Bearer ${token}` },
      signal,
      credentials: "same-origin",
    })
  } catch {
    if (generation !== connectionGeneration) {
      return
    }
    stopSSE()
    isConnecting = false
    updateChatStore({ connectionState: "error" })
    scheduleReconnect(generation, sessionId)
    return
  }

  if (generation !== connectionGeneration) {
    stopSSE()
    return
  }

  if (!res.ok || !res.body) {
    stopSSE()
    isConnecting = false
    updateChatStore({ connectionState: "error" })
    scheduleReconnect(generation, sessionId)
    return
  }

  sseActive = true
  isConnecting = false
  reconnectAttempts = 0
  updateChatStore({ connectionState: "connected" })

  try {
    await readPicoSSEStream(res.body, signal, (data) => {
      if (generation !== connectionGeneration) {
        return
      }
      const message = parsePicoSSEData(data)
      if (message) {
        handlePicoMessage(message, sessionId)
      }
    })
  } catch {
    /* aborted or read error */
  } finally {
    sseActive = false
    sseAbort = null
    picoSendUrlRef = null
    picoTokenRef = null

    if (
      generation === connectionGeneration &&
      shouldReconnectFor(generation, sessionId)
    ) {
      updateChatStore({
        connectionState: "disconnected",
        isTyping: false,
      })
      scheduleReconnect(generation, sessionId)
    }
  }
}

export async function connectChat() {
  if (
    store.get(gatewayAtom).status !== "running" ||
    needsActiveSessionHydration()
  ) {
    return
  }

  if (isConnecting) {
    return
  }
  if (sseActive) {
    return
  }
  if (
    wsRef &&
    (wsRef.readyState === WebSocket.OPEN ||
      wsRef.readyState === WebSocket.CONNECTING)
  ) {
    return
  }

  const generation = connectionGeneration + 1
  connectionGeneration = generation
  isConnecting = true
  clearReconnectTimer()

  invalidateSocket(wsRef)
  wsRef = null
  stopSSE()

  updateChatStore({ connectionState: "connecting" })

  const sessionId = activeSessionIdRef

  try {
    const { token, ws_url, events_url, send_url } = await getPicoToken()

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

    const wsOk = await tryOpenWebSocket(
      generation,
      sessionId,
      token,
      ws_url,
    )

    if (generation !== connectionGeneration) {
      isConnecting = false
      return
    }

    if (wsOk) {
      return
    }

    isConnecting = true
    updateChatStore({ connectionState: "connecting" })
    await openSSETransport(generation, sessionId, token, events_url, send_url)
  } catch (error) {
    if (generation !== connectionGeneration) {
      isConnecting = false
      return
    }
    console.error("Failed to connect to pico:", error)
    stopSSE()
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

export async function sendChatMessage(content: string): Promise<boolean> {
  const id = `msg-${++msgIdCounter}-${Date.now()}`

  const optimistic = () =>
    updateChatStore((prev) => ({
      messages: [
        ...prev.messages,
        { id, role: "user" as const, content, timestamp: Date.now() },
      ],
      isTyping: true,
    }))

  const rollback = () =>
    updateChatStore((prev) => ({
      messages: prev.messages.filter((message) => message.id !== id),
      isTyping: false,
    }))

  if (wsRef && wsRef.readyState === WebSocket.OPEN) {
    optimistic()
    try {
      wsRef.send(
        JSON.stringify({
          type: "message.send",
          id,
          payload: { content },
        }),
      )
      return true
    } catch (error) {
      console.error("Failed to send pico message:", error)
      rollback()
      return false
    }
  }

  if (sseActive && picoSendUrlRef && picoTokenRef) {
    optimistic()
    try {
      const res = await launcherFetch(picoSendUrlRef, {
        method: "POST",
        credentials: "same-origin",
        headers: {
          Authorization: `Bearer ${picoTokenRef}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          type: "message.send",
          id,
          session_id: activeSessionIdRef,
          payload: { content },
        }),
      })
      if (!res.ok) {
        rollback()
        return false
      }
      return true
    } catch (error) {
      console.error("Failed to send pico message:", error)
      rollback()
      return false
    }
  }

  console.warn("Pico chat not connected")
  return false
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
