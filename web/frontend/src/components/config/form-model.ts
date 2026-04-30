export type JsonRecord = Record<string, unknown>

export interface CoreConfigForm {
  workspace: string
  restrictToWorkspace: boolean
  splitOnMarker: boolean
  toolFeedbackEnabled: boolean
  toolFeedbackMaxArgsLength: string
  toolFeedbackSeparateMessages: boolean
  execEnabled: boolean
  allowRemote: boolean
  enableDenyPatterns: boolean
  customDenyPatternsText: string
  customAllowPatternsText: string
  execTimeoutSeconds: string
  allowCommand: boolean
  cronExecTimeoutMinutes: string
  maxTokens: string
  contextWindow: string
  maxToolIterations: string
  summarizeMessageThreshold: string
  summarizeTokenPercent: string
  dmScope: string
  heartbeatEnabled: boolean
  heartbeatInterval: string
  devicesEnabled: boolean
  monitorUSB: boolean
}

export interface LauncherForm {
  port: string
  publicAccess: boolean
  allowedCIDRsText: string
  dashboardPassword: string
  dashboardPasswordConfirm: string
}

export const DM_SCOPE_OPTIONS = [
  {
    value: "per-channel-peer",
    labelKey: "pages.config.session_scope_per_channel_peer",
    labelDefault: "Per Channel + Peer",
    descKey: "pages.config.session_scope_per_channel_peer_desc",
    descDefault: "Separate context for each user in each channel.",
  },
  {
    value: "per-channel",
    labelKey: "pages.config.session_scope_per_channel",
    labelDefault: "Per Channel",
    descKey: "pages.config.session_scope_per_channel_desc",
    descDefault: "One shared context per channel.",
  },
  {
    value: "per-peer",
    labelKey: "pages.config.session_scope_per_peer",
    labelDefault: "Per Peer",
    descKey: "pages.config.session_scope_per_peer_desc",
    descDefault: "One context per user across channels.",
  },
  {
    value: "global",
    labelKey: "pages.config.session_scope_global",
    labelDefault: "Global",
    descKey: "pages.config.session_scope_global_desc",
    descDefault: "All messages share one global context.",
  },
] as const

export const EMPTY_FORM: CoreConfigForm = {
  workspace: "",
  restrictToWorkspace: true,
  splitOnMarker: false,
  toolFeedbackEnabled: false,
  toolFeedbackMaxArgsLength: "300",
  toolFeedbackSeparateMessages: false,
  execEnabled: true,
  allowRemote: true,
  enableDenyPatterns: true,
  customDenyPatternsText: "",
  customAllowPatternsText: "",
  execTimeoutSeconds: "0",
  allowCommand: true,
  cronExecTimeoutMinutes: "5",
  maxTokens: "32768",
  contextWindow: "",
  maxToolIterations: "50",
  summarizeMessageThreshold: "20",
  summarizeTokenPercent: "75",
  dmScope: "per-channel-peer",
  heartbeatEnabled: true,
  heartbeatInterval: "30",
  devicesEnabled: false,
  monitorUSB: true,
}

export const EMPTY_LAUNCHER_FORM: LauncherForm = {
  port: "18800",
  publicAccess: false,
  allowedCIDRsText: "",
  dashboardPassword: "",
  dashboardPasswordConfirm: "",
}

function asRecord(value: unknown): JsonRecord {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as JsonRecord
  }
  return {}
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asBool(value: unknown): boolean {
  return value === true
}

function asNumberString(value: unknown, fallback: string): string {
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value)
  }
  if (typeof value === "string" && value.trim() !== "") {
    return value
  }
  return fallback
}

