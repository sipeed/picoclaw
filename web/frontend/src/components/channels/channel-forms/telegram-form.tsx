import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import {
  AdvancedSection,
  Field,
  KeyInput,
} from "@/components/models/shared-form"

interface TelegramFormProps {
  config: Record<string, any>
  onChange: (key: string, value: any) => void
  isEdit: boolean
}

export function TelegramForm({ config, onChange, isEdit }: TelegramFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.token")}
        hint={
          isEdit && config.token
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={config._token ?? ""}
          onChange={(v) => onChange("_token", v)}
          placeholder={
            isEdit && config.token
              ? t("channels.field.secretPlaceholderSet")
              : t("channels.field.tokenPlaceholder")
          }
        />
      </Field>

      <AdvancedSection>
        <Field label={t("channels.field.baseUrl")}>
          <Input
            value={config.base_url ?? ""}
            onChange={(e) => onChange("base_url", e.target.value)}
            placeholder="https://api.telegram.org"
          />
        </Field>
        <Field
          label={t("channels.field.proxy")}
          hint={t("channels.field.proxyHint")}
        >
          <Input
            value={config.proxy ?? ""}
            onChange={(e) => onChange("proxy", e.target.value)}
            placeholder="http://127.0.0.1:7890"
          />
        </Field>
        <Field
          label={t("channels.field.allowFrom")}
          hint={t("channels.field.allowFromHint")}
        >
          <Input
            value={(config.allow_from ?? []).join(", ")}
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
