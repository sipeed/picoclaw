import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import {
  AdvancedSection,
  Field,
  KeyInput,
} from "@/components/models/shared-form"

interface SlackFormProps {
  config: Record<string, any>
  onChange: (key: string, value: any) => void
  isEdit: boolean
}

export function SlackForm({ config, onChange, isEdit }: SlackFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.botToken")}
        hint={
          isEdit && config.bot_token
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={config._bot_token ?? ""}
          onChange={(v) => onChange("_bot_token", v)}
          placeholder={
            isEdit && config.bot_token
              ? t("channels.field.secretPlaceholderSet")
              : "xoxb-xxxx"
          }
        />
      </Field>

      <Field
        label={t("channels.field.appToken")}
        hint={
          isEdit && config.app_token
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={config._app_token ?? ""}
          onChange={(v) => onChange("_app_token", v)}
          placeholder={
            isEdit && config.app_token
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
