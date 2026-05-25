import { useTranslation } from "react-i18next"

import { Field, SwitchCardField } from "@/components/shared-form"
import { Input } from "@/components/ui/input"

interface StreamingConfigFieldProps {
  value: unknown
  onChange: (value: Record<string, unknown>) => void
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

function numberInputValue(value: unknown): string {
  return typeof value === "number" && value > 0 ? String(value) : ""
}

export function StreamingConfigField({
  value,
  onChange,
}: StreamingConfigFieldProps) {
  const { t } = useTranslation()
  const streamingConfig = asRecord(value)
  const streamingEnabled = asBool(streamingConfig.enabled)

  const update = (patch: Record<string, unknown>) => {
    onChange({ ...streamingConfig, ...patch })
  }

  const handleEnabledChange = (checked: boolean) => {
    if (!checked) {
      onChange({
        enabled: false,
        throttle_seconds: null,
        min_growth_chars: null,
      })
      return
    }
    update({ enabled: true })
  }

  return (
    <SwitchCardField
      label={t("channels.field.streamingEnabled")}
      hint={t("channels.form.desc.streamingEnabled")}
      checked={streamingEnabled}
      onCheckedChange={handleEnabledChange}
      ariaLabel={t("channels.field.streamingEnabled")}
    >
      {streamingEnabled && (
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label={t("channels.field.streamingThrottleSeconds")}
            hint={t("channels.form.desc.streamingThrottleSeconds")}
          >
            <Input
              type="number"
              min={0}
              value={numberInputValue(streamingConfig.throttle_seconds)}
              onChange={(e) =>
                update({
                  throttle_seconds:
                    e.target.value === "" ? 0 : Number(e.target.value),
                })
              }
              placeholder="0"
            />
          </Field>
          <Field
            label={t("channels.field.streamingMinGrowthChars")}
            hint={t("channels.form.desc.streamingMinGrowthChars")}
          >
            <Input
              type="number"
              min={0}
              value={numberInputValue(streamingConfig.min_growth_chars)}
              onChange={(e) =>
                update({
                  min_growth_chars:
                    e.target.value === "" ? 0 : Number(e.target.value),
                })
              }
              placeholder="0"
            />
          </Field>
        </div>
      )}
    </SwitchCardField>
  )
}
