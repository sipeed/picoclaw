import { IconLoader2 } from "@tabler/icons-react"
import { useAtomValue } from "jotai"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ChannelConfig,
  type SupportedChannel,
  getAppConfig,
  getChannelsCatalog,
  patchAppConfig,
} from "@/api/channels"
import { getChannelDisplayName } from "@/components/channels/channel-display-name"
import { DiscordForm } from "@/components/channels/channel-forms/discord-form"
import { FeishuForm } from "@/components/channels/channel-forms/feishu-form"
import { GenericForm } from "@/components/channels/channel-forms/generic-form"
import { SlackForm } from "@/components/channels/channel-forms/slack-form"
import { TelegramForm } from "@/components/channels/channel-forms/telegram-form"
import { WeixinForm } from "@/components/channels/channel-forms/weixin-form"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { gatewayAtom } from "@/store/gateway"

interface ChannelConfigPageProps {
  channelName: string
}

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
  password: "_password",
  nickserv_password: "_nickserv_password",
  sasl_password: "_sasl_password",
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asBool(value: unknown): boolean {
  return value === true
}

function buildEditConfig(config: ChannelConfig): ChannelConfig {
  const edit: ChannelConfig = { ...config }
  for (const secretKey of Object.keys(SECRET_FIELD_MAP)) {
    if (secretKey in config) {
      edit[SECRET_FIELD_MAP[secretKey]] = ""
    }
  }
  return edit
}

function normalizeConfig(
  channel: SupportedChannel,
  rawConfig: ChannelConfig,
): ChannelConfig {
  const config = { ...rawConfig }
  if (channel.name === "whatsapp_native") {
    config.use_native = true
  }
  if (channel.name === "whatsapp") {
    config.use_native = false
  }
  return config
}

function buildSavePayload(
  channel: SupportedChannel,
  editConfig: ChannelConfig,
  enabled: boolean,
): ChannelConfig {
  const payload: ChannelConfig = { enabled }

  for (const [key, value] of Object.entries(editConfig)) {
    if (key.startsWith("_")) continue
    if (key === "enabled") continue

    if (key in SECRET_FIELD_MAP) {
      const editKey = SECRET_FIELD_MAP[key]
      const incoming = asString(editConfig[editKey])
      payload[key] = incoming !== "" ? incoming : value
      continue
    }

    payload[key] = value
  }

  if (channel.name === "whatsapp_native") {
    payload.use_native = true
  }
  if (channel.name === "whatsapp") {
    payload.use_native = false
  }

  return payload
}

function isConfigured(
  channel: SupportedChannel,
  config: ChannelConfig,
): boolean {
  switch (channel.name) {
    case "telegram":
      return asString(config.token) !== ""
    case "discord":
      return asString(config.token) !== ""
    case "slack":
      return asString(config.bot_token) !== ""
    case "feishu":
      return (
        asString(config.app_id) !== "" && asString(config.app_secret) !== ""
      )
    case "dingtalk":
      return (
        asString(config.client_id) !== "" &&
        asString(config.client_secret) !== ""
      )
    case "line":
      return asString(config.channel_access_token) !== ""
    case "qq":
      return (
        asString(config.app_id) !== "" && asString(config.app_secret) !== ""
      )
    case "onebot":
      return asString(config.ws_url) !== ""
    case "weixin":
      return asString(config.account_id) !== ""
    case "wecom":
      return asString(config.token) !== ""
    case "wecom_app":
      return (
        asString(config.corp_id) !== "" && asString(config.corp_secret) !== ""
      )
    case "wecom_aibot":
      return asString(config.token) !== ""
    case "whatsapp":
      return asString(config.bridge_url) !== ""
    case "whatsapp_native":
      return asBool(config.use_native)
    case "pico":
      return asString(config.token) !== ""
    case "maixcam":
      return asString(config.host) !== ""
    case "matrix":
      return (
        asString(config.homeserver) !== "" &&
        asString(config.user_id) !== "" &&
        asString(config.access_token) !== ""
      )
    case "irc":
      return asString(config.server) !== ""
    default:
      return false
  }
}

