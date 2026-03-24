import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Input } from "@/components/ui/input"

interface GenericFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  isEdit: boolean
  hiddenKeys?: string[]
  requiredKeys?: string[]
  fieldErrors?: Record<string, string>
}

// Secret field names that should use masked input.
const SECRET_FIELDS = new Set([
  "token",
  "app_secret",
  "client_secret",
  "corp_secret",
  "channel_secret",
  "channel_access_token",
  "access_token",
  "bot_token",
  "app_token",
  "encoding_aes_key",
  "encrypt_key",
  "verification_token",
  "secret",
  "password",
  "nickserv_password",
  "sasl_password",
])

// Fields to skip in the generic form (handled by enabled toggle or internal).
const SKIP_FIELDS = new Set(["enabled", "reasoning_channel_id"])

// Fields that are objects/nested — show as JSON or skip.
const OBJECT_FIELDS = new Set([
  "group_trigger",
  "typing",
  "placeholder",
  "allow_token_query",
  "allow_from",
  "allow_origins",
  "groups",
])

function formatLabel(key: string): string {
  return key
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ")
}

function formatSentenceFieldName(key: string): string {
  const label = formatLabel(key)
  return label.charAt(0).toLowerCase() + label.slice(1)
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === "string")
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function asBool(value: unknown): boolean {
  return value === true
}

