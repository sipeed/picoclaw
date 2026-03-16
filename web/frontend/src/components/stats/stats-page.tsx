import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { PageHeader } from "@/components/page-header"
import { getRequestStats, type RequestStats } from "@/api/stats"

export function StatsPage() {
  const { t } = useTranslation()
  const [stats, setStats] = useState<RequestStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function loadStats() {
      try {
        const data = await getRequestStats()
        setStats(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load stats")
      } finally {
        setLoading(false)
      }
    }
    loadStats()
  }, [])

  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <PageHeader title={t("pages.stats.title")} />
        <div className="mt-6 text-muted-foreground">{t("pages.stats.loading")}</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container mx-auto p-6">
        <PageHeader title={t("pages.stats.title")} />
        <div className="mt-6 text-red-500">{t("pages.stats.error", { error })}</div>
      </div>
    )
  }

  const channelEntries = stats?.by_channel
    ? Object.entries(stats.by_channel).sort((a, b) => b[1] - a[1])
    : []

  const dayEntries = stats?.by_day
    ? Object.entries(stats.by_day).sort((a, b) => a[0].localeCompare(b[0]))
    : []

  return (
    <div className="container mx-auto p-6">
      <PageHeader title={t("pages.stats.title")} />

      <div className="mt-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">{t("pages.stats.total_requests")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.total ?? 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">{t("pages.stats.active_channels")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{channelEntries.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">{t("pages.stats.days_tracked")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{dayEntries.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">{t("pages.stats.top_senders")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.top_senders?.length ?? 0}</div>
          </CardContent>
        </Card>
      </div>

      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("pages.stats.requests_by_channel")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {channelEntries.map(([channel, count]) => (
                <div key={channel} className="flex items-center justify-between">
                  <span className="font-medium">{channel}</span>
                  <span className="text-muted-foreground">{count}</span>
                </div>
              ))}
              {channelEntries.length === 0 && (
                <div className="text-muted-foreground">{t("pages.stats.no_data")}</div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("pages.stats.top_senders")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {stats?.top_senders?.map((sender, idx) => (
                <div key={idx} className="flex items-center justify-between">
                  <div>
                    <span className="font-medium">{sender.sender}</span>
                    <span className="text-muted-foreground text-sm ml-2">
                      ({sender.channel})
                    </span>
                  </div>
                  <span className="text-muted-foreground">{sender.count}</span>
                </div>
              ))}
              {(!stats?.top_senders || stats.top_senders.length === 0) && (
                <div className="text-muted-foreground">{t("pages.stats.no_data")}</div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>{t("pages.stats.requests_by_day")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {dayEntries.slice(-14).map(([date, count]) => (
              <div key={date} className="flex items-center justify-between">
                <span className="font-medium">{date}</span>
                <div className="flex items-center gap-2">
                  <div
                    className="bg-primary h-2 rounded"
                    style={{
                      width: `${Math.min(100, (count / (stats?.total || 1)) * 100 * 10)}%`,
                    }}
                  />
                  <span className="text-muted-foreground w-16 text-right">{count}</span>
                </div>
              </div>
            ))}
            {dayEntries.length === 0 && (
              <div className="text-muted-foreground">{t("pages.stats.no_data")}</div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
