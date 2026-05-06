import { useState } from "react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import {
  type CoreConfigForm,
  DM_SCOPE_OPTIONS,
  type LauncherForm,
} from "@/components/config/form-model"
import { Field, SwitchCardField } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
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

interface ConfigSectionCardProps {
  title: string
  description?: string
  children: ReactNode
}

function ConfigSectionCard({
  title,
  description,
  children,
}: ConfigSectionCardProps) {
  return (
    <Card size="sm">
      <CardHeader className="border-border border-b">
        <CardTitle>{title}</CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className="pt-0">
        <div className="divide-border/70 divide-y">{children}</div>
      </CardContent>
    </Card>
  )
}

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
    <ConfigSectionCard title={t("pages.config.sections.agent")}>
      <Field
        label={t("pages.config.workspace")}
        hint={t("pages.config.workspace_hint")}
        layout="setting-row"
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
        layout="setting-row"
        checked={form.restrictToWorkspace}
        onCheckedChange={(checked) =>
          onFieldChange("restrictToWorkspace", checked)
        }
      />

      <SwitchCardField
        label={t("pages.config.split_on_marker")}
        hint={t("pages.config.split_on_marker_hint")}
        layout="setting-row"
        checked={form.splitOnMarker}
        onCheckedChange={(checked) => onFieldChange("splitOnMarker", checked)}
      />

      <SwitchCardField
        label={t("pages.config.tool_feedback_enabled")}
        hint={t("pages.config.tool_feedback_enabled_hint")}
        layout="setting-row"
        checked={form.toolFeedbackEnabled}
        onCheckedChange={(checked) =>
          onFieldChange("toolFeedbackEnabled", checked)
        }
      />

      {form.toolFeedbackEnabled && (
        <SwitchCardField
          label={t("pages.config.tool_feedback_separate_messages")}
          hint={t("pages.config.tool_feedback_separate_messages_hint")}
          layout="setting-row"
          checked={form.toolFeedbackSeparateMessages}
          onCheckedChange={(checked) =>
            onFieldChange("toolFeedbackSeparateMessages", checked)
          }
        />
      )}

      {form.toolFeedbackEnabled && (
        <Field
          label={t("pages.config.tool_feedback_max_args_length")}
          hint={t("pages.config.tool_feedback_max_args_length_hint")}
          layout="setting-row"
        >
          <Input
            type="number"
            min={0}
            value={form.toolFeedbackMaxArgsLength}
            onChange={(e) =>
              onFieldChange("toolFeedbackMaxArgsLength", e.target.value)
            }
          />
        </Field>
      )}

      <Field
        label={t("pages.config.max_tokens")}
        hint={t("pages.config.max_tokens_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          value={form.maxTokens}
          onChange={(e) => onFieldChange("maxTokens", e.target.value)}
        />
      </Field>

      <Field
        label={t("pages.config.context_window")}
        hint={t("pages.config.context_window_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          value={form.contextWindow}
          onChange={(e) => onFieldChange("contextWindow", e.target.value)}
          placeholder="131072"
        />
      </Field>

      <Field
        label={t("pages.config.max_tool_iterations")}
        hint={t("pages.config.max_tool_iterations_hint")}
        layout="setting-row"
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
        layout="setting-row"
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
        layout="setting-row"
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
    </ConfigSectionCard>
  )
}

interface ExecSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function ExecSection({ form, onFieldChange }: ExecSectionProps) {
  const { t } = useTranslation()
  const [testCommand, setTestCommand] = useState("")
  const [testResult, setTestResult] = useState<{
    allowed: boolean
    blocked: boolean
    matchedWhitelist: string | null
    matchedBlacklist: string | null
  } | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const testPatterns = async () => {
    if (!testCommand.trim()) {
      setTestResult(null)
      return
    }

    const allowPatterns = form.customAllowPatternsText
      .split("\n")
      .map((p) => p.trim())
      .filter((p) => p.length > 0)
    const denyPatterns = form.enableDenyPatterns
      ? form.customDenyPatternsText
          .split("\n")
          .map((p) => p.trim())
          .filter((p) => p.length > 0)
      : []

    setIsLoading(true)
    try {
      const res = await fetch("/api/config/test-command-patterns", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          allow_patterns: allowPatterns,
          deny_patterns: denyPatterns,
          command: testCommand,
        }),
      })
      const data = await res.json()
      setTestResult({
        allowed: data.allowed,
        blocked: data.blocked,
        matchedWhitelist: data.matched_whitelist ?? null,
        matchedBlacklist: data.matched_blacklist ?? null,
      })
    } catch {
      setTestResult(null)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <ConfigSectionCard title={t("pages.config.sections.exec")}>
      <SwitchCardField
        label={t("pages.config.exec_enabled")}
        hint={t("pages.config.exec_enabled_hint")}
        layout="setting-row"
        checked={form.execEnabled}
        onCheckedChange={(checked) => onFieldChange("execEnabled", checked)}
      />

      {form.execEnabled && (
        <>
          <SwitchCardField
            label={t("pages.config.allow_remote")}
            hint={t("pages.config.allow_remote_hint")}
            layout="setting-row"
            checked={form.allowRemote}
            onCheckedChange={(checked) => onFieldChange("allowRemote", checked)}
          />

          <SwitchCardField
            label={t("pages.config.enable_deny_patterns")}
            hint={t("pages.config.enable_deny_patterns_hint")}
            layout="setting-row"
            checked={form.enableDenyPatterns}
            onCheckedChange={(checked) =>
              onFieldChange("enableDenyPatterns", checked)
            }
          />

          {form.enableDenyPatterns && (
            <Field
              label={t("pages.config.custom_deny_patterns")}
              hint={t("pages.config.custom_deny_patterns_hint")}
              layout="setting-row"
              controlClassName="md:max-w-md"
            >
              <Textarea
                value={form.customDenyPatternsText}
                placeholder={t("pages.config.custom_patterns_placeholder")}
                className="min-h-[88px]"
                onChange={(e) =>
                  onFieldChange("customDenyPatternsText", e.target.value)
                }
              />
            </Field>
          )}

          <Field
            label={t("pages.config.custom_allow_patterns")}
            hint={t("pages.config.custom_allow_patterns_hint")}
            layout="setting-row"
            controlClassName="md:max-w-md"
          >
            <Textarea
              value={form.customAllowPatternsText}
              placeholder={t("pages.config.custom_patterns_placeholder")}
              className="min-h-[88px]"
              onChange={(e) =>
                onFieldChange("customAllowPatternsText", e.target.value)
              }
            />
          </Field>

          <Field
            label={t("pages.config.pattern_detector_title")}
            hint={t("pages.config.pattern_detector_hint")}
            layout="setting-row"
            controlClassName="md:max-w-md"
          >
            <div className="flex w-full flex-col gap-2">
              <div className="flex gap-2">
                <Input
                  value={testCommand}
                  placeholder={t(
                    "pages.config.pattern_detector_input_placeholder",
                  )}
                  onChange={(e) => setTestCommand(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      testPatterns()
                    }
                  }}
                />
                <Button onClick={testPatterns} disabled={isLoading}>
                  {t("pages.config.pattern_detector_test_button")}
                </Button>
              </div>
              {testResult && (
                <div
                  className={`rounded-md p-2 text-sm ${
                    testResult.allowed
                      ? "bg-green-500/10 text-green-600"
                      : testResult.blocked
                        ? "bg-red-500/10 text-red-600"
                        : "bg-muted text-muted-foreground"
                  }`}
                >
                  {testResult.allowed
                    ? `${t("pages.config.pattern_detector_result_allowed")}${testResult.matchedWhitelist ? ` (${testResult.matchedWhitelist})` : ""}`
                    : testResult.blocked
                      ? `${t("pages.config.pattern_detector_result_blocked")}${testResult.matchedBlacklist ? ` (${testResult.matchedBlacklist})` : ""}`
                      : t("pages.config.pattern_detector_result_no_match")}
                </div>
              )}
            </div>
          </Field>

          <Field
            label={t("pages.config.exec_timeout_seconds")}
            hint={t("pages.config.exec_timeout_seconds_hint")}
            layout="setting-row"
          >
            <Input
              type="number"
              min={0}
              value={form.execTimeoutSeconds}
              onChange={(e) =>
                onFieldChange("execTimeoutSeconds", e.target.value)
              }
            />
          </Field>
        </>
      )}
    </ConfigSectionCard>
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
    <ConfigSectionCard title={t("pages.config.sections.runtime")}>
      <Field
        label={t("pages.config.session_scope")}
        hint={t("pages.config.session_scope_hint")}
        layout="setting-row"
      >
        <Select
          value={form.dmScope}
          onValueChange={(value) => onFieldChange("dmScope", value)}
        >
          <SelectTrigger className="w-full">
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
        layout="setting-row"
        checked={form.heartbeatEnabled}
        onCheckedChange={(checked) =>
          onFieldChange("heartbeatEnabled", checked)
        }
      />

      {form.heartbeatEnabled && (
        <Field
          label={t("pages.config.heartbeat_interval")}
          hint={t("pages.config.heartbeat_interval_hint")}
          layout="setting-row"
        >
          <Input
            type="number"
            min={1}
            value={form.heartbeatInterval}
            onChange={(e) => onFieldChange("heartbeatInterval", e.target.value)}
          />
        </Field>
      )}
    </ConfigSectionCard>
  )
}