function getRequiredFieldKeys(channelName: string): string[] {
  switch (channelName) {
    case "telegram":
      return ["token"]
    case "discord":
      return ["token"]
    case "slack":
      return ["bot_token"]
    case "feishu":
      return ["app_id", "app_secret"]
    case "dingtalk":
      return ["client_id", "client_secret"]
    case "line":
      return ["channel_secret", "channel_access_token"]
    case "qq":
      return ["app_id", "app_secret"]
    case "onebot":
      return ["ws_url"]
    case "wecom":
      return ["token"]
    case "wecom_app":
      return ["corp_id", "corp_secret"]
    case "wecom_aibot":
      return ["token"]
    case "whatsapp":
      return ["bridge_url"]
    case "pico":
      return ["token"]
    case "maixcam":
      return ["host"]
    case "matrix":
      return ["homeserver", "user_id", "access_token"]
    case "irc":
      return ["server"]
    default:
      return []
  }
}

function isMissingRequiredValue(value: unknown): boolean {
  if (value === null || value === undefined) {
    return true
  }
  if (typeof value === "string") {
    return value.trim() === ""
  }
  if (Array.isArray(value)) {
    return value.length === 0
  }
  return false
}

function getChannelDocSlug(channelName: string): string {
  return channelName.replaceAll("_", "-")
}

const CHANNELS_WITHOUT_DOCS = new Set([
  "pico",
  "wecom",
  "matrix",
  "irc",
  "whatsapp",
  "whatsapp_native",
])

