import { useAtomValue } from "jotai"
import { useCallback, useEffect, useState } from "react"

import { restartGateway, startGateway, stopGateway } from "@/api/gateway"
import {
  beginGatewayStoppingTransition,
  cancelGatewayStoppingTransition,
  gatewayAtom,
  refreshGatewayState,
  subscribeGatewayPolling,
  updateGatewayStore,
} from "@/store"

export function useGateway() {
  const gateway = useAtomValue(gatewayAtom)
  const { status: state, canStart, restartRequired } = gateway
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    return subscribeGatewayPolling()
  }, [])

  const start = useCallback(async () => {
    if (!canStart) return

    setLoading(true)
    try {
      await startGateway()
      updateGatewayStore({
        status: "starting",
        restartRequired: false,
      })
    } catch (err) {
      console.error("Failed to start gateway:", err)
    } finally {
      await refreshGatewayState({ force: true })
      setLoading(false)
    }
  }, [canStart])

  const stop = useCallback(async () => {
    setLoading(true)
    beginGatewayStoppingTransition()
    try {
      await stopGateway()
    } catch (err) {
      console.error("Failed to stop gateway:", err)
      cancelGatewayStoppingTransition()
    } finally {
      await refreshGatewayState({ force: true })
      setLoading(false)
    }
  }, [])

  const restart = useCallback(async () => {
    if (state !== "running") return

    setLoading(true)
    try {
      await restartGateway()
      updateGatewayStore({
        status: "restarting",
        restartRequired: false,
      })
    } catch (err) {
      console.error("Failed to restart gateway:", err)
    } finally {
      await refreshGatewayState({ force: true })
      setLoading(false)
    }
  }, [state])

  return { state, loading, canStart, restartRequired, start, stop, restart }
}
