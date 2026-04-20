import { IconLoader2 } from "@tabler/icons-react"
import { useNavigate } from "@tanstack/react-router"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ChannelConfig,
  type SupportedChannel,
  getChannelConfig,
  getChannelsCatalog,
  patchAppConfig,
} from "@/api/channels"
import {
  SECRET_FIELD_MAP,
  buildEditConfig,
  getFieldValueForValidation,
  isSecretField,
} from "@/components/channels/channel-config-fields"
import { getChannelDisplayName } from "@/components/channels/channel-display-name"
import { DiscordForm } from "@/components/channels/channel-forms/discord-form"
import { FeishuForm } from "@/components/channels/channel-forms/feishu-form"
import { GenericForm } from "@/components/channels/channel-forms/generic-form"
import { SlackForm } from "@/components/channels/channel-forms/slack-form"
import { TelegramForm } from "@/components/channels/channel-forms/telegram-form"
import { WecomForm } from "@/components/channels/channel-forms/wecom-form"
import { WeixinForm } from "@/components/channels/channel-forms/weixin-form"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { useGateway } from "@/hooks/use-gateway"
import { refreshGatewayState } from "@/store/gateway"

interface ChannelConfigPageProps {
  channelName: string
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

const CHANNEL_COMMON_CONFIG_KEYS = new Set([
  "allow_from",
  "group_trigger",
  "placeholder",
  "reasoning_channel_id",
  "typing",
])

function normalizeConfig(
  channel: SupportedChannel,
  rawConfig: ChannelConfig,
): ChannelConfig {
  const config = { ...rawConfig }
  if (getChannelType(channel) === "whatsapp_native") {
    config.use_native = true
  }
  if (getChannelType(channel) === "whatsapp") {
    config.use_native = false
  }
  return config
}

function getChannelType(channel: SupportedChannel): string {
  return channel.type ?? channel.name
}

function buildSavePayload(
  channel: SupportedChannel,
  editConfig: ChannelConfig,
  enabled: boolean,
): ChannelConfig {
  const channelType = getChannelType(channel)
  const payload: ChannelConfig = { enabled, type: channelType }
  const settings: ChannelConfig = {}

  for (const [key, value] of Object.entries(editConfig)) {
    if (key.startsWith("_")) continue
    if (key === "enabled") continue
    if (CHANNEL_COMMON_CONFIG_KEYS.has(key)) {
      payload[key] = value
      continue
    }
    if (isSecretField(key)) continue

    settings[key] = value
  }

  for (const [secretKey, editKey] of Object.entries(SECRET_FIELD_MAP)) {
    const incoming = asString(editConfig[editKey])
    if (incoming !== "") {
      settings[secretKey] = incoming
      continue
    }
    const existing = asString(editConfig[secretKey]).trim()
    if (existing !== "") {
      settings[secretKey] = existing
    }
  }

  if (channelType === "whatsapp_native") {
    settings.use_native = true
  }
  if (channelType === "whatsapp") {
    settings.use_native = false
  }

  if (Object.keys(settings).length > 0) {
    payload.settings = settings
  }

  return payload
}

function isConfigured(
  channel: SupportedChannel,
  config: ChannelConfig,
  configuredSecrets: readonly string[],
): boolean {
  const channelType = getChannelType(channel)
  const hasValue = (key: string) =>
    !isMissingRequiredValue(
      getFieldValueForValidation(config, configuredSecrets, key),
    )

  switch (channelType) {
    case "telegram":
      return hasValue("token")
    case "discord":
      return hasValue("token")
    case "slack":
      return hasValue("bot_token")
    case "feishu":
      return hasValue("app_id") && hasValue("app_secret")
    case "dingtalk":
      return hasValue("client_id") && hasValue("client_secret")
    case "line":
      return hasValue("channel_secret") && hasValue("channel_access_token")
    case "qq":
      return hasValue("app_id") && hasValue("app_secret")
    case "onebot":
      return hasValue("ws_url")
    case "weixin":
      return hasValue("account_id")
    case "wecom":
      return hasValue("bot_id")
    case "whatsapp":
      return hasValue("bridge_url")
    case "whatsapp_native":
      return asBool(config.use_native)
    case "pico":
      return hasValue("token")
    case "maixcam":
      return hasValue("host")
    case "matrix":
      return (
        hasValue("homeserver") &&
        hasValue("user_id") &&
        hasValue("access_token")
      )
    case "irc":
      return hasValue("server")
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
      return []
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

function buildSuggestedWeixinChannelName(existingNames: readonly string[]): string {
  const used = new Set(existingNames)
  if (!used.has("weixin_2")) {
    return "weixin_2"
  }
  let index = 3
  for (;;) {
    const candidate = `weixin_${index}`
    if (!used.has(candidate)) {
      return candidate
    }
    index++
  }
}

function isValidChannelName(name: string): boolean {
  return /^[A-Za-z0-9_.-]+$/.test(name)
}

function notifyChannelsUpdated() {
  window.dispatchEvent(new CustomEvent("picoclaw:channels-updated"))
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
  const { state: gatewayState } = useGateway()
  const navigate = useNavigate()

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creatingChannel, setCreatingChannel] = useState(false)
  const [fetchError, setFetchError] = useState("")
  const [serverError, setServerError] = useState("")
  const [createChannelError, setCreateChannelError] = useState("")
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const [catalogChannels, setCatalogChannels] = useState<SupportedChannel[]>([])
  const [channel, setChannel] = useState<SupportedChannel | null>(null)
  const [baseConfig, setBaseConfig] = useState<ChannelConfig>({})
  const [editConfig, setEditConfig] = useState<ChannelConfig>({})
  const [configuredSecrets, setConfiguredSecrets] = useState<string[]>([])
  const [enabled, setEnabled] = useState(false)
  const [newWeixinChannelName, setNewWeixinChannelName] = useState("")

  const loadData = useCallback(
    async (silent = false) => {
      if (!silent) setLoading(true)
      try {
        const catalog = await getChannelsCatalog()
        setCatalogChannels(catalog.channels)
        const matched =
          catalog.channels.find((item) => item.name === channelName) ?? null

        if (!matched) {
          setChannel(null)
          setBaseConfig({})
          setEditConfig({})
          setConfiguredSecrets([])
          setEnabled(false)
          setFetchError(
            t("channels.page.notFound", {
              name: channelName,
            }),
          )
          return
        }

        const channelConfig = await getChannelConfig(channelName)
        const raw = asRecord(channelConfig.config)
        const normalized = normalizeConfig(matched, raw)
        const currentType = getChannelType(matched)

        setChannel(matched)
        setBaseConfig(normalized)
        setEditConfig(buildEditConfig(currentType, normalized))
        setConfiguredSecrets(channelConfig.configured_secrets ?? [])
        setEnabled(asBool(normalized.enabled))
        setFetchError("")
        setServerError("")
        setCreateChannelError("")
        setFieldErrors({})
        setNewWeixinChannelName((prev) => {
          const next = buildSuggestedWeixinChannelName(
            catalog.channels.map((item) => item.name),
          )
          return prev.trim() === "" ? next : prev
        })
      } catch (e) {
        setCatalogChannels([])
        setConfiguredSecrets([])
        setFetchError(e instanceof Error ? e.message : t("channels.loadError"))
      } finally {
        if (!silent) setLoading(false)
      }
    },
    [channelName, t],
  )

  useEffect(() => {
    loadData()
  }, [loadData])

  const previousGatewayStatusRef = useRef(gatewayState)
  useEffect(() => {
    const previousStatus = previousGatewayStatusRef.current
    if (previousStatus !== "running" && gatewayState === "running") {
      void loadData()
    }
    previousGatewayStatusRef.current = gatewayState
  }, [gatewayState, loadData])

  const savePayload = useMemo(() => {
    if (!channel) return null
    return buildSavePayload(channel, editConfig, enabled)
  }, [channel, editConfig, enabled])

  const configured = useMemo(() => {
    if (!channel) return false
    return isConfigured(channel, editConfig, configuredSecrets)
  }, [channel, configuredSecrets, editConfig])

  const docsUrl = useMemo(() => {
    if (!channel) return ""
    const channelType = getChannelType(channel)
    if (CHANNELS_WITHOUT_DOCS.has(channelType)) return ""
    const language = (
      i18n.resolvedLanguage ??
      i18n.language ??
      ""
    ).toLowerCase()
    const base = language.startsWith("zh")
      ? "https://docs.picoclaw.io/zh-Hans/docs/channels"
      : "https://docs.picoclaw.io/docs/channels"
    return `${base}/${getChannelDocSlug(channelType)}`
  }, [channel, i18n.language, i18n.resolvedLanguage])

  const channelDisplayName = useMemo(() => {
    if (!channel) return channelName
    return getChannelDisplayName(channel, t)
  }, [channel, channelName, t])

  const channelType = channel ? getChannelType(channel) : channelName
  const hidesPageLevelEnableToggle = channelType === "wecom"
  const supportsMultiple = channel?.supports_multiple === true

  const hiddenKeys = useMemo(() => {
    if (!channel) return []
    const currentType = getChannelType(channel)
    if (currentType === "whatsapp") {
      return ["use_native"]
    }
    if (currentType === "whatsapp_native") {
      return ["use_native", "bridge_url"]
    }
    return []
  }, [channel])
  const requiredKeys = useMemo(
    () => getRequiredFieldKeys(channelType),
    [channelType],
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
    if (!channel) return
    setEditConfig(buildEditConfig(getChannelType(channel), baseConfig))
    setEnabled(asBool(baseConfig.enabled))
    setServerError("")
    setFieldErrors({})
  }

  const handleSave = async () => {
    if (!channel || !savePayload) return

    const missingRequiredFields = requiredKeys.filter((key) =>
      isMissingRequiredValue(
        getFieldValueForValidation(editConfig, configuredSecrets, key),
      ),
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
        channel_list: {
          [channel.config_key]: savePayload,
        },
      })
      notifyChannelsUpdated()
      await loadData()
    } catch (e) {
      const message =
        e instanceof Error ? e.message : t("channels.page.saveError")
      setServerError(message)
    } finally {
      setSaving(false)
    }
  }

