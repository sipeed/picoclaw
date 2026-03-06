import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Field, KeyInput } from "@/components/models/shared-form"

interface GenericFormProps {
  channelName: string
  config: Record<string, any>
  onChange: (key: string, value: any) => void
  isEdit: boolean
}

// Secret field names that should use masked input.
const SECRET_FIELDS = new Set([
  "token",
  "app_secret",
  "client_secret",
  "corp_secret",
  "channel_secret",
  "channel_access_token",
  "access_token",
  "bot_token",
  "app_token",
  "encoding_aes_key",
  "encrypt_key",
  "verification_token",
])

// Fields to skip in the generic form (handled by enabled toggle or internal).
const SKIP_FIELDS = new Set(["enabled", "reasoning_channel_id"])

// Fields that are objects/nested — show as JSON or skip.
const OBJECT_FIELDS = new Set([
  "group_trigger",
  "typing",
  "placeholder",
  "allow_from",
])

function formatLabel(key: string): string {
  return key
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ")
}

export function GenericForm({
  config,
  onChange,
  isEdit,
}: GenericFormProps) {
  const { t } = useTranslation()

  const fields = Object.keys(config).filter(
    (k) => !k.startsWith("_") && !SKIP_FIELDS.has(k) && !OBJECT_FIELDS.has(k),
  )

  return (
    <div className="space-y-5">
      {fields.map((key) => {
        if (SECRET_FIELDS.has(key)) {
          const editKey = `_${key}`
          return (
            <Field
              key={key}
              label={formatLabel(key)}
              hint={
                isEdit && config[key]
                  ? t("channels.field.secretHintSet")
                  : undefined
              }
            >
              <KeyInput
                value={config[editKey] ?? ""}
                onChange={(v) => onChange(editKey, v)}
                placeholder={
                  isEdit && config[key]
                    ? t("channels.field.secretPlaceholderSet")
                    : ""
                }
              />
            </Field>
          )
        }

        const value = config[key]
        if (typeof value === "boolean") {
          return null // Booleans are less common in generic; skip for now
        }

        return (
          <Field key={key} label={formatLabel(key)}>
            <Input
              value={String(value ?? "")}
              onChange={(e) => {
                // Attempt to preserve number types
                const v = e.target.value
                if (typeof config[key] === "number") {
                  onChange(key, v === "" ? 0 : Number(v))
                } else {
                  onChange(key, v)
                }
              }}
            />
          </Field>
        )
      })}

      {/* Allow From field */}
      {config.allow_from !== undefined && (
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
      )}
    </div>
  )
}
