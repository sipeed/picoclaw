import { useAtom } from "jotai"
import { useCallback, useEffect, useState } from "react"

import {
  type GatewayStatusResponse,
  getGatewayStatus,
  startGateway,
  stopGateway,
} from "@/api/gateway"
import { gatewayAtom } from "@/store"

// Global variable to ensure we only have one SSE connection
let sseInitialized = false

export function useGateway() {
  const [{ status: state, canStart }, setGateway] = useAtom(gatewayAtom)
  const [loading, setLoading] = useState(false)

  const applyGatewayStatus = useCallback(
    (data: GatewayStatusResponse) => {
      setGateway((prev) => ({
        ...prev,
        status: data.gateway_status ?? "unknown",
        canStart: data.gateway_start_allowed ?? true,
      }))
    },
    [setGateway],
  )

  // Initialize global SSE connection once
  useEffect(() => {
    if (sseInitialized) return
    sseInitialized = true

    getGatewayStatus()
      .then((data) => applyGatewayStatus(data))
      .catch(() => {
        setGateway({
          status: "unknown",
          canStart: true,
        })
      })

    const statusPoll = window.setInterval(() => {
      getGatewayStatus()
        .then((data) => applyGatewayStatus(data))
        .catch(() => {
          // ignore polling errors
        })
    }, 5000)

    // Subscribe to SSE for real-time updates globally
    const es = new EventSource("/api/gateway/events")

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (
          data.gateway_status ||
          typeof data.gateway_start_allowed === "boolean"
        ) {
          setGateway((prev) => ({
            ...prev,
            status: data.gateway_status ?? prev.status,
            canStart:
              typeof data.gateway_start_allowed === "boolean"
                ? data.gateway_start_allowed
                : prev.canStart,
          }))
        }
      } catch {
        // ignore
      }
    }

    es.onerror = () => {
      // EventSource will auto-reconnect
      setGateway((prev) => ({ ...prev, status: "unknown" }))
    }

    return () => {
      window.clearInterval(statusPoll)
      es.close()
      sseInitialized = false
    }
  }, [applyGatewayStatus, setGateway])

  const start = useCallback(async () => {
    if (!canStart) return

    setLoading(true)
    try {
      await startGateway()
      // SSE will push the real state changes, but set optimistic state
      setGateway((prev) => ({ ...prev, status: "starting" }))
    } catch (err) {
      console.error("Failed to start gateway:", err)
      try {
        const status = await getGatewayStatus()
        applyGatewayStatus(status)
      } catch {
        setGateway((prev) => ({ ...prev, status: "unknown" }))
      }
    } finally {
      setLoading(false)
    }
  }, [applyGatewayStatus, canStart, setGateway])

  const stop = useCallback(async () => {
    setLoading(true)
    try {
      await stopGateway()
    } catch (err) {
      console.error("Failed to stop gateway:", err)
    } finally {
      setLoading(false)
    }
  }, [])

  return { state, loading, canStart, start, stop }
}