export function buildFormFromConfig(config: unknown): CoreConfigForm {
  const root = asRecord(config)
  const agents = asRecord(root.agents)
  const defaults = asRecord(agents.defaults)
  const session = asRecord(root.session)
  const heartbeat = asRecord(root.heartbeat)
  const devices = asRecord(root.devices)
  const tools = asRecord(root.tools)
  const cron = asRecord(tools.cron)
  const exec = asRecord(tools.exec)
  const toolFeedback = asRecord(defaults.tool_feedback)

  return {
    workspace: asString(defaults.workspace) || EMPTY_FORM.workspace,
    restrictToWorkspace:
      defaults.restrict_to_workspace === undefined
        ? EMPTY_FORM.restrictToWorkspace
        : asBool(defaults.restrict_to_workspace),
    splitOnMarker:
      defaults.split_on_marker === undefined
        ? EMPTY_FORM.splitOnMarker
        : asBool(defaults.split_on_marker),
    toolFeedbackEnabled:
      toolFeedback.enabled === undefined
        ? EMPTY_FORM.toolFeedbackEnabled
        : asBool(toolFeedback.enabled),
    toolFeedbackMaxArgsLength: asNumberString(
      toolFeedback.max_args_length,
      EMPTY_FORM.toolFeedbackMaxArgsLength,
    ),
    toolFeedbackSeparateMessages:
      toolFeedback.separate_messages === undefined
        ? EMPTY_FORM.toolFeedbackSeparateMessages
        : asBool(toolFeedback.separate_messages),
    execEnabled:
      exec.enabled === undefined
        ? EMPTY_FORM.execEnabled
        : asBool(exec.enabled),
    allowRemote:
      exec.allow_remote === undefined
        ? EMPTY_FORM.allowRemote
        : asBool(exec.allow_remote),
    enableDenyPatterns:
      exec.enable_deny_patterns === undefined
        ? EMPTY_FORM.enableDenyPatterns
        : asBool(exec.enable_deny_patterns),
    customDenyPatternsText: Array.isArray(exec.custom_deny_patterns)
      ? exec.custom_deny_patterns
          .filter((value): value is string => typeof value === "string")
          .join("\n")
      : EMPTY_FORM.customDenyPatternsText,
    customAllowPatternsText: Array.isArray(exec.custom_allow_patterns)
      ? exec.custom_allow_patterns
          .filter((value): value is string => typeof value === "string")
          .join("\n")
      : EMPTY_FORM.customAllowPatternsText,
    execTimeoutSeconds: asNumberString(
      exec.timeout_seconds,
      EMPTY_FORM.execTimeoutSeconds,
    ),
    allowCommand:
      cron.allow_command === undefined
        ? EMPTY_FORM.allowCommand
        : asBool(cron.allow_command),
    cronExecTimeoutMinutes: asNumberString(
      cron.exec_timeout_minutes,
      EMPTY_FORM.cronExecTimeoutMinutes,
    ),
    maxTokens: asNumberString(defaults.max_tokens, EMPTY_FORM.maxTokens),
    contextWindow: asNumberString(
      defaults.context_window,
      EMPTY_FORM.contextWindow,
    ),
    maxToolIterations: asNumberString(
      defaults.max_tool_iterations,
      EMPTY_FORM.maxToolIterations,
    ),
    summarizeMessageThreshold: asNumberString(
      defaults.summarize_message_threshold,
      EMPTY_FORM.summarizeMessageThreshold,
    ),
    summarizeTokenPercent: asNumberString(
      defaults.summarize_token_percent,
      EMPTY_FORM.summarizeTokenPercent,
    ),
    dmScope: asString(session.dm_scope) || EMPTY_FORM.dmScope,
    heartbeatEnabled:
      heartbeat.enabled === undefined
        ? EMPTY_FORM.heartbeatEnabled
        : asBool(heartbeat.enabled),
    heartbeatInterval: asNumberString(
      heartbeat.interval,
      EMPTY_FORM.heartbeatInterval,
    ),
    devicesEnabled:
      devices.enabled === undefined
        ? EMPTY_FORM.devicesEnabled
        : asBool(devices.enabled),
    monitorUSB:
      devices.monitor_usb === undefined
        ? EMPTY_FORM.monitorUSB
        : asBool(devices.monitor_usb),
  }
}

export function parseIntField(
  rawValue: string,
  label: string,
  options: { min?: number; max?: number } = {},
): number {
  const value = Number(rawValue)
  if (!Number.isInteger(value)) {
    throw new Error(`${label} must be an integer.`)
  }
  if (options.min !== undefined && value < options.min) {
    throw new Error(`${label} must be >= ${options.min}.`)
  }
  if (options.max !== undefined && value > options.max) {
    throw new Error(`${label} must be <= ${options.max}.`)
  }
  return value
}

export function parseCIDRText(raw: string): string[] {
  if (!raw.trim()) {
    return []
  }
  return raw
    .split(/[\n,]/)
    .map((v) => v.trim())
    .filter((v) => v.length > 0)
}

export function parseMultilineList(raw: string): string[] {
  if (!raw.trim()) {
    return []
  }
  return raw
    .split("\n")
    .map((value) => value.trim())
    .filter((value) => value.length > 0)
}
