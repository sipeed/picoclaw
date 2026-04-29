import { IconLoader2 } from "@tabler/icons-react"
import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ChannelConfig,
  type SupportedChannel,
  getChannelConfig,
  getChannelsCatalog,
  patchAppConfig,
} from "@/api/channels"
import { type ArrayFieldFlusher } from "@/components/channels/channel-array-list-field"
import {
  normalizeAllowFromValues,
  serializeStringArrayForSubmit,
} from "@/components/channels/channel-array-utils"
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
import { MqttForm } from "@/components/channels/channel-forms/mqtt-form"
import { SlackForm } from "@/components/channels/channel-forms/slack-form"
import { TelegramForm } from "@/components/channels/channel-forms/telegram-form"
import { WecomForm } from "@/components/channels/channel-forms/wecom-form"
import { WeixinForm } from "@/components/channels/channel-forms/weixin-form"
import { ConfigChangeNotice } from "@/components/config-change-notice"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { useGateway } from "@/hooks/use-gateway"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
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

function setRecordValueByPath(
  source: Record<string, unknown>,
  pathSegments: string[],
  value: unknown,
): Record<string, unknown> {
  const [segment, ...rest] = pathSegments
  if (!segment) {
    return source
  }
  if (rest.length === 0) {
    return { ...source, [segment]: value }
  }
  return {
    ...source,
    [segment]: setRecordValueByPath(asRecord(source[segment]), rest, value),
  }
}

function setConfigValueByPath(
  source: ChannelConfig,
  fieldPath: string,
  value: unknown,
): ChannelConfig {
  return setRecordValueByPath(source, fieldPath.split("."), value)
}

