import { useAtomValue } from "jotai"
import { useEffect, useRef, useState } from "react"

import { clearGatewayLogs, getGatewayLogs } from "@/api/gateway"
import { gatewayAtom } from "@/store/gateway"

export function useGatewayLogs() {
  const [logs, setLogs] = useState<string[]>([])
  const [clearing, setClearing] = useState(false)
  const logOffsetRef = useRef(0)
  const logRunIdRef = useRef(-1)
  const syncTokenRef = useRef(0)

  const gateway = useAtomValue(gatewayAtom)

  const clearLogs = async () => {
    setClearing(true)
    try {
      const data = await clearGatewayLogs()
      syncTokenRef.current += 1
      setLogs([])
      logOffsetRef.current = data.log_total ?? 0
      if (data.log_run_id !== undefined) {
        logRunIdRef.current = data.log_run_id
      }
    } catch {
      // Ignore clear failures silently to avoid noisy transient errors.
    } finally {
      setClearing(false)
    }
  }

  useEffect(() => {
    let mounted = true
    let timeout: ReturnType<typeof setTimeout>

    const fetchLogs = async () => {
      if (
        !mounted ||
        !["running", "starting", "restarting"].includes(gateway.status)
      ) {
        if (mounted) {
          timeout = setTimeout(fetchLogs, 1000)
        }
        return
      }

      try {
        const requestToken = syncTokenRef.current
        const requestOffset = logOffsetRef.current
        const requestRunId = logRunIdRef.current
        const data = await getGatewayLogs({
          log_offset: requestOffset,
          log_run_id: requestRunId,
        })

        if (!mounted || requestToken !== syncTokenRef.current) {
          return
        }

        if (data.log_run_id !== undefined && data.log_run_id !== requestRunId) {
          logRunIdRef.current = data.log_run_id
          logOffsetRef.current = 0
          if (data.logs) {
            setLogs(data.logs)
            logOffsetRef.current = data.log_total || data.logs.length
          }
        } else if (data.logs && data.logs.length > 0) {
          const nextLogs = data.logs
          setLogs((prev) => [...prev, ...nextLogs])
          logOffsetRef.current =
            data.log_total || logOffsetRef.current + nextLogs.length
        }
      } catch {
        // Ignore simple fetch errors during polling.
      } finally {
        if (mounted) {
          timeout = setTimeout(fetchLogs, 1000)
        }
      }
    }

    fetchLogs()

    return () => {
      mounted = false
      clearTimeout(timeout)
    }
  }, [gateway.status])

  return {
    clearLogs,
    clearing,
    logs,
  }
}
