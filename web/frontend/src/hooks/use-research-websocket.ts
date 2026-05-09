import { useEffect, useRef, useState, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"

export interface WebSocketMessage {
  type: string
  payload: unknown
}

export interface AgentUpdate {
  id: string
  name: string
  active: boolean
  progress: number
  status: string
  type: string
}

export interface ReportUpdate {
  id: string
  title: string
  status: string
  progress: number
  words: number
  pages: number
}

export interface ConfigChange {
  type: string
  depth: string
  restrict_to_graph: boolean
}

export function useResearchWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
  const queryClient = useQueryClient()
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const connect = useCallback(() => {
    // Determine WebSocket URL based on current location
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsUrl = `${protocol}//${window.location.host}/ws/research`

    try {
      const ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        setIsConnected(true)
        console.log("[Research WS] Connected")
      }

      ws.onmessage = (event) => {
        // Handle multi-line messages ( WebSocket may batch messages)
        const messages = event.data.split("\n")

        messages.forEach((msgStr: string) => {
          if (!msgStr.trim()) return

          try {
            const message = JSON.parse(msgStr) as WebSocketMessage
            setLastMessage(message)

            // Update React Query cache based on message type
            switch (message.type) {
              case "agent_update": {
                const update = message.payload as AgentUpdate
                queryClient.setQueryData(["research", "agents"], (old: AgentUpdate[] | undefined) => {
                  if (!old) return [update]
                  return old.map((a) => (a.id === update.id ? update : a))
                })
                break
              }
              case "report_update": {
                const update = message.payload as ReportUpdate
                queryClient.setQueryData(["research", "reports"], (old: ReportUpdate[] | undefined) => {
                  if (!old) return [update]
                  return old.map((r) => (r.id === update.id ? update : r))
                })
                break
              }
              case "config_change": {
                const config = message.payload as ConfigChange
                queryClient.setQueryData(["research", "config"], config)
                break
              }
            }
          } catch (e) {
            console.warn("[Research WS] Failed to parse message:", e)
          }
        })
      }

      ws.onclose = () => {
        setIsConnected(false)
        console.log("[Research WS] Disconnected")

        // Auto-reconnect after 3 seconds
        reconnectTimeoutRef.current = setTimeout(() => {
          connect()
        }, 3000)
      }

      ws.onerror = (error) => {
        console.error("[Research WS] Error:", error)
      }

      wsRef.current = ws
    } catch (error) {
      console.error("[Research WS] Failed to connect:", error)
    }
  }, [queryClient])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  const sendMessage = useCallback((message: object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    }
  }, [])

  return {
    isConnected,
    lastMessage,
    connect,
    disconnect,
    sendMessage,
  }
}