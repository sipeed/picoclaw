import { type Page, type Route } from "@playwright/test"

import type { RepositoryReviewRawFinding } from "../../src/api/repository-reviews"

export type CollectionVisualState = "ready" | "empty" | "error" | "loading"

const fixedNow = "2026-08-25T14:30:00Z"

const querySchemas = {
  accounts: schema([
    field("id", "string"),
    field("provider", "string"),
    field("account", "string"),
    field("status", "enum", [
      "connected",
      "expired",
      "needs_refresh",
      "not_logged_in",
    ]),
    field("auth_method", "string"),
    field("expires_at", "string"),
  ]),
  accountRouters: schema([
    field("name", "string"),
    field("enabled", "boolean", ["true", "false"]),
    field("is_default", "boolean", ["true", "false"]),
    field("status", "enum", [
      "available",
      "disabled",
      "invalid",
      "unconfigured",
      "unreachable",
    ]),
    field("entry", "string"),
    field("accounts", "number"),
    field("blocks", "number"),
  ]),
  eventSources: schema([
    field("name", "string"),
    field("kind", "enum", ["webhook", "channel"]),
    field("enabled", "boolean", ["true", "false"]),
    field("format", "enum", ["standard", "github", "deltachat"]),
    field("status", "enum", [
      "available",
      "disabled",
      "unconfigured",
      "unreachable",
      "invalid",
    ]),
    field("repositories", "number"),
    field("poll_notifications", "boolean", ["true", "false"]),
  ]),
  developmentRepositoryAssignments: schema(
    [
      field("repository", "string"),
      field("configuration", "string", ["default", "automated-review"]),
      field("default_branch", "string", ["main", "release"]),
    ],
    { field: "repository", direction: "ASC" },
  ),
  developmentWorkflowConfigurations: schema(
    [
      field("id", "string", ["default", "automated-review"]),
      field("name", "string"),
      field("is_default", "boolean", ["true", "false"]),
      field("bindings", "number"),
      field("deferred_issues", "enum", ["off", "ask", "automatic"]),
    ],
    { field: "name", direction: "ASC" },
  ),
  developmentWorkspaces: schema(
    [
      field("id", "string"),
      field("intent", "enum", ["implement_feature", "pickup_pr"]),
      field("source", "enum", ["issue", "brief", "pull_request"]),
      field("repository", "string"),
      field("title", "string"),
      field("phase", "enum", [
        "intake",
        "charter",
        "planning",
        "review",
        "triage",
        "implementation",
        "validation",
        "completion_audit",
        "publication",
        "complete",
      ]),
      field("execution_state", "enum", [
        "queued",
        "running",
        "waiting_gate",
        "waiting_user",
        "succeeded",
        "failed",
        "blocked",
        "canceled",
        "stale",
        "unknown",
      ]),
      field("created", "timestamp"),
      field("updated", "timestamp"),
    ],
    { field: "updated", direction: "DESC" },
  ),
  aliases: schema([
    field("name", "string"),
    field("model", "string"),
    field("overrides", "number"),
    field("disabled_accounts", "number"),
  ]),
  routers: schema([
    field("name", "string"),
    field("enabled", "boolean", ["true", "false"]),
    field("blocks", "number"),
    field("rules", "number"),
  ]),
  mcp: schema([
    field("name", "string"),
    field("enabled", "boolean", ["true", "false"]),
    field("deferred", "boolean", ["true", "false"]),
    field("type", "enum", ["stdio", "http", "sse"]),
    field("auth", "enum", ["none", "custom", "bearer", "oauth"]),
  ]),
  agents: schema([
    field("id", "string"),
    field("name", "string"),
    field("workspace", "string"),
    field("account", "string"),
    field("model", "string"),
    field("default", "boolean", ["true", "false"]),
    field("implicit", "boolean", ["true", "false"]),
    field("position", "number"),
  ]),
  evaluations: schema([
    field("id", "string"),
    field("status", "enum", [
      "draft",
      "queued",
      "running",
      "completed",
      "failed",
      "canceled",
    ]),
    field("repository", "string"),
    field("ref", "string"),
    field("models", "number"),
    field("progress", "number"),
    field("version", "number"),
    field("created", "timestamp"),
    field("updated", "timestamp"),
  ]),
  reviews: schema([
    field("id", "string"),
    field("name", "string"),
    field("repository", "string"),
    field("branch", "string"),
    field("status", "enum", [
      "idle",
      "running",
      "stopping",
      "paused",
      "completed",
      "failed",
    ]),
    field("progress", "number"),
    field("reviewed", "number"),
    field("raw_findings", "number"),
    field("findings", "number"),
    field("updated", "timestamp"),
  ]),
  reviewRunFindings: schema(
    [
      field("id", "string"),
      field("repository", "string"),
      field("title", "string"),
      field("path", "string"),
      field("symbol", "string"),
      field("severity", "enum", ["critical", "high", "medium", "low"]),
      field("status", "enum", ["open", "dismissed", "posted"]),
      field("run_status", "enum", [
        "pending",
        "processing",
        "failed",
        "associated_new",
        "associated_existing",
        "needs_review",
      ]),
      field("association", "enum", [
        "unassociated",
        "new",
        "existing",
        "needs_review",
      ]),
      field("contributors", "string"),
      field("sources", "number"),
      field("created", "timestamp"),
      field("updated", "timestamp"),
    ],
    [
      { field: "severity", direction: "DESC" },
      { field: "updated", direction: "DESC" },
    ],
  ),
  reviewRawFindings: schema(
    [
      field("id", "string"),
      field("path", "string"),
      field("severity", "enum", ["critical", "high", "medium", "low"]),
      field("title", "string"),
      field("symbol", "string"),
      field("model", "string"),
      field("reviewer", "string"),
      field("deduplication_state", "enum", [
        "pending",
        "running",
        "failed",
        "completed",
      ]),
      field("disposition", "enum", ["undecided", "new", "duplicate"]),
      field("finding", "string"),
      field("created", "timestamp"),
      field("updated", "timestamp"),
    ],
    [{ field: "created", direction: "DESC" }],
  ),
  reviewFindingsProcessing: schema(
    [
      field("id", "string"),
      field("campaign", "string"),
      field("title", "string"),
      field("path", "string"),
      field("symbol", "string"),
      field("severity", "enum", ["critical", "high", "medium", "low"]),
      field("model", "string"),
      field("reviewer", "string"),
      field("state", "enum", ["pending", "running", "failed", "completed"]),
      field("disposition", "enum", ["undecided", "new", "duplicate"]),
      field("created", "timestamp"),
      field("updated", "timestamp"),
    ],
    [{ field: "updated", direction: "DESC" }],
  ),
  reviewRepositoryFindings: schema(
    [
      field("id", "string"),
      field("repository", "string"),
      field("title", "string"),
      field("path", "string"),
      field("symbol", "string"),
      field("severity", "enum", ["critical", "high", "medium", "low"]),
      field("match", "enum", ["new", "known", "provisional"]),
      field("lifecycle", "enum", [
        "open",
        "resolution_pending",
        "resolved",
        "regressed",
        "dismissed",
      ]),
      field("issue", "enum", ["none", "draft", "open", "closed", "unknown"]),
      field("validation", "enum", [
        "not_requested",
        "pending",
        "running",
        "confirmed",
        "not_fixed",
        "inconclusive",
        "failed",
      ]),
      field("occurrences", "number"),
      field("commits", "number"),
      field("created", "timestamp"),
      field("updated", "timestamp"),
    ],
    [
      { field: "severity", direction: "DESC" },
      { field: "updated", direction: "DESC" },
    ],
  ),
  reviewIssues: schema(
    [
      field("id", "string"),
      field("repository", "string"),
      field("title", "string"),
      field("generation", "string"),
      field("state", "enum", [
        "generating",
        "failed",
        "editing",
        "publishing",
        "posted",
        "unknown",
      ]),
      field("origin", "enum", [
        "ai_generated",
        "linked",
        "discovered",
        "legacy",
      ]),
      field("canonical", "boolean", ["true", "false"]),
      field("publishable", "boolean", ["true", "false"]),
      field("findings", "number"),
      field("created", "timestamp"),
      field("updated", "timestamp"),
    ],
    { field: "updated", direction: "DESC" },
  ),
  reviewProfiles: schema([
    field("id", "string"),
    field("name", "string"),
    field("account", "string"),
    field("reviewer", "string"),
    field("issue_writer", "string"),
    field("parallel", "number"),
    field("updated", "timestamp"),
  ]),
  skills: schema([
    field("name", "string"),
    field("source", "enum", ["workspace", "global", "builtin"]),
    field("origin", "enum", ["builtin", "manual", "third_party"]),
    field("registry", "string"),
    field("version", "string"),
    field("installed_at", "number"),
  ]),
  tools: schema([
    field("name", "string"),
    field("category", "enum", [
      "agents",
      "automation",
      "communication",
      "discovery",
      "filesystem",
      "hardware",
      "skills",
      "web",
    ]),
    field("status", "enum", ["enabled", "disabled", "blocked"]),
    field("reason", "string"),
    field("config_key", "string"),
  ]),
  workflowDefinitions: schema([
    field("ref", "string"),
    field("name", "string"),
    field("status", "enum", [
      "valid",
      "invalid",
      "pending_revalidation",
      "needs_review",
    ]),
    field("trigger", "enum", [
      "manual",
      "schedule",
      "channel_message",
      "command",
      "runtime_event",
      "event",
      "workflow_call",
      "multiple",
      "none",
    ]),
    field("inputs", "number"),
    field("secrets", "number"),
  ]),
  workflowRuns: schema([
    field("id", "string"),
    field("workflow", "string"),
    field("status", "enum", [
      "running",
      "waiting",
      "succeeded",
      "failed",
      "canceled",
      "skipped",
    ]),
    field("session", "string"),
    field("origin", "enum", [
      "manual",
      "external_event",
      "external_event_draft_test",
    ]),
    field("created", "timestamp"),
    field("updated", "timestamp"),
    field("completed", "timestamp"),
  ]),
  gitWorkspaces: schema(
    [
      field("id", "string"),
      field("repository", "string"),
      field("branch", "string"),
      field("status", "enum", ["available", "locked", "dropped"]),
      field("locked", "boolean", ["true", "false"]),
      field("dirty", "boolean", ["true", "false"]),
      field("size", "number"),
      field("ignored", "number"),
      field("updated", "timestamp"),
    ],
    { field: "updated", direction: "DESC" },
  ),
  gitWorkspaceHistory: schema(
    [
      field("action", "string"),
      field("workspace", "string"),
      field("repository", "string"),
      field("agent", "string"),
      field("time", "timestamp"),
    ],
    { field: "time", direction: "DESC" },
  ),
}

export const accountVisualIDs = {
  primary: "YWNjb3VudABvcGVuYWk6cHJpbWFyeQ",
  review: "YWNjb3VudABhbnRocm9waWM6cmV2aWV3",
  copilot: "YWNjb3VudABnaXRodWItY29waWxvdDp3b3Jr",
  router: "balanced-router",
} as const

const accounts = [
  {
    id: accountVisualIDs.review,
    provider: "anthropic",
    account: "anthropic:review",
    status: "needs_refresh",
    auth_method: "token",
    expires_at: "",
  },
  {
    id: accountVisualIDs.copilot,
    provider: "github-copilot",
    account: "github-copilot:work",
    status: "connected",
    auth_method: "token",
    expires_at: "2026-09-02T12:00:00Z",
  },
  {
    id: accountVisualIDs.primary,
    provider: "openai",
    account: "openai:primary",
    status: "connected",
    auth_method: "oauth",
    expires_at: "2026-09-01T16:00:00Z",
  },
] as const

