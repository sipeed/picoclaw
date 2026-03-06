import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import {
  AdvancedSection,
  Field,
  KeyInput,
} from "@/components/models/shared-form"

interface FeishuFormProps {
  config: Record<string, any>
  onChange: (key: string, value: any) => void
  isEdit: boolean
}

export function FeishuForm({ config, onChange, isEdit }: FeishuFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-5">
      <Field label={t("channels.field.appId")}>
        <Input
          value={config.app_id ?? ""}
          onChange={(e) => onChange("app_id", e.target.value)}
          placeholder="cli_xxxx"
        />
      </Field>

      <Field
        label={t("channels.field.appSecret")}
        hint={
          isEdit && config.app_secret
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={config._app_secret ?? ""}
          onChange={(v) => onChange("_app_secret", v)}
          placeholder={
            isEdit && config.app_secret
              ? t("channels.field.secretPlaceholderSet")
              : t("channels.field.secretPlaceholder")
          }
        />
      </Field>

      <AdvancedSection>
        <Field label={t("channels.field.verificationToken")}>
          <KeyInput
            value={config._verification_token ?? ""}
            onChange={(v) => onChange("_verification_token", v)}
            placeholder={
              isEdit && config.verification_token
                ? t("channels.field.secretPlaceholderSet")
                : t("channels.field.secretPlaceholder")
            }
          />
        </Field>
        <Field label={t("channels.field.encryptKey")}>
          <KeyInput
            value={config._encrypt_key ?? ""}
            onChange={(v) => onChange("_encrypt_key", v)}
            placeholder={
              isEdit && config.encrypt_key
                ? t("channels.field.secretPlaceholderSet")
                : t("channels.field.secretPlaceholder")
            }
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
