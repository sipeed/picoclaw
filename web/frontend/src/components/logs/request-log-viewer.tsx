import { IconDownload, IconFilter, IconRefresh } from "@tabler/icons-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  getRequestLogs,
  getExportLogsUrl,
  type RequestRecord,
} from "@/api/stats"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { ScrollArea } from "@/components/ui/scroll-area"

const CHANNELS = ["", "telegram", "discord", "slack", "feishu", "dingtalk", "irc"]

export function RequestLogViewer() {
  const { t } = useTranslation()
  const [records, setRecords] = useState<RequestRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [channel, setChannel] = useState<string>("")
  const [offset, setOffset] = useState(0)
  const [limit] = useState(50)

  useEffect(() => {
    loadLogs()
  }, [channel, offset])

  async function loadLogs() {
    setLoading(true)
    setError(null)
    try {
      const response = await getRequestLogs({
        channel: channel || undefined,
        limit,
        offset,
      })
      setRecords(response.records)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load logs")
    } finally {
      setLoading(false)
    }
  }

  function handleRefresh() {
    loadLogs()
  }

  function handleExport(format: "json" | "csv") {
    const url = getExportLogsUrl({
      channel: channel || undefined,
      format,
    })
    window.open(url, "_blank")
  }

  function formatTimestamp(ts: string) {
    return new Date(ts).toLocaleString()
  }

  function truncateContent(content: string, maxLength = 100) {
    if (content.length <= maxLength) return content
    return content.slice(0, maxLength) + "..."
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("pages.logs.requests_title")} />

      <div className="flex flex-1 flex-col overflow-hidden p-4 sm:p-8">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <IconFilter className="size-4" />
              <Select value={channel} onValueChange={setChannel}>
                <SelectTrigger className="w-40">
                  <SelectValue placeholder={t("pages.logs.all_channels")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">{t("pages.logs.all_channels")}</SelectItem>
                  {CHANNELS.slice(1).map((ch) => (
                    <SelectItem key={ch} value={ch}>
                      {ch}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <Button variant="outline" size="sm" onClick={handleRefresh}>
              <IconRefresh className="size-4" />
              {t("common.refresh", "Refresh")}
            </Button>
          </div>

          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => handleExport("json")}>
              <IconDownload className="size-4" />
              JSON
            </Button>
            <Button variant="outline" size="sm" onClick={() => handleExport("csv")}>
              <IconDownload className="size-4" />
              CSV
            </Button>
          </div>
        </div>

        {error && (
          <div className="mb-4 rounded-md bg-red-500/10 p-4 text-red-500">
            {error}
          </div>
        )}

        <div className="flex-1 overflow-hidden rounded-lg border">
          <ScrollArea className="h-full">
            {loading ? (
              <div className="flex h-64 items-center justify-center text-muted-foreground">
                {t("labels.loading")}
              </div>
            ) : records.length === 0 ? (
              <div className="flex h-64 items-center justify-center text-muted-foreground">
                {t("pages.logs.no_logs")}
              </div>
            ) : (
              <div className="divide-y">
                <div className="grid grid-cols-[180px_100px_150px_1fr_100px] gap-4 bg-muted/50 p-3 text-sm font-medium">
                  <div>{t("pages.logs.timestamp")}</div>
                  <div>{t("pages.logs.channel")}</div>
                  <div>{t("pages.logs.sender")}</div>
                  <div>{t("pages.logs.content")}</div>
                  <div className="text-right">{t("pages.logs.proc_time")}</div>
                </div>
                {records.map((record) => (
                  <div
                    key={record.request_id}
                    className="grid grid-cols-[180px_100px_150px_1fr_100px] gap-4 p-3 text-sm hover:bg-muted/30"
                  >
                    <div className="whitespace-nowrap text-muted-foreground">
                      {formatTimestamp(record.timestamp)}
                    </div>
                    <div className="font-medium">{record.channel}</div>
                    <div className="text-muted-foreground">
                      {record.sender_info.username || record.sender_id}
                    </div>
                    <div className="truncate text-muted-foreground">
                      {truncateContent(record.content)}
                    </div>
                    <div className="text-right text-muted-foreground">
                      {record.processing_time_ms}ms
                    </div>
                  </div>
                ))}
              </div>
            )}
          </ScrollArea>
        </div>

        {records.length > 0 && (
          <div className="mt-4 flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              {t("pages.logs.showing")} {offset + 1}-{offset + records.length}
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
              >
                {t("pages.logs.previous")}
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={records.length < limit}
                onClick={() => setOffset(offset + limit)}
              >
                {t("pages.logs.next")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