const accountRouters = [
  {
    id: accountVisualIDs.router,
    name: "balanced-router",
    enabled: true,
    is_default: true,
    status: "available",
    entry: "pool",
    accounts: ["credential:openai:primary", "credential:anthropic:review"],
    refresh_interval_seconds: 60,
    blocks: [
      {
        id: "pool",
        type: "load_balance",
        accounts: ["credential:openai:primary", "credential:anthropic:review"],
        strategy: "closest_limit",
        fallback: "review",
      },
      {
        id: "review",
        type: "account",
        account: "credential:anthropic:review",
      },
    ],
  },
  {
    id: "batch-router",
    name: "batch-router",
    enabled: true,
    is_default: false,
    status: "unconfigured",
    entry: "batch",
    accounts: ["credential:github-copilot:work"],
    refresh_interval_seconds: 120,
    blocks: [
      {
        id: "batch",
        type: "account",
        account: "credential:github-copilot:work",
      },
    ],
  },
  {
    id: "offline-router",
    name: "offline-router",
    enabled: false,
    is_default: false,
    status: "disabled",
    entry: "offline",
    accounts: ["credential:openai:primary"],
    refresh_interval_seconds: 60,
    blocks: [
      {
        id: "offline",
        type: "account",
        account: "credential:openai:primary",
      },
    ],
  },
] as const

const accountRouterSummaries = accountRouters.map((router) => ({
  id: router.id,
  name: router.name,
  enabled: router.enabled,
  is_default: router.is_default,
  status: router.status,
  entry: router.entry,
  accounts: router.accounts.length,
  blocks: router.blocks.length,
}))

export const eventSourceVisualIDs = {
  github: "eD_ny6SHSWjo-BISQjbmabgVi1UiA9LivNmhCwLoVCY",
  standard: "AtpwqLgv6OwA1riKGYdyok1K5JqqZT0X9HnhIzQlmis",
  channel: "MLlxF-61ul3LWbQ4FYizmxMV4hRFb0pm7lh1O4uJ7gs",
} as const

const eventSources = [
  {
    id: eventSourceVisualIDs.standard,
    name: "build-system",
    kind: "webhook",
    enabled: false,
    format: "standard",
    status: "disabled",
    repositories: 0,
    poll_notifications: false,
  },
  {
    id: eventSourceVisualIDs.github,
    name: "github-primary",
    kind: "webhook",
    enabled: true,
    format: "github",
    status: "available",
    repositories: 3,
    poll_notifications: true,
  },
  {
    id: eventSourceVisualIDs.channel,
    name: "primary-inbox",
    kind: "channel",
    enabled: true,
    format: "deltachat",
    status: "available",
    repositories: 0,
    poll_notifications: false,
  },
] as const

const eventSourceDetails = {
  [eventSourceVisualIDs.standard]: {
    ...eventSources[0],
    repositories: [],
    target_user: "",
    secret_configured: true,
    endpoint: "/webhooks/events/build-system",
  },
  [eventSourceVisualIDs.github]: {
    ...eventSources[1],
    repositories: ["openai/openai", "sipeed/picoclaw", "octo/launcher"],
    target_user: "octocat",
    secret_configured: true,
    endpoint: "/webhooks/events/github-primary",
  },
  [eventSourceVisualIDs.channel]: {
    ...eventSources[2],
    source: "email",
    mode: "mirror",
    allow_unverified_email: false,
    channel_enabled: true,
    channel_type: "deltachat",
  },
} as const

const eventSourceSettings = {
  enabled: true,
  database_path: "eventing/events.db",
  retention_days: 30,
  max_payload_bytes: 1_048_576,
  redact_fields: ["tenant_secret", "deployment_token"],
}

const eligibleEventChannelAdapters = [
  {
    name: "secondary-inbox",
    channel_type: "deltachat",
    channel_enabled: true,
  },
] as const

export const developmentAdminVisualIDs = {
  picoclaw: "Vkv3iKEbkMfo1PVdQHYM6x0QjgzAWBog3Bth79rGdVg",
  launcher: "VogbVnIeaaq1a1DEqicX1PdL-EoLT9LF5Z_jDIJutOA",
  automation: "xL9ecJ15Rm1erpekUuu5T5pMRxvmYK-2Lc9U1M4sdWk",
} as const

const developmentRepositoryAssignments = [
  {
    id: developmentAdminVisualIDs.picoclaw,
    repository: "sipeed/picoclaw",
    configuration: "automated-review",
    default_branch: "main",
  },
  {
    id: developmentAdminVisualIDs.launcher,
    repository: "octo/launcher",
    configuration: "default",
    default_branch: "main",
  },
  {
    id: developmentAdminVisualIDs.automation,
    repository: "automation/rules",
    configuration: "automated-review",
    default_branch: "release",
  },
] as const

const developmentWorkflowConfigurations = [
  {
    id: "automated-review",
    name: "Automated review",
    is_default: false,
    bindings: 8,
    deferred_issues: "automatic",
  },
  {
    id: "default",
    name: "Default",
    is_default: true,
    bindings: 0,
    deferred_issues: "ask",
  },
  {
    id: "strict-release",
    name: "Strict release",
    is_default: false,
    bindings: 3,
    deferred_issues: "off",
  },
] as const

const developmentAdminEffects = {
  gateway_effect: "applied",
  deferred_policy_effect: "applied",
} as const

export const developmentWorkspaceVisualIDs = {
  review: `devw_${"1".repeat(32)}`,
  feature: `devw_${"2".repeat(32)}`,
  brief: `devw_${"3".repeat(32)}`,
} as const

const developmentWorkspaces = [
  {
    id: developmentWorkspaceVisualIDs.review,
    intent: "pickup_pr",
    source: "pull_request",
    repository: "octo/launcher",
    title: "Pull request #418",
    phase: "completion_audit",
    execution_state: "waiting_user",
    created: "2026-08-24T10:00:00Z",
    updated: "2026-08-25T14:24:00Z",
  },
  {
    id: developmentWorkspaceVisualIDs.feature,
    intent: "implement_feature",
    source: "issue",
    repository: "sipeed/picoclaw",
    title: "Issue #932",
    phase: "implementation",
    execution_state: "running",
    created: "2026-08-23T09:15:00Z",
    updated: "2026-08-25T13:48:00Z",
  },
  {
    id: developmentWorkspaceVisualIDs.brief,
    intent: "implement_feature",
    source: "brief",
    repository: "automation/rules",
    title: "Feature brief",
    phase: "complete",
    execution_state: "succeeded",
    created: "2026-08-20T16:30:00Z",
    updated: "2026-08-24T18:45:00Z",
  },
] as const

export const gitWorkspaceVisualIDs = {
  primary: "gw-111111111111",
  locked: "gw-222222222222",
  cache: "gw-333333333333-2",
} as const

const gitWorkspaces = [
  {
    id: gitWorkspaceVisualIDs.primary,
    repository: "github.com/sipeed/picoclaw.git",
    branch: "main",
    status: "available",
    locked: false,
    dirty: false,
    size: 18_874_368,
    ignored: 524_288,
    updated: "2026-08-25T14:24:00Z",
  },
  {
    id: gitWorkspaceVisualIDs.locked,
    repository: "github.com/octo/launcher.git",
    branch: "feature/collection-routes",
    status: "locked",
    locked: true,
    dirty: true,
    size: 8_388_608,
    ignored: 131_072,
    updated: "2026-08-25T14:12:00Z",
  },
  {
    id: gitWorkspaceVisualIDs.cache,
    repository: "git.example.test/automation/rules.git",
    branch: "release",
    status: "available",
    locked: false,
    dirty: false,
    size: 3_145_728,
    ignored: 1_048_576,
    updated: "2026-08-24T18:45:00Z",
  },
] as const

const gitWorkspaceDetails = Object.fromEntries(
  gitWorkspaces.map((workspace) => [
    workspace.id,
    {
      ...workspace,
      repository_id: workspace.id.replace(/-[0-9]+$/, ""),
      remote_url: `https://${workspace.repository}`,
      path: `/srv/picoclaw/git-workspaces/checkouts/${workspace.id}`,
      ref: workspace.branch,
      created: "2026-08-20T10:00:00Z",
      last_work: workspace.updated,
      ...(workspace.locked
        ? {
            locked_by: {
              agent_id: "reviewer",
              locked_at: "2026-08-25T14:00:00Z",
              heartbeat_at: "2026-08-25T14:12:00Z",
            },
          }
        : {}),
    },
  ]),
) as Record<string, Record<string, unknown>>

const gitWorkspaceHistory = [
  {
    id: "aaaaaaaaaaaa",
    action: "cleaned_ignored",
    workspace: gitWorkspaceVisualIDs.primary,
    repository: "github.com/sipeed/picoclaw.git",
    agent: "main",
    time: "2026-08-25T14:26:00Z",
  },
  {
    id: "bbbbbbbbbbbb",
    action: "acquired",
    workspace: gitWorkspaceVisualIDs.locked,
    repository: "github.com/octo/launcher.git",
    agent: "reviewer",
    time: "2026-08-25T14:00:00Z",
  },
  {
    id: "cccccccccccc",
    action: "released",
    workspace: gitWorkspaceVisualIDs.cache,
    repository: "git.example.test/automation/rules.git",
    agent: "main",
    time: "2026-08-24T18:46:00Z",
  },
] as const

const gitWorkspaceSettings = {
  configured: {
    max_total_size_bytes: 0,
    ignored_cleanup_delay_seconds: 0,
    drop_delay_seconds: 0,
  },
  effective: {
    max_total_size_bytes: 21_474_836_480,
    ignored_cleanup_delay_seconds: 86_400,
    drop_delay_seconds: 2_592_000,
  },
  config_revision: "cfg-git-workspaces-visual-1",
}

export const workflowVisualIDs = {
  summarize: "tpC6ep8WMy-TZuE8ZnYqpGVom7Yf4X3fhF01wP528JU",
  triage: "16gg1Z-92X47AwRe90IKZvdf1a6cNTojnoCdW4qXQQw",
  review: "W41H3aj_ymqpg1aP8Vs6TJXUddSpYgjW7VRRcTHj_-I",
  run: "wr_visual_running",
} as const

const workflowDefinitions = [
  {
    id: workflowVisualIDs.review,
    ref: "workflows/code-review.yml",
    name: "Repository code review",
    status: "needs_review",
    trigger: "workflow_call",
    inputs: 3,
    secrets: 1,
  },
  {
    id: workflowVisualIDs.triage,
    ref: "workflows/github-issue-triage.yml",
    name: "GitHub issue triage",
    status: "valid",
    trigger: "event",
    inputs: 0,
    secrets: 0,
  },
  {
    id: workflowVisualIDs.summarize,
    ref: "workflows/summarize-text.yml",
    name: "Summarize text",
    status: "valid",
    trigger: "multiple",
    inputs: 2,
    secrets: 0,
  },
] as const

const workflowRuns = [
  {
    id: workflowVisualIDs.run,
    workflow_id: workflowVisualIDs.summarize,
    workflow_ref: "workflows/summarize-text.yml",
    status: "running",
    session: "workflow:manual:visual",
    created_at: "2026-08-25T14:20:00Z",
    updated_at: "2026-08-25T14:28:00Z",
  },
  {
    id: "wr_visual_event",
    workflow_id: workflowVisualIDs.triage,
    workflow_ref: "workflows/github-issue-triage.yml",
    status: "succeeded",
    session: "workflow:event:visual",
    origin: {
      kind: "external_event",
      event_id: `ev_${"1".repeat(32)}`,
      dispatch_id: `dsp_${"2".repeat(32)}`,
      root_run_id: "wr_visual_event",
    },
    created_at: "2026-08-25T13:55:00Z",
    updated_at: "2026-08-25T14:00:00Z",
    completed_at: "2026-08-25T14:00:00Z",
  },
  {
    id: "wr_visual_failed",
    workflow_id: workflowVisualIDs.review,
    workflow_ref: "workflows/code-review.yml",
    status: "failed",
    created_at: "2026-08-24T18:15:00Z",
    updated_at: "2026-08-24T18:16:00Z",
    completed_at: "2026-08-24T18:16:00Z",
  },
] as const

