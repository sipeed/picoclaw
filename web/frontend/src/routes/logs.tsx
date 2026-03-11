import { IconTrash } from "@tabler/icons-react"
import { createFileRoute } from "@tanstack/react-router"
import { useAtomValue } from "jotai"
import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { clearGatewayLogs, getGatewayStatus } from "@/api/gateway"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { gatewayAtom } from "@/store/gateway"

export const Route = createFileRoute("/logs")({
  component: LogsPage,
})

function LogsPage() {
  const { t } = useTranslation()
  const [logs, setLogs] = useState<string[]>([])
  const [clearing, setClearing] = useState(false)
  const logOffsetRef = useRef<number>(0)
  const logRunIdRef = useRef<number>(-1)
  const syncTokenRef = useRef<number>(0)
  const scrollRef = useRef<HTMLDivElement>(null)

  const gateway = useAtomValue(gatewayAtom)

  const handleClearLogs = async () => {
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
      // Only fetch logs if the gateway is running or starting
      if (
        !mounted ||
        (gateway.status !== "running" && gateway.status !== "starting")
      ) {
        if (mounted) {
          // Still poll the state, but maybe at a slower rate, or we just rely on SSE for status
          // and restart fast polling when it's running. Let's just re-evaluate every second
          timeout = setTimeout(fetchLogs, 1000)
        }
        return
      }

      try {
        const requestToken = syncTokenRef.current
        const requestOffset = logOffsetRef.current
        const requestRunId = logRunIdRef.current
        const data = await getGatewayStatus({
          log_offset: requestOffset,
          log_run_id: requestRunId,
        })

        if (!mounted || requestToken !== syncTokenRef.current) return

        if (data.log_run_id !== undefined && data.log_run_id !== requestRunId) {
          logRunIdRef.current = data.log_run_id
          logOffsetRef.current = 0
          if (data.logs) {
            setLogs(data.logs)
            logOffsetRef.current = data.log_total || data.logs.length
          }
        } else if (data.logs && data.logs.length > 0) {
          setLogs((prev) => [...prev, ...data.logs!])
          logOffsetRef.current =
            data.log_total || logOffsetRef.current + data.logs.length
        }
      } catch {
        // Ignore simple fetch errors during polling
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

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: "smooth" })
    }
  }, [logs])

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.logs")} />

      <div className="flex flex-1 flex-col overflow-hidden p-4 sm:p-8">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">
              {t("navigation.logs")}
            </h1>
            <p className="text-muted-foreground mt-2 text-sm">
              {t("pages.logs.description")}
            </p>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleClearLogs}
            disabled={logs.length === 0 || clearing}
          >
            <IconTrash className="size-4" />
            {t("pages.logs.clear")}
          </Button>
        </div>

        <div className="bg-muted/30 relative flex-1 overflow-hidden rounded-lg border">
          <ScrollArea className="h-full">
            <div className="p-4 font-mono text-sm leading-relaxed">
              {logs.length === 0 ? (
                <div className="text-muted-foreground italic">
                  {t("pages.logs.empty")}
                </div>
              ) : (
                logs.map((log, i) => (
                  <div key={i} className="break-all whitespace-pre-wrap">
                    {log}
                  </div>
                ))
              )}
              <div ref={scrollRef} />
            </div>
          </ScrollArea>
        </div>
      </div>
    </div>
  )
}
