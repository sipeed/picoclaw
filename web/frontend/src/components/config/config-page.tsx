import { IconCode, IconDeviceFloppy, IconTag } from "@tabler/icons-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { patchAppConfig } from "@/api/channels"
import { launcherFetch } from "@/api/http"
import { postLauncherDashboardSetup } from "@/api/launcher-auth"
import {
  getAutoStartStatus,
  getLauncherConfig,
  getSystemVersionInfo,
  setAutoStartEnabled as updateAutoStartEnabled,
  setLauncherConfig as updateLauncherConfig,
} from "@/api/system"
import { ConfigChangeNotice } from "@/components/config-change-notice"
import {
  AgentDefaultsSection,
  CronSection,
  DevicesSection,
  EvolutionSection,
  ExecSection,
  LauncherSection,
  MCPSection,
  RuntimeSection,
} from "@/components/config/config-sections"
import {
  type CoreConfigForm,
  EMPTY_FORM,
  EMPTY_LAUNCHER_FORM,
  type LauncherForm,
  type MCPServerForm,
  buildFormFromConfig,
  parseCIDRText,
  parseFloatField,
  parseIntField,
  parseJSONObjectField,
  parseMultilineList,
} from "@/components/config/form-model"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"

function buildStringMapMergePatch(
  next: Record<string, string>,
  previous: Record<string, string>,
): Record<string, string | null> {
  const patch: Record<string, string | null> = { ...next }

  for (const key of Object.keys(previous)) {
    if (!(key in next)) {
      patch[key] = null
    }
  }

  return patch
}

