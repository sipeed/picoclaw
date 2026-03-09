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
          label={t("pages.config.workspace", "Workspace Directory")}
          hint={t(
            "pages.config.workspace_hint",
            "Base directory for agent file operations.",
          )}
        >
          <Input
            value={form.workspace}
            onChange={(e) => onFieldChange("workspace", e.target.value)}
            placeholder="~/.picoclaw/workspace"
          />
        </Field>

        <SwitchCardField
          label={t("pages.config.restrict_workspace", "Restrict to Workspace")}
          hint={t(
            "pages.config.restrict_workspace_hint",
            "Only allow file operations inside workspace.",
          )}
          checked={form.restrictToWorkspace}
          onCheckedChange={(checked) =>
            onFieldChange("restrictToWorkspace", checked)
          }
        />

        <Field
          label={t("pages.config.max_tokens", "Max Tokens")}
          hint={t(
            "pages.config.max_tokens_hint",
            "Upper token limit per model response.",
          )}
        >
          <Input
            type="number"
            min={1}
            value={form.maxTokens}
            onChange={(e) => onFieldChange("maxTokens", e.target.value)}
          />
        </Field>

        <Field
          label={t("pages.config.max_tool_iterations", "Max Tool Iterations")}
          hint={t(
            "pages.config.max_tool_iterations_hint",
            "Maximum tool-call loops in a single task.",
          )}
        >
          <Input
            type="number"
            min={1}
            value={form.maxToolIterations}
            onChange={(e) => onFieldChange("maxToolIterations", e.target.value)}
          />
        </Field>

        <Field
          label={t(
            "pages.config.summarize_threshold",
            "Summarize Message Threshold",
          )}
          hint={t(
            "pages.config.summarize_threshold_hint",
            "Start summarization after this many messages.",
          )}
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
          label={t(
            "pages.config.summarize_token_percent",
            "Summarize Token Percent",
          )}
          hint={t(
            "pages.config.summarize_token_percent_hint",
            "Used when conversation summary is triggered.",
          )}
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
          label={t("pages.config.session_scope", "Session Scope")}
          hint={t(
            "pages.config.session_scope_hint",
            "How chat context is isolated across peers/channels.",
          )}
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
                    <span className="font-medium">
                      {t(scope.labelKey, scope.labelDefault)}
                    </span>
                    <span className="text-muted-foreground text-xs">
                      {t(scope.descKey, scope.descDefault)}
                    </span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>

        <SwitchCardField
          label={t("pages.config.heartbeat_enabled", "Heartbeat")}
          hint={t(
            "pages.config.heartbeat_enabled_hint",
            "Send periodic heartbeat messages.",
          )}
          checked={form.heartbeatEnabled}
          onCheckedChange={(checked) =>
            onFieldChange("heartbeatEnabled", checked)
          }
        />

        {form.heartbeatEnabled && (
          <Field
            label={t(
              "pages.config.heartbeat_interval",
              "Heartbeat Interval (minutes)",
            )}
            hint={t(
              "pages.config.heartbeat_interval_hint",
              "Interval in minutes between heartbeat signals.",
            )}
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
          label={t("pages.config.server_port", "Service Port")}
          hint={t(
            "pages.config.server_port_hint",
            "HTTP port used by PicoClaw Web.",
          )}
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
          label={t("pages.config.lan_access", "Enable LAN Access")}
          hint={t(
            "pages.config.lan_access_hint",
            "Allow access from other devices on your local network.",
          )}
          checked={launcherForm.publicAccess}
          disabled={disabled}
          onCheckedChange={(checked) => onFieldChange("publicAccess", checked)}
        />

        <Field
          label={t("pages.config.allowed_cidrs", "Allowed Network CIDRs")}
          hint={t(
            "pages.config.allowed_cidrs_hint",
            "Only clients from these CIDR ranges can access the service. One per line or comma-separated. Leave empty to allow all.",
          )}
        >
          <Textarea
            value={launcherForm.allowedCIDRsText}
            disabled={disabled}
            placeholder={t(
              "pages.config.allowed_cidrs_placeholder",
              "192.168.1.0/24\n10.0.0.0/8",
            )}
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
          label={t("pages.config.devices_enabled", "Enable Devices")}
          hint={t(
            "pages.config.devices_enabled_hint",
            "Enable hardware-device integrations.",
          )}
          checked={form.devicesEnabled}
          onCheckedChange={(checked) =>
            onFieldChange("devicesEnabled", checked)
          }
        />

        <SwitchCardField
          label={t("pages.config.monitor_usb", "Monitor USB")}
          hint={t(
            "pages.config.monitor_usb_hint",
            "Watch USB plug/unplug events when devices are enabled.",
          )}
          checked={form.monitorUSB}
          onCheckedChange={(checked) => onFieldChange("monitorUSB", checked)}
        />

        <SwitchCardField
          label={t("pages.config.autostart_label", "Launch at Login")}
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
        {t(
          "pages.config.advanced_desc",
          "Open the raw JSON page to edit every field directly.",
        )}
      </p>
      <div>
        <Button variant="outline" asChild>
          <Link to="/config/raw">
            <IconCode className="size-4" />
            {t("pages.config.open_raw", "Raw Config")}
          </Link>
        </Button>
      </div>
    </section>
  )
}
