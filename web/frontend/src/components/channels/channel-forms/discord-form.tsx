import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Input } from "@/components/ui/input"

interface DiscordFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  isEdit: boolean
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
  isEdit,
  fieldErrors = {},
}: DiscordFormProps) {
  const { t } = useTranslation()
  const groupTriggerConfig = asRecord(config.group_trigger)
  const tokenExtraHint =
    isEdit && asString(config.token)
      ? ` ${t("channels.field.secretHintSet")}`
      : ""

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.token")}
        required
        hint={`${t("channels.form.desc.token")}${tokenExtraHint}`}
        error={fieldErrors.token}
      >
        <KeyInput
          value={asString(config._token)}
          onChange={(v) => onChange("_token", v)}
          placeholder={maskedSecretPlaceholder(
            config.token,
            t("channels.field.tokenPlaceholder"),
          )}
        />
      </Field>

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
  )
}