export const skillToolVisualIDs = {
  skill: "c2tpbGwAcmV2aWV3LWhlbHBlcg",
  tool: "dG9vbAB3ZWJfc2VhcmNo",
} as const

const skills = [
  {
    id: "c2tpbGwAY29kZS1yZXZpZXc",
    name: "code-review",
    path: "/usr/share/picoclaw/skills/code-review",
    source: "builtin",
    description: "Inspect changes for correctness and maintainability.",
    origin: "builtin",
    origin_kind: "builtin",
    version: "bundled",
    installed_version: "bundled",
    removable: false,
  },
  {
    id: "c2tpbGwAZGVwbG95LWNoZWNrbGlzdA",
    name: "deploy-checklist",
    path: "/opt/picoclaw/skills/deploy-checklist",
    source: "global",
    description: "Prepare a safe deployment checklist.",
    origin: "manual",
    origin_kind: "manual",
    version: "1.7.0",
    installed_version: "1.7.0",
    installed_at: 1_776_864_600_000,
    removable: false,
  },
  {
    id: skillToolVisualIDs.skill,
    name: "review-helper",
    path: "/workspace/skills/review-helper",
    source: "workspace",
    description: "Review code changes with repository-aware checks.",
    origin: "third_party",
    origin_kind: "third_party",
    registry: "clawhub",
    registry_name: "clawhub",
    registry_url: "https://clawhub.example.test/skills/review-helper",
    version: "2.4.1",
    installed_version: "2.4.1",
    installed_at: 1_777_296_600_000,
    removable: true,
  },
] as const

const tools = [
  {
    id: "dG9vbABzZW5kX3R0cw",
    name: "send_tts",
    description: "Generate and deliver speech to the active channel.",
    category: "communication",
    config_key: "tools.send_tts",
    status: "disabled",
    reason: "Disabled in configuration.",
    reason_code: "configured_disabled",
  },
  {
    id: "dG9vbABpbnN0YWxsX3NraWxs",
    name: "install_skill",
    description: "Install a skill from a configured registry.",
    category: "skills",
    config_key: "tools.install_skill",
    status: "blocked",
    reason: "No writable workspace skill directory is configured.",
    reason_code: "dependency_unavailable",
  },
  {
    id: skillToolVisualIDs.tool,
    name: "web_search",
    description: "Search configured web providers for current information.",
    category: "web",
    config_key: "tools.web_search",
    status: "enabled",
  },
] as const

const oauthProviders = [
  {
    provider: "openai",
    credential_id: "openai",
    display_name: "OpenAI",
    methods: ["browser", "device_code", "token"],
    logged_in: true,
    status: "connected",
    credentials: [
      {
        provider: "openai",
        credential_id: "openai:primary",
        display_name: "OpenAI",
        methods: ["browser", "device_code", "token"],
        logged_in: true,
        status: "connected",
        auth_method: "oauth",
        account_id: "acct-primary",
      },
    ],
  },
  {
    provider: "anthropic",
    credential_id: "anthropic",
    display_name: "Anthropic",
    methods: ["token"],
    logged_in: true,
    status: "needs_refresh",
    credentials: [
      {
        provider: "anthropic",
        credential_id: "anthropic:review",
        display_name: "Anthropic",
        methods: ["token"],
        logged_in: true,
        status: "needs_refresh",
        auth_method: "token",
      },
    ],
  },
] as const

const aliases = [
  {
    name: "code",
    model: "gpt-5.6-codex",
    override_count: 1,
    disabled_account_count: 0,
    account_overrides: { "openai-primary": "gpt-5.6-codex" },
    disabled_accounts: [],
  },
  {
    name: "fast",
    model: "gpt-5.6-mini",
    override_count: 0,
    disabled_account_count: 1,
    account_overrides: {},
    disabled_accounts: ["offline-lab"],
  },
  {
    name: "review",
    model: "claude-sonnet-4.6",
    override_count: 0,
    disabled_account_count: 0,
    account_overrides: {},
    disabled_accounts: [],
  },
]

const routers = [
  {
    name: "task-router",
    enabled: true,
    entry: "entry",
    block_count: 3,
    rule_count: 1,
    blocks: [
      {
        id: "entry",
        type: "rules",
        rules: [{ match: "has_code", target: "code" }],
        fallback: "fast",
      },
      { id: "code", type: "model", model: "code" },
      { id: "fast", type: "model", model: "fast" },
    ],
  },
  {
    name: "media-router",
    enabled: false,
    entry: "entry",
    block_count: 1,
    rule_count: 1,
    blocks: [
      {
        id: "entry",
        type: "rules",
        rules: [{ match: "has_media", target: "review" }],
        fallback: "fast",
      },
    ],
  },
]

const mcpServers = [
  {
    name: "github",
    enabled: true,
    deferred: false,
    type: "http",
    url: "https://mcp.example.test/github",
    command: "",
    args: [],
    env_file: "",
    env_keys: [],
    header_keys: ["X-Workspace"],
    auth: { type: "oauth", configured: true, expired: false },
  },
  {
    name: "local-files",
    enabled: false,
    deferred: true,
    type: "stdio",
    url: "",
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"],
    env_file: "",
    env_keys: ["FILESYSTEM_TOKEN"],
    header_keys: [],
    auth: { type: "none", configured: false, expired: false },
  },
]

const mcpServerSummaries = mcpServers.map((server) => ({
  name: server.name,
  enabled: server.enabled,
  deferred: server.deferred,
  type: server.type,
  address:
    server.type === "stdio"
      ? [server.command, ...server.args].filter(Boolean).join(" ")
      : server.url,
  environment_key_count: server.env_keys.length,
  header_key_count: server.header_keys.length,
  auth: server.auth,
}))

const agents = [
  {
    id: "main",
    name: "Main agent",
    workspace: "/workspace/main",
    account_ref: "openai-primary",
    model: { primary: "code", fallbacks: ["fast"] },
    skills: null,
    subagents: null,
    is_default: true,
    default_configured: true,
    implicit: false,
  },
  {
    id: "reviewer",
    name: "Repository reviewer",
    workspace: "/workspace/reviewer",
    account_ref: "anthropic-review",
    model: { primary: "review", fallbacks: [] },
    skills: ["repository-review"],
    subagents: { allow: ["main"] },
    is_default: false,
    default_configured: true,
    implicit: false,
  },
]

const progress = {
  stage: "draft",
  languages: {},
  total_files: 0,
  selected_files: 0,
  completed_files: 0,
  total_tasks: 0,
  completed_tasks: 0,
  percent: 0,
  updated_at: fixedNow,
}

const usage = {
  requests: 0,
  input_tokens: 0,
  cached_input_tokens: 0,
  output_tokens: 0,
  reasoning_tokens: 0,
  duration_millis: 0,
}

const evaluation = {
  schema_version: 1,
  id: "rme_0123456789abcdef0123456789abcdef",
  version: 3,
  status: "draft",
  repository: "octo/picoclaw",
  ref: "main",
  candidate_models: ["code", "fast", "review"],
  selector_model_alias: "code",
  judge_model_alias: "review",
  focus: {},
  default_files_per_language: 20,
  files_per_language: {},
  profile: {
    id: "balanced",
    version: 2,
    name: "Balanced review",
    reviewer_model: "review",
    account_ref: "anthropic-review",
    review_focus: "Correctness and safety",
    focus: {},
    max_files_per_batch: 20,
    max_content_bytes_per_batch: 100000,
    max_parallel_children: 2,
  },
  progress,
  usage,
  model_stats: {},
  comparisons: [],
  warnings: [],
  run_ids: [],
  created_at: "2026-08-25T13:00:00Z",
  updated_at: fixedNow,
}

export const repositoryReviewVisualIDs = {
  automation: "rra_visual",
  finding: "rdf_visual_1",
  secondFinding: "rdf_visual_2",
  attentionFinding: "rdf_visual_attention",
  rawFinding: "rrw_visual_1",
  secondRawFinding: "rrw_visual_2",
  thirdRawFinding: "rrw_visual_3",
  processingPending: "rrw_visual_processing_pending",
  processingRunning: "rrw_visual_processing_running",
  processingFailed: "rrw_visual_processing_failed",
  processingCompleted: "rrw_visual_processing_completed",
  processingOldCampaign: "rrw_visual_processing_old_campaign",
  repositoryFinding: "rrf_visual_1",
  normalRepositoryFinding: "rrf_visual_normal",
  provisionalRepositoryFinding: "rrf_visual_provisional",
  candidateRepositoryFinding: "rrf_visual_candidate",
  combinedAttentionRepositoryFinding: "rrf_visual_combined_attention",
  conflictRepositoryFinding: "rrf_visual_conflict",
  failedResolutionRepositoryFinding: "rrf_visual_failed_resolution",
  issue: "rrid_visual_1",
  failedIssue: "rrid_visual_2",
  generation: "rig_visual",
} as const

const repositoryReviewCommit = "a".repeat(40)

const repositoryReviewAutomation = {
  id: repositoryReviewVisualIDs.automation,
  version: 9,
  profile_id: "rrpf_visual",
  profile_version: 4,
  branch: "main",
  name: "Correctness review",
  repository: "octo/picoclaw",
  ref: "main",
  target: "all",
  account_ref: "openai-primary",
  effective_account_ref: "openai-primary",
  review_focus: "Find concrete correctness and reliability defects.",
  scope_policy: {
    code_types: ["hotpath-code", "code", "test"],
    include_folders: ["pkg", "web"],
    exclude_folders: ["vendor"],
    free_text: "Prioritize persistent state transitions.",
  },
  reviewer_models: ["review"],
  issue_writer_model: "issue-writer",
  compare_models: false,
  force: false,
  max_files_per_run: 24,
  max_content_bytes: 524288,
  max_parallel_children: 4,
  auto_continue: true,
  model_prices: {},
  budget: { guard_expression: "tokens.total < 250000" },
  status: "failed",
  pause_detail:
    "The reviewer provider stopped after repeated safe failures. Durable checkpoints remain available.",
  run_ids: ["rrun_visual_1", "rrun_visual_2"],
  usage: {
    prompt_tokens: 48240,
    completion_tokens: 7840,
    total_tokens: 56080,
    cached_tokens: 12120,
  },
  estimated_cost_usd: 1.27,
  progress: {
    stage: "failed",
    completed_batches: 5,
    total_batches: 5,
    reviewed_files: 39,
    remaining_files: 0,
    unsupported_files: 1,
    raw_findings: 3,
    deduplicated_findings: 2,
    findings: 2,
    scope_frozen: true,
  },
  model_stats: [],
  account_limits: [],
  scope_plan: {
    commit_sha: repositoryReviewCommit,
    policy_hash: "sha256:visual-policy",
    hash: "sha256:visual-scope",
    summary: "40 source and test files pinned at the selected commit.",
    warnings: ["One generated fixture file was unsupported."],
    counts: {
      total_files: 48,
      code_type_files: 46,
      include_files: 43,
      excluded_files: 3,
      selected_files: 40,
    },
  },
  resolved_commit_sha: repositoryReviewCommit,
  started_at: "2026-08-25T13:00:00Z",
  completed_at: "2026-08-25T14:20:00Z",
  created_at: "2026-08-24T12:00:00Z",
  updated_at: "2026-08-25T14:20:00Z",
}

