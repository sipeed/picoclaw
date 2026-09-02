import { beforeEach, describe, expect, it, vi } from "vitest"

import { launcherFetch } from "@/api/http"
import {
  type RepositoryReviewProfileConfig,
  createRepositoryReviewAutomation,
  createRepositoryReviewIssueDraft,
  createRepositoryReviewProfile,
  deleteRepositoryReviewAutomation,
  deleteRepositoryReviewAutomationIssue,
  deleteRepositoryReviewProfile,
  generateRepositoryReviewIssues,
  getRepositoryReview,
  getRepositoryReviewAutomation,
  getRepositoryReviewAutomationDetail,
  getRepositoryReviewAutomationFinding,
  getRepositoryReviewAutomationFindings,
  getRepositoryReviewAutomationIssue,
  getRepositoryReviewAutomationOptions,
  getRepositoryReviewAutomationRepositoryFinding,
  getRepositoryReviewAutomationRunFinding,
  getRepositoryReviewCommitOptions,
  getRepositoryReviewFindingHealth,
  getRepositoryReviewFindingsProcessingSource,
  getRepositoryReviewProfile,
  getRepositoryReviewRawSource,
  listRepositoryReviewAutomationFileAttributionsPage,
  listRepositoryReviewAutomationFindingsPage,
  listRepositoryReviewAutomationIssues,
  listRepositoryReviewAutomationIssuesPage,
  listRepositoryReviewAutomationRawFindingsPage,
  listRepositoryReviewAutomationRepositoryFindingsPage,
  listRepositoryReviewAutomations,
  listRepositoryReviewAutomationsPage,
  listRepositoryReviewFindingRawSources,
  listRepositoryReviewFindingsProcessingPage,
  listRepositoryReviewProfiles,
  listRepositoryReviewProfilesPage,
  listRepositoryReviews,
  pauseRepositoryReviewAutomation,
  publishRepositoryReviewIssueDraft,
  purgeRepositoryReviewAutomationHistory,
  repositoryReviewDefaultIssuePrompt,
  restartRepositoryReviewAutomation,
  restartRepositoryReviewHistoricalDeduplication,
  resumeRepositoryReviewAutomation,
  retryRepositoryReviewFindingsProcessingSource,
  retryRepositoryReviewFindingsProcessingSources,
  retryRepositoryReviewHistoricalDeduplication,
  retryRepositoryReviewRawSource,
  retryRepositoryReviewRunFindingStatuses,
  startRepositoryReviewAutomation,
  updateRepositoryReviewAutomation,
  updateRepositoryReviewAutomationIssue,
  updateRepositoryReviewFinding,
  updateRepositoryReviewIssueDraft,
  updateRepositoryReviewProfile,
} from "@/api/repository-reviews"

vi.mock("@/api/http", () => ({ launcherFetch: vi.fn() }))

const mockedLauncherFetch = vi.mocked(launcherFetch)

