import { IconSearch } from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelInfo } from "@/api/channels"
import { getChannels, toggleChannel } from "@/api/channels"
import { ChannelCard } from "@/components/channels/channel-card"
import { EditChannelSheet } from "@/components/channels/edit-channel-sheet"
import { PageHeader } from "@/components/page-header"
import { Input } from "@/components/ui/input"

export function ChannelsPage() {
  const { t } = useTranslation()
  const [channels, setChannels] = useState<ChannelInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState("")
  const [editingChannel, setEditingChannel] = useState<ChannelInfo | null>(null)
  const [togglingChannel, setTogglingChannel] = useState<string | null>(null)
  const [search, setSearch] = useState("")

  const fetchChannels = useCallback(async () => {
    try {
      const data = await getChannels()
      // Sort: enabled first, then configured, then alphabetical
      const sorted = [...data.channels].sort((a, b) => {
        if (a.enabled !== b.enabled) return a.enabled ? -1 : 1
        if (a.configured !== b.configured) return a.configured ? -1 : 1
        return a.display_name.localeCompare(b.display_name)
      })
      setChannels(sorted)
      setFetchError("")
    } catch (e) {
      setFetchError(
        e instanceof Error ? e.message : t("channels.loadError"),
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchChannels()
  }, [fetchChannels])

  const handleToggle = async (name: string, enabled: boolean) => {
    setTogglingChannel(name)
    try {
      await toggleChannel(name, enabled)
      await fetchChannels()
    } catch {
      // Refresh to show actual state
      await fetchChannels()
    } finally {
      setTogglingChannel(null)
    }
  }

  const filtered = search
    ? channels.filter(
        (ch) =>
          ch.display_name.toLowerCase().includes(search.toLowerCase()) ||
          ch.name.toLowerCase().includes(search.toLowerCase()),
      )
    : channels

  const enabledCount = channels.filter((ch) => ch.enabled).length

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={t("navigation.channels", "Channels")}
        titleExtra={
          !loading && channels.length > 0 ? (
            <span className="text-muted-foreground text-sm font-normal">
              {enabledCount} {t("channels.header.enabled")}
            </span>
          ) : undefined
        }
      />

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="border-primary size-6 animate-spin rounded-full border-2 border-t-transparent" />
          </div>
        ) : fetchError ? (
          <div className="flex flex-col items-center justify-center gap-2 py-20">
            <p className="text-destructive text-sm">{fetchError}</p>
          </div>
        ) : (
          <div className="pb-8">
            <p className="text-muted-foreground mb-4 text-sm">
              {t("channels.description")}
            </p>

            {channels.length > 6 && (
              <div className="relative mb-4">
                <IconSearch className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={t("channels.search")}
                  className="pl-9"
                />
              </div>
            )}

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {filtered.map((channel) => (
                <ChannelCard
                  key={channel.name}
                  channel={channel}
                  onToggle={handleToggle}
                  onEdit={setEditingChannel}
                  toggling={togglingChannel === channel.name}
                />
              ))}
            </div>

            {filtered.length === 0 && search && (
              <p className="text-muted-foreground py-10 text-center text-sm">
                {t("channels.noResults")}
              </p>
            )}
          </div>
        )}
      </div>

      <EditChannelSheet
        channel={editingChannel}
        open={editingChannel !== null}
        onClose={() => setEditingChannel(null)}
        onSaved={fetchChannels}
      />
    </div>
  )
}
