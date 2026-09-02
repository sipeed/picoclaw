import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import {
  RepositoryReviewAPIError,
  type RepositoryReviewAutomation,
  type RepositoryReviewAutomationDetail,
  createRepositoryReviewAutomation,
  deleteRepositoryReviewAutomation,
  getRepositoryReviewAutomation,
  getRepositoryReviewAutomationDetail,
  listRepositoryReviewAutomations,
  listRepositoryReviewAutomationsPage,
  listRepositoryReviewProfiles,
  purgeRepositoryReviewAutomationHistory,
  updateRepositoryReviewAutomation,
} from "@/api/repository-reviews"
import {
  RepositoryReviewRepositoriesPage,
  RepositoryReviewRepositoryDetailPage,
  RepositoryReviewRepositoryEditorPage,
} from "@/components/repository-reviews/repository-review-repositories-page"

vi.mock("@/components/page-header", () => ({
  PageHeader: ({
    title,
    children,
  }: {
    title: string
    children?: React.ReactNode
  }) => (
    <header>
      <h1>{title}</h1>
      {children}
    </header>
  ),
}))

vi.mock("@/api/repository-reviews", () => ({
  RepositoryReviewAPIError: class RepositoryReviewAPIError extends Error {
    status: number

    constructor(status: number, message: string) {
      super(message)
      this.status = status
    }
  },
  createRepositoryReviewAutomation: vi.fn(),
  deleteRepositoryReviewAutomation: vi.fn(),
  getRepositoryReviewAutomation: vi.fn(),
  getRepositoryReviewAutomationDetail: vi.fn(),
  listRepositoryReviewAutomations: vi.fn(),
  listRepositoryReviewAutomationsPage: vi.fn(),
  listRepositoryReviewProfiles: vi.fn(),
  purgeRepositoryReviewAutomationHistory: vi.fn(),
  updateRepositoryReviewAutomation: vi.fn(),
}))

const profile = {
  id: "profile_1",
  version: 2,
  name: "Core bugs",
  account_ref: "",
  reviewer_model: "review-model",
  review_focus: "Correctness bugs",
  issue_prompt: "Present the confirmed diagnosis with evidence.",
  scope_policy: {
    code_types: ["code" as const],
    include_folders: [],
    exclude_folders: [],
    free_text: "",
  },
  force: false,
  auto_continue: true,
  max_files_per_run: 24,
  max_content_bytes: 524288,
  max_parallel_children: 8,
  assignment_timeout_seconds: 3_600,
  budget: {
    guard_expression: "spend.total.usd < 25",
  },
  created_at: "2026-08-23T00:00:00Z",
  updated_at: "2026-08-23T00:00:00Z",
}

const repository = {
  id: "auto_1",
  version: 1,
  profile_id: profile.id,
  profile_version: profile.version,
  branch: "",
  name: profile.name,
  repository: "owner/repo",
  ref: "",
  target: "all",
  account_ref: profile.account_ref,
  review_focus: profile.review_focus,
  scope_policy: profile.scope_policy,
  reviewer_models: [profile.reviewer_model],
  compare_models: false,
  force: false,
  max_files_per_run: 24,
  max_content_bytes: 524288,
  max_parallel_children: 8,
  assignment_timeout_seconds: profile.assignment_timeout_seconds,
  auto_continue: true,
  model_prices: {
    [profile.reviewer_model]: {
      input_price_per_1m: 1,
      output_price_per_1m: 4,
    },
  },
  budget: profile.budget,
  status: "idle" as const,
  run_ids: [],
  usage: {
    prompt_tokens: 0,
    completion_tokens: 0,
    total_tokens: 0,
    cached_tokens: 0,
  },
  estimated_cost_usd: 0,
  progress: {
    stage: "waiting",
    completed_batches: 0,
    total_batches: 0,
    coverage_available: false,
    coverage_exact: false,
    selected_files: 0,
    inspected_files: 0,
    reviewed_files: 0,
    remaining_files: 0,
    unsupported_files: 0,
    findings: 0,
    finding_aggregates: 0,
    unaggregated_findings: 0,
    assignment_progress: {
      total: 0,
      completed: 0,
      pending: 0,
      active: 0,
      by_focus: {
        correctness_state: { total: 0, completed: 0, pending: 0, active: 0 },
        security_trust: { total: 0, completed: 0, pending: 0, active: 0 },
        concurrency_recovery: {
          total: 0,
          completed: 0,
          pending: 0,
          active: 0,
        },
        integration_validation: {
          total: 0,
          completed: 0,
          pending: 0,
          active: 0,
        },
      },
    },
  },
  model_stats: [],
  account_limits: [],
  created_at: "2026-08-23T00:00:00Z",
  updated_at: "2026-08-23T00:00:00Z",
}