const repositoryReviewProfile = {
  id: "rrpf_visual",
  version: 4,
  name: "Correctness review",
  account_ref: "openai-primary",
  reviewer_model: "review",
  issue_writer_model: "issue-writer",
  review_focus: "Find concrete correctness and reliability defects.",
  issue_prompt: "Present the confirmed diagnosis with evidence and provenance.",
  scope_policy: {
    code_types: ["hotpath-code", "code", "test"],
    include_folders: ["pkg", "web"],
    exclude_folders: ["vendor"],
    free_text: "Prioritize persistent state transitions.",
  },
  force: false,
  auto_continue: true,
  max_files_per_run: 24,
  max_content_bytes: 524288,
  max_parallel_children: 4,
  budget: { guard_expression: "" },
  created_at: "2026-08-20T12:00:00Z",
  updated_at: fixedNow,
}

const repositoryReviewSummary = {
  schema_version: 1,
  id: "rrs_visual",
  repository: "octo/picoclaw",
  version: 14,
  review_version: 5,
  last_commit_sha: repositoryReviewCommit,
  finding_count: 2,
  open_finding_count: 2,
  issue_draft_count: 2,
  unsupported_count: 1,
  reviewed_file_count: 40,
  excluded_file_count: 3,
  updated_at: "2026-08-25T14:20:00Z",
}

const repositoryReviewFindings = [
  {
    id: repositoryReviewVisualIDs.finding,
    fingerprint: "sha256:visual-lost-update",
    repository: "octo/picoclaw",
    commit_sha: repositoryReviewCommit,
    file: {
      path: "pkg/repoaudit/store.go",
      blob_sha: "b".repeat(40),
      size_bytes: 16842,
      category: "hotpath-code",
    },
    line: 418,
    severity: "high",
    title: "Concurrent checkpoint writes can lose a finding",
    symbol: "Store.SaveFinding",
    message: "The read-modify-write sequence has no version fence.",
    evidence:
      "Two callers load the same ledger version and each replaces the findings slice; the later atomic rename discards the earlier checkpoint.",
    impact:
      "A validated repository finding can disappear from the findings view.",
    validation: {
      status: "confirmed",
      summary: "Traced both writers through the atomic rename path.",
      checks: ["Compared caller snapshots", "Verified no CAS guard"],
    },
    context_ids: ["rrctx_visual_1"],
    models: ["review", "code"],
    observation_count: 2,
    observations: [
      {
        context_id: "rrctx_visual_1",
        model: "review",
        reviewer: "review-child-2",
        severity: "high",
        title: "Concurrent checkpoint writes can lose a finding",
        symbol: "Store.SaveFinding",
        line: 418,
        evidence: "Both writers persist snapshots derived from version 13.",
        impact: "One validated checkpoint is overwritten.",
        validation: {
          status: "confirmed",
          summary: "Interleaving reproduced from both call paths.",
        },
      },
    ],
    status: "open",
    issue_draft_id: repositoryReviewVisualIDs.issue,
    repository_finding_id: repositoryReviewVisualIDs.repositoryFinding,
    repository_match_state: "known",
    run_finding_status: "associated_existing",
    version: 3,
    created_at: "2026-08-25T14:05:00Z",
    updated_at: "2026-08-25T14:10:00Z",
    raw_source_total: 2,
  },
  {
    id: repositoryReviewVisualIDs.secondFinding,
    fingerprint: "sha256:visual-cancel-retry",
    repository: "octo/picoclaw",
    commit_sha: repositoryReviewCommit,
    file: {
      path: "pkg/repoaudit/control.go",
      blob_sha: "c".repeat(40),
      size_bytes: 9240,
      category: "code",
    },
    line: 206,
    severity: "medium",
    title: "Canceled batches are retried without backoff",
    symbol: "Controller.continueRun",
    message: "Cancellation is classified as a transient review error.",
    evidence: "context.Canceled reaches the immediate continuation branch.",
    impact: "Shutdown can produce a tight retry loop and duplicate work.",
    validation: {
      status: "confirmed",
      summary: "Followed cancellation through continuation scheduling.",
      checks: ["Verified the retry delay remains zero"],
    },
    context_ids: ["rrctx_visual_2"],
    models: ["review"],
    observation_count: 1,
    status: "open",
    issue_draft_id: repositoryReviewVisualIDs.failedIssue,
    repository_finding_id: "rrf_visual_2",
    repository_match_state: "known",
    run_finding_status: "associated_existing",
    version: 2,
    created_at: "2026-08-25T14:12:00Z",
    updated_at: "2026-08-25T14:15:00Z",
    raw_source_total: 1,
  },
]

const repositoryReviewAttentionFinding = {
  ...repositoryReviewFindings[1],
  id: repositoryReviewVisualIDs.attentionFinding,
  fingerprint: "sha256:visual-provisional-cache-race",
  file: {
    ...repositoryReviewFindings[1].file,
    path: "pkg/cache/refresh_coordinator.go",
    blob_sha: "d".repeat(40),
    size_bytes: 11720,
  },
  line: 287,
  severity: "high",
  title: "Concurrent refreshes can publish an expired cache generation",
  symbol: "RefreshCoordinator.publish",
  message: "The generation comparison happens before the final cache swap.",
  evidence:
    "A delayed refresh resumes after a newer generation commits and replaces the current cache pointer with its expired snapshot.",
  impact:
    "Readers can observe credentials and routing metadata from an expired cache generation.",
  validation: {
    status: "confirmed",
    summary: "Traced the stale generation through the unlocked publish path.",
    checks: ["Compared generation fences", "Verified the final swap is stale"],
  },
  context_ids: ["rrctx_visual_attention"],
  models: ["review"],
  observation_count: 1,
  issue_draft_id: undefined,
  repository_finding_id: repositoryReviewVisualIDs.provisionalRepositoryFinding,
  repository_match_state: "provisional",
  run_finding_status: "needs_review",
  version: 1,
  created_at: "2026-08-25T14:16:00Z",
  updated_at: "2026-08-25T14:18:00Z",
  raw_source_total: 1,
}

const repositoryReviewDetailFindings = [
  ...repositoryReviewFindings,
  repositoryReviewAttentionFinding,
]

const baseRepositoryFindings = repositoryReviewFindings.map(
  (finding, index) => ({
    id:
      index === 0
        ? repositoryReviewVisualIDs.repositoryFinding
        : `rrf_visual_${index + 1}`,
    repository: finding.repository,
    canonical_title: finding.title,
    canonical_severity: finding.severity,
    review_finding_ids: [finding.id],
    occurrence_count: 1,
    found_commits: [finding.commit_sha],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: finding.id,
        commit_sha: finding.commit_sha,
        path: finding.file.path,
        symbol: finding.symbol,
        observed_at: finding.created_at,
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: {
      state: "draft",
      origin: "ai_generated",
      title: finding.title,
      snapshot_at: finding.updated_at,
      conflict: false,
    },
    validation_state: "not_requested",
    possible_duplicates: [],
    version: 1,
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }),
)

const repositoryFindings = [
  ...baseRepositoryFindings,
  {
    id: repositoryReviewVisualIDs.normalRepositoryFinding,
    repository: "octo/picoclaw",
    canonical_title: "Request retries can repeat a committed ledger update",
    canonical_severity: "medium",
    review_finding_ids: [
      "rdf_visual_normal_1",
      "rdf_visual_normal_2",
      "rdf_visual_normal_3",
    ],
    occurrence_count: 3,
    found_commits: ["1".repeat(40), "2".repeat(40)],
    found_commit_count: 2,
    path_symbol_history: [
      {
        review_finding_id: "rdf_visual_normal_1",
        commit_sha: "1".repeat(40),
        path: "pkg/repoaudit/ledger_writer.go",
        symbol: "LedgerWriter.Commit",
        observed_at: "2026-08-23T09:00:00Z",
      },
      {
        review_finding_id: "rdf_visual_normal_3",
        commit_sha: "2".repeat(40),
        path: "pkg/repoaudit/transaction_writer.go",
        symbol: "TransactionWriter.Commit",
        observed_at: "2026-08-25T13:40:00Z",
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: {
      state: "none",
      snapshot_at: undefined,
      conflict: false,
    },
    validation_state: "not_requested",
    possible_duplicates: [],
    version: 4,
    created_at: "2026-08-23T09:00:00Z",
    updated_at: "2026-08-25T13:40:00Z",
  },
  {
    id: repositoryReviewVisualIDs.provisionalRepositoryFinding,
    repository: "octo/picoclaw",
    canonical_title: repositoryReviewAttentionFinding.title,
    canonical_severity: repositoryReviewAttentionFinding.severity,
    review_finding_ids: [repositoryReviewAttentionFinding.id],
    occurrence_count: 1,
    found_commits: [repositoryReviewAttentionFinding.commit_sha],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: repositoryReviewAttentionFinding.id,
        commit_sha: repositoryReviewAttentionFinding.commit_sha,
        path: repositoryReviewAttentionFinding.file.path,
        symbol: repositoryReviewAttentionFinding.symbol,
        observed_at: repositoryReviewAttentionFinding.created_at,
      },
    ],
    match_state: "provisional",
    lifecycle: "open",
    issue: {
      state: "none",
      snapshot_at: undefined,
      conflict: false,
    },
    validation_state: "not_requested",
    possible_duplicates: [
      {
        candidate_id: repositoryReviewVisualIDs.candidateRepositoryFinding,
        relation: "uncertain",
        confidence: 0.94,
        explanation:
          "Both diagnoses describe an older cache generation replacing a newer committed generation after asynchronous refresh work resumes.",
        matching_anchors: [
          "generation fence",
          "final cache pointer swap",
          "delayed refresh completion",
        ],
        conflicting_anchors: ["different refresh entry point"],
        created_at: "2026-08-25T14:18:00Z",
      },
    ],
    version: 2,
    created_at: repositoryReviewAttentionFinding.created_at,
    updated_at: repositoryReviewAttentionFinding.updated_at,
  },
  {
    id: repositoryReviewVisualIDs.candidateRepositoryFinding,
    repository: "octo/picoclaw",
    canonical_title: "Delayed cache refresh overwrites a newer generation",
    canonical_severity: "high",
    review_finding_ids: [
      "rdf_visual_candidate_1",
      "rdf_visual_candidate_2",
      "rdf_visual_candidate_3",
      "rdf_visual_candidate_4",
    ],
    occurrence_count: 4,
    found_commits: ["7".repeat(40), "8".repeat(40), "9".repeat(40)],
    found_commit_count: 3,
    path_symbol_history: [
      {
        review_finding_id: "rdf_visual_candidate_4",
        commit_sha: "9".repeat(40),
        path: "pkg/cache/generation_store.go",
        symbol: "GenerationStore.Replace",
        observed_at: "2026-08-24T16:20:00Z",
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: {
      state: "none",
      snapshot_at: undefined,
      conflict: false,
    },
    validation_state: "not_requested",
    possible_duplicates: [],
    version: 7,
    created_at: "2026-08-20T10:00:00Z",
    updated_at: "2026-08-24T16:20:00Z",
  },
  {
    id: repositoryReviewVisualIDs.combinedAttentionRepositoryFinding,
    repository: "octo/picoclaw",
    canonical_title: "Combined attention remains usable on narrow screens",
    canonical_severity: "critical",
    review_finding_ids: ["rdf_visual_combined_attention"],
    occurrence_count: 1,
    found_commits: ["6".repeat(40)],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: "rdf_visual_combined_attention",
        commit_sha: "6".repeat(40),
        path: "pkg/review/attention_projection.go",
        symbol: "AttentionProjection.Render",
        observed_at: "2026-08-25T14:19:00Z",
      },
    ],
    match_state: "provisional",
    lifecycle: "resolution_pending",
    issue: {
      state: "open",
      snapshot_at: "2026-08-25T14:19:00Z",
      conflict: true,
      conflict_urls: [
        "https://github.com/octo/picoclaw/issues/101",
        "https://github.com/octo/picoclaw/issues/102",
      ],
    },
    validation_state: "failed",
    possible_duplicates: [],
    version: 3,
    created_at: "2026-08-25T14:18:30Z",
    updated_at: "2026-08-25T14:19:00Z",
  },
  {
    id: repositoryReviewVisualIDs.conflictRepositoryFinding,
    repository: "octo/picoclaw",
    canonical_title: "Merged occurrences reference different GitHub issues",
    canonical_severity: "high",
    review_finding_ids: ["rdf_visual_conflict_1", "rdf_visual_conflict_2"],
    occurrence_count: 2,
    found_commits: ["3".repeat(40), "4".repeat(40)],
    found_commit_count: 2,
    path_symbol_history: [
      {
        review_finding_id: "rdf_visual_conflict_2",
        commit_sha: "4".repeat(40),
        path: "pkg/issues/association.go",
        symbol: "Association.Merge",
        observed_at: "2026-08-25T13:50:00Z",
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: {
      state: "open",
      snapshot_at: "2026-08-25T13:55:00Z",
      conflict: true,
      conflict_urls: [
        "https://github.com/octo/picoclaw/issues/81",
        "https://github.com/octo/picoclaw/issues/94",
      ],
    },
    validation_state: "not_requested",
    possible_duplicates: [],
    version: 3,
    created_at: "2026-08-24T11:00:00Z",
    updated_at: "2026-08-25T13:55:00Z",
  },
  {
    id: repositoryReviewVisualIDs.failedResolutionRepositoryFinding,
    repository: "octo/picoclaw",
    canonical_title: "Cleanup leaves a stale workspace lease behind",
    canonical_severity: "medium",
    review_finding_ids: ["rdf_visual_failed_resolution"],
    occurrence_count: 1,
    found_commits: ["5".repeat(40)],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: "rdf_visual_failed_resolution",
        commit_sha: "5".repeat(40),
        path: "pkg/workspace/cleanup.go",
        symbol: "Cleanup.Release",
        observed_at: "2026-08-25T14:00:00Z",
      },
    ],
    match_state: "new",
    lifecycle: "resolution_pending",
    issue: {
      state: "closed",
      snapshot_at: "2026-08-25T14:01:00Z",
      conflict: false,
    },
    validation_state: "failed",
    possible_duplicates: [],
    version: 5,
    created_at: "2026-08-25T12:00:00Z",
    updated_at: "2026-08-25T14:01:00Z",
  },
]