  const handleWeixinBindSuccess = useCallback(async () => {
    try {
      setEnabled(true)
      notifyChannelsUpdated()
      await Promise.all([loadData(true), refreshGatewayState({ force: true })])
    } catch (e) {
      const message =
        e instanceof Error ? e.message : t("channels.page.saveError")
      setServerError(message)
      await loadData(true)
    }
  }, [loadData, t])

  const handleWecomBindSuccess = useCallback(async () => {
    try {
      setEnabled(true)
      notifyChannelsUpdated()
      await Promise.all([loadData(true), refreshGatewayState({ force: true })])
    } catch (e) {
      const message =
        e instanceof Error ? e.message : t("channels.page.saveError")
      setServerError(message)
      await loadData(true)
    }
  }, [loadData, t])

  const handleWecomEnabledChange = useCallback(
    async (nextEnabled: boolean) => {
      try {
        setEnabled(nextEnabled)
        await Promise.all([
          loadData(true),
          refreshGatewayState({ force: true }),
        ])
      } catch (e) {
        const message =
          e instanceof Error ? e.message : t("channels.page.saveError")
        setServerError(message)
        await loadData(true)
      }
    },
    [loadData, t],
  )

  const handleCreateWeixinChannel = useCallback(async () => {
    const nextChannelName = newWeixinChannelName.trim()
    if (nextChannelName === "") {
      setCreateChannelError(t("channels.weixin.channelNameRequired"))
      return
    }
    if (!isValidChannelName(nextChannelName)) {
      setCreateChannelError(t("channels.weixin.invalidChannelName"))
      return
    }
    if (catalogChannels.some((item) => item.name === nextChannelName)) {
      setCreateChannelError(
        t("channels.weixin.duplicateChannel", { name: nextChannelName }),
      )
      return
    }

    setCreatingChannel(true)
    setCreateChannelError("")
    try {
      await patchAppConfig({
        channel_list: {
          [nextChannelName]: {
            enabled: false,
            type: "weixin",
          },
        },
      })
      notifyChannelsUpdated()
      await navigate({
        to: "/channels/$name",
        params: { name: nextChannelName },
      })
    } catch (e) {
      setCreateChannelError(
        e instanceof Error ? e.message : t("channels.page.saveError"),
      )
    } finally {
      setCreatingChannel(false)
    }
  }, [catalogChannels, navigate, newWeixinChannelName, t])

