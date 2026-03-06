import { IconSettings } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ChannelInfo } from "@/api/channels"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"

interface ChannelCardProps {
  channel: ChannelInfo
  onToggle: (name: string, enabled: boolean) => void
  onEdit: (channel: ChannelInfo) => void
  toggling: boolean
}

export function ChannelCard({
  channel,
  onToggle,
  onEdit,
  toggling,
}: ChannelCardProps) {
  const { t } = useTranslation()

  return (
    <div className="border-border/60 bg-card flex items-center justify-between rounded-xl border p-4 transition-colors">
      <div className="flex items-center gap-3">
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold">
              {channel.display_name}
            </span>
            {channel.configured ? (
              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
                {t("channels.status.configured")}
              </span>
            ) : (
              <span className="text-muted-foreground bg-muted rounded-full px-2 py-0.5 text-[10px] font-medium">
                {t("channels.status.unconfigured")}
              </span>
            )}
          </div>
          <span className="text-muted-foreground text-xs">{channel.name}</span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Switch
          checked={channel.enabled}
          onCheckedChange={(checked) => onToggle(channel.name, checked)}
          disabled={toggling}
        />
        <Button
          size="sm"
          variant="ghost"
          className="size-8 p-0"
          onClick={() => onEdit(channel)}
        >
          <IconSettings className="size-4" />
          <span className="sr-only">{t("channels.action.configure")}</span>
        </Button>
      </div>
    </div>
  )
}
