import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import {
  AdvancedSection,
  Field,
  KeyInput,
} from "@/components/models/shared-form"
import { Input } from "@/components/ui/input"

interface FeishuFormProps {
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

export function FeishuForm({ config, onChange, isEdit }: FeishuFormProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-5">
      <Field label={t("channels.field.appId")}>
        <Input
          value={asString(config.app_id)}
          onChange={(e) => onChange("app_id", e.target.value)}
          placeholder="cli_xxxx"
        />
      </Field>

      <Field
        label={t("channels.field.appSecret")}
        hint={
          isEdit && asString(config.app_secret)
            ? t("channels.field.secretHintSet")
            : undefined
        }
      >
        <KeyInput
          value={asString(config._app_secret)}
          onChange={(v) => onChange("_app_secret", v)}
          placeholder={
            isEdit && asString(config.app_secret)
              ? t("channels.field.secretPlaceholderSet")
              : t("channels.field.secretPlaceholder")
          }
        />
      </Field>

      <AdvancedSection>
        <Field label={t("channels.field.verificationToken")}>
          <KeyInput
            value={asString(config._verification_token)}
            onChange={(v) => onChange("_verification_token", v)}
            placeholder={
              isEdit && asString(config.verification_token)
                ? t("channels.field.secretPlaceholderSet")
                : t("channels.field.secretPlaceholder")
            }
          />
        </Field>
        <Field label={t("channels.field.encryptKey")}>
          <KeyInput
            value={asString(config._encrypt_key)}
            onChange={(v) => onChange("_encrypt_key", v)}
            placeholder={
              isEdit && asString(config.encrypt_key)
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
