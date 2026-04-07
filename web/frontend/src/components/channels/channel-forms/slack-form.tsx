import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { getSecretInputPlaceholder } from "@/components/channels/channel-config-fields"
import { Field, KeyInput } from "@/components/shared-form"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

interface SlackFormProps {
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

export function SlackForm({
  config,
  onChange,
  configuredSecrets,
  fieldErrors = {},
}: SlackFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-6">
      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label={t("channels.field.botToken")}
            required
            hint={t("channels.form.desc.botToken")}
            error={fieldErrors.bot_token}
          >
            <KeyInput
              value={asString(config._bot_token)}
              onChange={(v) => onChange("_bot_token", v)}
              placeholder={getSecretInputPlaceholder(
                configuredSecrets,
                "bot_token",
                t("channels.field.secretHintSet"),
                "xoxb-xxxx",
              )}
            />
          </Field>

          <Field
            label={t("channels.field.appToken")}
            hint={t("channels.form.desc.appToken")}
          >
            <KeyInput
              value={asString(config._app_token)}
              onChange={(v) => onChange("_app_token", v)}
              placeholder={getSecretInputPlaceholder(
                configuredSecrets,
                "app_token",
                t("channels.field.secretHintSet"),
                "xapp-xxxx",
              )}
            />
          </Field>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
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
        </CardContent>
      </Card>
    </div>
  )
}