export function ChannelConfigPage({ channelName }: ChannelConfigPageProps) {
  const { t, i18n } = useTranslation()
  const gateway = useAtomValue(gatewayAtom)

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [fetchError, setFetchError] = useState("")
  const [serverError, setServerError] = useState("")
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const [channel, setChannel] = useState<SupportedChannel | null>(null)
  const [baseConfig, setBaseConfig] = useState<ChannelConfig>({})
  const [editConfig, setEditConfig] = useState<ChannelConfig>({})
  const [enabled, setEnabled] = useState(false)

  const loadData = useCallback(async (silent = false) => {
    if (!silent) setLoading(true)
    try {
      const [catalog, appConfig] = await Promise.all([
        getChannelsCatalog(),
        getAppConfig(),
      ])
      const matched =
        catalog.channels.find((item) => item.name === channelName) ?? null

      if (!matched) {
        setChannel(null)
        setFetchError(
          t("channels.page.notFound", {
            name: channelName,
          }),
        )
        return
      }

      const channelsConfig = asRecord(asRecord(appConfig).channels)
      const raw = asRecord(channelsConfig[matched.config_key])
      const normalized = normalizeConfig(matched, raw)

      setChannel(matched)
      setBaseConfig(normalized)
      setEditConfig(buildEditConfig(normalized))
      setEnabled(asBool(normalized.enabled))
      setFetchError("")
      setServerError("")
      setFieldErrors({})
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : t("channels.loadError"))
    } finally {
      if (!silent) setLoading(false)
    }
  }, [channelName, t])

  useEffect(() => {
    loadData()
  }, [loadData])

  const previousGatewayStatusRef = useRef(gateway.status)
  useEffect(() => {
    const previousStatus = previousGatewayStatusRef.current
    if (previousStatus !== "running" && gateway.status === "running") {
      void loadData()
    }
    previousGatewayStatusRef.current = gateway.status
  }, [gateway.status, loadData])

  const savePayload = useMemo(() => {
    if (!channel) return null
    return buildSavePayload(channel, editConfig, enabled)
  }, [channel, editConfig, enabled])

  const configured = useMemo(() => {
    if (!channel || !savePayload) return false
    return isConfigured(channel, savePayload)
  }, [channel, savePayload])

  const docsUrl = useMemo(() => {
    if (!channel) return ""
    if (CHANNELS_WITHOUT_DOCS.has(channel.name)) return ""
    const language = (
      i18n.resolvedLanguage ??
      i18n.language ??
      ""
    ).toLowerCase()
    const base = language.startsWith("zh")
      ? "https://docs.picoclaw.io/zh-Hans/docs/channels"
      : "https://docs.picoclaw.io/docs/channels"
    return `${base}/${getChannelDocSlug(channel.name)}`
  }, [channel, i18n.language, i18n.resolvedLanguage])

  const channelDisplayName = useMemo(() => {
    if (!channel) return channelName
    return getChannelDisplayName(channel, t)
  }, [channel, channelName, t])

  const hiddenKeys = useMemo(() => {
    if (!channel) return []
    if (channel.name === "whatsapp") {
      return ["use_native"]
    }
    if (channel.name === "whatsapp_native") {
      return ["use_native", "bridge_url"]
    }
    return []
  }, [channel])
  const requiredKeys = useMemo(
    () => getRequiredFieldKeys(channelName),
    [channelName],
  )

  const handleChange = useCallback((key: string, value: unknown) => {
    const normalizedKey = key.startsWith("_") ? key.slice(1) : key
    setEditConfig((prev) => ({ ...prev, [key]: value }))
    setFieldErrors((prev) => {
      if (!(key in prev) && !(normalizedKey in prev)) {
        return prev
      }
      const next = { ...prev }
      delete next[key]
      delete next[normalizedKey]
      return next
    })
  }, [])

  const handleReset = () => {
    setEditConfig(buildEditConfig(baseConfig))
    setEnabled(asBool(baseConfig.enabled))
    setServerError("")
    setFieldErrors({})
  }

  const handleSave = async () => {
    if (!channel || !savePayload) return

    const missingRequiredFields = requiredKeys.filter((key) =>
      isMissingRequiredValue(savePayload[key]),
    )
    if (missingRequiredFields.length > 0) {
      const requiredFieldError = t("channels.validation.requiredField")
      const nextFieldErrors: Record<string, string> = {}
      for (const key of missingRequiredFields) {
        nextFieldErrors[key] = requiredFieldError
      }
      setFieldErrors(nextFieldErrors)
      setServerError("")
      return
    }

    setSaving(true)
    setServerError("")
    setFieldErrors({})
    try {
      await patchAppConfig({
        channels: {
          [channel.config_key]: savePayload,
        },
      })
      toast.success(t("channels.page.saveSuccess"))
      await loadData()
    } catch (e) {
      const message =
        e instanceof Error ? e.message : t("channels.page.saveError")
      setServerError(message)
      toast.error(message)
    } finally {
      setSaving(false)
    }
  }

  const renderForm = () => {
    if (!channel) return null
    const isEdit = configured

    switch (channel.name) {
      case "telegram":
        return (
          <TelegramForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            fieldErrors={fieldErrors}
          />
        )
      case "discord":
        return (
          <DiscordForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            fieldErrors={fieldErrors}
          />
        )
      case "slack":
        return (
          <SlackForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            fieldErrors={fieldErrors}
          />
        )
      case "feishu":
        return (
          <FeishuForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            fieldErrors={fieldErrors}
          />
        )
      case "weixin":
        return (
          <WeixinForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            onBindSuccess={() => void loadData(true)}
          />
        )
      default:
        return (
          <GenericForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            hiddenKeys={hiddenKeys}
            requiredKeys={requiredKeys}
            fieldErrors={fieldErrors}
          />
        )
    }
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={channelDisplayName}
        titleExtra={
          channel ? (
            <div className="flex items-center gap-1.5">
              {enabled ? (
                <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
                  {t("channels.page.enabled")}
                </span>
              ) : configured ? (
                <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium text-amber-600 dark:text-amber-400">
                  {t("channels.status.configured")}
                </span>
              ) : null}
            </div>
          ) : undefined
        }
      />

      <div className="flex min-h-0 flex-1 justify-center overflow-y-auto px-4 pb-8 sm:px-6">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <IconLoader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        ) : fetchError ? (
          <div className="text-destructive bg-destructive/10 rounded-lg px-4 py-3 text-sm">
            {fetchError}
          </div>
        ) : (
          <div className="w-full max-w-250 space-y-5 pt-2">
            <div className="flex items-center gap-2 text-sm">
              <p className="font-medium">
                {t("channels.edit", {
                  name: channelDisplayName,
                })}
              </p>
              {channel && docsUrl && (
                <a
                  href={docsUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="text-muted-foreground hover:text-foreground text-xs underline underline-offset-2"
                >
                  {t("channels.page.docLink")}
                </a>
              )}
            </div>

            <div className="border-border/60 bg-background flex items-center justify-between rounded-lg border px-4 py-3">
              <p className="text-sm font-medium">
                {t("channels.page.enableLabel")}
              </p>
              <Switch checked={enabled} onCheckedChange={setEnabled} />
            </div>

            {renderForm()}

            {serverError && (
              <p className="text-destructive text-sm">{serverError}</p>
            )}

            <div className="border-border/60 flex justify-end gap-2 border-t py-4">
              <Button variant="outline" onClick={handleReset} disabled={saving}>
                {t("common.reset")}
              </Button>
              <Button onClick={handleSave} disabled={saving}>
                {saving ? t("common.saving") : t("common.save")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
