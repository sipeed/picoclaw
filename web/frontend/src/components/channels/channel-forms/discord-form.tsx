import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { getSecretInputPlaceholder } from "@/components/channels/channel-config-fields"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

interface DiscordFormProps {
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

function asBool(value: unknown): boolean {
  return value === true
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

export function DiscordForm({
  config,
  onChange,
  configuredSecrets,
  fieldErrors = {},
}: DiscordFormProps) {
  const { t } = useTranslation()
  const groupTriggerConfig = asRecord(config.group_trigger)

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
              label={t("channels.field.mentionOnly")}
              hint={t("channels.form.desc.mentionOnly")}
              checked={asBool(groupTriggerConfig.mention_only)}
              onCheckedChange={(checked) => {
                onChange("group_trigger", {
                  ...groupTriggerConfig,
                  mention_only: checked,
                })
              }}
              ariaLabel={t("channels.field.mentionOnly")}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