  const renderForm = () => {
    if (!channel) return null
    const isEdit = configured
    const currentType = getChannelType(channel)

    switch (currentType) {
      case "telegram":
        return (
          <TelegramForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
          />
        )
      case "discord":
        return (
          <DiscordForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
          />
        )
      case "slack":
        return (
          <SlackForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
          />
        )
      case "feishu":
        return (
          <FeishuForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
          />
        )
      case "weixin":
        return (
          <WeixinForm
            channel={channel}
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            onBindSuccess={() => void handleWeixinBindSuccess()}
          />
        )
      case "wecom":
        return (
          <>
            <WecomForm
              config={editConfig}
              isEdit={isEdit}
              onBindSuccess={() => void handleWecomBindSuccess()}
              onEnabledChange={(nextEnabled) =>
                void handleWecomEnabledChange(nextEnabled)
              }
            />
            <GenericForm
              config={editConfig}
              onChange={handleChange}
              configuredSecrets={configuredSecrets}
              hiddenKeys={[...hiddenKeys, "bot_id"]}
              requiredKeys={requiredKeys}
              fieldErrors={fieldErrors}
            />
          </>
        )
      default:
        return (
          <GenericForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
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
          channel &&
          docsUrl && (
            <a
              href={docsUrl}
              target="_blank"
              rel="noreferrer"
              className="text-muted-foreground hover:text-foreground text-xs underline underline-offset-2"
            >
              {t("channels.page.docLink")}
            </a>
          )
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
          <div className="w-full max-w-4xl space-y-6 pt-5">
            {!hidesPageLevelEnableToggle && (
              <div className="bg-card text-card-foreground border-border/60 flex items-center justify-between rounded-xl border px-6 py-4 shadow-sm">
                <p className="text-sm font-medium">
                  {t("channels.page.enableLabel")}
                </p>
                <Switch checked={enabled} onCheckedChange={setEnabled} />
              </div>
            )}

            {channelType === "weixin" && supportsMultiple && (
              <div className="bg-card text-card-foreground border-border/60 space-y-4 rounded-xl border px-6 py-5 shadow-sm">
                <div className="space-y-1">
                  <p className="text-sm font-medium">
                    {t("channels.weixin.addChannelTitle")}
                  </p>
                  <p className="text-muted-foreground text-sm">
                    {t("channels.weixin.addChannelDesc")}
                  </p>
                </div>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                  <div className="min-w-0 flex-1">
                    <label
                      htmlFor="new-weixin-channel-name"
                      className="mb-1 block text-sm font-medium"
                    >
                      {t("channels.weixin.channelName")}
                    </label>
                    <Input
                      id="new-weixin-channel-name"
                      value={newWeixinChannelName}
                      onChange={(e) => {
                        setNewWeixinChannelName(e.target.value)
                        setCreateChannelError("")
                      }}
                      placeholder="weixin_2"
                    />
                  </div>
                  <Button
                    type="button"
                    onClick={() => void handleCreateWeixinChannel()}
                    disabled={creatingChannel}
                  >
                    {creatingChannel
                      ? t("common.saving")
                      : t("channels.weixin.createChannel")}
                  </Button>
                </div>
                {createChannelError && (
                  <p className="text-destructive text-sm">{createChannelError}</p>
                )}
              </div>
            )}

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