const repositoryReviewContexts = repositoryReviewDetailFindings.map(
  (finding, index) => ({
    id: finding.context_ids[0],
    repository: finding.repository,
    commit_sha: finding.commit_sha,
    inventory_hash: "sha256:visual-inventory",
    profile_hash: "sha256:visual-profile",
    run_id: `rrun_visual_${index + 1}`,
    model: finding.models[0],
    reviewer: `review-child-${index + 1}`,
    files: [finding.file],
    raw_digest: `sha256:visual-context-${index + 1}`,
    created_at: finding.created_at,
  }),
)

const repositoryReviewRawFindings = [
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.rawFinding,
    repositoryReviewFindings[0],
    "review",
    "review-child-1",
    "new",
  ),
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.secondRawFinding,
    repositoryReviewFindings[0],
    "code",
    "code-child-2",
    "duplicate",
  ),
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.thirdRawFinding,
    repositoryReviewFindings[1],
    "review",
    "review-child-2",
    "new",
  ),
]

function repositoryReviewRawFinding(
  id: string,
  finding: (typeof repositoryReviewFindings)[number],
  model: string,
  reviewer: string,
  disposition: "new" | "duplicate",
  processing: {
    state?: "pending" | "running" | "failed" | "completed"
    campaignID?: string
    failure?: RepositoryReviewRawFinding["failure"]
    ordinal?: number
  } = {},
): RepositoryReviewRawFinding {
  const processingState = processing.state ?? "completed"
  const completed = processingState === "completed"
  const ordinal =
    processing.ordinal ?? (repositoryReviewVisualIDs.rawFinding === id ? 1 : 2)
  return {
    id,
    version: 1,
    campaign_id: processing.campaignID ?? "rrc_visual",
    admission_bucket: `rdb_visual_${finding.file.path}`,
    insertion_ordinal: ordinal,
    diagnosis_digest: `sha256:${id}`,
    repository: finding.repository,
    commit_sha: finding.commit_sha,
    file: finding.file,
    path: finding.file.path,
    line: finding.line,
    severity: finding.severity,
    title: finding.title,
    symbol: finding.symbol,
    message: finding.message,
    evidence: finding.evidence,
    impact: finding.impact,
    validation: finding.validation,
    context_id: finding.context_ids[0],
    run_id: `rrun_visual_${finding === repositoryReviewFindings[0] ? 1 : 2}`,
    assignment_id: `assignment-${id}`,
    model,
    model_alias: model,
    account: "visual-review-account",
    reviewer,
    deduplication_state: processingState,
    disposition: completed ? disposition : "undecided",
    deduplicated_finding_id: completed ? finding.id : undefined,
    history: [
      {
        state: processingState,
        disposition: completed ? disposition : "undecided",
        deduplicated_finding_id: completed ? finding.id : undefined,
        attempt: processingState === "pending" ? undefined : 1,
        failure: processing.failure,
        at: finding.updated_at,
      },
    ],
    failure: processing.failure,
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }
}

const repositoryReviewProcessingSources: RepositoryReviewRawFinding[] = [
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.processingPending,
    repositoryReviewFindings[0],
    "review",
    "review-child-pending",
    "new",
    { state: "pending", ordinal: 11 },
  ),
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.processingRunning,
    repositoryReviewFindings[1],
    "code",
    "review-child-running",
    "new",
    { state: "running", ordinal: 12 },
  ),
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.processingFailed,
    repositoryReviewFindings[1],
    "review",
    "review-child-failed",
    "new",
    {
      state: "failed",
      ordinal: 13,
      failure: {
        code: "attempt_limit",
        message: "Finding grouping reached its retry limit.",
        retryable: true,
        at: "2026-08-25T14:22:00Z",
      },
    },
  ),
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.processingCompleted,
    repositoryReviewFindings[0],
    "review",
    "review-child-completed",
    "duplicate",
    { state: "completed", ordinal: 14 },
  ),
  repositoryReviewRawFinding(
    repositoryReviewVisualIDs.processingOldCampaign,
    repositoryReviewFindings[1],
    "legacy-review-model",
    "historical-review",
    "new",
    {
      state: "failed",
      campaignID: "rrc_visual_previous",
      ordinal: 15,
      failure: {
        code: "processing_interrupted",
        message: "Historical finding grouping was interrupted.",
        retryable: true,
        at: "2026-08-24T17:00:00Z",
      },
    },
  ),
]

const repositoryReviewFindingHealth = {
  run_findings: {
    total: 6,
    pending: 1,
    processing: 1,
    failed: 1,
    needs_review: 1,
    associated_new: 1,
    associated_existing: 1,
    unrepresented: 3,
  },
  repository_findings: {
    total: repositoryFindings.length,
    provisional: 2,
    validation_failed: 2,
    issue_conflicts: 2,
  },
  findings_processing: repositoryReviewProcessingHealth(
    repositoryReviewProcessingSources,
  ),
  historical_consolidation: {
    required: true,
    status: "failed" as
      | "not_required"
      | "pending"
      | "replaying"
      | "merging"
      | "failed"
      | "completed",
    retryable: true,
  },
  updated_at: "2026-08-25T14:22:00Z",
}

function repositoryReviewProcessingHealth(
  sources: RepositoryReviewRawFinding[],
) {
  return {
    total: sources.length,
    pending: sources.filter(
      (source) => source.deduplication_state === "pending",
    ).length,
    processing: sources.filter(
      (source) => source.deduplication_state === "running",
    ).length,
    failed: sources.filter((source) => source.deduplication_state === "failed")
      .length,
    completed: sources.filter(
      (source) => source.deduplication_state === "completed",
    ).length,
  }
}

function repositoryReviewProcessingDetail(
  source: RepositoryReviewRawFinding,
  sources: RepositoryReviewRawFinding[],
  historicalConsolidation: (typeof repositoryReviewFindingHealth)["historical_consolidation"],
) {
  const finding = repositoryReviewDetailFindings.find(
    (candidate) => candidate.id === source.deduplicated_finding_id,
  )
  const repositoryFinding = finding?.repository_finding_id
    ? repositoryFindings.find(
        (candidate) => candidate.id === finding.repository_finding_id,
      )
    : undefined
  return {
    automation: repositoryReviewAutomation,
    repository: repositoryReviewSummary,
    source,
    context: repositoryReviewContexts.find(
      (context) => context.id === source.context_id,
    ),
    ...(finding ? { finding } : {}),
    ...(repositoryFinding ? { repository_finding: repositoryFinding } : {}),
    findings_processing: repositoryReviewProcessingHealth(sources),
    historical_consolidation: historicalConsolidation,
  }
}

const repositoryReviewIssues = [
  {
    id: repositoryReviewVisualIDs.issue,
    repository: "octo/picoclaw",
    finding_ids: [repositoryReviewVisualIDs.finding],
    origin: "ai_generated",
    generation_id: repositoryReviewVisualIDs.generation,
    resolved_instructions:
      "Write a concise grounded issue with evidence, impact, validation, location, and commit provenance.",
    instructions_mode: "default",
    generator_model: "issue-writer",
    generator_account: "openai-primary",
    generator_profile_id: "rrpf_visual",
    generator_profile_version: 4,
    canonical: true,
    publishable: true,
    deletable: true,
    regeneratable: true,
    title: "Concurrent checkpoint writes can lose a finding",
    body: [
      "## Evidence",
      "",
      "Two writers persist snapshots derived from the same ledger version.",
      "",
      "| Location | Commit |",
      "| --- | --- |",
      `| \`pkg/repoaudit/store.go:418\` | \`${repositoryReviewCommit}\` |`,
      "",
      "## Impact",
      "",
      "A validated finding can disappear from the findings view.",
      "",
      "## Validation",
      "",
      "- Compared both caller snapshots",
      "- Verified there is no version fence",
    ].join("\n"),
    labels: ["bug", "data-loss"],
    state: "editing",
    version: 3,
    created_at: "2026-08-25T14:10:00Z",
    updated_at: "2026-08-25T14:10:00Z",
  },
  {
    id: repositoryReviewVisualIDs.failedIssue,
    repository: "octo/picoclaw",
    finding_ids: [repositoryReviewVisualIDs.secondFinding],
    origin: "ai_generated",
    generation_id: repositoryReviewVisualIDs.generation,
    resolved_instructions:
      "Write a concise grounded issue with evidence, impact, validation, location, and commit provenance.",
    instructions_mode: "default",
    generator_model: "issue-writer",
    generator_account: "openai-primary",
    generator_profile_id: "rrpf_visual",
    generator_profile_version: 4,
    generation_error: "The issue writer returned an invalid structured body.",
    canonical: true,
    publishable: false,
    deletable: true,
    regeneratable: true,
    title: "",
    body: "",
    labels: [],
    state: "failed",
    version: 1,
    created_at: "2026-08-25T14:15:00Z",
    updated_at: "2026-08-25T14:15:00Z",
  },
]

