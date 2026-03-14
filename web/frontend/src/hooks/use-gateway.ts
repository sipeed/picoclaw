import { useAtomValue } from "jotai"
import { useCallback, useEffect, useState } from "react"

import {
  type GatewayStatusResponse,
  getGatewayStatus,
  restartGateway,
  startGateway,
  stopGateway,
} from "@/api/gateway"
import {
  applyGatewayStatusToStore,
  gatewayAtom,
  updateGatewayStore,
} from "@/store"

// Global variable to ensure we only have one SSE connection
let sseInitialized = false

export function useGateway() {
  const gateway = useAtomValue(gatewayAtom)
  const { status: state, canStart, startReason, passphraseState, restartRequired } = gateway
  const [loading, setLoading] = useState(false)

  const applyGatewayStatus = useCallback((data: GatewayStatusResponse) => {
    applyGatewayStatusToStore(data)
  }, [])

  // Initialize global SSE connection once
  useEffect(() => {
    if (sseInitialized) return
    sseInitialized = true

    getGatewayStatus()
      .then((data) => applyGatewayStatus(data))
      .catch(() => {
        updateGatewayStore({
          status: "unknown",
          canStart: true,
          startReason: "",
          passphraseState: "",
          restartRequired: false,
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
          applyGatewayStatus(data)
        }
      } catch {
        // ignore
      }
    }

    es.onerror = () => {
      // EventSource will auto-reconnect
      updateGatewayStore((prev) =>
        prev.status === "restarting" ? {} : { status: "unknown" },
      )
    }

    return () => {
      window.clearInterval(statusPoll)
      es.close()
      sseInitialized = false
    }
  }, [applyGatewayStatus])

  const start = useCallback(async () => {
    if (!canStart) return

    setLoading(true)
    try {
      await startGateway()
      // SSE will push the real state changes, but set optimistic state
      updateGatewayStore({ status: "starting" })
    } catch (err) {
      console.error("Failed to start gateway:", err)
      try {
        const status = await getGatewayStatus()
        applyGatewayStatus(status)
      } catch {
        updateGatewayStore({ status: "unknown" })
      }
    } finally {
      setLoading(false)
    }
  }, [applyGatewayStatus, canStart])

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

  const restart = useCallback(async () => {
    if (state !== "running") return

    const previousState = state
    const previousCanStart = canStart
    const previousRestartRequired = restartRequired

    setLoading(true)
    updateGatewayStore({
      status: "restarting",
      restartRequired: false,
    })

    try {
      await restartGateway()
    } catch (err) {
      console.error("Failed to restart gateway:", err)
      try {
        const status = await getGatewayStatus()
        applyGatewayStatus(status)
      } catch {
        updateGatewayStore({
          status: previousState,
          canStart: previousCanStart,
          restartRequired: previousRestartRequired,
        })
      }
    } finally {
      setLoading(false)
    }
  }, [applyGatewayStatus, canStart, restartRequired, state])

  return { state, loading, canStart, startReason, passphraseState, restartRequired, start, stop, restart }
}
