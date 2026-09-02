import AxeBuilder from "@axe-core/playwright"
import { type Page, type Route, expect, test } from "@playwright/test"

import type {
  AccountRouter,
  AccountRouterSummary,
  AccountSummary,
} from "../src/api/accounts"
import type {
  AgentCapabilitiesResponse,
  AgentInfo,
  AgentMutationInput,
  AgentsResponse,
} from "../src/api/agents"
import type {
  MCPConfigResponse,
  MCPServer,
  MCPServerInput,
} from "../src/api/mcp"
import type { OAuthProviderStatus } from "../src/api/oauth"
import type {
  RepositoryFinding,
  RepositoryReviewAutomation,
  RepositoryReviewFinding,
  RepositoryReviewFindingContext,
  RepositoryReviewHistoricalConsolidation,
  RepositoryReviewIssueDraft,
  RepositoryReviewRawFinding,
  RepositoryReviewSummary,
} from "../src/api/repository-reviews"
import prLifecycleFlowFixture from "./fixtures/pr-lifecycle-flow.json" with { type: "json" }

const smokeAccountIDs = {
  primary: "YWNjb3VudABvcGVuYWk6cHJpbWFyeQ",
  review: "YWNjb3VudABhbnRocm9waWM6cmV2aWV3",
  router: "balanced-router",
} as const

const smokeSkillToolIDs = {
  skill: "c2tpbGwAcmV2aWV3LWhlbHBlcg",
  tool: "dG9vbAB3ZWJfc2VhcmNo",
} as const

const smokeEventSourceIDs = {
  github: "eD_ny6SHSWjo-BISQjbmabgVi1UiA9LivNmhCwLoVCY",
  standard: "AtpwqLgv6OwA1riKGYdyok1K5JqqZT0X9HnhIzQlmis",
  channel: "MLlxF-61ul3LWbQ4FYizmxMV4hRFb0pm7lh1O4uJ7gs",
} as const

const smokeGitWorkspaceIDs = {
  primary: "gw-444444444444",
  locked: "gw-555555555555",
} as const

const smokeDevelopmentAdminIDs = {
  repositoryAssignment: "Rjljc2epaibQOt_BhFLZFLSNQFrkJGFxU2BnbKKqal8",
  workflowConfiguration: "editable",
} as const
const smokeDevelopmentWorkspaceID = `devw_${"1".repeat(32)}`

const smokeWorkflowDefinitionIDs: Record<string, string> = {
  "workflows/code-review.yml": "W41H3aj_ymqpg1aP8Vs6TJXUddSpYgjW7VRRcTHj_-I",
  "workflows/github-issue-triage.yml":
    "16gg1Z-92X47AwRe90IKZvdf1a6cNTojnoCdW4qXQQw",
  "workflows/summarize-text.yml": "tpC6ep8WMy-TZuE8ZnYqpGVom7Yf4X3fhF01wP528JU",
  "workflows/support-triage.yml": "uGbxNNOw68IoG1RENU5o7x_Cn0-Z03-MHPhAZUSRANc",
}

const smokeRoutes = [
  "/",
  "/models/aliases",
  "/models/routers",
  "/accounts",
  "/accounts/new",
  `/accounts/${smokeAccountIDs.primary}`,
  `/accounts/${smokeAccountIDs.primary}/edit`,
  "/accounts/routers",
  "/accounts/routers/new",
  `/accounts/routers/${smokeAccountIDs.router}`,
  `/accounts/routers/${smokeAccountIDs.router}/edit`,
  "/events",
  "/event-sources",
  "/event-sources/new",
  `/event-sources/${smokeEventSourceIDs.github}`,
  `/event-sources/${smokeEventSourceIDs.github}/edit`,
  "/event-sources/settings",
  "/development",
  "/development/new",
  `/development/${smokeDevelopmentWorkspaceID}`,
  "/development/repositories",
  "/development/repositories/new",
  `/development/repositories/${smokeDevelopmentAdminIDs.repositoryAssignment}`,
  `/development/repositories/${smokeDevelopmentAdminIDs.repositoryAssignment}/edit`,
  "/development/workflow-configurations",
  "/development/workflow-configurations/new",
  `/development/workflow-configurations/${smokeDevelopmentAdminIDs.workflowConfiguration}`,
  `/development/workflow-configurations/${smokeDevelopmentAdminIDs.workflowConfiguration}/edit`,
  "/development/settings",
  "/notifications",
  "/repository-reviews",
  "/repository-reviews/repositories",
  "/repository-reviews/repositories/new",
  "/repository-reviews/repositories/rra_smoke",
  "/repository-reviews/repositories/rra_smoke/edit",
  "/repository-reviews/repositories/rra_smoke/findings",
  "/repository-reviews/repositories/rra_smoke/findings/rrf_smoke_1",
  "/repository-reviews/repositories/rra_smoke/findings/rrf_smoke_3/link-issue",
  "/repository-reviews/profiles",
  "/repository-reviews/rra_smoke",
  "/repository-reviews/rra_smoke/findings",
  "/repository-reviews/rra_smoke/findings/rdf_smoke_1",
  "/repository-reviews/rra_smoke/findings/rdf_smoke_3/link-issue",
  "/repository-reviews/rra_smoke/raw-findings",
  "/repository-reviews/rra_smoke/raw-findings/rrw_smoke_1",
  "/repository-reviews/rra_smoke/findings-processing",
  "/repository-reviews/rra_smoke/findings-processing/rrw_smoke_processing_failed",
  "/repository-reviews/rra_smoke/issues",
  "/repository-reviews/rra_smoke/issues/rrid_smoke_1",
  "/repository-reviews/rra_smoke/issues/rrid_smoke_1/edit",
  "/model-evaluations",
  "/logs",
  "/agent/agents",
  "/agent/git-workspaces",
  `/agent/git-workspaces/${smokeGitWorkspaceIDs.primary}`,
  "/agent/git-workspaces/history",
  "/agent/git-workspaces/settings",
  "/agent/mcp/servers",
  "/agent/tools",
  `/agent/tools/${smokeSkillToolIDs.tool}`,
  `/agent/tools/${smokeSkillToolIDs.tool}/edit`,
  "/agent/tools/settings/adaptation",
  "/agent/workflows",
  "/agent/workflows/new",
  `/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}`,
  `/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}/edit`,
  "/agent/workflows/settings",
  "/agent/workflows/runs",
  "/agent/workflows/runs/wr_test",
  "/agent/skills",
  "/agent/skills/new",
  `/agent/skills/${smokeSkillToolIDs.skill}`,
  "/agent/hub",
] as const

const modelResponse = {
  models: [
    {
      index: 0,
      model_name: "gpt-4o-mini",
      provider: "openai",
      model: "gpt-4o-mini",
      api_key: "",
      enabled: true,
      available: true,
      status: "available",
      is_default: true,
      is_virtual: false,
    },
    {
      index: 1,
      model_name: "gpt-4o",
      provider: "openai",
      model: "gpt-4o",
      api_key: "sk-****test",
      enabled: true,
      available: true,
      status: "available",
      is_default: false,
      is_virtual: false,
    },
    {
      index: 2,
      model_name: "task-router",
      provider: "model-router",
      model: "task-router",
      api_key: "",
      enabled: true,
      available: true,
      status: "available",
      is_default: false,
      is_virtual: true,
      model_router: {
        name: "task-router",
        enabled: true,
        entry: "entry",
        blocks: [
          {
            id: "entry",
            type: "rules",
            fallback: "default-gpt-4o-mini",
            rules: [{ match: "has_code", target: "code-gpt-4o" }],
          },
          {
            id: "code-gpt-4o",
            type: "model",
            model: "code",
          },
          {
            id: "default-gpt-4o-mini",
            type: "model",
            model: "fast",
          },
        ],
      },
    },
  ],
  model_aliases: [
    {
      name: "code",
      model: "gpt-4o-mini",
      account_overrides: {
        "gpt-4o": "gpt-4o",
      },
    },
    {
      name: "fast",
      model: "gpt-4o-mini",
    },
  ],
  model_alias_catalog: [
    {
      name: "chat",
      description: "General discussion, planning, and technical writing.",
    },
    {
      name: "code",
      description: "Implementation, refactoring, debugging, and tests.",
    },
    {
      name: "investigate",
      description: "Deep research, root-cause analysis, and unfamiliar code.",
    },
    {
      name: "review",
      description: "Correctness, maintainability, and security review.",
    },
    {
      name: "fast",
      description:
        "Low-latency summaries, classification, and routine automation.",
    },
  ],
  total: 3,
  default_account_ref: "gpt-4o-mini",
  default_model: "code",
  provider_options: [
    {
      id: "openai",
      display_name: "OpenAI",
      default_api_base: "https://api.openai.com/v1",
      empty_api_key_allowed: false,
      create_allowed: true,
      supports_fetch: true,
    },
    {
      id: "gemini",
      display_name: "Google Gemini",
      default_api_base: "https://generativelanguage.googleapis.com/v1beta",
      empty_api_key_allowed: false,
      create_allowed: true,
    },
    {
      id: "deepseek",
      display_name: "DeepSeek",
      default_api_base: "https://api.deepseek.com/v1",
      empty_api_key_allowed: false,
      create_allowed: true,
      supports_fetch: true,
    },
  ],
}

const toolsResponse = {
  tools: [
    {
      id: "dG9vbABmaW5kX3NraWxscw",
      name: "find_skills",
      description: "Find skills",
      category: "skills",
      config_key: "tools.find_skills",
      status: "enabled",
    },
    {
      id: "dG9vbABpbnN0YWxsX3NraWxs",
      name: "install_skill",
      description: "Install skills",
      category: "skills",
      config_key: "tools.install_skill",
      status: "blocked",
      reason: "No writable workspace skill directory is configured.",
      reason_code: "dependency_unavailable",
    },
    {
      id: smokeSkillToolIDs.tool,
      name: "web_search",
      description: "Search the web",
      category: "web",
      config_key: "tools.web_search",
      status: "enabled",
    },
  ],
  total: 3,
  next_cursor: "",
  canonical_query: "ORDER BY category ASC, name ASC",
  query_schema: collectionSchema([
    ["name", "string"],
    [
      "category",
      "enum",
      [
        "agents",
        "automation",
        "communication",
        "discovery",
        "filesystem",
        "hardware",
        "skills",
        "web",
      ],
    ],
    ["status", "enum", ["enabled", "disabled", "blocked"]],
    ["reason", "string"],
    ["config_key", "string"],
  ]),
}

const mcpResponse: MCPConfigResponse = {
  enabled: true,
  discovery: {
    enabled: false,
    ttl: 5,
    max_search_results: 5,
    use_bm25: true,
    use_regex: false,
  },
  servers: [
    {
      name: "github",
      enabled: true,
      deferred: null,
      type: "http",
      url: "https://mcp.example.test/github",
      command: "",
      args: [],
      env_file: "",
      env_keys: [],
      header_keys: [],
      auth: {
        type: "oauth",
        configured: true,
        expired: false,
      },
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
      auth: {
        type: "none",
        configured: false,
        expired: false,
      },
    },
  ],
}

const mockCollectionSchemas = {
  accounts: collectionSchema([
    ["id", "string"],
    ["provider", "string"],
    ["account", "string"],
    ["status", "enum"],
    ["auth_method", "string"],
    ["expires_at", "string"],
  ]),
  accountRouters: collectionSchema([
    ["name", "string"],
    ["enabled", "boolean"],
    ["is_default", "boolean"],
    ["status", "enum"],
    ["entry", "string"],
    ["accounts", "number"],
    ["blocks", "number"],
  ]),
  eventSources: collectionSchema([
    ["name", "string"],
    ["kind", "enum", ["webhook", "channel"]],
    ["enabled", "boolean", ["true", "false"]],
    ["format", "enum", ["standard", "github", "deltachat"]],
    [
      "status",
      "enum",
      ["available", "disabled", "unconfigured", "unreachable", "invalid"],
    ],
    ["repositories", "number"],
    ["poll_notifications", "boolean", ["true", "false"]],
  ]),
  developmentRepositoryAssignments: collectionSchema([
    ["repository", "string"],
    ["configuration", "string", ["default", "editable"]],
    ["default_branch", "string", ["main"]],
  ]),
  developmentWorkflowConfigurations: collectionSchema([
    ["id", "string", ["default", "editable"]],
    ["name", "string"],
    ["is_default", "boolean", ["true", "false"]],
    ["bindings", "number"],
    ["deferred_issues", "enum", ["off", "ask", "automatic"]],
  ]),
  developmentWorkspaces: collectionSchema(
    [
      ["id", "string"],
      ["intent", "enum", ["implement_feature", "pickup_pr"]],
      ["source", "enum", ["issue", "brief", "pull_request"]],
      ["repository", "string"],
      ["title", "string"],
      [
        "phase",
        "enum",
        [
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
        ],
      ],
      [
        "execution_state",
        "enum",
        [
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
        ],
      ],
      ["created", "timestamp"],
      ["updated", "timestamp"],
    ],
    { field: "updated", direction: "DESC" },
  ),
  aliases: collectionSchema([
    ["name", "string"],
    ["model", "string"],
    ["overrides", "number"],
    ["disabled_accounts", "number"],
  ]),
  routers: collectionSchema([
    ["name", "string"],
    ["enabled", "boolean"],
    ["blocks", "number"],
    ["rules", "number"],
  ]),
  mcp: collectionSchema([
    ["name", "string"],
    ["enabled", "boolean"],
    ["deferred", "boolean"],
    ["type", "enum"],
    ["auth", "enum"],
  ]),
  agents: collectionSchema([
    ["id", "string"],
    ["name", "string"],
    ["workspace", "string"],
    ["account", "string"],
    ["model", "string"],
    ["default", "boolean"],
    ["implicit", "boolean"],
    ["position", "number"],
  ]),
  evaluations: collectionSchema([
    ["id", "string"],
    ["status", "enum"],
    ["repository", "string"],
    ["ref", "string"],
    ["models", "number"],
    ["progress", "number"],
    ["version", "number"],
    ["created", "timestamp"],
    ["updated", "timestamp"],
  ]),
  reviewRunFindings: collectionSchema(
    [
      ["id", "string"],
      ["repository", "string"],
      ["title", "string"],
      ["path", "string"],
      ["symbol", "string"],
      ["severity", "enum", ["critical", "high", "medium", "low"]],
      ["status", "enum", ["open", "dismissed", "posted"]],
      [
        "run_status",
        "enum",
        [
          "pending",
          "processing",
          "failed",
          "associated_new",
          "associated_existing",
          "needs_review",
        ],
      ],
      [
        "association",
        "enum",
        ["unassociated", "new", "existing", "needs_review"],
      ],
      ["contributors", "string"],
      ["sources", "number"],
      ["created", "timestamp"],
      ["updated", "timestamp"],
    ],
    [
      { field: "severity", direction: "DESC" },
      { field: "updated", direction: "DESC" },
    ],
  ),
  reviewRawFindings: collectionSchema(
    [
      ["id", "string"],
      ["path", "string"],
      ["severity", "enum", ["critical", "high", "medium", "low"]],
      ["title", "string"],
      ["symbol", "string"],
      ["model", "string"],
      ["reviewer", "string"],
      [
        "deduplication_state",
        "enum",
        ["pending", "running", "failed", "completed"],
      ],
      ["disposition", "enum", ["undecided", "new", "duplicate"]],
      ["finding", "string"],
      ["created", "timestamp"],
      ["updated", "timestamp"],
    ],
    { field: "created", direction: "DESC" },
  ),
  reviewFindingsProcessing: collectionSchema(
    [
      ["id", "string"],
      ["campaign", "string"],
      ["title", "string"],
      ["path", "string"],
      ["symbol", "string"],
      ["severity", "enum", ["critical", "high", "medium", "low"]],
      ["model", "string"],
      ["reviewer", "string"],
      ["state", "enum", ["pending", "running", "failed", "completed"]],
      ["disposition", "enum", ["undecided", "new", "duplicate"]],
      ["created", "timestamp"],
      ["updated", "timestamp"],
    ],
    { field: "updated", direction: "DESC" },
  ),
  reviewRepositoryFindings: collectionSchema(
    [
      ["id", "string"],
      ["repository", "string"],
      ["title", "string"],
      ["path", "string"],
      ["symbol", "string"],
      ["severity", "enum", ["critical", "high", "medium", "low"]],
      ["match", "enum", ["new", "known", "provisional"]],
      [
        "lifecycle",
        "enum",
        ["open", "resolution_pending", "resolved", "regressed", "dismissed"],
      ],
      ["issue", "enum", ["none", "draft", "open", "closed", "unknown"]],
      [
        "validation",
        "enum",
        [
          "not_requested",
          "pending",
          "running",
          "confirmed",
          "not_fixed",
          "inconclusive",
          "failed",
        ],
      ],
      ["occurrences", "number"],
      ["commits", "number"],
      ["created", "timestamp"],
      ["updated", "timestamp"],
    ],
    [
      { field: "severity", direction: "DESC" },
      { field: "updated", direction: "DESC" },
    ],
  ),
  reviewIssues: collectionSchema(
    [
      ["id", "string"],
      ["repository", "string"],
      ["title", "string"],
      ["generation", "string"],
      [
        "state",
        "enum",
        ["generating", "failed", "editing", "publishing", "posted", "unknown"],
      ],
      ["origin", "enum", ["ai_generated", "linked", "discovered", "legacy"]],
      ["canonical", "boolean"],
      ["publishable", "boolean"],
      ["findings", "number"],
      ["created", "timestamp"],
      ["updated", "timestamp"],
    ],
    { field: "updated", direction: "DESC" },
  ),
  skills: collectionSchema([
    ["name", "string"],
    ["source", "enum", ["workspace", "global", "builtin"]],
    ["origin", "enum", ["builtin", "manual", "third_party"]],
    ["registry", "string"],
    ["version", "string"],
    ["installed_at", "number"],
  ]),
  tools: collectionSchema([
    ["name", "string"],
    [
      "category",
      "enum",
      [
        "agents",
        "automation",
        "communication",
        "discovery",
        "filesystem",
        "hardware",
        "skills",
        "web",
      ],
    ],
    ["status", "enum", ["enabled", "disabled", "blocked"]],
    ["reason", "string"],
    ["config_key", "string"],
  ]),
  workflowDefinitions: collectionSchema([
    ["ref", "string"],
    ["name", "string"],
    [
      "status",
      "enum",
      ["valid", "invalid", "pending_revalidation", "needs_review"],
    ],
    [
      "trigger",
      "enum",
      [
        "manual",
        "schedule",
        "channel_message",
        "command",
        "runtime_event",
        "event",
        "workflow_call",
        "multiple",
        "none",
      ],
    ],
    ["inputs", "number"],
    ["secrets", "number"],
  ]),
  workflowRuns: collectionSchema([
    ["id", "string"],
    ["workflow", "string"],
    [
      "status",
      "enum",
      ["running", "waiting", "succeeded", "failed", "canceled", "skipped"],
    ],
    ["session", "string"],
    [
      "origin",
      "enum",
      ["manual", "external_event", "external_event_draft_test"],
    ],
    ["created", "timestamp"],
    ["updated", "timestamp"],
    ["completed", "timestamp"],
  ]),
  gitWorkspaces: collectionSchema([
    ["id", "string"],
    ["repository", "string"],
    ["branch", "string"],
    ["status", "enum", ["available", "locked", "dropped"]],
    ["locked", "boolean", ["true", "false"]],
    ["dirty", "boolean", ["true", "false"]],
    ["size", "number"],
    ["ignored", "number"],
    ["updated", "timestamp"],
  ]),
  gitWorkspaceHistory: collectionSchema([
    ["action", "string"],
    ["workspace", "string"],
    ["repository", "string"],
    ["agent", "string"],
    ["time", "timestamp"],
  ]),
}

const defaultOAuthProviders: OAuthProviderStatus[] = [
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
  {
    provider: "deepseek",
    credential_id: "deepseek",
    display_name: "DeepSeek",
    methods: ["token"],
    logged_in: false,
    status: "not_logged_in",
    credentials: [],
  },
  {
    provider: "gemini",
    credential_id: "gemini",
    display_name: "Google Gemini",
    methods: ["token"],
    logged_in: false,
    status: "not_logged_in",
    credentials: [],
  },
]

const defaultSmokeAccounts: AccountSummary[] = [
  {
    id: smokeAccountIDs.review,
    provider: "anthropic",
    account: "anthropic:review",
    status: "needs_refresh",
    auth_method: "token",
    expires_at: "",
  },
  {
    id: smokeAccountIDs.primary,
    provider: "openai",
    account: "openai:primary",
    status: "connected",
    auth_method: "oauth",
    expires_at: "2026-09-01T16:00:00Z",
  },
]

const defaultSmokeAccountRouters: AccountRouter[] = [
  {
    id: smokeAccountIDs.router,
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
    accounts: ["credential:openai:primary"],
    refresh_interval_seconds: 120,
    blocks: [
      {
        id: "batch",
        type: "account",
        account: "credential:openai:primary",
      },
    ],
  },
]

function accountRouterSummary(router: AccountRouter): AccountRouterSummary {
  return {
    id: router.id,
    name: router.name,
    enabled: router.enabled,
    is_default: router.is_default,
    status: router.status,
    entry: router.entry,
    accounts: router.accounts.length,
    blocks: router.blocks.length,
  }
}

type MockEventSource =
  | {
      id: string
      name: string
      kind: "webhook"
      enabled: boolean
      format: "standard" | "github"
      status: string
      poll_notifications: boolean
      repositories: string[]
      target_user: string
      secret_configured: boolean
      endpoint: string
    }
  | {
      id: string
      name: string
      kind: "channel"
      enabled: boolean
      format: "deltachat"
      status: string
      poll_notifications: false
      source: "email"
      mode: "mirror" | "event_only"
      allow_unverified_email: boolean
      channel_enabled: boolean
      channel_type: string
    }

const defaultSmokeEventSources: MockEventSource[] = [
  {
    id: smokeEventSourceIDs.standard,
    name: "build-system",
    kind: "webhook",
    enabled: false,
    format: "standard",
    status: "disabled",
    poll_notifications: false,
    repositories: [],
    target_user: "",
    secret_configured: true,
    endpoint: "/webhooks/events/build-system",
  },
  {
    id: smokeEventSourceIDs.github,
    name: "github-primary",
    kind: "webhook",
    enabled: true,
    format: "github",
    status: "available",
    poll_notifications: true,
    repositories: ["sipeed/picoclaw", "octo/launcher"],
    target_user: "octocat",
    secret_configured: true,
    endpoint: "/webhooks/events/github-primary",
  },
  {
    id: smokeEventSourceIDs.channel,
    name: "primary-inbox",
    kind: "channel",
    enabled: true,
    format: "deltachat",
    status: "available",
    poll_notifications: false,
    source: "email",
    mode: "mirror",
    allow_unverified_email: false,
    channel_enabled: true,
    channel_type: "deltachat",
  },
]

const defaultEventSourceSettings = {
  enabled: true,
  database_path: "eventing/events.db",
  retention_days: 30,
  max_payload_bytes: 1_048_576,
  redact_fields: ["tenant_secret", "deployment_token"],
}

const defaultEligibleEventChannelAdapters = [
  {
    name: "secondary-inbox",
    channel_type: "deltachat",
    channel_enabled: true,
  },
]

function eventSourceSummary(source: MockEventSource) {
  return {
    id: source.id,
    name: source.name,
    kind: source.kind,
    enabled: source.enabled,
    format: source.format,
    status: source.status,
    repositories: source.kind === "webhook" ? source.repositories.length : 0,
    poll_notifications: source.poll_notifications,
  }
}

function accountIDForCredential(credentialID: string): string {
  return Buffer.from(`account\0${credentialID}`).toString("base64url")
}

function accountsFromOAuthProviders(
  providers: OAuthProviderStatus[],
): AccountSummary[] {
  return providers
    .flatMap((provider) => {
      const credentials = provider.credentials?.length
        ? provider.credentials
        : provider.logged_in
          ? [provider]
          : []
      return credentials.map((credential) => {
        const account = credential.credential_id ?? credential.provider
        return {
          id: accountIDForCredential(account),
          provider: credential.provider,
          account,
          status: credential.status,
          auth_method: credential.auth_method ?? "",
          expires_at: credential.expires_at ?? "",
        } satisfies AccountSummary
      })
    })
    .sort(
      (left, right) =>
        left.provider.localeCompare(right.provider) ||
        left.id.localeCompare(right.id),
    )
}

function collectionSchema(
  fields: Array<
    [string, "string" | "enum" | "boolean" | "number" | "timestamp", string[]?]
  >,
  defaultOrder?:
    | { field: string; direction: "ASC" | "DESC" }
    | Array<{ field: string; direction: "ASC" | "DESC" }>,
) {
  return {
    fields: fields.map(([name, type, suggestedValues]) => ({
      name,
      type,
      operators:
        type === "string"
          ? ["=", "!=", "~", "!~", "IN", "NOT IN"]
          : type === "enum" || type === "boolean"
            ? ["=", "!=", "IN", "NOT IN"]
            : ["=", "!=", ">", ">=", "<", "<=", "IN", "NOT IN"],
      sortable: true,
      ...(type === "boolean" ? { suggested_values: ["true", "false"] } : {}),
      ...(type === "enum"
        ? { suggested_values: suggestedValues ?? ["draft", "running"] }
        : {}),
    })),
    ...(defaultOrder
      ? {
          default_order: Array.isArray(defaultOrder)
            ? defaultOrder
            : [defaultOrder],
        }
      : {}),
  }
}

const gitWorkspacePrivateCanary = "controller-private-workspace-must-not-render"

const gitWorkspaceDetails = [
  {
    id: smokeGitWorkspaceIDs.primary,
    repository: "github.com/sipeed/picoclaw.git",
    repository_id: "gw-666666666666",
    remote_url: "https://github.com/sipeed/picoclaw.git",
    branch: "main",
    current_branch: "main",
    ref: "main",
    path: "/tmp/picoclaw-git-workspaces/checkouts/gw-444444444444",
    status: "available",
    locked: false,
    dirty: false,
    size: 4_194_304,
    ignored: 524_288,
    created: "2026-07-15T10:00:00Z",
    updated: "2026-07-16T12:00:00Z",
    last_work: "2026-07-16T12:00:00Z",
    last_cleaned: undefined as string | undefined,
  },
  {
    id: smokeGitWorkspaceIDs.locked,
    repository: "github.com/octo/launcher.git",
    repository_id: "gw-777777777777",
    remote_url: "https://github.com/octo/launcher.git",
    branch: "feature/collections",
    current_branch: "feature/collections",
    ref: "feature/collections",
    path: "/tmp/picoclaw-git-workspaces/checkouts/gw-555555555555",
    status: "locked",
    locked: true,
    dirty: true,
    size: 2_097_152,
    ignored: 65_536,
    created: "2026-07-15T11:00:00Z",
    updated: "2026-07-16T11:55:00Z",
    last_work: "2026-07-16T11:55:00Z",
    last_cleaned: undefined as string | undefined,
    locked_by: {
      agent_id: "reviewer",
      locked_at: "2026-07-16T11:50:00Z",
      heartbeat_at: "2026-07-16T11:55:00Z",
    },
  },
]

const gitWorkspaceSummaries = gitWorkspaceDetails.map(
  ({
    id,
    repository,
    branch,
    status,
    locked,
    dirty,
    size,
    ignored,
    updated,
  }) => ({
    id,
    repository,
    branch,
    status,
    locked,
    dirty,
    size,
    ignored,
    updated,
  }),
)

const gitWorkspaceHistory = [
  {
    id: "dddddddddddd",
    action: "cleaned_ignored",
    workspace: smokeGitWorkspaceIDs.primary,
    repository: "github.com/sipeed/picoclaw.git",
    agent: "main",
    time: "2026-07-16T12:02:00Z",
  },
  {
    id: "eeeeeeeeeeee",
    action: "acquired",
    workspace: smokeGitWorkspaceIDs.locked,
    repository: "github.com/octo/launcher.git",
    agent: "reviewer",
    time: "2026-07-16T11:50:00Z",
  },
]

const gitWorkspaceResponse = {
  workspaces: gitWorkspaceSummaries,
  total: gitWorkspaceSummaries.length,
  next_cursor: "",
  canonical_query: "ORDER BY updated DESC",
  query_schema: {
    ...mockCollectionSchemas.gitWorkspaces,
    default_order: [{ field: "updated", direction: "DESC" }],
  },
  max_total_size_bytes: 21_474_836_480,
  total_size_bytes: 6_291_456,
  ignored_bytes: 589_824,
  repository_count: 2,
  workspace_count: 2,
  locked_workspace_count: 1,
  ignored_cleanup_delay_seconds: 86_400,
  drop_delay_seconds: 2_592_000,
}

const gitWorkspaceHistoryResponse = {
  history: gitWorkspaceHistory,
  total: gitWorkspaceHistory.length,
  next_cursor: "",
  canonical_query: "ORDER BY time DESC",
  query_schema: {
    ...mockCollectionSchemas.gitWorkspaceHistory,
    default_order: [{ field: "time", direction: "DESC" }],
  },
}

const gitWorkspaceSettingsResponse = {
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
  config_revision: "sha256:git-workspace-settings-1",
  effects: {
    launcher_effect: "applied",
    catalog_effect: "applied",
    gateway_effect: "applied",
  },
}

function gitWorkspaceSummaryFromDetail(
  workspace: (typeof gitWorkspaceDetails)[number],
) {
  return {
    id: workspace.id,
    repository: workspace.repository,
    branch: workspace.branch,
    status: workspace.status,
    locked: workspace.locked,
    dirty: workspace.dirty,
    size: workspace.size,
    ignored: workspace.ignored,
    updated: workspace.updated,
  }
}

const eventResponse = {
  id: "ev_0123456789abcdef0123456789abcdef",
  source: "github",
  connector: "triage",
  type: "issues.opened",
  actor: {
    id: "octocat",
    type: "user",
    display_name: "The Octocat",
  },
  subject: {
    id: "42",
    type: "issue",
    name: "Printer is offline",
    url: "https://example.test/issues/42",
  },
  occurred_at: "2026-07-16T11:59:59Z",
  received_at: "2026-07-16T12:00:00Z",
  attributes: {
    body_authenticated: "true",
    signature_algorithm: "hmac-sha256",
    repository_full_name: "octo/repo",
    issue_number: "42",
    issue_url: "https://github.com/octo/repo/issues/42",
  },
  payload_bytes: 84,
  routing: {
    status: "succeeded",
    available_at: "2026-07-16T12:00:00Z",
    attempts: 1,
    updated_at: "2026-07-16T12:00:01Z",
  },
}

const eventDispatchResponse = {
  id: "dsp_0123456789abcdef0123456789abcdef",
  event_id: eventResponse.id,
  workflow_id: "16gg1Z-92X47AwRe90IKZvdf1a6cNTojnoCdW4qXQQw",
  workflow_ref: "workflows/github-issue-triage.yml",
  workflow_revision: "sha256:0123456789abcdef",
  run_id: "wr_smoke",
  status: "succeeded",
  available_at: "2026-07-16T12:00:00Z",
  attempts: 1,
  created_at: "2026-07-16T12:00:00Z",
  updated_at: "2026-07-16T12:00:02Z",
  linked_at: "2026-07-16T12:00:01Z",
  finished_at: "2026-07-16T12:00:02Z",
}

const eventPayloadText =
  '{"issue":42,"estimate":9007199254740993,"title":"Printer is offline"}'

const replayEventID = "ev_fedcba9876543210fedcba9876543210"

const webSearchConfigResponse = {
  provider: "openai",
  current_service: "openai",
  prefer_native: true,
  providers: [
    {
      id: "openai",
      label: "OpenAI",
      configured: true,
      current: true,
      requires_auth: true,
    },
  ],
  settings: {
    openai: {
      enabled: true,
      max_results: 5,
      api_key_set: true,
    },
  },
}

const toolAdaptationResponse = {
  enabled: true,
  visible_tool_surface: "auto",
  learn_from_tool_calls: true,
  run_model_probes: true,
  allow_runtime_downgrade: "auto",
  allow_runtime_promotion: "auto",
  apply_visible_changes: "next_session",
  cache_sensitive_apis: "auto",
  cache_breaking_downgrade: false,
  profile_overrides: [
    {
      provider: "openai",
      model:
        "very-long-model-name-with-reasoning-context-and-tool-capabilities",
      visible_tool_surface: "simple",
      cache_sensitive_apis: "never",
    },
  ],
  profiles: [
    {
      id: "openai/gpt-4o-mini",
      label: "gpt-4o-mini",
      source: "model alias",
      is_default: true,
      is_override: false,
      probe_available: true,
      resolved: {
        provider: "openai",
        model: "gpt-4o-mini",
        state_path: "/tmp/tool-adaptation-state.json",
        visible_tool_surface: "codex",
        pinned_tool_surface: "codex",
        surface_evidence: "heuristic",
        runtime_downgrade: false,
        runtime_promotion: false,
        apply_visible_changes: "next_session",
        cache_sensitive: true,
        cache_evidence: "heuristic",
      },
    },
    {
      id: "openai/very-long-model-name-with-reasoning-context-and-tool-capabilities",
      label:
        "very-long-model-name-with-reasoning-context-and-tool-capabilities",
      source:
        "manual override for a configured provider profile with a long label",
      is_default: false,
      is_override: true,
      probe_available: false,
      resolved: {
        provider: "openai",
        model:
          "very-long-model-name-with-reasoning-context-and-tool-capabilities",
        state_path: "/tmp/tool-adaptation-state.json",
        visible_tool_surface: "simple",
        pinned_tool_surface: "simple",
        surface_evidence: "config",
        runtime_downgrade: true,
        runtime_promotion: true,
        apply_visible_changes: "next_session",
        cache_sensitive: false,
        cache_evidence: "config",
      },
    },
  ],
}

const skillsResponse = {
  skills: [
    {
      id: "c2tpbGwAY29kZS1yZXZpZXc",
      name: "code-review",
      path: "/usr/share/picoclaw/skills/code-review",
      source: "builtin",
      description: "Inspect changes for correctness",
      origin: "builtin",
      origin_kind: "builtin",
      version: "bundled",
      installed_version: "bundled",
      removable: false,
    },
    {
      id: smokeSkillToolIDs.skill,
      name: "review-helper",
      path: "/workspace/skills/review-helper",
      source: "workspace",
      description: "Review code changes",
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
  ],
  total: 2,
  next_cursor: "",
  canonical_query: "ORDER BY name ASC",
  query_schema: mockCollectionSchemas.skills,
}

const workflowSettingsResponse = {
  configured: {
    enabled: true,
    tool_enabled: false,
    definitions_dir: "",
    max_concurrent_runs: 0,
    default_timeout_seconds: 0,
    max_call_depth: 0,
    retention_days: 0,
  },
  effective: {
    enabled: true,
    tool_enabled: false,
    definitions_dir: "workflows",
    max_concurrent_runs: 4,
    default_timeout_seconds: 300,
    max_call_depth: 8,
    retention_days: 30,
  },
  config_revision: "sha256:workflow-settings-1",
  effects: {
    launcher_effect: "applied",
    catalog_effect: "applied",
    gateway_effect: "applied",
  },
}

const workflowRun = {
  id: "wr_test",
  workflow_id: smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"],
  workflow_ref: "workflows/summarize-text.yml",
  status: "succeeded",
  session: "workflow:demo",
  inputs: { text: "hello" },
  outputs: { summary: "hello" },
  jobs: {
    main: { id: "main", status: "succeeded" },
  },
  steps: {
    "main/summarize": { id: "summarize", status: "succeeded" },
  },
  child_run_ids: [],
  created_at: "2026-07-16T12:00:00Z",
  updated_at: "2026-07-16T12:00:01Z",
  completed_at: "2026-07-16T12:00:01Z",
}

const nullableWorkflowRun = {
  ...workflowRun,
  id: "wr_nulls",
  child_run_ids: null,
  jobs: null,
  steps: null,
}

const retryWorkflowRun = {
  ...workflowRun,
  id: "wr_retry",
  retry_of_run_id: "wr_test",
  outputs: { summary: "retry summary" },
  created_at: "2026-07-16T12:00:02Z",
  updated_at: "2026-07-16T12:00:03Z",
  completed_at: "2026-07-16T12:00:03Z",
}

const lifecycleEventID = "ev_0123456789abcdef0123456789abcdef"
const lifecycleDispatchID = "dsp_0123456789abcdef0123456789abcdef"
const lifecycleDecoyEventID = "ev_fedcba9876543210fedcba9876543210"
const lifecycleDecoyDispatchID = "dsp_fedcba9876543210fedcba9876543210"

const lifecycleWorkflowRun = {
  ...workflowRun,
  id: "wr_lifecycle",
  workflow_ref: "workflows/github-issue-triage.yml",
  origin: {
    kind: "external_event",
    event_id: lifecycleEventID,
    dispatch_id: lifecycleDispatchID,
    root_run_id: "wr_lifecycle_root",
  },
  event: {
    id: lifecycleDecoyEventID,
    source: "github",
    connector: "primary",
    type: "issues.opened",
  },
  inputs: {
    event_id: lifecycleDecoyEventID,
    dispatch_id: lifecycleDecoyDispatchID,
  },
  session: `event:${lifecycleDecoyEventID}:dispatch:${lifecycleDecoyDispatchID}`,
}

const cancelableWorkflowRun = {
  ...workflowRun,
  id: "wr_cancel",
  status: "running",
  outputs: {},
  jobs: {},
  steps: {},
  updated_at: "2026-07-16T12:04:00Z",
  completed_at: undefined,
}

const workflowDraftYAML = `name: Support Triage
on:
  workflow_call:
    inputs:
      ticket:
        type: string
        required: true
jobs:
  triage:
    runs-on: picoclaw
    steps:
      - id: summarize
        uses: agent/main
        with:
          prompt: Summarize support tickets
`

const workflowEventDraftYAML = `name: Support Triage
on:
  workflow_call:
    inputs:
      ticket:
        type: string
        required: true
  event:
    sources:
      - github
    types:
      - issues.opened
jobs:
  triage:
    runs-on: picoclaw
    steps:
      - id: summarize
        uses: agent/main
        with:
          prompt: Summarize support tickets
`

const workflowInspectionSecretCanary =
  "ui-smoke-workflow-secret-must-not-render"
const workflowInspectionRawYAMLCanary =
  "name: ui-smoke-raw-workflow-yaml-must-not-render"

type MockWorkflowInspectionSource =
  | { kind: "published"; ref: string }
  | { kind: "template"; template_name: string }

function workflowDefinitionInspection(source: MockWorkflowInspectionSource) {
  const inspection = {
    source,
    revision:
      "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    complete: true,
    validation: {
      valid: true,
      issue_count: 0,
      issues: [],
      truncated: false,
    },
    triggers: {
      manual: { present: false, projected: true },
      schedule: {
        present: true,
        projected: true,
        value: [{ cron: "0 9 * * 1" }],
      },
      channel_message: { present: false, projected: true },
      command: { present: false, projected: true },
      runtime_event: { present: false, projected: true },
      event: {
        present: true,
        projected: true,
        value: {
          sources: ["github"],
          types: ["issues.opened"],
        },
      },
      workflow_call: { present: false, projected: true },
    },
    jobs: [
      {
        id: "review",
        kind: "steps",
        steps: [
          {
            index: 0,
            id: "analyze",
            kind: "agent",
            target: "agent/main",
          },
        ],
      },
    ],
    dependencies: [{ kind: "agent", target: "main", occurrences: 1 }],
    effects: [
      {
        kind: "model_or_delegated_action_possible",
        target: "main",
        occurrences: 1,
      },
    ],
    limits: [],
  }

  const serialized = JSON.stringify(inspection)
  expect(serialized).not.toContain(workflowInspectionSecretCanary)
  expect(serialized).not.toContain(workflowInspectionRawYAMLCanary)
  return inspection
}

const workflowCapabilityLongToolName = `z${"x".repeat(200)}`

function workflowAuthoringCapabilities() {
  return {
    complete: true,
    mcp_status: "ready",
    agents: [
      {
        id: "main",
        target: "agent/main",
        is_default: true,
        readiness: "ready",
      },
      {
        id: "reviewer",
        target: "agent/reviewer",
        is_default: false,
        readiness: "not_configured",
      },
    ],
    tools: [
      {
        name: "message",
        target: "tool/message",
        readiness: "ready",
        parameter_shape_projected: true,
        parameter_shape: {
          type: "object",
          properties: [
            {
              name: "channel",
              required: false,
              shape: { type: "string" },
            },
            {
              name: "text",
              required: true,
              shape: {
                type: "string",
                enum: ["brief", "full"],
              },
            },
          ],
          additional_properties: { allowed: false },
        },
      },
      {
        name: workflowCapabilityLongToolName,
        target: `tool/${workflowCapabilityLongToolName}`,
        readiness: "ready",
        parameter_shape_projected: true,
        parameter_shape: {},
      },
    ],
    mcp_tools: [
      {
        server: "github",
        tool: "create_issue",
        target: "mcp/github/create_issue",
        readiness: "ready",
        parameter_shape_projected: true,
        parameter_shape: {
          type: "object",
          additional_properties: {
            shape: { type: "string" },
          },
        },
      },
    ],
    functions: [
      {
        name: "git.diff",
        target: "function/git.diff",
        readiness: "ready",
      },
      {
        name: "git.filter",
        target: "function/git.filter",
        readiness: "ready",
      },
      {
        name: "git.inventory",
        target: "function/git.inventory",
        readiness: "ready",
      },
      {
        name: "review.repository",
        target: "function/review.repository",
        readiness: "ready",
      },
      {
        name: "workflow.artifact",
        target: "function/workflow.artifact",
        readiness: "ready",
      },
      {
        name: "workflow.state",
        target: "function/workflow.state",
        readiness: "ready",
      },
    ],
    limits: [],
  }
}

const supportTriageWorkflowDefinition = {
  ref: "workflows/support-triage.yml",
  name: "Support Triage",
  workflow_call: {
    inputs: {
      ticket: {
        type: "string",
        required: true,
      },
    },
  },
}

function smokeWorkflowDefinitionSummary(
  workflow:
    | typeof supportTriageWorkflowDefinition
    | { ref: string; name?: string },
  status: string,
) {
  const workflowCall =
    "workflow_call" in workflow ? workflow.workflow_call : undefined
  return {
    ...workflow,
    id:
      smokeWorkflowDefinitionIDs[workflow.ref] ??
      smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"],
    status,
    trigger: workflowCall ? "workflow_call" : "manual",
    inputs: Object.keys(workflowCall?.inputs ?? {}).length,
    secrets: Object.keys(
      (workflowCall as { secrets?: Record<string, unknown> } | undefined)
        ?.secrets ?? {},
    ).length,
  }
}

const workflowDraftSession = {
  id: "dev_test",
  session_revision: "opaque-session-revision",
  draft_revision: "opaque-draft-revision",
  base_target_revision: "opaque-base-target-revision",
  reason: "new",
  status: "editing",
  prompt: "Triage support tickets",
  target_workflow_ref: "workflows/support-triage.yml",
  target_picoclaw_version: "test",
  target_git_commit: "test",
  yaml: workflowDraftYAML,
  validation: {
    valid: true,
    validated_at: "2026-07-16T12:00:00Z",
  },
  created_at: "2026-07-16T12:00:00Z",
  updated_at: "2026-07-16T12:00:00Z",
}

const workflowDraftLastTest = {
  draft_key: workflowDraftKey(
    workflowDraftSession.target_workflow_ref,
    workflowDraftYAML,
  ),
  draft_revision: workflowDraftSession.draft_revision,
  target_workflow_ref: workflowDraftSession.target_workflow_ref,
  run_id: "wr_draft",
  status: "succeeded",
  tested_at: "2026-07-16T12:01:01Z",
}

type MockWorkflowDraftLastTest = typeof workflowDraftLastTest & {
  event_id?: string
}

type MockWorkflowDevelopmentSession = Omit<
  typeof workflowDraftSession,
  "last_test"
> & {
  source_workflow_ref?: string
  last_test?: MockWorkflowDraftLastTest
}

const draftWorkflowRun = {
  id: "wr_draft",
  workflow_ref: "draft:workflows/support-triage.yml",
  status: "succeeded",
  session: "workflow:draft",
  delivery: {
    channel: "telegram",
    chat_id: "support",
    topic_id: "draft-topic",
  },
  event: {
    source: "draft_test",
    request_id: "req_draft",
  },
  inputs: { ticket: "Printer is offline" },
  outputs: { summary: "draft summary" },
  jobs: {
    triage: {
      id: "triage",
      status: "succeeded",
      outputs: { summary: "draft summary" },
    },
  },
  steps: {
    "triage/summarize": {
      id: "summarize",
      status: "succeeded",
      outputs: { text: "draft summary" },
    },
  },
  child_run_ids: [],
  created_at: "2026-07-16T12:01:00Z",
  updated_at: "2026-07-16T12:01:01Z",
  completed_at: "2026-07-16T12:01:01Z",
}

const manualWorkflowRun = {
  id: "wr_manual",
  workflow_ref: "workflows/support-triage.yml",
  status: "succeeded",
  session: "workflow:manual",
  delivery: {
    channel: "telegram",
    chat_id: "support",
    topic_id: "manual-topic",
  },
  event: {
    source: "manual",
    request_id: "req_manual",
  },
  inputs: { ticket: "Printer is offline" },
  outputs: { summary: "manual summary" },
  jobs: {
    triage: {
      id: "triage",
      status: "succeeded",
      outputs: { summary: "manual summary" },
    },
  },
  steps: {
    "triage/summarize": {
      id: "summarize",
      status: "succeeded",
      outputs: { text: "manual summary" },
    },
  },
  child_run_ids: [],
  created_at: "2026-07-16T12:02:00Z",
  updated_at: "2026-07-16T12:02:01Z",
  completed_at: "2026-07-16T12:02:01Z",
}

const runningDraftWorkflowRun = {
  ...draftWorkflowRun,
  status: "running",
  outputs: {},
  jobs: {
    triage: {
      ...draftWorkflowRun.jobs.triage,
      status: "running",
      outputs: {},
    },
  },
  steps: {
    "triage/summarize": {
      ...draftWorkflowRun.steps["triage/summarize"],
      status: "running",
      outputs: {},
    },
  },
  completed_at: undefined,
}

const failedDraftWorkflowRun = {
  ...draftWorkflowRun,
  id: "wr_draft_failed",
  status: "failed",
  error: "agent step failed",
  outputs: {},
  jobs: {
    triage: {
      ...draftWorkflowRun.jobs.triage,
      status: "failed",
      error: "agent step failed",
      outputs: {},
    },
  },
  steps: {
    "triage/summarize": {
      ...draftWorkflowRun.steps["triage/summarize"],
      status: "failed",
      error: "agent step failed",
      outputs: {},
    },
  },
  updated_at: "2026-07-16T12:01:03Z",
  completed_at: "2026-07-16T12:01:03Z",
}

const runningManualWorkflowRun = {
  ...manualWorkflowRun,
  status: "running",
  outputs: {},
  jobs: {
    triage: {
      ...manualWorkflowRun.jobs.triage,
      status: "running",
      outputs: {},
    },
  },
  steps: {
    "triage/summarize": {
      ...manualWorkflowRun.steps["triage/summarize"],
      status: "running",
      outputs: {},
    },
  },
  completed_at: undefined,
}

function workflowStamp(ref: string, status = "valid") {
  const stamp: {
    workflow_ref: string
    workflow_hash: string
    validated_against_picoclaw_version: string
    validated_against_git_commit: string
    workflow_engine_version: string
    workflow_schema_version: string
    validator_fingerprint: string
    status: string
    validated_at: string
    warnings?: Array<{ message: string }>
  } = {
    workflow_ref: ref,
    workflow_hash: `${ref}:hash`,
    validated_against_picoclaw_version: "test",
    validated_against_git_commit: "test",
    workflow_engine_version: "1",
    workflow_schema_version: "1",
    validator_fingerprint: "test",
    status,
    validated_at: "2026-07-16T12:00:00Z",
  }
  if (status === "pending_revalidation") {
    stamp.warnings = [
      {
        message:
          "workflow must be revalidated after the current Picoclaw version change",
      },
    ]
  }
  return stamp
}

function workflowDraftKey(ref: string, yaml: string) {
  return `${ref.trim()}\u0000${normalizeWorkflowDraftYAML(yaml)}`
}

function normalizeWorkflowDraftYAML(yaml: string) {
  const trimmed = yaml.trimEnd()
  return trimmed === "" ? "" : `${trimmed}\n`
}

const channelCatalogResponse = {
  channels: [
    {
      name: "telegram",
      display_name: "Telegram",
      config_key: "telegram",
    },
    {
      name: "discord",
      display_name: "Discord",
      config_key: "discord",
    },
  ],
}

const repositoryReviewAutomationID = "rra_smoke"
const repositoryReviewFindingOneID = "rdf_smoke_1"
const repositoryReviewFindingTwoID = "rdf_smoke_2"
const repositoryReviewFindingThreeID = "rdf_smoke_3"
const repositoryReviewFindingFourID = "rdf_smoke_4"
const repositoryReviewAttentionFindingID = "rdf_smoke_attention"
const repositoryReviewCandidateFindingID = "rdf_smoke_candidate_4"
const repositoryReviewNormalAggregateID = "rrf_smoke_normal_attention"
const repositoryReviewProvisionalID = "rrf_smoke_provisional_attention"
const repositoryReviewCandidateID = "rrf_smoke_candidate_attention"
const repositoryReviewCombinedAttentionID = "rrf_smoke_combined_attention"
const repositoryReviewConflictID = "rrf_smoke_issue_conflict_attention"
const repositoryReviewFailedCheckID = "rrf_smoke_failed_check_attention"
const repositoryReviewIssueOneID = "rrid_smoke_1"
const repositoryReviewIssueTwoID = "rrid_smoke_2"
const repositoryReviewProcessingPendingID = "rrw_smoke_processing_pending"
const repositoryReviewProcessingRunningID = "rrw_smoke_processing_running"
const repositoryReviewProcessingFailedID = "rrw_smoke_processing_failed"
const repositoryReviewProcessingCompletedID = "rrw_smoke_processing_completed"
const repositoryReviewProcessingOldCampaignID =
  "rrw_smoke_processing_old_campaign"
const repositoryReviewCommitSHA = "a".repeat(40)

const repositoryReviewAutomationFixture: RepositoryReviewAutomation = {
  id: repositoryReviewAutomationID,
  version: 7,
  profile_id: "rrpf_smoke",
  profile_version: 3,
  branch: "main",
  name: "Correctness review",
  repository: "octo/repo",
  ref: "main",
  target: "all",
  account_ref: "openai-primary",
  effective_account_ref: "openai-primary",
  review_focus: "Find concrete correctness and reliability defects.",
  scope_policy: {
    code_types: ["hotpath-code", "code", "test"],
    include_folders: ["pkg", "cmd"],
    exclude_folders: ["vendor"],
    free_text: "Prioritize persistent state transitions.",
  },
  reviewer_models: ["review"],
  issue_writer_model: "issue-writer",
  compare_models: false,
  force: false,
  max_files_per_run: 24,
  max_content_bytes: 524_288,
  max_parallel_children: 4,
  auto_continue: true,
  model_prices: {},
  budget: { guard_expression: "tokens.total < 250000" },
  status: "running",
  active_run_id: "rrun_smoke_current",
  run_ids: ["rrun_smoke_previous", "rrun_smoke_current"],
  usage: {
    prompt_tokens: 18_240,
    completion_tokens: 3_840,
    total_tokens: 22_080,
    cached_tokens: 5_120,
  },
  estimated_cost_usd: 0.42,
  progress: {
    stage: "reviewing checkpoints",
    completed_batches: 3,
    total_batches: 5,
    reviewed_files: 23,
    remaining_files: 16,
    unsupported_files: 1,
    raw_findings: 5,
    deduplicated_findings: 4,
    findings: 4,
    scope_frozen: true,
  },
  model_stats: [],
  account_limits: [],
  scope_plan: {
    commit_sha: repositoryReviewCommitSHA,
    policy_hash: "sha256:scope-policy-smoke",
    hash: "sha256:scope-plan-smoke",
    summary: "40 source and test files pinned at the selected commit.",
    warnings: [],
    counts: {
      total_files: 46,
      code_type_files: 44,
      include_files: 42,
      excluded_files: 2,
      selected_files: 40,
    },
  },
  resolved_commit_sha: repositoryReviewCommitSHA,
  started_at: "2026-08-26T12:00:00Z",
  created_at: "2026-08-25T12:00:00Z",
  updated_at: "2026-08-26T12:05:00Z",
}

const repositoryReviewSummaryFixture: RepositoryReviewSummary = {
  schema_version: 1,
  id: "rrs_smoke",
  repository: "octo/repo",
  version: 12,
  review_version: 4,
  last_commit_sha: repositoryReviewCommitSHA,
  finding_count: 4,
  repository_finding_count: 4,
  open_finding_count: 2,
  issue_draft_count: 2,
  unsupported_count: 1,
  reviewed_file_count: 24,
  excluded_file_count: 2,
  updated_at: "2026-08-26T12:05:00Z",
}

const repositoryReviewFindingsFixture: RepositoryReviewFinding[] = [
  {
    id: repositoryReviewFindingOneID,
    fingerprint: "sha256:lost-update",
    repository: "octo/repo",
    commit_sha: repositoryReviewCommitSHA,
    file: {
      path: "pkg/store/ledger.go",
      blob_sha: "b".repeat(40),
      size_bytes: 8_192,
      category: "hotpath-code",
    },
    line: 142,
    severity: "high",
    title: "Concurrent checkpoint writes can lose a finding",
    symbol: "Ledger.SaveFinding",
    message: "The read-modify-write sequence has no version fence.",
    evidence:
      "Two callers load the same ledger version and each replaces the findings slice; the later rename discards the earlier checkpoint.",
    impact:
      "A validated repository finding can disappear from the findings view.",
    validation: {
      status: "confirmed",
      summary: "Traced both writers through the atomic rename path.",
      checks: ["Compared both caller snapshots", "Verified no CAS guard"],
    },
    context_ids: ["rrctx_smoke_1"],
    models: ["review"],
    observation_count: 1,
    observations: [
      {
        context_id: "rrctx_smoke_1",
        model: "review",
        reviewer: "review-child-2",
        severity: "high",
        title: "Concurrent checkpoint writes can lose a finding",
        symbol: "Ledger.SaveFinding",
        line: 142,
        message: "The read-modify-write sequence has no version fence.",
        evidence: "Both writers persist snapshots derived from version 11.",
        impact: "One validated checkpoint is overwritten.",
        validation: {
          status: "confirmed",
          summary: "Interleaving reproduced from the two call paths.",
        },
      },
    ],
    status: "open",
    repository_finding_id: "rrf_smoke_1",
    repository_match_state: "known",
    run_finding_status: "associated_new",
    version: 2,
    created_at: "2026-08-26T12:02:00Z",
    updated_at: "2026-08-26T12:02:00Z",
    raw_source_total: 2,
  },
  {
    id: repositoryReviewFindingTwoID,
    fingerprint: "sha256:path-escape",
    repository: "octo/repo",
    commit_sha: repositoryReviewCommitSHA,
    file: {
      path: "pkg/archive/extract.go",
      blob_sha: "c".repeat(40),
      size_bytes: 4_096,
      category: "code",
    },
    line: 88,
    severity: "critical",
    title: "Archive extraction accepts paths outside the workspace",
    symbol: "Extractor.Write",
    message: "The joined path is not checked against the destination root.",
    evidence: "A ../ entry reaches os.Create after filepath.Join.",
    impact: "A crafted archive can overwrite files outside the workspace.",
    validation: {
      status: "confirmed",
      summary: "Normalized an escaping entry and followed it to os.Create.",
      checks: ["Checked platform path normalization"],
    },
    context_ids: ["rrctx_smoke_2"],
    models: ["review", "code"],
    observation_count: 2,
    status: "open",
    issue_draft_id: repositoryReviewIssueOneID,
    repository_finding_id: "rrf_smoke_2",
    repository_match_state: "known",
    run_finding_status: "associated_new",
    version: 3,
    created_at: "2026-08-26T12:02:30Z",
    updated_at: "2026-08-26T12:03:00Z",
    raw_source_total: 1,
  },
  {
    id: repositoryReviewFindingThreeID,
    fingerprint: "sha256:retry-storm",
    repository: "octo/repo",
    commit_sha: repositoryReviewCommitSHA,
    file: {
      path: "pkg/queue/worker.go",
      blob_sha: "d".repeat(40),
      size_bytes: 6_144,
      category: "hotpath-code",
    },
    line: 211,
    severity: "medium",
    title: "Canceled jobs are retried without backoff",
    symbol: "Worker.retry",
    message: "Cancellation is classified as a transient queue error.",
    evidence: "context.Canceled reaches the immediate retry branch.",
    impact: "Shutdown can produce a tight retry loop and noisy duplicate work.",
    validation: {
      status: "confirmed",
      summary: "Followed cancellation from dequeue through retry scheduling.",
      checks: ["Verified retry delay remains zero"],
    },
    context_ids: ["rrctx_smoke_3"],
    models: ["review"],
    observation_count: 1,
    status: "open",
    repository_finding_id: "rrf_smoke_3",
    repository_match_state: "known",
    run_finding_status: "associated_new",
    version: 1,
    created_at: "2026-08-26T12:03:30Z",
    updated_at: "2026-08-26T12:03:30Z",
    raw_source_total: 1,
  },
  {
    id: repositoryReviewFindingFourID,
    fingerprint: "sha256:stale-cache",
    repository: "octo/repo",
    commit_sha: repositoryReviewCommitSHA,
    file: {
      path: "pkg/config/cache.go",
      blob_sha: "e".repeat(40),
      size_bytes: 3_072,
      category: "code",
    },
    line: 57,
    severity: "low",
    title: "Configuration cache survives a failed reload",
    symbol: "Cache.Reload",
    message: "The stale value remains readable after an invalidation failure.",
    evidence: "The error branch returns before clearing the cached revision.",
    impact:
      "Callers can observe configuration older than the reported revision.",
    validation: {
      status: "confirmed",
      summary: "Compared revision and cached value updates on every branch.",
    },
    context_ids: ["rrctx_smoke_4"],
    models: ["review"],
    observation_count: 1,
    status: "open",
    issue_draft_id: repositoryReviewIssueTwoID,
    repository_finding_id: "rrf_smoke_4",
    repository_match_state: "known",
    run_finding_status: "associated_new",
    version: 2,
    created_at: "2026-08-26T12:04:00Z",
    updated_at: "2026-08-26T12:04:00Z",
    raw_source_total: 1,
  },
  {
    id: repositoryReviewAttentionFindingID,
    fingerprint: "sha256:attention-cache-generation-race",
    repository: "octo/repo",
    commit_sha: repositoryReviewCommitSHA,
    file: {
      path: "pkg/cache/refresh_coordinator_with_a_long_operational_name.go",
      blob_sha: "f".repeat(40),
      size_bytes: 12_288,
      category: "hotpath-code",
    },
    line: 287,
    severity: "high",
    title:
      "Concurrent refresh completion can publish an expired cache generation after a newer generation commits",
    symbol: "RefreshCoordinator.publishCompletedGeneration",
    message: "The generation comparison happens before the final cache swap.",
    evidence:
      "A delayed refresh resumes after a newer generation commits and replaces the current pointer with an expired snapshot because the final swap is not fenced.",
    impact:
      "Readers can observe credentials and routing metadata from an expired cache generation.",
    validation: {
      status: "confirmed",
      summary: "Traced the stale generation through the unlocked publish path.",
      checks: ["Compared generation fences", "Verified the stale final swap"],
    },
    context_ids: ["rrctx_smoke_attention"],
    models: ["review"],
    observation_count: 1,
    status: "open",
    repository_finding_id: repositoryReviewProvisionalID,
    repository_match_state: "provisional",
    run_finding_status: "needs_review",
    version: 2,
    created_at: "2026-08-26T12:04:15Z",
    updated_at: "2026-08-26T12:04:45Z",
    raw_source_total: 1,
  },
  {
    id: repositoryReviewCandidateFindingID,
    fingerprint: "sha256:candidate-cache-generation-race",
    repository: "octo/repo",
    commit_sha: "9".repeat(40),
    file: {
      path: "pkg/cache/generation_store.go",
      blob_sha: "9".repeat(40),
      size_bytes: 9_216,
      category: "hotpath-code",
    },
    line: 164,
    severity: "high",
    title: "Delayed cache refresh overwrites a newer generation",
    symbol: "GenerationStore.Replace",
    message: "A stale refresh completion is allowed to replace the pointer.",
    evidence:
      "The completion path swaps the cached generation without comparing the generation committed while refresh work was in flight.",
    impact: "Readers can observe an expired cache snapshot.",
    validation: {
      status: "confirmed",
      summary: "Verified the missing generation fence at the final swap.",
    },
    context_ids: ["rrctx_smoke_candidate"],
    models: ["review"],
    observation_count: 4,
    status: "open",
    repository_finding_id: repositoryReviewCandidateID,
    repository_match_state: "known",
    run_finding_status: "associated_existing",
    version: 4,
    created_at: "2026-08-25T16:20:00Z",
    updated_at: "2026-08-25T16:20:00Z",
    raw_source_total: 1,
  },
]

const repositoryFindingsFixture: RepositoryFinding[] = [
  ...repositoryReviewFindingsFixture.slice(0, 4).map((finding, index) => ({
    id: `rrf_smoke_${index + 1}`,
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
    issue: finding.issue_draft_id
      ? {
          state: "draft",
          origin: "ai_generated",
          title: finding.title,
          snapshot_at: finding.updated_at,
        }
      : { state: "none" },
    validation_state: "not_requested",
    version: 1,
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  })),
  {
    id: repositoryReviewNormalAggregateID,
    repository: "octo/repo",
    canonical_title: "Request retries can repeat a committed ledger update",
    canonical_severity: "medium",
    review_finding_ids: [
      "rdf_smoke_normal_1",
      "rdf_smoke_normal_2",
      "rdf_smoke_normal_3",
    ],
    occurrence_count: 3,
    found_commits: ["1".repeat(40), "2".repeat(40)],
    found_commit_count: 2,
    path_symbol_history: [
      {
        review_finding_id: "rdf_smoke_normal_1",
        commit_sha: "1".repeat(40),
        path: "pkg/repoaudit/ledger_writer.go",
        symbol: "LedgerWriter.Commit",
        observed_at: "2026-08-24T09:00:00Z",
      },
      {
        review_finding_id: "rdf_smoke_normal_3",
        commit_sha: "2".repeat(40),
        path: "pkg/repoaudit/transaction_writer.go",
        symbol: "TransactionWriter.Commit",
        observed_at: "2026-08-26T11:30:00Z",
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: { state: "none" },
    validation_state: "not_requested",
    version: 4,
    created_at: "2026-08-24T09:00:00Z",
    updated_at: "2026-08-26T11:30:00Z",
  },
  {
    id: repositoryReviewProvisionalID,
    repository: "octo/repo",
    canonical_title:
      "Concurrent refresh completion can publish an expired cache generation after a newer generation commits",
    canonical_severity: "high",
    review_finding_ids: [repositoryReviewAttentionFindingID],
    occurrence_count: 1,
    found_commits: [repositoryReviewCommitSHA],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: repositoryReviewAttentionFindingID,
        commit_sha: repositoryReviewCommitSHA,
        path: "pkg/cache/refresh_coordinator_with_a_long_operational_name.go",
        symbol: "RefreshCoordinator.publishCompletedGeneration",
        observed_at: "2026-08-26T12:04:15Z",
      },
    ],
    match_state: "provisional",
    lifecycle: "open",
    issue: { state: "none" },
    validation_state: "not_requested",
    possible_duplicates: [
      {
        candidate_id: repositoryReviewCandidateID,
        relation: "uncertain",
        confidence: 0.94,
        matching_anchors: [
          "generation fence",
          "final cache pointer swap",
          "delayed refresh completion",
        ],
        conflicting_anchors: ["different refresh entry point"],
        explanation:
          "Both diagnoses describe an older cache generation replacing a newer committed generation after asynchronous refresh work resumes.",
        created_at: "2026-08-26T12:04:45Z",
      },
    ],
    version: 2,
    created_at: "2026-08-26T12:04:15Z",
    updated_at: "2026-08-26T12:04:45Z",
  },
  {
    id: repositoryReviewCandidateID,
    repository: "octo/repo",
    canonical_title: "Delayed cache refresh overwrites a newer generation",
    canonical_severity: "high",
    review_finding_ids: [
      "rdf_smoke_candidate_1",
      "rdf_smoke_candidate_2",
      "rdf_smoke_candidate_3",
      repositoryReviewCandidateFindingID,
    ],
    occurrence_count: 4,
    found_commits: ["7".repeat(40), "8".repeat(40), "9".repeat(40)],
    found_commit_count: 3,
    path_symbol_history: [
      {
        review_finding_id: repositoryReviewCandidateFindingID,
        commit_sha: "9".repeat(40),
        path: "pkg/cache/generation_store.go",
        symbol: "GenerationStore.Replace",
        observed_at: "2026-08-25T16:20:00Z",
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: { state: "none" },
    validation_state: "not_requested",
    version: 7,
    created_at: "2026-08-20T10:00:00Z",
    updated_at: "2026-08-25T16:20:00Z",
  },
  {
    id: repositoryReviewCombinedAttentionID,
    repository: "octo/repo",
    canonical_title: "Combined attention remains usable on narrow screens",
    canonical_severity: "critical",
    review_finding_ids: ["rdf_smoke_combined_attention"],
    occurrence_count: 1,
    found_commits: ["6".repeat(40)],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: "rdf_smoke_combined_attention",
        commit_sha: "6".repeat(40),
        path: "pkg/review/attention_projection.go",
        symbol: "AttentionProjection.Render",
        observed_at: "2026-08-26T12:04:50Z",
      },
    ],
    match_state: "provisional",
    lifecycle: "resolution_pending",
    issue: {
      state: "open",
      conflict: true,
      conflict_urls: [
        "https://github.com/octo/repo/issues/101",
        "https://github.com/octo/repo/issues/102",
      ],
      snapshot_at: "2026-08-26T12:04:50Z",
    },
    validation_state: "failed",
    version: 3,
    created_at: "2026-08-26T12:04:40Z",
    updated_at: "2026-08-26T12:04:50Z",
  },
  {
    id: repositoryReviewConflictID,
    repository: "octo/repo",
    canonical_title: "Merged occurrences reference different GitHub issues",
    canonical_severity: "high",
    review_finding_ids: ["rdf_smoke_conflict_1", "rdf_smoke_conflict_2"],
    occurrence_count: 2,
    found_commits: ["3".repeat(40), "4".repeat(40)],
    found_commit_count: 2,
    path_symbol_history: [
      {
        review_finding_id: "rdf_smoke_conflict_2",
        commit_sha: "4".repeat(40),
        path: "pkg/issues/association.go",
        symbol: "Association.Merge",
        observed_at: "2026-08-26T11:50:00Z",
      },
    ],
    match_state: "known",
    lifecycle: "open",
    issue: {
      state: "open",
      conflict: true,
      conflict_urls: [
        "https://github.com/octo/repo/issues/81",
        "https://github.com/octo/repo/issues/94",
      ],
      snapshot_at: "2026-08-26T11:55:00Z",
    },
    validation_state: "not_requested",
    version: 3,
    created_at: "2026-08-25T11:00:00Z",
    updated_at: "2026-08-26T11:55:00Z",
  },
  {
    id: repositoryReviewFailedCheckID,
    repository: "octo/repo",
    canonical_title: "Cleanup leaves a stale workspace lease behind",
    canonical_severity: "medium",
    review_finding_ids: ["rdf_smoke_failed_check"],
    occurrence_count: 1,
    found_commits: ["5".repeat(40)],
    found_commit_count: 1,
    path_symbol_history: [
      {
        review_finding_id: "rdf_smoke_failed_check",
        commit_sha: "5".repeat(40),
        path: "pkg/workspace/cleanup.go",
        symbol: "Cleanup.Release",
        observed_at: "2026-08-26T12:00:00Z",
      },
    ],
    match_state: "new",
    lifecycle: "resolution_pending",
    issue: {
      state: "closed",
      snapshot_at: "2026-08-26T12:01:00Z",
    },
    validation_state: "failed",
    version: 5,
    created_at: "2026-08-26T10:00:00Z",
    updated_at: "2026-08-26T12:01:00Z",
  },
]

const repositoryReviewContextsFixture: RepositoryReviewFindingContext[] =
  repositoryReviewFindingsFixture.map((finding, index) => ({
    id: finding.context_ids[0]!,
    repository: finding.repository,
    commit_sha: finding.commit_sha,
    inventory_hash: "sha256:inventory-smoke",
    profile_hash: "sha256:profile-smoke",
    run_id: index < 2 ? "rrun_smoke_previous" : "rrun_smoke_current",
    model: finding.models[0]!,
    reviewer: `review-child-${index + 1}`,
    files: [finding.file],
    raw_digest: `sha256:raw-context-${index + 1}`,
    created_at: finding.created_at,
  }))

const repositoryReviewRawFindingsFixture: RepositoryReviewRawFinding[] = [
  repositoryReviewRawFindingFixture(
    "rrw_smoke_1",
    repositoryReviewFindingsFixture[0]!,
    "review",
    "review-child-1",
    "new",
    1,
  ),
  repositoryReviewRawFindingFixture(
    "rrw_smoke_2",
    repositoryReviewFindingsFixture[0]!,
    "code",
    "code-child-2",
    "duplicate",
    2,
  ),
  ...repositoryReviewFindingsFixture
    .slice(1, 4)
    .map((finding, index) =>
      repositoryReviewRawFindingFixture(
        `rrw_smoke_${index + 3}`,
        finding,
        finding.models[0]!,
        `review-child-${index + 2}`,
        "new",
        index + 3,
      ),
    ),
]

const repositoryReviewProcessingSourcesFixture: RepositoryReviewRawFinding[] = [
  repositoryReviewProcessingSourceFixture(
    repositoryReviewProcessingPendingID,
    repositoryReviewRawFindingsFixture[0]!,
    "pending",
    21,
  ),
  repositoryReviewProcessingSourceFixture(
    repositoryReviewProcessingRunningID,
    repositoryReviewRawFindingsFixture[2]!,
    "running",
    22,
  ),
  repositoryReviewProcessingSourceFixture(
    repositoryReviewProcessingFailedID,
    repositoryReviewRawFindingsFixture[3]!,
    "failed",
    23,
    {
      code: "attempt_limit",
      message: "Finding grouping reached its retry limit.",
      retryable: true,
      at: "2026-08-26T12:05:10Z",
    },
  ),
  repositoryReviewProcessingSourceFixture(
    repositoryReviewProcessingCompletedID,
    repositoryReviewRawFindingsFixture[1]!,
    "completed",
    24,
  ),
  repositoryReviewProcessingSourceFixture(
    repositoryReviewProcessingOldCampaignID,
    repositoryReviewRawFindingsFixture[4]!,
    "failed",
    25,
    {
      code: "processing_interrupted",
      message: "Historical finding grouping was interrupted.",
      retryable: true,
      at: "2026-08-25T17:00:00Z",
    },
    "rrc_smoke_previous",
  ),
  ...Array.from({ length: 52 }, (_, index) => ({
    ...repositoryReviewProcessingSourceFixture(
      `rrw_smoke_processing_completed_${index + 1}`,
      repositoryReviewRawFindingsFixture[0]!,
      "completed",
      30 + index,
    ),
    title: `Completed diagnosis ${index + 1}`,
    updated_at: `2026-08-26T11:30:${String(index).padStart(2, "0")}Z`,
  })),
]

function repositoryReviewRawFindingFixture(
  id: string,
  finding: RepositoryReviewFinding,
  model: string,
  reviewer: string,
  disposition: "new" | "duplicate",
  ordinal: number,
): RepositoryReviewRawFinding {
  return {
    id,
    version: 1,
    campaign_id: "rrc_smoke",
    admission_bucket: `rdb_smoke_${ordinal}`,
    insertion_ordinal: ordinal,
    diagnosis_digest: `sha256:${id}`,
    ...(id === "rrw_smoke_1" ? { legacy_finding_id: "rfn_smoke_legacy" } : {}),
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
    run_id: ordinal < 3 ? "rrun_smoke_previous" : "rrun_smoke_current",
    assignment_id: `assignment-${ordinal}`,
    model,
    reviewer,
    deduplication_state: "completed",
    disposition,
    deduplicated_finding_id: finding.id,
    history: [
      {
        state: "completed",
        disposition,
        deduplicated_finding_id: finding.id,
        attempt: 1,
        at: finding.updated_at,
      },
    ],
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }
}

function repositoryReviewProcessingSourceFixture(
  id: string,
  source: RepositoryReviewRawFinding,
  processingState: RepositoryReviewRawFinding["deduplication_state"],
  ordinal: number,
  failure?: RepositoryReviewRawFinding["failure"],
  campaignID = "rrc_smoke",
): RepositoryReviewRawFinding {
  const completed = processingState === "completed"
  return {
    ...structuredClone(source),
    id,
    version: 1,
    campaign_id: campaignID,
    insertion_ordinal: ordinal,
    diagnosis_digest: `sha256:${id}`,
    legacy_finding_id: undefined,
    assignment_id: `assignment-processing-${ordinal}`,
    model_alias: source.model,
    account: "openai-primary",
    deduplication_state: processingState,
    disposition: completed ? source.disposition : "undecided",
    deduplicated_finding_id: completed
      ? source.deduplicated_finding_id
      : undefined,
    history: [
      {
        state: processingState,
        disposition: completed ? source.disposition : "undecided",
        deduplicated_finding_id: completed
          ? source.deduplicated_finding_id
          : undefined,
        attempt: processingState === "pending" ? undefined : 1,
        failure,
        at: failure?.at ?? source.updated_at,
      },
    ],
    failure,
    updated_at: failure?.at ?? source.updated_at,
  }
}

const repositoryReviewIssuesFixture: RepositoryReviewIssueDraft[] = [
  {
    id: repositoryReviewIssueOneID,
    repository: "octo/repo",
    finding_ids: [repositoryReviewFindingTwoID],
    origin: "ai_generated",
    generation_id: "rig_smoke_existing",
    resolved_instructions:
      "Write a concise grounded issue with evidence, impact, validation, location, and commit provenance.",
    instructions_mode: "default",
    generator_model: "issue-writer",
    generator_account: "openai-primary",
    generator_profile_id: "rrpf_smoke",
    generator_profile_version: 3,
    canonical: true,
    publishable: true,
    deletable: true,
    regeneratable: true,
    title: "Archive extraction can write outside the workspace",
    body: [
      "## Evidence",
      "",
      "A `../` archive entry reaches `os.Create` without a destination-root check.",
      "",
      "| Location | Commit |",
      "| --- | --- |",
      `| \`pkg/archive/extract.go:88\` | \`${repositoryReviewCommitSHA}\` |`,
      "",
      "## Impact",
      "",
      "A crafted archive can overwrite files outside the workspace.",
      "",
      "## Validation",
      "",
      "- Normalized an escaping entry",
      "- Followed it to `os.Create`",
    ].join("\n"),
    labels: ["bug", "security"],
    state: "editing",
    version: 3,
    created_at: "2026-08-26T12:03:00Z",
    updated_at: "2026-08-26T12:03:00Z",
  },
  {
    id: repositoryReviewIssueTwoID,
    repository: "octo/repo",
    finding_ids: [repositoryReviewFindingFourID],
    origin: "ai_generated",
    generation_id: "rig_smoke_existing",
    resolved_instructions:
      "Write a concise grounded issue with evidence, impact, validation, location, and commit provenance.",
    instructions_mode: "default",
    generator_model: "issue-writer",
    generator_account: "openai-primary",
    generator_profile_id: "rrpf_smoke",
    generator_profile_version: 3,
    canonical: true,
    publishable: true,
    deletable: true,
    regeneratable: true,
    title: "Failed reload leaves the configuration cache stale",
    body: "## Evidence\n\nThe error branch preserves the prior cached revision.\n\n## Impact\n\nReaders can observe stale configuration.",
    labels: ["bug"],
    state: "editing",
    version: 2,
    created_at: "2026-08-26T12:04:30Z",
    updated_at: "2026-08-26T12:04:30Z",
  },
]

function repositoryReviewRunFindingSummary(finding: RepositoryReviewFinding) {
  const runStatus = finding.run_finding_status ?? "pending"
  const association =
    runStatus === "associated_new"
      ? "new"
      : runStatus === "associated_existing"
        ? "existing"
        : runStatus === "needs_review"
          ? "needs_review"
          : "unassociated"
  return {
    id: finding.id,
    repository: finding.repository,
    path: finding.file.path,
    line: finding.line,
    severity: finding.severity,
    title: finding.title,
    symbol: finding.symbol,
    status: finding.status,
    run_finding_status: runStatus,
    association,
    repository_finding_id: finding.repository_finding_id,
    contributors: [
      ...new Set([
        ...(finding.observations ?? []).map(
          (observation) => observation.reviewer || observation.model,
        ),
        ...finding.models,
      ]),
    ],
    raw_source_count: finding.raw_source_total ?? 0,
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }
}

function repositoryFindingCollectionSummary(finding: RepositoryFinding) {
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
      url: finding.issue.url,
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
}

function repositoryReviewIssueCollectionSummary(
  issue: RepositoryReviewIssueDraft,
) {
  return {
    id: issue.id,
    repository: issue.repository,
    finding_count: issue.finding_ids.length,
    origin: issue.origin ?? "legacy",
    generation_id: issue.generation_id,
    canonical: issue.canonical ?? false,
    publishable: issue.publishable ?? false,
    title: issue.title,
    state: issue.state,
    version: issue.version,
    created_at: issue.created_at,
    updated_at: issue.updated_at,
  }
}

interface RepositoryReviewMockState {
  automation: RepositoryReviewAutomation
  summary: RepositoryReviewSummary
  findings: RepositoryReviewFinding[]
  rawFindings: RepositoryReviewRawFinding[]
  processingSources: RepositoryReviewRawFinding[]
  repositoryFindings: RepositoryFinding[]
  contexts: RepositoryReviewFindingContext[]
  issues: RepositoryReviewIssueDraft[]
  historicalConsolidation: RepositoryReviewHistoricalConsolidation
  healthReads: number
  stagedHealth: boolean
}

function createRepositoryReviewMockState(): RepositoryReviewMockState {
  return {
    automation: structuredClone(repositoryReviewAutomationFixture),
    summary: structuredClone(repositoryReviewSummaryFixture),
    findings: structuredClone(repositoryReviewFindingsFixture),
    rawFindings: structuredClone(repositoryReviewRawFindingsFixture),
    processingSources: structuredClone(
      repositoryReviewProcessingSourcesFixture,
    ),
    repositoryFindings: structuredClone(repositoryFindingsFixture),
    contexts: structuredClone(repositoryReviewContextsFixture),
    issues: structuredClone(repositoryReviewIssuesFixture),
    historicalConsolidation: {
      required: true,
      status: "failed",
      retryable: true,
    },
    healthReads: 0,
    stagedHealth: false,
  }
}

const developmentWorkspaceID = smokeDevelopmentWorkspaceID
const developmentWorkspaceCharterID = `pcr_${"2".repeat(32)}`
const developmentWorkspaceAggregate = {
  workspace: {
    id: developmentWorkspaceID,
    intent: "pickup_pr",
    source_kind: "pull_request",
    source_id: "200",
    source_number: 42,
    provider: "github",
    provider_origin: "https://github.com",
    repository_id: "100",
    repository: "octo/repo",
    pull_request_id: "200",
    pull_number: 42,
    phase: "completion_audit",
    execution_state: "waiting_user",
    active_charter_id: developmentWorkspaceCharterID,
    provider_head_sha: "b".repeat(40),
    version: 4,
    created_at: "2026-08-13T10:00:00Z",
    updated_at: "2026-08-13T10:05:00Z",
  },
  provider_snapshot: {
    intent: "pickup_pr",
    source_kind: "pull_request",
    source_id: "200",
    source_number: 42,
    source_url: "https://github.com/octo/repo/pull/42",
    provider: "github",
    provider_origin: "https://github.com",
    repository_id: "100",
    repository: "octo/repo",
    pull_request_id: "200",
    pull_number: 42,
    title: "Fix lost updates",
    body: "Keep optimistic concurrency intact.",
    author_id: "300",
    author_login: "octocat",
    authenticated_user_id: "300",
    base_ref: "main",
    base_sha: "a".repeat(40),
    head_repository_id: "100",
    head_ref: "fix/store",
    head_sha: "b".repeat(40),
    state: "open",
    owned: true,
    head_writable: true,
    can_review: true,
    can_create_issue: true,
    provider_revision: "github-etag-4",
    observed_at: "2026-08-13T10:00:00Z",
  },
  charters: [
    {
      id: developmentWorkspaceCharterID,
      revision: 1,
      type: "fix",
      goal: "Prevent lost updates.",
      acceptance_criteria: ["Concurrent writes conflict."],
      included_areas: ["pkg/store"],
      excluded_areas: ["Broad refactor"],
      non_goals: ["New storage engine"],
      base_sha: "a".repeat(40),
      head_sha: "b".repeat(40),
      confirmed: true,
      created_at: "2026-08-13T10:01:00Z",
      confirmed_at: "2026-08-13T10:02:00Z",
    },
  ],
  stage_runs: [
    {
      id: `psr_${"3".repeat(32)}`,
      stage: "review",
      state: "succeeded",
      charter_id: developmentWorkspaceCharterID,
      head_sha: "b".repeat(40),
      attempt: 1,
      summary: "Review completed after distinct coverage challenges.",
      started_at: "2026-08-13T10:03:00Z",
      finished_at: "2026-08-13T10:03:01Z",
    },
    {
      id: `psr_${"6".repeat(32)}`,
      stage: "implementation",
      state: "succeeded",
      charter_id: developmentWorkspaceCharterID,
      head_sha: "b".repeat(40),
      attempt: 1,
      summary: "Implemented the confirmed charter.",
      started_at: "2026-08-13T10:03:30Z",
      finished_at: "2026-08-13T10:03:59Z",
    },
    {
      id: `psr_${"4".repeat(32)}`,
      stage: "completion_audit",
      state: "succeeded",
      charter_id: developmentWorkspaceCharterID,
      head_sha: "b".repeat(40),
      attempt: 1,
      summary: "Implementation is complete within the charter.",
      started_at: "2026-08-13T10:04:00Z",
      finished_at: "2026-08-13T10:04:01Z",
    },
  ],
  findings: [],
  messages: [],
  corrections: [],
  repository_lessons: [],
  nudge_rounds: [
    {
      id: `pnr_${"5".repeat(32)}`,
      stage_run_id: `psr_${"3".repeat(32)}`,
      stage: "review",
      round: 1,
      minimum_rounds: 2,
      hard_cap: 5,
      strategy: "coverage_gaps",
      challenge: "Inspect unchecked callers.",
      variant_digest: "sha256:variant",
      prompt_digest: "sha256:prompt",
      state: "succeeded",
      novel_findings: 0,
      duplicate_count: 0,
      resolved_findings: 0,
      reward: 0.25,
      reward_provenance: "retained_open",
      created_at: "2026-08-13T10:04:00Z",
    },
  ],
  deferred_groups: [],
  repair_attempts: [
    {
      id: `pra_${"6".repeat(32)}`,
      stage_run_id: `psr_${"6".repeat(32)}`,
      number: 1,
      state: "succeeded",
      instruction: "Implement the confirmed charter.",
      candidate_sha: "c".repeat(40),
      scope: {
        distance: "S0_exact",
        size: "XS",
        presence: "candidate_present",
        files: 1,
        semantic_lines: 5,
        modules: 1,
        estimated: false,
        type_compatible: true,
        confidence: 1,
      },
      prompt_digest: "sha256:repair",
      started_at: "2026-08-13T10:03:30Z",
      finished_at: "2026-08-13T10:03:45Z",
    },
  ],
  validation_runs: [
    {
      id: `pvr_${"6".repeat(32)}`,
      stage_run_id: `psr_${"6".repeat(32)}`,
      state: "succeeded",
      candidate_sha: "c".repeat(40),
      checks: [{ id: "tests", name: "Tests", status: "passed" }],
      started_at: "2026-08-13T10:03:45Z",
      finished_at: "2026-08-13T10:03:50Z",
    },
  ],
  gates: [],
  publications: [],
  activity: [],
}

const developmentWorkspaceCollectionResponse = {
  workspaces: [
    {
      id: developmentWorkspaceID,
      intent: "pickup_pr",
      source: "pull_request",
      repository: "octo/repo",
      title: "Pull request #42",
      phase: "completion_audit",
      execution_state: "waiting_user",
      created: "2026-08-13T10:00:00Z",
      updated: "2026-08-13T10:05:00Z",
    },
    {
      id: `devw_${"8".repeat(32)}`,
      intent: "implement_feature",
      source: "issue",
      repository: "octo/launcher",
      title: "Issue #81",
      phase: "implementation",
      execution_state: "running",
      created: "2026-08-12T09:00:00Z",
      updated: "2026-08-13T09:45:00Z",
    },
  ],
  total: 2,
  next_cursor: "",
  canonical_query: "ORDER BY updated DESC",
  query_schema: mockCollectionSchemas.developmentWorkspaces,
}

const developmentNotificationID = `dnt_${"7".repeat(32)}`
const developmentNotification = {
  id: developmentNotificationID,
  source_key: `${developmentWorkspaceID}:publication_approval:gate-1`,
  generation: 1,
  workspace_id: developmentWorkspaceID,
  repository: "octo/repo",
  intent: "pickup_pr",
  source_kind: "pull_request",
  phase: "publication",
  reason: "publication_approval",
  priority: "high",
  status: "open",
  read: false,
  title: "Publication approval needed",
  summary: "Review the exact branch publication before it is pushed.",
  target: { panel: "publication", entity_id: "gate-1" },
  version: 1,
  created_at: "2026-08-24T10:00:00Z",
  updated_at: "2026-08-24T10:05:00Z",
}

const notificationViews = {
  version: 1,
  views: [],
}

const prLifecycleWorkflowConfigurations = {
  "workflow-configurations": {
    default: {
      name: "Default",
      bindings: [],
      "deferred-issues": { mode: "ask" },
      "scope-disposition": {
        default: { mode: "strict", prompt: "" },
        "by-type": {},
      },
    },
    editable: {
      name: "Editable",
      bindings: [],
      "deferred-issues": { mode: "ask" },
      "scope-disposition": {
        default: { mode: "relaxed", prompt: "Keep changes relevant." },
        "by-type": {},
      },
    },
  },
  "default-workflow-configuration": "default",
  nudge: {
    "review-minimum-additional": 2,
    "review-maximum-additional": 5,
    "completion-minimum-additional": 2,
    "completion-maximum-additional": 5,
  },
  scope: {
    xs: { files: 1, "semantic-lines": 20, modules: 1 },
    s: { files: 3, "semantic-lines": 100, modules: 1 },
    m: { files: 10, "semantic-lines": 500, modules: 3 },
  },
  "gate-catalog": Object.fromEntries(
    prLifecycleFlowFixture.flow.flows.flatMap((flow) =>
      flow.nodes.flatMap((node) =>
        node.decision_point
          ? [
              [
                node.decision_point,
                {
                  "workflow-ref": "workflows/pr-lifecycle.yml",
                  "gate-ref": `gates.${node.decision_point.replace(/^pr\./, "").replaceAll(".", "-")}`,
                  "source-ai-supported":
                    node.decision_point === "pr.finding.classify",
                  prompt: `Complete ${node.title}.`,
                  fields: [
                    {
                      id: "action",
                      type: "select",
                      label: "What should happen?",
                      "min-selections": 1,
                      "max-selections": 1,
                      options: [
                        { id: "approve", label: "Approve" },
                        { id: "revise", label: "Request revision" },
                      ],
                    },
                  ],
                  "workflow-revision": "workflow-revision-1",
                  "default-action": { type: "human" },
                  "effective-action": { type: "human" },
                  "action-source": "workflow-default",
                },
              ] as const,
            ]
          : [],
      ),
    ),
  ),
  flow: prLifecycleFlowFixture.flow,
  "flow-revision": prLifecycleFlowFixture.flow_revision,
  "catalog-revision": "sha256:catalog",
  "config-revision": "sha256:config",
  effects: {
    "gateway-effect": "applied",
    "deferred-policy-effect": "applied",
  },
}

const prLifecycleRepositoryAssignments = {
  repositories: {
    "https://github.com|100": {
      name: "octo/repo",
      "default-branch": "main",
    },
  },
  "workflow-configurations": {
    default: {
      name: "Default",
      "deferred-issues": { mode: "ask" },
    },
    editable: {
      name: "Editable",
      "deferred-issues": { mode: "ask" },
    },
  },
  "default-workflow-configuration": "default",
  "repository-assignments": {},
  "config-revision": "sha256:config",
  effects: {
    "gateway-effect": "applied",
    "deferred-policy-effect": "applied",
  },
}

const smokeDevelopmentRepositoryAssignmentID =
  "Rjljc2epaibQOt_BhFLZFLSNQFrkJGFxU2BnbKKqal8"

const prLifecycleRepositoryAssignmentCollection = {
  repository_assignments: [
    {
      id: smokeDevelopmentRepositoryAssignmentID,
      repository: "octo/repo",
      configuration: "default",
      default_branch: "main",
    },
  ],
  total: 1,
  next_cursor: "",
  canonical_query: "ORDER BY repository ASC",
  query_schema: {
    ...mockCollectionSchemas.developmentRepositoryAssignments,
    default_order: [{ field: "repository", direction: "ASC" }],
  },
  config_revision: "sha256:config",
  effects: {
    gateway_effect: "applied",
    deferred_policy_effect: "applied",
  },
}

const prLifecycleWorkflowConfigurationCollection = {
  workflow_configurations: [
    {
      id: "default",
      name: "Default",
      is_default: true,
      bindings: 0,
      deferred_issues: "ask",
    },
    {
      id: "editable",
      name: "Editable",
      is_default: false,
      bindings: 0,
      deferred_issues: "ask",
    },
  ],
  total: 2,
  next_cursor: "",
  canonical_query: "ORDER BY name ASC",
  query_schema: {
    ...mockCollectionSchemas.developmentWorkflowConfigurations,
    default_order: [{ field: "name", direction: "ASC" }],
  },
  config_revision: "sha256:config",
  effects: {
    gateway_effect: "applied",
    deferred_policy_effect: "applied",
  },
}

const prLifecycleCollectionConfigurationChoices = {
  default: { name: "Default", deferred_issues: { mode: "ask" } },
  editable: { name: "Editable", deferred_issues: { mode: "ask" } },
}

const prLifecycleRepositoryAssignmentDetail = {
  repository_assignment: {
    ...prLifecycleRepositoryAssignmentCollection.repository_assignments[0],
    provider_origin: "https://github.com",
    repository_id: "100",
  },
  workflow_configurations: prLifecycleCollectionConfigurationChoices,
  config_revision: "sha256:config",
  effects: prLifecycleRepositoryAssignmentCollection.effects,
}

function prLifecycleWorkflowConfigurationDetail(id: "default" | "editable") {
  const source =
    prLifecycleWorkflowConfigurations["workflow-configurations"][id]
  const snakeGateCatalog = Object.fromEntries(
    Object.entries(prLifecycleWorkflowConfigurations["gate-catalog"]).map(
      ([decisionPoint, entry]) => [
        decisionPoint,
        {
          workflow_ref: entry["workflow-ref"],
          gate_ref: entry["gate-ref"],
          source_ai_supported: entry["source-ai-supported"],
          prompt: entry.prompt,
          fields: entry.fields.map((field) => ({
            id: field.id,
            type: field.type,
            label: field.label,
            required: false,
            min_selections: field["min-selections"],
            max_selections: field["max-selections"],
            options: field.options,
          })),
          workflow_revision: entry["workflow-revision"],
          default_action: entry["default-action"],
          effective_action: entry["effective-action"],
          action_source: entry["action-source"],
        },
      ],
    ),
  )
  return {
    workflow_configuration: {
      id,
      name: source.name,
      is_default: id === "default",
      bindings: source.bindings,
      deferred_issues: source["deferred-issues"],
      scope_disposition: {
        default: source["scope-disposition"].default,
        by_type: source["scope-disposition"]["by-type"],
      },
    },
    gate_catalog: snakeGateCatalog,
    flow: prLifecycleWorkflowConfigurations.flow,
    flow_revision: prLifecycleWorkflowConfigurations["flow-revision"],
    catalog_revision: prLifecycleWorkflowConfigurations["catalog-revision"],
    config_revision: prLifecycleWorkflowConfigurations["config-revision"],
    effects: prLifecycleWorkflowConfigurationCollection.effects,
  }
}

interface MockLauncherApiOptions {
  accounts?: AccountSummary[]
  accountRouters?: AccountRouter[]
  eventSources?: MockEventSource[]
  eventSourceBulkFailureIDs?: string[]
  eventSourceRequests?: Array<{
    method: string
    path: string
    body: Record<string, unknown> | null
  }>
  agentActivityRequests?: Array<{ method: string; path: string }>
  agentCapabilityRequests?: Array<{
    method: string
    path: string
    body: unknown
  }>
  agentRequests?: Array<{
    method: string
    path: string
    body: unknown
  }>
  completeDraftViaPolling?: boolean
  codexAccountLimits?: unknown
  fetchModelEmptyCredentials?: string[]
  fetchModelFailures?: Record<string, string>
  modelResponse?: unknown
  repositoryReviewAutomationOptions?: unknown
  repositoryReviewFailed?: boolean
  repositoryReviewStagedHealth?: boolean
  repositoryReviewTerminal?: boolean
  repositoryReviewRequests?: Array<{
    method: string
    path: string
    body: Record<string, unknown> | null
  }>
  modelEvaluationRequests?: Array<{
    method: string
    path: string
    body: Record<string, unknown> | null
  }>
  statefulModelEvaluations?: boolean
  nullableWorkflowPayloads?: boolean
  oauthProviders?: OAuthProviderStatus[]
  statefulAgents?: boolean
  statefulMCP?: boolean
  gatewayRunning?: boolean
  mcpRequests?: Array<{
    method: string
    path: string
    body: unknown
  }>
  workflowInspectionRequests?: Array<{
    method: string
    path: string
    body: unknown
  }>
  workflowCapabilityRequests?: Array<{
    method: string
    path: string
  }>
  workflowJobRequests?: Array<{
    method: string
    path: string
    body: unknown
  }>
  workflowDevelopmentYAML?: string
  workflowTriggerSimulationRequests?: Array<{
    method: string
    path: string
    body: Record<string, unknown>
  }>
  workflowTriggerExecutionRequests?: Array<{
    method: string
    path: string
    body: Record<string, unknown>
  }>
  workflowEventPayloadRequests?: string[]
  workflowCancelReasons?: string[]
  gitWorkspaceRequests?: Array<{
    method: string
    path: string
    body: Record<string, unknown> | null
  }>
}

async function mockLauncherApis(
  page: Page,
  options: MockLauncherApiOptions = {},
) {
  const currentOAuthProviders = structuredClone(
    options.oauthProviders ?? defaultOAuthProviders,
  )
  const currentAccounts = structuredClone(
    options.accounts ??
      (options.oauthProviders
        ? accountsFromOAuthProviders(options.oauthProviders)
        : defaultSmokeAccounts),
  )
  const currentAccountRouters = structuredClone(
    options.accountRouters ?? defaultSmokeAccountRouters,
  )
  const currentEventSources = structuredClone(
    options.eventSources ?? defaultSmokeEventSources,
  )
  let currentEventSourceSettings = structuredClone(defaultEventSourceSettings)
  let currentEventSourceRevision = 1
  const currentSkills = structuredClone(skillsResponse.skills)
  const currentTools = structuredClone(toolsResponse.tools)
  const currentGitWorkspaceDetails = structuredClone(gitWorkspaceDetails)
  let currentGitWorkspaceSettings = structuredClone(
    gitWorkspaceSettingsResponse,
  )
  let activeDevelopmentSession: MockWorkflowDevelopmentSession | null = null
  let workflowDefinitions = [
    {
      ref: "workflows/summarize-text.yml",
      name: "Summarize text",
    },
  ]
  let runs = options.nullableWorkflowPayloads
    ? [nullableWorkflowRun]
    : [workflowRun]
  let workflowsRevalidated = false
  let completeDraftViaPolling = false
  let reviseRequestCount = 0
  let currentMCPResponse = structuredClone(mcpResponse)
  let currentCancelableWorkflowRun = structuredClone(cancelableWorkflowRun)
  let currentAgentRevision = 1
  let currentCapabilityRevision = 1
  let currentDefaultAgentID = "main"
  const repositoryReviewState = createRepositoryReviewMockState()
  repositoryReviewState.stagedHealth =
    options.repositoryReviewStagedHealth ?? false
  if (options.repositoryReviewFailed) {
    repositoryReviewState.automation.status = "failed"
    repositoryReviewState.automation.active_run_id = undefined
    repositoryReviewState.automation.pause_detail =
      "The reviewer provider stopped after repeated safe failures."
  }
  if (options.repositoryReviewTerminal) {
    repositoryReviewState.automation.status = "completed"
    repositoryReviewState.automation.active_run_id = undefined
    repositoryReviewState.processingSources =
      repositoryReviewState.processingSources.map((source) => ({
        ...source,
        deduplication_state: "completed",
        disposition: source.disposition === "duplicate" ? "duplicate" : "new",
        deduplicated_finding_id:
          source.deduplicated_finding_id ?? repositoryReviewFindingOneID,
        failure: undefined,
      }))
    repositoryReviewState.historicalConsolidation = {
      required: false,
      status: "completed",
      retryable: false,
    }
  }
  let currentModelEvaluation: Record<string, unknown> | null = null
  let modelEvaluationDetailReads = 0
  const modelEvaluationProfile = {
    id: "rrpf_model_probe",
    version: 3,
    name: "Production bugs",
    reviewer_model: "code",
    account_ref: "",
    review_focus: "Find concrete correctness and reliability defects.",
    focus: {
      code_types: ["hotpath-code", "code", "test", "bench-test"],
      include_folders: ["pkg", "cmd"],
      exclude_folders: [],
      free_text: "",
    },
    max_files_per_batch: 8,
    max_content_bytes_per_batch: 131_072,
    max_parallel_children: 3,
  }
  const modelSizingResult = (
    pointID: string,
    axis: "files_per_batch" | "content_bytes_per_batch",
    modelAlias: string,
    files: number,
    contentBytes: number,
    score: number,
  ) => ({
    point_id: pointID,
    axis,
    model_alias: modelAlias,
    completion: "completed",
    files_per_batch: files,
    content_bytes_per_batch: contentBytes,
    batch_samples: 2,
    files_analyzed: 2,
    bytes_analyzed: 12_000,
    attempts: 2,
    successes: 2,
    failures: 0,
    observed_min_files_per_batch: files,
    observed_max_files_per_batch: files,
    observed_mean_files_per_batch: files,
    observed_min_content_bytes_per_batch: contentBytes,
    observed_max_content_bytes_per_batch: contentBytes,
    observed_mean_content_bytes_per_batch: contentBytes,
    scores: {
      overall: {
        samples: 2,
        weighted_mean: score,
        minimum: score - 1,
        maximum: score + 1,
        standard_deviation: 0.5,
      },
    },
    confirmed_findings: 4,
    unsupported_claims: 1,
    usage: {
      requests: 2,
      input_tokens: 2_000,
      cached_input_tokens: 1_000,
      output_tokens: 500,
      reasoning_tokens: 100,
      duration_millis: 20_000,
    },
    effective_tokens: 1_600,
    effective_tokens_per_kib: 136.533,
  })

  const modelEvaluationFromBody = (
    body: Record<string, unknown>,
    previous?: Record<string, unknown> | null,
  ): Record<string, unknown> => ({
    schema_version: 1,
    id: previous?.id ?? `rme_${"1".repeat(32)}`,
    version: (previous?.version as number | undefined) ?? 1,
    status: previous?.status ?? "draft",
    repository: body.repository ?? previous?.repository ?? "owner/repo",
    ref: body.ref ?? previous?.ref ?? "main",
    candidate_models: body.candidate_models ??
      previous?.candidate_models ?? ["code", "fast"],
    selector_model_alias:
      body.selector_model_alias ?? previous?.selector_model_alias ?? "review",
    judge_model_alias:
      body.judge_model_alias ?? previous?.judge_model_alias ?? "review",
    focus: body.focus ??
      previous?.focus ?? {
        code_types: ["hotpath-code", "code", "test", "bench-test"],
        include_folders: [],
        exclude_folders: [],
        free_text: "",
      },
    default_files_per_language:
      body.default_files_per_language ??
      previous?.default_files_per_language ??
      20,
    files_per_language:
      body.files_per_language ?? previous?.files_per_language ?? {},
    profile: previous?.profile ?? modelEvaluationProfile,
    work_sizing_plan: previous?.work_sizing_plan ?? [
      {
        id: "configured",
        axis: "configured",
        files_per_batch: 8,
        content_bytes_per_batch: 131_072,
      },
    ],
    work_sizing_results: previous?.work_sizing_results ?? [],
    progress: previous?.progress ?? {
      stage: "idle",
      languages: {},
      total_files: 0,
      selected_files: 0,
      completed_files: 0,
      total_tasks: 0,
      completed_tasks: 0,
      percent: 0,
    },
    usage: previous?.usage ?? {
      requests: 0,
      input_tokens: 0,
      cached_input_tokens: 0,
      output_tokens: 0,
      reasoning_tokens: 0,
      duration_millis: 0,
    },
    model_stats: previous?.model_stats ?? {},
    comparisons: previous?.comparisons ?? [],
    warnings: previous?.warnings ?? [],
    run_ids: previous?.run_ids ?? [],
    created_at: previous?.created_at ?? "2026-08-21T12:00:00Z",
    updated_at: "2026-08-21T12:01:00Z",
  })
  let currentAgents: AgentInfo[] = [
    {
      id: "main",
      name: "Main",
      workspace: "",
      account_ref: "",
      model: null,
      skills: null,
      subagents: null,
      is_default: true,
      default_configured: options.statefulAgents === true,
      implicit: options.statefulAgents !== true,
    },
    {
      id: "reviewer",
      name: "Reviewer",
      workspace: "/workspace/reviewer",
      account_ref: "gpt-4o",
      model: {
        primary: "code",
        fallbacks: [],
      },
      skills: ["review-helper"],
      subagents: { allow_agents: ["main"] },
      is_default: false,
      default_configured: false,
      implicit: false,
    },
  ]

  const agentEffects = {
    launcher_effect: "applied",
    catalog_effect: "applied",
    gateway_effect: "applied",
  } as const

  let currentAgentCapabilities: AgentCapabilitiesResponse = {
    agent_id: "reviewer",
    source: "agent",
    editable: true,
    issue_code: "",
    legacy_upgrade_required: false,
    capabilities: {
      tools: {
        mode: "selected",
        values: ["web_search", "legacy_unknown"],
      },
      skills: {
        mode: "inherit",
        values: [],
        inherited_values: ["review-helper"],
      },
      mcp_servers: {
        mode: "all",
        values: [],
      },
    },
    catalogs: {
      tools: [
        {
          name: "web_search",
          description: "Search the web",
          category: "web",
          status: "enabled",
          reason_code: "",
        },
        {
          name: "filesystem",
          description: "Read approved workspace files",
          category: "workspace",
          status: "enabled",
          reason_code: "",
        },
      ],
      skills: [{ name: "review-helper", source: "workspace" }],
      mcp_servers: [{ name: "github", enabled: true }],
    },
    catalog_truncated: {
      tools: false,
      skills: false,
      mcp_servers: false,
    },
    revision: "capability-revision-1",
    config_revision: "agent-revision-1",
    effects: agentEffects,
  }

  function currentAgentsResponse(): AgentsResponse {
    return {
      agents: structuredClone(currentAgents),
      default_agent_id: currentDefaultAgentID,
      config_revision: `agent-revision-${currentAgentRevision}`,
      effects: agentEffects,
      total: currentAgents.length,
      next_cursor: "",
      canonical_query: "ORDER BY position ASC",
      query_schema: mockCollectionSchemas.agents,
    }
  }

  function advanceAgentsRevision() {
    currentAgentRevision += 1
    return currentAgentsResponse()
  }

  function eventSourceRevision() {
    return `event-source-revision-${currentEventSourceRevision}`
  }

  function eventSourceEffects() {
    return {
      launcher_effect: "applied",
      catalog_effect: "applied",
      gateway_effect: "restart_required",
    } as const
  }

  function advanceEventSourceRevision() {
    currentEventSourceRevision += 1
    return eventSourceRevision()
  }

  function mcpServerFromInput(
    input: MCPServerInput,
    existing?: MCPServer,
  ): MCPServer {
    const envKeys = input.env_keys ?? Object.keys(input.env ?? {})
    const headerKeys = input.header_keys ?? Object.keys(input.headers ?? {})
    const authMode = input.auth_mode ?? existing?.auth.type ?? "none"
    let auth = existing?.auth ?? { type: "none", configured: false }
    if (authMode === "none") {
      auth = { type: "none", configured: false }
    } else if (authMode === "custom") {
      auth = { type: "custom", configured: headerKeys.length > 0 }
    } else if (authMode !== auth.type) {
      auth = { type: authMode, configured: false }
    }
    return {
      name: input.name,
      enabled: input.enabled,
      deferred: input.deferred,
      type: input.type,
      url: input.url ?? "",
      command: input.command ?? "",
      args: input.args ?? [],
      env_file: input.env_file ?? "",
      env_keys: envKeys,
      header_keys: headerKeys,
      auth,
    }
  }

  function compatibilityResponse() {
    const stamps = workflowDefinitions.map((workflow) =>
      workflowStamp(
        workflow.ref,
        workflowsRevalidated ? "valid" : "pending_revalidation",
      ),
    )
    const pending = stamps.filter(
      (stamp) => stamp.status === "pending_revalidation",
    ).length
    return {
      current: {
        picoclaw_version: "test",
        git_commit: "test",
        workflow_engine_version: "1",
        workflow_schema_version: "1",
        validator_fingerprint: "test",
      },
      workflows: stamps,
      counts: workflowsRevalidated
        ? { valid: workflowDefinitions.length }
        : { pending_revalidation: pending },
      version_changed: !workflowsRevalidated,
      manifest_missing: false,
      has_blocking: !workflowsRevalidated,
    }
  }

  function runByID(id: string) {
    return runs.find((run) => run.id === id) ?? workflowRun
  }

  function currentDraftKey(session: MockWorkflowDevelopmentSession) {
    return workflowDraftKey(session.target_workflow_ref, session.yaml)
  }

  function eventTriggerRevision(yaml: string) {
    return `mock-revision:${normalizeWorkflowDraftYAML(yaml).length}`
  }

  function workflowTriggerInspection(yaml: string) {
    const eventTrigger = yaml.includes("\n  event:")
      ? {
          sources: ["github"],
          types: ["issues.opened"],
        }
      : null
    const workflowCall = yaml.includes("\n  workflow_call:")
      ? {
          inputs: {
            ticket: {
              type: "string",
              required: true,
            },
          },
        }
      : null
    const absent = { present: false, editable: true, value: null }
    return {
      revision: eventTriggerRevision(yaml),
      triggers: {
        manual: absent,
        schedule: absent,
        channel_message: absent,
        command: absent,
        runtime_event: absent,
        event:
          eventTrigger == null
            ? absent
            : { present: true, editable: true, value: eventTrigger },
        workflow_call:
          workflowCall == null
            ? absent
            : { present: true, editable: true, value: workflowCall },
      },
      validation: {
        valid: true,
        validated_at: "2026-07-16T12:00:00Z",
      },
    }
  }

  function workflowJobsRevision(yaml: string) {
    return `mock-jobs-revision:${normalizeWorkflowDraftYAML(yaml).length}`
  }

  function workflowJobsInspection(yaml: string) {
    const projectedName = yaml.match(/^# mock-step-name: (.*)$/m)?.[1]
    const absent = { present: false, value: null }
    return {
      revision: workflowJobsRevision(yaml),
      editable: true,
      complete: true,
      limits: [],
      jobs: [
        {
          id: "triage",
          index: 0,
          editable: true,
          advanced_fields_present: false,
          steps_present: true,
          fields: {
            name: absent,
            runs_on: { present: true, value: "picoclaw" },
            needs: absent,
            uses: absent,
            if: absent,
            continue_on_error: absent,
            with: absent,
            secrets: absent,
            outputs: absent,
            context: absent,
          },
          steps: [
            {
              index: 0,
              editable: true,
              advanced_fields_present: false,
              fields: {
                id: { present: true, value: "summarize" },
                name:
                  projectedName == null
                    ? absent
                    : { present: true, value: projectedName },
                uses: { present: true, value: "agent/main" },
                if: absent,
                continue_on_error: absent,
                with: {
                  present: true,
                  value: { prompt: "Summarize support tickets" },
                },
                context: absent,
              },
            },
          ],
        },
      ],
      validation: {
        valid: true,
        validated_at: "2026-07-16T12:00:00Z",
      },
    }
  }

  await page.route(
    (url) => url.pathname.startsWith("/api/"),
    async (route) => {
      const request = route.request()
      const url = new URL(request.url())
      const path = url.pathname
      const method = request.method()

      if (path.startsWith("/api/repository-reviews/automations")) {
        return mockRepositoryReviewAutomationRequest(
          route,
          url,
          repositoryReviewState,
          options.repositoryReviewRequests,
        )
      }

      if (
        path === "/api/event-source-settings" ||
        path === "/api/event-sources" ||
        path.startsWith("/api/event-sources/")
      ) {
        const rawBody = request.postData()
        const body = rawBody
          ? (request.postDataJSON() as Record<string, unknown>)
          : null
        if (method !== "GET") {
          options.eventSourceRequests?.push({ method, path, body })
        }

        if (method === "GET" && path === "/api/event-source-settings") {
          return json(route, {
            event_source_settings: structuredClone(currentEventSourceSettings),
            eligible_channel_adapters: structuredClone(
              defaultEligibleEventChannelAdapters,
            ),
            config_revision: eventSourceRevision(),
          })
        }
        if (method === "PUT" && path === "/api/event-source-settings") {
          if (body?.expected_config_revision !== eventSourceRevision()) {
            return json(
              route,
              {
                code: "config_revision_mismatch",
                message: "Configuration changed; reload and try again",
              },
              409,
            )
          }
          currentEventSourceSettings = structuredClone(
            body.event_source_settings as typeof defaultEventSourceSettings,
          )
          const configRevision = advanceEventSourceRevision()
          return json(route, {
            event_source_settings: structuredClone(currentEventSourceSettings),
            eligible_channel_adapters: structuredClone(
              defaultEligibleEventChannelAdapters,
            ),
            config_revision: configRevision,
            effects: eventSourceEffects(),
          })
        }
        if (method === "GET" && path === "/api/event-sources") {
          return json(route, {
            event_sources: currentEventSources.map(eventSourceSummary),
            total: currentEventSources.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: mockCollectionSchemas.eventSources,
            config_revision: eventSourceRevision(),
          })
        }
        if (method === "POST" && path === "/api/event-sources") {
          if (body?.expected_config_revision !== eventSourceRevision()) {
            return json(route, { code: "config_revision_mismatch" }, 409)
          }
          const input = structuredClone(
            body.event_source as Record<string, unknown>,
          )
          const name = String(input.name ?? "new-source")
          const secretConfigured =
            input.kind === "webhook" && input.secret_update === "replace"
          delete input.secret
          delete input.secret_update
          const created = {
            ...input,
            id: Buffer.from(
              `event-source\0${String(input.kind ?? "webhook")}:${name}`,
            ).toString("base64url"),
            status: input.enabled ? "available" : "disabled",
            endpoint:
              input.kind === "webhook" ? `/webhooks/events/${name}` : undefined,
            secret_configured: secretConfigured,
          } as MockEventSource
          currentEventSources.push(created)
          const configRevision = advanceEventSourceRevision()
          return json(
            route,
            {
              event_source: structuredClone(created),
              config_revision: configRevision,
              effects: eventSourceEffects(),
            },
            201,
          )
        }
        if (method === "POST" && path === "/api/event-sources/bulk-delete") {
          if (body?.config_revision !== eventSourceRevision()) {
            return json(route, { code: "config_revision_mismatch" }, 409)
          }
          const ids = Array.isArray(body.ids)
            ? body.ids.filter((id): id is string => typeof id === "string")
            : []
          const forcedFailures = new Set(
            options.eventSourceBulkFailureIDs ?? [],
          )
          const deletedIDs: string[] = []
          const failures: Array<{ id: string; code: string }> = []
          for (const id of ids) {
            const index = currentEventSources.findIndex(
              (source) => source.id === id,
            )
            if (index < 0 || forcedFailures.has(id)) {
              failures.push({ id, code: "not_found" })
              continue
            }
            currentEventSources.splice(index, 1)
            deletedIDs.push(id)
          }
          const configRevision =
            deletedIDs.length > 0
              ? advanceEventSourceRevision()
              : eventSourceRevision()
          return json(route, {
            deleted_ids: deletedIDs,
            failures,
            config_revision: configRevision,
            effects: eventSourceEffects(),
          })
        }

        const detailMatch = path.match(/^\/api\/event-sources\/([^/]+)$/)
        if (detailMatch) {
          const id = decodeURIComponent(detailMatch[1])
          const index = currentEventSources.findIndex(
            (source) => source.id === id,
          )
          if (index < 0) {
            return json(
              route,
              {
                code: "event_source_not_found",
                message: "Event source not found",
              },
              404,
            )
          }
          if (method === "GET") {
            return json(route, {
              event_source: structuredClone(currentEventSources[index]),
              config_revision: eventSourceRevision(),
            })
          }
          const suppliedRevision = body?.expected_config_revision
          if (suppliedRevision !== eventSourceRevision()) {
            return json(route, { code: "config_revision_mismatch" }, 409)
          }
          if (method === "PUT") {
            const input = structuredClone(
              body.event_source as Record<string, unknown>,
            )
            const secretUpdate = input.secret_update
            delete input.secret
            delete input.secret_update
            currentEventSources[index] = {
              ...currentEventSources[index],
              ...(input as Partial<MockEventSource>),
              ...(input.kind === "webhook" && secretUpdate === "replace"
                ? { secret_configured: true }
                : input.kind === "webhook" && secretUpdate === "clear"
                  ? { secret_configured: false }
                  : {}),
            } as MockEventSource
            const configRevision = advanceEventSourceRevision()
            return json(route, {
              event_source: structuredClone(currentEventSources[index]),
              config_revision: configRevision,
              effects: eventSourceEffects(),
            })
          }
          if (method === "DELETE") {
            currentEventSources.splice(index, 1)
            const configRevision = advanceEventSourceRevision()
            return json(route, {
              deleted_ids: [id],
              failures: [],
              config_revision: configRevision,
              effects: eventSourceEffects(),
            })
          }
        }

        return json(route, { code: "unsupported_event_source_request" }, 405)
      }

      if (
        path === "/api/git-workspaces" ||
        path.startsWith("/api/git-workspaces/")
      ) {
        const rawBody = request.postData()
        const body = rawBody
          ? (request.postDataJSON() as Record<string, unknown>)
          : null
        if (method !== "GET") {
          options.gitWorkspaceRequests?.push({ method, path, body })
        }
        const summaries = currentGitWorkspaceDetails.map(
          gitWorkspaceSummaryFromDetail,
        )
        const activeSummaries = summaries.filter(
          (workspace) => workspace.status !== "dropped",
        )
        const inventoryResponse = () => ({
          ...gitWorkspaceResponse,
          workspaces: structuredClone(summaries),
          total: summaries.length,
          workspace_count: activeSummaries.length,
          locked_workspace_count: activeSummaries.filter(
            (workspace) => workspace.locked,
          ).length,
          total_size_bytes: activeSummaries.reduce(
            (total, workspace) => total + workspace.size,
            0,
          ),
          ignored_bytes: activeSummaries.reduce(
            (total, workspace) => total + workspace.ignored,
            0,
          ),
          canonical_query:
            url.searchParams.get("query") ?? "ORDER BY updated DESC",
        })

        if (method === "GET" && path === "/api/git-workspaces") {
          return json(route, inventoryResponse())
        }
        if (method === "GET" && path === "/api/git-workspaces/history") {
          return json(route, {
            ...gitWorkspaceHistoryResponse,
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY time DESC",
          })
        }
        if (path === "/api/git-workspaces/settings") {
          if (method === "GET") {
            return json(route, structuredClone(currentGitWorkspaceSettings))
          }
          if (method === "PUT") {
            const settings = body?.settings as
              | typeof gitWorkspaceSettingsResponse.configured
              | undefined
            if (!settings) {
              return json(
                route,
                {
                  code: "invalid_git_workspace_settings",
                  message: "Settings are required",
                },
                400,
              )
            }
            currentGitWorkspaceSettings = {
              ...currentGitWorkspaceSettings,
              configured: structuredClone(settings),
              effective: structuredClone(settings),
              config_revision: "sha256:git-workspace-settings-2",
            }
            return json(route, structuredClone(currentGitWorkspaceSettings))
          }
        }
        if (method === "POST" && path === "/api/git-workspaces/reconcile") {
          return json(route, {
            cleaned: [],
            dropped: [],
            stats: inventoryResponse(),
          })
        }
        if (method === "POST" && path === "/api/git-workspaces/cleanup") {
          const workspaceID = String(body?.workspace_id ?? "")
          const workspace = currentGitWorkspaceDetails.find(
            (candidate) => candidate.id === workspaceID,
          )
          if (!workspace || workspace.locked) {
            return json(
              route,
              { code: "git_workspace_locked", message: "Workspace is locked" },
              409,
            )
          }
          const before = workspace.ignored
          workspace.ignored = 0
          workspace.updated = "2026-07-16T12:03:00Z"
          workspace.last_cleaned = workspace.updated
          return json(route, {
            workspace: structuredClone(workspace),
            before_ignored_bytes: before,
            after_ignored_bytes: 0,
          })
        }
        const detailMatch = path.match(/^\/api\/git-workspaces\/([^/]+)$/)
        if (detailMatch) {
          const workspaceID = decodeURIComponent(detailMatch[1])
          const workspace = currentGitWorkspaceDetails.find(
            (candidate) => candidate.id === workspaceID,
          )
          if (!workspace) {
            return json(
              route,
              {
                code: "git_workspace_not_found",
                message: "Git workspace not found",
              },
              404,
            )
          }
          if (method === "GET") {
            return json(route, {
              workspace: structuredClone(workspace),
              root_dir: "/tmp/picoclaw-git-workspaces",
            })
          }
          if (method === "DELETE") {
            if (workspace.locked) {
              return json(
                route,
                {
                  code: "git_workspace_locked",
                  message: "Workspace is locked",
                },
                409,
              )
            }
            workspace.status = "dropped"
            return json(route, { workspace: structuredClone(workspace) })
          }
        }
        return json(route, { code: "unsupported_git_workspace_request" }, 405)
      }

      if (
        options.statefulModelEvaluations &&
        path.startsWith("/api/model-evaluations")
      ) {
        const body = request.postData()
          ? (request.postDataJSON() as Record<string, unknown>)
          : null
        if (method !== "GET") {
          options.modelEvaluationRequests?.push({ method, path, body })
        }
        if (method === "GET" && path === "/api/model-evaluations/options") {
          return json(route, {
            models: [
              {
                alias: "code",
                resolved_model: "gpt-code",
                provider: "openai",
                available: true,
              },
              {
                alias: "fast",
                resolved_model: "gpt-fast",
                provider: "openai",
                available: true,
              },
              {
                alias: "review",
                resolved_model: "gpt-review",
                provider: "openai",
                available: true,
              },
            ],
            repositories: [],
            profiles: [
              {
                ...modelEvaluationProfile,
                available_models: ["code", "fast"],
              },
            ],
            code_types: ["hotpath-code", "code", "test", "bench-test"],
            default_files_per_language: 20,
            max_files_per_language: 20,
            max_candidate_models: 8,
          })
        }
        if (method === "GET" && path === "/api/model-evaluations") {
          return json(route, {
            evaluations: currentModelEvaluation
              ? [structuredClone(currentModelEvaluation)]
              : [],
            total: currentModelEvaluation ? 1 : 0,
            next_cursor: "",
            canonical_query: "ORDER BY updated DESC",
            query_schema: mockCollectionSchemas.evaluations,
          })
        }
        if (method === "POST" && path === "/api/model-evaluations/run") {
          const created = modelEvaluationFromBody(body ?? {})
          currentModelEvaluation = {
            ...created,
            status: "running",
            progress: {
              stage: "candidate_execution",
              languages: {
                go: {
                  available_files: 4,
                  selected_files: 2,
                  completed_files: 1,
                  selected_bytes: 12_000,
                  regions: ["pkg", "cmd"],
                  limited: false,
                },
              },
              total_files: 4,
              selected_files: 2,
              completed_files: 1,
              total_tasks: 4,
              completed_tasks: 2,
              current_batch: 1,
              total_batches: 2,
              completed_calls: 1,
              total_calls: 4,
              failed_calls: 0,
              active_children: [
                {
                  index: 2,
                  label: "scope chunk 1 of 2, reviewer 2 of 2 (code)",
                  model_alias: "code",
                  scope_count: 2,
                  started_at: "2026-08-24T09:45:14Z",
                },
                {
                  index: 3,
                  label: "scope chunk 2 of 2, reviewer 1 of 2 (fast)",
                  model_alias: "fast",
                  scope_count: 2,
                  started_at: "2026-08-24T09:45:15Z",
                },
              ],
              message: "Running candidate models.",
              current_model: "code",
              current_path: "pkg/service.go",
              percent: 40,
            },
            run_ids: ["wr_selector", "wr_candidate"],
          }
          modelEvaluationDetailReads = 0
          return json(
            route,
            { evaluation: structuredClone(currentModelEvaluation) },
            202,
          )
        }
        if (method === "POST" && path === "/api/model-evaluations") {
          currentModelEvaluation = modelEvaluationFromBody(body ?? {})
          return json(
            route,
            { evaluation: structuredClone(currentModelEvaluation) },
            201,
          )
        }
        const corpusMatch = path.match(
          /^\/api\/model-evaluations\/([^/]+)\/corpus$/,
        )
        if (method === "GET" && corpusMatch) {
          return json(route, {
            commit_sha: "a".repeat(40),
            inventory_hash: "sha256:inventory",
            language_counts: { go: 2 },
            files: [],
            offset: 0,
            total: 2,
          })
        }
        const actionMatch = path.match(
          /^\/api\/model-evaluations\/([^/]+)\/(preflight|start|run|cancel|resume|restart)$/,
        )
        if (method === "POST" && actionMatch && currentModelEvaluation) {
          const action = actionMatch[2]
          const version = Number(currentModelEvaluation.version) + 1
          if (action === "preflight") {
            currentModelEvaluation = {
              ...currentModelEvaluation,
              version,
              status: "ready",
              progress: {
                stage: "ready",
                languages: {
                  go: {
                    available_files: 4,
                    selected_files: 2,
                    completed_files: 0,
                    selected_bytes: 12_000,
                    regions: ["pkg", "cmd"],
                    limited: false,
                  },
                },
                total_files: 4,
                selected_files: 2,
                completed_files: 0,
                total_tasks: 4,
                completed_tasks: 0,
                message: "Corpus ready.",
                percent: 20,
              },
              run_ids: ["wr_selector"],
            }
          } else if (
            action === "start" ||
            action === "run" ||
            action === "resume"
          ) {
            currentModelEvaluation = {
              ...currentModelEvaluation,
              version,
              status: "running",
              progress: {
                ...(currentModelEvaluation.progress as Record<string, unknown>),
                stage: "candidate_execution",
                message: "Running candidate models.",
                current_model: "code",
                current_path: "pkg/service.go",
                percent: 40,
              },
            }
          } else if (action === "cancel") {
            currentModelEvaluation = {
              ...currentModelEvaluation,
              version,
              status: "canceled",
              progress: {
                ...(currentModelEvaluation.progress as Record<string, unknown>),
                stage: "canceled",
                message: "Canceled.",
                active_children: [],
              },
            }
          }
          return json(
            route,
            { evaluation: structuredClone(currentModelEvaluation) },
            202,
          )
        }
        const detailMatch = path.match(/^\/api\/model-evaluations\/([^/]+)$/)
        if (detailMatch && currentModelEvaluation) {
          if (method === "GET") {
            modelEvaluationDetailReads += 1
            if (
              currentModelEvaluation.status === "running" &&
              modelEvaluationDetailReads >= 2
            ) {
              currentModelEvaluation = {
                ...currentModelEvaluation,
                version: Number(currentModelEvaluation.version) + 1,
                status: "completed",
                progress: {
                  ...(currentModelEvaluation.progress as Record<
                    string,
                    unknown
                  >),
                  stage: "completed",
                  message: "Repository model evaluation completed.",
                  completed_files: 2,
                  completed_tasks: 4,
                  completed_calls: 4,
                  active_children: [],
                  current_model: "",
                  current_path: "",
                  percent: 100,
                },
                usage: {
                  requests: 12,
                  input_tokens: 120_000,
                  cached_input_tokens: 10_000,
                  output_tokens: 24_000,
                  reasoning_tokens: 8_000,
                  duration_millis: 180_000,
                },
                comparisons: [
                  {
                    model_alias: "code",
                    concrete_models: { "gpt-code": 2 },
                    completion: "completed",
                    failures: 0,
                    rank: 1,
                    overall_score: 92.5,
                    scores: {
                      correctness: 95,
                      evidence: 93,
                      coverage: 88,
                      actionability: 94,
                    },
                    languages: ["go"],
                    regions: ["pkg", "cmd"],
                    files_analyzed: 2,
                    bytes_analyzed: 12_000,
                    confirmed_findings: 8,
                    unsupported_claims: 1,
                    unsupported_files: 1,
                    usage: {
                      requests: 4,
                      input_tokens: 40_000,
                      cached_input_tokens: 0,
                      output_tokens: 10_000,
                      reasoning_tokens: 4_000,
                      duration_millis: 120_000,
                    },
                    verdict: "Best evidence-grounded analysis.",
                    strengths: ["Precise source evidence"],
                    limitations: ["Higher cumulative model time"],
                  },
                  {
                    model_alias: "fast",
                    concrete_models: { "gpt-fast": 2 },
                    completion: "completed",
                    failures: 0,
                    rank: 2,
                    overall_score: 84,
                    scores: {
                      correctness: 86,
                      evidence: 85,
                      coverage: 76,
                      actionability: 82,
                    },
                    languages: ["go"],
                    regions: ["pkg", "cmd"],
                    files_analyzed: 2,
                    bytes_analyzed: 12_000,
                    confirmed_findings: 5,
                    unsupported_claims: 2,
                    unsupported_files: 2,
                    usage: {
                      requests: 4,
                      input_tokens: 40_000,
                      cached_input_tokens: 0,
                      output_tokens: 6_000,
                      reasoning_tokens: 2_000,
                      duration_millis: 60_000,
                    },
                    verdict: "Fast, but materially narrower.",
                    strengths: ["Lower cumulative model time"],
                    limitations: ["Missed important findings"],
                  },
                ],
                work_sizing_results: [
                  modelSizingResult(
                    "code-files-1",
                    "files_per_batch",
                    "code",
                    1,
                    131_072,
                    94,
                  ),
                  modelSizingResult(
                    "code-files-8",
                    "files_per_batch",
                    "code",
                    8,
                    131_072,
                    87,
                  ),
                  modelSizingResult(
                    "code-bytes-32k",
                    "content_bytes_per_batch",
                    "code",
                    8,
                    32_768,
                    94,
                  ),
                  modelSizingResult(
                    "code-bytes-128k",
                    "content_bytes_per_batch",
                    "code",
                    8,
                    131_072,
                    92,
                  ),
                ],
                warnings: [
                  "Quality scores are comparative AI judgments, not ground truth.",
                ],
                finished_at: "2026-08-21T12:03:00Z",
                updated_at: "2026-08-21T12:03:00Z",
              }
            }
            return json(route, {
              evaluation: structuredClone(currentModelEvaluation),
            })
          }
          if (method === "PATCH") {
            currentModelEvaluation = modelEvaluationFromBody(
              body ?? {},
              currentModelEvaluation,
            )
            currentModelEvaluation.version =
              Number(currentModelEvaluation.version) + 1
            return json(route, {
              evaluation: structuredClone(currentModelEvaluation),
            })
          }
          if (method === "DELETE") {
            currentModelEvaluation = null
            return json(route, undefined, 204)
          }
        }
        return json(route, { code: "not_found", message: "not found" }, 404)
      }

      const capabilitiesMatch = path.match(
        /^\/api\/agents\/([^/]+)\/capabilities$/,
      )
      if (capabilitiesMatch) {
        const body =
          method === "PATCH"
            ? (request.postDataJSON() as Record<string, unknown>)
            : undefined
        options.agentCapabilityRequests?.push({ method, path, body })
        if (decodeURIComponent(capabilitiesMatch[1]) !== "reviewer") {
          return json(route, { error: "agent_not_found" }, 404)
        }
        if (method === "GET") {
          return json(route, structuredClone(currentAgentCapabilities))
        }
        if (
          method === "PATCH" &&
          body?.expected_revision === currentAgentCapabilities.revision
        ) {
          currentCapabilityRevision += 1
          currentAgentCapabilities = {
            ...currentAgentCapabilities,
            capabilities: {
              tools:
                (body.tools as AgentCapabilitiesResponse["capabilities"]["tools"]) ??
                currentAgentCapabilities.capabilities.tools,
              skills: body.skills
                ? {
                    ...(body.skills as {
                      mode: "inherit" | "none" | "selected"
                      values: string[]
                    }),
                    inherited_values:
                      currentAgentCapabilities.capabilities.skills
                        .inherited_values,
                  }
                : currentAgentCapabilities.capabilities.skills,
              mcp_servers:
                (body.mcp_servers as AgentCapabilitiesResponse["capabilities"]["mcp_servers"]) ??
                currentAgentCapabilities.capabilities.mcp_servers,
            },
            revision: `capability-revision-${currentCapabilityRevision}`,
          }
          return json(route, structuredClone(currentAgentCapabilities))
        }
        return json(route, { error: "capabilities_revision_mismatch" }, 409)
      }

      const activityMatch = path.match(/^\/api\/agents\/([^/]+)\/activity$/)
      if (activityMatch) {
        options.agentActivityRequests?.push({ method, path })
        if (
          method !== "GET" ||
          decodeURIComponent(activityMatch[1]) !== "reviewer"
        ) {
          return json(route, { error: "agent_not_found" }, 404)
        }
        return json(route, {
          agent_id: "reviewer",
          events: [
            {
              sequence: "1",
              agent_id: "reviewer",
              timestamp: "2026-07-30T12:00:00.000000001Z",
              kind: "agent.tool.exec_end",
              severity: "info",
              details: {
                tool_name: "web_search",
                duration_ms: "25",
                is_error: false,
                async: false,
                arguments: "CANARY_ARGUMENT_SECRET",
                result: "CANARY_RESULT_SECRET",
              },
              prompt: "CANARY_PROMPT_SECRET",
              error: "CANARY_ERROR_SECRET",
            },
          ],
          next_cursor: "opaque-cursor-1",
          reset: true,
          truncated: true,
          dropped: {
            subscription: "1",
            retention: "2",
            projection: "3",
          },
          raw_payload: "CANARY_RAW_SECRET",
        })
      }

      if (options.statefulAgents && path.startsWith("/api/agents")) {
        const rawBody = request.postData()
        const body = rawBody
          ? (request.postDataJSON() as {
              expected_config_revision?: string
              agent?: AgentMutationInput
            })
          : undefined
        if (method !== "GET") {
          options.agentRequests?.push({ method, path, body })
        }

        if (method === "GET" && path === "/api/agents") {
          return json(route, currentAgentsResponse())
        }

        const defaultMatch = path.match(/^\/api\/agents\/([^/]+)\/default$/)
        const itemMatch = path.match(/^\/api\/agents\/([^/]+)$/)

        if (method === "GET" && itemMatch) {
          const id = decodeURIComponent(itemMatch[1])
          const agent = currentAgents.find((candidate) => candidate.id === id)
          if (agent == null) {
            return json(route, { error: "agent_not_found" }, 404)
          }
          const collection = currentAgentsResponse()
          return json(route, {
            agent: structuredClone(agent),
            default_agent_id: collection.default_agent_id,
            config_revision: collection.config_revision,
            effects: collection.effects,
          })
        }

        if (
          body?.expected_config_revision !==
          currentAgentsResponse().config_revision
        ) {
          return json(route, { error: "config_revision_mismatch" }, 409)
        }

        if (method === "POST" && path === "/api/agents" && body.agent) {
          currentAgents.push({
            ...structuredClone(body.agent),
            is_default: false,
            default_configured: false,
            implicit: false,
          })
          return json(route, advanceAgentsRevision(), 201)
        }

        if (method === "POST" && defaultMatch) {
          const id = decodeURIComponent(defaultMatch[1])
          currentDefaultAgentID = id
          currentAgents = currentAgents.map((agent) => ({
            ...agent,
            is_default: agent.id === id,
            default_configured: agent.id === id,
          }))
          return json(route, advanceAgentsRevision())
        }

        if (itemMatch) {
          const id = decodeURIComponent(itemMatch[1])
          const index = currentAgents.findIndex((agent) => agent.id === id)
          if (index < 0) {
            return json(route, { error: "agent_not_found" }, 404)
          }
          if (method === "PUT" && body.agent) {
            const existing = currentAgents[index]
            currentAgents[index] = {
              ...structuredClone(body.agent),
              is_default: existing.is_default,
              default_configured: existing.default_configured,
              implicit: false,
            }
            return json(route, advanceAgentsRevision())
          }
          if (method === "DELETE") {
            currentAgents.splice(index, 1)
            if (currentDefaultAgentID === id) {
              currentDefaultAgentID = currentAgents[0]?.id ?? "main"
              currentAgents = currentAgents.map((agent, agentIndex) => ({
                ...agent,
                is_default: agentIndex === 0,
                default_configured: agentIndex === 0,
              }))
            }
            return json(route, advanceAgentsRevision())
          }
        }

        return json(route, { error: "unsupported_agent_request" }, 405)
      }

      if (options.statefulMCP && path.startsWith("/api/mcp")) {
        const rawBody = request.postData()
        const body = rawBody ? (request.postDataJSON() as unknown) : undefined
        if (method !== "GET") {
          options.mcpRequests?.push({ method, path, body })
        }

        if (method === "PATCH" && path === "/api/mcp/settings") {
          const settings = body as Pick<
            MCPConfigResponse,
            "enabled" | "discovery"
          >
          currentMCPResponse = {
            ...currentMCPResponse,
            enabled: settings.enabled,
            discovery: settings.discovery,
          }
          return json(route, currentMCPResponse)
        }
        if (method === "POST" && path === "/api/mcp/servers") {
          const input = body as MCPServerInput
          currentMCPResponse.servers.push(mcpServerFromInput(input))
          return json(route, currentMCPResponse)
        }
        if (method === "POST" && path === "/api/mcp/servers/test") {
          return json(route, {
            ok: true,
            tool_count: 2,
            tools: ["issues_list", "issue_create"],
          })
        }

        const credentialMatch = path.match(
          /^\/api\/mcp\/servers\/([^/]+)\/credential$/,
        )
        if (credentialMatch) {
          const name = decodeURIComponent(credentialMatch[1])
          const server = currentMCPResponse.servers.find(
            (candidate) => candidate.name === name,
          )
          if (server && method === "PUT") {
            server.auth = { type: "bearer", configured: true }
          } else if (
            server &&
            method === "DELETE" &&
            server.auth.type !== "custom"
          ) {
            server.auth = { type: "none", configured: false }
          }
          return json(route, { status: "ok" })
        }

        const serverMatch = path.match(/^\/api\/mcp\/servers\/([^/]+)$/)
        if (serverMatch) {
          const currentName = decodeURIComponent(serverMatch[1])
          const index = currentMCPResponse.servers.findIndex(
            (candidate) => candidate.name === currentName,
          )
          if (method === "PUT" && index >= 0) {
            currentMCPResponse.servers[index] = mcpServerFromInput(
              body as MCPServerInput,
              currentMCPResponse.servers[index],
            )
            return json(route, currentMCPResponse)
          }
          if (method === "DELETE" && index >= 0) {
            currentMCPResponse.servers.splice(index, 1)
            return json(route, { status: "ok" })
          }
        }
      }

      if (method === "POST" && path === "/api/skills/bulk-delete") {
        const body = request.postDataJSON() as { ids?: string[] }
        const ids = Array.isArray(body.ids) ? body.ids : []
        const deletedIDs: string[] = []
        const failures: Array<{ id: string; code: string }> = []
        for (const id of ids) {
          const index = currentSkills.findIndex((skill) => skill.id === id)
          if (index < 0) {
            failures.push({ id, code: "not_found" })
          } else if (!currentSkills[index].removable) {
            failures.push({ id, code: "read_only_origin" })
          } else {
            currentSkills.splice(index, 1)
            deletedIDs.push(id)
          }
        }
        return json(route, { deleted_ids: deletedIDs, failures })
      }

      const skillMutationMatch = path.match(/^\/api\/skills\/([^/]+)$/)
      if (method === "DELETE" && skillMutationMatch) {
        const id = decodeURIComponent(skillMutationMatch[1])
        const index = currentSkills.findIndex(
          (skill) => skill.id === id || skill.name === id,
        )
        if (index < 0) {
          return json(
            route,
            { code: "not_found", message: "Skill not found" },
            404,
          )
        }
        if (!currentSkills[index].removable) {
          return json(
            route,
            { code: "read_only_origin", message: "Skill is read-only" },
            409,
          )
        }
        const [deleted] = currentSkills.splice(index, 1)
        return json(route, { status: "deleted", id: deleted.id })
      }

      const toolMutationMatch = path.match(/^\/api\/tools\/([^/]+)\/state$/)
      if (method === "PUT" && toolMutationMatch) {
        const id = decodeURIComponent(toolMutationMatch[1])
        const tool = currentTools.find(
          (candidate) => candidate.id === id || candidate.name === id,
        )
        if (!tool) {
          return json(
            route,
            { code: "not_found", message: "Tool not found" },
            404,
          )
        }
        const body = request.postDataJSON() as { enabled?: boolean }
        tool.status = body.enabled ? "enabled" : "disabled"
        delete tool.reason
        delete tool.reason_code
        return json(route, { status: "ok" })
      }

      if (method === "PATCH" && path === "/api/workflows/settings") {
        const body = request.postDataJSON() as Record<string, unknown>
        return json(route, {
          ...workflowSettingsResponse,
          configured: {
            ...workflowSettingsResponse.configured,
            ...body,
          },
          config_revision: "sha256:workflow-settings-2",
        })
      }

      if (method === "POST") {
        switch (path) {
          case "/api/accounts/models/fetch": {
            const body = request.postDataJSON() as {
              credential_id?: string
              account_ref?: string
            }
            const failure = body.credential_id
              ? options.fetchModelFailures?.[body.credential_id]
              : undefined
            if (failure) {
              return route.fulfill({
                status: 502,
                contentType: "text/plain",
                body: failure,
              })
            }
            if (
              body.credential_id &&
              options.fetchModelEmptyCredentials?.includes(body.credential_id)
            ) {
              return json(route, {
                models: [],
                total: 0,
              })
            }
            const accountModels =
              body.account_ref === "gpt-4o-mini"
                ? ["gpt-4o-mini", "gpt-5.4"]
                : body.account_ref === "gpt-4o"
                  ? ["gpt-4o", "gpt-5.4", "gpt-5.5-sol"]
                  : ["gpt-4o", "gpt-5.4"]
            return json(route, {
              models: accountModels.map((id) => ({
                id,
                owned_by: "openai",
              })),
              total: accountModels.length,
            })
          }
          case "/api/workflows/development/start": {
            const body = request.postDataJSON() as {
              reason?: string
              prompt?: string
              ref?: string
              target_ref?: string
            }
            if (
              body.reason === "version_revalidation" ||
              body.reason === "edit"
            ) {
              activeDevelopmentSession = {
                ...workflowDraftSession,
                reason: body.reason,
                prompt: body.prompt ?? "",
                source_workflow_ref: body.ref,
                source_workflow_id:
                  body.ref == null
                    ? undefined
                    : smokeWorkflowDefinitionIDs[body.ref],
                target_workflow_ref:
                  body.target_ref ??
                  body.ref ??
                  workflowDraftSession.target_workflow_ref,
                yaml: workflowDraftYAML,
              }
            } else {
              activeDevelopmentSession = {
                ...workflowDraftSession,
                prompt: body.prompt ?? workflowDraftSession.prompt,
                target_workflow_ref:
                  body.target_ref ?? workflowDraftSession.target_workflow_ref,
                yaml:
                  options.workflowDevelopmentYAML ?? workflowDraftSession.yaml,
              }
            }
            return json(route, { session: activeDevelopmentSession })
          }
          case "/api/workflows/definitions/inspect": {
            const body = request.postDataJSON() as { ref?: unknown }
            expect(request.headers()["content-type"]).toBe("application/json")
            expect(typeof body.ref).toBe("string")
            const ref = body.ref as string
            expect(body).toEqual({ ref })
            expect(ref).toMatch(/^workflows\/[^/]+\.ya?ml$/)
            options.workflowInspectionRequests?.push({
              method,
              path,
              body,
            })
            return json(
              route,
              workflowDefinitionInspection({ kind: "published", ref }),
            )
          }
          case "/api/workflows/development/event-trigger/inspect": {
            const body = request.postDataJSON() as { yaml: string }
            const eventTrigger = body.yaml.includes("\n  event:")
              ? {
                  sources: ["github"],
                  types: ["issues.opened"],
                }
              : null
            return json(route, {
              revision: eventTriggerRevision(body.yaml),
              editable: true,
              event_trigger: eventTrigger,
              validation: {
                valid: true,
                validated_at: "2026-07-16T12:00:00Z",
              },
            })
          }
          case "/api/workflows/development/triggers/inspect": {
            const body = request.postDataJSON() as { yaml: string }
            return json(route, workflowTriggerInspection(body.yaml))
          }
          case "/api/workflows/development/triggers/render": {
            const body = request.postDataJSON() as {
              yaml: string
              revision: string
              trigger_type: string
              trigger: unknown
            }
            expect(body.revision).toBe(eventTriggerRevision(body.yaml))
            expect(body.trigger_type).toBe("event")
            const renderedYAML =
              body.trigger == null ? workflowDraftYAML : workflowEventDraftYAML
            return json(route, {
              yaml: renderedYAML,
              ...workflowTriggerInspection(renderedYAML),
            })
          }
          case "/api/workflows/development/jobs/inspect": {
            const body = request.postDataJSON() as { yaml: string }
            options.workflowJobRequests?.push({ method, path, body })
            return json(route, workflowJobsInspection(body.yaml))
          }
          case "/api/workflows/development/jobs/render": {
            const body = request.postDataJSON() as {
              yaml: string
              revision: string
              operation: {
                type: string
                fields?: {
                  name?: { mode: string; value?: string }
                }
              }
            }
            expect(body.revision).toBe(workflowJobsRevision(body.yaml))
            options.workflowJobRequests?.push({ method, path, body })
            const nameMutation = body.operation.fields?.name
            const renderedYAML =
              body.operation.type === "step.patch" &&
              nameMutation?.mode === "set"
                ? `${normalizeWorkflowDraftYAML(body.yaml)}# mock-step-name: ${nameMutation.value ?? ""}\n`
                : normalizeWorkflowDraftYAML(body.yaml)
            return json(route, {
              yaml: renderedYAML,
              ...workflowJobsInspection(renderedYAML),
            })
          }
          case "/api/workflows/development/event-trigger/render": {
            const body = request.postDataJSON() as {
              yaml: string
              revision: string
              event_trigger: {
                sources?: string[]
                connectors?: string[]
                types?: string[]
              } | null
            }
            expect(body.revision).toBe(eventTriggerRevision(body.yaml))
            const renderedYAML =
              body.event_trigger == null
                ? workflowDraftYAML
                : workflowEventDraftYAML
            return json(route, {
              yaml: renderedYAML,
              revision: eventTriggerRevision(renderedYAML),
              editable: true,
              event_trigger: body.event_trigger,
              validation: {
                valid: true,
                validated_at: "2026-07-16T12:00:00Z",
              },
            })
          }
          case "/api/workflows/development/event-trigger/match": {
            const body = request.postDataJSON() as {
              yaml: string
              event_id: string
            }
            expect(body).toEqual({
              yaml: workflowEventDraftYAML,
              event_id: eventResponse.id,
            })
            return json(route, {
              event_id: eventResponse.id,
              matched: true,
              checks: [
                {
                  path: "on.event.sources",
                  present: true,
                  value: "github",
                  matched: true,
                },
                {
                  path: "on.event.types",
                  present: true,
                  value: "issues.opened",
                  matched: true,
                },
              ],
              validation: {
                valid: true,
                validated_at: "2026-07-16T12:00:00Z",
              },
            })
          }
          case "/api/workflows/development/triggers/simulate": {
            const body = request.postDataJSON() as Record<string, unknown>
            options.workflowTriggerSimulationRequests?.push({
              method,
              path,
              body,
            })
            const trigger = body.trigger as {
              type: string
              schedule_index?: number
            }
            const scenario = body.scenario as {
              inputs?: Record<string, unknown>
              secrets?: Record<string, string>
              session?: string
              delivery?: Record<string, unknown>
            }
            return json(route, {
              simulation: {
                selected_kind: trigger.type,
                effective_kind: trigger.type,
                ...(trigger.type === "schedule"
                  ? { schedule_index: trigger.schedule_index }
                  : {}),
                present: true,
                matched: true,
                executable: true,
                reason: "matched",
                context_summary: {
                  input_count: Object.keys(scenario.inputs ?? {}).length,
                  secret_count: Object.keys(scenario.secrets ?? {}).length,
                  has_event:
                    trigger.type === "event" ||
                    trigger.type === "runtime_event",
                  has_session:
                    typeof scenario.session === "string" &&
                    scenario.session !== "",
                  has_delivery:
                    scenario.delivery != null &&
                    Object.keys(scenario.delivery).length > 0,
                },
              },
              review: {
                job_count: 1,
                step_count: 1,
                targets: ["agent/main"],
                effects: [
                  {
                    kind: "model_or_delegated_action_possible",
                    target: "agent/main",
                    occurrences: 1,
                  },
                ],
                complete: true,
                validation: {
                  valid: true,
                  issue_count: 0,
                  issues: [],
                  truncated: false,
                },
                limits: [],
              },
              review_token: `review-token:${trigger.type}`,
            })
          }
          case "/api/workflows/development/test/execute": {
            const body = request.postDataJSON() as Record<string, unknown>
            options.workflowTriggerExecutionRequests?.push({
              method,
              path,
              body,
            })
            const current =
              activeDevelopmentSession ??
              ({
                ...workflowDraftSession,
                yaml:
                  options.workflowDevelopmentYAML ?? workflowDraftSession.yaml,
              } satisfies MockWorkflowDevelopmentSession)
            activeDevelopmentSession = {
              ...current,
              prompt:
                typeof body.prompt === "string" ? body.prompt : current.prompt,
              target_workflow_ref:
                typeof body.target_ref === "string"
                  ? body.target_ref
                  : current.target_workflow_ref,
              yaml: typeof body.yaml === "string" ? body.yaml : current.yaml,
              session_revision: "opaque-session-reviewed-running",
              status: "testing",
              last_test: {
                ...workflowDraftLastTest,
                draft_key: workflowDraftKey(
                  typeof body.target_ref === "string"
                    ? body.target_ref
                    : current.target_workflow_ref,
                  typeof body.yaml === "string" ? body.yaml : current.yaml,
                ),
                draft_revision: current.draft_revision,
                run_id: "wr_draft",
                status: "running",
                tested_at: "2026-07-30T12:01:01Z",
              },
              updated_at: "2026-07-30T12:01:01Z",
            }
            completeDraftViaPolling = options.completeDraftViaPolling === true
            return json(
              route,
              {
                session: activeDevelopmentSession,
                result: {
                  run_id: "wr_draft",
                  status: "running",
                },
              },
              202,
            )
          }
          case "/api/workflows/development/ai-revise": {
            const body = request.postDataJSON() as {
              prompt?: string
              target_ref?: string
              yaml?: string
            }
            if (body.prompt?.includes("Last draft test failed")) {
              expect(body.prompt).toContain("Run ID: wr_draft_failed")
              expect(body.prompt).toContain("Error: agent step failed")
              expect(body.prompt).toContain(
                '"workflow_ref": "draft:workflows/support-triage.yml"',
              )
              expect(body.prompt).toContain('"triage/summarize"')
              expect(body.prompt).toContain('"kind": "workflow.run.end"')
              expect(body.prompt).not.toContain("draft failure event")
              expect(body.prompt).not.toContain('"payload"')
              expect(body.prompt).not.toContain('"message"')
            }
            const previous = activeDevelopmentSession ?? workflowDraftSession
            activeDevelopmentSession = {
              ...previous,
              prompt: body.prompt ?? previous.prompt,
              target_workflow_ref:
                body.target_ref ?? previous.target_workflow_ref,
              yaml:
                typeof body.yaml === "string"
                  ? normalizeWorkflowDraftYAML(body.yaml)
                  : previous.yaml,
              validation: {
                valid: true,
                validated_at: "2026-07-16T12:00:02Z",
              },
              updated_at: "2026-07-16T12:00:02Z",
            }
            return json(route, { session: activeDevelopmentSession })
          }
          case "/api/workflows/development/revise": {
            reviseRequestCount += 1
            const body = request.postDataJSON() as {
              prompt?: string
              target_ref?: string
              yaml?: string
              regenerate?: boolean
            }
            const previous = activeDevelopmentSession ?? workflowDraftSession
            const nextYAML =
              typeof body.yaml === "string"
                ? normalizeWorkflowDraftYAML(body.yaml)
                : previous.yaml
            const nextTargetRef =
              typeof body.target_ref === "string" && body.target_ref !== ""
                ? body.target_ref
                : previous.target_workflow_ref
            const draftChanged =
              nextTargetRef !== previous.target_workflow_ref ||
              normalizeWorkflowDraftYAML(nextYAML) !==
                normalizeWorkflowDraftYAML(previous.yaml)
            activeDevelopmentSession = {
              ...previous,
              prompt: body.prompt ?? previous.prompt,
              target_workflow_ref: nextTargetRef,
              yaml: nextYAML,
              updated_at: "2026-07-16T12:01:02Z",
            }
            if (draftChanged) {
              activeDevelopmentSession = {
                ...activeDevelopmentSession,
                status: "editing",
              }
              delete activeDevelopmentSession.last_test
            }
            return json(route, { session: activeDevelopmentSession })
          }
          case "/api/workflows/dependencies/check": {
            const body = request.postDataJSON() as {
              ref?: string
              draft?: {
                target_ref: string
                yaml: string
              }
            }
            const workflowRef = body.ref ?? body.draft?.target_ref
            expect(workflowRef).toBeTruthy()
            if (body.draft != null) {
              expect(body.draft.target_ref).toBe(body.draft.target_ref.trim())
              expect(body.draft.yaml.trim()).not.toBe("")
            } else {
              expect(body).toEqual({ ref: workflowRef })
            }
            return json(route, {
              root_ref: workflowRef,
              revision: "opaque-dependency-revision",
              ready: true,
              workflow_enabled: true,
              structural_ready: true,
              runtime_ready: true,
              dependencies: [
                {
                  dependency: {
                    kind: "agent",
                    name: "main",
                    workflow_ref: workflowRef,
                    path: "jobs.triage.steps[0].uses",
                  },
                  code: "ready",
                  ready: true,
                },
              ],
              structural_issues: [],
            })
          }
          case "/api/workflows/development/discard": {
            const previous = activeDevelopmentSession
            activeDevelopmentSession = null
            return json(route, { session: previous })
          }
          case "/api/workflows/development/test": {
            const testBody = request.postDataJSON() as {
              async: boolean
              prompt?: string
              target_ref?: string
              yaml?: string
              inputs?: { ticket?: string }
              secrets?: Record<string, string>
              session?: string
              delivery?: Record<string, unknown>
              event_id?: string
            }
            if (testBody.event_id) {
              expect(testBody).toEqual({
                async: true,
                prompt: workflowDraftSession.prompt,
                target_ref: workflowDraftSession.target_workflow_ref,
                yaml: workflowEventDraftYAML,
                event_id: eventResponse.id,
              })
              activeDevelopmentSession = {
                ...workflowDraftSession,
                status: "testing",
                yaml: workflowEventDraftYAML,
                last_test: {
                  ...workflowDraftLastTest,
                  draft_key: workflowDraftKey(
                    workflowDraftSession.target_workflow_ref,
                    workflowEventDraftYAML,
                  ),
                  event_id: eventResponse.id,
                  status: "running",
                },
              }
              runs = [
                runningDraftWorkflowRun,
                ...runs.filter((run) => run.id !== "wr_draft"),
              ]
              return json(route, {
                session: activeDevelopmentSession,
                result: {
                  run_id: draftWorkflowRun.id,
                  status: "running",
                },
              })
            }
            expect(testBody).toMatchObject({
              async: true,
              session: "workflow:draft",
              delivery: {
                channel: "telegram",
                chat_id: "support",
              },
            })
            if (testBody.inputs?.ticket === "Trigger failure") {
              activeDevelopmentSession = {
                ...workflowDraftSession,
                status: "editing",
                last_test: {
                  ...workflowDraftLastTest,
                  run_id: failedDraftWorkflowRun.id,
                  status: "failed",
                  error: "agent step failed",
                },
              }
              runs = [
                failedDraftWorkflowRun,
                ...runs.filter((run) => run.id !== "wr_draft_failed"),
              ]
              return json(route, {
                session: activeDevelopmentSession,
                result: {
                  run_id: failedDraftWorkflowRun.id,
                  status: "failed",
                  error: "agent step failed",
                },
                error: "agent step failed",
              })
            }
            expect(testBody).toMatchObject({
              inputs: { ticket: "Printer is offline" },
            })
            activeDevelopmentSession = {
              ...workflowDraftSession,
              session_revision: "opaque-session-testing",
              status: "testing",
              last_test: {
                ...workflowDraftLastTest,
                status: "running",
              },
            }
            runs = [
              runningDraftWorkflowRun,
              ...runs.filter((run) => run.id !== "wr_draft"),
            ]
            completeDraftViaPolling = options.completeDraftViaPolling === true
            return json(route, {
              session: activeDevelopmentSession,
              result: {
                run_id: draftWorkflowRun.id,
                status: "running",
              },
            })
          }
          case "/api/workflows/development/publish": {
            const publishSession = activeDevelopmentSession
            expect(reviseRequestCount).toBe(0)
            expect(request.postDataJSON()).toEqual({
              session_id: publishSession?.id,
              expected_session_revision: publishSession?.session_revision,
              expected_draft_revision: publishSession?.draft_revision,
              expected_base_target_revision:
                publishSession?.base_target_revision,
              expected_dependency_revision: "opaque-dependency-revision",
            })
            if (
              publishSession?.last_test?.status !== "succeeded" ||
              publishSession.last_test.draft_key !==
                currentDraftKey(publishSession) ||
              publishSession.last_test.draft_revision !==
                publishSession.draft_revision
            ) {
              return json(
                route,
                {
                  error:
                    "workflow draft must pass a current test run before publish",
                },
                409,
              )
            }
            activeDevelopmentSession = null
            // Publish stamps the complete compatibility catalog against the
            // current runtime, so the newly routed definition is immediately
            // runnable in the same way as the production response.
            workflowsRevalidated = true
            if (
              !workflowDefinitions.some(
                (workflow) =>
                  workflow.ref === workflowDraftSession.target_workflow_ref,
              )
            ) {
              workflowDefinitions = [
                ...workflowDefinitions,
                supportTriageWorkflowDefinition,
              ]
            }
            return json(route, {
              workflow_id:
                smokeWorkflowDefinitionIDs[
                  workflowDraftSession.target_workflow_ref
                ],
              workflow_ref: workflowDraftSession.target_workflow_ref,
              session: publishSession,
            })
          }
          case "/api/workflows/run":
            expect(request.postDataJSON()).toMatchObject({
              async: true,
              ref: "workflows/support-triage.yml",
              expected_dependency_revision: "opaque-dependency-revision",
              inputs: { ticket: "Printer is offline" },
              session: "workflow:manual",
              delivery: {
                channel: "telegram",
                chat_id: "support",
              },
            })
            runs = [
              runningManualWorkflowRun,
              ...runs.filter((run) => run.id !== "wr_manual"),
            ]
            return json(route, {
              run_id: manualWorkflowRun.id,
              status: "running",
            })
          case "/api/workflows/runs/wr_test/retry":
            expect(request.postDataJSON()).toMatchObject({
              expected_dependency_revision: "opaque-dependency-revision",
              secrets: { token: "retry-token" },
            })
            runs = [
              retryWorkflowRun,
              ...runs.filter((run) => run.id !== "wr_retry"),
            ]
            return json(route, {
              run_id: retryWorkflowRun.id,
              status: retryWorkflowRun.status,
            })
          case "/api/workflows/runs/wr_cancel/cancel": {
            const body = request.postDataJSON() as { reason: string }
            expect(body).toEqual({ reason: "operator intervention" })
            options.workflowCancelReasons?.push(body.reason)
            currentCancelableWorkflowRun = {
              ...currentCancelableWorkflowRun,
              status: "canceled",
              cancel_reason: body.reason,
              cancel_requested_at: "2026-07-16T12:05:00Z",
              completed_at: "2026-07-16T12:05:00Z",
              updated_at: "2026-07-16T12:05:00Z",
            }
            return json(route, currentCancelableWorkflowRun)
          }
          case "/api/workflows/revalidate":
            workflowsRevalidated = true
            return json(route, compatibilityResponse())
          case "/api/workflows/compatibility":
            return json(route, compatibilityResponse())
          case "/api/workflows/reload":
            return json(route, {
              reloaded_at: "2026-07-16T12:00:00Z",
              workflows: workflowDefinitions,
              errors: [],
            })
          case `/api/events/${eventResponse.id}/replay`:
            expect(request.postData()).toBe("{}")
            return json(
              route,
              {
                event: {
                  ...eventResponse,
                  id: replayEventID,
                  replay_of: eventResponse.id,
                  routing: {
                    ...eventResponse.routing,
                    status: "pending",
                    attempts: 0,
                  },
                },
              },
              201,
            )
          case "/api/development-workspaces/repositories/resolve":
            return json(route, {
              identity: "https://github.com|200",
              name: "other/repo",
              default_branch: "main",
              can_implement: true,
            })
          default:
            return json(route, { status: "ok" })
        }
      }

      if (method !== "GET") {
        return json(route, { status: "ok" })
      }

      const accountDetailMatch = path.match(/^\/api\/accounts\/([^/]+)$/)
      if (accountDetailMatch && accountDetailMatch[1] !== "models") {
        const id = decodeURIComponent(accountDetailMatch[1])
        const account = currentAccounts.find((candidate) => candidate.id === id)
        return account
          ? json(route, { account })
          : json(
              route,
              { code: "account_not_found", message: "Account not found" },
              404,
            )
      }

      const accountRouterDetailMatch = path.match(
        /^\/api\/account-routers\/([^/]+)$/,
      )
      if (accountRouterDetailMatch) {
        const id = decodeURIComponent(accountRouterDetailMatch[1])
        const accountRouter = currentAccountRouters.find(
          (candidate) => candidate.id === id,
        )
        return accountRouter
          ? json(route, {
              account_router: accountRouter,
              config_revision: "account-router-revision-1",
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

      const workflowDefinitionDetailMatch = path.match(
        /^\/api\/workflows\/definitions\/([^/]+)$/,
      )
      if (workflowDefinitionDetailMatch) {
        const id = decodeURIComponent(workflowDefinitionDetailMatch[1])
        const workflow = workflowDefinitions
          .map((candidate) =>
            smokeWorkflowDefinitionSummary(
              candidate,
              workflowsRevalidated ? "valid" : "pending_revalidation",
            ),
          )
          .find((candidate) => candidate.id === id)
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

      const skillDetailMatch = path.match(/^\/api\/skills\/([^/]+)$/)
      if (
        skillDetailMatch &&
        !["search", "install", "import"].includes(skillDetailMatch[1])
      ) {
        const id = decodeURIComponent(skillDetailMatch[1])
        const index = currentSkills.findIndex(
          (skill) => skill.id === id || skill.name === id,
        )
        if (method === "GET") {
          return index >= 0
            ? json(route, {
                ...currentSkills[index],
                content: `# ${currentSkills[index].name}\n\n${currentSkills[index].description}`,
              })
            : json(
                route,
                { code: "not_found", message: "Skill not found" },
                404,
              )
        }
      }

      const toolDetailMatch = path.match(/^\/api\/tools\/([^/]+)$/)
      if (
        method === "GET" &&
        toolDetailMatch &&
        !["web-search-config", "adaptation", "thread-policy"].includes(
          toolDetailMatch[1],
        )
      ) {
        const id = decodeURIComponent(toolDetailMatch[1])
        const tool = currentTools.find(
          (candidate) => candidate.id === id || candidate.name === id,
        )
        return tool
          ? json(route, { tool })
          : json(route, { code: "not_found", message: "Tool not found" }, 404)
      }

      const templateInspectionMatch = path.match(
        /^\/api\/workflows\/templates\/([^/]+)\/inspect$/,
      )
      if (templateInspectionMatch) {
        const templateName = decodeURIComponent(templateInspectionMatch[1])
        expect(templateName.trim()).not.toBe("")
        expect(request.postData()).toBeNull()
        options.workflowInspectionRequests?.push({
          method,
          path,
          body: null,
        })
        return json(
          route,
          workflowDefinitionInspection({
            kind: "template",
            template_name: templateName,
          }),
        )
      }

      switch (path) {
        case "/api/auth/status":
          return json(route, { authenticated: true, initialized: true })
        case "/api/gateway/status":
          return json(route, {
            gateway_status: options.gatewayRunning ? "running" : "stopped",
            gateway_start_allowed: true,
            gateway_restart_required: false,
            boot_default_model: "gpt-4o-mini",
            config_default_model: "gpt-4o-mini",
          })
        case "/api/gateway/logs":
          return json(route, { logs: [], log_total: 0, log_run_id: 1 })
        case "/api/channels/catalog":
          return json(route, channelCatalogResponse)
        case "/api/config":
          return json(route, {
            channels: {
              telegram: { enabled: true },
              discord: { enabled: false },
            },
            channel_list: {
              telegram: { enabled: true, type: "telegram", settings: {} },
              discord: { enabled: false, type: "discord", settings: {} },
              deltachat: {
                enabled: true,
                type: "deltachat",
                settings: { email: "events@example.test" },
              },
            },
            gateway: { host: "127.0.0.1", port: 18789 },
            events: {
              ingress: {
                enabled: true,
                retention_days: 30,
                max_payload_bytes: 1048576,
                redact_fields: ["authorization"],
                webhooks: {
                  github: {
                    enabled: true,
                    format: "github",
                    secret: "[NOT_HERE]",
                  },
                },
                channels: {
                  deltachat: {
                    enabled: true,
                    source: "email",
                    mode: "mirror",
                  },
                },
              },
            },
          })
        case "/api/accounts":
          return json(route, {
            accounts: currentAccounts,
            total: currentAccounts.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY provider ASC, id ASC",
            query_schema: mockCollectionSchemas.accounts,
          })
        case "/api/account-routers":
          return json(route, {
            account_routers: currentAccountRouters.map(accountRouterSummary),
            total: currentAccountRouters.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: mockCollectionSchemas.accountRouters,
            config_revision: "account-router-revision-1",
          })
        case "/api/accounts/models":
          return json(route, options.modelResponse ?? modelResponse)
        case "/api/model-aliases": {
          const models = (options.modelResponse ??
            modelResponse) as typeof modelResponse
          return json(route, {
            model_aliases: models.model_aliases.map((alias) => ({
              name: alias.name,
              model: alias.model,
              override_count: Object.keys(alias.account_overrides ?? {}).length,
              disabled_account_count: alias.disabled_accounts?.length ?? 0,
            })),
            total: models.model_aliases.length,
            next_cursor: "",
            canonical_query: "ALL ORDER BY name ASC",
            query_schema: mockCollectionSchemas.aliases,
            config_revision: models.revision,
          })
        }
        case "/api/model-routers": {
          const models = (options.modelResponse ??
            modelResponse) as typeof modelResponse
          const routers = models.models
            .map((model) => model.model_router)
            .filter((router) => router != null)
            .map((router) => ({
              name: router.name ?? "",
              enabled: router.enabled ?? false,
              entry: router.entry ?? "",
              block_count: router.blocks?.length ?? 0,
              rule_count:
                router.blocks?.reduce(
                  (count, block) => count + (block.rules?.length ?? 0),
                  0,
                ) ?? 0,
            }))
          return json(route, {
            model_routers: routers,
            total: routers.length,
            next_cursor: "",
            canonical_query: "ALL ORDER BY name ASC",
            query_schema: mockCollectionSchemas.routers,
            config_revision: models.revision,
          })
        }
        case "/api/accounts/models/catalog":
          return json(route, { entries: [], total: 0 })
        case "/api/oauth/providers":
          return json(route, { providers: currentOAuthProviders })
        case "/api/oauth/codex-account-limits":
          return json(
            route,
            options.codexAccountLimits ?? {
              accounts: [
                {
                  id: "openai:primary",
                  provider: "openai",
                  default: true,
                  email: "primary@example.test",
                  account_id: "acct-primary",
                  plan: "pro",
                  limits_status: "available",
                  entries: [
                    {
                      name: "codex",
                      status: "available",
                      window: "weekly",
                      used_percent: 64,
                      refreshes_at: "2026-07-28 13:05:32 -04:00",
                    },
                  ],
                },
              ],
            },
          )
        case "/api/sessions":
          return json(route, [])
        case "/api/tools":
          return json(route, {
            tools: currentTools,
            total: currentTools.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ??
              "ORDER BY category ASC, name ASC",
            query_schema: mockCollectionSchemas.tools,
          })
        case "/api/mcp":
          return json(
            route,
            options.statefulMCP ? currentMCPResponse : mcpResponse,
          )
        case "/api/mcp/servers": {
          const response = options.statefulMCP
            ? currentMCPResponse
            : mcpResponse
          return json(route, {
            servers: response.servers,
            total: response.servers.length,
            next_cursor: "",
            canonical_query: "ORDER BY name ASC",
            query_schema: mockCollectionSchemas.mcp,
            config_revision: "mcp-revision-1",
          })
        }
        case "/api/git-workspaces":
          return json(route, gitWorkspaceResponse)
        case "/api/agents":
          return json(route, currentAgentsResponse())
        case "/api/development-workspaces":
          return json(route, {
            ...developmentWorkspaceCollectionResponse,
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY updated DESC",
          })
        case "/api/development-workspaces/repositories":
          return json(route, {
            repositories: [
              {
                identity: "https://github.com|100",
                name: "octo/repo",
                default_branch: "main",
                can_implement: true,
              },
            ],
          })
        case "/api/development-workspaces/repositories/resolve":
          return json(route, {
            identity: "https://github.com|200",
            name: "other/repo",
            default_branch: "main",
            can_implement: true,
          })
        case "/api/repository-reviews":
          return json(route, { repositories: [] })
        case "/api/repository-reviews/automations":
          return json(route, { automations: [] })
        case "/api/repository-reviews/profiles":
          return json(route, { profiles: [] })
        case "/api/repository-reviews/automation-options":
          return json(
            route,
            options.repositoryReviewAutomationOptions ?? {
              models: [],
              accounts: [],
            },
          )
        case "/api/model-evaluations":
          return json(route, {
            evaluations: [],
            total: 0,
            next_cursor: "",
            canonical_query: "ORDER BY updated DESC",
            query_schema: mockCollectionSchemas.evaluations,
          })
        case "/api/model-evaluations/options":
          return json(route, {
            models: [],
            repositories: [],
            code_types: ["hotpath-code", "code", "test", "bench-test"],
            default_files_per_language: 20,
            max_files_per_language: 20,
            max_candidate_models: 8,
          })
        case `/api/development-workspaces/${developmentWorkspaceID}`:
          return json(route, developmentWorkspaceAggregate)
        case `/api/development-workspaces/${developmentWorkspaceID}/conversation/messages`:
          return json(route, { revision: 0, messages: [] })
        case "/api/development/workflow-configurations":
          return json(route, prLifecycleWorkflowConfigurations)
        case "/api/development/workflow-configurations/items":
          return json(route, {
            ...prLifecycleWorkflowConfigurationCollection,
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
          })
        case "/api/development/workflow-configurations/items/default":
          return json(route, prLifecycleWorkflowConfigurationDetail("default"))
        case "/api/development/workflow-configurations/items/editable":
          return json(route, prLifecycleWorkflowConfigurationDetail("editable"))
        case "/api/development/repositories":
          return json(route, prLifecycleRepositoryAssignments)
        case "/api/development/repository-assignments":
          return json(route, {
            ...prLifecycleRepositoryAssignmentCollection,
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY repository ASC",
          })
        case `/api/development/repository-assignments/${smokeDevelopmentAdminIDs.repositoryAssignment}`:
          return json(route, prLifecycleRepositoryAssignmentDetail)
        case "/api/notifications":
          return json(route, {
            notifications: [developmentNotification],
            counts: { open: 1, unread: 1, snoozed: 0 },
          })
        case `/api/notifications/${developmentNotificationID}`:
          return json(route, developmentNotification)
        case `/api/notifications/${developmentNotificationID}/neighbors`:
          return json(route, {})
        case "/api/notification-views":
          return json(route, notificationViews)
        case "/api/notification-settings":
          return json(route, {
            include_repository_in_push: false,
            vapid_public_key: "test-public-key",
            version: 1,
          })
        case "/api/push-subscriptions":
          return json(route, { devices: [] })
        case "/api/notifications/events/stream":
          return route.fulfill({
            status: 200,
            contentType: "text/event-stream",
            body: "",
          })
        case "/api/events":
          return json(route, { events: [eventResponse] })
        case "/api/events/dispatches":
          return json(route, { dispatches: [eventDispatchResponse] })
        case `/api/events/dispatches/${eventDispatchResponse.id}`:
          return json(route, eventDispatchResponse)
        case `/api/events/${eventResponse.id}`:
          return json(route, eventResponse)
        case `/api/events/${eventResponse.id}/payload`:
          options.workflowEventPayloadRequests?.push(path)
          return route.fulfill({
            status: 200,
            contentType: "application/json",
            body: eventPayloadText,
          })
        case "/api/workflows":
          return json(route, {
            workflows: options.nullableWorkflowPayloads
              ? null
              : workflowDefinitions,
            compatibility: options.nullableWorkflowPayloads
              ? {
                  ...compatibilityResponse(),
                  workflows: null,
                  counts: null,
                }
              : compatibilityResponse(),
          })
        case "/api/workflows/definitions": {
          const workflows = options.nullableWorkflowPayloads
            ? []
            : workflowDefinitions.map((workflow) =>
                smokeWorkflowDefinitionSummary(
                  workflow,
                  workflowsRevalidated ? "valid" : "pending_revalidation",
                ),
              )
          return json(route, {
            workflows,
            total: workflows.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY ref ASC",
            query_schema: mockCollectionSchemas.workflowDefinitions,
          })
        }
        case "/api/workflows/compatibility":
          return json(
            route,
            options.nullableWorkflowPayloads
              ? {
                  ...compatibilityResponse(),
                  workflows: null,
                  counts: null,
                }
              : compatibilityResponse(),
          )
        case "/api/workflows/settings":
          return json(route, workflowSettingsResponse)
        case "/api/workflows/development":
          return json(route, { session: activeDevelopmentSession })
        case "/api/workflows/authoring/capabilities":
          options.workflowCapabilityRequests?.push({ method, path })
          return json(route, workflowAuthoringCapabilities())
        case "/api/workflows/templates":
          return json(route, {
            templates: [
              {
                name: "code-review",
                ref: "workflows/code-review.yml",
                state: "available",
              },
              {
                name: "github-issue-triage",
                ref: "workflows/github-issue-triage.yml",
                state: "modified",
              },
            ],
          })
        case "/api/workflows/runs":
          if (completeDraftViaPolling) {
            activeDevelopmentSession = {
              ...workflowDraftSession,
              session_revision: "opaque-session-tested",
              status: "ready_to_publish",
              last_test: workflowDraftLastTest,
            }
            runs = [
              draftWorkflowRun,
              ...runs.filter((run) => run.id !== "wr_draft"),
            ]
            completeDraftViaPolling = false
          }
          return json(route, {
            runs: runs.map((run) => ({
              ...run,
              workflow_id: smokeWorkflowDefinitionIDs[run.workflow_ref],
            })),
            total: runs.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY created DESC",
            query_schema: mockCollectionSchemas.workflowRuns,
          })
        case "/api/workflows/runs/wr_nulls":
          return json(route, {
            ...nullableWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[nullableWorkflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_test":
          return json(route, {
            ...workflowRun,
            workflow_id: smokeWorkflowDefinitionIDs[workflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_retry":
          return json(route, {
            ...retryWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[retryWorkflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_lifecycle":
          return json(route, {
            ...lifecycleWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[lifecycleWorkflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_cancel":
          return json(route, {
            ...currentCancelableWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[
                currentCancelableWorkflowRun.workflow_ref
              ],
          })
        case "/api/workflows/runs/wr_draft":
          return json(route, {
            ...draftWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[draftWorkflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_draft_failed":
          return json(route, {
            ...failedDraftWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[failedDraftWorkflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_manual":
          return json(route, {
            ...manualWorkflowRun,
            workflow_id:
              smokeWorkflowDefinitionIDs[manualWorkflowRun.workflow_ref],
          })
        case "/api/workflows/runs/wr_test/events":
          return json(route, {
            run_id: "wr_test",
            events: [
              {
                time: "2026-07-16T12:00:00Z",
                kind: "workflow.run.start",
                run_id: "wr_test",
              },
              {
                time: "2026-07-16T12:00:01Z",
                kind: "workflow.run.end",
                run_id: "wr_test",
              },
            ],
          })
        case "/api/workflows/runs/wr_nulls/events":
          return json(route, {
            run_id: "wr_nulls",
            events: null,
          })
        case "/api/workflows/runs/wr_retry/events":
          return json(route, {
            run_id: "wr_retry",
            events: [
              {
                time: "2026-07-16T12:00:02Z",
                kind: "workflow.run.start",
                run_id: "wr_retry",
              },
              {
                time: "2026-07-16T12:00:03Z",
                kind: "workflow.run.end",
                run_id: "wr_retry",
                payload: {
                  result: "retry event",
                },
              },
            ],
          })
        case "/api/workflows/runs/wr_lifecycle/events":
          return json(route, {
            run_id: "wr_lifecycle",
            events: [],
          })
        case "/api/workflows/runs/wr_cancel/events":
          return json(route, {
            run_id: "wr_cancel",
            events: [],
          })
        case "/api/workflows/runs/wr_draft/events":
        case "/api/workflows/runs/wr_draft_failed/events":
        case "/api/workflows/runs/wr_manual/events": {
          const runID = path.split("/")[4]
          const eventResult =
            runID === "wr_manual"
              ? "manual event"
              : runID === "wr_draft_failed"
                ? "draft failure event"
                : "draft event"
          return json(route, {
            run_id: runID,
            events: [
              {
                time: "2026-07-16T12:00:00Z",
                kind: "workflow.run.start",
                run_id: runID,
                payload: {
                  source: "dashboard",
                },
              },
              {
                time: "2026-07-16T12:00:01Z",
                kind: "workflow.run.end",
                run_id: runID,
                job_id: "triage",
                step_id: "summarize",
                message: "Workflow completed",
                payload: {
                  result: eventResult,
                },
              },
            ],
          })
        }
        case "/api/workflows/runs/wr_test/events/stream":
          return sse(route, [
            {
              time: "2026-07-16T12:00:02Z",
              kind: "workflow.run.end",
              run_id: "wr_test",
              payload: {
                streamed: "test stream",
              },
            },
          ])
        case "/api/workflows/runs/wr_nulls/events/stream":
          return sse(route, [])
        case "/api/workflows/runs/wr_retry/events/stream":
          return sse(route, [
            {
              time: "2026-07-16T12:00:04Z",
              kind: "workflow.run.end",
              run_id: "wr_retry",
              payload: {
                streamed: "retry stream",
              },
            },
          ])
        case "/api/workflows/runs/wr_lifecycle/events/stream":
        case "/api/workflows/runs/wr_cancel/events/stream":
          return sse(route, [])
        case "/api/workflows/runs/wr_draft/events/stream":
        case "/api/workflows/runs/wr_draft_failed/events/stream":
        case "/api/workflows/runs/wr_manual/events/stream": {
          const runID = path.split("/")[4]
          const streamResult =
            runID === "wr_manual"
              ? "manual stream"
              : runID === "wr_draft_failed"
                ? "draft failure stream"
                : "draft stream"
          if (runID === "wr_draft") {
            activeDevelopmentSession =
              activeDevelopmentSession?.last_test?.event_id != null
                ? {
                    ...activeDevelopmentSession,
                    session_revision: "opaque-session-event-tested",
                    status: "ready_to_publish",
                    last_test: {
                      ...activeDevelopmentSession.last_test,
                      status: "succeeded",
                    },
                  }
                : {
                    ...workflowDraftSession,
                    session_revision: "opaque-session-tested",
                    status: "ready_to_publish",
                    last_test: workflowDraftLastTest,
                  }
            runs = [
              draftWorkflowRun,
              ...runs.filter((run) => run.id !== "wr_draft"),
            ]
          } else if (runID === "wr_manual") {
            runs = [
              manualWorkflowRun,
              ...runs.filter((run) => run.id !== "wr_manual"),
            ]
          }
          return sse(route, [
            {
              time: "2026-07-16T12:00:02Z",
              kind: "workflow.step.end",
              run_id: runID,
              job_id: "triage",
              step_id: "summarize",
              payload: {
                streamed: streamResult,
              },
            },
            {
              time: "2026-07-16T12:00:03Z",
              kind: "workflow.run.end",
              run_id: runID,
              payload: {
                streamed: streamResult,
              },
            },
          ])
        }
        case "/api/workflows/runs/wr_test/graph":
          return json(route, {
            run_id: "wr_test",
            nodes: [
              {
                id: "wr_test",
                workflow_ref: "workflows/summarize-text.yml",
                status: "succeeded",
              },
            ],
            edges: [],
          })
        case "/api/workflows/runs/wr_nulls/graph":
          return json(route, {
            run_id: "wr_nulls",
            nodes: null,
            edges: null,
          })
        case "/api/workflows/runs/wr_retry/graph":
          return json(route, {
            run_id: "wr_retry",
            nodes: [
              {
                id: "wr_retry",
                workflow_ref: retryWorkflowRun.workflow_ref,
                status: retryWorkflowRun.status,
                retry_of_run_id: "wr_test",
              },
            ],
            edges: [
              {
                from: "wr_test",
                to: "wr_retry",
                kind: "retry",
              },
            ],
          })
        case "/api/workflows/runs/wr_lifecycle/graph":
          return json(route, {
            run_id: "wr_lifecycle",
            nodes: [],
            edges: [],
          })
        case "/api/workflows/runs/wr_cancel/graph":
          return json(route, {
            run_id: "wr_cancel",
            nodes: [],
            edges: [],
          })
        case "/api/workflows/runs/wr_draft/graph":
        case "/api/workflows/runs/wr_draft_failed/graph":
        case "/api/workflows/runs/wr_manual/graph": {
          const runID = path.split("/")[4]
          const run = runByID(runID)
          return json(route, {
            run_id: runID,
            nodes: [
              {
                id: runID,
                workflow_ref: run.workflow_ref,
                status: run.status,
              },
            ],
            edges: [],
          })
        }
        case "/api/tools/web-search-config":
          return json(route, webSearchConfigResponse)
        case "/api/tools/adaptation":
          return json(route, toolAdaptationResponse)
        case "/api/skills":
          return json(route, {
            skills: currentSkills,
            total: currentSkills.length,
            next_cursor: "",
            canonical_query:
              url.searchParams.get("query") ?? "ORDER BY name ASC",
            query_schema: mockCollectionSchemas.skills,
          })
        case "/api/skills/search":
          return json(route, {
            results: [],
            limit: Number(url.searchParams.get("limit") ?? 20),
            offset: Number(url.searchParams.get("offset") ?? 0),
            has_more: false,
          })
        case "/api/system/autostart":
          return json(route, {
            enabled: false,
            supported: true,
            platform: "linux",
          })
        case "/api/system/launcher-config":
          return json(route, {
            port: 18800,
            public: false,
            allowed_cidrs: [],
            allow_localhost_bypass: true,
            trusted_proxy_cidrs: [],
          })
        case "/api/system/version":
          return json(route, {
            version: "test",
            git_commit: "test",
            build_time: "test",
            go_version: "go1.25",
          })
        default:
          return json(route, {})
      }
    },
  )
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  })
}

async function sse(route: Route, events: Array<Record<string, unknown>>) {
  await route.fulfill({
    status: 200,
    contentType: "text/event-stream",
    body: events
      .map(
        (event) => `event: ${event.kind}\ndata: ${JSON.stringify(event)}\n\n`,
      )
      .join(""),
  })
}

function collectPageErrors(page: Page) {
  const errors: string[] = []
  page.on("console", (message) => {
    if (message.type() === "error") {
      errors.push(message.text())
    }
  })
  page.on("pageerror", (error) => {
    errors.push(error.message)
  })
  return errors
}

async function expectNoHorizontalOverflow(page: Page) {
  const hasHorizontalOverflow = await page.evaluate(() => {
    const doc = document.documentElement
    const body = document.body
    const scrollWidth = Math.max(doc.scrollWidth, body.scrollWidth)
    const clientWidth = Math.max(doc.clientWidth, window.innerWidth)
    return scrollWidth > clientWidth + 1
  })

  expect(hasHorizontalOverflow).toBe(false)
}

async function expectNoPersistentLoadingOrLoadError(page: Page) {
  const unresolvedState = page
    .locator("main")
    .getByText(
      /^(?:Loading\b|(?:Failed|Unable) to load\b|.*\b(?:is|are) unavailable\.?$)/i,
    )
  await expect(unresolvedState).toHaveCount(0)
}

async function expectElementFitsViewport(
  page: Page,
  selector: string,
  label: string,
) {
  await expect
    .poll(
      () =>
        page.locator(selector).evaluate((element) => {
          const rect = element.getBoundingClientRect()
          const tolerance = 1
          return (
            rect.left >= -tolerance &&
            rect.top >= -tolerance &&
            rect.right <= window.innerWidth + tolerance &&
            rect.bottom <= window.innerHeight + tolerance
          )
        }),
      { message: `${label} should fit in the viewport` },
    )
    .toBe(true)
}

async function expectNoSeriousA11yViolations(page: Page) {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze()
  const blocking = results.violations.filter(
    (violation) =>
      violation.impact === "serious" || violation.impact === "critical",
  )

  expect(
    blocking.map((violation) => ({
      id: violation.id,
      impact: violation.impact,
      nodes: violation.nodes.map((node) => ({
        target: node.target.join(" "),
        html: node.html,
      })),
    })),
  ).toEqual([])
}

async function confirmTriggerExecutionReview(page: Page) {
  const dialog = page.getByRole("dialog", {
    name: "Review trigger execution",
  })
  await expect(dialog).toBeVisible()
  const confirm = dialog.getByRole("button", {
    name: "Confirm and execute",
  })
  await expect(confirm).toBeDisabled()
  await dialog
    .getByRole("switch", {
      name: "I reviewed this server simulation and its possible effects",
    })
    .click()
  await expect(confirm).toBeEnabled()
  await confirm.click()
  await expect(dialog).toBeHidden()
}

async function gotoMockedRoute(
  page: Page,
  routePath: string,
  options?: MockLauncherApiOptions,
) {
  await page.addInitScript(() => {
    window.localStorage.setItem(
      "picoclaw-tour-state",
      JSON.stringify({ currentStep: "completed", isActive: false }),
    )
  })
  await mockLauncherApis(page, options)
  await page.goto(routePath)
  await expect(page.getByRole("banner")).toBeVisible()
  await expect(page.locator("#main-content")).toBeVisible()
}

async function mockRepositoryReviewAutomationRequest(
  route: Route,
  url: URL,
  state: RepositoryReviewMockState,
  requests?: MockLauncherApiOptions["repositoryReviewRequests"],
) {
  const request = route.request()
  const method = request.method()
  const path = url.pathname
  const rawBody = request.postData()
  const body = rawBody
    ? (request.postDataJSON() as Record<string, unknown>)
    : null
  if (method !== "GET") requests?.push({ method, path, body })

  const automationRoot = `/api/repository-reviews/automations/${repositoryReviewAutomationID}`
  const capabilities = {
    github: true,
    can_generate: true,
    can_publish: true,
    can_search_issues: true,
    can_link_issue: true,
    can_unlink_issue: true,
    can_replace_issue: true,
    can_edit: true,
    can_delete: true,
    can_regenerate: true,
    can_purge_history: true,
    can_remove_repository: true,
    purge_blockers: [],
    purge_summary: {
      repository_version: state.summary.version,
      ledger_fence: "rplf_ui_smoke",
      raw_findings: state.rawFindings.length,
      deduplicated_findings: state.findings.length,
      repository_findings: state.repositoryFindings.length,
      issue_previews: state.issues.length,
      external_issue_associations: state.repositoryFindings.filter((finding) =>
        Boolean(finding.issue.url),
      ).length,
    },
  }
  const findFinding = (findingID: string) =>
    state.findings.find((finding) => finding.id === findingID)
  const findRepositoryFinding = (findingID: string) =>
    state.repositoryFindings.find((finding) => finding.id === findingID)
  const findIssue = (draftID: string) =>
    state.issues.find((issue) => issue.id === draftID)
  const contextsFor = (finding: RepositoryReviewFinding) =>
    state.contexts.filter((context) => finding.context_ids.includes(context.id))
  const issueFor = (finding: RepositoryReviewFinding) =>
    finding.issue_draft_id
      ? state.issues.find((issue) => issue.id === finding.issue_draft_id)
      : undefined
  const occurrencesFor = (repositoryFinding: RepositoryFinding) =>
    repositoryFinding.review_finding_ids.flatMap((findingID) => {
      const finding = findFinding(findingID)
      return finding ? [finding] : []
    })
  const issueForOccurrences = (findings: RepositoryReviewFinding[]) =>
    findings.map(issueFor).find((issue) => issue !== undefined)
  const findingDetail = (finding: RepositoryReviewFinding) => {
    const repositoryFinding = finding.repository_finding_id
      ? findRepositoryFinding(finding.repository_finding_id)
      : undefined
    const issue =
      issueFor(finding) ||
      (repositoryFinding
        ? issueForOccurrences(occurrencesFor(repositoryFinding))
        : undefined)
    return {
      automation: state.automation,
      repository: state.summary,
      finding,
      contexts: contextsFor(finding),
      raw_source_total: state.rawFindings.filter(
        (source) => source.deduplicated_finding_id === finding.id,
      ).length,
      ...(repositoryFinding ? { repository_finding: repositoryFinding } : {}),
      ...(issue ? { issue } : {}),
      capabilities,
    }
  }
  const repositoryFindingDetail = (repositoryFinding: RepositoryFinding) => {
    const occurrences = occurrencesFor(repositoryFinding)
    const finding = occurrences.at(-1)
    const aggregateCanAcceptIssue =
      repositoryFinding.match_state !== "provisional" &&
      (repositoryFinding.lifecycle === "open" ||
        repositoryFinding.lifecycle === "regressed") &&
      repositoryFinding.issue.state === "none"
    const actionFinding = aggregateCanAcceptIssue
      ? occurrences
          .toReversed()
          .find(
            (candidate) =>
              candidate.status === "open" && !candidate.issue_draft_id,
          )
      : undefined
    const issue = issueForOccurrences(occurrences)
    return finding
      ? {
          automation: state.automation,
          repository: state.summary,
          finding,
          ...(actionFinding ? { action_finding: actionFinding } : {}),
          repository_finding: repositoryFinding,
          occurrences,
          possible_duplicate_findings:
            repositoryFinding.possible_duplicates?.flatMap((duplicate) => {
              const candidate = findRepositoryFinding(duplicate.candidate_id)
              return candidate ? [candidate] : []
            }) ?? [],
          contexts: state.contexts.filter((context) =>
            occurrences.some((occurrence) =>
              occurrence.context_ids.includes(context.id),
            ),
          ),
          ...(issue ? { issue } : {}),
          capabilities: {
            ...capabilities,
            can_generate: Boolean(actionFinding),
            can_search_issues: capabilities.github && Boolean(actionFinding),
            can_link_issue: capabilities.github && Boolean(actionFinding),
          },
        }
      : undefined
  }
  const issueDetail = (issue: RepositoryReviewIssueDraft) => ({
    automation: state.automation,
    repository: state.summary,
    issue,
    finding: state.findings.find((finding) =>
      issue.finding_ids.includes(finding.id),
    ),
    capabilities: {
      ...capabilities,
      can_edit: issue.state === "editing" && issue.canonical !== false,
      can_delete:
        (issue.state === "editing" || issue.state === "failed") &&
        issue.canonical !== false,
      can_regenerate:
        (issue.state === "editing" || issue.state === "failed") &&
        issue.canonical !== false,
      can_publish:
        new Set(["editing", "publishing", "unknown"]).has(issue.state) &&
        issue.canonical !== false,
    },
  })
  const processingHealth = () => ({
    total: state.processingSources.length,
    pending: state.processingSources.filter(
      (source) => source.deduplication_state === "pending",
    ).length,
    processing: state.processingSources.filter(
      (source) => source.deduplication_state === "running",
    ).length,
    failed: state.processingSources.filter(
      (source) => source.deduplication_state === "failed",
    ).length,
    completed: state.processingSources.filter(
      (source) => source.deduplication_state === "completed",
    ).length,
  })
  const findingHealth = () => ({
    run_findings: {
      total: state.findings.length + 3,
      pending: state.automation.status === "completed" ? 0 : 1,
      processing: state.automation.status === "completed" ? 0 : 1,
      failed: 1,
      needs_review: state.findings.filter(
        (finding) => finding.run_finding_status === "needs_review",
      ).length,
      associated_new: state.findings.filter(
        (finding) => finding.run_finding_status === "associated_new",
      ).length,
      associated_existing: state.findings.filter(
        (finding) => finding.run_finding_status === "associated_existing",
      ).length,
      unrepresented: state.automation.status === "completed" ? 1 : 3,
    },
    repository_findings: {
      total: state.repositoryFindings.length,
      provisional: state.repositoryFindings.filter(
        (finding) => finding.match_state === "provisional",
      ).length,
      validation_failed: state.repositoryFindings.filter(
        (finding) => finding.validation_state === "failed",
      ).length,
      issue_conflicts: state.repositoryFindings.filter(
        (finding) => finding.issue.conflict === true,
      ).length,
    },
    findings_processing: processingHealth(),
    historical_consolidation: state.historicalConsolidation,
    updated_at: "2026-08-26T12:05:10Z",
  })
  const processingDetail = (source: RepositoryReviewRawFinding) => {
    const finding = source.deduplicated_finding_id
      ? findFinding(source.deduplicated_finding_id)
      : undefined
    const repositoryFinding = finding?.repository_finding_id
      ? findRepositoryFinding(finding.repository_finding_id)
      : undefined
    return {
      automation: state.automation,
      repository: state.summary,
      source,
      context: state.contexts.find(
        (context) => context.id === source.context_id,
      ),
      ...(finding ? { finding } : {}),
      ...(repositoryFinding ? { repository_finding: repositoryFinding } : {}),
      findings_processing: processingHealth(),
      historical_consolidation: state.historicalConsolidation,
    }
  }

  if (path === "/api/repository-reviews/automations") {
    if (method !== "GET") {
      return json(route, { code: "method_not_allowed" }, 405)
    }
    return json(route, {
      automations: [state.automation],
      total: 1,
      next_cursor: "",
      canonical_query:
        url.searchParams.get("query") ?? "ORDER BY repository ASC",
      query_schema: collectionSchema([
        ["id", "string"],
        ["name", "string"],
        ["repository", "string"],
        ["branch", "string"],
        ["status", "enum"],
        ["progress", "number"],
        ["reviewed", "number"],
        ["raw_findings", "number"],
        ["findings", "number"],
        ["updated", "timestamp"],
      ]),
    })
  }

  if (path === `${automationRoot}/file-attributions` && method === "GET") {
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      file_attributions: [],
      total: 0,
      next_cursor: "",
      canonical_query:
        url.searchParams.get("query") ??
        "ALL ORDER BY path ASC, focus ASC, reviewer ASC",
      query_schema: collectionSchema([
        ["path", "string"],
        ["commit", "string"],
        ["blob", "string"],
        ["focus", "enum"],
        ["agent", "string"],
        ["reviewer", "string"],
        ["account", "string"],
        ["model", "string"],
        ["source", "enum"],
        ["attempts", "number"],
        ["runs", "number"],
        ["latest", "timestamp"],
      ]),
    })
  }

  if (path === `${automationRoot}/finding-health` && method === "GET") {
    state.healthReads += 1
    if (state.stagedHealth && state.healthReads === 1) {
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
        updated_at: "2026-08-26T12:00:00Z",
      })
    }
    return json(route, findingHealth())
  }

  if (path === `${automationRoot}/findings-processing` && method === "GET") {
    const query = url.searchParams.get("query") ?? "ALL ORDER BY updated DESC"
    const stateMatch =
      /\bstate\s*=\s*["']?(pending|running|failed|completed)["']?/iu.exec(query)
    const filtered = state.processingSources
      .filter(
        (source) => !stateMatch || source.deduplication_state === stateMatch[1],
      )
      .toSorted((left, right) =>
        right.updated_at.localeCompare(left.updated_at),
      )
    const cursor = Number(url.searchParams.get("cursor") ?? 0)
    const offset = Number.isSafeInteger(cursor) && cursor >= 0 ? cursor : 0
    const limit = Number(url.searchParams.get("limit") ?? 50)
    const sources = filtered.slice(offset, offset + limit)
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      raw_findings: sources,
      total: filtered.length,
      next_cursor:
        offset + sources.length < filtered.length
          ? String(offset + sources.length)
          : "",
      canonical_query: query,
      query_schema: mockCollectionSchemas.reviewFindingsProcessing,
      findings_processing: processingHealth(),
      historical_consolidation: state.historicalConsolidation,
      capabilities,
    })
  }

  if (
    path === `${automationRoot}/findings-processing/retry` &&
    method === "POST"
  ) {
    const sourceIDs = Array.isArray(body?.source_ids)
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
    for (const sourceID of sourceIDs) {
      const source = state.processingSources.find(
        (candidate) => candidate.id === sourceID,
      )
      if (
        source?.id === repositoryReviewProcessingFailedID &&
        source.deduplication_state === "failed"
      ) {
        source.deduplication_state = "pending"
        source.disposition = "undecided"
        source.failure = undefined
        retriedIDs.push(sourceID)
        continue
      }
      failures.push({
        source_id: sourceID,
        code:
          source?.id === repositoryReviewProcessingOldCampaignID
            ? "historical_replay_required"
            : source
              ? "not_retryable"
              : "not_found",
        message:
          source?.id === repositoryReviewProcessingOldCampaignID
            ? "Historical sources must be retried through historical consolidation."
            : source
              ? "Finding processing source is not retryable."
              : "Finding processing source was not found.",
      })
    }
    return json(
      route,
      {
        retried_ids: retriedIDs,
        failures,
        findings_processing: processingHealth(),
        health: findingHealth(),
      },
      202,
    )
  }

  const processingRetryMatch = path.match(
    new RegExp(`^${automationRoot}/findings-processing/sources/([^/]+)/retry$`),
  )
  if (processingRetryMatch && method === "POST") {
    const source = state.processingSources.find(
      (candidate) =>
        candidate.id === decodeURIComponent(processingRetryMatch[1]!),
    )
    if (!source) return json(route, { code: "not_found" }, 404)
    if (source.deduplication_state === "failed") {
      source.deduplication_state = "pending"
      source.disposition = "undecided"
      source.failure = undefined
    }
    return json(route, processingDetail(source), 202)
  }

  const processingDetailMatch = path.match(
    new RegExp(`^${automationRoot}/findings-processing/sources/([^/]+)$`),
  )
  if (processingDetailMatch && method === "GET") {
    const source = state.processingSources.find(
      (candidate) =>
        candidate.id === decodeURIComponent(processingDetailMatch[1]!),
    )
    return source
      ? json(route, processingDetail(source))
      : json(route, { code: "not_found" }, 404)
  }

  if (
    path === `${automationRoot}/historical-deduplication/retry` &&
    method === "POST"
  ) {
    return json(
      route,
      {
        code: "historical_consolidation_restart_required",
        message: "The saved profile no longer matches this historical replay.",
      },
      409,
    )
  }

  if (
    path === `${automationRoot}/historical-deduplication/restart` &&
    method === "POST"
  ) {
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
    state.historicalConsolidation = {
      required: true,
      status: "pending",
      retryable: false,
    }
    return json(
      route,
      {
        automation: state.automation,
        repository: state.summary,
        historical_deduplication: {
          required: true,
          status: "pending",
        },
      },
      202,
    )
  }

  if (
    (path === `${automationRoot}/findings` ||
      path === `${automationRoot}/run-findings`) &&
    method === "GET"
  ) {
    const cursor = Number(url.searchParams.get("cursor") ?? 0)
    const offset = Number.isSafeInteger(cursor) && cursor >= 0 ? cursor : 0
    const limit = Number(url.searchParams.get("limit") ?? 50)
    const findings = state.findings.slice(offset, offset + limit)
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      findings: findings.map(repositoryReviewRunFindingSummary),
      total: state.findings.length,
      next_cursor:
        offset + findings.length < state.findings.length
          ? String(offset + findings.length)
          : "",
      canonical_query:
        url.searchParams.get("query") ??
        "ALL ORDER BY severity DESC, updated DESC",
      query_schema: mockCollectionSchemas.reviewRunFindings,
      findings_processing: {
        raw_total: state.rawFindings.length,
        pending: state.rawFindings.filter(
          (finding) => finding.deduplication_state === "pending",
        ).length,
        processing: state.rawFindings.filter(
          (finding) => finding.deduplication_state === "running",
        ).length,
        failed: state.rawFindings.filter(
          (finding) => finding.deduplication_state === "failed",
        ).length,
        completed: state.rawFindings.filter(
          (finding) => finding.deduplication_state === "completed",
        ).length,
        new: state.rawFindings.filter(
          (finding) => finding.disposition === "new",
        ).length,
        duplicates: state.rawFindings.filter(
          (finding) => finding.disposition === "duplicate",
        ).length,
      },
      historical_deduplication: { required: false, status: "completed" },
      capabilities,
    })
  }

  if (path === `${automationRoot}/raw-findings` && method === "GET") {
    const cursor = Number(url.searchParams.get("cursor") ?? 0)
    const offset = Number.isSafeInteger(cursor) && cursor >= 0 ? cursor : 0
    const limit = Number(url.searchParams.get("limit") ?? 50)
    const findings = state.rawFindings.slice(offset, offset + limit)
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      raw_findings: findings,
      total: state.rawFindings.length,
      next_cursor:
        offset + findings.length < state.rawFindings.length
          ? String(offset + findings.length)
          : "",
      canonical_query:
        url.searchParams.get("query") ?? "ALL ORDER BY created DESC",
      query_schema: mockCollectionSchemas.reviewRawFindings,
      findings_processing: {
        raw_total: state.rawFindings.length,
        pending: 0,
        processing: 0,
        failed: 0,
        completed: state.rawFindings.length,
        new: state.rawFindings.filter(
          (finding) => finding.disposition === "new",
        ).length,
        duplicates: state.rawFindings.filter(
          (finding) => finding.disposition === "duplicate",
        ).length,
      },
      historical_deduplication: { required: false, status: "completed" },
      capabilities,
    })
  }

  if (path === `${automationRoot}/repository-findings` && method === "GET") {
    const cursor = Number(url.searchParams.get("cursor") ?? 0)
    const offset = Number.isSafeInteger(cursor) && cursor >= 0 ? cursor : 0
    const limit = Number(url.searchParams.get("limit") ?? 50)
    const findings = state.repositoryFindings.slice(offset, offset + limit)
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      repository_findings: findings.map(repositoryFindingCollectionSummary),
      total: state.repositoryFindings.length,
      next_cursor:
        offset + findings.length < state.repositoryFindings.length
          ? String(offset + findings.length)
          : "",
      canonical_query:
        url.searchParams.get("query") ??
        "ALL ORDER BY severity DESC, updated DESC",
      query_schema: mockCollectionSchemas.reviewRepositoryFindings,
      capabilities,
    })
  }

  if (path === `${automationRoot}/findings/status` && method === "POST") {
    const findingIDs = Array.isArray(body?.finding_ids)
      ? body.finding_ids.filter(
          (candidate): candidate is string => typeof candidate === "string",
        )
      : []
    const findings = findingIDs.flatMap((findingID) => {
      const finding = findFinding(findingID)
      if (!finding) return []
      if (finding.run_finding_status === "failed") {
        finding.run_finding_status = "pending"
      }
      return [
        {
          id: finding.id,
          run_finding_status: finding.run_finding_status ?? "pending",
        },
      ]
    })
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      findings,
    })
  }

  const candidateMatch = path.match(
    new RegExp(`^${automationRoot}/findings/([^/]+)/issue-link/candidates$`),
  )
  if (candidateMatch && method === "POST") {
    const finding = findFinding(decodeURIComponent(candidateMatch[1]!))
    if (!finding) return json(route, { code: "not_found" }, 404)
    return json(route, {
      automation: state.automation,
      finding,
      generator_model: state.automation.issue_writer_model,
      generator_account: state.automation.effective_account_ref,
      candidates: [
        {
          id: "github:184",
          number: 184,
          title: "Worker retries cancellation immediately",
          url: "https://github.com/octo/repo/issues/184",
          state: "open",
          labels: ["bug", "queue"],
          score: 0.94,
          rank: 1,
          explanation:
            "The issue names the same worker retry behavior and queue path.",
        },
        {
          id: "github:133",
          number: 133,
          title: "Reduce shutdown retry noise",
          url: "https://github.com/octo/repo/issues/133",
          state: "closed",
          labels: ["reliability"],
          score: 0.68,
          rank: 2,
          explanation:
            "This older issue mentions shutdown noise but not the zero-delay branch.",
        },
      ],
    })
  }

  const linkMatch = path.match(
    new RegExp(`^${automationRoot}/findings/([^/]+)/issue-link$`),
  )
  if (linkMatch) {
    const finding = findFinding(decodeURIComponent(linkMatch[1]!))
    if (!finding) return json(route, { code: "not_found" }, 404)
    if (method === "DELETE") {
      const associated = issueFor(finding)
      if (associated?.origin === "linked") {
        state.issues = state.issues.filter(
          (issue) => issue.id !== associated.id,
        )
        finding.issue_draft_id = undefined
        finding.status = "open"
        finding.version += 1
      }
      return json(route, findingDetail(finding))
    }
    if (method === "POST") {
      const issueURL = String(body?.issue_url ?? "")
      const number = Number(issueURL.match(/\/issues\/(\d+)/u)?.[1] ?? 184)
      const previous = issueFor(finding)
      if (previous?.origin === "linked") {
        state.issues = state.issues.filter((issue) => issue.id !== previous.id)
      }
      const linked: RepositoryReviewIssueDraft = {
        id: `rrid_linked_${number}`,
        repository: finding.repository,
        finding_ids: [finding.id],
        origin: "linked",
        canonical: true,
        read_only: true,
        publishable: false,
        deletable: false,
        regeneratable: false,
        unlinkable: true,
        title:
          number === 184
            ? "Worker retries cancellation immediately"
            : `Existing GitHub issue #${number}`,
        body: "This record points to an existing GitHub issue.",
        labels: ["bug", "queue"],
        state: "posted",
        external_id: String(number),
        external_url: issueURL,
        version: 1,
        created_at: "2026-08-26T12:06:00Z",
        updated_at: "2026-08-26T12:06:00Z",
      }
      state.issues.push(linked)
      finding.issue_draft_id = linked.id
      finding.status = "posted"
      finding.version += 1
      return json(route, findingDetail(finding))
    }
  }

  const findingMatch = path.match(
    new RegExp(`^${automationRoot}/findings/([^/]+)$`),
  )
  if (findingMatch) {
    const findingID = decodeURIComponent(findingMatch[1]!)
    const finding = findFinding(findingID)
    if (!finding) {
      return json(route, { code: "not_found" }, 404)
    }
    if (method === "PATCH") {
      finding.status = body?.status === "dismissed" ? "dismissed" : "open"
      finding.version += 1
    }
    return json(route, findingDetail(finding))
  }

  const findingSourcesMatch = path.match(
    new RegExp(`^${automationRoot}/findings/([^/]+)/sources$`),
  )
  if (findingSourcesMatch && method === "GET") {
    const findingID = decodeURIComponent(findingSourcesMatch[1]!)
    const sources = state.rawFindings.filter(
      (source) => source.deduplicated_finding_id === findingID,
    )
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      finding_id: findingID,
      sources,
      offset: 0,
      total: sources.length,
    })
  }

  const rawFindingMatch = path.match(
    new RegExp(`^${automationRoot}/raw-findings/([^/]+)$`),
  )
  if (rawFindingMatch && method === "GET") {
    const requestedID = decodeURIComponent(rawFindingMatch[1]!)
    const source = state.rawFindings.find(
      (candidate) =>
        candidate.id === requestedID ||
        candidate.legacy_finding_id === requestedID,
    )
    if (!source) return json(route, { code: "not_found" }, 404)
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      source,
      context: state.contexts.find(
        (context) => context.id === source.context_id,
      ),
      finding: state.findings.find(
        (finding) => finding.id === source.deduplicated_finding_id,
      ),
    })
  }

  const runFindingMatch = path.match(
    new RegExp(`^${automationRoot}/run-findings/([^/]+)$`),
  )
  if (runFindingMatch && method === "GET") {
    const finding = findFinding(decodeURIComponent(runFindingMatch[1]!))
    return finding
      ? json(route, findingDetail(finding))
      : json(route, { code: "not_found" }, 404)
  }

  const duplicateDecisionMatch = path.match(
    new RegExp(`^${automationRoot}/repository-findings/([^/]+)/duplicates$`),
  )
  if (duplicateDecisionMatch && method === "POST") {
    const provisional = findRepositoryFinding(
      decodeURIComponent(duplicateDecisionMatch[1]!),
    )
    const candidateID = String(body?.candidate_id ?? "")
    const candidate = findRepositoryFinding(candidateID)
    if (!provisional || !candidate) {
      return json(route, { code: "not_found" }, 404)
    }
    if (body?.decision === "merge") {
      state.repositoryFindings = state.repositoryFindings.filter(
        (finding) => finding.id !== provisional.id,
      )
      return json(route, {
        automation: state.automation,
        repository: state.summary,
        repository_finding: candidate,
      })
    }
    provisional.match_state = "new"
    provisional.possible_duplicates = []
    provisional.version += 1
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      repository_finding: provisional,
    })
  }

  const repositoryFindingMatch = path.match(
    new RegExp(`^${automationRoot}/repository-findings/([^/]+)$`),
  )
  if (repositoryFindingMatch && method === "GET") {
    const repositoryFinding = findRepositoryFinding(
      decodeURIComponent(repositoryFindingMatch[1]!),
    )
    const detail = repositoryFinding
      ? repositoryFindingDetail(repositoryFinding)
      : undefined
    return detail
      ? json(route, detail)
      : json(route, { code: "not_found" }, 404)
  }

  if (path === `${automationRoot}/issues/generations` && method === "POST") {
    const generationID = String(body?.generation_id ?? "rig_smoke_generated")
    const requestedFindingIDs = Array.isArray(body?.finding_ids)
      ? body.finding_ids.filter(
          (candidate): candidate is string => typeof candidate === "string",
        )
      : []
    const existing = state.issues.filter(
      (issue) => issue.generation_id === generationID,
    )
    const generated =
      existing.length > 0
        ? existing
        : requestedFindingIDs.flatMap((findingID, index) => {
            const finding = findFinding(findingID)
            if (
              !finding ||
              finding.status !== "open" ||
              finding.issue_draft_id
            ) {
              return []
            }
            const issue: RepositoryReviewIssueDraft = {
              id: `rrid_generated_${state.issues.length + index + 1}`,
              repository: finding.repository,
              finding_ids: [finding.id],
              origin: "ai_generated",
              generation_id: generationID,
              resolved_instructions:
                body?.instructions_mode === "custom" &&
                typeof body.instructions === "string"
                  ? body.instructions
                  : "Write a concise grounded issue with evidence, impact, validation, location, and commit provenance.",
              instructions_mode:
                body?.instructions_mode === "custom" ? "custom" : "default",
              generator_model: state.automation.issue_writer_model,
              generator_account: state.automation.effective_account_ref,
              generator_profile_id: state.automation.profile_id,
              generator_profile_version: state.automation.profile_version,
              canonical: true,
              publishable: true,
              deletable: true,
              regeneratable: true,
              title: finding.title,
              body: `## Evidence\n\n${finding.evidence}\n\n## Impact\n\n${finding.impact}\n\n## Validation\n\n${finding.validation.summary}\n\n## Location\n\n\`${finding.file.path}:${finding.line ?? 1}\` at commit \`${finding.commit_sha}\`.`,
              labels: ["bug"],
              state: "editing",
              version: 1,
              created_at: "2026-08-26T12:06:00Z",
              updated_at: "2026-08-26T12:06:00Z",
            }
            state.issues.push(issue)
            finding.issue_draft_id = issue.id
            finding.version += 1
            return [issue]
          })
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      generation_id: generationID,
      issues: generated,
      results: requestedFindingIDs.map((findingID) => {
        const issue = generated.find((candidate) =>
          candidate.finding_ids.includes(findingID),
        )
        return issue
          ? {
              id: findingID,
              draft_id: issue.id,
              state: issue.state,
              success: true,
            }
          : {
              id: findingID,
              outcome: "failed",
              success: false,
              message: "Finding is not available for generation.",
            }
      }),
    })
  }

  if (path === `${automationRoot}/issues/publish` && method === "POST") {
    const requested = Array.isArray(body?.issues)
      ? (body.issues as Array<{ id?: unknown }>)
      : []
    const results = requested.map((candidate) => {
      const id = typeof candidate.id === "string" ? candidate.id : ""
      const issue = findIssue(id)
      if (!issue) {
        return {
          draft_id: id,
          outcome: "failed",
          success: false,
          message: "Preview not found.",
        }
      }
      issue.state = "posted"
      issue.external_id = String(300 + state.issues.indexOf(issue))
      issue.external_url = `https://github.com/octo/repo/issues/${issue.external_id}`
      issue.publishable = false
      issue.deletable = false
      issue.regeneratable = false
      issue.version += 1
      issue.updated_at = "2026-08-26T12:07:00Z"
      const finding = state.findings.find((candidateFinding) =>
        issue.finding_ids.includes(candidateFinding.id),
      )
      if (finding) {
        finding.status = "posted"
        finding.version += 1
      }
      return {
        draft_id: issue.id,
        state: issue.state,
        outcome: "posted",
        success: true,
        external_url: issue.external_url,
      }
    })
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      issues: state.issues,
      results,
    })
  }

  if (path === `${automationRoot}/issues` && method === "GET") {
    const cursor = Number(url.searchParams.get("cursor") ?? 0)
    const offset = Number.isSafeInteger(cursor) && cursor >= 0 ? cursor : 0
    const limit = Number(url.searchParams.get("limit") ?? 50)
    const generationID = url.searchParams.get("generation_id") ?? undefined
    const filtered = generationID
      ? state.issues.filter((issue) => issue.generation_id === generationID)
      : state.issues
    const issues = filtered.slice(offset, offset + limit)
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      issues: issues.map(repositoryReviewIssueCollectionSummary),
      total: filtered.length,
      next_cursor:
        offset + issues.length < filtered.length
          ? String(offset + issues.length)
          : "",
      canonical_query:
        url.searchParams.get("query") ?? "ALL ORDER BY updated DESC",
      query_schema: mockCollectionSchemas.reviewIssues,
      ...(generationID ? { generation_id: generationID } : {}),
      capabilities,
    })
  }

  const issueActionMatch = path.match(
    new RegExp(`^${automationRoot}/issues/([^/]+)/(regenerate|publish)$`),
  )
  if (issueActionMatch && method === "POST") {
    const issue = findIssue(decodeURIComponent(issueActionMatch[1]!))
    if (!issue) return json(route, { code: "not_found" }, 404)
    if (issueActionMatch[2] === "regenerate") {
      issue.title = `${issue.title} (regenerated)`
      issue.version += 1
      issue.updated_at = "2026-08-26T12:07:00Z"
    } else {
      issue.state = "posted"
      issue.external_id = "401"
      issue.external_url = "https://github.com/octo/repo/issues/401"
      issue.publishable = false
      issue.deletable = false
      issue.regeneratable = false
      issue.version += 1
    }
    return json(route, issueDetail(issue))
  }

  const issueMatch = path.match(
    new RegExp(`^${automationRoot}/issues/([^/]+)$`),
  )
  if (issueMatch) {
    const issue = findIssue(decodeURIComponent(issueMatch[1]!))
    if (!issue) return json(route, { code: "not_found" }, 404)
    if (method === "DELETE") {
      state.issues = state.issues.filter(
        (candidate) => candidate.id !== issue.id,
      )
      for (const finding of state.findings) {
        if (finding.issue_draft_id === issue.id) {
          finding.issue_draft_id = undefined
          finding.version += 1
        }
      }
      return json(route, {
        outcome: "deleted",
        results: [{ draft_id: issue.id, outcome: "deleted", success: true }],
      })
    }
    if (method === "PATCH") {
      issue.title = String(body?.title ?? issue.title)
      issue.body = String(body?.body ?? issue.body)
      issue.labels = Array.isArray(body?.labels)
        ? body.labels.filter(
            (candidate): candidate is string => typeof candidate === "string",
          )
        : issue.labels
      issue.version += 1
      issue.updated_at = "2026-08-26T12:07:00Z"
    }
    return json(route, issueDetail(issue))
  }

  if (path === `${automationRoot}/commit-options` && method === "GET") {
    return json(route, {
      expected_version: state.automation.version,
      remembered: {
        sha: repositoryReviewCommitSHA,
        short_sha: repositoryReviewCommitSHA.slice(0, 12),
      },
      latest: {
        sha: "f".repeat(40),
        short_sha: "f".repeat(12),
      },
      newer_commit_available: true,
    })
  }

  const actionMatch = path.match(
    new RegExp(`^${automationRoot}/(start|pause|resume|restart)$`),
  )
  if (actionMatch && method === "POST") {
    const action = actionMatch[1]
    state.automation.status =
      action === "pause"
        ? "paused"
        : action === "restart" || action === "start" || action === "resume"
          ? "running"
          : state.automation.status
    state.automation.version += 1
    state.automation.updated_at = "2026-08-26T12:08:00Z"
    return json(route, { automation: state.automation })
  }

  if (path === automationRoot && method === "GET") {
    return json(route, {
      automation: state.automation,
      repository: state.summary,
      capabilities,
    })
  }

  return json(
    route,
    { code: "not_found", message: "Repository review fixture not found." },
    404,
  )
}

for (const routePath of smokeRoutes) {
  test(`${routePath} renders without console errors or horizontal overflow`, async ({
    page,
  }) => {
    const errors = collectPageErrors(page)

    await gotoMockedRoute(page, routePath)
    await expect(page.getByRole("button").first()).toBeVisible()
    await page.waitForTimeout(500)
    await expectNoPersistentLoadingOrLoadError(page)
    await expectNoHorizontalOverflow(page)
    await expectNoSeriousA11yViolations(page)
    expect(errors).toEqual([])
  })
}

test("repository review history deletion requires exact typed confirmation", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}`,
    { repositoryReviewTerminal: true },
  )

  await page.getByRole("button", { name: "Purge review history" }).click()
  const dialog = page.getByRole("alertdialog")
  const confirmation = dialog.getByLabel("Type octo/repo to confirm")
  const purge = dialog.getByRole("button", { name: "Purge review history" })
  await expect(dialog).toContainText(
    "Existing GitHub issues are not changed or deleted",
  )
  await expect(purge).toBeDisabled()
  await confirmation.fill("Octo/repo")
  await expect(purge).toBeDisabled()
  await confirmation.fill("octo/repo")
  await expect(purge).toBeEnabled()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

for (const collection of [
  {
    title: "findings",
    route: `/repository-reviews/${repositoryReviewAutomationID}/findings`,
    defaultQuery: "ALL ORDER BY severity DESC, updated DESC",
  },
  {
    title: "raw findings",
    route: `/repository-reviews/${repositoryReviewAutomationID}/raw-findings`,
    defaultQuery: "ALL ORDER BY created DESC",
  },
  {
    title: "findings processing",
    route: `/repository-reviews/${repositoryReviewAutomationID}/findings-processing`,
    defaultQuery: "ALL ORDER BY updated DESC",
  },
  {
    title: "repository findings",
    route: `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
    defaultQuery: "ALL ORDER BY severity DESC, updated DESC",
  },
  {
    title: "issue previews",
    route: `/repository-reviews/${repositoryReviewAutomationID}/issues?generation_id=rig_smoke_existing`,
    defaultQuery: "ALL ORDER BY updated DESC",
  },
] as const) {
  test(`repository review ${collection.title} use every standard collection view`, async ({
    page,
  }) => {
    const errors = collectPageErrors(page)
    await gotoMockedRoute(page, collection.route)
    await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
    await expect(
      page.locator('[data-slot="collection-results"] [data-item-id]').first(),
    ).toBeVisible()
    await expect
      .poll(() => new URL(page.url()).searchParams.get("q"))
      .toBe(collection.defaultQuery)

    if (collection.title === "findings processing") {
      await page.getByRole("button", { name: "Load more" }).click()
      await expect(
        page.locator('[data-item-id="rrw_smoke_processing_completed_1"]'),
      ).toBeVisible()
    }

    for (const view of ["list", "table", "grid"] as const) {
      await page
        .getByRole("button", {
          name: `${view[0]!.toUpperCase()}${view.slice(1)} view`,
        })
        .click()
      await expect(
        page.getByRole("button", {
          name: `${view[0]!.toUpperCase()}${view.slice(1)} view`,
        }),
      ).toHaveAttribute("aria-pressed", "true")
      expect(new URL(page.url()).searchParams.get("view")).toBe(view)
      await expect(
        page.locator('[data-slot="collection-results"]'),
      ).toBeVisible()
    }

    await expectNoHorizontalOverflow(page)
    await expectNoSeriousA11yViolations(page)
    expect(errors).toEqual([])
  })
}

test("findings processing selects only failures and preserves partial bulk retry failures", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["repositoryReviewRequests"]
  > = []
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing`,
    { repositoryReviewRequests: requests },
  )

  await expect(
    page.getByRole("heading", { name: "Historical consolidation" }),
  ).toBeVisible()
  await expect(
    page.getByRole("button", { name: "Restart incompatible work" }),
  ).toHaveCount(0)
  await page
    .getByRole("button", { name: "Resume historical consolidation" })
    .click()
  await expect
    .poll(() =>
      requests.find((request) =>
        request.path.endsWith("/historical-deduplication/retry"),
      ),
    )
    .toMatchObject({ method: "POST", body: {} })
  const restart = page.getByRole("button", {
    name: "Restart incompatible work",
  })
  await expect(restart).toBeVisible()
  await restart.click()
  const restartDialog = page.getByRole("alertdialog")
  await expect(
    restartDialog.getByText(
      /Completed results in affected historical buckets will be reprocessed/u,
    ),
  ).toBeVisible()
  await expect(
    restartDialog.getByText(
      /Completed work in unrelated buckets will remain preserved/u,
    ),
  ).toBeVisible()
  expect(
    requests.find((request) =>
      request.path.endsWith("/historical-deduplication/restart"),
    ),
  ).toBeUndefined()
  await restartDialog
    .getByRole("button", { name: "Restart incompatible work" })
    .click()
  const historicalToast = page.getByText(
    "Incompatible historical work restarted.",
  )
  await expect(historicalToast).toBeVisible()
  await expect
    .poll(() =>
      requests.find((request) =>
        request.path.endsWith("/historical-deduplication/restart"),
      ),
    )
    .toMatchObject({ method: "POST", body: { confirmed: true } })
  const pending = page.locator(
    `[data-item-id="${repositoryReviewProcessingPendingID}"]`,
  )
  await pending.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toHaveCount(0)

  const failedQuery = "state = failed ORDER BY updated DESC"
  const query = page.getByRole("combobox", { name: "Collection query" })
  await query.fill(failedQuery)
  await page.keyboard.press("Enter")
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe(failedQuery)

  for (const [index, sourceID] of [
    repositoryReviewProcessingFailedID,
    repositoryReviewProcessingOldCampaignID,
  ].entries()) {
    const item = page.locator(`[data-item-id="${sourceID}"]`)
    await expect(item).toBeVisible()
    await item.focus()
    await page.keyboard.press(index === 0 ? "Space" : "Control+Space")
  }
  await expect(page.getByText("2 selected", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Retry selected" }).click()

  const partialToast = page.getByText(
    "1 of 2 selected findings queued. 1 remained selected.",
  )
  await expect(partialToast).toBeVisible()
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await expect(
    page.locator(`[data-item-id="${repositoryReviewProcessingOldCampaignID}"]`),
  ).toBeVisible()
  await expect(
    page
      .locator(`[data-item-id="${repositoryReviewProcessingOldCampaignID}"]`)
      .getByText(
        "Historical sources must be retried through historical consolidation.",
        { exact: true },
      ),
  ).toBeVisible()
  await expect(
    page.locator(`[data-item-id="${repositoryReviewProcessingFailedID}"]`),
  ).toHaveCount(0)
  await expect
    .poll(() =>
      requests.find((request) =>
        request.path.endsWith("/findings-processing/retry"),
      ),
    )
    .toMatchObject({
      method: "POST",
      body: {
        source_ids: [
          repositoryReviewProcessingFailedID,
          repositoryReviewProcessingOldCampaignID,
        ],
      },
    })
  await expect(historicalToast).toBeHidden({ timeout: 10_000 })
  await expect(partialToast).toBeHidden({ timeout: 10_000 })
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(
    errors.filter(
      (message) =>
        !message.includes(
          "Failed to load resource: the server responded with a status of 409",
        ),
    ),
  ).toEqual([])
})

test("findings processing detail exposes safe failure, immutable provenance, links, and retry", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["repositoryReviewRequests"]
  > = []
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing/${repositoryReviewProcessingFailedID}`,
    { repositoryReviewRequests: requests },
  )

  await expect(
    page.getByRole("heading", { name: "Processing failed" }),
  ).toBeVisible()
  await expect(
    page.getByText("Finding grouping reached its retry limit."),
  ).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "Immutable diagnosis" }),
  ).toBeVisible()
  await expect(page.getByRole("heading", { name: "Provenance" })).toBeVisible()
  await expect(page.getByText("openai-primary", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Retry", exact: true }).click()
  await expect(page.getByText("Finding processing queued.")).toBeVisible()
  await expect(page.getByText("Queued", { exact: true })).toBeVisible()
  await expect
    .poll(() =>
      requests.find((request) =>
        request.path.endsWith(
          `/findings-processing/sources/${repositoryReviewProcessingFailedID}/retry`,
        ),
      ),
    )
    .toMatchObject({ method: "POST", body: {} })

  await page.goto(
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing/${repositoryReviewProcessingCompletedID}`,
  )
  await expect(
    page.getByRole("heading", { name: "Linked findings" }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Deduplicated finding" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/findings/${repositoryReviewFindingOneID}`,
    ),
  )
  await page.goBack()
  await page.getByRole("button", { name: "Repository finding" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/rrf_smoke_1`,
    ),
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("findings processing Back restores query, view, selection, and scroll", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const search = new URLSearchParams({
    q: "ALL ORDER BY updated DESC",
    view: "grid",
  })
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing?${search.toString()}`,
  )
  await page.getByRole("button", { name: "Load more" }).click()
  const item = page.locator(
    `[data-item-id="${repositoryReviewProcessingOldCampaignID}"]`,
  )
  await item.scrollIntoViewIfNeeded()
  await item.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  const scrollContainer = page.locator(
    '[data-slot="collection-scroll-container"]',
  )
  const rememberedScroll = await scrollContainer.evaluate((node) =>
    Math.floor(node.scrollTop),
  )
  expect(rememberedScroll).toBeGreaterThan(0)
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(
      `/findings-processing/${repositoryReviewProcessingOldCampaignID}`,
    ),
  )
  await expect(
    page.getByText("rrc_smoke_previous", { exact: true }),
  ).toBeVisible()
  await page.goBack()

  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  const restored = new URL(page.url())
  expect(restored.searchParams.get("q")).toBe("ALL ORDER BY updated DESC")
  expect(restored.searchParams.get("view")).toBe("grid")
  await expect
    .poll(() => scrollContainer.evaluate((node) => Math.floor(node.scrollTop)))
    .toBe(rememberedScroll)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("finding health polling runs only while review or processing work is active", async ({
  page,
}) => {
  await page.clock.install({ time: new Date("2026-08-26T12:06:00Z") })
  let activeReads = 0
  page.on("request", (request) => {
    if (new URL(request.url()).pathname.endsWith("/finding-health")) {
      activeReads += 1
    }
  })
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing`,
  )
  await expect.poll(() => activeReads).toBeGreaterThanOrEqual(1)
  const initialActiveReads = activeReads
  await page.clock.fastForward(2_100)
  await expect.poll(() => activeReads).toBeGreaterThan(initialActiveReads)

  const terminalPage = await page.context().newPage()
  await terminalPage.clock.install({ time: new Date("2026-08-26T12:06:00Z") })
  let terminalReads = 0
  terminalPage.on("request", (request) => {
    if (new URL(request.url()).pathname.endsWith("/finding-health")) {
      terminalReads += 1
    }
  })
  await gotoMockedRoute(
    terminalPage,
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing`,
    { repositoryReviewTerminal: true },
  )
  await expect.poll(() => terminalReads).toBeGreaterThanOrEqual(1)
  const initialTerminalReads = terminalReads
  await terminalPage.clock.fastForward(6_100)
  expect(terminalReads).toBe(initialTerminalReads)
  await terminalPage.close()
})

test("active collection keeps polling health when its first snapshot has no work", async ({
  page,
}) => {
  await page.clock.install({ time: new Date("2026-08-26T12:06:00Z") })
  let healthReads = 0
  page.on("request", (request) => {
    if (new URL(request.url()).pathname.endsWith("/finding-health")) {
      healthReads += 1
    }
  })
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
    { repositoryReviewStagedHealth: true },
  )
  await expect.poll(() => healthReads).toBeGreaterThanOrEqual(1)
  await page.clock.fastForward(2_100)
  await expect(
    page.getByRole("heading", { name: "Repository coverage is incomplete" }),
  ).toBeVisible()
})

test("review health puts a failed alert first and uses exact health cards", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}`,
    { repositoryReviewFailed: true },
  )

  const failure = page.getByRole("alert").filter({
    hasText: "The reviewer provider stopped after repeated safe failures.",
  })
  await expect(failure).toBeVisible()
  const runFindings = page.getByRole("button", { name: /^Run findings/u })
  const repositoryFindings = page.getByRole("button", {
    name: /^Repository findings/u,
  })
  const processing = page.getByRole("button", {
    name: /^Findings processing/u,
  })
  await expect(runFindings).toContainText("9 run findings")
  await expect(repositoryFindings).toContainText(
    `${repositoryFindingsFixture.length} repository findings`,
  )
  await expect(processing).toContainText(
    `${repositoryReviewProcessingSourcesFixture.length} processing records`,
  )
  const [failureBox, runFindingsBox] = await Promise.all([
    failure.boundingBox(),
    runFindings.boundingBox(),
  ])
  expect(failureBox).not.toBeNull()
  expect(runFindingsBox).not.toBeNull()
  expect(failureBox!.y).toBeLessThan(runFindingsBox!.y)

  const repositoryMetric = page
    .getByText("Repository findings", { exact: true })
    .filter({ visible: true })
    .last()
    .locator("xpath=..")
  await expect(repositoryMetric).toContainText(
    String(repositoryFindingsFixture.length),
  )
  const unrepresentedMetric = page
    .getByText("Unrepresented run findings", { exact: true })
    .locator("xpath=..")
  await expect(unrepresentedMetric).toContainText("3")
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("repository findings show one scoped incomplete-coverage notice", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
  )

  const notice = page.getByRole("heading", {
    name: "Repository coverage is incomplete",
  })
  await expect(notice).toHaveCount(1)
  const section = notice.locator("xpath=ancestor::section")
  await expect(section).toContainText(
    "3 run findings are still pending, processing, or failed",
  )
  await section
    .getByRole("button", { name: "View unrepresented run findings" })
    .click()
  await expect(page).toHaveURL(
    new RegExp(`/repository-reviews/${repositoryReviewAutomationID}/findings`),
  )
  expect(new URL(page.url()).searchParams.get("q")).toBe(
    "run_status IN (pending, processing, failed) ORDER BY updated DESC",
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("findings processing Updated header stays inside the desktop table viewport", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "desktop",
    "Canonical desktop geometry is asserted at the 1280px desktop viewport.",
  )
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings-processing?view=table`,
  )
  const results = page.locator('[data-slot="collection-results"]')
  const table = results.locator("table")
  const updatedHeader = table.getByRole("columnheader", {
    name: "Updated",
    exact: true,
  })
  await expect(updatedHeader).toBeVisible()
  const [resultsBox, tableBox, headerBox] = await Promise.all([
    results.boundingBox(),
    table.boundingBox(),
    updatedHeader.boundingBox(),
  ])
  expect(resultsBox).not.toBeNull()
  expect(tableBox).not.toBeNull()
  expect(headerBox).not.toBeNull()
  expect(headerBox!.x).toBeGreaterThanOrEqual(tableBox!.x - 1)
  expect(headerBox!.x + headerBox!.width).toBeLessThanOrEqual(
    tableBox!.x + tableBox!.width + 1,
  )
  expect(headerBox!.x + headerBox!.width).toBeLessThanOrEqual(
    resultsBox!.x + resultsBox!.width + 1,
  )
  expect(headerBox!.x + headerBox!.width).toBeLessThanOrEqual(
    (page.viewportSize()?.width ?? 0) + 1,
  )
  const headerWidths = await updatedHeader.evaluate((element) => ({
    client: element.clientWidth,
    scroll: element.scrollWidth,
  }))
  expect(headerWidths.scroll).toBeLessThanOrEqual(headerWidths.client + 1)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("standard collection query completes, synchronizes, and recovers from errors", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const collectionPath = `/api/repository-reviews/automations/${repositoryReviewAutomationID}/findings`
  const collectionQueries: string[] = []
  page.on("request", (request) => {
    const url = new URL(request.url())
    if (request.method() === "GET" && url.pathname === collectionPath) {
      const query = url.searchParams.get("query")
      if (query) collectionQueries.push(query)
    }
  })

  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings`,
  )
  const editor = page.locator('[data-slot="collection-query-input"]')
  const input = editor.getByRole("combobox", {
    name: "Collection query",
  })

  await input.fill("sev")
  await expect(input).toHaveAttribute("aria-autocomplete", "list")
  await expect(input).toHaveAttribute("aria-expanded", "true")
  const listbox = page.getByRole("listbox", {
    name: "Collection query suggestions",
  })
  await expect(listbox).toBeVisible()
  const listboxID = await listbox.getAttribute("id")
  expect(listboxID).toBeTruthy()
  await expect(input).toHaveAttribute("aria-controls", listboxID!)

  const describedBy = (await input.getAttribute("aria-describedby"))
    ?.split(/\s+/)
    .filter(Boolean)
  expect(describedBy).toHaveLength(2)
  for (const descriptionID of describedBy ?? []) {
    await expect(page.locator(`[id="${descriptionID}"]`)).toBeVisible()
  }

  await input.press("ArrowDown")
  const severityOption = listbox.getByRole("option", {
    name: "severity, enum field",
    exact: true,
  })
  await expect(severityOption).toHaveAttribute("aria-selected", "true")
  const severityOptionID = await severityOption.getAttribute("id")
  expect(severityOptionID).toBeTruthy()
  await expect(input).toHaveAttribute(
    "aria-activedescendant",
    severityOptionID!,
  )
  await expectNoHorizontalOverflow(page)
  await expectElementFitsViewport(
    page,
    `[id="${listboxID}"]`,
    "Collection query suggestions",
  )

  await input.press("Enter")
  await expect(input).toHaveValue("severity ")
  await input.pressSequentially("= hi")
  await expect(
    page.getByRole("option", {
      name: "high, enum value",
      exact: true,
    }),
  ).toBeVisible()
  await input.press("ArrowDown")
  await input.press("Enter")
  await expect(input).toHaveValue("severity = high ")

  const completedQuery = "severity = high"
  await input.press("Enter")
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe(completedQuery)
  await expect.poll(() => collectionQueries.includes(completedQuery)).toBe(true)

  const rejectedQuery = 'severity = "🔥"'
  await page.route(`**${collectionPath}?*`, async (route) => {
    const url = new URL(route.request().url())
    if (url.searchParams.get("query") === rejectedQuery) {
      return json(
        route,
        {
          code: "invalid_query",
          message: "Expected a configured severity value.",
          position: 12,
        },
        400,
      )
    }
    return route.fallback()
  })

  await input.fill(rejectedQuery)
  await input.press("Enter")
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe(rejectedQuery)
  const queryAlert = editor.getByRole("alert")
  await expect(queryAlert).toHaveText(
    "Character 13: Expected a configured severity value.",
  )
  await expect(input).toHaveAttribute("aria-invalid", "true")
  const queryAlertID = await queryAlert.getAttribute("id")
  expect(queryAlertID).toBeTruthy()
  await expect(input).toHaveAttribute("aria-errormessage", queryAlertID!)
  await expect
    .poll(() =>
      input.evaluate((element: HTMLInputElement) => ({
        start: element.selectionStart,
        end: element.selectionEnd,
      })),
    )
    .toEqual({ start: 12, end: 14 })

  const recoveredQuery = "severity = low"
  await input.fill(recoveredQuery)
  await expect(queryAlert).toHaveCount(0)
  await expect(input).toHaveAttribute("aria-invalid", "false")
  await input.press("Enter")
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe(recoveredQuery)
  await expect.poll(() => collectionQueries.includes(recoveredQuery)).toBe(true)
  await expect(
    page.locator('[data-slot="collection-results"] [data-item-id]').first(),
  ).toBeVisible()

  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(
    errors.filter(
      (message) =>
        !message.includes(
          "Failed to load resource: the server responded with a status of 400",
        ),
    ),
  ).toEqual([])
})

test("repository findings Updated header stays inside the canonical desktop table viewport", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "desktop",
    "Canonical desktop geometry is asserted at the 1280px desktop viewport.",
  )
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings?view=table`,
  )
  await expect(
    page.getByRole("button", { name: "Table view" }),
  ).toHaveAttribute("aria-pressed", "true")

  const results = page.locator('[data-slot="collection-results"]')
  const table = results.locator("table")
  const updatedHeader = table.getByRole("columnheader", {
    name: "Updated",
    exact: true,
  })
  await expect(updatedHeader).toBeVisible()
  const [resultsBox, tableBox, headerBox] = await Promise.all([
    results.boundingBox(),
    table.boundingBox(),
    updatedHeader.boundingBox(),
  ])
  expect(resultsBox).not.toBeNull()
  expect(tableBox).not.toBeNull()
  expect(headerBox).not.toBeNull()
  expect(headerBox!.x).toBeGreaterThanOrEqual(tableBox!.x - 1)
  expect(headerBox!.x + headerBox!.width).toBeLessThanOrEqual(
    tableBox!.x + tableBox!.width + 1,
  )
  expect(headerBox!.x + headerBox!.width).toBeLessThanOrEqual(
    resultsBox!.x + resultsBox!.width + 1,
  )
  expect(headerBox!.x + headerBox!.width).toBeLessThanOrEqual(
    (page.viewportSize()?.width ?? 0) + 1,
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("repository finding identity text uses the available width in every view", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings?view=list`,
  )

  const title = "Request retries can repeat a committed ledger update"
  const metadata =
    "pkg/repoaudit/transaction_writer.go · TransactionWriter.Commit · medium · Open · Updated Aug 26, 2026"

  for (const view of ["list", "table", "grid"] as const) {
    const viewButton = page.getByRole("button", {
      name: `${view[0]!.toUpperCase()}${view.slice(1)} view`,
    })
    await viewButton.click()
    await expect(viewButton).toHaveAttribute("aria-pressed", "true")

    const item = page
      .locator(`[data-item-id="${repositoryReviewNormalAggregateID}"]`)
      .filter({ visible: true })
    await expect(item).toHaveCount(1)

    for (const text of [title, metadata]) {
      const content = item.getByText(text, { exact: true })
      await expect(content).toBeVisible()
      const bounds = await content.evaluate((element) => {
        let identity = element.parentElement
        while (identity) {
          const hasTitle = Array.from(identity.children).some((child) =>
            child.classList.contains("font-medium"),
          )
          if (identity.classList.contains("min-w-0") && hasTitle) break
          identity = identity.parentElement
        }
        if (!identity) throw new Error("Identity block not found")
        const contentBox = element.getBoundingClientRect()
        const identityBox = identity.getBoundingClientRect()
        return {
          contentLeft: contentBox.left,
          contentRight: contentBox.right,
          identityLeft: identityBox.left,
          identityRight: identityBox.right,
        }
      })
      expect(
        bounds.contentLeft,
        `${view} identity content should start at the identity block edge`,
      ).toBeLessThanOrEqual(bounds.identityLeft + 1)
      expect(
        bounds.contentRight,
        `${view} identity content should use the identity block width`,
      ).toBeGreaterThanOrEqual(bounds.identityRight - 1)
      expect(
        bounds.contentRight,
        `${view} identity content should stay inside the identity block`,
      ).toBeLessThanOrEqual(bounds.identityRight + 1)
    }
  }

  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("combined repository finding attention badges stay inside the mobile result", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "mobile",
    "Combined attention wrapping is asserted at the 390px mobile viewport.",
  )
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
  )

  const row = page.locator(
    `[data-item-id="${repositoryReviewCombinedAttentionID}"]`,
  )
  await row.scrollIntoViewIfNeeded()
  await expect(row).toBeVisible()
  for (const label of [
    "Duplicate review",
    "Issue conflict",
    "Fix check failed",
  ]) {
    await expect(row.getByText(label, { exact: true })).toBeVisible()
  }
  const badges = row.locator('[data-slot="badge"]')
  await expect(badges).toHaveCount(3)
  const rowBox = await row.boundingBox()
  expect(rowBox).not.toBeNull()
  for (let index = 0; index < (await badges.count()); index += 1) {
    const badgeBox = await badges.nth(index).boundingBox()
    expect(badgeBox).not.toBeNull()
    expect(badgeBox!.x).toBeGreaterThanOrEqual(rowBox!.x - 1)
    expect(badgeBox!.x + badgeBox!.width).toBeLessThanOrEqual(
      rowBox!.x + rowBox!.width + 1,
    )
    expect(badgeBox!.x + badgeBox!.width).toBeLessThanOrEqual(
      (page.viewportSize()?.width ?? 0) + 1,
    )
    expect(badgeBox!.y + badgeBox!.height).toBeLessThanOrEqual(
      rowBox!.y + rowBox!.height + 1,
    )
  }
  const rowWidths = await row.evaluate((element) => ({
    client: element.clientWidth,
    scroll: element.scrollWidth,
  }))
  expect(rowWidths.scroll).toBeLessThanOrEqual(rowWidths.client + 1)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("repository finding attention states stay actionable without normal-row noise", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["repositoryReviewRequests"]
  > = []
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
    { repositoryReviewRequests: requests },
  )

  const normalRow = page.locator(
    `[data-item-id="${repositoryReviewNormalAggregateID}"]`,
  )
  await expect(normalRow).toBeVisible()
  await expect(
    normalRow.getByText("3 occurrences across 2 commits", { exact: true }),
  ).toBeVisible()
  await expect(
    normalRow.getByText("Duplicate review", { exact: true }),
  ).toHaveCount(0)
  await expect(
    normalRow.getByText("Issue conflict", { exact: true }),
  ).toHaveCount(0)
  await expect(
    normalRow.getByText("Fix check failed", { exact: true }),
  ).toHaveCount(0)

  const provisionalRow = page.locator(
    `[data-item-id="${repositoryReviewProvisionalID}"]`,
  )
  const conflictRow = page.locator(
    `[data-item-id="${repositoryReviewConflictID}"]`,
  )
  const failedCheckRow = page.locator(
    `[data-item-id="${repositoryReviewFailedCheckID}"]`,
  )
  await expect(
    provisionalRow.getByText("Duplicate review", { exact: true }),
  ).toBeVisible()
  await expect(
    conflictRow.getByText("Issue conflict", { exact: true }),
  ).toBeVisible()
  await expect(
    failedCheckRow.getByText("Fix check failed", { exact: true }),
  ).toBeVisible()

  await provisionalRow.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/${repositoryReviewProvisionalID}`,
    ),
  )
  await expect(
    page.getByText("Needs duplicate review", { exact: true }),
  ).toBeVisible()
  const candidateCard = page
    .getByRole("heading", {
      name: "Delayed cache refresh overwrites a newer generation",
    })
    .locator("xpath=ancestor::article")
  await expect(candidateCard).toBeVisible()
  await expect(
    candidateCard.getByText(/pkg\/cache\/generation_store\.go/u),
  ).toBeVisible()
  await expect(candidateCard.getByText("4", { exact: true })).toBeVisible()
  await expect(candidateCard.getByText("3", { exact: true })).toBeVisible()
  await expect(
    candidateCard.getByText(
      /Both diagnoses describe an older cache generation/u,
    ),
  ).toBeVisible()

  await candidateCard.getByRole("button", { name: "View candidate" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/${repositoryReviewCandidateID}`,
    ),
  )
  await page.goBack()
  await expect(
    page.getByRole("button", { name: "Merge with candidate" }),
  ).toBeVisible()

  await page.getByRole("button", { name: "Merge with candidate" }).click()
  const confirmation = page.getByRole("alertdialog", {
    name: "Merge this finding with the candidate?",
  })
  await expect(confirmation).toBeVisible()
  await expectElementFitsViewport(
    page,
    '[role="alertdialog"]',
    "duplicate merge confirmation",
  )
  expect(requests.some((request) => request.path.endsWith("/duplicates"))).toBe(
    false,
  )
  await confirmation.getByRole("button", { name: "Cancel" }).click()
  await expect(confirmation).toBeHidden()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)

  await page.getByRole("button", { name: "Keep separate" }).click()
  await expect
    .poll(() =>
      requests.find((request) => request.path.endsWith("/duplicates")),
    )
    .toMatchObject({
      body: {
        candidate_id: repositoryReviewCandidateID,
        decision: "distinct",
        expected_provisional_version: 2,
      },
    })
  await expectNoHorizontalOverflow(page)
  expect(errors).toEqual([])
})

test("repository finding merge confirmation is keyboard operable at desktop and mobile widths", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["repositoryReviewRequests"]
  > = []
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/${repositoryReviewProvisionalID}`,
    { repositoryReviewRequests: requests },
  )

  const mergeTrigger = page.getByRole("button", {
    name: "Merge with candidate",
  })
  await mergeTrigger.focus()
  await expect(mergeTrigger).toBeFocused()
  await page.keyboard.press("Space")
  const confirmation = page.getByRole("alertdialog", {
    name: "Merge this finding with the candidate?",
  })
  await expect(confirmation).toBeVisible()
  expect(requests.some((request) => request.path.endsWith("/duplicates"))).toBe(
    false,
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  const cancel = confirmation.getByRole("button", { name: "Cancel" })
  const confirm = confirmation.getByRole("button", {
    name: "Merge with candidate",
  })
  await expect(cancel).toBeFocused()
  await page.keyboard.press("Tab")
  await expect(confirm).toBeFocused()
  await page.keyboard.press("Enter")
  await expect
    .poll(() =>
      requests.find((request) => request.path.endsWith("/duplicates")),
    )
    .toMatchObject({
      body: {
        candidate_id: repositoryReviewCandidateID,
        decision: "merge",
        expected_provisional_version: 2,
        expected_candidate_version: 7,
      },
    })
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/${repositoryReviewCandidateID}`,
    ),
  )
  await expectNoHorizontalOverflow(page)
  expect(errors).toEqual([])
})

test("repository review raw findings navigate through canonical source detail", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/repository-reviews")

  const reviewRow = page.locator(
    `[data-item-id="${repositoryReviewAutomationID}"]`,
  )
  await reviewRow.getByRole("button", { name: "Raw findings: 5" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/raw-findings`,
    ),
  )
  const rawRow = page.locator('[data-item-id="rrw_smoke_1"]')
  await expect(rawRow).toBeVisible()
  await rawRow.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/raw-findings/rrw_smoke_1`,
    ),
  )
  await expect(
    page.getByRole("heading", { name: "Raw diagnosis" }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Open finding" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/findings/${repositoryReviewFindingOneID}`,
    ),
  )
  await expect(
    page.getByRole("heading", { name: "Raw run findings" }),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("legacy repository review finding bookmarks redirect to canonical raw detail", async ({
  page,
}) => {
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings/rfn_smoke_legacy`,
  )
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/raw-findings/rrw_smoke_1`,
    ),
  )
  await expect(
    page.getByRole("heading", { name: "Raw diagnosis" }),
  ).toBeVisible()
})

test("repository review routing preserves run context through repository finding generation and subset publication", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["repositoryReviewRequests"]
  > = []
  await gotoMockedRoute(
    page,
    "/repository-reviews?q=ORDER+BY+repository+ASC&view=list",
    { repositoryReviewRequests: requests },
  )

  const reviewRow = page.locator(
    `[data-item-id="${repositoryReviewAutomationID}"]`,
  )
  await expect(reviewRow).toBeVisible()
  await reviewRow.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(`/repository-reviews/${repositoryReviewAutomationID}`),
  )
  await expect(
    page.getByRole("heading", { name: "octo/repo", exact: true }),
  ).toBeVisible()

  await page.getByRole("button", { name: "Stop safely" }).click()
  await expect(page.getByText("paused", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: /^Run findings/ }).click()
  await expect(page).toHaveURL(
    new RegExp(`/repository-reviews/${repositoryReviewAutomationID}/findings`),
  )

  const findingRow = page.locator(
    `[data-item-id="${repositoryReviewFindingOneID}"]`,
  )
  await expect(findingRow).toBeVisible()
  await expect(
    findingRow
      .getByText("Created repository finding", { exact: true })
      .or(findingRow.getByText("New", { exact: true }))
      .filter({ visible: true }),
  ).toBeVisible()
  await findingRow.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await expect(
    page.getByRole("button", { name: "Draft issue previews" }),
  ).toHaveCount(0)
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/findings/${repositoryReviewFindingOneID}`,
    ),
  )
  await expect(
    page.getByRole("heading", { name: "Finding status" }),
  ).toBeVisible()
  await expect(
    page.getByText("A repository finding was created from this finding."),
  ).toBeVisible()
  await expect(page.getByText("Traced both writers through")).toBeVisible()
  await page.getByRole("button", { name: "Open repository finding" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/rrf_smoke_1`,
    ),
  )
  await expect(
    page.getByRole("heading", { name: "Occurrence history" }),
  ).toBeVisible()

  await page.goBack()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/findings/${repositoryReviewFindingOneID}`,
    ),
  )
  await page.goBack()
  await expect(page).toHaveURL(
    new RegExp(`/repository-reviews/${repositoryReviewAutomationID}/findings`),
  )
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  const restoredRunFindingsURL = new URL(page.url())
  expect(restoredRunFindingsURL.searchParams.get("q")).toBe(
    "ALL ORDER BY severity DESC, updated DESC",
  )
  expect(restoredRunFindingsURL.searchParams.get("scope")).toBeNull()

  await page.getByRole("button", { name: "View repository findings" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
    ),
  )
  const repositoryFindingRow = page.locator('[data-item-id="rrf_smoke_1"]')
  await expect(repositoryFindingRow).toBeVisible()
  await repositoryFindingRow.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Draft issue previews" }).click()
  const generationDialog = page.getByRole("dialog", {
    name: "Draft 1 issue",
  })
  await expect(generationDialog).toBeVisible()
  await generationDialog.getByRole("button", { name: "Draft previews" }).click()

  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/issues\\?.*generation_id=rig_`,
    ),
  )
  await expect(page.getByText(/Showing generation/)).toBeVisible()
  await page.getByRole("button", { name: "Show all previews" }).click()
  await expect(page.getByText(/Showing generation/)).toHaveCount(0)

  const publishRow = page.locator(
    `[data-item-id="${repositoryReviewIssueOneID}"]`,
  )
  await publishRow.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Post selected" }).click()
  await expect(page.getByText("Posting outcomes")).toBeVisible()
  await expect(
    page.getByText(`${repositoryReviewIssueOneID}: posted`),
  ).toBeVisible()

  await page.getByRole("button", { name: "Review details" }).click()
  await expect(page).toHaveURL(
    new RegExp(`/repository-reviews/${repositoryReviewAutomationID}(?:\\?|$)`),
  )
  const restoredURL = new URL(page.url())
  expect(restoredURL.searchParams.get("q")).toBe("ORDER BY repository ASC")
  expect(restoredURL.searchParams.get("scope")).toBeNull()

  const generationRequest = requests.find((request) =>
    request.path.endsWith("/issues/generations"),
  )
  expect(generationRequest?.body).toMatchObject({
    finding_ids: [repositoryReviewFindingOneID],
    instructions_mode: "default",
  })
  const publicationRequest = requests.find((request) =>
    request.path.endsWith("/issues/publish"),
  )
  expect(publicationRequest?.body).toMatchObject({
    confirmed: true,
    issues: [{ id: repositoryReviewIssueOneID, expected_version: 3 }],
  })
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("repository review candidate linking requires explicit confirmation", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["repositoryReviewRequests"]
  > = []
  await gotoMockedRoute(
    page,
    `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings/rrf_smoke_3/link-issue`,
    { repositoryReviewRequests: requests },
  )

  await page
    .getByRole("button", { name: "Ask AI to find existing issues" })
    .click()
  await expect(
    page.getByRole("heading", {
      name: "#184 Worker retries cancellation immediately",
    }),
  ).toBeVisible()
  await page
    .getByRole("heading", {
      name: "#184 Worker retries cancellation immediately",
    })
    .locator("xpath=ancestor::article")
    .getByRole("button", { name: "Select" })
    .click()

  const confirmation = page.getByRole("alertdialog", {
    name: "Link this existing issue?",
  })
  await expect(confirmation).toBeVisible()
  expect(requests.some((request) => request.path.endsWith("/issue-link"))).toBe(
    false,
  )
  await confirmation.getByRole("button", { name: "Cancel" }).click()
  await expect(confirmation).toBeHidden()

  await page
    .getByRole("heading", {
      name: "#184 Worker retries cancellation immediately",
    })
    .locator("xpath=ancestor::article")
    .getByRole("button", { name: "Select" })
    .click()
  await confirmation.getByRole("button", { name: "Link issue" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/issues/rrid_linked_184`,
    ),
  )
  await expect(
    page.getByRole("link", { name: "Open GitHub issue" }),
  ).toHaveAttribute("href", "https://github.com/octo/repo/issues/184")
  const linkRequest = requests.find((request) =>
    request.path.endsWith("/issue-link"),
  )
  expect(linkRequest?.body).toEqual({
    issue_url: "https://github.com/octo/repo/issues/184",
    expected_version: 1,
    confirmed: true,
  })
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("legacy repository review results redirect to the standard collection", async ({
  page,
}) => {
  await gotoMockedRoute(
    page,
    "/repository-reviews/results?q=status+%3D+running&view=grid",
  )
  await expect(page).toHaveURL(
    /\/repository-reviews\?q=status(?:\+|%20)%3D(?:\+|%20)running&view=grid$/,
  )
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
})

test("legacy repository finding scope and offset normalize to cursor collections", async ({
  page,
}) => {
  const collectionRequests: URL[] = []
  const findingsPath = `/api/repository-reviews/automations/${repositoryReviewAutomationID}/findings`
  page.on("request", (request) => {
    const url = new URL(request.url())
    if (
      url.pathname === findingsPath ||
      url.pathname ===
        `/api/repository-reviews/automations/${repositoryReviewAutomationID}/repository-findings`
    ) {
      collectionRequests.push(url)
    }
  })

  const legacyQuery = "ALL ORDER BY repository ASC"
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/findings?${new URLSearchParams(
      {
        q: legacyQuery,
        scope: "current",
        offset: "75",
        view: "grid",
      },
    ).toString()}`,
  )
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect
    .poll(() => {
      const current = new URL(page.url())
      return {
        path: current.pathname,
        q: current.searchParams.get("q"),
        view: current.searchParams.get("view"),
        scope: current.searchParams.get("scope"),
        offset: current.searchParams.get("offset"),
      }
    })
    .toEqual({
      path: `/repository-reviews/${repositoryReviewAutomationID}/findings`,
      q: legacyQuery,
      view: "grid",
      scope: null,
      offset: null,
    })
  const currentRequest = collectionRequests.findLast(
    (request) => request.pathname === findingsPath,
  )
  expect(currentRequest?.pathname).toBe(findingsPath)
  expect(currentRequest?.searchParams.get("query")).toBe(legacyQuery)
  expect(currentRequest?.searchParams.has("cursor")).toBe(false)
  expect(currentRequest?.searchParams.has("offset")).toBe(false)
  expect(currentRequest?.searchParams.has("scope")).toBe(false)

  await page.goto(
    `/repository-reviews/${repositoryReviewAutomationID}/findings?${new URLSearchParams(
      {
        q: legacyQuery,
        scope: "all",
        offset: "75",
        view: "table",
      },
    ).toString()}`,
  )
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect
    .poll(() => {
      const current = new URL(page.url())
      return {
        path: current.pathname,
        q: current.searchParams.get("q"),
        view: current.searchParams.get("view"),
        scope: current.searchParams.get("scope"),
        offset: current.searchParams.get("offset"),
      }
    })
    .toEqual({
      path: `/repository-reviews/repositories/${repositoryReviewAutomationID}/findings`,
      q: legacyQuery,
      view: "table",
      scope: null,
      offset: null,
    })
  const allRequest = collectionRequests.findLast((request) =>
    request.pathname.endsWith("/repository-findings"),
  )
  expect(allRequest?.searchParams.get("query")).toBe(legacyQuery)
  expect(allRequest?.searchParams.has("cursor")).toBe(false)
  expect(allRequest?.searchParams.has("offset")).toBe(false)
  expect(allRequest?.searchParams.has("scope")).toBe(false)
})

test("issue preview detail opens its dedicated editor and preserves collection state", async ({
  page,
}) => {
  const search = new URLSearchParams({
    q: "ALL ORDER BY updated DESC",
    view: "grid",
    generation_id: "rig_smoke_existing",
  })
  await gotoMockedRoute(
    page,
    `/repository-reviews/${repositoryReviewAutomationID}/issues/${repositoryReviewIssueOneID}?${search.toString()}`,
  )

  await page.getByRole("button", { name: "Edit", exact: true }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/issues/${repositoryReviewIssueOneID}/edit`,
    ),
  )
  await expect(
    page.getByRole("heading", { name: "Edit issue preview" }),
  ).toBeVisible()
  await expect(page.getByRole("button", { name: "Save preview" })).toBeVisible()
  expect(new URL(page.url()).searchParams.get("q")).toBe(
    "ALL ORDER BY updated DESC",
  )
  expect(new URL(page.url()).searchParams.get("view")).toBe("grid")
  expect(new URL(page.url()).searchParams.get("generation_id")).toBe(
    "rig_smoke_existing",
  )

  await page.getByRole("button", { name: "Cancel", exact: true }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/repository-reviews/${repositoryReviewAutomationID}/issues/${repositoryReviewIssueOneID}(?:\\?|$)`,
    ),
  )
})

test("sidebar navigation survives collection query canonicalization", async ({
  page,
}, testInfo) => {
  const errors = collectPageErrors(page)
  let routerRequests = 0
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === "/api/model-routers") {
      routerRequests += 1
    }
  })

  await gotoMockedRoute(page, "/models/aliases?q=ALL+ORDER+BY+name+ASC")
  await expect(page).toHaveURL(/\/models\/aliases\?q=ALL\+ORDER\+BY\+name\+ASC/)
  if (testInfo.project.name === "mobile") {
    await page.getByRole("button", { name: "Toggle Sidebar" }).click()
    const sidebar = page.getByRole("dialog", { name: "Sidebar" })
    await sidebar.getByRole("button", { name: "Services", exact: true }).click()
    await sidebar.getByRole("link", { name: "Model routers" }).click()
  } else {
    await page.getByRole("button", { name: "Services", exact: true }).click()
    await page.getByRole("link", { name: "Model routers" }).click()
  }

  await expect(page).toHaveURL(/\/models\/routers(?:\?|$)/)
  await expect(
    page.getByRole("heading", { name: "Model routers", exact: true }),
  ).toBeVisible()
  expect(routerRequests).toBeGreaterThan(0)
  expect(errors).toEqual([])
})

test("collection rows use gesture selection and contextual actions", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/models/aliases")
  const results = page.locator('[data-slot="collection-results"]')
  const rows = results.locator("[data-item-id]")
  await expect(rows).toHaveCount(2)
  await expect(results.getByRole("checkbox")).toHaveCount(0)
  await expect(
    results.getByRole("button", { name: /Actions for/ }),
  ).toHaveCount(0)

  await rows.nth(0).click()
  await rows.nth(1).click({ modifiers: ["Shift"] })
  await expect(page.getByText("2 selected", { exact: true })).toBeVisible()

  await rows.nth(0).click({ button: "right" })
  await expect(page.getByRole("menuitem", { name: "Open" })).toBeVisible()
  await expect(page.getByRole("menuitem", { name: "Edit alias" })).toBeVisible()
  await page.keyboard.press("Escape")

  await rows.nth(0).dblclick()
  await expect(page).toHaveURL(/\/models\/aliases\/code(?:\?|$)/)
  expect(errors).toEqual([])
})

test("removed collection URLs use the normal not-found boundary", async ({
  page,
}) => {
  for (const routePath of ["/models", "/agent/mcp"] as const) {
    await gotoMockedRoute(page, routePath)
    await expect(page.locator('[data-slot="collection-shell"]')).toHaveCount(0)
    await expect(page).toHaveURL(
      new RegExp(`${routePath.replaceAll("/", "\\/")}$`),
    )
  }
})

test("legacy selected-item searches do not compatibility-render details", async ({
  page,
}) => {
  await gotoMockedRoute(page, "/agent/agents?agent=reviewer")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect(page).not.toHaveURL(/agent=/)
  await gotoMockedRoute(page, `/model-evaluations?probe=rme_${"1".repeat(32)}`)
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect(page).not.toHaveURL(/probe=/)
})

test("development intake stays exclusive and preserves an issue event prefill", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const issueURL = "https://github.com/octo/repo/issues/42"
  await gotoMockedRoute(
    page,
    `/development/new?${new URLSearchParams({ issue: issueURL }).toString()}`,
  )

  await expect(
    page.getByRole("button", { name: /^Implement feature/ }),
  ).toHaveAttribute("aria-pressed", "true")
  await expect(page.getByLabel("GitHub issue URL")).toHaveValue(issueURL)
  await expect(page.getByLabel("GitHub pull request URL")).toHaveCount(0)
  await expect(page.getByLabel("Feature brief")).toHaveCount(0)

  await page.getByRole("button", { name: /^Pick up PR/ }).click()
  await expect(page.getByLabel("GitHub pull request URL")).toBeVisible()
  await expect(page.getByLabel("GitHub issue URL")).toHaveCount(0)
  await expect(page.getByLabel("Feature brief")).toHaveCount(0)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("development portfolio opens a durable workspace with inspection tabs", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    "/development?filter=waiting&q=repository%20~%20octo&view=grid",
  )

  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("repository ~ octo")
  expect(new URL(page.url()).searchParams.get("filter")).toBeNull()
  expect(new URL(page.url()).searchParams.get("view")).toBe("grid")

  const workspace = page.locator(
    `[data-item-id="${developmentWorkspaceID}"]:visible`,
  )
  await expect(workspace).toBeVisible()
  await workspace.focus()
  await page.keyboard.press("Enter")
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe(`/development/${developmentWorkspaceID}`)
  let routed = new URL(page.url())
  expect(routed.searchParams.get("q")).toBe("repository ~ octo")
  expect(routed.searchParams.get("view")).toBe("grid")
  expect(routed.searchParams.get("tab")).toBe("overview")
  await expect(page.getByRole("button", { name: "Overview" })).toHaveAttribute(
    "aria-current",
    "page",
  )
  await expect(page.getByRole("button", { name: "Changes" })).toBeVisible()
  await expect(page.getByRole("button", { name: "Files" })).toBeVisible()
  await expect(page.getByRole("button", { name: "Activity" })).toBeVisible()
  await expect(page.getByText("Tests", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "All development workspaces" }).click()
  await expect.poll(() => new URL(page.url()).pathname).toBe("/development")
  routed = new URL(page.url())
  expect(routed.searchParams.get("q")).toBe("repository ~ octo")
  expect(routed.searchParams.get("view")).toBe("grid")
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("notification inbox opens the exact development action target", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/notifications")

  const item = page.getByRole("button", {
    name: "Open Publication approval needed",
  })
  await expect(item).toBeVisible()
  await item.click()
  await expect(
    page.getByRole("button", { name: "Open required action" }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Open required action" }).click()
  await expect(page).toHaveURL(
    new RegExp(
      `/development/${developmentWorkspaceID}\\?tab=overview&panel=publication&entity=gate-1&q=ORDER\\+BY\\+updated\\+DESC$`,
    ),
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("review profile guard autocompletes fields and keeps reference behind help", async ({
  page,
}) => {
  await gotoMockedRoute(page, "/repository-reviews/profiles", {
    repositoryReviewAutomationOptions: {
      models: [
        { alias: "safe-a", available: true },
        { alias: "safe-b", available: true },
        {
          alias: "blocked",
          available: false,
          blocked_reason: "Reviewer route is blocked by policy.",
        },
      ],
      accounts: [
        {
          id: "primary",
          label: "Primary",
          status: "available",
          available: true,
          default: true,
          models: ["safe-a", "blocked"],
          entries: [],
        },
        {
          id: "backup",
          label: "Backup",
          status: "available",
          available: true,
          models: ["safe-a", "safe-b"],
          entries: [],
        },
      ],
    },
  })
  await page.getByRole("button", { name: "New profile" }).click()
  await expect(page).toHaveURL(/\/repository-reviews\/profiles\/new(?:\?|$)/)
  const editor = page.locator('[data-slot="collection-detail-shell"]')
  await expect(
    editor.getByRole("heading", { name: "New review profile" }),
  ).toBeVisible()
  const account = editor.getByRole("combobox", {
    name: "Execution account",
  })
  const model = editor.getByRole("combobox", { name: "Reviewer model" })
  const writer = editor.getByRole("combobox", {
    name: "Issue writer model",
  })
  await expect(account).toBeVisible()
  await expect(model).toHaveValue("safe-a")
  await expect(writer).toHaveValue("")
  await expect(
    writer.getByRole("option", { name: "Same as reviewer (safe-a)" }),
  ).toHaveCount(1)
  await expect(
    model.getByRole("option", {
      name: "blocked (Reviewer route is blocked by policy.)",
    }),
  ).toBeDisabled()
  await account.selectOption("backup")
  await expect(model).toHaveValue("safe-a")
  await model.selectOption("safe-b")
  await writer.selectOption("safe-b")
  await expect(writer).toHaveValue("safe-b")
  await account.selectOption("")
  await expect(model).toHaveValue("safe-a")
  await expect(writer).toHaveValue("")
  await expect(
    writer.getByRole("option", { name: "Same as reviewer (safe-a)" }),
  ).toHaveCount(1)
  await editor.getByRole("button", { name: /^Advanced/ }).click()

  await expect(editor.getByText("Guard expression reference")).toHaveCount(0)
  await editor.getByRole("button", { name: "Guard expression help" }).click()
  await expect(page.getByText("Guard expression reference")).toBeVisible()
  await page.keyboard.press("Escape")

  const guard = editor.getByRole("combobox", { name: "Guard expression" })
  await guard.fill("spent.tok")
  await expect(
    editor.getByRole("listbox", { name: "Guard expression suggestions" }),
  ).toBeVisible()
  await editor
    .getByRole("option", { name: "spent.tokens.total number field" })
    .click()
  await expect(guard).toHaveValue("spent.tokens.total ")
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
})

test.skip("legacy combined model review workspace is not compatibility-rendered", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<
    MockLauncherApiOptions["modelEvaluationRequests"]
  > = []
  await gotoMockedRoute(page, "/model-evaluations", {
    statefulModelEvaluations: true,
    modelEvaluationRequests: requests,
  })
  const workspace = page.getByRole("region", {
    name: "Model probe workspace",
  })

  await expect(workspace.getByText(/Comparison-only flow/)).toBeVisible()
  await expect(
    workspace.getByText(/do not create repository findings/),
  ).toBeVisible()

  await workspace.getByLabel("Repository", { exact: true }).fill("owner/repo")
  await expect(
    workspace.getByLabel("Review profile", { exact: true }),
  ).toHaveValue("rrpf_model_probe")
  await expect(
    workspace.getByRole("checkbox", { name: "Select candidate model code" }),
  ).toBeChecked()
  await expect(
    workspace.getByRole("checkbox", { name: "Select candidate model fast" }),
  ).toBeChecked()
  await expect(workspace.getByLabel("Frozen review profile")).toContainText(
    "128.0 KiB",
  )
  await workspace.getByRole("button", { name: "Run probe" }).click()

  await expect(
    workspace.getByRole("progressbar", { name: "Model probe progress" }),
  ).toHaveAttribute("aria-valuenow", "40")
  await expect(
    workspace.getByText("Model code · File pkg/service.go"),
  ).toBeVisible()
  const candidateCalls = workspace.getByRole("region", {
    name: "Candidate call progress",
  })
  await expect(candidateCalls.getByText("Batch 1/2")).toBeVisible()
  await expect(candidateCalls.getByText("1/4 candidate calls")).toBeVisible()
  await expect(candidateCalls.getByText("2 active")).toBeVisible()
  await expect(
    workspace.getByText("Repository model evaluation completed."),
  ).toBeVisible({ timeout: 5_000 })
  const runTabs = workspace.getByRole("tablist", {
    name: "Probe run details",
  })
  await expect(runTabs.getByRole("tab", { name: "Status" })).toHaveAttribute(
    "aria-selected",
    "true",
  )
  await expect(runTabs.getByRole("tab", { name: "Final report" })).toBeEnabled()
  await expect(
    workspace.getByLabel("Repository", { exact: true }),
  ).toBeDisabled()
  await expect(
    workspace.getByRole("button", { name: "Run probe" }),
  ).toHaveCount(0)
  await expect(workspace.getByRole("button", { name: "Restart" })).toHaveCount(
    0,
  )
  await expect(
    workspace.getByRole("button", { name: "Start over" }),
  ).toHaveCount(0)
  await expect(workspace.getByRole("button", { name: "Delete" })).toHaveCount(0)

  await runTabs.getByRole("tab", { name: "Final report" }).click()
  await expect(
    workspace.getByRole("heading", {
      name: "Use code when review quality matters.",
    }),
  ).toBeVisible()
  await expect(
    workspace.getByRole("heading", { name: "code work sizing" }),
  ).toBeVisible()
  await expect(
    workspace.getByText(/First eligible observed workload at 8 files/),
  ).toBeVisible()
  await expect(workspace.getByText(/Effective tokens =/)).toBeVisible()
  await workspace.getByRole("button", { name: "Open dedicated report" }).click()
  await expect(page).toHaveURL(/\/model-evaluations\/rme_[0-9a-f]{32}\/report$/)
  const report = page.getByRole("region", { name: "Model probe report" })
  await expect(
    report.getByRole("heading", {
      name: "Use code when review quality matters.",
    }),
  ).toBeVisible()
  await expect(
    report.getByRole("heading", { name: "Quality score comparison" }),
  ).toBeVisible()
  if ((page.viewportSize()?.width ?? 0) < 640) {
    await expect(
      report.getByText("Quality", { exact: true }).first(),
    ).toBeVisible()
    await expect(
      report.getByText("Time", { exact: true }).first(),
    ).toBeVisible()
  } else {
    await expect(
      report.getByRole("img", { name: /Efficiency graph/i }),
    ).toBeVisible()
  }
  await expect(
    report.getByRole("img", { name: /code: AI-judge supported claims 8/i }),
  ).toBeVisible()
  await expect(
    report.getByText("Best evidence-grounded analysis."),
  ).toBeVisible()
  await expect(report.getByText("Precise source evidence")).toBeVisible()
  await report.getByRole("button", { name: "View analysis" }).click()
  await expect(report.getByText("Missed important findings")).toBeVisible()

  expect(requests.map(({ method, path }) => `${method} ${path}`)).toEqual([
    "POST /api/model-evaluations/run",
  ])
  expect(requests[0]?.body).toMatchObject({
    repository: "owner/repo",
    profile_id: "rrpf_model_probe",
    candidate_models: ["code", "fast"],
    ref: "",
  })
  expect(requests[0]?.body).toEqual({
    repository: "owner/repo",
    profile_id: "rrpf_model_probe",
    candidate_models: ["code", "fast"],
    ref: "",
  })
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("Workflow configurations use canonical collections and routed editor URLs", async ({
  page,
}) => {
  test.setTimeout(60_000)
  const expectEditorURL = async (id: string, gate?: string) => {
    await expect
      .poll(() => new URL(page.url()).pathname)
      .toBe(`/development/workflow-configurations/${id}/edit`)
    expect(new URL(page.url()).searchParams.get("flow")).toBe("review")
    expect(new URL(page.url()).searchParams.get("gate")).toBe(gate ?? null)
  }
  await gotoMockedRoute(
    page,
    "/development/workflow-configurations?config=editable&view=grid",
  )

  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("ORDER BY name ASC")
  expect(new URL(page.url()).searchParams.get("config")).toBeNull()
  expect(new URL(page.url()).searchParams.get("view")).toBe("grid")
  await expect(
    page.getByRole("heading", { name: "Workflow configurations" }),
  ).toBeVisible()
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await page.goto(
    "/development/workflow-configurations/default/edit?flow=review",
  )
  await expectEditorURL("default")
  await expect(
    page.getByRole("textbox", { name: "Configuration name" }),
  ).toHaveValue("Default")
  await expect(
    page.getByRole("textbox", { name: "Configuration name" }),
  ).toBeDisabled()
  await expect(
    page.getByRole("combobox", { name: "Deferred issue mode" }),
  ).toBeEnabled()
  await page
    .getByRole("button", { name: "Why Configuration name is fixed" })
    .focus()
  await expect(page.getByRole("tooltip")).toContainText(
    "built-in default configuration has a fixed name",
  )

  await page.getByRole("button", { name: "Approve purpose and scope" }).click()
  await expectEditorURL("default", "pr.charter.confirm")
  const lockedDialog = page.getByRole("dialog", {
    name: "Approve purpose and scope",
  })
  await expect(
    lockedDialog.getByRole("combobox", { name: "Execution action" }),
  ).toBeDisabled()
  await expect(lockedDialog.getByRole("note")).toContainText(
    "Create a custom configuration to override Gate actions",
  )
  await lockedDialog.getByRole("button", { name: "Done editing" }).click()
  await page.goto(
    "/development/workflow-configurations/editable/edit?flow=review",
  )
  await expectEditorURL("editable")
  await page.getByRole("button", { name: "Approve purpose and scope" }).click()
  await expectEditorURL("editable", "pr.charter.confirm")
  const dialog = page.getByRole("dialog", {
    name: "Approve purpose and scope",
  })
  await expect(
    dialog.getByRole("combobox", { name: "Execution action" }),
  ).toContainText("Use workflow default (Human)")
  await expect(
    dialog.getByRole("heading", { name: "Expected answer" }),
  ).toBeVisible()
  await expect(dialog.getByText("What should happen?")).toBeVisible()
  await expect(
    dialog.getByText("Choose one: Approve · Request revision"),
  ).toBeVisible()
  await expect(dialog.getByText("Workflow reference")).toHaveCount(0)
  await expect(dialog.getByText(/Approve \(approve\)/)).toHaveCount(0)
  await dialog.getByRole("combobox", { name: "Execution action" }).click()
  await page.getByRole("option", { name: "AI", exact: true }).click()
  await expect(dialog.getByLabel("Agent ID")).toHaveValue("main")
  await dialog.getByRole("combobox", { name: "Session" }).click()
  await expect(
    page.getByRole("option", { name: "Originating snapshot" }),
  ).toHaveCount(0)
  await page.keyboard.press("Escape")
  await dialog.getByRole("button", { name: "Done editing" }).click()
  await expectEditorURL("editable")

  await page
    .getByRole("button", { name: "Decide ambiguous finding scope" })
    .click()
  await expectEditorURL("editable", "pr.finding.classify")
  const sourceDialog = page.getByRole("dialog", {
    name: "Decide ambiguous finding scope",
  })
  await sourceDialog.getByRole("combobox", { name: "Execution action" }).click()
  await page.getByRole("option", { name: "AI", exact: true }).click()

  await expect(sourceDialog.getByLabel("Agent ID")).toHaveValue("main")
  await expect(sourceDialog.getByLabel("Agent ID")).toBeEnabled()
  await expect(
    sourceDialog.getByText(/no history, cache, or tools/i),
  ).toBeVisible()
  await expect(sourceDialog.getByLabel("History", { exact: true })).toHaveCount(
    0,
  )
  await expect(sourceDialog.getByLabel("Cache", { exact: true })).toHaveCount(0)
  await expect(sourceDialog.getByLabel("Tools", { exact: true })).toHaveCount(0)

  await sourceDialog.getByRole("combobox", { name: "Session" }).click()
  await page.getByRole("option", { name: "Private snapshot" }).click()
  await expect(sourceDialog.getByLabel("Agent ID")).toBeEnabled()
  await expect(
    sourceDialog.getByRole("combobox", { name: "Cache" }),
  ).toHaveText("Session")
  await expect(
    sourceDialog.getByText(
      /frozen read-only history.*session cache.*no tools/i,
    ),
  ).toBeVisible()
  await expect(sourceDialog.getByLabel("History", { exact: true })).toHaveCount(
    0,
  )
  await expect(sourceDialog.getByLabel("Tools", { exact: true })).toHaveCount(0)

  await sourceDialog.getByRole("combobox", { name: "Session" }).click()
  await page.getByRole("option", { name: "Originating snapshot" }).click()
  await expect(
    sourceDialog.getByText(
      /same originating agent.*exact read-only snapshot.*no cache or tools/i,
    ),
  ).toBeVisible()
  await expect(sourceDialog.getByText(/stops execution/i)).toBeVisible()
  await expect(
    sourceDialog.getByLabel("Agent ID", { exact: true }),
  ).toHaveCount(0)
  await expect(sourceDialog.getByLabel("History", { exact: true })).toHaveCount(
    0,
  )
  await expect(sourceDialog.getByLabel("Cache", { exact: true })).toHaveCount(0)
  await expect(sourceDialog.getByLabel("Tools", { exact: true })).toHaveCount(0)
  await sourceDialog.getByRole("combobox", { name: "Execution action" }).click()
  await page.getByRole("option", { name: "Deterministic" }).click()
  await sourceDialog.getByText("JSON keys and allowed values").click()
  await expect(sourceDialog.getByText("action", { exact: true })).toBeVisible()
  await expect(sourceDialog.getByText("approve", { exact: true })).toBeVisible()
  await sourceDialog.getByRole("button", { name: "Done editing" }).click()
  await expectEditorURL("editable")
  await expectNoSeriousA11yViolations(page)
  await expect(
    page.getByRole("combobox", { name: "Deferred issue mode" }),
  ).toHaveText("Ask")
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
})

test("Repository assignments have a separate canonical UI and URL", async ({
  page,
}) => {
  await gotoMockedRoute(
    page,
    "/development/repositories?config=editable&view=table",
  )

  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("ORDER BY repository ASC")
  expect(new URL(page.url()).searchParams.get("config")).toBeNull()
  expect(new URL(page.url()).searchParams.get("view")).toBe("table")
  await expect(
    page.getByRole("heading", { name: "Repository assignments" }),
  ).toBeVisible()
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect(
    page.locator(
      `[data-item-id="${smokeDevelopmentAdminIDs.repositoryAssignment}"]:visible`,
    ),
  ).toBeVisible()

  await page.goto(
    `/development/repositories/${smokeDevelopmentAdminIDs.repositoryAssignment}`,
  )
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expect(page.getByText("https://github.com|100")).toBeVisible()

  await page.goto("/development/repositories/new")
  await expect(
    page.getByRole("textbox", { name: "Repository", exact: true }),
  ).toBeEditable()
  await expect(
    page.getByText("Repository routing", { exact: true }),
  ).toBeVisible()

  await page.goto(
    `/development/repositories/${smokeDevelopmentAdminIDs.repositoryAssignment}/edit`,
  )
  const repository = page.getByRole("textbox", {
    name: "Repository",
    exact: true,
  })
  await expect(repository).toHaveValue("octo/repo")
  await expect(repository).toHaveAttribute("readonly")
  await expect(
    page.getByRole("button", { name: "Save assignment" }),
  ).toBeDisabled()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
})

test.skip("legacy agent card and sheet workflow is not compatibility-rendered", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  const errors = collectPageErrors(page)
  const agentRequests: NonNullable<MockLauncherApiOptions["agentRequests"]> = []
  await gotoMockedRoute(page, "/agent/agents", {
    statefulAgents: true,
    agentRequests,
  })

  // Establish an in-app history entry, then verify browser Back uses the same
  // discard confirmation as closing a dirty editor.
  await page.getByRole("link", { name: "Models", exact: true }).click()
  await expect(page).toHaveURL(/\/models$/)
  await page.getByRole("link", { name: "Agents", exact: true }).click()
  await expect(page).toHaveURL(/\/agent\/agents$/)

  await page.getByRole("button", { name: "Edit reviewer" }).click()
  const reviewerSheet = page.getByRole("dialog", { name: "Edit agent" })
  await reviewerSheet
    .getByRole("textbox", { name: "Configured name" })
    .fill("Review team")
  await page.evaluate(() => window.history.back())

  const navigationDiscard = page.getByRole("alertdialog", {
    name: "Discard unsaved changes?",
  })
  await expect(navigationDiscard).toBeVisible()
  await navigationDiscard.getByRole("button", { name: "Keep editing" }).click()
  await expect(page).toHaveURL(/\/agent\/agents$/)
  await reviewerSheet.getByRole("button", { name: "Cancel" }).click()
  await page
    .getByRole("alertdialog", { name: "Discard unsaved changes?" })
    .getByRole("button", { name: "Discard changes" })
    .click()
  await expect(reviewerSheet).toBeHidden()

  await page.getByRole("button", { name: "Create agent" }).click()
  const createSheet = page.getByRole("dialog", { name: "Create agent" })
  await createSheet.getByRole("textbox", { name: "Agent ID" }).fill("triager")
  await createSheet
    .getByRole("textbox", { name: "Configured name" })
    .fill("Triager")
  await createSheet.getByRole("combobox", { name: "Provider account" }).click()
  await page.getByRole("option", { name: "gpt-4o", exact: true }).click()
  await createSheet
    .getByRole("combobox", { name: "Primary alias policy" })
    .click()
  await page.getByRole("option", { name: "Custom", exact: true }).click()
  await createSheet
    .getByRole("combobox", { name: "Primary model alias" })
    .click()
  await page.getByRole("option", { name: "code", exact: true }).click()
  await createSheet
    .getByRole("combobox", { name: "Fallback alias policy" })
    .click()
  await page.getByRole("option", { name: "Custom", exact: true }).click()
  await createSheet.getByRole("combobox", { name: "Fallback order" }).click()
  await page.getByRole("option", { name: "fast", exact: true }).click()
  await createSheet
    .getByRole("button", { name: "Add fallback order entry" })
    .click()
  await createSheet.getByRole("button", { name: "Save" }).click()

  const triagerCard = page.locator('[data-agent-id="triager"]')
  await expect(triagerCard).toBeVisible()
  await triagerCard.getByRole("button", { name: "Edit triager" }).click()

  const editSheet = page.getByRole("dialog", { name: "Edit agent" })
  await editSheet
    .getByRole("combobox", { name: "Fallback alias policy" })
    .click()
  await page.getByRole("option", { name: "None", exact: true }).click()
  await editSheet.getByRole("button", { name: "Save" }).click()
  await expect(triagerCard).toContainText("None")

  await triagerCard.getByRole("button", { name: "Set default" }).click()
  await expect(triagerCard.getByText("Default", { exact: true })).toBeVisible()

  await triagerCard.getByRole("button", { name: "Delete triager" }).click()
  const deleteDialog = page.getByRole("alertdialog", {
    name: "Delete agent?",
  })
  await deleteDialog
    .getByRole("button", { name: "Delete agent", exact: true })
    .click()
  await expect(triagerCard).toHaveCount(0)

  expect(agentRequests).toEqual([
    {
      method: "POST",
      path: "/api/agents",
      body: {
        expected_config_revision: "agent-revision-1",
        agent: {
          id: "triager",
          name: "Triager",
          workspace: "",
          account_ref: "gpt-4o",
          model: { primary: "code", fallbacks: ["fast"] },
          skills: null,
          subagents: null,
        },
      },
    },
    {
      method: "PUT",
      path: "/api/agents/triager",
      body: {
        expected_config_revision: "agent-revision-2",
        agent: {
          id: "triager",
          name: "Triager",
          workspace: "",
          account_ref: "gpt-4o",
          model: { primary: "code", fallbacks: [] },
          skills: null,
          subagents: null,
        },
      },
    },
    {
      method: "POST",
      path: "/api/agents/triager/default",
      body: { expected_config_revision: "agent-revision-3" },
    },
    {
      method: "DELETE",
      path: "/api/agents/triager",
      body: { expected_config_revision: "agent-revision-4" },
    },
  ])
  expect(errors).toEqual([])
})

test.skip("legacy agent query-tab detail workflow is not compatibility-rendered", async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 720 })
  const errors = collectPageErrors(page)
  const capabilityRequests: NonNullable<
    MockLauncherApiOptions["agentCapabilityRequests"]
  > = []
  const activityRequests: NonNullable<
    MockLauncherApiOptions["agentActivityRequests"]
  > = []
  await page.routeWebSocket(/\/pico\/ws/, () => undefined)
  await gotoMockedRoute(page, "/agent/agents", {
    agentCapabilityRequests: capabilityRequests,
    agentActivityRequests: activityRequests,
    gatewayRunning: true,
  })

  await page
    .locator('[data-agent-id="reviewer"]')
    .getByRole("button", { name: "Manage reviewer" })
    .click()
  await expect(page).toHaveURL(/\/agent\/agents\?agent=reviewer&tab=overview$/)
  const tabs = page.getByRole("tablist", { name: "Agent management" })
  await tabs.getByRole("tab", { name: "Capabilities" }).click()
  await expect(page).toHaveURL(/tab=capabilities$/)
  await expect(tabs.getByRole("tab", { name: "Capabilities" })).toHaveAttribute(
    "aria-selected",
    "true",
  )
  await expect(page.getByText("Existing unknown selections")).toBeVisible()
  await page
    .getByRole("button", {
      name: "Remove unknown selection legacy_unknown",
    })
    .click()
  await page.getByRole("button", { name: "Save capabilities" }).click()
  await expect
    .poll(() => capabilityRequests.filter((entry) => entry.method === "PATCH"))
    .toEqual([
      {
        method: "PATCH",
        path: "/api/agents/reviewer/capabilities",
        body: {
          expected_revision: "capability-revision-1",
          tools: { mode: "selected", values: ["web_search"] },
        },
      },
    ])

  const tools = page.getByRole("group", { name: "Tools" })
  await tools.getByRole("radio", { name: "No tools" }).click()
  await tabs.getByRole("tab", { name: "Activity" }).click()
  const discard = page.getByRole("alertdialog", {
    name: "Discard capability changes?",
  })
  await expect(discard).toBeVisible()
  await discard.getByRole("button", { name: "Keep editing" }).click()
  await expect(page).toHaveURL(/tab=capabilities$/)

  await tabs.getByRole("tab", { name: "Activity" }).click()
  await page
    .getByRole("alertdialog", { name: "Discard capability changes?" })
    .getByRole("button", { name: "Discard changes" })
    .click()
  await expect(page).toHaveURL(/tab=activity$/)
  await expect(page.getByText("Tool execution ended")).toBeVisible()
  await expect(page.getByText(/web_search; 25 ms; completed/)).toBeVisible()
  await expect(page.getByText(/cursor was reset/)).toBeVisible()
  expect(activityRequests.length).toBeGreaterThan(0)
  await expect(page.locator("body")).not.toContainText("CANARY_")

  await tabs.getByRole("tab", { name: "Activity" }).focus()
  await page.keyboard.press("Home")
  await expect(page).toHaveURL(/tab=overview$/)
  await expect(tabs.getByRole("tab", { name: "Overview" })).toBeFocused()

  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("events payload stays opt-in and replay remains deliberate", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  let payloadRequests = 0
  let replayRequests = 0

  page.on("request", (request) => {
    const path = new URL(request.url()).pathname
    if (path === `/api/events/${eventResponse.id}/payload`) {
      payloadRequests += 1
    }
    if (path === `/api/events/${eventResponse.id}/replay`) {
      replayRequests += 1
    }
  })

  await gotoMockedRoute(page, "/events")
  await expect(
    page.getByRole("button", { name: /issues\.opened.*github\/triage/ }),
  ).toBeVisible()
  await expect(
    page.getByRole("link", { name: "Implement feature" }),
  ).toHaveAttribute(
    "href",
    "/development/new?issue=https%3A%2F%2Fgithub.com%2Focto%2Frepo%2Fissues%2F42",
  )
  await expect(page.getByRole("button", { name: "Show payload" })).toBeVisible()
  expect(payloadRequests).toBe(0)

  await page.getByRole("button", { name: "Show payload" }).click()
  await expect(page.locator("pre")).toHaveText(eventPayloadText)
  expect(payloadRequests).toBe(1)

  await page.getByRole("button", { name: "Replay", exact: true }).click()
  const dialog = page.getByRole("alertdialog")
  await expect(dialog).toContainText(
    "may repeat workflows and external effects",
  )
  await dialog.getByRole("button", { name: "Cancel" }).click()
  await expect(dialog).toBeHidden()
  expect(replayRequests).toBe(0)

  await page.getByRole("button", { name: "Replay", exact: true }).click()
  await dialog
    .getByRole("button", { name: "Replay event", exact: true })
    .click()
  await expect.poll(() => replayRequests).toBe(1)
  await expect(page).toHaveURL(new RegExp(`event=${replayEventID}`))
  await expect(page.locator("pre")).toHaveCount(0)

  await page.waitForTimeout(100)
  expect(replayRequests).toBe(1)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("dispatch and workflow deep links survive reload with exact relationships", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    `/events?view=dispatches&dispatch=${eventDispatchResponse.id}`,
  )

  await expect(
    page.getByText(eventDispatchResponse.workflow_revision),
  ).toBeVisible()
  await expect(page.getByRole("link", { name: "Open event" })).toHaveAttribute(
    "href",
    `/events?event=${eventResponse.id}`,
  )
  await expect(
    page.getByRole("link", { name: "Open workflow" }),
  ).toHaveAttribute(
    "href",
    `/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/github-issue-triage.yml"]}`,
  )
  await expect(page.getByRole("link", { name: "Open run" })).toHaveAttribute(
    "href",
    "/agent/workflows/runs/wr_smoke",
  )
  await page.reload()
  await expect(
    page.getByText(eventDispatchResponse.workflow_revision),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)

  await page.goto("/agent/workflows/runs/wr_test")
  await expect(page.getByText("wr_test", { exact: true }).first()).toBeVisible()
  await expect(
    page.getByRole("link", { name: "workflows/summarize-text.yml" }),
  ).toHaveAttribute(
    "href",
    new RegExp(
      `^/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}(?:\\?|$)`,
    ),
  )
  await page.reload()
  await expect(page.getByText("wr_test", { exact: true }).first()).toBeVisible()
  await expect(page).toHaveURL(/\/agent\/workflows\/runs\/wr_test(?:\?|$)/)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow lifecycle links use only validated server origin", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/agent/workflows/runs/wr_lifecycle")

  await expect(
    page.getByRole("link", { name: lifecycleEventID }),
  ).toHaveAttribute("href", `/events?event=${lifecycleEventID}`)
  await expect(
    page.getByRole("link", { name: lifecycleDispatchID }),
  ).toHaveAttribute(
    "href",
    `/events?view=dispatches&dispatch=${lifecycleDispatchID}`,
  )
  await expect(
    page.getByRole("link", { name: "wr_lifecycle_root" }),
  ).toHaveAttribute(
    "href",
    /^\/agent\/workflows\/runs\/wr_lifecycle_root(?:\?|$)/,
  )
  await expect(page.locator(`a[href*="${lifecycleDecoyEventID}"]`)).toHaveCount(
    0,
  )
  await expect(
    page.locator(`a[href*="${lifecycleDecoyDispatchID}"]`),
  ).toHaveCount(0)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow cancellation requires and persists an accessible explicit reason", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const workflowCancelReasons: string[] = []
  await gotoMockedRoute(page, "/agent/workflows/runs/wr_cancel", {
    workflowCancelReasons,
  })

  const cancelButton = page.getByRole("button", { name: "Cancel" })
  await expect(cancelButton).toBeEnabled()
  await cancelButton.click()
  const dialog = page.getByRole("alertdialog")
  await expect(dialog).toContainText("wr_cancel")
  await expectElementFitsViewport(
    page,
    '[data-slot="alert-dialog-content"]',
    "workflow cancel dialog",
  )
  await expect(
    dialog.getByRole("button", { name: "Cancel run" }),
  ).toBeDisabled()
  await dialog.getByRole("textbox", { name: "Cancel reason" }).fill("   ")
  await expect(dialog.getByText("A cancel reason is required.")).toBeVisible()
  await expect(
    dialog.getByRole("button", { name: "Cancel run" }),
  ).toBeDisabled()
  await dialog
    .getByRole("textbox", { name: "Cancel reason" })
    .fill("operator intervention")
  await dialog.getByRole("button", { name: "Cancel run" }).click()

  await expect
    .poll(() => workflowCancelReasons)
    .toEqual(["operator intervention"])
  await expect(dialog).toBeHidden()
  await expect(page.getByText("Cancel requested")).toBeVisible()
  await expect(page.getByText("Completed")).toBeVisible()
  await expect(
    page.getByText("operator intervention", { exact: true }).first(),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test.skip("legacy combined MCP settings and server page is not compatibility-rendered", async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 720 })
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/agent/mcp")

  await expect(page.getByText("github", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "Reconnect" })).toBeVisible()
  await expect(page.getByText("local-files", { exact: true })).toBeVisible()

  await page.getByRole("button", { name: "Add server" }).first().click()
  const sheet = page.getByRole("dialog", { name: "Add MCP server" })
  await expect(sheet).toBeVisible()
  await expectElementFitsViewport(
    page,
    '[data-slot="sheet-content"]',
    "MCP add sheet",
  )
  await sheet.getByRole("combobox", { name: "Authentication" }).first().click()
  await page.getByRole("option", { name: "OAuth login" }).click()
  await expect(
    sheet.getByRole("button", { name: "Save & log in" }),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)

  await sheet.getByRole("button", { name: "Close" }).click()
  await page.getByRole("button", { name: "Settings" }).first().click()
  const settings = page.getByRole("dialog", { name: "MCP settings" })
  await expect(settings).toBeVisible()
  await settings.getByRole("switch", { name: "Deferred discovery" }).click()
  await expect(
    settings.getByRole("spinbutton", { name: "Tool-use TTL" }),
  ).toBeVisible()
  await expectElementFitsViewport(
    page,
    '[data-slot="sheet-content"]',
    "MCP settings sheet",
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test.skip("legacy MCP server sheet workflow is not compatibility-rendered", async ({
  page,
}) => {
  const requests: NonNullable<MockLauncherApiOptions["mcpRequests"]> = []
  await gotoMockedRoute(page, "/agent/mcp", {
    statefulMCP: true,
    mcpRequests: requests,
  })

  await page.getByRole("button", { name: "Add server" }).first().click()
  let sheet = page.getByRole("dialog", { name: "Add MCP server" })
  await sheet.getByRole("textbox", { name: "Name" }).fill("linear")
  await sheet
    .getByRole("textbox", { name: "Server URL" })
    .fill("https://mcp.linear.example/api")
  await sheet.getByRole("combobox", { name: "Authentication" }).click()
  await page.getByRole("option", { name: "Bearer token" }).click()
  await sheet.getByLabel("Set token").fill("linear-secret")
  await sheet.getByRole("button", { name: "Save & test" }).click()

  await expect(page.getByText("linear", { exact: true })).toBeVisible()
  expect(
    requests.some(
      (request) =>
        request.method === "POST" &&
        request.path === "/api/mcp/servers" &&
        (request.body as MCPServerInput).auth_mode === "bearer",
    ),
  ).toBe(true)
  expect(
    requests.some(
      (request) =>
        request.method === "PUT" &&
        request.path === "/api/mcp/servers/linear/credential" &&
        (request.body as { token?: string }).token === "linear-secret",
    ),
  ).toBe(true)
  expect(
    requests.some(
      (request) =>
        request.method === "POST" && request.path === "/api/mcp/servers/test",
    ),
  ).toBe(true)

  await page.getByRole("button", { name: "Edit linear" }).click()
  sheet = page.getByRole("dialog", { name: "Edit MCP server" })
  await sheet.getByRole("textbox", { name: "Name" }).fill("linear-team")
  await sheet.getByRole("combobox", { name: "Authentication" }).click()
  await page.getByRole("option", { name: "Custom headers" }).click()
  await sheet.getByRole("button", { name: "Add entry" }).click()
  await sheet.getByRole("textbox", { name: "Key" }).fill("X-Linear-Key")
  await sheet
    .getByRole("textbox", { name: "Value", exact: true })
    .fill("header-secret")
  await sheet.getByRole("button", { name: "Save", exact: true }).click()

  await expect(page.getByText("linear-team", { exact: true })).toBeVisible()
  expect(
    requests.some((request) => {
      if (
        request.method !== "PUT" ||
        request.path !== "/api/mcp/servers/linear"
      ) {
        return false
      }
      const body = request.body as MCPServerInput
      return (
        body.name === "linear-team" &&
        body.auth_mode === "custom" &&
        body.headers?.["X-Linear-Key"] === "header-secret"
      )
    }),
  ).toBe(true)

  await page.getByRole("button", { name: "Delete linear-team" }).click()
  const confirm = page.getByRole("alertdialog", {
    name: "Delete MCP server?",
  })
  await confirm.getByRole("button", { name: "Delete server" }).click()
  await expect(page.getByText("linear-team", { exact: true })).toHaveCount(0)
  expect(
    requests.some(
      (request) =>
        request.method === "DELETE" &&
        request.path === "/api/mcp/servers/linear-team",
    ),
  ).toBe(true)
})

test("accounts collection preserves route state through detail and onboarding", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/accounts?view=list")

  const url = new URL(page.url())
  expect(url.pathname).toBe("/accounts")
  expect(url.searchParams.get("q")).toBe("ORDER BY provider ASC, id ASC")
  expect(url.searchParams.get("view")).toBe("list")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect(page.locator("[data-item-id]")).toHaveCount(2)
  await expect(page.getByText("primary", { exact: true })).toBeVisible()
  await expect(page.getByText("review", { exact: true })).toBeVisible()
  await expect(page.getByText("Usage limits")).toHaveCount(0)

  const primary = page.locator(`[data-item-id="${smokeAccountIDs.primary}"]`)
  await primary.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(`/accounts/${smokeAccountIDs.primary}\\?.*view=list`),
  )
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expect(
    page.getByRole("definition").filter({ hasText: "openai:primary" }),
  ).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "Usage limits" }),
  ).toBeVisible()
  await expect(page.getByText("64%", { exact: true })).toBeVisible()

  await page.getByRole("button", { name: "All accounts" }).click()
  await expect(page).toHaveURL(/\/accounts\?.*view=list/)
  await page.getByRole("button", { name: "Add account" }).click()
  await expect(page).toHaveURL(/\/accounts\/new\?.*view=list/)
  await expect(
    page.getByRole("heading", { name: "Onboard Account" }),
  ).toBeVisible()
  await page.getByRole("combobox", { name: "Provider" }).click()
  await expect(page.getByRole("option", { name: "DeepSeek" })).toBeVisible()
  await expect(
    page.getByRole("option", { name: "Google Gemini" }),
  ).toBeVisible()
  await page.keyboard.press("Escape")
  await expect(page.getByPlaceholder("work")).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("Codex renewal opens device login directly for the exact account", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const credentialID = "openai:expired-work"
  const accountID = accountIDForCredential(credentialID)
  const flowID = "device-renewal"
  const userCode = "ABCD-EFGH"
  const verifyURL = "https://auth.openai.com/device"
  let releaseDeviceLogin!: () => void
  const deviceLoginPending = new Promise<void>((resolve) => {
    releaseDeviceLogin = resolve
  })

  await gotoMockedRoute(page, `/accounts/${accountID}/edit`, {
    accounts: [
      {
        id: accountID,
        provider: "openai",
        account: credentialID,
        status: "expired",
        auth_method: "oauth",
        expires_at: "2026-08-01T12:00:00Z",
      },
    ],
    oauthProviders: [
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
            credential_id: credentialID,
            display_name: "OpenAI",
            methods: ["browser", "device_code", "token"],
            logged_in: true,
            status: "expired",
            auth_method: "oauth",
            account_id: "acct-expired-work",
          },
        ],
      },
    ],
    codexAccountLimits: { accounts: [] },
  })
  await page.route("**/api/oauth/login", async (route) => {
    await deviceLoginPending
    return json(route, {
      status: "pending",
      provider: "openai",
      credential_id: credentialID,
      method: "device_code",
      flow_id: flowID,
      user_code: userCode,
      verify_url: verifyURL,
      interval: 30,
    })
  })
  await page.route(`**/api/oauth/flows/${flowID}/poll`, async (route) => {
    return json(route, {
      flow_id: flowID,
      provider: "openai",
      credential_id: credentialID,
      method: "device_code",
      status: "pending",
      user_code: userCode,
      verify_url: verifyURL,
      interval: 30,
    })
  })

  await expect(
    page.getByRole("heading", { name: "Renew Account Login" }),
  ).toBeVisible()
  await expect(page.getByLabel("Credential ID")).toHaveValue(credentialID)
  const renew = page.getByRole("button", { name: "Start Renewal" })
  await expect(renew).toBeVisible()
  const firstLoginRequest = page.waitForRequest((request) => {
    const url = new URL(request.url())
    return request.method() === "POST" && url.pathname === "/api/oauth/login"
  })
  const firstLoginResponse = page.waitForResponse((response) => {
    const url = new URL(response.url())
    return (
      response.request().method() === "POST" &&
      url.pathname === "/api/oauth/login"
    )
  })
  await renew.click()

  expect((await firstLoginRequest).postDataJSON()).toEqual({
    provider: "openai",
    credential_id: credentialID,
    method: "device_code",
  })
  const deviceLogin = page.getByRole("dialog", {
    name: "OpenAI Device Login",
  })
  await expect(deviceLogin).toBeVisible()
  await expect(deviceLogin.getByText("Starting device login...")).toBeVisible()
  await expect(
    deviceLogin.getByRole("button", { name: "Copy User Code" }),
  ).toBeDisabled()
  await expect(
    deviceLogin.getByRole("button", { name: "Copy Verification URL" }),
  ).toBeDisabled()
  await expect(
    deviceLogin.getByRole("button", { name: "Open Verification Page" }),
  ).toBeDisabled()

  await deviceLogin.getByRole("button", { name: "Cancel" }).click()
  await expect(deviceLogin).toBeHidden()
  releaseDeviceLogin()
  await firstLoginResponse
  await page.waitForTimeout(100)
  await expect(deviceLogin).toBeHidden()
  await expect(renew).toBeEnabled()

  const loginRequest = page.waitForRequest((request) => {
    const url = new URL(request.url())
    return request.method() === "POST" && url.pathname === "/api/oauth/login"
  })
  await renew.click()
  expect((await loginRequest).postDataJSON()).toEqual({
    provider: "openai",
    credential_id: credentialID,
    method: "device_code",
  })
  await expect(deviceLogin).toBeVisible()

  await expect(deviceLogin.getByText(userCode)).toBeVisible()
  await expect(deviceLogin.getByRole("link", { name: verifyURL })).toBeVisible()
  await expect(
    deviceLogin.getByRole("button", { name: "Copy User Code" }),
  ).toContainText("📋")
  await expect(
    deviceLogin.getByRole("button", { name: "Copy Verification URL" }),
  ).toContainText("📋")

  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  await deviceLogin.getByRole("button", { name: "Cancel" }).click()
  expect(errors).toEqual([])
})

test("token account renewal keeps its method and exact identity locked", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const credentialID = "anthropic:expired-work"
  const accountID = accountIDForCredential(credentialID)
  const replacementToken = "replacement-account-token"

  await gotoMockedRoute(page, `/accounts/${accountID}/edit`, {
    accounts: [
      {
        id: accountID,
        provider: "anthropic",
        account: credentialID,
        status: "expired",
        auth_method: "token",
        expires_at: "",
      },
    ],
    oauthProviders: [
      {
        provider: "anthropic",
        credential_id: "anthropic",
        display_name: "Anthropic",
        methods: ["token"],
        logged_in: true,
        status: "connected",
        credentials: [
          {
            provider: "anthropic",
            credential_id: credentialID,
            display_name: "Anthropic",
            methods: ["token"],
            logged_in: true,
            status: "expired",
            auth_method: "token",
          },
        ],
      },
    ],
    codexAccountLimits: { accounts: [] },
  })

  const renewal = page.locator('[data-slot="collection-detail-shell"]')
  await expect(renewal).toBeVisible()
  await expect(renewal.getByLabel("Provider")).toHaveValue("Anthropic")
  await expect(renewal.getByLabel("Provider")).toHaveJSProperty(
    "readOnly",
    true,
  )
  await expect(renewal.getByLabel("Login Method")).toHaveValue("Token")
  await expect(renewal.getByLabel("Login Method")).toHaveJSProperty(
    "readOnly",
    true,
  )
  await expect(renewal.getByLabel("Credential ID")).toHaveValue(credentialID)
  await expect(renewal.getByLabel("Credential ID")).toHaveJSProperty(
    "readOnly",
    true,
  )
  await expect(renewal.getByRole("combobox")).toHaveCount(0)
  await renewal.getByPlaceholder("Anthropic token").fill(replacementToken)

  const loginRequest = page.waitForRequest((request) => {
    const url = new URL(request.url())
    return request.method() === "POST" && url.pathname === "/api/oauth/login"
  })
  await renewal.getByRole("button", { name: "Save New Token" }).click()

  expect((await loginRequest).postDataJSON()).toEqual({
    provider: "anthropic",
    credential_id: credentialID,
    method: "token",
    token: replacementToken,
  })
  await expect(page).toHaveURL(new RegExp(`/accounts/${accountID}(?:\\?|$)`))
  expect(errors).toEqual([])
})

test("a rejected account renewal remains visible on its editor route", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop", "desktop error smoke coverage")
  const credentialID = "anthropic:expired-work"
  const accountID = accountIDForCredential(credentialID)

  await gotoMockedRoute(page, `/accounts/${accountID}/edit`, {
    accounts: [
      {
        id: accountID,
        provider: "anthropic",
        account: credentialID,
        status: "expired",
        auth_method: "token",
        expires_at: "",
      },
    ],
    oauthProviders: [
      {
        provider: "anthropic",
        credential_id: "anthropic",
        display_name: "Anthropic",
        methods: ["token"],
        logged_in: true,
        status: "connected",
        credentials: [
          {
            provider: "anthropic",
            credential_id: credentialID,
            display_name: "Anthropic",
            methods: ["token"],
            logged_in: true,
            status: "expired",
            auth_method: "token",
          },
        ],
      },
    ],
    codexAccountLimits: { accounts: [] },
  })
  await page.route("**/api/oauth/login", async (route) => {
    await route.fulfill({
      status: 400,
      contentType: "text/plain",
      body: "Replacement token was rejected",
    })
  })

  const renewal = page.locator('[data-slot="collection-detail-shell"]')
  await renewal.getByLabel("Token").fill("rejected-token")
  await renewal.getByRole("button", { name: "Save New Token" }).click()

  await expect(renewal).toBeVisible()
  await expect(renewal.getByRole("alert")).toHaveText(
    "Replacement token was rejected",
  )
  await expect(page).toHaveURL(new RegExp(`/accounts/${accountID}/edit`))
})

test.skip("legacy combined models page is not compatibility-rendered", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/models")

  await expect(
    page.getByRole("heading", { name: "Runtime selection" }),
  ).toHaveCount(0)
  await expect(
    page.getByRole("heading", { name: "Provider accounts" }),
  ).toHaveCount(0)

  const aliasSection = page.locator("section").filter({
    has: page.getByRole("heading", { name: "Developer aliases" }),
  })
  const codingAlias = aliasSection.locator("article").filter({
    has: page.getByRole("heading", { name: "code" }),
  })
  await expect(codingAlias.getByRole("heading", { name: "code" })).toBeVisible()
  await expect(codingAlias.getByText("gpt-4o-mini")).toBeVisible()
  const investigateAlias = aliasSection.locator("article").filter({
    has: page.getByRole("heading", { name: "investigate" }),
  })
  await expect(investigateAlias.getByText("Not configured")).toBeVisible()
  await expect(
    investigateAlias.getByText(
      "Deep research, root-cause analysis, and unfamiliar code.",
    ),
  ).toBeVisible()
  await codingAlias.getByRole("button", { name: "Edit model alias" }).click()

  const editor = page.getByRole("dialog", { name: "Edit model alias" })
  await expect(editor).toBeVisible()
  await expect(editor.getByRole("textbox").first()).toHaveValue("code")
  await expect(editor.getByRole("textbox").first()).toBeDisabled()
  await expect(
    editor.getByText(
      "Choose another model or disable this alias for a concrete account.",
    ),
  ).toBeVisible()
  await editor.getByRole("combobox", { name: "Default model" }).click()
  const sharedModel = page.getByRole("option").filter({ hasText: "gpt-5.4" })
  await expect(sharedModel.getByText("All accounts (2)")).toBeVisible()
  const accountSpecificModel = page.getByRole("option", {
    name: /^gpt-4o-mini/,
  })
  await expect(accountSpecificModel.getByText(/Missing: gpt-4o/)).toBeVisible()
  const solModel = page.getByRole("option", { name: /^gpt-5\.5-sol/ })
  await expect(solModel.getByText(/Missing: gpt-4o-mini/)).toBeVisible()
  const defaultModelSearch = page.getByPlaceholder("Search models...")
  await expect(defaultModelSearch).toBeFocused()
  await page.keyboard.type("gpt-5.5-sol")
  await expect(defaultModelSearch).toHaveValue("gpt-5.5-sol")
  await expect(solModel).toBeVisible()
  await expect(sharedModel).toHaveCount(0)
  await expect(accountSpecificModel).toHaveCount(0)
  await defaultModelSearch.fill("")
  await page.keyboard.press("Escape")
  await expect(
    editor.getByRole("button", { name: "Add override" }),
  ).toBeEnabled()
  await editor.getByRole("button", { name: "Add override" }).click()
  await editor.getByRole("combobox", { name: "Override model" }).last().click()
  await expect(
    page.getByRole("option", { name: "Disabled for this account" }),
  ).toBeVisible()
  await expect(
    page.getByRole("option", { name: /^gpt-4o-mini All accounts/ }),
  ).toBeVisible()
  await expect(
    page.getByRole("option", { name: "gpt-5.4 All accounts (1)" }),
  ).toBeVisible()
  const crossAccountModel = page
    .getByRole("option", { name: /^gpt-4o / })
    .last()
  await expect(
    crossAccountModel.getByText("Missing: gpt-4o-mini", { exact: false }),
  ).toBeVisible()
  const overrideSolModel = page
    .getByRole("option", {
      name: /^gpt-5\.5-sol/,
    })
    .last()
  await expect(
    overrideSolModel.getByText("Missing: gpt-4o-mini", { exact: false }),
  ).toBeVisible()
  const overrideModelSearch = page.getByPlaceholder("Search models...").last()
  await expect(overrideModelSearch).toBeFocused()
  await page.keyboard.type("gpt-5.5-sol")
  await expect(overrideModelSearch).toHaveValue("gpt-5.5-sol")
  await expect(overrideSolModel).toBeVisible()
  await expect(
    page.getByRole("option", { name: /^gpt-4o-mini All accounts/ }),
  ).toHaveCount(0)
  await expect(crossAccountModel).toHaveCount(0)
  await page.keyboard.press("Escape")
  expect(errors).toEqual([])
})

test.skip("legacy model alias dialog is not compatibility-rendered", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/models", {
    modelResponse: {
      ...modelResponse,
      models: modelResponse.models
        .filter((model) => model.provider !== "model-router")
        .map((model) => ({ ...model, enabled: false })),
      total: 2,
    },
  })

  const aliasSection = page.locator("section").filter({
    has: page.getByRole("heading", { name: "Developer aliases" }),
  })
  const investigateAlias = aliasSection.locator("article").filter({
    has: page.getByRole("heading", { name: "investigate" }),
  })
  await investigateAlias
    .getByRole("button", { name: "Configure model alias" })
    .click()

  const editor = page.getByRole("dialog", { name: "Configure model alias" })
  await expect(
    editor.getByText(
      "No enabled accounts are available. Add or restore one on the Accounts page before choosing models or overrides.",
    ),
  ).toBeVisible()
  await expect(
    editor.getByRole("combobox", { name: "Default model" }),
  ).toBeDisabled()
  await expect(
    editor.getByRole("button", { name: "Add override" }),
  ).toBeDisabled()
  expect(errors).toEqual([])
})

test("account routers use a separate keyboard-complete collection and detail route", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/accounts")
  await expect(page.getByText("balanced-router")).toHaveCount(0)
  if ((page.viewportSize()?.width ?? 0) >= 640) {
    await page.getByRole("button", { name: "Account routers" }).click()
  } else {
    await page.goto("/accounts/routers")
  }
  await expect(page).toHaveURL(/\/accounts\/routers\?.*q=ORDER/)
  await expect(
    page.getByRole("heading", { name: "Account routers", exact: true }),
  ).toBeVisible()

  const balanced = page.locator(`[data-item-id="${smokeAccountIDs.router}"]`)
  await balanced.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await balanced.click({ button: "right" })
  await expect(page.getByRole("menuitem", { name: "Open" })).toBeVisible()
  await expect(
    page.getByRole("menuitem", { name: "Edit router" }),
  ).toBeVisible()
  await page.keyboard.press("Escape")
  await balanced.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(`/accounts/routers/${smokeAccountIDs.router}`),
  )
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "Decision blocks" }),
  ).toBeVisible()
  await expect(page.getByText("pool · load_balance")).toBeVisible()
  await page.getByRole("button", { name: "All account routers" }).click()
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("event sources use configured-only routed collection and revision-fenced bulk actions", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<MockLauncherApiOptions["eventSourceRequests"]> =
    []
  await gotoMockedRoute(page, "/event-sources?view=list", {
    eventSourceBulkFailureIDs: [smokeEventSourceIDs.channel],
    eventSourceRequests: requests,
  })

  const listURL = new URL(page.url())
  expect(listURL.pathname).toBe("/event-sources")
  expect(listURL.searchParams.get("q")).toBe("ORDER BY name ASC")
  expect(listURL.searchParams.get("view")).toBe("list")
  await expect(
    page.getByRole("heading", { name: "Event sources", exact: true }),
  ).toBeVisible()
  await expect(page.getByText("secondary-inbox", { exact: true })).toHaveCount(
    0,
  )

  const github = page.locator(`[data-item-id="${smokeEventSourceIDs.github}"]`)
  await github.focus()
  await page.keyboard.press("Space")
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await github.click({ button: "right" })
  await expect(
    page.getByRole("menuitem", { name: "Edit source" }),
  ).toBeVisible()
  await expect(
    page.getByRole("menuitem", { name: "Disable source" }),
  ).toBeVisible()
  await page.keyboard.press("Escape")
  await github.focus()
  await page.keyboard.press("Enter")

  await expect(page).toHaveURL(
    new RegExp(`/event-sources/${smokeEventSourceIDs.github}`),
  )
  const detail = page.locator('[data-slot="collection-detail-shell"]')
  await expect(detail).toBeVisible()
  await expect(
    detail.getByText("Webhook configuration", { exact: true }),
  ).toBeVisible()
  await expect(
    detail.getByText("POST /webhooks/events/github-primary"),
  ).toBeVisible()
  await expect(detail.getByText("Configured", { exact: true })).toBeVisible()
  await detail.getByRole("button", { name: "Edit", exact: true }).click()
  await expect(page).toHaveURL(
    new RegExp(`/event-sources/${smokeEventSourceIDs.github}/edit`),
  )
  await expect(page.getByLabel("Connector name")).toHaveValue("github-primary")
  await page.getByRole("button", { name: "Back to source" }).click()
  await page.getByRole("button", { name: "All event sources" }).click()
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Clear selection" }).click()

  const standard = page.locator(
    `[data-item-id="${smokeEventSourceIDs.standard}"]`,
  )
  const channel = page.locator(
    `[data-item-id="${smokeEventSourceIDs.channel}"]`,
  )
  await standard.focus()
  await page.keyboard.press("Space")
  await channel.click({ modifiers: ["Control"] })
  await expect(page.getByText("2 selected", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Delete" }).click()
  const confirmation = page.getByRole("alertdialog", {
    name: "Delete 2 selected event sources?",
  })
  await confirmation.getByRole("button", { name: "Delete selected" }).click()
  await expect(standard).toHaveCount(0)
  await expect(channel).toBeVisible()
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()

  const bulkRequest = requests.find(
    (request) =>
      request.method === "POST" &&
      request.path === "/api/event-sources/bulk-delete",
  )
  expect(bulkRequest?.body?.config_revision).toBe("event-source-revision-1")
  expect(new Set(bulkRequest?.body?.ids as string[])).toEqual(
    new Set([smokeEventSourceIDs.standard, smokeEventSourceIDs.channel]),
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("event-source creation choices and global settings use scoped revision fences", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<MockLauncherApiOptions["eventSourceRequests"]> =
    []
  await gotoMockedRoute(page, "/event-sources/new", {
    eventSourceRequests: requests,
  })

  await expect(
    page.getByRole("heading", { name: "Add event source", exact: true }),
  ).toBeVisible()
  const sourceChoice = page.getByRole("combobox", { name: "Create from" })
  await sourceChoice.click()
  await expect(
    page.getByRole("option", {
      name: "Delta Chat adapter — secondary-inbox",
    }),
  ).toBeVisible()
  await expect(page.getByRole("option", { name: /primary-inbox/ })).toHaveCount(
    0,
  )
  await page
    .getByRole("option", { name: "Delta Chat adapter — secondary-inbox" })
    .click()
  await expect(
    page.getByText("Delta Chat email adapter", { exact: true }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Save", exact: true }).click()

  await expect
    .poll(() =>
      requests.find(
        (request) =>
          request.method === "POST" && request.path === "/api/event-sources",
      ),
    )
    .toBeTruthy()
  const createRequest = requests.find(
    (request) =>
      request.method === "POST" && request.path === "/api/event-sources",
  )
  expect(createRequest?.body).toEqual({
    expected_config_revision: "event-source-revision-1",
    event_source: {
      kind: "channel",
      name: "secondary-inbox",
      enabled: false,
      source: "email",
      mode: "mirror",
      allow_unverified_email: false,
    },
  })
  await expect(page).toHaveURL(/\/event-sources\/[^/]+(?:\?|$)/)
  await expect(
    page
      .getByText("secondary-inbox", { exact: true })
      .filter({ visible: true })
      .first(),
  ).toBeVisible()

  await page.goto("/event-sources/settings")
  await expect(
    page.getByRole("heading", { name: "Event source settings", exact: true }),
  ).toBeVisible()
  await page.getByLabel("Retention days").fill("45")
  await page.getByRole("button", { name: "Save", exact: true }).click()
  await expect
    .poll(() =>
      requests.find(
        (request) =>
          request.method === "PUT" &&
          request.path === "/api/event-source-settings",
      ),
    )
    .toBeTruthy()
  const settingsRequest = requests.find(
    (request) =>
      request.method === "PUT" && request.path === "/api/event-source-settings",
  )
  expect(settingsRequest?.body).toEqual({
    expected_config_revision: "event-source-revision-2",
    event_source_settings: {
      enabled: true,
      database_path: "eventing/events.db",
      retention_days: 45,
      max_payload_bytes: 1_048_576,
      redact_fields: ["tenant_secret", "deployment_token"],
    },
  })
  await page
    .locator("[data-sonner-toaster]")
    .evaluateAll((toasters) => toasters.forEach((toast) => toast.remove()))
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("index-addressed account router URLs use the not-found boundary", async ({
  page,
}) => {
  for (const routePath of [
    "/accounts/account-router/0",
    "/accounts/account-router/new",
  ] as const) {
    await gotoMockedRoute(page, routePath)
    expect(new URL(page.url()).pathname).toBe(routePath)
    await expect(page.locator('[data-slot="collection-shell"]')).toHaveCount(0)
    await expect(
      page.locator('[data-slot="collection-detail-shell"]'),
    ).toHaveCount(0)
  }
})

test("account router editor supports block fallback graph editing", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/accounts/routers/new", {
    oauthProviders: [
      {
        provider: "openai",
        credential_id: "openai",
        display_name: "OpenAI",
        methods: ["browser", "device_code", "token"],
        logged_in: true,
        status: "connected",
        auth_method: "oauth",
        account_id: "acct-primary",
        credentials: [
          {
            provider: "openai",
            credential_id: "openai",
            display_name: "OpenAI",
            methods: ["browser", "device_code", "token"],
            logged_in: true,
            status: "connected",
            auth_method: "oauth",
            account_id: "acct-primary",
          },
          {
            provider: "openai",
            credential_id: "openai:backup",
            display_name: "OpenAI",
            methods: ["browser", "device_code", "token"],
            logged_in: true,
            status: "connected",
            auth_method: "oauth",
            account_id: "acct-backup",
          },
          {
            provider: "openai",
            credential_id: "openai:empty",
            display_name: "OpenAI",
            methods: ["browser", "device_code", "token"],
            logged_in: true,
            status: "connected",
            auth_method: "oauth",
            account_id: "acct-empty",
          },
        ],
      },
    ],
  })
  await expect(page).toHaveURL(/\/accounts\/routers\/new(?:\?|$)/)
  await expect(
    page.getByRole("heading", { name: "Create Account Router" }),
  ).toBeVisible()
  await expect(
    page.getByText("Add an account or load balancer block to start."),
  ).toHaveCount(1)
  await expect(page.getByText("No accounts connected.")).toBeVisible()
  await expect(page.getByText("Entry Block")).toHaveCount(0)

  await page.getByRole("button", { name: "Add Account" }).click()
  const accountDialog = page.getByRole("dialog", { name: "account-1" })
  await expect(accountDialog).toBeVisible()
  if ((page.viewportSize()?.width ?? 0) >= 700) {
    const dialogBox = await accountDialog.boundingBox()
    const viewport = page.viewportSize()
    expect(dialogBox).not.toBeNull()
    expect(viewport).not.toBeNull()
    expect(dialogBox!.height).toBeLessThan(viewport!.height * 0.85)
    expect(dialogBox!.width).toBeLessThan(viewport!.width * 0.8)
  }
  await page.getByRole("combobox", { name: "Account" }).click()
  await page.getByRole("option", { name: "OpenAI: acct-primary" }).click()
  await accountDialog.getByRole("button", { name: "Close" }).last().click()
  await expect(accountDialog).toBeHidden()

  await page.getByRole("button", { name: "Add Load Balancer" }).click()
  const loadBalancerDialog = page.getByRole("dialog", {
    name: "load-balancer-1",
  })
  await expect(loadBalancerDialog).toBeVisible()
  await page.getByRole("button", { name: "OpenAI: acct-backup" }).click()
  await page.getByRole("button", { name: "OpenAI: acct-empty" }).click()
  await loadBalancerDialog.getByRole("button", { name: "Close" }).last().click()
  await expect(loadBalancerDialog).toBeHidden()
  await page.getByRole("button", { name: "Add Branch" }).click()
  const branchDialog = page.getByRole("dialog", { name: "branch-1" })
  await expect(branchDialog).toBeVisible()
  await expect(page.getByText("Branch Condition")).toBeVisible()
  await expect(
    page.getByText("Type one condition. Use accounts:provider:name.metric"),
  ).toBeVisible()
  await expect(
    page.getByText("Start typing to autocomplete account metrics"),
  ).toBeVisible()
  await expect(
    page.getByText("Math functions: add, subtract, multiply"),
  ).toBeVisible()
  const branchCondition = page.getByRole("combobox", {
    name: "Branch Condition",
  })
  await expect(branchCondition).toHaveValue(
    "accounts:openai:acct-primary.rpm > 0",
  )
  await branchCondition.fill("accounts:openai:acct-primary.")
  await expect(
    page.getByText("Use syntax like accounts:openai:work.limit_pressure >= 80"),
  ).toBeVisible()
  await expect(
    page.getByRole("listbox", { name: "Branch Condition Suggestions" }),
  ).toBeVisible()
  const limitPressureMetric = "accounts:openai:acct-primary.limit_pressure"
  await expect(
    page.getByRole("option", {
      name: /accounts:openai:acct-primary\.rpm metric/,
    }),
  ).toBeVisible()
  await expect(
    page.getByRole("option", { name: />= 0\.8 example/ }),
  ).toHaveCount(0)
  await page
    .getByRole("option", {
      name: new RegExp(`${limitPressureMetric.replaceAll(".", "\\.")}.*metric`),
    })
    .click()
  await expect(branchCondition).toHaveValue(limitPressureMetric)
  await branchCondition.press("End")
  await branchCondition.press("Space")
  await expect(page.getByRole("option", { name: /> comparison/ })).toBeVisible()
  await expect(
    page.getByRole("option", { name: /limit_pressure metric/ }),
  ).toHaveCount(0)
  await page.getByRole("option", { name: /> comparison/ }).click()
  await expect(branchCondition).toHaveValue(`${limitPressureMetric} > `)
  const textCondition = `multiply(${limitPressureMetric}, 100) >= 75`
  await branchCondition.press("Control+A")
  await branchCondition.press("Backspace")
  await branchCondition.fill(textCondition)
  await expect(branchCondition).toHaveValue(textCondition)
  await expect(
    page.getByText("Use syntax like accounts:openai:work.limit_pressure >= 80"),
  ).toHaveCount(0)
  await expect(page.getByText("Left Value")).toHaveCount(0)
  await expect(page.getByText("Right Value")).toHaveCount(0)
  await expect(page.getByText("Operand", { exact: true })).toHaveCount(0)
  await expect(page.getByText("Threshold", { exact: true })).toHaveCount(0)
  await expect(page.getByText("When True")).toBeVisible()
  await expect(page.getByText("When False")).toBeVisible()
  await branchDialog.getByRole("button", { name: "Close" }).last().click()
  await expect(branchDialog).toBeHidden()
  await page.getByRole("button", { name: "Raw JSON" }).click()
  const rawRouterConfig = JSON.parse(
    await page.getByRole("textbox", { name: "Raw JSON" }).inputValue(),
  )
  const rawBranch = rawRouterConfig.blocks.find(
    (block: { id: string }) => block.id === "branch-1",
  )
  expect(rawBranch.condition).toMatchObject({
    operator: "gte",
    left: {
      op: "multiply",
      left: {
        account: "credential:openai",
        metric: "limit_pressure",
      },
      right: {
        value: 100,
      },
    },
    right: {
      value: 75,
    },
  })
  await page.getByRole("button", { name: "UI Editor" }).click()

  await page.getByRole("button", { name: "Edit block account-1" }).click()
  const reopenedAccountDialog = page.getByRole("dialog", { name: "account-1" })
  await expect(reopenedAccountDialog).toBeVisible()
  await page.getByRole("combobox", { name: "Fallback Connection" }).click()
  await page.getByRole("option", { name: "load-balancer-1" }).click()
  await reopenedAccountDialog
    .getByRole("button", { name: "Close" })
    .last()
    .click()
  await expect(reopenedAccountDialog).toBeHidden()

  await expect(page.getByText("Fallback -> load-balancer-1")).toBeVisible()
  await page.getByRole("button", { name: "Pile fallback chain" }).click()
  await page.getByRole("combobox", { name: "Scale" }).click()
  await page.getByRole("option", { name: "125%" }).click()
  await expect(page.getByRole("combobox", { name: "Scale" })).toContainText(
    "125%",
  )

  if ((page.viewportSize()?.width ?? 0) >= 700) {
    const canvas = page.locator('svg[aria-label="Router Diagram"]')
    const world = page.locator('svg[aria-label="Router Diagram"] > g')
    const canvasBox = await canvas.boundingBox()
    expect(canvasBox).not.toBeNull()

    const loadBalancerNode = page.getByRole("button", {
      name: "Edit block load-balancer-1",
    })
    const beforeDragTransform = await loadBalancerNode.evaluate((node) =>
      node.getAttribute("transform"),
    )
    const loadBalancerBox = await loadBalancerNode.boundingBox()
    expect(loadBalancerBox).not.toBeNull()
    await page.mouse.move(loadBalancerBox!.x + 24, loadBalancerBox!.y + 24)
    await page.mouse.down()
    await page.mouse.move(loadBalancerBox!.x + 96, loadBalancerBox!.y + 72)
    await page.mouse.up()
    await expect
      .poll(() =>
        loadBalancerNode.evaluate((node) => node.getAttribute("transform")),
      )
      .not.toBe(beforeDragTransform)

    await canvas.evaluate((element) => {
      element.dispatchEvent(
        new WheelEvent("wheel", {
          bubbles: true,
          cancelable: true,
          deltaY: -240,
          shiftKey: true,
        }),
      )
    })
    await expect(page.getByRole("combobox", { name: "Scale" })).toContainText(
      "150%",
    )

    const beforePanTransform = await world.evaluate((node) =>
      node.getAttribute("transform"),
    )
    await page.mouse.move(
      canvasBox!.x + canvasBox!.width - 36,
      canvasBox!.y + 36,
    )
    await page.mouse.down()
    await page.mouse.move(
      canvasBox!.x + canvasBox!.width - 116,
      canvasBox!.y + 92,
    )
    await page.mouse.up()
    await expect
      .poll(() => world.evaluate((node) => node.getAttribute("transform")))
      .not.toBe(beforePanTransform)
  }

  await expect(page.getByRole("button", { name: "Raw JSON" })).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("skills use routed collection, detail, import, and removable-only bulk actions", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/skills?view=list")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  const listURL = new URL(page.url())
  expect(listURL.searchParams.get("q")).toBe("ORDER BY name ASC")
  expect(listURL.searchParams.get("view")).toBe("list")
  const workspaceSkill = page.locator(
    `[data-item-id="${smokeSkillToolIDs.skill}"]`,
  )
  const builtinSkill = page.locator('[data-item-id="c2tpbGwAY29kZS1yZXZpZXc"]')
  await expect(workspaceSkill).toBeVisible()
  await expect(builtinSkill).toBeVisible()

  await workspaceSkill.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(`/agent/skills/${smokeSkillToolIDs.skill}\\?.*view=list`),
  )
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "review-helper" }).first(),
  ).toBeVisible()
  await expect(page.getByText("Review code changes").first()).toBeVisible()

  await page.goBack()
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await workspaceSkill.click()
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await builtinSkill.click({ modifiers: ["Control"] })
  await expect(page.getByText("1 selected", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Delete" }).click()
  const confirmation = page.getByRole("alertdialog", {
    name: "Remove 1 selected skill?",
  })
  await confirmation.getByRole("button", { name: "Remove selected" }).click()
  await expect(workspaceSkill).toHaveCount(0)
  await expect(builtinSkill).toBeVisible()

  await page.getByRole("button", { name: "Import skill" }).click()
  await expect(page).toHaveURL(/\/agent\/skills\/new(?:\?|$)/)
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "Import skill" }),
  ).toBeVisible()
  const marketplace = page.getByRole("button", { name: "Browse marketplace" })
  await expect(marketplace).toBeVisible()
  await expect(page.getByRole("dialog")).toHaveCount(0)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("tools use routed collection, detail, editor, state, and adaptation", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/tools?view=list")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  const listURL = new URL(page.url())
  expect(listURL.searchParams.get("q")).toBe("ORDER BY category ASC, name ASC")
  expect(listURL.searchParams.get("view")).toBe("list")
  const webSearchTool = page.locator(
    `[data-item-id="${smokeSkillToolIDs.tool}"]`,
  )
  await expect(webSearchTool).toBeVisible()
  await webSearchTool.focus()
  await page.keyboard.press("Enter")
  await expect(page).toHaveURL(
    new RegExp(`/agent/tools/${smokeSkillToolIDs.tool}\\?.*view=list`),
  )
  const detail = page.locator('[data-slot="collection-detail-shell"]')
  await expect(detail).toBeVisible()
  await expect(page.getByRole("heading", { name: "web_search" })).toBeVisible()
  const stateRequest = page.waitForRequest((request) => {
    const url = new URL(request.url())
    return (
      request.method() === "PUT" &&
      url.pathname === `/api/tools/${smokeSkillToolIDs.tool}/state`
    )
  })
  await detail.getByRole("button", { name: "Disable" }).click()
  expect((await stateRequest).postDataJSON()).toEqual({ enabled: false })
  await detail.getByRole("button", { name: "Configure" }).click()
  await expect(page).toHaveURL(
    new RegExp(`/agent/tools/${smokeSkillToolIDs.tool}/edit`),
  )
  await expect(
    page.getByRole("heading", { name: "Configure web_search" }),
  ).toBeVisible()
  await expect(
    page.getByText("Primary Provider", { exact: true }),
  ).toBeVisible()

  await page.goto("/agent/tools?tab=adaptation")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "Tool adaptation" }),
  ).toHaveCount(0)
  await expect
    .poll(() => new URL(page.url()).searchParams.get("tab"))
    .toBeNull()
  const cutoverURL = new URL(page.url())
  expect(cutoverURL.pathname).toBe("/agent/tools")
  expect(cutoverURL.searchParams.get("q")).toBe(
    "ORDER BY category ASC, name ASC",
  )
  expect(cutoverURL.searchParams.get("view")).toBeNull()
  await expect(page.getByRole("button", { name: "List view" })).toHaveAttribute(
    "aria-pressed",
    "true",
  )

  await page.getByRole("button", { name: "Adaptation settings" }).click()
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe("/agent/tools/settings/adaptation")
  await expect(
    page.getByRole("heading", { name: "Tool adaptation" }),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("web-search provider settings expand without overflow", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, `/agent/tools/${smokeSkillToolIDs.tool}/edit`)
  await expect(
    page.getByRole("heading", { name: "Configure web_search" }),
  ).toBeVisible()
  await expect(
    page.getByText("Primary Provider", { exact: true }),
  ).toBeVisible()

  await page.getByRole("button", { name: /OpenAI/ }).click()
  await expect(page.getByText("Max Results")).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("tool adaptation profile override dialog fits the viewport", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/tools/settings/adaptation")
  await expect(
    page.getByRole("heading", { name: "Tool adaptation" }),
  ).toBeVisible()
  await expect(page.getByText("Profiles", { exact: true })).toBeVisible()
  await page
    .getByRole("button", { name: "Add override for openai / gpt-4o-mini" })
    .click()

  await expect(
    page.getByRole("dialog", { name: "Add profile override" }),
  ).toBeVisible()
  await expectElementFitsViewport(page, '[role="dialog"]', "profile override")
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("tool adaptation worst-state row fits a narrow mobile viewport", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "mobile", "mobile-only layout regression")
  await page.setViewportSize({ width: 320, height: 720 })
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/tools/settings/adaptation")
  const unavailableProbe = page.getByRole("button", {
    name: /^Probe unavailable for /,
  })
  await expect(unavailableProbe).toBeVisible()
  await expect(unavailableProbe).toHaveAttribute("aria-describedby", /.+/)
  await expect(
    page.getByText(
      "No configured credentials or endpoint are available for this profile.",
    ),
  ).toBeVisible()

  const mobileMetrics = page.getByTestId(
    "adaptation-profile-mobile-metrics-openai/very-long-model-name-with-reasoning-context-and-tool-capabilities",
  )
  await expect(mobileMetrics).toBeVisible()
  await expect(mobileMetrics.getByText("Surface")).toBeVisible()
  await expect(mobileMetrics.getByText("simple")).toBeVisible()
  await expect(mobileMetrics.getByText("Cache")).toBeVisible()
  await expect(mobileMetrics.getByText("Flexible")).toBeVisible()

  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("git workspace inventory and history use canonical routed collections", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const publicGitWorkspaceBodies: string[] = []
  page.on("response", async (response) => {
    const url = new URL(response.url())
    if (
      url.pathname.startsWith("/api/git-workspaces") &&
      response.request().method() === "GET"
    ) {
      publicGitWorkspaceBodies.push(await response.text())
    }
  })
  await gotoMockedRoute(page, "/agent/git-workspaces?tab=history&view=grid")

  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("ORDER BY updated DESC")
  const normalized = new URL(page.url())
  expect(normalized.searchParams.get("tab")).toBeNull()
  expect(normalized.searchParams.get("view")).toBe("grid")
  await expect(page.locator("body")).not.toContainText(
    gitWorkspacePrivateCanary,
  )

  const workspace = page.locator(
    `[data-item-id="${smokeGitWorkspaceIDs.primary}"]`,
  )
  await expect(workspace).toBeVisible()
  await workspace.focus()
  await page.keyboard.press("Enter")
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe(`/agent/git-workspaces/${smokeGitWorkspaceIDs.primary}`)
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expect(page.getByText("Checkout", { exact: true })).toBeVisible()

  await page.goBack()
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe("/agent/git-workspaces")
  const restored = new URL(page.url())
  expect(restored.searchParams.get("q")).toBe("ORDER BY updated DESC")
  expect(restored.searchParams.get("view")).toBe("grid")
  await expect(workspace).toBeVisible()

  await page.goto("/agent/git-workspaces/history")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("ORDER BY time DESC")
  await expect(page.getByRole("button", { name: "Grid view" })).toHaveCount(0)
  await expect(
    page.getByText("Cleaned ignored", { exact: true }).first(),
  ).toBeVisible()
  await expect(page.locator("body")).not.toContainText(
    gitWorkspacePrivateCanary,
  )
  expect(publicGitWorkspaceBodies.join("\n")).not.toContain(
    gitWorkspacePrivateCanary,
  )
  expect(publicGitWorkspaceBodies.join("\n")).not.toContain("session_key")
  expect(publicGitWorkspaceBodies.join("\n")).not.toContain("reservation")
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("git workspace maintenance and settings require explicit actions", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  const requests: NonNullable<MockLauncherApiOptions["gitWorkspaceRequests"]> =
    []
  await gotoMockedRoute(
    page,
    `/agent/git-workspaces/${smokeGitWorkspaceIDs.primary}`,
    { gitWorkspaceRequests: requests },
  )

  await page.getByRole("button", { name: "Clean ignored files" }).click()
  const cleanDialog = page.getByRole("alertdialog", {
    name: "Clean ignored files?",
  })
  await expect(cleanDialog).toBeVisible()
  expect(requests).toEqual([])
  await cleanDialog.getByRole("button", { name: "Clean", exact: true }).click()
  await expect.poll(() => requests.length).toBe(1)
  expect(requests[0]).toEqual({
    method: "POST",
    path: "/api/git-workspaces/cleanup",
    body: { workspace_id: smokeGitWorkspaceIDs.primary },
  })

  await page.getByRole("button", { name: "Drop workspace" }).click()
  const dropDialog = page.getByRole("alertdialog", {
    name: "Drop local checkout?",
  })
  await expect(dropDialog).toBeVisible()
  expect(requests).toHaveLength(1)
  await dropDialog.getByRole("button", { name: "Drop", exact: true }).click()
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe("/agent/git-workspaces")
  expect(requests.at(-1)).toEqual({
    method: "DELETE",
    path: `/api/git-workspaces/${smokeGitWorkspaceIDs.primary}`,
    body: null,
  })

  await page.goto(`/agent/git-workspaces/${smokeGitWorkspaceIDs.locked}`)
  await expect(page.getByText("Locked", { exact: true }).first()).toBeVisible()
  await expect(
    page.getByRole("button", { name: "Clean ignored files" }),
  ).toBeDisabled()
  await expect(
    page.getByRole("button", { name: "Drop workspace" }),
  ).toBeDisabled()

  await page.goto("/agent/git-workspaces/settings")
  const maximum = page.getByLabel("Maximum total size (bytes)")
  await expect(maximum).toHaveValue("0")
  await maximum.fill("10737418240")
  await page.getByRole("button", { name: "Save settings" }).click()
  await expect.poll(() => requests.length).toBe(3)
  expect(requests[2]).toEqual({
    method: "PUT",
    path: "/api/git-workspaces/settings",
    body: {
      expected_config_revision: "sha256:git-workspace-settings-1",
      settings: {
        max_total_size_bytes: 10_737_418_240,
        ignored_cleanup_delay_seconds: 0,
        drop_delay_seconds: 0,
      },
    },
  })
  await page
    .locator("[data-sonner-toaster]")
    .evaluateAll((toasters) => toasters.forEach((toast) => toast.remove()))
  await expect(page.locator("body")).not.toContainText(
    gitWorkspacePrivateCanary,
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow definitions and runs use canonical routed collections", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(
    page,
    "/agent/workflows?mode=operate&workflow=workflows%2Fsummarize-text.yml&run=wr_test&view=grid",
  )

  const definitions = page.locator('[data-slot="collection-shell"]')
  await expect(definitions).toBeVisible()
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("ORDER BY ref ASC")
  const normalized = new URL(page.url())
  expect(normalized.searchParams.get("mode")).toBeNull()
  expect(normalized.searchParams.get("workflow")).toBeNull()
  expect(normalized.searchParams.get("run")).toBeNull()
  expect(normalized.searchParams.get("view")).toBe("grid")

  const summary = page.locator(
    `[data-item-id="${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}"]`,
  )
  await expect(summary).toBeVisible()
  await summary.focus()
  await page.keyboard.press("Enter")
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe(
      `/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}`,
    )
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()

  await page.goBack()
  await expect.poll(() => new URL(page.url()).pathname).toBe("/agent/workflows")
  const restored = new URL(page.url())
  expect(restored.searchParams.get("q")).toBe("ORDER BY ref ASC")
  expect(restored.searchParams.get("view")).toBe("grid")
  await expect(summary).toBeVisible()

  await page.goto("/agent/workflows/settings")
  await expect(
    page.getByRole("heading", { name: "Workflow settings" }),
  ).toBeVisible()
  await expect(page.getByText("Runtime policy", { exact: true })).toBeVisible()

  await page.goto("/agent/workflows/runs")
  await expect(page.locator('[data-slot="collection-shell"]')).toBeVisible()
  await expect
    .poll(() => new URL(page.url()).searchParams.get("q"))
    .toBe("ORDER BY created DESC")
  await expect(page.getByRole("button", { name: "Grid view" })).toHaveCount(0)
  const run = page.locator('[data-item-id="wr_test"]')
  await expect(run).toBeVisible()
  await run.focus()
  await page.keyboard.press("Enter")
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe("/agent/workflows/runs/wr_test")
  await expect(
    page.locator('[data-slot="collection-detail-shell"]'),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow authoring routes preserve the singleton active draft", async ({
  page,
}) => {
  const errors = collectPageErrors(page)
  await gotoMockedRoute(page, "/agent/workflows/new")
  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets")
  await page.getByRole("button", { name: "Start with AI" }).click()
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()

  await page.goto(
    `/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}/edit`,
  )
  const conflict = page.getByRole("alert")
  await expect(
    conflict.getByRole("heading", { name: "Active workflow draft conflict" }),
  ).toBeVisible()
  await expect(
    conflict.getByRole("button", { name: "Open active draft" }),
  ).toBeVisible()
  await expect(
    conflict.getByRole("button", { name: "Discard draft" }),
  ).toBeVisible()
  await conflict.getByRole("button", { name: "Open active draft" }).click()
  await expect
    .poll(() => new URL(page.url()).pathname)
    .toBe("/agent/workflows/new")
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow run detail tolerates null persisted collections", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await page.addInitScript(() => {
    window.localStorage.setItem(
      "picoclaw-tour-state",
      JSON.stringify({ currentStep: "completed", isActive: false }),
    )
  })
  await mockLauncherApis(page, { nullableWorkflowPayloads: true })
  await page.goto("/agent/workflows/runs/wr_nulls")

  await expect(page.getByRole("banner")).toBeVisible()
  await expect(page.locator("main")).toBeVisible()
  await expect(page.getByText("wr_nulls").first()).toBeVisible()
  await expect(page.getByText("No events")).toBeVisible()
  await expect(page.getByText("No graph")).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow detail shows only the sanitized published definition inspection", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "desktop",
    "desktop workflow inspection coverage",
  )
  const errors = collectPageErrors(page)
  const inspectionRequests: NonNullable<
    MockLauncherApiOptions["workflowInspectionRequests"]
  > = []

  await gotoMockedRoute(
    page,
    `/agent/workflows/${smokeWorkflowDefinitionIDs["workflows/summarize-text.yml"]}`,
    { workflowInspectionRequests: inspectionRequests },
  )

  const inspectionTrigger = page.getByRole("button", {
    name: "Published definition: workflows/summarize-text.yml",
  })
  await expect(inspectionTrigger).toBeVisible()
  const inspection = inspectionTrigger.locator("..").locator("..")
  await expect(inspection).toContainText("Inspected")
  await expect(inspection).toContainText("workflows/summarize-text.yml")
  await expect(inspection).toContainText("issues.opened")
  await expect(inspection).toContainText("review")
  await expect(inspection).toContainText("main")
  await expect(inspection).toContainText("Possible effects")
  await expect(inspection).toContainText("model or delegated action possible")

  expect(inspectionRequests.length).toBeGreaterThan(0)
  for (const request of inspectionRequests) {
    expect(request).toEqual({
      method: "POST",
      path: "/api/workflows/definitions/inspect",
      body: { ref: "workflows/summarize-text.yml" },
    })
  }
  await expect(inspection).not.toContainText(workflowInspectionSecretCanary)
  await expect(inspection).not.toContainText(workflowInspectionRawYAMLCanary)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow authoring capability catalog is lazy and searchable", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "desktop",
    "desktop workflow capability coverage",
  )
  const errors = collectPageErrors(page)
  const capabilityRequests: NonNullable<
    MockLauncherApiOptions["workflowCapabilityRequests"]
  > = []

  await gotoMockedRoute(page, "/agent/workflows/new", {
    workflowCapabilityRequests: capabilityRequests,
  })
  const capabilitiesButton = page.getByRole("button", {
    name: "Workflow capabilities",
  })
  await expect(capabilitiesButton).toBeVisible()
  expect(capabilityRequests).toEqual([])

  await capabilitiesButton.click()
  const dialog = page.getByRole("dialog", {
    name: "Workflow capabilities",
  })
  await expect(dialog).toBeVisible()
  await expect(
    dialog.getByRole("region", { name: "Workflow capability results" }),
  ).toHaveAttribute("tabindex", "0")
  await expect(dialog.getByText("agent/main")).toBeVisible()
  await expect(dialog.getByText("tool/message")).toBeVisible()
  await expect(dialog.getByText("mcp/github/create_issue")).toBeVisible()
  await expect(dialog.getByText("Additional property values")).toBeVisible()
  await expect(
    dialog.getByRole("button", { name: "Copy agent/reviewer" }),
  ).toBeDisabled()
  expect(capabilityRequests).toEqual([
    {
      method: "GET",
      path: "/api/workflows/authoring/capabilities",
    },
  ])

  await dialog
    .getByRole("searchbox", { name: "Search capabilities" })
    .fill("create_issue")
  await expect(dialog.getByText("mcp/github/create_issue")).toBeVisible()
  await expect(dialog.getByText("agent/main")).toHaveCount(0)
  await dialog.getByRole("button", { name: "MCP tools" }).click()
  await expect(
    dialog.getByText(
      "No capabilities match the current search and category filters.",
    ),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  await dialog.getByRole("button", { name: "Close" }).click()

  await expect(capabilitiesButton).toBeVisible()
  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets")
  await page.getByRole("button", { name: "Start with AI" }).click()
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()
  await expect(capabilitiesButton).toBeVisible()
  await expect(capabilitiesButton).toBeEnabled()

  await expectNoHorizontalOverflow(page)
  expect(errors).toEqual([])
})

test("workflow capability catalog fits and wraps exact targets at 320px", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "mobile",
    "mobile workflow capability coverage",
  )
  await page.setViewportSize({ width: 320, height: 720 })
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/workflows/new")
  await page.getByRole("button", { name: "Workflow capabilities" }).click()
  const dialog = page.getByRole("dialog", {
    name: "Workflow capabilities",
  })
  await expect(dialog).toBeVisible()
  await dialog
    .getByRole("searchbox", { name: "Search capabilities" })
    .fill(workflowCapabilityLongToolName)
  await expect(
    dialog.getByText(`tool/${workflowCapabilityLongToolName}`),
  ).toBeVisible()
  await expectElementFitsViewport(
    page,
    '[role="dialog"]',
    "workflow capability catalog",
  )
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("narrow mobile lazily opens a sanitized built-in workflow inspection", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "mobile",
    "mobile workflow inspection coverage",
  )
  await page.setViewportSize({ width: 320, height: 720 })
  const errors = collectPageErrors(page)
  const inspectionRequests: NonNullable<
    MockLauncherApiOptions["workflowInspectionRequests"]
  > = []

  await gotoMockedRoute(page, "/agent/workflows/new", {
    workflowInspectionRequests: inspectionRequests,
  })
  const template = page.getByRole("article", {
    name: "Code review template",
  })
  await expect(template).toBeVisible()
  expect(inspectionRequests).toEqual([])

  await template
    .getByRole("button", { name: "Built-in definition: code-review" })
    .click()
  await expect(template.getByText("Inspected")).toBeVisible()
  await expect(template).toContainText("code-review")
  await expect(template).toContainText("issues.opened")
  await expect(template).toContainText("main")

  expect(inspectionRequests.length).toBeGreaterThan(0)
  for (const request of inspectionRequests) {
    expect(request).toEqual({
      method: "GET",
      path: "/api/workflows/templates/code-review/inspect",
      body: null,
    })
  }
  await expect(template).not.toContainText(workflowInspectionSecretCanary)
  await expect(template).not.toContainText(workflowInspectionRawYAMLCanary)
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("mobile workflow trigger simulator reviews one explicit redacted scenario", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "mobile",
    "mobile workflow trigger simulator coverage",
  )
  const errors = collectPageErrors(page)
  const simulationRequests: NonNullable<
    MockLauncherApiOptions["workflowTriggerSimulationRequests"]
  > = []
  const executionRequests: NonNullable<
    MockLauncherApiOptions["workflowTriggerExecutionRequests"]
  > = []
  const payloadRequests: string[] = []
  const secretCanary = "mobile-trigger-secret-value-must-not-render"

  await gotoMockedRoute(page, "/agent/workflows/new", {
    workflowDevelopmentYAML: workflowEventDraftYAML,
    workflowTriggerSimulationRequests: simulationRequests,
    workflowTriggerExecutionRequests: executionRequests,
    workflowEventPayloadRequests: payloadRequests,
  })
  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets from calls and durable events")
  const startWithAI = page.getByRole("button", { name: "Start with AI" })
  await expect(startWithAI).toBeEnabled()
  await startWithAI.click()

  await expect(
    page.getByText("Trigger simulator", { exact: true }),
  ).toBeVisible()
  const triggerScenario = page.getByRole("combobox", {
    name: "Trigger scenario",
  })
  await expect(triggerScenario).toHaveValue("")
  await expect(page.getByText(/dashboard will not guess/i)).toBeVisible()
  expect(simulationRequests).toEqual([])
  await expectNoHorizontalOverflow(page)

  await triggerScenario.selectOption("event")
  const durableEvent = page.getByRole("combobox", { name: "Durable event" })
  await durableEvent.selectOption(eventResponse.id)
  await expect
    .poll(() =>
      simulationRequests.some(
        (entry) =>
          (entry.body.trigger as { type?: string }).type === "event" &&
          (entry.body.scenario as { event_id?: string }).event_id ===
            eventResponse.id,
      ),
    )
    .toBe(true)
  await expect(
    page.getByText(/protected payload content stays server-side/i),
  ).toBeVisible()
  expect(payloadRequests).toEqual([])
  await expect(page.locator("body")).not.toContainText(eventPayloadText)

  await triggerScenario.selectOption("workflow_call")
  await page.getByLabel("Inputs JSON").fill('{"ticket":"PIC-mobile-review"}')
  await page
    .getByLabel("Secrets JSON")
    .fill(JSON.stringify({ github_token: secretCanary }))
  await expect
    .poll(() => {
      const latest = simulationRequests.at(-1)?.body
      return (
        (latest?.trigger as { type?: string } | undefined)?.type ===
          "workflow_call" &&
        (latest?.scenario as { secrets?: Record<string, string> } | undefined)
          ?.secrets?.github_token === secretCanary
      )
    })
    .toBe(true)

  const reviewButton = page.getByRole("button", {
    name: "Review & execute",
  })
  await expect(reviewButton).toBeEnabled()
  await reviewButton.click()
  const dialog = page.getByRole("dialog", {
    name: "Review trigger execution",
  })
  await expect(dialog).toBeVisible()
  await expect(dialog).toContainText("Provided secrets")
  await expect(dialog).not.toContainText(secretCanary)
  await expect(dialog).not.toContainText(eventPayloadText)
  await expectElementFitsViewport(
    page,
    '[role="dialog"]',
    "workflow trigger execution review",
  )
  await expectNoHorizontalOverflow(page)

  await dialog
    .getByRole("switch", {
      name: "I reviewed this server simulation and its possible effects",
    })
    .click()
  const confirm = dialog.getByRole("button", {
    name: "Confirm and execute",
  })
  await expect(confirm).toBeEnabled()
  await confirm.evaluate((button: HTMLButtonElement) => {
    button.click()
    button.click()
  })
  await expect.poll(() => executionRequests.length).toBe(1)
  expect(executionRequests).toEqual([
    {
      method: "POST",
      path: "/api/workflows/development/test/execute",
      body: expect.objectContaining({
        trigger: { type: "workflow_call" },
        scenario: expect.objectContaining({
          inputs: { ticket: "PIC-mobile-review" },
          secrets: { github_token: secretCanary },
        }),
        review_token: "review-token:workflow_call",
      }),
    },
  ])
  await expect(dialog).toBeHidden()
  expect(payloadRequests).toEqual([])
  await expect(page.locator("body")).not.toContainText(eventPayloadText)
  await expectNoHorizontalOverflow(page)
  expect(errors).toEqual([])
})

test("workflow event builder routes one exact durable event through server review", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/workflows/new")
  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets")
  await page.getByRole("button", { name: "Start with AI" }).click()
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()

  await page.getByRole("tab", { name: "Builder" }).click()
  await page.getByRole("combobox", { name: "Trigger type" }).click()
  await page
    .getByRole("option", { name: "Durable event · Not configured" })
    .click()
  await page
    .getByRole("switch", { name: "Enable durable event trigger" })
    .click()
  await expect(page.getByText("Deterministic event routing")).toBeVisible()
  await page
    .getByRole("textbox", { name: "Sources", exact: true })
    .fill("github")
  await page
    .getByRole("textbox", { name: "Event types", exact: true })
    .fill("issues.opened")
  await expect(
    page.getByRole("button", { name: "Apply to YAML" }),
  ).toBeEnabled()
  await page.getByRole("button", { name: "Apply to YAML" }).click()

  const triggerScenario = page.getByRole("combobox", {
    name: "Trigger scenario",
  })
  await expect(triggerScenario).toHaveValue("")
  await triggerScenario.selectOption("event")
  const eventPicker = page.getByRole("combobox", { name: "Durable event" })
  await expect(eventPicker).toBeVisible()
  await expect(page.getByLabel("Secrets JSON")).toHaveCount(0)
  const reviewButton = page.getByRole("button", {
    name: "Review & execute",
  })
  await expect(reviewButton).toBeDisabled()
  await eventPicker.selectOption(eventResponse.id)
  await expect(
    page.getByText(/server-reviewed execution token is ready/i),
  ).toBeVisible()
  await expect(reviewButton).toBeEnabled()
  await expect(page.locator("body")).not.toContainText(eventPayloadText)

  await reviewButton.click()
  const eventReview = page.getByRole("dialog", {
    name: "Review trigger execution",
  })
  await expect(eventReview).toContainText("Event context")
  await expect(eventReview).toContainText("Yes")
  await expect(eventReview).not.toContainText(eventPayloadText)
  await eventReview
    .getByRole("switch", {
      name: "I reviewed this server simulation and its possible effects",
    })
    .click()
  await eventReview.getByRole("button", { name: "Confirm and execute" }).click()
  await expect(page.getByText("wr_draft", { exact: true })).toBeVisible()

  await expect(page.locator("body")).not.toContainText(eventPayloadText)
  await expectNoHorizontalOverflow(page)
  expect(errors).toEqual([])
})

test("workflow jobs builder applies a surgical action field edit", async ({
  page,
}, testInfo) => {
  if (testInfo.project.name === "mobile") {
    await page.setViewportSize({ width: 320, height: 720 })
  }
  const errors = collectPageErrors(page)
  const jobRequests: NonNullable<
    MockLauncherApiOptions["workflowJobRequests"]
  > = []

  await gotoMockedRoute(page, "/agent/workflows/new", {
    workflowJobRequests: jobRequests,
  })
  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets")
  await page.getByRole("button", { name: "Start with AI" }).click()
  await page.getByRole("tab", { name: "Builder" }).click()
  await page.getByRole("tab", { name: "Jobs & actions" }).click()

  await expect(page.getByText("Job graph")).toBeVisible()
  await expect(page.getByText("agent/main").first()).toBeVisible()
  const actionSection = page
    .getByRole("heading", { name: "Action 1" })
    .locator("xpath=ancestor::section[1]")
  await actionSection
    .getByRole("combobox", { name: "Display name mutation" })
    .click()
  await page.getByRole("option", { name: "Set value" }).click()
  await actionSection
    .getByRole("textbox", { name: "Display name value" })
    .fill("Summarize ticket")
  await page.getByRole("button", { name: "Apply fields to YAML" }).click()

  await expect
    .poll(
      () =>
        jobRequests.filter(
          (request) =>
            request.path === "/api/workflows/development/jobs/render",
        ).length,
    )
    .toBe(1)
  const renderRequest = jobRequests.find(
    (request) => request.path === "/api/workflows/development/jobs/render",
  )
  expect(renderRequest?.body).toMatchObject({
    operation: {
      type: "step.patch",
      job_id: "triage",
      step_index: 0,
      fields: {
        name: { mode: "set", value: "Summarize ticket" },
      },
    },
  })
  await expect(
    actionSection.getByText("Source: Summarize ticket"),
  ).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("routed workflow authoring supports AI draft, publish, and manual run", async ({
  page,
}) => {
  test.slow()
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/agent/workflows/new", {
    completeDraftViaPolling: true,
  })
  await expect(
    page.getByText("Describe the workflow outcome before starting."),
  ).toBeVisible()
  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets")
  await expect(
    page.getByText(
      "Ready to start. One workflow draft can be active at a time.",
    ),
  ).toBeVisible()
  await page.getByRole("button", { name: "Start with AI" }).click()

  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()
  await expect(page.getByText("Only active draft")).toBeVisible()
  await expect(page.getByText("Publish readiness")).toBeVisible()
  const yamlEditor = page.getByRole("textbox", { name: "Workflow YAML" })
  const localDraftYAML = `${workflowDraftYAML}# local edit\n`
  await yamlEditor.fill(localDraftYAML)
  const developmentRefresh = page.waitForResponse(
    (response) =>
      response.url().endsWith("/api/workflows/development") &&
      response.request().method() === "GET",
  )
  await page.getByRole("button", { name: "Refresh" }).click()
  await developmentRefresh
  await expect(yamlEditor).toHaveValue(localDraftYAML)
  await yamlEditor.fill(workflowDraftYAML)
  const reviewDraft = page.getByRole("button", {
    name: "Review & execute",
  })
  await expect(reviewDraft).toBeEnabled()
  await page.getByLabel("Inputs JSON").fill("{")
  await expect(page.getByText(/Inputs must be valid JSON/)).toBeVisible()
  await expect(reviewDraft).toBeDisabled()
  await expect(page.getByRole("button", { name: "Publish" })).toBeDisabled()
  await expect(
    page.getByText("Run a successful draft test before publishing."),
  ).toBeVisible()

  await page.getByLabel("Inputs JSON").fill('{"ticket":"Printer is offline"}')
  await page.getByLabel("Session").fill("workflow:draft")
  await page
    .getByLabel("Delivery JSON")
    .fill('{"channel":"telegram","chat_id":"support"}')
  await expect(reviewDraft).toBeEnabled()
  await reviewDraft.click()
  await expect(page.getByText("wr_draft", { exact: true })).toHaveCount(0)
  await confirmTriggerExecutionReview(page)
  await expect(page.getByText("wr_draft", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "Publish" })).toBeEnabled()
  await expect(page.getByText("Ready to publish.")).toBeVisible()

  const whitespaceOnlyDraft = `${workflowDraftYAML}\n`
  const whitespaceDependencyRequest = page.waitForRequest((request) => {
    if (
      !request.url().endsWith("/api/workflows/dependencies/check") ||
      request.method() !== "POST"
    ) {
      return false
    }
    const body = request.postDataJSON() as {
      draft?: { yaml?: string }
    }
    return body.draft?.yaml === whitespaceOnlyDraft
  })
  await yamlEditor.fill(whitespaceOnlyDraft)
  await whitespaceDependencyRequest
  await expect(page.getByRole("button", { name: "Publish" })).toBeDisabled()
  await expect(
    page.getByText("Validate the draft again after the latest edits."),
  ).toBeVisible()
  await yamlEditor.fill(workflowDraftYAML)
  await expect(page.getByRole("button", { name: "Publish" })).toBeEnabled()

  await yamlEditor.fill(`${workflowDraftYAML}# readiness is stale\n`)
  await expect(page.getByRole("button", { name: "Publish" })).toBeDisabled()
  await expect(
    page.getByText("Validate the draft again after the latest edits."),
  ).toBeVisible()
  await yamlEditor.fill(workflowDraftYAML)
  await expect(page.getByRole("button", { name: "Publish" })).toBeEnabled()

  await page.reload()
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()
  await expect(page.getByText("wr_draft", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "Publish" })).toBeEnabled()
  await expect(page.getByText("Ready to publish.")).toBeVisible()

  await page.getByRole("button", { name: "Open Run" }).click()
  await expect(page.getByText("draft summary").first()).toBeVisible()
  await expect(page.getByText('"request_id": "req_draft"')).toBeVisible()
  await expect(page.getByText('"result": "draft event"')).toBeVisible()
  await expect(
    page.getByText('"streamed": "draft stream"').first(),
  ).toBeVisible()

  await page.goBack()
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Publish" }).click()

  await expect(page.getByText("Run workflow").first()).toBeVisible()
  await expect(page.locator("#workflow-run-selected-ref")).toHaveText(
    "workflows/support-triage.yml",
  )
  await page.getByRole("button", { name: "Run workflow" }).first().click()
  await expect(page.locator("#workflow-run-input-ticket")).toBeVisible()
  await expect(page.getByText('Input "ticket" is required.')).toBeVisible()
  await expect(
    page.getByRole("button", { name: "Run workflow" }).last(),
  ).toBeDisabled()
  await page.locator("#workflow-run-input-ticket").fill("Printer is offline")
  await expect(
    page.getByRole("button", { name: "Run workflow" }).last(),
  ).toBeEnabled()
  await page.getByRole("button", { name: "Advanced options" }).click()
  await page.locator("#workflow-run-session").fill("workflow:manual")
  await page
    .locator("#workflow-run-delivery")
    .fill('{"channel":"telegram","chat_id":"support"}')
  await page.getByRole("button", { name: "Run workflow" }).last().focus()
  await page.keyboard.press("Enter")

  await expect(page.getByText("wr_manual").first()).toBeVisible()
  await expect(page.getByText("manual summary").first()).toBeVisible()
  await expect(page.getByText('"request_id": "req_manual"')).toBeVisible()
  await expect(page.getByText('"topic_id": "manual-topic"')).toBeVisible()
  await expect(page.getByText('"result": "manual event"')).toBeVisible()
  await expect(
    page.getByText('"streamed": "manual stream"').first(),
  ).toBeVisible()
  await page
    .locator("[data-sonner-toaster]")
    .evaluateAll((toasters) => toasters.forEach((toast) => toast.remove()))
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})

test("workflow editor refreshes async draft status from polling without SSE", async ({
  page,
}) => {
  const errors = collectPageErrors(page)

  await page.addInitScript(() => {
    window.localStorage.setItem(
      "picoclaw-tour-state",
      JSON.stringify({ currentStep: "completed", isActive: false }),
    )
    Object.defineProperty(window, "EventSource", {
      configurable: true,
      value: undefined,
    })
  })
  await mockLauncherApis(page, { completeDraftViaPolling: true })
  await page.goto("/agent/workflows/new")
  await expect(page.getByRole("banner")).toBeVisible()
  await expect(page.locator("main")).toBeVisible()

  await page
    .getByPlaceholder("Describe the workflow outcome")
    .fill("Triage support tickets")
  await page.getByRole("button", { name: "Start with AI" }).click()
  await expect(
    page.getByRole("heading", { name: "Workflow YAML", exact: true }),
  ).toBeVisible()
  await page.getByLabel("Inputs JSON").fill('{"ticket":"Printer is offline"}')
  await page.getByLabel("Session").fill("workflow:draft")
  await page
    .getByLabel("Delivery JSON")
    .fill('{"channel":"telegram","chat_id":"support"}')
  const reviewDraft = page.getByRole("button", {
    name: "Review & execute",
  })
  await expect(reviewDraft).toBeEnabled()
  await reviewDraft.click()
  await confirmTriggerExecutionReview(page)

  await expect(page.getByText("wr_draft", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "Publish" })).toBeEnabled()
  await expect(page.getByText("Ready to publish.")).toBeVisible()
  expect(errors).toEqual([])
})

test("mobile sidebar opens, fits the viewport, and navigates", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "mobile", "mobile-only interaction")
  const errors = collectPageErrors(page)

  await gotoMockedRoute(page, "/")
  await page.getByRole("button", { name: "Toggle Sidebar" }).click()

  const sidebar = page.getByRole("dialog", { name: "Sidebar" })
  await expect(sidebar).toBeVisible()
  await page.waitForTimeout(300)
  await expectElementFitsViewport(
    page,
    '[data-sidebar="sidebar"][data-mobile="true"]',
    "mobile sidebar",
  )
  await sidebar.getByRole("button", { name: "Services" }).click()
  await sidebar.getByRole("link", { name: /Accounts/ }).click()
  await expect(page).toHaveURL(/\/accounts\?q=/)
  expect(new URL(page.url()).searchParams.get("q")).toBe(
    "ORDER BY provider ASC, id ASC",
  )
  await expect(sidebar).toBeHidden()
  await expectNoHorizontalOverflow(page)
  await expectNoSeriousA11yViolations(page)
  expect(errors).toEqual([])
})