const repositoryReviewRunFindingSummaries = repositoryReviewFindings.map(
  (finding) => ({
    id: finding.id,
    repository: finding.repository,
    path: finding.file.path,
    line: finding.line,
    severity: finding.severity,
    title: finding.title,
    symbol: finding.symbol,
    status: finding.status,
    run_finding_status: finding.run_finding_status,
    association: "existing",
    repository_finding_id: finding.repository_finding_id,
    contributors: finding.models,
    raw_source_count: finding.raw_source_total,
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }),
)

const combinedAttentionRepositoryFinding = repositoryFindings.find(
  (finding) =>
    finding.id === repositoryReviewVisualIDs.combinedAttentionRepositoryFinding,
)!
const repositoryFindingsForCollection = [
  ...repositoryFindings.slice(0, 2),
  combinedAttentionRepositoryFinding,
  ...repositoryFindings.filter(
    (finding, index) =>
      index >= 2 && finding.id !== combinedAttentionRepositoryFinding.id,
  ),
]

const repositoryFindingSummaries = repositoryFindingsForCollection.map(
  (finding) => {
    const latest = finding.path_symbol_history.at(-1)
    return {
      id: finding.id,
      repository: finding.repository,
      canonical_title: finding.canonical_title,
      canonical_severity: finding.canonical_severity,
      path: latest?.path,
      symbol: latest?.symbol,
      match_state: finding.match_state,
      lifecycle: finding.lifecycle,
      issue: {
        state: finding.issue.state,
        snapshot_at: finding.issue.snapshot_at,
        conflict: finding.issue.conflict,
      },
      validation_state: finding.validation_state,
      occurrence_count:
        finding.occurrence_count ?? finding.review_finding_ids.length,
      found_commit_count:
        finding.found_commit_count ?? finding.found_commits.length,
      created_at: finding.created_at,
      updated_at: finding.updated_at,
    }
  },
)

const repositoryReviewIssueSummaries = repositoryReviewIssues.map((issue) => ({
  id: issue.id,
  repository: issue.repository,
  finding_count: issue.finding_ids.length,
  origin: issue.origin,
  generation_id: issue.generation_id,
  canonical: issue.canonical,
  publishable: issue.publishable,
  title: issue.title,
  state: issue.state,
  version: issue.version,
  created_at: issue.created_at,
  updated_at: issue.updated_at,
}))

const repositoryReviewCapabilities = {
  github: true,
  can_generate: true,
  can_publish: true,
  can_search_issues: true,
  can_link_issue: true,
  can_edit: true,
  can_delete: true,
  can_regenerate: true,
  can_purge_history: true,
  can_remove_repository: true,
  purge_blockers: [],
  purge_summary: {
    repository_version: repositoryReviewSummary.version,
    ledger_fence: "rplf_collection_fixture",
    raw_findings: repositoryReviewRawFindings.length,
    deduplicated_findings: repositoryReviewFindings.length,
    repository_findings: repositoryFindings.length,
    issue_previews: repositoryReviewIssues.length,
    external_issue_associations: repositoryFindings.filter((finding) =>
      Boolean(finding.issue.url),
    ).length,
  },
}

