import { IconCode } from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import {
  type CoreConfigForm,
  DM_SCOPE_OPTIONS,
  type LauncherForm,
} from "@/components/config/form-model"
import { Field, SwitchCardField } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

type UpdateCoreField = <K extends keyof CoreConfigForm>(
  key: K,
  value: CoreConfigForm[K],
) => void

type UpdateLauncherField = <K extends keyof LauncherForm>(
  key: K,
  value: LauncherForm[K],
) => void

interface AgentDefaultsSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function AgentDefaultsSection({
  form,
  onFieldChange,
}: AgentDefaultsSectionProps) {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <div className="space-y-4">
        <Field
          label={t("pages.config.workspace")}
          hint={t("pages.config.workspace_hint")}
        >
          <Input
            value={form.workspace}
            onChange={(e) => onFieldChange("workspace", e.target.value)}
            placeholder="~/.picoclaw/workspace"
          />
        </Field>

        <SwitchCardField
          label={t("pages.config.restrict_workspace")}
          hint={t("pages.config.restrict_workspace_hint")}
          checked={form.restrictToWorkspace}
          onCheckedChange={(checked) =>
            onFieldChange("restrictToWorkspace", checked)
          }
        />

        <Field
          label={t("pages.config.max_tokens")}
          hint={t("pages.config.max_tokens_hint")}
        >
          <Input
            type="number"
            min={1}
            value={form.maxTokens}
            onChange={(e) => onFieldChange("maxTokens", e.target.value)}
          />
        </Field>

        <Field
          label={t("pages.config.max_tool_iterations")}
          hint={t("pages.config.max_tool_iterations_hint")}
        >
          <Input
            type="number"
            min={1}
            value={form.maxToolIterations}
            onChange={(e) => onFieldChange("maxToolIterations", e.target.value)}
          />
        </Field>

        <Field
          label={t("pages.config.summarize_threshold")}
          hint={t("pages.config.summarize_threshold_hint")}
        >
          <Input
            type="number"
            min={1}
            value={form.summarizeMessageThreshold}
            onChange={(e) =>
              onFieldChange("summarizeMessageThreshold", e.target.value)
            }
          />
        </Field>

        <Field
          label={t("pages.config.summarize_token_percent")}
          hint={t("pages.config.summarize_token_percent_hint")}
        >
          <Input
            type="number"
            min={1}
            max={100}
            value={form.summarizeTokenPercent}
            onChange={(e) =>
              onFieldChange("summarizeTokenPercent", e.target.value)
            }
          />
        </Field>
      </div>
    </section>
  )
}

interface RuntimeSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function RuntimeSection({ form, onFieldChange }: RuntimeSectionProps) {
  const { t } = useTranslation()
  const selectedDmScopeOption = DM_SCOPE_OPTIONS.find(
    (scope) => scope.value === form.dmScope,
  )

  return (
    <section className="space-y-3">
      <div className="space-y-4">
        <Field
          label={t("pages.config.session_scope")}
          hint={t("pages.config.session_scope_hint")}
        >
          <Select
            value={form.dmScope}
            onValueChange={(value) => onFieldChange("dmScope", value)}
          >
            <SelectTrigger>
              <SelectValue>
                {selectedDmScopeOption
                  ? t(
                      selectedDmScopeOption.labelKey,
                      selectedDmScopeOption.labelDefault,
                    )
                  : form.dmScope}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {DM_SCOPE_OPTIONS.map((scope) => (
                <SelectItem key={scope.value} value={scope.value}>
                  <div className="flex flex-col gap-0.5">
                    <span className="font-medium">{t(scope.labelKey)}</span>
                    <span className="text-muted-foreground text-xs">
                      {t(scope.descKey)}
                    </span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>

        <SwitchCardField
          label={t("pages.config.heartbeat_enabled")}
          hint={t("pages.config.heartbeat_enabled_hint")}
          checked={form.heartbeatEnabled}
          onCheckedChange={(checked) =>
            onFieldChange("heartbeatEnabled", checked)
          }
        />

        {form.heartbeatEnabled && (
          <Field
            label={t("pages.config.heartbeat_interval")}
            hint={t("pages.config.heartbeat_interval_hint")}
          >
            <Input
              type="number"
              min={1}
              value={form.heartbeatInterval}
              onChange={(e) =>
                onFieldChange("heartbeatInterval", e.target.value)
              }
            />
          </Field>
        )}
      </div>
    </section>
  )
}

interface LauncherSectionProps {
  launcherForm: LauncherForm
  onFieldChange: UpdateLauncherField
  launcherHint: string
  disabled: boolean
}

export function LauncherSection({
  launcherForm,
  onFieldChange,
  launcherHint,
  disabled,
}: LauncherSectionProps) {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <div className="space-y-4">
        <Field
          label={t("pages.config.server_port")}
          hint={t("pages.config.server_port_hint")}
        >
          <Input
            type="number"
            min={1}
            max={65535}
            value={launcherForm.port}
            disabled={disabled}
            onChange={(e) => onFieldChange("port", e.target.value)}
          />
        </Field>

        <SwitchCardField
          label={t("pages.config.lan_access")}
          hint={t("pages.config.lan_access_hint")}
          checked={launcherForm.publicAccess}
          disabled={disabled}
          onCheckedChange={(checked) => onFieldChange("publicAccess", checked)}
        />

        <Field
          label={t("pages.config.allowed_cidrs")}
          hint={t("pages.config.allowed_cidrs_hint")}
        >
          <Textarea
            value={launcherForm.allowedCIDRsText}
            disabled={disabled}
            placeholder={t("pages.config.allowed_cidrs_placeholder")}
            className="min-h-[88px]"
            onChange={(e) => onFieldChange("allowedCIDRsText", e.target.value)}
          />
        </Field>

        <p className="text-muted-foreground text-xs">{launcherHint}</p>
      </div>
    </section>
  )
}

interface DevicesSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
  autoStartEnabled: boolean
  autoStartHint: string
  autoStartDisabled: boolean
  onAutoStartChange: (checked: boolean) => void
}

export function DevicesSection({
  form,
  onFieldChange,
  autoStartEnabled,
  autoStartHint,
  autoStartDisabled,
  onAutoStartChange,
}: DevicesSectionProps) {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <div className="space-y-4">
        <SwitchCardField
          label={t("pages.config.devices_enabled")}
          hint={t("pages.config.devices_enabled_hint")}
          checked={form.devicesEnabled}
          onCheckedChange={(checked) =>
            onFieldChange("devicesEnabled", checked)
          }
        />

        <SwitchCardField
          label={t("pages.config.monitor_usb")}
          hint={t("pages.config.monitor_usb_hint")}
          checked={form.monitorUSB}
          onCheckedChange={(checked) => onFieldChange("monitorUSB", checked)}
        />

        <SwitchCardField
          label={t("pages.config.autostart_label")}
          hint={autoStartHint}
          checked={autoStartEnabled}
          disabled={autoStartDisabled}
          onCheckedChange={onAutoStartChange}
        />
      </div>
    </section>
  )
}

export function AdvancedSection() {
  const { t } = useTranslation()

  return (
    <section className="space-y-3">
      <p className="text-muted-foreground text-sm">
        {t("pages.config.advanced_desc")}
      </p>
      <div>
        <Button variant="outline" asChild>
          <Link to="/config/raw">
            <IconCode className="size-4" />
            {t("pages.config.open_raw")}
          </Link>
        </Button>
      </div>
    </section>
  )
}
