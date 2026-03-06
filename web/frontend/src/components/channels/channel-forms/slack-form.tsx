import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import {
  AdvancedSection,
  Field,
  KeyInput,
} from "@/components/models/shared-form"
import { Input } from "@/components/ui/input"

interface SlackFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  isEdit: boolean
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === "string")
}

export function SlackForm({ config, onChange, isEdit }: SlackFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.botToken")}
        hint={
          isEdit && asString(config.bot_token)
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={asString(config._bot_token)}
          onChange={(v) => onChange("_bot_token", v)}
          placeholder={
            isEdit && asString(config.bot_token)
              ? t("channels.field.secretPlaceholderSet")
              : "xoxb-xxxx"
          }
        />
      </Field>

      <Field
        label={t("channels.field.appToken")}
        hint={
          isEdit && asString(config.app_token)
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={asString(config._app_token)}
          onChange={(v) => onChange("_app_token", v)}
          placeholder={
            isEdit && asString(config.app_token)
              ? t("channels.field.secretPlaceholderSet")
              : "xapp-xxxx"
          }
        />
      </Field>

      <AdvancedSection>
        <Field
          label={t("channels.field.allowFrom")}
          hint={t("channels.field.allowFromHint")}
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
      </AdvancedSection>
    </div>
  )
}
