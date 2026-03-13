import { useCallback, useEffect, useRef, useState } from "react"

export function useWebSocket(path: string) {
  const [message, setMessage] = useState<string>("No messages yet")
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const url = `${protocol}//${window.location.host}${path}`
    const socket = new WebSocket(url)

    socket.onopen = () => {
      setConnected(true)
      setMessage("Connected to WebSocket server.")
    }

    socket.onmessage = (event) => {
      setMessage(event.data)
    }

    socket.onclose = () => {
      setConnected(false)
      setMessage("WebSocket connection closed.")
    }

    socket.onerror = (error) => {
      setConnected(false)
      setMessage("WebSocket error occurred.")
      console.error("WebSocket Error:", error)
    }

    wsRef.current = socket
  }, [path])

  useEffect(() => {
    return () => {
      wsRef.current?.close()
    }
  }, [])

  return { message, connected, connect }
}
