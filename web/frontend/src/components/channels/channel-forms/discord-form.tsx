import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import {
  AdvancedSection,
  Field,
  KeyInput,
} from "@/components/models/shared-form"
import { Input } from "@/components/ui/input"

interface DiscordFormProps {
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

export function DiscordForm({ config, onChange, isEdit }: DiscordFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.token")}
        hint={
          isEdit && asString(config.token)
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={asString(config._token)}
          onChange={(v) => onChange("_token", v)}
          placeholder={
            isEdit && asString(config.token)
              ? t("channels.field.secretPlaceholderSet")
              : t("channels.field.tokenPlaceholder")
          }
        />
      </Field>

      <AdvancedSection>
        <Field
          label={t("channels.field.proxy")}
          hint={t("channels.field.proxyHint")}
        >
          <Input
            value={asString(config.proxy)}
            onChange={(e) => onChange("proxy", e.target.value)}
            placeholder="http://127.0.0.1:7890"
          />
        </Field>
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
