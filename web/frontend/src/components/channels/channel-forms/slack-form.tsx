import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput } from "@/components/shared-form"
import { Input } from "@/components/ui/input"

interface SlackFormProps {
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

export function SlackForm({
  config,
  onChange,
  isEdit,
  fieldErrors = {},
}: SlackFormProps) {
  const { t } = useTranslation()
  const botTokenExtraHint =
    isEdit && asString(config.bot_token)
      ? ` ${t("channels.field.secretHintSet")}`
      : ""
  const appTokenExtraHint =
    isEdit && asString(config.app_token)
      ? ` ${t("channels.field.secretHintSet")}`
      : ""

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.botToken")}
        required
        hint={`${t("channels.form.desc.botToken")}${botTokenExtraHint}`}
        error={fieldErrors.bot_token}
      >
        <KeyInput
          value={asString(config._bot_token)}
          onChange={(v) => onChange("_bot_token", v)}
          placeholder={maskedSecretPlaceholder(config.bot_token, "xoxb-xxxx")}
        />
      </Field>

      <Field
        label={t("channels.field.appToken")}
        hint={`${t("channels.form.desc.appToken")}${appTokenExtraHint}`}
      >
        <KeyInput
          value={asString(config._app_token)}
          onChange={(v) => onChange("_app_token", v)}
          placeholder={maskedSecretPlaceholder(config.app_token, "xapp-xxxx")}
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
    </div>
  )
}