describe("repository review API", () => {
  beforeEach(() => mockedLauncherFetch.mockReset())

  it("loads legacy occurrence bookmarks from the deprecated run-findings resource", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse({
        automation: { id: "auto/slash", repository: "owner/repo" },
        finding: { id: "rfn/legacy", context_ids: [] },
        contexts: [],
      }),
    )

    await expect(
      getRepositoryReviewAutomationRunFinding("auto/slash", "rfn/legacy"),
    ).resolves.toMatchObject({ finding: { id: "rfn/legacy" } })
    expect(mockedLauncherFetch).toHaveBeenCalledWith(
      "/api/repository-reviews/automations/auto%2Fslash/run-findings/rfn%2Flegacy",
      { signal: undefined },
    )
  })

  it("lists and loads repository review state", async () => {
    mockedLauncherFetch
      .mockResolvedValueOnce(jsonResponse({ repositories: [] }))
      .mockResolvedValueOnce(
        jsonResponse({ id: "rrp_repo", repository: "owner/repo" }),
      )

    await expect(listRepositoryReviews()).resolves.toEqual({ repositories: [] })
    await expect(getRepositoryReview("rrp_repo/slash")).resolves.toMatchObject({
      repository: "owner/repo",
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/rrp_repo%2Fslash",
      { signal: undefined },
    )
  })

  it("requests a bounded finding page", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse({
        id: "rrp_repo",
        repository: "owner/repo",
        finding_offset: 50,
        finding_total: 75,
      }),
    )

    await getRepositoryReview("rrp_repo", undefined, {
      offset: 50,
      limit: 25,
    })

    expect(mockedLauncherFetch).toHaveBeenCalledWith(
      "/api/repository-reviews/rrp_repo?offset=50&limit=25",
      { signal: undefined },
    )
  })

  it("sends version-fenced finding and issue-draft mutations", async () => {
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          repository: { id: "rrp_repo", review_version: 1 },
          finding: {
            id: "rfn_1",
            context_ids: [],
            models: [],
            validation: { status: "confirmed", summary: "confirmed" },
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          repository: { id: "rrp_repo" },
          draft: { id: "rid_1" },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          repository: { id: "rrp_repo" },
          draft: { id: "rid_1" },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          repository: { id: "rrp_repo" },
          draft: { id: "rid_1" },
        }),
      )

    await updateRepositoryReviewFinding("rrp_repo", "rfn/1", {
      status: "dismissed",
      expected_version: 4,
    })
    await createRepositoryReviewIssueDraft("rrp_repo", {
      finding_ids: ["rfn_1", "rfn_2"],
      expected_version: 5,
    })
    await updateRepositoryReviewIssueDraft("rrp_repo", "rid/1", {
      title: "Lost update",
      body: "The write needs a version fence.",
      labels: ["bug", "concurrency"],
      expected_version: 2,
    })
    await publishRepositoryReviewIssueDraft("rrp_repo", "rid/1", {
      expected_version: 3,
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/rrp_repo/findings/rfn%2F1",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({
          status: "dismissed",
          expected_version: 4,
        }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/rrp_repo/issue-drafts",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          finding_ids: ["rfn_1", "rfn_2"],
          expected_version: 5,
        }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/rrp_repo/issue-drafts/rid%2F1",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({
          title: "Lost update",
          body: "The write needs a version fence.",
          labels: ["bug", "concurrency"],
          expected_version: 2,
        }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/rrp_repo/issue-drafts/rid%2F1/publish",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ expected_version: 3 }),
      }),
    )
  })

  it("surfaces structured API errors", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse(
        { code: "repository_review_conflict", message: "Review changed." },
        409,
      ),
    )

    await expect(getRepositoryReview("rrp_repo")).rejects.toMatchObject({
      status: 409,
      code: "repository_review_conflict",
      message: "Review changed.",
    })
  })

  it("normalizes nullable stored collections", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse({
        id: "rrp_repo",
        repository: "owner/repo",
        files: null,
        findings: null,
        contexts: null,
        runs: null,
        issue_drafts: null,
        repository_findings: null,
        mapping_jobs: null,
        validation_jobs: null,
      }),
    )

    await expect(getRepositoryReview("rrp_repo")).resolves.toMatchObject({
      files: {},
      findings: [],
      contexts: [],
      runs: [],
      issue_drafts: [],
      repository_findings: [],
      mapping_jobs: [],
      validation_jobs: [],
    })
  })

  it("loads and normalizes automation options and model-stat maps", async () => {
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          models: [
            {
              alias: "fast",
              resolved_model: "provider/fast",
              provider: "provider",
              available: true,
              price_known: true,
              input_price_per_1m: 0.2,
              output_price_per_1m: 0.8,
            },
            {
              alias: "blocked",
              resolved_model: "agentic-cli/model",
              provider: "agentic-cli",
              available: false,
              blocked_reason: "Agentic CLI models cannot run as reviewers.",
              price_known: false,
            },
          ],
          limits_error: "account telemetry offline",
          accounts: [
            {
              id: "acct",
              models: null,
              entries: [
                {
                  name: "Weekly",
                  window: "weekly",
                  remaining_percent: 75,
                  refreshes_at: "2026-08-21T12:00:00Z",
                },
              ],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automations: [
            {
              id: "auto_1",
              pause_reason: "no_progress",
              reviewer_models: null,
              run_ids: null,
              model_prices: null,
              budget: null,
              usage: null,
              progress: null,
              model_stats: {
                fast: {
                  tokens: {
                    prompt_tokens: 75,
                    completion_tokens: 25,
                    total_tokens: 100,
                  },
                  estimated_cost_usd: 0.01,
                  latency_millis: 250,
                },
              },
              account_limits: [
                {
                  account_id: "acct",
                  name: "Premium",
                  window: "weekly",
                  remaining_percent: 75,
                  resets_at: "0001-01-01T00:00:00Z",
                  checked_at: "2026-08-20T12:00:00Z",
                },
              ],
              scope_plan: {
                commit_sha: "a".repeat(40),
                policy_hash: "b".repeat(64),
                hash: "c".repeat(64),
                summary: "Production files selected",
                counts: null,
                warnings: null,
              },
            },
          ],
        }),
      )

    await expect(getRepositoryReviewAutomationOptions()).resolves.toMatchObject(
      {
        models: [
          { alias: "fast", available: true },
          {
            alias: "blocked",
            available: false,
            blocked_reason: "Agentic CLI models cannot run as reviewers.",
          },
        ],
        limits_error: "account telemetry offline",
        accounts: [
          {
            id: "acct",
            available: false,
            models: [],
            entries: [
              {
                label: "Weekly",
                reset_at: "2026-08-21T12:00:00Z",
              },
            ],
          },
        ],
      },
    )
    await expect(listRepositoryReviewAutomations()).resolves.toMatchObject({
      automations: [
        {
          id: "auto_1",
          pause_reason: "no_progress",
          account_ref: "",
          reviewer_models: [],
          max_parallel_children: 8,
          scope_policy: {
            code_types: ["hotpath-code", "code"],
            include_folders: [],
            exclude_folders: [],
            free_text: "",
          },
          scope_plan: {
            summary: "Production files selected",
            warnings: [],
            counts: {
              total_files: 0,
              code_type_files: 0,
              include_files: 0,
              excluded_files: 0,
              selected_files: 0,
            },
          },
          usage: { total_tokens: 0 },
          budget: { guard_expression: "" },
          progress: { stage: "waiting", scope_frozen: false },
          model_stats: [{ model: "fast", total_tokens: 100, latency_ms: 250 }],
          account_limits: [
            {
              id: "acct",
              entries: [
                {
                  window: "weekly",
                  label: "Premium",
                  remaining_percent: 75,
                  reset_at: undefined,
                },
              ],
            },
          ],
        },
      ],
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automation-options",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations",
      { signal: undefined },
    )
  })

  it("loads a canonical paged review-profile collection", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse({
        profiles: [
          {
            id: "rrpf_one",
            name: "Core review",
            reviewer_model: "reviewer",
            issue_writer_model: null,
          },
        ],
        total: 1,
        next_cursor: "cursor-2",
        canonical_query: "ORDER BY name ASC",
        query_schema: { fields: [] },
      }),
    )

    await expect(
      listRepositoryReviewProfilesPage({
        query: "ORDER BY name ASC",
        cursor: "cursor-1",
        limit: 25,
      }),
    ).resolves.toMatchObject({
      profiles: [
        {
          id: "rrpf_one",
          issue_writer_model: "",
          issue_prompt: repositoryReviewDefaultIssuePrompt,
          max_parallel_children: 8,
        },
      ],
      total: 1,
      next_cursor: "cursor-2",
      canonical_query: "ORDER BY name ASC",
    })
    expect(mockedLauncherFetch).toHaveBeenCalledWith(
      "/api/repository-reviews/profiles?query=ORDER+BY+name+ASC&cursor=cursor-1&limit=25",
      { signal: undefined },
    )
  })

  it("sends strict profile CRUD envelopes and normalizes wrappers", async () => {
    const config: RepositoryReviewProfileConfig = {
      name: "Core bugs",
      account_ref: "",
      review_focus: "Find correctness bugs.",
      issue_prompt: "Present confirmed diagnosis with evidence.",
      scope_policy: {
        code_types: ["hotpath-code", "code"],
        include_folders: ["pkg"],
        exclude_folders: ["generated"],
        free_text: "Prioritize state transitions.",
      },
      reviewer_model: "review-model",
      force: false,
      auto_continue: true,
      max_files_per_run: 24,
      max_content_bytes: 524288,
      max_parallel_children: 8,
      assignment_timeout_seconds: 3_600,
      budget: {
        guard_expression:
          "account.limits.weekly.remaining_percent >= 10 and spend.total.usd < 25",
      },
    }
    const profile = {
      schema_version: 1,
      id: "profile/slash",
      version: 2,
      ...config,
      created_at: "2026-08-23T00:00:00Z",
      updated_at: "2026-08-23T00:00:00Z",
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          profiles: [
            {
              ...profile,
              max_parallel_children: undefined,
              assignment_timeout_seconds: undefined,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ profile }))
      .mockResolvedValueOnce(jsonResponse({ profile }))
      .mockResolvedValueOnce(jsonResponse({ profile }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))

    await expect(listRepositoryReviewProfiles()).resolves.toMatchObject({
      profiles: [
        {
          id: "profile/slash",
          reviewer_model: "review-model",
          max_parallel_children: 8,
          assignment_timeout_seconds: 3_600,
        },
      ],
    })
    await expect(
      getRepositoryReviewProfile("profile/slash"),
    ).resolves.toMatchObject({
      id: "profile/slash",
    })
    await createRepositoryReviewProfile(config)
    await updateRepositoryReviewProfile("profile/slash", {
      ...profile,
      expected_version: 2,
    })
    await deleteRepositoryReviewProfile("profile/slash", {
      expected_version: 3,
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/profiles",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/profiles/profile%2Fslash",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/profiles",
      expect.objectContaining({ method: "POST", body: JSON.stringify(config) }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/profiles/profile%2Fslash",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({ ...config, expected_version: 2 }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      5,
      "/api/repository-reviews/profiles/profile%2Fslash",
      expect.objectContaining({
        method: "DELETE",
        body: JSON.stringify({ expected_version: 3 }),
      }),
    )
  })

  it("sends minimal repository assignment payloads and normalizes branch snapshots", async () => {
    const automation = {
      id: "auto_1",
      version: 1,
      repository: "owner/repo",
      ref: "release",
      profile_id: "profile_1",
      profile_version: 3,
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(jsonResponse({ automation }))
      .mockResolvedValueOnce(jsonResponse({ automation }))

    await expect(
      createRepositoryReviewAutomation({
        repository: "owner/repo",
        branch: "",
        profile_id: "profile_1",
      }),
    ).resolves.toMatchObject({ branch: "release", reviewer_models: [] })
    await updateRepositoryReviewAutomation("auto_1", {
      repository: "owner/repo",
      branch: "release",
      profile_id: "profile_1",
      expected_version: 1,
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          repository: "owner/repo",
          branch: "",
          profile_id: "profile_1",
        }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto_1",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({
          repository: "owner/repo",
          branch: "release",
          profile_id: "profile_1",
          expected_version: 1,
        }),
      }),
    )
  })

  it("sends version-fenced automation CRUD and lifecycle mutations", async () => {
    const automation = {
      id: "auto/slash",
      version: 4,
      reviewer_models: [],
      run_ids: [],
      model_prices: {},
      budget: {},
      usage: {},
      progress: {},
      model_stats: [],
      account_limits: [],
    }
    for (let index = 0; index < 7; index += 1) {
      mockedLauncherFetch.mockResolvedValueOnce(
        index === 2
          ? new Response(null, { status: 204 })
          : jsonResponse({ automation }),
      )
    }
    const config = {
      repository: "owner/repo",
      branch: "main",
      profile_id: "profile_1",
    }

    await expect(
      createRepositoryReviewAutomation(config),
    ).resolves.toMatchObject({
      assignment_timeout_seconds: 3_600,
      progress: {
        assignment_progress: {
          total: 0,
          completed: 0,
          pending: 0,
          active: 0,
          by_focus: {
            correctness_state: { total: 0 },
            security_trust: { total: 0 },
            concurrency_recovery: { total: 0 },
            integration_validation: { total: 0 },
          },
        },
      },
    })
    await updateRepositoryReviewAutomation("auto/slash", {
      ...config,
      expected_version: 4,
    })
    await deleteRepositoryReviewAutomation("auto/slash", {
      expected_version: 5,
      expected_repository_version: 8,
      expected_ledger_fence: "rplf_delete",
      confirm_repository: "owner/repo",
    })
    await startRepositoryReviewAutomation("auto/slash", {
      expected_version: 6,
    })
    await pauseRepositoryReviewAutomation("auto/slash", {
      expected_version: 7,
      run_id: "wr_observed",
    })
    await resumeRepositoryReviewAutomation("auto/slash", {
      expected_version: 8,
    })
    await restartRepositoryReviewAutomation("auto/slash", {
      expected_version: 9,
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations",
      expect.objectContaining({ method: "POST", body: JSON.stringify(config) }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({ ...config, expected_version: 4 }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/automations/auto%2Fslash",
      expect.objectContaining({
        method: "DELETE",
        body: JSON.stringify({
          expected_version: 5,
          expected_repository_version: 8,
          expected_ledger_fence: "rplf_delete",
          confirm_repository: "owner/repo",
        }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/automations/auto%2Fslash/start",
      expect.objectContaining({
        body: JSON.stringify({ expected_version: 6 }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      5,
      "/api/repository-reviews/automations/auto%2Fslash/pause",
      expect.objectContaining({
        body: JSON.stringify({ expected_version: 7, run_id: "wr_observed" }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      6,
      "/api/repository-reviews/automations/auto%2Fslash/resume",
      expect.objectContaining({
        body: JSON.stringify({ expected_version: 8 }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      7,
      "/api/repository-reviews/automations/auto%2Fslash/restart",
      expect.objectContaining({
        body: JSON.stringify({ expected_version: 9 }),
      }),
    )
  })

  it("loads compound retention detail and purges history with both version fences", async () => {
    const automation = {
      id: "auto/slash",
      version: 4,
      repository: "owner/repo",
      reviewer_models: [],
      run_ids: [],
      model_prices: {},
      budget: {},
      usage: {},
      progress: {},
      model_stats: [],
      account_limits: [],
    }
    const summary = {
      repository_version: 9,
      ledger_fence: "rplf_purge",
      raw_findings: 12,
      deduplicated_findings: 8,
      repository_findings: 5,
      issue_previews: 3,
      external_issue_associations: 2,
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          repository: {
            id: "repo_1",
            repository: "owner/repo",
            version: 9,
          },
          capabilities: {
            can_purge_history: false,
            can_remove_repository: false,
            purge_blockers: [
              { code: "review_active", message: "Stop the review first." },
            ],
            purge_summary: summary,
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation: { ...automation, version: 5 },
          outcome: "history_purged",
        }),
      )

    await expect(
      getRepositoryReviewAutomationDetail("auto/slash"),
    ).resolves.toMatchObject({
      automation: { id: "auto/slash", repository: "owner/repo" },
      repository: { id: "repo_1", review_version: 0 },
      capabilities: {
        can_purge_history: false,
        purge_blockers: [
          {
            code: "review_active",
            count: 0,
            message: "Stop the review first.",
          },
        ],
        purge_summary: summary,
      },
    })
    await expect(
      purgeRepositoryReviewAutomationHistory("auto/slash", {
        expected_version: 4,
        expected_repository_version: 9,
        expected_ledger_fence: "rplf_purge",
        confirm_repository: "owner/repo",
      }),
    ).resolves.toMatchObject({
      automation: { id: "auto/slash", version: 5 },
      outcome: "history_purged",
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto%2Fslash",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/purge-history",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          expected_version: 4,
          expected_repository_version: 9,
          expected_ledger_fence: "rplf_purge",
          confirm_repository: "owner/repo",
        }),
      }),
    )
  })

  it("loads commit choices and submits an exact commit when resuming", async () => {
    const rememberedSHA = "a".repeat(40)
    const latestSHA = "b".repeat(40)
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          expected_version: 11,
          remembered: {
            sha: rememberedSHA,
            short_sha: rememberedSHA.slice(0, 8),
            url: `https://github.com/owner/repo/commit/${rememberedSHA}`,
          },
          latest: {
            sha: latestSHA,
            short_sha: latestSHA.slice(0, 8),
            url: `https://github.com/owner/repo/commit/${latestSHA}`,
          },
          newer_commit_available: true,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation: {
            id: "auto/slash",
            version: 12,
            resolved_commit_sha: latestSHA,
          },
        }),
      )

    await expect(
      getRepositoryReviewCommitOptions("auto/slash"),
    ).resolves.toMatchObject({
      expected_version: 11,
      newer_commit_available: true,
      remembered: { sha: rememberedSHA },
      latest: { sha: latestSHA },
    })
    await expect(
      resumeRepositoryReviewAutomation("auto/slash", {
        expected_version: 11,
        commit_sha: latestSHA,
      }),
    ).resolves.toMatchObject({ resolved_commit_sha: latestSHA })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto%2Fslash/commit-options",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/resume",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          expected_version: 11,
          commit_sha: latestSHA,
        }),
      }),
    )
  })

  it("loads a query-bound repository review run collection page", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse({
        automations: [{ id: "auto_1", status: "paused" }],
        total: 1,
        next_cursor: "next-page",
        canonical_query: 'status = "paused" ORDER BY repository ASC',
        query_schema: { fields: [{ name: "status", type: "enum" }] },
      }),
    )

    await expect(
      listRepositoryReviewAutomationsPage({
        query: "status = paused ORDER BY repository ASC",
        cursor: "cursor",
        limit: 50,
      }),
    ).resolves.toMatchObject({
      automations: [{ id: "auto_1", status: "paused" }],
      total: 1,
      next_cursor: "next-page",
    })
    expect(mockedLauncherFetch).toHaveBeenCalledWith(
      "/api/repository-reviews/automations?query=status+%3D+paused+ORDER+BY+repository+ASC&cursor=cursor&limit=50",
      { signal: undefined },
    )
  })

  it("loads and normalizes a bounded file attribution page", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse({
        automation: {
          id: "auto_1",
          repository: "owner/repo",
          reviewer_models: ["review"],
          progress: {},
        },
        file_attributions: [
          {
            id: "attribution_1",
            path: "pkg/store.go",
            commit_sha: "a".repeat(40),
            blob_sha: "b".repeat(40),
            focus_id: "security_trust",
            root_agent_id: "main",
            reviewer_identity: "review",
            model: "gpt-5.6-sol",
            source: "legacy",
            attempts: 2,
            run_ids: null,
            sources: null,
            latest_completed_at: "2026-08-26T01:00:00Z",
          },
        ],
        total: 1,
        next_cursor: "next-page",
        canonical_query: "ALL ORDER BY path ASC, focus ASC, reviewer ASC",
        query_schema: { fields: [{ name: "path", type: "string" }] },
      }),
    )

    await expect(
      listRepositoryReviewAutomationFileAttributionsPage("auto/slash", {
        limit: 200,
      }),
    ).resolves.toMatchObject({
      file_attributions: [
        {
          id: "attribution_1",
          path: "pkg/store.go",
          commit_sha: "a".repeat(40),
          blob_sha: "b".repeat(40),
          root_agent_id: "main",
          attempts: 2,
          run_ids: [],
          run_count: 0,
          sources: [],
        },
      ],
      total: 1,
      next_cursor: "next-page",
    })
    expect(mockedLauncherFetch).toHaveBeenCalledWith(
      "/api/repository-reviews/automations/auto%2Fslash/file-attributions?query=ALL+ORDER+BY+path+ASC%2C+focus+ASC%2C+reviewer+ASC&limit=200",
      undefined,
    )
  })

  it("separates run finding detail from deduplicated finding sources", async () => {
    const automation = {
      id: "auto/slash",
      repository: "owner/repo",
      reviewer_models: ["reviewer"],
      issue_writer_model: "writer",
      progress: { findings: 1 },
    }
    const finding = {
      id: "finding/slash",
      context_ids: null,
      models: null,
      observations: null,
      validation: { status: "confirmed", summary: "Confirmed", checks: null },
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(jsonResponse({ automation }))
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          findings: [finding],
          repository_findings: [
            {
              id: "rrf/one",
              canonical_title: "Cross-commit finding",
              review_finding_ids: null,
              found_commits: null,
              path_symbol_history: null,
              issue: { state: "none", conflict_urls: null },
              possible_duplicates: null,
              resolution_history: null,
            },
          ],
          contexts: [],
          scope: "all",
          offset: 50,
          total: 51,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({ automation, finding, contexts: [], capabilities: {} }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          sources: [
            {
              id: "source/slash",
              model: "provider/review",
              model_alias: "review",
              account: "review-account",
            },
          ],
          offset: 25,
          total: 26,
        }),
      )

    await expect(
      getRepositoryReviewAutomation("auto/slash"),
    ).resolves.toMatchObject({
      id: "auto/slash",
      issue_writer_model: "writer",
    })
    await expect(
      getRepositoryReviewAutomationFindings("auto/slash", {
        scope: "all",
        offset: 50,
        limit: 50,
      }),
    ).resolves.toMatchObject({
      scope: "all",
      offset: 50,
      total: 51,
      repository_finding_total: 1,
      repository_findings: [
        {
          id: "rrf/one",
          review_finding_ids: [],
          issue: { state: "none", conflict_urls: [] },
        },
      ],
    })
    await expect(
      getRepositoryReviewAutomationFinding("auto/slash", "finding/slash"),
    ).resolves.toMatchObject({
      finding: { id: "finding/slash", context_ids: [] },
    })
    await expect(
      listRepositoryReviewFindingRawSources(
        "auto/slash",
        "deduplicated/slash",
        { offset: 25, limit: 25 },
      ),
    ).resolves.toMatchObject({
      sources: [
        {
          id: "source/slash",
          model: "provider/review",
          model_alias: "review",
          account: "review-account",
        },
      ],
      offset: 25,
      total: 26,
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto%2Fslash",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/report?scope=all&offset=50&limit=50",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/automations/auto%2Fslash/findings/finding%2Fslash",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/automations/auto%2Fslash/findings/deduplicated%2Fslash/sources?offset=25&limit=25",
      { signal: undefined },
    )
  })

  it("uses cursor collection contracts for findings, repository findings, and issues", async () => {
    const automation = { id: "auto/slash", repository: "owner/repo" }
    const finding = {
      id: "finding/slash",
      repository: "owner/repo",
      path: "pkg/service.go",
      severity: "high",
      title: "Lost update",
      status: "open",
      run_finding_status: "failed",
      association: "unassociated",
      contributors: ["reviewer"],
      created_at: "2026-08-28T11:00:00Z",
      updated_at: "2026-08-28T12:00:00Z",
    }
    const repositoryFinding = {
      id: "rrf/slash",
      repository: "owner/repo",
      canonical_title: "Lost update",
      canonical_severity: "high",
      path: "pkg/service.go",
      symbol: "Save",
      match_state: "known",
      lifecycle: "open",
      issue: { state: "none" },
      validation_state: "not_requested",
      occurrence_count: 1,
      found_commit_count: 1,
      created_at: "2026-08-28T11:00:00Z",
      updated_at: "2026-08-28T12:00:00Z",
    }
    const issue = {
      id: "issue/slash",
      repository: "owner/repo",
      finding_count: 1,
      title: "Lost update",
      state: "editing",
      origin: "ai_generated",
      canonical: true,
      publishable: false,
      publish_blockers: [
        {
          code: "finding_status_unresolved",
          count: 3,
          message: "Three findings still need status processing.",
        },
      ],
      version: 3,
      created_at: "2026-08-28T11:00:00Z",
      updated_at: "2026-08-28T12:00:00Z",
    }
    const metadata = {
      total: 1,
      next_cursor: "next-page",
      canonical_query: "ALL ORDER BY updated DESC",
      query_schema: { fields: [{ name: "updated", type: "timestamp" }] },
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({ ...metadata, automation, findings: [finding] }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          ...metadata,
          automation,
          repository_findings: [repositoryFinding],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          finding,
          repository_finding: repositoryFinding,
          contexts: [],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          ...metadata,
          automation,
          issues: [issue],
          generation_id: "generation/slash",
          capabilities: {
            can_publish: false,
            publish_blockers: [
              {
                code: "duplicate_review_required",
                count: 1,
                message: "One duplicate decision is required.",
              },
            ],
          },
        }),
      )

    const runPage = await listRepositoryReviewAutomationFindingsPage(
      "auto/slash",
      {
        query: "severity = high ORDER BY updated DESC",
        cursor: "run-cursor",
        limit: 25,
      },
    )
    expect(runPage).toMatchObject({
      findings: [{ id: "finding/slash", contributors: ["reviewer"] }],
      next_cursor: "next-page",
    })
    expect(runPage.findings[0]).not.toHaveProperty("models")
    expect(runPage.findings[0]).not.toHaveProperty("evidence")

    const repositoryPage =
      await listRepositoryReviewAutomationRepositoryFindingsPage("auto/slash", {
        limit: 25,
      })
    expect(repositoryPage).toMatchObject({
      repository_findings: [
        {
          id: "rrf/slash",
          path: "pkg/service.go",
          occurrence_count: 1,
          found_commit_count: 1,
        },
      ],
    })
    expect(repositoryPage.repository_findings[0]).not.toHaveProperty(
      "review_finding_ids",
    )
    expect(repositoryPage.repository_findings[0]).not.toHaveProperty(
      "path_symbol_history",
    )
    await expect(
      getRepositoryReviewAutomationRepositoryFinding("auto/slash", "rrf/slash"),
    ).resolves.toMatchObject({ repository_finding: { id: "rrf/slash" } })
    const issuePage = await listRepositoryReviewAutomationIssuesPage(
      "auto/slash",
      {
        query: "   ",
        cursor: "issue-cursor",
        limit: 10,
        generation_id: "generation/slash",
      },
    )
    expect(issuePage).toMatchObject({
      issues: [
        {
          id: "issue/slash",
          finding_count: 1,
          version: 3,
          publishable: false,
          publish_blockers: [
            {
              code: "finding_status_unresolved",
              count: 3,
              message: "Three findings still need status processing.",
            },
          ],
        },
      ],
      generation_id: "generation/slash",
      capabilities: {
        can_publish: false,
        publish_blockers: [
          {
            code: "duplicate_review_required",
            count: 1,
            message: "One duplicate decision is required.",
          },
        ],
      },
    })
    expect(issuePage.issues[0]).not.toHaveProperty("finding_ids")
    expect(issuePage.issues[0]).not.toHaveProperty("body")

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto%2Fslash/findings?query=severity+%3D+high+ORDER+BY+updated+DESC&cursor=run-cursor&limit=25",
      undefined,
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/repository-findings?query=ALL+ORDER+BY+severity+DESC%2C+updated+DESC&limit=25",
      undefined,
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/automations/auto%2Fslash/repository-findings/rrf%2Fslash",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/automations/auto%2Fslash/issues?query=ALL+ORDER+BY+updated+DESC&cursor=issue-cursor&limit=10&generation_id=generation%2Fslash",
      undefined,
    )
  })

  it("preserves structured collection query error positions", async () => {
    mockedLauncherFetch.mockResolvedValueOnce(
      jsonResponse(
        {
          code: "invalid_query",
          message: "Expected a predicate after AND",
          position: 18,
        },
        400,
      ),
    )

    await expect(
      listRepositoryReviewAutomationFindingsPage("auto", {
        query: 'title = "é" AND',
      }),
    ).rejects.toMatchObject({
      name: "CollectionAPIError",
      status: 400,
      code: "invalid_query",
      position: 18,
    })
  })

  it("uses canonical raw finding collection, detail, and retry endpoints", async () => {
    const automation = { id: "auto/slash", repository: "owner/repo" }
    const rawFinding = {
      id: "rrw/slash",
      path: "pkg/store.go",
      severity: "high",
      title: "Lost update",
      model: "reviewer",
      deduplication_state: "failed",
      disposition: "undecided",
      failure: {
        code: "provider_failed",
        message: "Provider failed",
        retryable: true,
        at: "2026-08-29T12:00:00Z",
      },
      created_at: "2026-08-29T11:00:00Z",
      updated_at: "2026-08-29T12:00:00Z",
    }
    const findingsProcessing = {
      raw_total: 1,
      pending: 0,
      processing: 0,
      failed: 1,
      completed: 0,
      new: 0,
      duplicates: 0,
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          raw_findings: [rawFinding],
          total: 1,
          next_cursor: "raw-next",
          canonical_query: "ALL ORDER BY created DESC",
          query_schema: { fields: [] },
          findings_processing: findingsProcessing,
          historical_deduplication: {
            required: true,
            status: "failed",
            error: "Replay failed",
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({ automation, source: { ...rawFinding, id: "rrw_1" } }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          source: {
            ...rawFinding,
            id: "rrw_1",
            deduplication_state: "pending",
          },
          findings_processing: { ...findingsProcessing, failed: 0, pending: 1 },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          historical_deduplication: { required: true, status: "pending" },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          historical_deduplication: { required: true, status: "pending" },
        }),
      )

    await expect(
      listRepositoryReviewAutomationRawFindingsPage("auto/slash", {
        cursor: "raw/cursor",
        limit: 25,
      }),
    ).resolves.toMatchObject({
      raw_findings: [{ id: "rrw/slash", path: "pkg/store.go" }],
      next_cursor: "raw-next",
      findings_processing: { raw_total: 1, failed: 1 },
      historical_deduplication: { status: "failed" },
    })
    await expect(
      getRepositoryReviewRawSource("auto/slash", "rfn/legacy"),
    ).resolves.toMatchObject({ source: { id: "rrw_1" } })
    await expect(
      retryRepositoryReviewRawSource("auto/slash", "rrw_1"),
    ).resolves.toMatchObject({
      source: { id: "rrw_1", deduplication_state: "pending" },
    })
    await expect(
      retryRepositoryReviewHistoricalDeduplication("auto/slash"),
    ).resolves.toMatchObject({
      historical_deduplication: { required: true, status: "pending" },
    })
    await expect(
      restartRepositoryReviewHistoricalDeduplication("auto/slash"),
    ).resolves.toMatchObject({
      historical_deduplication: { required: true, status: "pending" },
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto%2Fslash/raw-findings?query=ALL+ORDER+BY+created+DESC&cursor=raw%2Fcursor&limit=25",
      undefined,
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/raw-findings/rfn%2Flegacy",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/automations/auto%2Fslash/raw-findings/rrw_1/retry",
      expect.objectContaining({ method: "POST", body: "{}" }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/automations/auto%2Fslash/historical-deduplication/retry",
      expect.objectContaining({ method: "POST", body: "{}" }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      5,
      "/api/repository-reviews/automations/auto%2Fslash/historical-deduplication/restart",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ confirmed: true }),
      }),
    )
  })

  it("uses health-backed findings-processing collection and recovery endpoints", async () => {
    const automation = { id: "auto/slash", repository: "owner/repo" }
    const source = {
      id: "source/slash",
      path: "pkg/store.go",
      severity: "high",
      title: "Lost update",
      model: "reviewer",
      reviewer: "correctness",
      deduplication_state: "failed",
      disposition: "undecided",
      failure: {
        code: "provider_failed",
        message: "The provider is temporarily unavailable.",
        retryable: true,
        at: "2026-08-31T12:00:00Z",
      },
      created_at: "2026-08-31T11:00:00Z",
      updated_at: "2026-08-31T12:00:00Z",
    }
    const counters = {
      total: 4,
      pending: 1,
      processing: 1,
      failed: 1,
      completed: 1,
    }
    const health = {
      run_findings: {
        total: 3,
        pending: 1,
        processing: 0,
        failed: 1,
        needs_review: 0,
        associated_new: 1,
        associated_existing: 0,
        unrepresented: 2,
      },
      repository_findings: {
        total: 1,
        provisional: 0,
        validation_failed: 0,
        issue_conflicts: 0,
      },
      findings_processing: counters,
      historical_consolidation: {
        required: true,
        status: "failed",
        retryable: true,
      },
      updated_at: "2026-08-31T12:00:00Z",
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(jsonResponse(health))
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          raw_findings: [source],
          total: 1,
          next_cursor: "next/source",
          canonical_query: "state = failed ORDER BY updated DESC",
          query_schema: { fields: [] },
          findings_processing: counters,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          source,
          repository_finding: { id: "rrf_1", issue: { state: "none" } },
          historical_consolidation: health.historical_consolidation,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          source: { ...source, deduplication_state: "pending" },
          findings_processing: { ...counters, pending: 2, failed: 0 },
          health: {
            ...health,
            findings_processing: { ...counters, pending: 2, failed: 0 },
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          retried_ids: ["source/slash"],
          failures: [
            {
              source_id: "source/blocked",
              code: "not_retryable",
              message: "This source is not retryable.",
            },
          ],
          findings_processing: { ...counters, pending: 2, failed: 0 },
          health,
        }),
      )

    await expect(
      getRepositoryReviewFindingHealth("auto/slash"),
    ).resolves.toMatchObject({
      run_findings: { unrepresented: 2 },
      findings_processing: { total: 4 },
      historical_consolidation: { status: "failed", retryable: true },
    })
    await expect(
      listRepositoryReviewFindingsProcessingPage("auto/slash", {
        query: "state = failed ORDER BY updated DESC",
        cursor: "next/source",
        limit: 25,
      }),
    ).resolves.toMatchObject({
      sources: [{ id: "source/slash", path: "pkg/store.go" }],
      next_cursor: "next/source",
      findings_processing: { total: 4, failed: 1 },
    })
    await expect(
      getRepositoryReviewFindingsProcessingSource("auto/slash", "source/slash"),
    ).resolves.toMatchObject({
      source: { id: "source/slash" },
      repository_finding: { id: "rrf_1", review_finding_ids: [] },
    })
    await expect(
      retryRepositoryReviewFindingsProcessingSource(
        "auto/slash",
        "source/slash",
      ),
    ).resolves.toMatchObject({
      source: { deduplication_state: "pending" },
      health: { findings_processing: { pending: 2 } },
    })
    await expect(
      retryRepositoryReviewFindingsProcessingSources("auto/slash", [
        "source/slash",
        "source/blocked",
      ]),
    ).resolves.toMatchObject({
      retried_ids: ["source/slash"],
      failures: [
        {
          source_id: "source/blocked",
          message: "This source is not retryable.",
        },
      ],
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto%2Fslash/finding-health",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/findings-processing?query=state+%3D+failed+ORDER+BY+updated+DESC&cursor=next%2Fsource&limit=25",
      undefined,
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/automations/auto%2Fslash/findings-processing/sources/source%2Fslash",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/automations/auto%2Fslash/findings-processing/sources/source%2Fslash/retry",
      expect.objectContaining({ method: "POST", body: "{}" }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      5,
      "/api/repository-reviews/automations/auto%2Fslash/findings-processing/retry",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          source_ids: ["source/slash", "source/blocked"],
        }),
      }),
    )
  })

  it("normalizes public run finding status and retries explicit findings", async () => {
    const automation = {
      id: "auto/slash",
      repository: "owner/repo",
      progress: {},
    }
    const finding = {
      id: "finding/slash",
      context_ids: null,
      models: null,
      observations: null,
      validation: { status: "confirmed", summary: "Confirmed", checks: null },
      run_finding_status: "failed",
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          findings: [finding],
          repository_findings: [],
          repository_finding_total: 0,
          scope: "current",
          offset: 0,
          total: 1,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          {
            automation,
            repository: { id: "rrp_repo", repository: "owner/repo" },
            findings: [{ id: "finding/slash", run_finding_status: "pending" }],
          },
          202,
        ),
      )

    await expect(
      getRepositoryReviewAutomationFindings("auto/slash"),
    ).resolves.toMatchObject({
      findings: [
        { id: "finding/slash", run_finding_status: "failed", context_ids: [] },
      ],
    })
    await expect(
      retryRepositoryReviewRunFindingStatuses("auto/slash", ["finding/slash"]),
    ).resolves.toMatchObject({
      findings: [{ id: "finding/slash", run_finding_status: "pending" }],
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      2,
      "/api/repository-reviews/automations/auto%2Fslash/findings/status",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ finding_ids: ["finding/slash"] }),
      }),
    )
  })

  it("uses automation-owned issue endpoints and strict mutation bodies", async () => {
    const automation = { id: "auto", repository: "owner/repo" }
    const issue = {
      id: "draft/slash",
      finding_ids: ["finding"],
      state: "editing",
      labels: null,
    }
    mockedLauncherFetch
      .mockResolvedValueOnce(
        jsonResponse({ automation, issues: [issue], offset: 0, total: 1 }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          automation,
          issue,
          capabilities: { can_publish: false, publish_blockers: null },
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ automation, issue }))
      .mockResolvedValueOnce(
        jsonResponse({ generation_id: "rrig_1", issues: [issue], results: [] }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          results: [{ draft_id: "draft/slash", outcome: "deleted" }],
        }),
      )

    await listRepositoryReviewAutomationIssues("auto", {
      generation_id: "rrig_1",
      limit: 200,
    })
    await expect(
      getRepositoryReviewAutomationIssue("auto", "draft/slash"),
    ).resolves.toMatchObject({
      capabilities: { can_publish: false, publish_blockers: [] },
    })
    await updateRepositoryReviewAutomationIssue("auto", "draft/slash", {
      title: "Title",
      body: "Body",
      labels: ["bug"],
      expected_version: 2,
    })
    await generateRepositoryReviewIssues("auto", {
      generation_id: "rrig_1",
      finding_ids: ["finding"],
      instructions_mode: "default",
    })
    await deleteRepositoryReviewAutomationIssue("auto", "draft/slash", {
      expected_version: 3,
      confirmed: true,
    })

    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      1,
      "/api/repository-reviews/automations/auto/issues?generation_id=rrig_1&limit=200",
      { signal: undefined },
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      3,
      "/api/repository-reviews/automations/auto/issues/draft%2Fslash",
      expect.objectContaining({
        method: "PATCH",
        body: JSON.stringify({
          title: "Title",
          body: "Body",
          labels: ["bug"],
          expected_version: 2,
        }),
      }),
    )
    expect(mockedLauncherFetch).toHaveBeenNthCalledWith(
      4,
      "/api/repository-reviews/automations/auto/issues/generations",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          generation_id: "rrig_1",
          finding_ids: ["finding"],
          instructions_mode: "default",
        }),
      }),
    )
  })
})

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}