function serializeGroupTriggerForSubmit(value: unknown): unknown {
  const groupTrigger = asRecord(value)
  if (Object.keys(groupTrigger).length === 0) {
    return value
  }
  return {
    ...groupTrigger,
    prefixes: serializeStringArrayForSubmit(groupTrigger.prefixes),
  }
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
  const payload: ChannelConfig = { enabled, type: channel.config_key }
  const settings: ChannelConfig = {}

  for (const [key, value] of Object.entries(editConfig)) {
    if (key.startsWith("_")) continue
    if (key === "enabled") continue
    if (CHANNEL_COMMON_CONFIG_KEYS.has(key)) {
      if (key === "allow_from") {
        payload[key] = serializeStringArrayForSubmit(
          normalizeAllowFromValues(value),
        )
      } else if (key === "group_trigger") {
        payload[key] = serializeGroupTriggerForSubmit(value)
      } else {
        payload[key] = value
      }
      continue
    }
    if (isSecretField(key)) continue

    settings[key] = serializeStringArrayForSubmit(value)
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

  if (channel.name === "whatsapp_native") {
    settings.use_native = true
  }
  if (channel.name === "whatsapp") {
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
  const hasValue = (key: string) =>
    !isMissingRequiredValue(
      getFieldValueForValidation(config, configuredSecrets, key),
    )

  switch (channel.name) {
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
    case "mqtt":
      return hasValue("broker") && hasValue("agent_id")
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
    case "mqtt":
      return ["broker", "agent_id"]
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
  "mqtt",
])

export function ChannelConfigPage({ channelName }: ChannelConfigPageProps) {
  const { t, i18n } = useTranslation()
  const { state: gatewayState } = useGateway()

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [fetchError, setFetchError] = useState("")
  const [serverError, setServerError] = useState("")
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const [channel, setChannel] = useState<SupportedChannel | null>(null)
  const [baseConfig, setBaseConfig] = useState<ChannelConfig>({})
  const [editConfig, setEditConfig] = useState<ChannelConfig>({})
  const [configuredSecrets, setConfiguredSecrets] = useState<string[]>([])
  const [enabled, setEnabled] = useState(false)
  const [arrayFieldResetVersion, setArrayFieldResetVersion] = useState(0)
  const arrayFieldFlushersRef = useRef(new Map<string, ArrayFieldFlusher>())
  const loadRequestIdRef = useRef(0)

  const resetPageState = useCallback(() => {
    arrayFieldFlushersRef.current.clear()
    setChannel(null)
    setBaseConfig({})
    setEditConfig({})
    setConfiguredSecrets([])
    setEnabled(false)
    setFetchError("")
    setServerError("")
    setFieldErrors({})
    setArrayFieldResetVersion((version) => version + 1)
  }, [])

  const loadData = useCallback(
    async (silent = false) => {
      const requestId = loadRequestIdRef.current + 1
      loadRequestIdRef.current = requestId
      if (!silent) setLoading(true)
      try {
        const catalog = await getChannelsCatalog()
        if (loadRequestIdRef.current !== requestId) return
        const matched =
          catalog.channels.find((item) => item.name === channelName) ?? null

        if (!matched) {
          resetPageState()
          setFetchError(
            t("channels.page.notFound", {
              name: channelName,
            }),
          )
          return
        }

        const channelConfig = await getChannelConfig(channelName)
        if (loadRequestIdRef.current !== requestId) return
        const raw = asRecord(channelConfig.config)
        const normalized = normalizeConfig(matched, raw)

        setChannel(matched)
        setBaseConfig(normalized)
        setEditConfig(buildEditConfig(matched.name, normalized))
        setConfiguredSecrets(channelConfig.configured_secrets ?? [])
        setEnabled(asBool(normalized.enabled))
        setFetchError("")
        setServerError("")
        setFieldErrors({})
      } catch (e) {
        if (loadRequestIdRef.current !== requestId) return
        setConfiguredSecrets([])
        setFetchError(e instanceof Error ? e.message : t("channels.loadError"))
      } finally {
        if (!silent && loadRequestIdRef.current === requestId) {
          setLoading(false)
        }
      }
    },
    [channelName, resetPageState, t],
  )

  useEffect(() => {
    resetPageState()
    setLoading(true)
    loadData()
  }, [loadData, resetPageState])

  const previousGatewayStatusRef = useRef(gatewayState)
  useEffect(() => {
    const previousStatus = previousGatewayStatusRef.current
    if (previousStatus !== "running" && gatewayState === "running") {
      void loadData()
    }
    previousGatewayStatusRef.current = gatewayState
  }, [gatewayState, loadData])

  const configured = useMemo(() => {
    if (!channel) return false
    return isConfigured(channel, editConfig, configuredSecrets)
  }, [channel, configuredSecrets, editConfig])

  const isDirty = useMemo(() => {
    if (loading || !channel || channel.name !== channelName) return false
    const basePayload = buildSavePayload(
      channel,
      buildEditConfig(channel.name, baseConfig),
      asBool(baseConfig.enabled),
    )
    const currentPayload = buildSavePayload(channel, editConfig, enabled)
    return JSON.stringify(basePayload) !== JSON.stringify(currentPayload)
  }, [baseConfig, channel, channelName, editConfig, enabled, loading])

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

  const hidesPageLevelEnableToggle = channel?.name === "wecom"

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

  const registerArrayFieldFlusher = useCallback(
    (fieldPath: string, flusher: ArrayFieldFlusher | null) => {
      if (flusher) {
        arrayFieldFlushersRef.current.set(fieldPath, flusher)
        return
      }
      arrayFieldFlushersRef.current.delete(fieldPath)
    },
    [],
  )

  const flushPendingArrayFieldDrafts = useCallback(
    (sourceConfig: ChannelConfig): ChannelConfig => {
      let nextConfig = sourceConfig
      for (const [fieldPath, flusher] of arrayFieldFlushersRef.current) {
        const flushedValue = flusher()
        if (flushedValue === null) {
          continue
        }
        nextConfig = setConfigValueByPath(nextConfig, fieldPath, flushedValue)
      }
      return nextConfig
    },
    [],
  )

  const handleReset = () => {
    if (!channel) return
    setEditConfig(buildEditConfig(channel.name, baseConfig))
    setEnabled(asBool(baseConfig.enabled))
    setServerError("")
    setFieldErrors({})
    setArrayFieldResetVersion((version) => version + 1)
  }

  const handleSave = async () => {
    if (!channel) return

    const preparedEditConfig = flushPendingArrayFieldDrafts(editConfig)
    if (preparedEditConfig !== editConfig) {
      setEditConfig(preparedEditConfig)
    }

    const missingRequiredFields = requiredKeys.filter((key) =>
      isMissingRequiredValue(
        getFieldValueForValidation(preparedEditConfig, configuredSecrets, key),
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
      const savePayload = buildSavePayload(channel, preparedEditConfig, enabled)
      await patchAppConfig({
        channel_list: {
          [channel.config_key]: savePayload,
        },
      })
      await loadData()
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("channels.page.saveSuccess"),
        channelDisplayName,
        gateway?.restartRequired === true,
      )
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

  const renderForm = () => {
    if (!channel) return null
    const isEdit = configured

    switch (channel.name) {
      case "telegram":
        return (
          <TelegramForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
            registerArrayFieldFlusher={registerArrayFieldFlusher}
            arrayFieldResetVersion={arrayFieldResetVersion}
          />
        )
      case "discord":
        return (
          <DiscordForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
            registerArrayFieldFlusher={registerArrayFieldFlusher}
            arrayFieldResetVersion={arrayFieldResetVersion}
          />
        )
      case "slack":
        return (
          <SlackForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
            registerArrayFieldFlusher={registerArrayFieldFlusher}
            arrayFieldResetVersion={arrayFieldResetVersion}
          />
        )
      case "feishu":
        return (
          <FeishuForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
            registerArrayFieldFlusher={registerArrayFieldFlusher}
            arrayFieldResetVersion={arrayFieldResetVersion}
          />
        )
      case "mqtt":
        return (
          <MqttForm
            config={editConfig}
            onChange={handleChange}
            configuredSecrets={configuredSecrets}
            fieldErrors={fieldErrors}
          />
        )
      case "weixin":
        return (
          <WeixinForm
            config={editConfig}
            onChange={handleChange}
            isEdit={isEdit}
            onBindSuccess={() => void handleWeixinBindSuccess()}
            registerArrayFieldFlusher={registerArrayFieldFlusher}
            arrayFieldResetVersion={arrayFieldResetVersion}
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
              registerArrayFieldFlusher={registerArrayFieldFlusher}
              arrayFieldResetVersion={arrayFieldResetVersion}
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
            registerArrayFieldFlusher={registerArrayFieldFlusher}
            arrayFieldResetVersion={arrayFieldResetVersion}
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

            {renderForm()}

            {serverError && (
              <p className="text-destructive text-sm">{serverError}</p>
            )}

            {isDirty && (
              <ConfigChangeNotice
                kind="save"
                title={t("common.saveChangesTitle")}
                description={t("channels.page.savePrompt")}
              />
            )}

            <div className="border-border/60 flex justify-end gap-2 border-t py-4">
              <Button
                variant="outline"
                onClick={handleReset}
                disabled={!isDirty || saving}
              >
                {t("common.reset")}
              </Button>
              <Button onClick={handleSave} disabled={!isDirty || saving}>
                {saving ? t("common.saving") : t("common.save")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