export function ConfigPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<CoreConfigForm>(EMPTY_FORM)
  const [baseline, setBaseline] = useState<CoreConfigForm>(EMPTY_FORM)
  const [launcherForm, setLauncherForm] =
    useState<LauncherForm>(EMPTY_LAUNCHER_FORM)
  const [launcherBaseline, setLauncherBaseline] =
    useState<LauncherForm>(EMPTY_LAUNCHER_FORM)
  const [autoStartEnabled, setAutoStartEnabled] = useState(false)
  const [autoStartBaseline, setAutoStartBaseline] = useState(false)
  const [saving, setSaving] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ["config"],
    queryFn: async () => {
      const res = await launcherFetch("/api/config")
      if (!res.ok) {
        throw new Error("Failed to load config")
      }
      return res.json()
    },
  })

  const { data: launcherConfig, isLoading: isLauncherLoading } = useQuery({
    queryKey: ["system", "launcher-config"],
    queryFn: getLauncherConfig,
  })

  const { data: versionInfo } = useQuery({
    queryKey: ["system", "version"],
    queryFn: getSystemVersionInfo,
    staleTime: 5 * 60 * 1000,
  })

  const {
    data: autoStartStatus,
    isLoading: isAutoStartLoading,
    error: autoStartError,
  } = useQuery({
    queryKey: ["system", "autostart"],
    queryFn: getAutoStartStatus,
  })

  useEffect(() => {
    if (!data) return
    const parsed = buildFormFromConfig(data)
    setForm(parsed)
    setBaseline(parsed)
  }, [data])

  useEffect(() => {
    if (!launcherConfig) return
    const parsed: LauncherForm = {
      port: String(launcherConfig.port),
      publicAccess: launcherConfig.public,
      allowedCIDRsText: (launcherConfig.allowed_cidrs ?? []).join("\n"),
      dashboardPassword: "",
      dashboardPasswordConfirm: "",
    }
    setLauncherForm(parsed)
    setLauncherBaseline(parsed)
  }, [launcherConfig])

  useEffect(() => {
    if (!autoStartStatus) return
    setAutoStartEnabled(autoStartStatus.enabled)
    setAutoStartBaseline(autoStartStatus.enabled)
  }, [autoStartStatus])

  const configDirty = JSON.stringify(form) !== JSON.stringify(baseline)
  const launcherSettingsDirty =
    launcherForm.port !== launcherBaseline.port ||
    launcherForm.publicAccess !== launcherBaseline.publicAccess ||
    launcherForm.allowedCIDRsText !== launcherBaseline.allowedCIDRsText
  const launcherPasswordDirty =
    launcherForm.dashboardPassword.trim() !== "" ||
    launcherForm.dashboardPasswordConfirm.trim() !== ""
  const launcherDirty = launcherSettingsDirty || launcherPasswordDirty
  const autoStartDirty = autoStartEnabled !== autoStartBaseline
  const isDirty = configDirty || launcherDirty || autoStartDirty

  const autoStartSupported = autoStartStatus?.supported !== false
  const autoStartHint = autoStartError
    ? t("pages.config.autostart_load_error")
    : !autoStartSupported
      ? t("pages.config.autostart_unsupported")
      : t("pages.config.autostart_hint")

  const updateField = <K extends keyof CoreConfigForm>(
    key: K,
    value: CoreConfigForm[K],
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const updateLauncherField = <K extends keyof LauncherForm>(
    key: K,
    value: LauncherForm[K],
  ) => {
    setLauncherForm((prev) => ({ ...prev, [key]: value }))
  }

  const handleMCPServerAdd = () => {
    const nextIndex = form.mcpServers.length + 1
    const server: MCPServerForm = {
      id: `mcp-${Date.now()}-${nextIndex}`,
      name: "",
      enabled: true,
      deferredOverride: null,
      type: "stdio",
      url: "",
      command: "",
      argsText: "",
      envText: "{}",
      envFile: "",
      headersText: "{}",
    }
    updateField("mcpServers", [...form.mcpServers, server])
  }

  const handleMCPServerRemove = (id: string) => {
    updateField(
      "mcpServers",
      form.mcpServers.filter((server) => server.id !== id),
    )
  }

  const handleMCPServerFieldChange = <K extends keyof MCPServerForm>(
    id: string,
    key: K,
    value: MCPServerForm[K],
  ) => {
    updateField(
      "mcpServers",
      form.mcpServers.map((server) =>
        server.id === id ? { ...server, [key]: value } : server,
      ),
    )
  }

  const handleReset = () => {
    setForm(baseline)
    setLauncherForm(launcherBaseline)
    setAutoStartEnabled(autoStartBaseline)
    toast.info(t("pages.config.reset_success"))
  }

  const handleSave = async () => {
    try {
      setSaving(true)
      const password = launcherForm.dashboardPassword.trim()
      const confirm = launcherForm.dashboardPasswordConfirm.trim()
      if (launcherPasswordDirty) {
        if (!password) {
          throw new Error(t("pages.config.dashboard_password_required"))
        }
        if (password !== confirm) {
          throw new Error(t("pages.config.dashboard_password_mismatch"))
        }
        if (Array.from(password).length < 8) {
          throw new Error(t("pages.config.dashboard_password_min_length"))
        }
      }

      if (configDirty) {
        const workspace = form.workspace.trim()
        const dmScope = form.dmScope.trim()

        if (!workspace) {
          throw new Error("Workspace path is required.")
        }
        if (!dmScope) {
          throw new Error("Session scope is required.")
        }

        if (
          form.mcpEnabled &&
          form.mcpDiscoveryEnabled &&
          !form.mcpDiscoveryUseBM25 &&
          !form.mcpDiscoveryUseRegex
        ) {
          throw new Error(
            "MCP discovery requires at least one search method (BM25 or regex).",
          )
        }

        const maxTokens = parseIntField(form.maxTokens, "Max tokens", {
          min: 1,
        })
        const contextWindow = form.contextWindow.trim()
          ? parseIntField(form.contextWindow, "Context window", { min: 1 })
          : undefined
        const maxToolIterations = parseIntField(
          form.maxToolIterations,
          "Max tool iterations",
          { min: 1 },
        )
        const toolFeedbackMaxArgsLength = parseIntField(
          form.toolFeedbackMaxArgsLength,
          "Tool feedback max args length",
          { min: 0 },
        )
        const summarizeMessageThreshold = parseIntField(
          form.summarizeMessageThreshold,
          "Summarize message threshold",
          { min: 1 },
        )
        const summarizeTokenPercent = parseIntField(
          form.summarizeTokenPercent,
          "Summarize token percent",
          { min: 1, max: 100 },
        )
        const heartbeatInterval = parseIntField(
          form.heartbeatInterval,
          "Heartbeat interval",
          { min: 1 },
        )
        const cronExecTimeoutMinutes = parseIntField(
          form.cronExecTimeoutMinutes,
          "Cron exec timeout",
          { min: 0 },
        )
        const evolutionMinTaskCount = parseIntField(
          form.evolutionMinTaskCount,
          "Evolution minimum task count",
          { min: 1 },
        )
        const evolutionMinSuccessRatio = parseFloatField(
          form.evolutionMinSuccessRatio,
          "Evolution minimum success ratio",
          { min: 0.01, max: 1 },
        )
        const mcpDiscoveryValidationEnabled =
          form.mcpEnabled && form.mcpDiscoveryEnabled
        const mcpDiscoveryPatch: Record<string, unknown> = {
          enabled: form.mcpDiscoveryEnabled,
          use_bm25: form.mcpDiscoveryUseBM25,
          use_regex: form.mcpDiscoveryUseRegex,
        }

        if (mcpDiscoveryValidationEnabled) {
          mcpDiscoveryPatch.ttl = parseIntField(
            form.mcpDiscoveryTTL,
            "MCP discovery ttl",
            {
              min: 1,
            },
          )
          mcpDiscoveryPatch.max_search_results = parseIntField(
            form.mcpDiscoveryMaxSearchResults,
            "MCP discovery max search results",
            { min: 1 },
          )
        }
        const execConfigPatch: Record<string, unknown> = {
          enabled: form.execEnabled,
        }

        let mcpServersPatch: Record<string, Record<string, unknown> | null> = {}
        if (form.mcpEnabled) {
          const baselineServerNames = new Set(
            baseline.mcpServers
              .map((server) => server.name.trim())
              .filter((name) => name !== ""),
          )

          const normalizedServers = form.mcpServers
            .map((server) => ({
              ...server,
              name: server.name.trim(),
              url: server.url.trim(),
              command: server.command.trim(),
              envFile: server.envFile.trim(),
            }))
            .filter((server) => server.name !== "")

          const serverNameCounts = new Map<string, number>()
          for (const server of normalizedServers) {
            serverNameCounts.set(
              server.name,
              (serverNameCounts.get(server.name) ?? 0) + 1,
            )
          }

          const duplicateNames = Array.from(serverNameCounts.entries())
            .filter(([, count]) => count > 1)
            .map(([name]) => name)
            .sort((a, b) => a.localeCompare(b))

          if (duplicateNames.length > 0) {
            throw new Error(
              `MCP server names must be unique. Duplicates: ${duplicateNames.join(", ")}.`,
            )
          }

          const currentServerNames = new Set(
            normalizedServers.map((server) => server.name),
          )

          const removedServerEntries = Array.from(baselineServerNames)
            .filter((name) => !currentServerNames.has(name))
            .map((name) => [name, null] as const)

          const baselineServersByName = new Map(
            baseline.mcpServers
              .map((server) => ({
                ...server,
                name: server.name.trim(),
              }))
              .filter((server) => server.name !== "")
              .map((server) => [server.name, server] as const),
          )

          const upsertServerEntries = normalizedServers.map((server) => {
            const deferredPatch = { deferred: server.deferredOverride }
            const baselineServer = baselineServersByName.get(server.name)
            const shouldValidateServer = server.enabled

            if (server.type !== "stdio") {
              if (shouldValidateServer && server.url === "") {
                throw new Error(`MCP server ${server.name} requires a URL.`)
              }

              if (shouldValidateServer) {
                try {
                  const parsedURL = new URL(server.url)
                  if (
                    parsedURL.protocol !== "http:" &&
                    parsedURL.protocol !== "https:"
                  ) {
                    throw new Error("invalid protocol")
                  }
                } catch {
                  throw new Error(
                    `MCP server ${server.name} requires a valid HTTP(S) URL.`,
                  )
                }
              }

              const baselineHeaders = baselineServer
                ? parseJSONObjectField(
                    baselineServer.headersText,
                    `Saved MCP server ${server.name} headers`,
                  )
                : {}

              return [
                server.name,
                {
                  ...deferredPatch,
                  enabled: server.enabled,
                  type: server.type,
                  url: server.url,
                  headers: buildStringMapMergePatch(
                    shouldValidateServer
                      ? parseJSONObjectField(
                          server.headersText,
                          `MCP server ${server.name} headers`,
                        )
                      : baselineHeaders,
                    baselineHeaders,
                  ),
                  command: null,
                  args: null,
                  env: null,
                  env_file: null,
                },
              ] as const
            }

            if (shouldValidateServer && server.command === "") {
              throw new Error(`MCP server ${server.name} requires a command.`)
            }

            const baselineEnv = baselineServer
              ? parseJSONObjectField(
                  baselineServer.envText,
                  `Saved MCP server ${server.name} env`,
                )
              : {}

            return [
              server.name,
              {
                ...deferredPatch,
                enabled: server.enabled,
                type: "stdio",
                command: server.command,
                args: parseMultilineList(server.argsText),
                env: buildStringMapMergePatch(
                  shouldValidateServer
                    ? parseJSONObjectField(
                        server.envText,
                        `MCP server ${server.name} env`,
                      )
                    : baselineEnv,
                  baselineEnv,
                ),
                env_file: server.envFile === "" ? null : server.envFile,
                url: null,
                headers: null,
              },
            ] as const
          })

          mcpServersPatch = Object.fromEntries([
            ...upsertServerEntries,
            ...removedServerEntries,
          ])
        }

        if (form.execEnabled) {
          execConfigPatch.allow_remote = form.allowRemote
          execConfigPatch.enable_deny_patterns = form.enableDenyPatterns
          execConfigPatch.custom_allow_patterns = parseMultilineList(
            form.customAllowPatternsText,
          )
          execConfigPatch.timeout_seconds = parseIntField(
            form.execTimeoutSeconds,
            "Exec timeout",
            { min: 0 },
          )

          if (form.enableDenyPatterns) {
            execConfigPatch.custom_deny_patterns = parseMultilineList(
              form.customDenyPatternsText,
            )
          }
        }

        await patchAppConfig({
          agents: {
            defaults: {
              workspace,
              restrict_to_workspace: form.restrictToWorkspace,
              streaming_enabled: form.streamingEnabled,
              split_on_marker: form.splitOnMarker,
              tool_feedback: {
                enabled: form.toolFeedbackEnabled,
                max_args_length: toolFeedbackMaxArgsLength,
                separate_messages: form.toolFeedbackSeparateMessages,
              },
              max_tokens: maxTokens,
              context_window: contextWindow,
              max_tool_iterations: maxToolIterations,
              summarize_message_threshold: summarizeMessageThreshold,
              summarize_token_percent: summarizeTokenPercent,
            },
          },
          session: {
            dm_scope: dmScope,
          },
          evolution: {
            enabled: form.evolutionEnabled,
            mode: form.evolutionMode,
            state_dir:
              form.evolutionStateDir.trim() === ""
                ? null
                : form.evolutionStateDir.trim(),
            min_task_count: evolutionMinTaskCount,
            min_success_ratio: evolutionMinSuccessRatio,
            cold_path_trigger: form.evolutionColdPathTrigger,
            cold_path_times: parseMultilineList(
              form.evolutionColdPathTimesText,
            ),
          },
          tools: {
            cron: {
              allow_command: form.allowCommand,
              exec_timeout_minutes: cronExecTimeoutMinutes,
            },
            exec: execConfigPatch,
            mcp: {
              enabled: form.mcpEnabled,
              discovery: mcpDiscoveryPatch,
              servers: mcpServersPatch,
            },
          },
          heartbeat: {
            enabled: form.heartbeatEnabled,
            interval: heartbeatInterval,
          },
          devices: {
            enabled: form.devicesEnabled,
            monitor_usb: form.monitorUSB,
          },
        })

        setBaseline(form)
        queryClient.invalidateQueries({ queryKey: ["config"] })
      }

      let savedLauncherForm: LauncherForm | null = null
      if (launcherSettingsDirty) {
        const port = parseIntField(launcherForm.port, "Service port", {
          min: 1,
          max: 65535,
        })
        const allowedCIDRs = parseCIDRText(launcherForm.allowedCIDRsText)
        const savedLauncherConfig = await updateLauncherConfig({
          port,
          public: launcherForm.publicAccess,
          allowed_cidrs: allowedCIDRs,
        })
        const parsedLauncher: LauncherForm = {
          port: String(savedLauncherConfig.port),
          publicAccess: savedLauncherConfig.public,
          allowedCIDRsText: (savedLauncherConfig.allowed_cidrs ?? []).join(
            "\n",
          ),
          dashboardPassword: "",
          dashboardPasswordConfirm: "",
        }
        savedLauncherForm = parsedLauncher
        setLauncherForm(parsedLauncher)
        setLauncherBaseline(parsedLauncher)
        queryClient.setQueryData(
          ["system", "launcher-config"],
          savedLauncherConfig,
        )
      }

      if (launcherPasswordDirty) {
        const result = await postLauncherDashboardSetup(password, confirm)
        if (!result.ok) {
          throw new Error(result.error)
        }

        const clearedLauncherForm = savedLauncherForm ?? {
          ...launcherForm,
          dashboardPassword: "",
          dashboardPasswordConfirm: "",
        }
        setLauncherForm(clearedLauncherForm)
        if (savedLauncherForm) {
          setLauncherBaseline(savedLauncherForm)
        }
      }

      if (autoStartDirty) {
        if (!autoStartSupported) {
          throw new Error(t("pages.config.autostart_unsupported"))
        }
        const status = await updateAutoStartEnabled(autoStartEnabled)
        setAutoStartEnabled(status.enabled)
        setAutoStartBaseline(status.enabled)
        queryClient.setQueryData(["system", "autostart"], status)
      }

      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        t("pages.config.save_success"),
        t("navigation.config"),
        gateway?.restartRequired === true,
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("pages.config.save_error"),
      )
    } finally {
      setSaving(false)
    }
  }

  const actionButtons = (
    <div className="flex justify-end gap-2">
      <Button
        variant="outline"
        onClick={handleReset}
        disabled={!isDirty || saving}
      >
        {t("common.reset")}
      </Button>
      <Button onClick={handleSave} disabled={!isDirty || saving}>
        <IconDeviceFloppy className="size-4" />
        {saving ? t("common.saving") : t("common.save")}
      </Button>
    </div>
  )

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={t("navigation.config")}
        titleExtra={
          versionInfo && (
            <Badge
              variant="secondary"
              className="gap-1 font-mono text-[11px] font-normal opacity-80"
            >
              <IconTag className="size-3 opacity-70" />
              {versionInfo.version}
            </Badge>
          )
        }
        children={
          <Button variant="outline" asChild>
            <Link to="/config/raw">
              <IconCode className="size-4" />
              {t("pages.config.open_raw")}
            </Link>
          </Button>
        }
      />
      <div className="flex-1 overflow-auto p-3 lg:p-6">
        <div className="mx-auto w-full max-w-[1000px] space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.config.load_error")}
            </div>
          ) : (
            <div className="space-y-6">
              <LauncherSection
                launcherForm={launcherForm}
                onFieldChange={updateLauncherField}
                disabled={saving || isLauncherLoading}
              />

              <AgentDefaultsSection form={form} onFieldChange={updateField} />

              <RuntimeSection form={form} onFieldChange={updateField} />

              <EvolutionSection form={form} onFieldChange={updateField} />

              <MCPSection
                form={form}
                onFieldChange={updateField}
                onAddServer={handleMCPServerAdd}
                onRemoveServer={handleMCPServerRemove}
                onServerFieldChange={handleMCPServerFieldChange}
              />

              <ExecSection form={form} onFieldChange={updateField} />

              <CronSection form={form} onFieldChange={updateField} />

              <DevicesSection
                form={form}
                onFieldChange={updateField}
                autoStartEnabled={autoStartEnabled}
                autoStartHint={autoStartHint}
                autoStartDisabled={
                  isAutoStartLoading ||
                  Boolean(autoStartError) ||
                  !autoStartSupported ||
                  saving
                }
                onAutoStartChange={setAutoStartEnabled}
              />

              {!isDirty && actionButtons}
            </div>
          )}
        </div>
      </div>
      {isDirty && (
        <div className="border-border/70 bg-background/95 supports-backdrop-filter:bg-background/80 shrink-0 border-t px-3 py-3 shadow-[0_-12px_30px_rgba(15,23,42,0.10)] backdrop-blur lg:px-6">
          <div className="mx-auto flex w-full max-w-[1000px] flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex-1">
              <ConfigChangeNotice
                kind="save"
                title={t("common.saveChangesTitle")}
                description={t("pages.config.unsaved_changes")}
              />
            </div>
            {actionButtons}
          </div>
        </div>
      )}
    </div>
  )
}
