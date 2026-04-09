import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { getSecretInputPlaceholder } from "@/components/channels/channel-config-fields"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

interface TelegramFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  configuredSecrets: string[]
  fieldErrors?: Record<string, string>
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

export function TelegramForm({
  config,
  onChange,
  configuredSecrets,
  fieldErrors = {},
}: TelegramFormProps) {
  const { t } = useTranslation()
  const typingConfig = asRecord(config.typing)
  const placeholderConfig = asRecord(config.placeholder)
  const placeholderEnabled = asBool(placeholderConfig.enabled)

  return (
    <div className="space-y-6">
      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label={t("channels.field.token")}
            required
            hint={t("channels.form.desc.token")}
            error={fieldErrors.token}
          >
            <KeyInput
              value={asString(config._token)}
              onChange={(v) => onChange("_token", v)}
              placeholder={getSecretInputPlaceholder(
                configuredSecrets,
                "token",
                t("channels.field.secretHintSet"),
                t("channels.field.tokenPlaceholder"),
              )}
            />
          </Field>

          <Field
            label={t("channels.field.baseUrl")}
            hint={t("channels.form.desc.baseUrl")}
          >
            <Input
              value={asString(config.base_url)}
              onChange={(e) => onChange("base_url", e.target.value)}
              placeholder="https://api.telegram.org"
            />
          </Field>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label={t("channels.field.proxy")}
            hint={t("channels.form.desc.proxy")}
          >
            <Input
              value={asString(config.proxy)}
              onChange={(e) => onChange("proxy", e.target.value)}
              placeholder="http://127.0.0.1:7890"
            />
          </Field>
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

          <div>
            <SwitchCardField
              label={t("channels.field.typingEnabled")}
              hint={t("channels.form.desc.typingEnabled")}
              checked={asBool(typingConfig.enabled)}
              onCheckedChange={(checked) =>
                onChange("typing", { ...typingConfig, enabled: checked })
              }
              ariaLabel={t("channels.field.typingEnabled")}
            />
          </div>

          <div>
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
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