export async function installCollectionVisualMocks(
  page: Page,
  state: CollectionVisualState = "ready",
) {
  const processingSources = structuredClone(repositoryReviewProcessingSources)
  const findingHealth = structuredClone(repositoryReviewFindingHealth)
  await page.route(
    (url) => url.pathname.startsWith("/api/"),
    async (route) => {
      const request = route.request()
      const url = new URL(request.url())
      const path = url.pathname
      const method = request.method()
      const reviewRoot = `/api/repository-reviews/automations/${repositoryReviewVisualIDs.automation}`

      if (method === "GET" && isCollectionList(path)) {
        if (state === "loading") await delay(5_000)
        if (state === "error") {
          return json(
            route,
            {
              code: "invalid_query",
              message: "Expected a value after the comparison operator.",
              position: 7,
            },
            400,
          )
        }
      }

      if (method !== "GET") {
        if (path === `${reviewRoot}/findings-processing/retry`) {
          const body = request.postDataJSON() as { source_ids?: unknown }
          const requested = Array.isArray(body.source_ids)
            ? body.source_ids.filter(
                (sourceID): sourceID is string => typeof sourceID === "string",
              )
            : []
          const retriedIDs: string[] = []
          const failures: Array<{
            source_id: string
            code: string
            message: string
          }> = []
          for (const sourceID of requested) {
            const source = processingSources.find(
              (candidate) => candidate.id === sourceID,
            )
            if (
              source?.id === repositoryReviewVisualIDs.processingFailed &&
              source.deduplication_state === "failed"
            ) {
              source.deduplication_state = "pending"
              source.failure = undefined
              source.disposition = "undecided"
              retriedIDs.push(sourceID)
              continue
            }
            failures.push({
              source_id: sourceID,
              code:
                source?.id === repositoryReviewVisualIDs.processingOldCampaign
                  ? "historical_replay_required"
                  : source
                    ? "not_retryable"
                    : "not_found",
              message:
                source?.id === repositoryReviewVisualIDs.processingOldCampaign
                  ? "Historical sources must be retried through historical consolidation."
                  : source
                    ? "Finding processing source is not retryable."
                    : "Finding processing source was not found.",
            })
          }
          findingHealth.findings_processing =
            repositoryReviewProcessingHealth(processingSources)
          findingHealth.updated_at = "2026-08-25T14:23:00Z"
          return json(route, {
            retried_ids: retriedIDs,
            failures,
            findings_processing: findingHealth.findings_processing,
            health: findingHealth,
          })
        }
        const processingRetryMatch = new RegExp(
          `^${reviewRoot}/findings-processing/sources/([^/]+)/retry$`,
        ).exec(path)
        if (processingRetryMatch) {
          const sourceID = decodeURIComponent(processingRetryMatch[1])
          const source = processingSources.find(
            (candidate) => candidate.id === sourceID,
          )
          if (!source) {
            return json(
              route,
              { code: "not_found", message: "Processing source not found" },
              404,
            )
          }
          if (source.deduplication_state === "failed") {
            source.deduplication_state = "pending"
            source.disposition = "undecided"
            source.failure = undefined
          }
          findingHealth.findings_processing =
            repositoryReviewProcessingHealth(processingSources)
          return json(
            route,
            repositoryReviewProcessingDetail(
              source,
              processingSources,
              findingHealth.historical_consolidation,
            ),
            202,
          )
        }
        if (path === `${reviewRoot}/historical-deduplication/retry`) {
          findingHealth.historical_consolidation = {
            required: true,
            status: "pending",
            retryable: false,
          }
          findingHealth.updated_at = "2026-08-25T14:23:00Z"
          return json(
            route,
            {
              automation: repositoryReviewAutomation,
              repository: repositoryReviewSummary,
              historical_deduplication: {
                required: true,
                status: "pending",
              },
            },
            202,
          )
        }
        if (path === `${reviewRoot}/historical-deduplication/restart`) {
          if (body?.confirmed !== true) {
            return json(
              route,
              {
                code: "confirmation_required",
                message: "Historical restart requires explicit confirmation.",
              },
              400,
            )
          }
          findingHealth.historical_consolidation = {
            required: true,
            status: "pending",
            retryable: false,
          }
          findingHealth.updated_at = "2026-08-25T14:23:00Z"
          return json(
            route,
            {
              automation: repositoryReviewAutomation,
              repository: repositoryReviewSummary,
              historical_deduplication: {
                required: true,
                status: "pending",
              },
            },
            202,
          )
        }
        if (path === "/api/event-sources/bulk-delete") {
          return json(route, {
            deleted_ids: [],
            failures: [],
            config_revision: "cfg-visual-2",
            effects: effects(),
          })
        }
        if (
          path === "/api/event-sources" ||
          path.startsWith("/api/event-sources/") ||
          path === "/api/event-source-settings"
        ) {
          return json(route, {
            config_revision: "cfg-visual-2",
            effects: effects(),
          })
        }
        if (path.endsWith("/bulk-delete")) {
          return json(route, { deleted_ids: [], failures: [] })
        }
        return json(route, { status: "ok" })
      }

      switch (path) {
        case "/api/auth/status":
          return json(route, { authenticated: true, initialized: true })
        case "/api/gateway/status":
          return json(route, {
            gateway_status: "running",
            gateway_start_allowed: true,
            gateway_restart_required: false,
            boot_default_model: "code",
            config_default_model: "code",
          })
        case "/api/gateway/logs":
          return json(route, { logs: [], log_total: 0, log_run_id: 1 })
        case "/api/channels/catalog":
          return json(route, { channels: [] })
        case "/api/config":
          return json(route, { channels: {}, channel_list: {} })
        case "/api/accounts":
          return json(route, {
            accounts: state === "empty" ? [] : accounts,
            total: state === "empty" ? 0 : accounts.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY provider ASC, id ASC",
            query_schema: querySchemas.accounts,
          })
        case "/api/account-routers":
          return json(route, {
            account_routers: state === "empty" ? [] : accountRouterSummaries,
            total: state === "empty" ? 0 : accountRouterSummaries.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: querySchemas.accountRouters,
            config_revision: "cfg-visual-1",
          })
        case "/api/event-sources":
          return json(route, {
            event_sources: state === "empty" ? [] : eventSources,
            total: state === "empty" ? 0 : eventSources.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: querySchemas.eventSources,
            config_revision: "cfg-visual-1",
          })
        case "/api/event-source-settings":
          return json(route, {
            event_source_settings: eventSourceSettings,
            eligible_channel_adapters: eligibleEventChannelAdapters,
            config_revision: "cfg-visual-1",
          })
        case "/api/development/repository-assignments":
          return json(route, {
            repository_assignments:
              state === "empty" ? [] : developmentRepositoryAssignments,
            total:
              state === "empty" ? 0 : developmentRepositoryAssignments.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY repository ASC",
            query_schema: querySchemas.developmentRepositoryAssignments,
            config_revision: "cfg-development-visual-1",
            effects: developmentAdminEffects,
          })
        case "/api/development-workspaces":
          return json(route, {
            workspaces: state === "empty" ? [] : developmentWorkspaces,
            total: state === "empty" ? 0 : developmentWorkspaces.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY updated DESC",
            query_schema: querySchemas.developmentWorkspaces,
          })
        case "/api/development/workflow-configurations/items":
          return json(route, {
            workflow_configurations:
              state === "empty" ? [] : developmentWorkflowConfigurations,
            total:
              state === "empty" ? 0 : developmentWorkflowConfigurations.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: querySchemas.developmentWorkflowConfigurations,
            config_revision: "cfg-development-visual-1",
            effects: developmentAdminEffects,
          })
        case "/api/git-workspaces":
          return json(route, {
            workspaces: state === "empty" ? [] : gitWorkspaces,
            total: state === "empty" ? 0 : gitWorkspaces.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY updated DESC",
            query_schema: querySchemas.gitWorkspaces,
            max_total_size_bytes: 21_474_836_480,
            total_size_bytes: 30_408_704,
            ignored_bytes: 1_703_936,
            repository_count: 3,
            workspace_count: 3,
            locked_workspace_count: 1,
            ignored_cleanup_delay_seconds: 86_400,
            drop_delay_seconds: 2_592_000,
          })
        case "/api/git-workspaces/history":
          return json(route, {
            history: state === "empty" ? [] : gitWorkspaceHistory,
            total: state === "empty" ? 0 : gitWorkspaceHistory.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY time DESC",
            query_schema: querySchemas.gitWorkspaceHistory,
          })
        case "/api/git-workspaces/settings":
          return json(route, gitWorkspaceSettings)
        case "/api/oauth/providers":
          return json(route, { providers: oauthProviders })
        case "/api/oauth/codex-account-limits":
          return json(route, {
            accounts: [
              {
                id: "openai:primary",
                provider: "openai",
                default: true,
                plan: "pro",
                limits_status: "available",
                entries: [
                  {
                    name: "codex",
                    status: "available",
                    window: "weekly",
                    used_percent: 36,
                    refreshes_at: "2026-09-01T16:00:00Z",
                  },
                ],
              },
            ],
          })
        case "/api/model-aliases":
          return json(route, {
            model_aliases: state === "empty" ? [] : aliases,
            total: state === "empty" ? 0 : aliases.length,
            next_cursor: "",
            canonical_query: url.searchParams.get("query") ?? "",
            query_schema: querySchemas.aliases,
            config_revision: "cfg-visual-1",
          })
        case "/api/model-routers":
          return json(route, {
            model_routers: state === "empty" ? [] : routers,
            total: state === "empty" ? 0 : routers.length,
            next_cursor: "",
            canonical_query: url.searchParams.get("query") ?? "",
            query_schema: querySchemas.routers,
            config_revision: "cfg-visual-1",
          })
        case "/api/mcp/servers":
          return json(route, {
            servers: state === "empty" ? [] : mcpServerSummaries,
            total: state === "empty" ? 0 : mcpServerSummaries.length,
            next_cursor: "",
            canonical_query: url.searchParams.get("query") ?? "",
            query_schema: querySchemas.mcp,
            config_revision: "cfg-visual-1",
          })
        case "/api/mcp/settings":
        case "/api/mcp":
          return json(route, {
            enabled: true,
            discovery: {
              enabled: false,
              ttl: 5,
              max_search_results: 5,
              use_bm25: true,
              use_regex: false,
            },
            servers: mcpServers,
          })
        case "/api/agents":
          return json(route, {
            agents: state === "empty" ? [] : agents,
            default_agent_id: "main",
            config_revision: "cfg-visual-1",
            effects: effects(),
            total: state === "empty" ? 0 : agents.length,
            next_cursor: "",
            canonical_query: url.searchParams.get("query") ?? "",
            query_schema: querySchemas.agents,
          })
        case "/api/model-evaluations":
          return json(route, {
            evaluations: state === "empty" ? [] : [evaluationSummary()],
            total: state === "empty" ? 0 : 1,
            next_cursor: "",
            canonical_query: url.searchParams.get("query") ?? "",
            query_schema: querySchemas.evaluations,
          })
        case "/api/model-evaluations/options":
          return json(route, evaluationOptions())
        case "/api/repository-reviews/automations":
          return json(route, {
            automations: state === "empty" ? [] : [repositoryReviewAutomation],
            total: state === "empty" ? 0 : 1,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY repository ASC",
            query_schema: querySchemas.reviews,
          })
        case "/api/repository-reviews/profiles":
          return json(route, {
            profiles: state === "empty" ? [] : [repositoryReviewProfile],
            total: state === "empty" ? 0 : 1,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ALL ORDER BY name ASC",
            query_schema: querySchemas.reviewProfiles,
          })
        case "/api/skills":
          return json(route, {
            skills: state === "empty" ? [] : skills,
            total: state === "empty" ? 0 : skills.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: querySchemas.skills,
          })
        case "/api/tools":
          return json(route, {
            tools: state === "empty" ? [] : tools,
            total: state === "empty" ? 0 : tools.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ??
              "ORDER BY category ASC, name ASC",
            query_schema: querySchemas.tools,
          })
        case "/api/workflows/definitions":
          return json(route, {
            workflows: state === "empty" ? [] : workflowDefinitions,
            total: state === "empty" ? 0 : workflowDefinitions.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY ref ASC",
            query_schema: querySchemas.workflowDefinitions,
          })
        case "/api/workflows/runs":
          return json(route, {
            runs: state === "empty" ? [] : workflowRuns,
            total: state === "empty" ? 0 : workflowRuns.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY created DESC",
            query_schema: querySchemas.workflowRuns,
          })
        case "/api/accounts/models":
          return json(route, modelOptions())
      }

      if (path === reviewRoot) {
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          capabilities: repositoryReviewCapabilities,
        })
      }
      if (path === `${reviewRoot}/finding-health`) {
        if (state === "empty") {
          return json(route, {
            run_findings: {
              total: 0,
              pending: 0,
              processing: 0,
              failed: 0,
              needs_review: 0,
              associated_new: 0,
              associated_existing: 0,
              unrepresented: 0,
            },
            repository_findings: {
              total: 0,
              provisional: 0,
              validation_failed: 0,
              issue_conflicts: 0,
            },
            findings_processing: {
              total: 0,
              pending: 0,
              processing: 0,
              failed: 0,
              completed: 0,
            },
            historical_consolidation: {
              required: false,
              status: "not_required",
              retryable: false,
            },
            updated_at: fixedNow,
          })
        }
        return json(route, findingHealth)
      }
      if (path === `${reviewRoot}/findings-processing`) {
        const query =
          url.searchParams.get("query") ?? "ALL ORDER BY updated DESC"
        const stateMatch =
          /\bstate\s*=\s*["']?(pending|running|failed|completed)["']?/iu.exec(
            query,
          )
        const filtered = processingSources
          .filter(
            (source) =>
              !stateMatch || source.deduplication_state === stateMatch[1],
          )
          .toSorted((left, right) =>
            right.updated_at.localeCompare(left.updated_at),
          )
        const cursor = Number(url.searchParams.get("cursor") ?? 0)
        const offset = Number.isSafeInteger(cursor) && cursor >= 0 ? cursor : 0
        const limit = Number(url.searchParams.get("limit") ?? 50)
        const items =
          state === "empty" ? [] : filtered.slice(offset, offset + limit)
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          raw_findings: items,
          total: state === "empty" ? 0 : filtered.length,
          next_cursor:
            state !== "empty" && offset + items.length < filtered.length
              ? String(offset + items.length)
              : "",
          canonical_query: query,
          query_schema: querySchemas.reviewFindingsProcessing,
          findings_processing:
            state === "empty"
              ? {
                  total: 0,
                  pending: 0,
                  processing: 0,
                  failed: 0,
                  completed: 0,
                }
              : repositoryReviewProcessingHealth(processingSources),
          historical_consolidation: findingHealth.historical_consolidation,
          capabilities: repositoryReviewCapabilities,
        })
      }
      const processingDetailMatch = new RegExp(
        `^${reviewRoot}/findings-processing/sources/([^/]+)$`,
      ).exec(path)
      if (processingDetailMatch) {
        const sourceID = decodeURIComponent(processingDetailMatch[1])
        const source = processingSources.find(
          (candidate) => candidate.id === sourceID,
        )
        return source
          ? json(
              route,
              repositoryReviewProcessingDetail(
                source,
                processingSources,
                findingHealth.historical_consolidation,
              ),
            )
          : json(
              route,
              { code: "not_found", message: "Processing source not found" },
              404,
            )
      }
      if (
        path === `${reviewRoot}/findings` ||
        path === `${reviewRoot}/run-findings`
      ) {
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          findings:
            state === "empty" ? [] : repositoryReviewRunFindingSummaries,
          total:
            state === "empty" ? 0 : repositoryReviewRunFindingSummaries.length,
          next_cursor: "",
          canonical_query:
            url.searchParams.get("query") ??
            "ALL ORDER BY severity DESC, updated DESC",
          query_schema: querySchemas.reviewRunFindings,
          findings_processing: {
            raw_total:
              state === "empty" ? 0 : repositoryReviewRawFindings.length,
            pending: 0,
            processing: 0,
            failed: 0,
            completed:
              state === "empty" ? 0 : repositoryReviewRawFindings.length,
            new: state === "empty" ? 0 : 2,
            duplicates: state === "empty" ? 0 : 1,
          },
          historical_deduplication: {
            required: false,
            status: "completed",
          },
          capabilities: repositoryReviewCapabilities,
        })
      }
      if (path === `${reviewRoot}/raw-findings`) {
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          raw_findings: state === "empty" ? [] : repositoryReviewRawFindings,
          total: state === "empty" ? 0 : repositoryReviewRawFindings.length,
          next_cursor: "",
          canonical_query:
            url.searchParams.get("query") ?? "ALL ORDER BY created DESC",
          query_schema: querySchemas.reviewRawFindings,
          findings_processing: {
            raw_total:
              state === "empty" ? 0 : repositoryReviewRawFindings.length,
            pending: 0,
            processing: 0,
            failed: 0,
            completed:
              state === "empty" ? 0 : repositoryReviewRawFindings.length,
            new: state === "empty" ? 0 : 2,
            duplicates: state === "empty" ? 0 : 1,
          },
          historical_deduplication: {
            required: false,
            status: "completed",
          },
          capabilities: repositoryReviewCapabilities,
        })
      }
      if (path === `${reviewRoot}/repository-findings`) {
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          repository_findings:
            state === "empty" ? [] : repositoryFindingSummaries,
          total: state === "empty" ? 0 : repositoryFindingSummaries.length,
          next_cursor: "",
          canonical_query:
            url.searchParams.get("query") ??
            "ALL ORDER BY severity DESC, updated DESC",
          query_schema: querySchemas.reviewRepositoryFindings,
          capabilities: repositoryReviewCapabilities,
        })
      }
      if (path === `${reviewRoot}/issues`) {
        const generationID = url.searchParams.get("generation_id")
        const issues = generationID
          ? repositoryReviewIssueSummaries.filter(
              (issue) => issue.generation_id === generationID,
            )
          : repositoryReviewIssueSummaries
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          issues: state === "empty" ? [] : issues,
          total: state === "empty" ? 0 : issues.length,
          next_cursor: "",
          canonical_query:
            url.searchParams.get("query") ?? "ALL ORDER BY updated DESC",
          query_schema: querySchemas.reviewIssues,
          ...(generationID ? { generation_id: generationID } : {}),
          capabilities: repositoryReviewCapabilities,
        })
      }
      const repositoryFindingPrefix = `${reviewRoot}/repository-findings/`
      if (path.startsWith(repositoryFindingPrefix)) {
        const findingID = decodeURIComponent(
          path.slice(repositoryFindingPrefix.length),
        )
        const repositoryFinding = repositoryFindings.find(
          (candidate) => candidate.id === findingID,
        )
        if (!repositoryFinding) {
          return json(
            route,
            { code: "not_found", message: "Repository finding not found" },
            404,
          )
        }
        const occurrences = repositoryReviewDetailFindings.filter((finding) =>
          repositoryFinding.review_finding_ids.includes(finding.id),
        )
        const finding = occurrences.at(-1)!
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          finding,
          action_finding: finding,
          repository_finding: repositoryFinding,
          occurrences,
          possible_duplicate_findings:
            repositoryFinding.possible_duplicates?.flatMap((duplicate) => {
              const candidate = repositoryFindings.find(
                (finding) => finding.id === duplicate.candidate_id,
              )
              return candidate ? [candidate] : []
            }) ?? [],
          contexts: repositoryReviewContexts.filter((context) =>
            occurrences.some((occurrence) =>
              occurrence.context_ids.includes(context.id),
            ),
          ),
          issue: repositoryReviewIssues.find((issue) =>
            occurrences.some((occurrence) =>
              issue.finding_ids.includes(occurrence.id),
            ),
          ),
          capabilities: repositoryReviewCapabilities,
        })
      }
      const reviewFindingSourcesMatch = new RegExp(
        `^${reviewRoot}/findings/([^/]+)/sources$`,
      ).exec(path)
      if (reviewFindingSourcesMatch) {
        const findingID = decodeURIComponent(reviewFindingSourcesMatch[1])
        const sources = repositoryReviewRawFindings.filter(
          (source) => source.deduplicated_finding_id === findingID,
        )
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          finding_id: findingID,
          sources,
          offset: 0,
          total: sources.length,
        })
      }
      const rawFindingPrefix = `${reviewRoot}/raw-findings/`
      if (path.startsWith(rawFindingPrefix)) {
        const sourceID = decodeURIComponent(path.slice(rawFindingPrefix.length))
        const source = repositoryReviewRawFindings.find(
          (candidate) => candidate.id === sourceID,
        )
        if (!source) {
          return json(
            route,
            { code: "not_found", message: "Raw finding not found" },
            404,
          )
        }
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          source,
          context: repositoryReviewContexts.find(
            (context) => context.id === source.context_id,
          ),
          finding: repositoryReviewFindings.find(
            (finding) => finding.id === source.deduplicated_finding_id,
          ),
        })
      }
      const reviewFindingPrefix = `${reviewRoot}/findings/`
      if (path.startsWith(reviewFindingPrefix)) {
        const findingID = decodeURIComponent(
          path.slice(reviewFindingPrefix.length),
        )
        const finding = repositoryReviewFindings.find(
          (candidate) => candidate.id === findingID,
        )
        if (!finding) {
          return json(
            route,
            { code: "not_found", message: "Finding not found" },
            404,
          )
        }
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          finding,
          contexts: repositoryReviewContexts.filter((context) =>
            finding.context_ids.includes(context.id),
          ),
          issue: repositoryReviewIssues.find((issue) =>
            issue.finding_ids.includes(finding.id),
          ),
          raw_source_total: finding.raw_source_total,
          capabilities: repositoryReviewCapabilities,
        })
      }
      const reviewIssuePrefix = `${reviewRoot}/issues/`
      if (path.startsWith(reviewIssuePrefix)) {
        const issueID = decodeURIComponent(path.slice(reviewIssuePrefix.length))
        const issue = repositoryReviewIssues.find(
          (candidate) => candidate.id === issueID,
        )
        if (!issue) {
          return json(
            route,
            { code: "not_found", message: "Issue preview not found" },
            404,
          )
        }
        return json(route, {
          automation: repositoryReviewAutomation,
          repository: repositoryReviewSummary,
          issue,
          finding: repositoryReviewFindings.find((finding) =>
            issue.finding_ids.includes(finding.id),
          ),
          capabilities: {
            ...repositoryReviewCapabilities,
            can_publish: issue.state === "editing",
            can_edit: issue.state === "editing",
          },
        })
      }

      const aliasName = decodedTail(path, "/api/model-aliases/")
      if (aliasName) {
        const alias = aliases.find((item) => item.name === aliasName)
        return alias
          ? json(route, { model_alias: alias, config_revision: "cfg-visual-1" })
          : json(route, { code: "not_found", message: "Alias not found" }, 404)
      }
      const accountID = decodedTail(path, "/api/accounts/")
      if (accountID && accountID !== "models") {
        const account = accounts.find((item) => item.id === accountID)
        return account
          ? json(route, { account })
          : json(
              route,
              { code: "account_not_found", message: "Account not found" },
              404,
            )
      }
      const accountRouterID = decodedTail(path, "/api/account-routers/")
      if (accountRouterID && !accountRouterID.includes("/")) {
        const accountRouter = accountRouters.find(
          (item) => item.id === accountRouterID,
        )
        return accountRouter
          ? json(route, {
              account_router: accountRouter,
              config_revision: "cfg-visual-1",
            })
          : json(
              route,
              {
                code: "account_router_not_found",
                message: "Account router not found",
              },
              404,
            )
      }
      const eventSourceID = decodedTail(path, "/api/event-sources/")
      if (eventSourceID && !eventSourceID.includes("/")) {
        const eventSource =
          eventSourceDetails[eventSourceID as keyof typeof eventSourceDetails]
        return eventSource
          ? json(route, {
              event_source: eventSource,
              config_revision: "cfg-visual-1",
            })
          : json(
              route,
              {
                code: "event_source_not_found",
                message: "Event source not found",
              },
              404,
            )
      }
      const workflowDefinitionID = decodedTail(
        path,
        "/api/workflows/definitions/",
      )
      if (workflowDefinitionID && !workflowDefinitionID.includes("/")) {
        const workflow = workflowDefinitions.find(
          (item) => item.id === workflowDefinitionID,
        )
        return workflow
          ? json(route, { workflow })
          : json(
              route,
              {
                code: "workflow_definition_not_found",
                message: "Workflow definition not found",
              },
              404,
            )
      }
      const gitWorkspaceID = decodedTail(path, "/api/git-workspaces/")
      if (
        gitWorkspaceID &&
        !gitWorkspaceID.includes("/") &&
        gitWorkspaceID !== "history" &&
        gitWorkspaceID !== "settings"
      ) {
        const workspace = gitWorkspaceDetails[gitWorkspaceID]
        return workspace
          ? json(route, {
              workspace,
              root_dir: "/srv/picoclaw/git-workspaces",
            })
          : json(
              route,
              {
                code: "git_workspace_not_found",
                message: "Git workspace not found",
              },
              404,
            )
      }
      const workflowRunID = decodedTail(path, "/api/workflows/runs/")
      if (workflowRunID && !workflowRunID.includes("/")) {
        const run = workflowRuns.find((item) => item.id === workflowRunID)
        return run
          ? json(route, run)
          : json(
              route,
              { code: "workflow_run_not_found", message: "Run not found" },
              404,
            )
      }
      const routerName = decodedTail(path, "/api/model-routers/")
      if (routerName) {
        const router = routers.find((item) => item.name === routerName)
        return router
          ? json(route, {
              model_router: router,
              config_revision: "cfg-visual-1",
            })
          : json(route, { code: "not_found", message: "Router not found" }, 404)
      }
      const serverName = decodedTail(path, "/api/mcp/servers/")
      if (serverName) {
        const server = mcpServers.find((item) => item.name === serverName)
        return server
          ? json(route, { server, config_revision: "cfg-visual-1" })
          : json(route, { code: "not_found", message: "Server not found" }, 404)
      }
      const agentID = decodedTail(path, "/api/agents/")
      if (agentID && !agentID.includes("/")) {
        const agent = agents.find((item) => item.id === agentID)
        return agent
          ? json(route, {
              agent,
              default_agent_id: "main",
              config_revision: "cfg-visual-1",
              effects: effects(),
            })
          : json(
              route,
              { code: "agent_not_found", message: "Agent not found" },
              404,
            )
      }
      const evaluationID = decodedTail(path, "/api/model-evaluations/")
      if (evaluationID === evaluation.id) {
        return json(route, { evaluation })
      }
      const skillID = decodedTail(path, "/api/skills/")
      if (skillID && !skillID.includes("/")) {
        const skill = skills.find((item) => item.id === skillID)
        return skill
          ? json(route, {
              ...skill,
              content: `# ${skill.name}\n\n${skill.description}`,
            })
          : json(route, { code: "not_found", message: "Skill not found" }, 404)
      }
      const toolID = decodedTail(path, "/api/tools/")
      if (toolID && !toolID.includes("/")) {
        const tool = tools.find((item) => item.id === toolID)
        return tool
          ? json(route, { tool })
          : json(route, { code: "not_found", message: "Tool not found" }, 404)
      }
      return json(route, {})
    },
  )
}