export function GenericForm({
  config,
  onChange,
  isEdit,
  hiddenKeys = [],
  requiredKeys = [],
  fieldErrors = {},
}: GenericFormProps) {
  const { t } = useTranslation()
  const hiddenFieldSet = new Set(hiddenKeys)
  const requiredFieldSet = new Set(requiredKeys)
  const groupTriggerConfig = asRecord(config.group_trigger)
  const typingConfig = asRecord(config.typing)
  const placeholderConfig = asRecord(config.placeholder)
  const placeholderEnabled = asBool(placeholderConfig.enabled)

  const fields = Object.keys(config).filter(
    (k) =>
      !k.startsWith("_") &&
      !SKIP_FIELDS.has(k) &&
      !OBJECT_FIELDS.has(k) &&
      !hiddenFieldSet.has(k),
  )

  const buildHint = (key: string): string => {
    const descriptions: Record<string, string> = {
      ws_url: t("channels.form.desc.wsUrl"),
      reconnect_interval: t("channels.form.desc.reconnectInterval"),
      bridge_url: t("channels.form.desc.bridgeUrl"),
      session_store_path: t("channels.form.desc.sessionStorePath"),
      use_native: t("channels.form.desc.useNative"),
      host: t("channels.form.desc.host"),
      port: t("channels.form.desc.port"),
      homeserver: t("channels.form.desc.homeserver"),
      user_id: t("channels.form.desc.userId"),
      device_id: t("channels.form.desc.deviceId"),
      join_on_invite: t("channels.form.desc.joinOnInvite"),
      app_id: t("channels.form.desc.appId"),
      client_id: t("channels.form.desc.clientId"),
      corp_id: t("channels.form.desc.corpId"),
      bot_id: t("channels.form.desc.appId"),
      websocket_url: t("channels.form.desc.wsUrl"),
      dm_policy: t("channels.form.desc.genericField", { field: "DM policy" }),
      group_policy: t("channels.form.desc.genericField", { field: "group policy" }),
      group_allow_from: t("channels.form.desc.allowFrom"),
      send_thinking_message: t("channels.form.desc.genericField", {
        field: "thinking message behavior",
      }),
      agent_id: t("channels.form.desc.agentId"),
      webhook_url: t("channels.form.desc.webhookUrl"),
      webhook_host: t("channels.form.desc.webhookHost"),
      webhook_port: t("channels.form.desc.webhookPort"),
      webhook_path: t("channels.form.desc.webhookPath"),
      reply_timeout: t("channels.form.desc.replyTimeout"),
      max_steps: t("channels.form.desc.maxSteps"),
      welcome_message: t("channels.form.desc.welcomeMessage"),
      allow_token_query: t("channels.form.desc.allowTokenQuery"),
      ping_interval: t("channels.form.desc.pingInterval"),
      read_timeout: t("channels.form.desc.readTimeout"),
      write_timeout: t("channels.form.desc.writeTimeout"),
      max_connections: t("channels.form.desc.maxConnections"),
      server: t("channels.form.desc.server"),
      tls: t("channels.form.desc.tls"),
      nick: t("channels.form.desc.nick"),
      user: t("channels.form.desc.user"),
      real_name: t("channels.form.desc.realName"),
      channels: t("channels.form.desc.channels"),
      request_caps: t("channels.form.desc.requestCaps"),
      max_base64_file_size_mib: t("channels.form.desc.maxBase64FileSizeMiB"),
    }
    return (
      descriptions[key] ??
      t("channels.form.desc.genericField", {
        field: formatSentenceFieldName(key),
      })
    )
  }

  return (
    <div className="space-y-5">
      {fields.map((key) => {
        const isRequired = requiredFieldSet.has(key)
        if (SECRET_FIELDS.has(key)) {
          const editKey = `_${key}`
          const extraHint =
            isEdit && config[key] ? ` ${t("channels.field.secretHintSet")}` : ""
          return (
            <Field
              key={key}
              label={formatLabel(key)}
              required={isRequired}
              hint={`${buildHint(key)}${extraHint}`}
              error={fieldErrors[key]}
            >
              <KeyInput
                value={asString(config[editKey])}
                onChange={(v) => onChange(editKey, v)}
                placeholder={maskedSecretPlaceholder(config[key])}
              />
            </Field>
          )
        }

        const value = config[key]
        if (typeof value === "boolean") {
          return (
            <SwitchCardField
              key={key}
              label={formatLabel(key)}
              hint={buildHint(key)}
              error={fieldErrors[key]}
              checked={value}
              onCheckedChange={(checked) => onChange(key, checked)}
              ariaLabel={formatLabel(key)}
            />
          )
        }

        if (Array.isArray(value)) {
          return (
            <Field
              key={key}
              label={formatLabel(key)}
              required={isRequired}
              hint={buildHint(key)}
              error={fieldErrors[key]}
            >
              <Input
                value={asStringArray(value).join(", ")}
                onChange={(e) =>
                  onChange(
                    key,
                    e.target.value
                      .split(",")
                      .map((s: string) => s.trim())
                      .filter(Boolean),
                  )
                }
              />
            </Field>
          )
        }

        return (
          <Field
            key={key}
            label={formatLabel(key)}
            required={isRequired}
            hint={buildHint(key)}
            error={fieldErrors[key]}
          >
            <Input
              value={String(value ?? "")}
              onChange={(e) => {
                // Attempt to preserve number types
                const v = e.target.value
                if (typeof config[key] === "number") {
                  onChange(key, v === "" ? 0 : Number(v))
                } else {
                  onChange(key, v)
                }
              }}
            />
          </Field>
        )
      })}

      {/* Allow From field */}
      {config.allow_from !== undefined && !hiddenFieldSet.has("allow_from") && (
        <Field
          label={t("channels.field.allowFrom")}
          hint={t("channels.form.desc.allowFrom")}
        >
          <Input
            value={asStringArray(config.allow_from).join(", ")}
            onChange={(e) =>
              onChange(
                "allow_from",
                e.target.value
                  .split(",")
                  .map((s: string) => s.trim())
                  .filter(Boolean),
              )
            }
            placeholder={t("channels.field.allowFromPlaceholder")}
          />
        </Field>
      )}

      {config.allow_origins !== undefined &&
        !hiddenFieldSet.has("allow_origins") && (
          <Field
            label={t("channels.field.allowOrigins")}
            hint={t("channels.form.desc.allowOrigins")}
          >
            <Input
              value={asStringArray(config.allow_origins).join(", ")}
              onChange={(e) =>
                onChange(
                  "allow_origins",
                  e.target.value
                    .split(",")
                    .map((s: string) => s.trim())
                    .filter(Boolean),
                )
              }
              placeholder={t("channels.field.allowOriginsPlaceholder")}
            />
          </Field>
        )}

      {config.allow_token_query !== undefined &&
        !hiddenFieldSet.has("allow_token_query") && (
          <SwitchCardField
            label={formatLabel("allow_token_query")}
            hint={buildHint("allow_token_query")}
            checked={asBool(config.allow_token_query)}
            onCheckedChange={(checked) =>
              onChange("allow_token_query", checked)
            }
            ariaLabel={formatLabel("allow_token_query")}
          />
        )}

      {config.group_trigger !== undefined &&
        !hiddenFieldSet.has("group_trigger") && (
          <>
            <SwitchCardField
              label={t("channels.field.groupTriggerMentionOnly")}
              hint={t("channels.form.desc.groupTriggerMentionOnly")}
              checked={asBool(groupTriggerConfig.mention_only)}
              onCheckedChange={(checked) =>
                onChange("group_trigger", {
                  ...groupTriggerConfig,
                  mention_only: checked,
                })
              }
              ariaLabel={t("channels.field.groupTriggerMentionOnly")}
            />
            <Field
              label={t("channels.field.groupTriggerPrefixes")}
              hint={t("channels.form.desc.groupTriggerPrefixes")}
            >
              <Input
                value={asStringArray(groupTriggerConfig.prefixes).join(", ")}
                onChange={(e) =>
                  onChange("group_trigger", {
                    ...groupTriggerConfig,
                    prefixes: e.target.value
                      .split(",")
                      .map((s: string) => s.trim())
                      .filter(Boolean),
                  })
                }
                placeholder={t("channels.field.groupTriggerPrefixes")}
              />
            </Field>
          </>
        )}

      {config.typing !== undefined && !hiddenFieldSet.has("typing") && (
        <SwitchCardField
          label={t("channels.field.typingEnabled")}
          hint={t("channels.form.desc.typingEnabled")}
          checked={asBool(typingConfig.enabled)}
          onCheckedChange={(checked) =>
            onChange("typing", { ...typingConfig, enabled: checked })
          }
          ariaLabel={t("channels.field.typingEnabled")}
        />
      )}

      {config.placeholder !== undefined &&
        !hiddenFieldSet.has("placeholder") && (
          <SwitchCardField
            label={t("channels.field.placeholderEnabled")}
            hint={t("channels.form.desc.placeholderEnabled")}
            checked={placeholderEnabled}
            onCheckedChange={(checked) =>
              onChange("placeholder", {
                ...placeholderConfig,
                enabled: checked,
              })
            }
            ariaLabel={t("channels.field.placeholderEnabled")}
          >
            {placeholderEnabled && (
              <div className="space-y-1">
                <Input
                  value={asString(placeholderConfig.text)}
                  onChange={(e) =>
                    onChange("placeholder", {
                      ...placeholderConfig,
                      text: e.target.value,
                    })
                  }
                  placeholder={t("channels.field.placeholderText")}
                  aria-label={t("channels.field.placeholderText")}
                />
              </div>
            )}
          </SwitchCardField>
        )}
    </div>
  )
}
