import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Input } from "@/components/ui/input"

interface FeishuFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  isEdit: boolean
  fieldErrors?: Record<string, string>
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asBool(value: unknown): boolean {
  return typeof value === "boolean" ? value : false
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === "string")
}

export function FeishuForm({
  config,
  onChange,
  isEdit,
  fieldErrors = {},
}: FeishuFormProps) {
  const { t } = useTranslation()
  const appSecretExtraHint =
    isEdit && asString(config.app_secret)
      ? ` ${t("channels.field.secretHintSet")}`
      : ""
  const verificationExtraHint =
    isEdit && asString(config.verification_token)
      ? ` ${t("channels.field.secretHintSet")}`
      : ""
  const encryptExtraHint =
    isEdit && asString(config.encrypt_key)
      ? ` ${t("channels.field.secretHintSet")}`
      : ""

  return (
    <div className="space-y-5">
      <Field
        label={t("channels.field.appId")}
        required
        hint={t("channels.form.desc.appId")}
        error={fieldErrors.app_id}
      >
        <Input
          value={asString(config.app_id)}
          onChange={(e) => onChange("app_id", e.target.value)}
          placeholder="cli_xxxx"
        />
      </Field>

      <Field
        label={t("channels.field.appSecret")}
        required
        hint={`${t("channels.form.desc.appSecret")}${appSecretExtraHint}`}
        error={fieldErrors.app_secret}
      >
        <KeyInput
          value={asString(config._app_secret)}
          onChange={(v) => onChange("_app_secret", v)}
          placeholder={maskedSecretPlaceholder(
            config.app_secret,
            t("channels.field.secretPlaceholder"),
          )}
        />
      </Field>

      <Field
        label={t("channels.field.verificationToken")}
        hint={`${t("channels.form.desc.verificationToken")}${verificationExtraHint}`}
      >
        <KeyInput
          value={asString(config._verification_token)}
          onChange={(v) => onChange("_verification_token", v)}
          placeholder={maskedSecretPlaceholder(
            config.verification_token,
            t("channels.field.secretPlaceholder"),
          )}
        />
      </Field>
      <Field
        label={t("channels.field.encryptKey")}
        hint={`${t("channels.form.desc.encryptKey")}${encryptExtraHint}`}
      >
        <KeyInput
          value={asString(config._encrypt_key)}
          onChange={(v) => onChange("_encrypt_key", v)}
          placeholder={maskedSecretPlaceholder(
            config.encrypt_key,
            t("channels.field.secretPlaceholder"),
          )}
        />
      </Field>
      <SwitchCardField
        label={t("channels.field.isLark")}
        hint={t("channels.form.desc.isLark")}
        checked={asBool(config.is_lark)}
        onCheckedChange={(checked) => onChange("is_lark", checked)}
      />
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
