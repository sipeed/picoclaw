import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelConfig, ChannelInfo } from "@/api/channels"
import { updateChannel } from "@/api/channels"
import { DiscordForm } from "@/components/channels/channel-forms/discord-form"
import { FeishuForm } from "@/components/channels/channel-forms/feishu-form"
import { GenericForm } from "@/components/channels/channel-forms/generic-form"
import { SlackForm } from "@/components/channels/channel-forms/slack-form"
import { TelegramForm } from "@/components/channels/channel-forms/telegram-form"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"

interface EditChannelSheetProps {
  channel: ChannelInfo | null
  open: boolean
  onClose: () => void
  onSaved: () => void
}

// Map of secret config keys to their edit-buffer keys.
// When editing, we use _token etc. to avoid overwriting with masked values.
const SECRET_FIELD_MAP: Record<string, string> = {
  token: "_token",
  app_secret: "_app_secret",
  client_secret: "_client_secret",
  corp_secret: "_corp_secret",
  channel_secret: "_channel_secret",
  channel_access_token: "_channel_access_token",
  access_token: "_access_token",
  bot_token: "_bot_token",
  app_token: "_app_token",
  encoding_aes_key: "_encoding_aes_key",
  encrypt_key: "_encrypt_key",
  verification_token: "_verification_token",
}

function buildEditConfig(config: ChannelConfig): ChannelConfig {
  const edit: ChannelConfig = { ...config }
  // Initialize edit buffer keys for secrets as empty (user fills new values)
  for (const secretKey of Object.keys(SECRET_FIELD_MAP)) {
    if (secretKey in config) {
      edit[SECRET_FIELD_MAP[secretKey]] = ""
    }
  }
  return edit
}

function buildSavePayload(
  channel: ChannelInfo,
  editConfig: ChannelConfig,
): ChannelConfig {
  const payload: ChannelConfig = { enabled: channel.enabled }

  for (const [key, value] of Object.entries(editConfig)) {
    // Skip the edit-buffer underscore keys — we use them to populate real keys
    if (key.startsWith("_")) continue
    // For secret fields, use the edit buffer value (empty means preserve existing)
    if (key in SECRET_FIELD_MAP) {
      const editKey = SECRET_FIELD_MAP[key]
      payload[key] = editConfig[editKey] ?? ""
    } else {
      payload[key] = value
    }
  }

  return payload
}

export function EditChannelSheet({
  channel,
  open,
  onClose,
  onSaved,
}: EditChannelSheetProps) {
  const { t } = useTranslation()
  const [editConfig, setEditConfig] = useState<ChannelConfig>({})
  const [saving, setSaving] = useState(false)
  const [serverError, setServerError] = useState("")

  useEffect(() => {
    if (channel) {
      setEditConfig(buildEditConfig(channel.config))
      setServerError("")
    }
  }, [channel])

  const handleChange = useCallback((key: string, value: unknown) => {
    setEditConfig((prev) => ({ ...prev, [key]: value }))
  }, [])

  const handleSave = async () => {
    if (!channel) return
    setSaving(true)
    setServerError("")
    try {
      await updateChannel(channel.name, buildSavePayload(channel, editConfig))
      onSaved()
      onClose()
    } catch (e) {
      setServerError(
        e instanceof Error ? e.message : t("channels.edit.saveError"),
      )
    } finally {
      setSaving(false)
    }
  }

  const renderForm = () => {
    if (!channel) return null
    const isEdit = channel.configured

    switch (channel.name) {
      case "telegram":
        return (
          <TelegramForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
          />
        )
      case "discord":
        return (
          <DiscordForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
          />
        )
      case "slack":
        return (
          <SlackForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
          />
        )
      case "feishu":
        return (
          <FeishuForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
          />
        )
      default:
        return (
          <GenericForm
            channelName={channel.name}
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
          />
        )
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent side="right" className="flex flex-col sm:max-w-md">
        <SheetHeader>
          <SheetTitle>
            {t("channels.edit.title", {
              name: channel?.display_name ?? "",
            })}
          </SheetTitle>
          <SheetDescription>{t("channels.edit.description")}</SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-4 py-4">{renderForm()}</div>

        {serverError && (
          <p className="px-1 text-sm text-red-500">{serverError}</p>
        )}

        <SheetFooter className="gap-2 pt-2">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? t("channels.edit.saving") : t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