const repositoryDetail = {
  automation: repository,
  repository: {
    schema_version: 1,
    id: "repo_1",
    repository: repository.repository,
    version: 7,
    review_version: 4,
    updated_at: repository.updated_at,
  },
  capabilities: {
    can_purge_history: true,
    can_remove_repository: true,
    purge_blockers: [],
    purge_summary: {
      repository_version: 7,
      ledger_fence: "rplf_repository",
      raw_findings: 12,
      deduplicated_findings: 8,
      repository_findings: 5,
      issue_previews: 3,
      external_issue_associations: 2,
    },
  },
} satisfies RepositoryReviewAutomationDetail

describe("RepositoryReviewRepositoriesPage", () => {
  beforeEach(() => {
    vi.mocked(listRepositoryReviewAutomations).mockResolvedValue({
      automations: [],
    })
    vi.mocked(listRepositoryReviewProfiles).mockResolvedValue({
      profiles: [profile],
    })
    vi.mocked(listRepositoryReviewAutomationsPage).mockResolvedValue({
      automations: [],
      total: 0,
      next_cursor: "",
      canonical_query: "ORDER BY repository ASC",
      query_schema: { fields: [] },
    })
    vi.mocked(getRepositoryReviewAutomation).mockReset()
    vi.mocked(getRepositoryReviewAutomation).mockResolvedValue(repository)
    vi.mocked(getRepositoryReviewAutomationDetail).mockReset()
    vi.mocked(getRepositoryReviewAutomationDetail).mockResolvedValue(
      repositoryDetail,
    )
    vi.mocked(createRepositoryReviewAutomation).mockReset()
    vi.mocked(updateRepositoryReviewAutomation).mockReset()
    vi.mocked(deleteRepositoryReviewAutomation).mockReset()
    vi.mocked(purgeRepositoryReviewAutomationHistory).mockReset()
    vi.mocked(purgeRepositoryReviewAutomationHistory).mockResolvedValue({
      automation: { ...repository, version: 2 },
      outcome: "history_purged",
    })
  })

  it("uses the standard collection and exposes repository findings on each item", async () => {
    const user = userEvent.setup()
    const onOpenFindings = vi.fn()
    vi.mocked(listRepositoryReviewAutomationsPage).mockResolvedValue({
      automations: [repository],
      total: 1,
      next_cursor: "",
      canonical_query: "ORDER BY repository ASC",
      query_schema: { fields: [] },
    })

    renderCollection({ onOpenFindings })

    await user.click(
      await screen.findByRole("button", {
        name: "Repository findings for owner/repo",
      }),
    )

    expect(listRepositoryReviewAutomationsPage).toHaveBeenCalledWith(
      {
        query: "ORDER BY repository ASC",
        cursor: undefined,
        limit: 50,
      },
      expect.any(AbortSignal),
    )
    expect(onOpenFindings).toHaveBeenCalledWith(repository)
    expect(
      screen.getByRole("region", { name: "Review repositories list" }),
    ).toBeVisible()
  })

  it("assigns one profile and defaults to the repository base branch", async () => {
    const user = userEvent.setup()
    vi.mocked(createRepositoryReviewAutomation).mockResolvedValue({
      id: "auto_1",
      version: 1,
      profile_id: profile.id,
      profile_version: profile.version,
      branch: "",
      name: profile.name,
      repository: "owner/repo",
      ref: "",
      target: "all",
      account_ref: profile.account_ref,
      review_focus: profile.review_focus,
      scope_policy: profile.scope_policy,
      reviewer_models: [profile.reviewer_model],
      compare_models: false,
      force: false,
      max_files_per_run: 24,
      max_content_bytes: 524288,
      max_parallel_children: 8,
      assignment_timeout_seconds: profile.assignment_timeout_seconds,
      auto_continue: true,
      model_prices: {
        [profile.reviewer_model]: {
          input_price_per_1m: 1,
          output_price_per_1m: 4,
        },
      },
      budget: profile.budget,
      status: "idle",
      run_ids: [],
      usage: {
        prompt_tokens: 0,
        completion_tokens: 0,
        total_tokens: 0,
        cached_tokens: 0,
      },
      estimated_cost_usd: 0,
      progress: repository.progress,
      model_stats: [],
      account_limits: [],
      created_at: "2026-08-23T00:00:00Z",
      updated_at: "2026-08-23T00:00:00Z",
    })
    renderEditor()

    await user.type(await screen.findByLabelText("Repository"), "owner/repo")
    expect(screen.queryByLabelText("Branch override")).not.toBeInTheDocument()
    await user.click(screen.getByText(/^Advanced/))
    expect(screen.getByLabelText("Branch override")).toHaveValue("")
    await user.click(screen.getByRole("button", { name: "Save repository" }))

    await waitFor(() =>
      expect(createRepositoryReviewAutomation).toHaveBeenCalledWith({
        repository: "owner/repo",
        profile_id: profile.id,
        branch: "",
      }),
    )
  })

  it("blocks a second configuration for the same repository", async () => {
    const user = userEvent.setup()
    vi.mocked(listRepositoryReviewAutomations).mockResolvedValue({
      automations: [{ id: "auto_1", repository: "Owner/Repo" } as never],
    })
    renderEditor()
    await user.type(await screen.findByLabelText("Repository"), "owner/repo")
    expect(
      screen.getByText(/already has a review configuration/i),
    ).toBeVisible()
    expect(
      screen.getByRole("button", { name: "Save repository" }),
    ).toBeDisabled()
  })

  it("accepts branches and rejects revision expressions like the backend", async () => {
    const user = userEvent.setup()
    renderEditor()
    await user.type(await screen.findByLabelText("Repository"), "owner/repo")
    await user.click(screen.getByRole("button", { name: /^Advanced/ }))
    const branch = screen.getByLabelText("Branch override")
    for (const invalid of [
      "HEAD",
      "deadbee",
      "refs/heads/main",
      "tags/v1",
      "feature#1",
      ".hidden/main",
      "feature/build.lock",
      "a".repeat(256),
    ]) {
      fireEvent.change(branch, { target: { value: invalid } })
      expect(
        screen.getByText(/Enter a branch name, or leave blank/i),
      ).toBeVisible()
      expect(
        screen.getByRole("button", { name: "Save repository" }),
      ).toBeDisabled()
    }
    fireEvent.change(branch, { target: { value: "feature/review-ui" } })
    expect(
      screen.queryByText(/Enter a branch name, or leave blank/i),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Save repository" }),
    ).toBeEnabled()
  })

  it("shows save errors inside the repository editor", async () => {
    const user = userEvent.setup()
    vi.mocked(createRepositoryReviewAutomation).mockRejectedValue(
      new Error("Repository is already assigned."),
    )
    renderEditor()
    await user.type(
      await screen.findByLabelText("Repository"),
      "owner/new-repo",
    )
    await user.click(screen.getByRole("button", { name: "Save repository" }))

    const alert = await screen.findByRole("alert")
    expect(alert).toHaveTextContent("already assigned")
  })

  it("shows deletion counts and requires the exact repository before purging history", async () => {
    const user = userEvent.setup()
    renderDetail()

    const history = await screen.findByRole("region", {
      name: "Local review history",
    })
    expect(within(history).getByText("12")).toBeVisible()
    expect(within(history).getByText("8")).toBeVisible()
    expect(within(history).getByText("5")).toBeVisible()
    expect(within(history).getByText("3")).toBeVisible()
    expect(within(history).getByText("2")).toBeVisible()

    await user.click(
      screen.getByRole("button", { name: "Purge review history" }),
    )
    const dialog = screen.getByRole("alertdialog")
    const confirmation = within(dialog).getByLabelText(
      "Type owner/repo to confirm",
    )
    const confirm = within(dialog).getByRole("button", {
      name: "Purge review history",
    })
    expect(confirm).toBeDisabled()
    await user.type(confirmation, "Owner/repo")
    expect(confirm).toBeDisabled()
    await user.clear(confirmation)
    await user.type(confirmation, "owner/repo")
    expect(confirm).toBeEnabled()
    await user.click(confirm)

    await waitFor(() =>
      expect(purgeRepositoryReviewAutomationHistory).toHaveBeenCalledWith(
        "auto_1",
        {
          expected_version: 1,
          expected_repository_version: 7,
          expected_ledger_fence: "rplf_repository",
          confirm_repository: "owner/repo",
        },
      ),
    )
    await waitFor(() =>
      expect(getRepositoryReviewAutomationDetail).toHaveBeenCalledTimes(2),
    )
    expect(await screen.findByText(/Review history was purged/i)).toBeVisible()
  })

  it("requires typed confirmation before removing configuration and all local history", async () => {
    const user = userEvent.setup()
    const onDeleted = vi.fn()
    renderDetail({ onDeleted })

    await user.click(await screen.findByRole("button", { name: "Remove" }))
    const dialog = screen.getByRole("alertdialog")
    expect(dialog).toHaveTextContent(
      "repository configuration, run controls, and repository-review ledger",
    )
    expect(dialog).toHaveTextContent(
      "Existing GitHub issues are not changed or deleted",
    )
    await user.type(
      within(dialog).getByLabelText("Type owner/repo to confirm"),
      "owner/repo",
    )
    await user.click(
      within(dialog).getByRole("button", { name: "Remove repository" }),
    )

    await waitFor(() =>
      expect(deleteRepositoryReviewAutomation).toHaveBeenCalledWith("auto_1", {
        expected_version: 1,
        expected_repository_version: 7,
        expected_ledger_fence: "rplf_repository",
        confirm_repository: "owner/repo",
      }),
    )
    await waitFor(() => expect(onDeleted).toHaveBeenCalledOnce())
  })

  it("fails destructive actions closed and shows backend blockers", async () => {
    vi.mocked(getRepositoryReviewAutomationDetail).mockResolvedValue({
      ...repositoryDetail,
      capabilities: {
        ...repositoryDetail.capabilities,
        can_purge_history: false,
        can_remove_repository: false,
        purge_blockers: [
          {
            code: "review_active",
            count: 1,
            message: "Stop the active repository review first.",
          },
        ],
      },
    })
    renderDetail()

    expect(
      await screen.findByText("Stop the active repository review first."),
    ).toBeVisible()
    expect(
      screen.getByRole("button", { name: "Purge review history" }),
    ).toBeDisabled()
    expect(screen.getByRole("button", { name: "Remove" })).toBeDisabled()
  })

  it("refetches compound detail after a destructive-action conflict", async () => {
    vi.mocked(purgeRepositoryReviewAutomationHistory).mockRejectedValueOnce(
      new RepositoryReviewAPIError(409, "Repository history changed."),
    )
    const user = userEvent.setup()
    renderDetail()

    await user.click(
      await screen.findByRole("button", { name: "Purge review history" }),
    )
    const dialog = screen.getByRole("alertdialog")
    await user.type(
      within(dialog).getByLabelText("Type owner/repo to confirm"),
      "owner/repo",
    )
    await user.click(
      within(dialog).getByRole("button", { name: "Purge review history" }),
    )

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Latest repository details have been reloaded",
    )
    await waitFor(() =>
      expect(getRepositoryReviewAutomationDetail).toHaveBeenCalledTimes(2),
    )
  })

  it("retries a transient new-editor context error without loading an empty repository ID", async () => {
    vi.mocked(listRepositoryReviewProfiles)
      .mockRejectedValueOnce(new Error("Temporary profile load failure."))
      .mockResolvedValueOnce({ profiles: [profile] })
    const user = userEvent.setup()

    renderEditor()

    expect(
      await screen.findByText("Temporary profile load failure."),
    ).toBeVisible()
    await user.click(screen.getByRole("button", { name: "Retry" }))

    expect(await screen.findByLabelText("Repository")).toBeVisible()
    expect(getRepositoryReviewAutomation).not.toHaveBeenCalled()
  })
})

function renderEditor() {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <RepositoryReviewRepositoryEditorPage
        onBack={vi.fn()}
        onSaved={vi.fn()}
      />
    </QueryClientProvider>,
  )
}

function renderDetail({
  onDeleted = vi.fn(),
}: {
  onDeleted?: () => void
} = {}) {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <RepositoryReviewRepositoryDetailPage
        automationID="auto_1"
        onBack={vi.fn()}
        onEdit={vi.fn()}
        onFindings={vi.fn()}
        onDeleted={onDeleted}
      />
    </QueryClientProvider>,
  )
}

function renderCollection({
  onOpenFindings = vi.fn(),
}: {
  onOpenFindings?: (repository: RepositoryReviewAutomation) => void
} = {}) {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <RepositoryReviewRepositoriesPage
        search={{ q: "ORDER BY repository ASC" }}
        onSearchChange={vi.fn()}
        onAdd={vi.fn()}
        onOpen={vi.fn()}
        onEdit={vi.fn()}
        onOpenFindings={onOpenFindings}
      />
    </QueryClientProvider>,
  )
}