interface CronSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function CronSection({ form, onFieldChange }: CronSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard title={t("pages.config.sections.cron")}>
      <SwitchCardField
        label={t("pages.config.allow_shell_execution")}
        hint={t("pages.config.allow_shell_execution_hint")}
        layout="setting-row"
        checked={form.allowCommand}
        disabled={!form.execEnabled}
        onCheckedChange={(checked) => onFieldChange("allowCommand", checked)}
      />

      <Field
        label={t("pages.config.cron_exec_timeout")}
        hint={t("pages.config.cron_exec_timeout_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={0}
          disabled={!form.execEnabled}
          value={form.cronExecTimeoutMinutes}
          onChange={(e) =>
            onFieldChange("cronExecTimeoutMinutes", e.target.value)
          }
        />
      </Field>
    </ConfigSectionCard>
  )
}

interface LauncherSectionProps {
  launcherForm: LauncherForm
  onFieldChange: UpdateLauncherField
  disabled: boolean
}

export function LauncherSection({
  launcherForm,
  onFieldChange,
  disabled,
}: LauncherSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard
      title={t("pages.config.sections.launcher")}
      description={t("pages.config.launcher_section_hint")}
    >
      <Field
        label={t("pages.config.dashboard_password")}
        hint={t("pages.config.dashboard_password_hint")}
        layout="setting-row"
        controlClassName="md:max-w-md"
      >
        <Input
          type="password"
          value={launcherForm.dashboardPassword}
          disabled={disabled}
          autoComplete="new-password"
          placeholder={t("pages.config.dashboard_password_placeholder")}
          onChange={(e) => onFieldChange("dashboardPassword", e.target.value)}
        />
      </Field>

      {launcherForm.dashboardPassword.trim() !== "" && (
        <Field
          label={t("pages.config.dashboard_password_confirm")}
          hint={t("pages.config.dashboard_password_confirm_hint")}
          layout="setting-row"
          controlClassName="md:max-w-md"
        >
          <Input
            type="password"
            value={launcherForm.dashboardPasswordConfirm}
            disabled={disabled}
            autoComplete="new-password"
            placeholder={t(
              "pages.config.dashboard_password_confirm_placeholder",
            )}
            onChange={(e) =>
              onFieldChange("dashboardPasswordConfirm", e.target.value)
            }
          />
        </Field>
      )}

      <SwitchCardField
        label={t("pages.config.lan_access")}
        hint={t("pages.config.lan_access_hint")}
        layout="setting-row"
        checked={launcherForm.publicAccess}
        disabled={disabled}
        onCheckedChange={(checked) => onFieldChange("publicAccess", checked)}
      />

      <Field
        label={t("pages.config.server_port")}
        hint={t("pages.config.server_port_hint")}
        layout="setting-row"
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

      <Field
        label={t("pages.config.allowed_cidrs")}
        hint={t("pages.config.allowed_cidrs_hint")}
        layout="setting-row"
        controlClassName="md:max-w-md"
      >
        <Textarea
          value={launcherForm.allowedCIDRsText}
          disabled={disabled}
          placeholder={t("pages.config.allowed_cidrs_placeholder")}
          className="min-h-[88px]"
          onChange={(e) => onFieldChange("allowedCIDRsText", e.target.value)}
        />
      </Field>
    </ConfigSectionCard>
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
    <ConfigSectionCard title={t("pages.config.sections.devices")}>
      <SwitchCardField
        label={t("pages.config.devices_enabled")}
        hint={t("pages.config.devices_enabled_hint")}
        layout="setting-row"
        checked={form.devicesEnabled}
        onCheckedChange={(checked) => onFieldChange("devicesEnabled", checked)}
      />

      <SwitchCardField
        label={t("pages.config.monitor_usb")}
        hint={t("pages.config.monitor_usb_hint")}
        layout="setting-row"
        checked={form.monitorUSB}
        onCheckedChange={(checked) => onFieldChange("monitorUSB", checked)}
      />

      <SwitchCardField
        label={t("pages.config.autostart_label")}
        hint={autoStartHint}
        layout="setting-row"
        checked={autoStartEnabled}
        disabled={autoStartDisabled}
        onCheckedChange={onAutoStartChange}
      />
    </ConfigSectionCard>
  )
}
