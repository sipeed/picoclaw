import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import {
  type ArrayFieldFlusher,
  ChannelArrayListField,
} from "@/components/channels/channel-array-list-field"
import {
  asStringArray,
  parseAllowFromInput,
} from "@/components/channels/channel-array-utils"
import { getSecretInputPlaceholder } from "@/components/channels/channel-config-fields"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

interface TelegramFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  configuredSecrets: string[]
  fieldErrors?: Record<string, string>
  registerArrayFieldFlusher?: (
    fieldPath: string,
    flusher: ArrayFieldFlusher | null,
  ) => void
  arrayFieldResetVersion?: number
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function asBool(value: unknown): boolean {
  return value === true
}

export function TelegramForm({
  config,
  onChange,
  configuredSecrets,
  fieldErrors = {},
  registerArrayFieldFlusher,
  arrayFieldResetVersion,
}: TelegramFormProps) {
  const { t } = useTranslation()
  const typingConfig = asRecord(config.typing)
  const placeholderConfig = asRecord(config.placeholder)
  const placeholderEnabled = asBool(placeholderConfig.enabled)

  return (
    <div className="space-y-6">
      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label={t("channels.field.token")}
            required
            hint={t("channels.form.desc.token")}
            error={fieldErrors.token}
          >
            <KeyInput
              value={asString(config._token)}
              onChange={(v) => onChange("_token", v)}
              placeholder={getSecretInputPlaceholder(
                configuredSecrets,
                "token",
                t("channels.field.secretHintSet"),
                t("channels.field.tokenPlaceholder"),
              )}
            />
          </Field>

          <Field
            label={t("channels.field.baseUrl")}
            hint={t("channels.form.desc.baseUrl")}
          >
            <Input
              value={asString(config.base_url)}
              onChange={(e) => onChange("base_url", e.target.value)}
              placeholder="https://api.telegram.org"
            />
          </Field>

          <div>
            <SwitchCardField
              label={t("channels.field.businessMode")}
              hint={t("channels.form.desc.businessMode")}
              checked={asBool(config.business_mode)}
              onCheckedChange={(checked) => onChange("business_mode", checked)}
              ariaLabel={t("channels.field.businessMode")}
            />
          </div>

          <Field
            label={t("channels.field.businessOwner")}
            hint={t("channels.form.desc.businessOwner")}
          >
            <Input
              value={asString(config.business_owner)}
              onChange={(e) => onChange("business_owner", e.target.value)}
              placeholder="123456789"
            />
          </Field>

          <div>
            <SwitchCardField
              label={t("channels.field.businessCommandsEnable")}
              hint={t("channels.form.desc.businessCommandsEnable")}
              checked={asBool(config.business_commands_enable)}
              onCheckedChange={(checked) =>
                onChange("business_commands_enable", checked)
              }
              ariaLabel={t("channels.field.businessCommandsEnable")}
            />
          </div>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
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
          <ChannelArrayListField
            label={t("channels.field.allowFrom")}
            hint={t("channels.form.desc.allowFrom")}
            value={asStringArray(config.allow_from)}
            onChange={(value) => onChange("allow_from", value)}
            placeholder={t("channels.field.allowFromPlaceholder")}
            parser={parseAllowFromInput}
            fieldPath="allow_from"
            registerFlusher={registerArrayFieldFlusher}
            resetVersion={arrayFieldResetVersion}
          />

          <div>
            <SwitchCardField
              label={t("channels.field.typingEnabled")}
              hint={t("channels.form.desc.typingEnabled")}
              checked={asBool(typingConfig.enabled)}
              onCheckedChange={(checked) =>
                onChange("typing", { ...typingConfig, enabled: checked })
              }
              ariaLabel={t("channels.field.typingEnabled")}
            />
          </div>

          <div>
            <SwitchCardField
              label={t("channels.field.placeholderEnabled")}
              hint={t("channels.form.desc.placeholderEnabled")}
              checked={placeholderEnabled}
              onCheckedChange={(checked) =>
                onChange("placeholder", {
                  ...placeholderConfig,
                  enabled: checked,
                })
              }
              ariaLabel={t("channels.field.placeholderEnabled")}
            >
              {placeholderEnabled && (
                <div className="space-y-1">
                  <Input
                    value={asString(placeholderConfig.text)}
                    onChange={(e) =>
                      onChange("placeholder", {
                        ...placeholderConfig,
                        text: e.target.value,
                      })
                    }
                    placeholder={t("channels.field.placeholderText")}
                    aria-label={t("channels.field.placeholderText")}
                  />
                </div>
              )}
            </SwitchCardField>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