function field(
  name: string,
  type: "string" | "enum" | "boolean" | "number" | "timestamp",
  suggestedValues: string[] = [],
) {
  const comparisons =
    type === "string"
      ? ["=", "!=", "~", "!~", "IN", "NOT IN"]
      : type === "enum" || type === "boolean"
        ? ["=", "!=", "IN", "NOT IN"]
        : ["=", "!=", ">", ">=", "<", "<=", "IN", "NOT IN"]
  return {
    name,
    type,
    operators: comparisons,
    sortable: true,
    ...(suggestedValues.length > 0
      ? { suggested_values: suggestedValues }
      : {}),
  }
}

function schema(
  fields: ReturnType<typeof field>[],
  defaultOrder?:
    | { field: string; direction: "ASC" | "DESC" }
    | Array<{ field: string; direction: "ASC" | "DESC" }>,
) {
  return {
    fields,
    ...(defaultOrder
      ? {
          default_order: Array.isArray(defaultOrder)
            ? defaultOrder
            : [defaultOrder],
        }
      : {}),
  }
}

function effects() {
  return {
    launcher_effect: "applied",
    catalog_effect: "applied",
    gateway_effect: "applied",
  }
}

function evaluationSummary() {
  return {
    id: evaluation.id,
    version: evaluation.version,
    status: evaluation.status,
    repository: evaluation.repository,
    ref: evaluation.ref,
    candidate_models: evaluation.candidate_models,
    progress,
    usage,
    warnings: [],
    created_at: evaluation.created_at,
    updated_at: evaluation.updated_at,
  }
}

function evaluationOptions() {
  return {
    models: [
      { alias: "code", resolved_model: "gpt-5.6-codex", available: true },
      { alias: "fast", resolved_model: "gpt-5.6-mini", available: true },
      {
        alias: "review",
        resolved_model: "claude-sonnet-4.6",
        available: true,
      },
    ],
    repositories: [
      {
        id: "octo/picoclaw",
        repository: "octo/picoclaw",
        label: "octo/picoclaw",
      },
    ],
    profiles: [
      { ...evaluation.profile, available_models: ["code", "fast", "review"] },
    ],
    profile_count: 1,
    code_types: ["hotpath-code", "code", "test", "bench-test"],
    max_files_per_language: 20,
    default_files_per_language: 20,
    max_candidate_models: 8,
  }
}

function modelOptions() {
  return {
    models: [
      {
        index: 0,
        model_name: "openai-primary",
        provider: "openai",
        model: "gpt-5.6-codex",
        api_key: "",
        enabled: true,
        available: true,
        status: "available",
        is_default: true,
        is_virtual: false,
      },
    ],
    model_aliases: aliases,
    model_alias_catalog: [
      { name: "code", description: "Implementation and debugging" },
      { name: "fast", description: "Low-latency routine work" },
      { name: "review", description: "Correctness and safety review" },
    ],
    total: 1,
    default_model: "code",
    default_account_ref: "openai-primary",
    revision: "cfg-visual-1",
    provider_options: [],
  }
}

function isCollectionList(path: string) {
  if (
    /^\/api\/repository-reviews\/automations\/[^/]+\/(findings|findings-processing|repository-findings|issues)$/u.test(
      path,
    )
  ) {
    return true
  }
  return [
    "/api/model-aliases",
    "/api/accounts",
    "/api/account-routers",
    "/api/event-sources",
    "/api/development-workspaces",
    "/api/development/repository-assignments",
    "/api/development/workflow-configurations/items",
    "/api/git-workspaces",
    "/api/git-workspaces/history",
    "/api/model-routers",
    "/api/mcp/servers",
    "/api/agents",
    "/api/model-evaluations",
    "/api/repository-reviews/automations",
    "/api/repository-reviews/profiles",
    "/api/skills",
    "/api/tools",
    "/api/workflows/definitions",
    "/api/workflows/runs",
  ].includes(path)
}

function decodedTail(path: string, prefix: string) {
  if (!path.startsWith(prefix)) return ""
  return decodeURIComponent(path.slice(prefix.length))
}

function delay(milliseconds: number) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds))
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  })
}
