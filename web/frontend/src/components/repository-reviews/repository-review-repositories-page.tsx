import {
  IconAlertTriangle,
  IconEdit,
  IconListDetails,
  IconPlus,
  IconRefresh,
  IconTrash,
} from "@tabler/icons-react"
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { type FormEvent, useMemo, useState } from "react"

import {
  RepositoryReviewAPIError,
  type RepositoryReviewAutomation,
  type RepositoryReviewAutomationDetail,
  type RepositoryReviewProfile,
  type RepositoryReviewPurgeSummary,
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
  type CollectionDefinition,
  CollectionDetailShell,
} from "@/components/collection"
import { StandardCollectionPage } from "@/components/collection/standard-collection-page"
import {
  repositoryReviewRepositoryDefaultQuery,
  repositoryReviewRepositoryViews,
} from "@/components/repository-reviews/repository-review-repositories-route-state"
import { ReviewAdvancedSection } from "@/components/repository-reviews/review-advanced-section"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  type CollectionRouteSearch,
  normalizeCollectionRouteSearch,
} from "@/hooks/use-collection-route-state"

const repositoriesKey = ["repository-review-automations"] as const
const repositoryDetailKey = (automationID: string) =>
  ["repository-review-automation", automationID] as const
const repositoryCompoundDetailKey = (automationID: string) =>
  ["repository-review-automation-detail", automationID] as const
const editorContextKey = [
  "repository-review-repository-editor-context",
] as const

interface RepositoryEditorValue {
  repository: string
  profileID: string
  branch: string
}

export function RepositoryReviewRepositoriesPage({
  search,
  onSearchChange,
  onAdd,
  onOpen,
  onEdit,
  onOpenFindings,
}: {
  search: { q?: string; view?: "list" | "table" | "grid" }
  onSearchChange: (search: CollectionRouteSearch, replace?: boolean) => void
  onAdd: () => void
  onOpen: (repository: RepositoryReviewAutomation) => void
  onEdit: (repository: RepositoryReviewAutomation) => void
  onOpenFindings: (repository: RepositoryReviewAutomation) => void
}) {
  const activeQuery = normalizeCollectionRouteSearch(search, {
    defaultQuery: repositoryReviewRepositoryDefaultQuery,
    supportedViews: repositoryReviewRepositoryViews,
  }).q
  const query = useInfiniteQuery({
    queryKey: [...repositoriesKey, "configurations", activeQuery],
    initialPageParam: "",
    queryFn: ({ pageParam, signal }) =>
      listRepositoryReviewAutomationsPage(
        {
          query: activeQuery,
          cursor: pageParam || undefined,
          limit: 50,
        },
        signal,
      ),
    getNextPageParam: (page) => page.next_cursor || undefined,
    retry: false,
  })
  const repositories = useMemo(
    () => query.data?.pages.flatMap((page) => page.automations) ?? [],
    [query.data?.pages],
  )
  const firstPage = query.data?.pages[0]

  const definition = useMemo<CollectionDefinition<RepositoryReviewAutomation>>(
    () => ({
      key: "repository-review-repositories",
      title: "Review repositories",
      defaultQuery: repositoryReviewRepositoryDefaultQuery,
      supportedViews: repositoryReviewRepositoryViews,
      defaultView: "list",
      getItemID: (item) => item.id,
      getItemLabel: (item) => item.repository || item.id,
      getItemIdentity: (item) => ({
        title: item.repository || "Review repository",
        description: item.name || item.profile_id || "Missing profile",
        metadata: (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span>
              Profile snapshot v{item.profile_version || "not reported"}
            </span>
            <Button
              type="button"
              size="sm"
              variant="link"
              className="h-auto p-0 text-xs"
              aria-label={`Repository findings for ${item.repository}`}
              onClick={() => onOpenFindings(item)}
            >
              <IconListDetails /> Repository findings
            </Button>
          </div>
        ),
      }),
      columns: [
        {
          id: "profile",
          header: "Profile",
          cell: (item) => item.name || item.profile_id || "Missing profile",
        },
        {
          id: "branch",
          header: "Branch",
          cell: repositoryBranchLabel,
        },
        {
          id: "updated",
          header: "Updated",
          cell: (item) => formatTimestamp(item.updated_at),
          className: "w-44",
        },
      ],
      gridFacts: [
        {
          id: "profile",
          label: "Profile",
          value: (item) => item.name || item.profile_id || "Missing profile",
        },
        { id: "branch", label: "Branch", value: repositoryBranchLabel },
        {
          id: "updated",
          label: "Updated",
          value: (item) => formatTimestamp(item.updated_at),
        },
      ],
      badges: [
        {
          id: "status",
          label: (item) => item.status,
          variant: "secondary",
        },
      ],
      actions: [
        {
          id: "findings",
          label: "Repository findings",
          icon: <IconListDetails />,
          onSelect: onOpenFindings,
        },
        {
          id: "edit",
          label: "Edit repository",
          icon: <IconEdit />,
          disabled: repositoryConfigurationBusy,
          onSelect: onEdit,
        },
      ],
    }),
    [onEdit, onOpenFindings],
  )

  return (
    <StandardCollectionPage
      definition={definition}
      search={search}
      onSearchChange={onSearchChange}
      items={repositories}
      total={firstPage?.total}
      schema={firstPage?.query_schema}
      canonicalQuery={firstPage?.canonical_query}
      loading={query.isLoading}
      fetching={query.isFetching}
      error={query.error}
      onRefresh={query.refetch}
      hasNextPage={query.hasNextPage}
      loadingMore={query.isFetchingNextPage}
      onLoadMore={query.fetchNextPage}
      onOpenItem={onOpen}
      addAction={
        <Button type="button" size="sm" onClick={onAdd}>
          <IconPlus /> Add repository
        </Button>
      }
      emptyTitle="No repository configured"
      emptyDescription="Add a repository and assign a review profile."
    />
  )
}

export function RepositoryReviewRepositoryDetailPage({
  automationID,
  onBack,
  onEdit,
  onFindings,
  onDeleted,
}: {
  automationID: string
  onBack: () => void
  onEdit: () => void
  onFindings: () => void
  onDeleted: () => void
}) {
  const queryClient = useQueryClient()
  const [actionError, setActionError] = useState("")
  const [actionNotice, setActionNotice] = useState("")
  const query = useQuery({
    queryKey: repositoryCompoundDetailKey(automationID),
    queryFn: ({ signal }) =>
      getRepositoryReviewAutomationDetail(automationID, signal),
    retry: false,
  })
  const detail = query.data
  const automation = detail?.automation
  const capabilities = detail?.capabilities
  const purgeSummary = capabilities?.purge_summary
  const purgeBlockers = capabilities?.purge_blockers ?? []
  const notFound =
    query.error instanceof RepositoryReviewAPIError &&
    query.error.status === 404
  const handleMutationError = (error: unknown) => {
    const conflict =
      error instanceof RepositoryReviewAPIError && error.status === 409
    setActionNotice("")
    setActionError(
      conflict
        ? `${errorMessage(error)} Latest repository details have been reloaded.`
        : errorMessage(error),
    )
    if (conflict) void query.refetch()
  }
  const purgeMutation = useMutation({
    mutationFn: async () => {
      if (!detail) throw new Error("Repository is unavailable.")
      const input = historyMutationInput(detail)
      return purgeRepositoryReviewAutomationHistory(detail.automation.id, input)
    },
    onMutate: () => {
      setActionError("")
      setActionNotice("")
    },
    onSuccess: async (result) => {
      queryClient.setQueryData(
        repositoryDetailKey(result.automation.id),
        result.automation,
      )
      await queryClient.invalidateQueries({ queryKey: repositoriesKey })
      setActionNotice(
        "Review history was purged. Existing GitHub issues were not changed.",
      )
      await query.refetch()
    },
    onError: handleMutationError,
  })
  const removeMutation = useMutation({
    mutationFn: async () => {
      if (!detail) throw new Error("Repository is unavailable.")
      const input = historyMutationInput(detail)
      await deleteRepositoryReviewAutomation(detail.automation.id, input)
    },
    onMutate: () => {
      setActionError("")
      setActionNotice("")
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: repositoriesKey })
      onDeleted()
    },
    onError: handleMutationError,
  })
  const busy = automation ? repositoryConfigurationBusy(automation) : false
  const destructiveActionPending =
    purgeMutation.isPending || removeMutation.isPending
  const destructiveActionUnavailableMessage =
    purgeBlockers[0]?.message ||
    (!purgeSummary
      ? "History deletion details are unavailable. Refresh and try again."
      : "This action is not available for the repository's current state.")
  const canPurgeHistory =
    capabilities?.can_purge_history === true && Boolean(purgeSummary)
  const canRemoveRepository =
    capabilities?.can_remove_repository === true && Boolean(purgeSummary)

  return (
    <CollectionDetailShell
      title={automation?.repository || "Review repository"}
      identity={<span className="font-mono text-xs">{automationID}</span>}
      status={
        automation ? (
          <Badge variant="secondary">{automation.status}</Badge>
        ) : undefined
      }
      loading={query.isLoading}
      error={!notFound ? errorMessage(query.error) || undefined : undefined}
      notFound={notFound}
      onBack={onBack}
      onRetry={() => void query.refetch()}
      backLabel="All review repositories"
      actions={
        automation ? (
          <>
            <Button type="button" size="sm" onClick={onFindings}>
              <IconListDetails /> Repository findings
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={onEdit}
            >
              <IconEdit /> Edit
            </Button>
            <RepositoryHistoryDestructiveDialog
              action="purge"
              repository={automation.repository}
              summary={purgeSummary}
              disabled={busy || !canPurgeHistory || destructiveActionPending}
              disabledReason={destructiveActionUnavailableMessage}
              pending={purgeMutation.isPending}
              onConfirm={() => purgeMutation.mutate()}
            />
            <RepositoryHistoryDestructiveDialog
              action="remove"
              repository={automation.repository}
              summary={purgeSummary}
              disabled={
                busy || !canRemoveRepository || destructiveActionPending
              }
              disabledReason={destructiveActionUnavailableMessage}
              pending={removeMutation.isPending}
              onConfirm={() => removeMutation.mutate()}
            />
          </>
        ) : undefined
      }
    >
      {automation && (
        <div className="space-y-5">
          {actionError && (
            <div
              role="alert"
              className="text-destructive flex items-center gap-2 text-sm"
            >
              <IconAlertTriangle className="size-4" /> {actionError}
            </div>
          )}
          {actionNotice && (
            <div
              role="status"
              className="border-border bg-muted/30 flex items-center gap-2 rounded-lg border px-3 py-2 text-sm"
            >
              <IconRefresh className="size-4" /> {actionNotice}
            </div>
          )}
          <section aria-labelledby="repository-configuration-heading">
            <h2
              id="repository-configuration-heading"
              className="mb-2 text-sm font-semibold"
            >
              Configuration
            </h2>
            <dl className="border-border divide-border divide-y rounded-lg border text-sm">
              <RepositoryDetailRow
                label="Repository"
                value={automation.repository}
              />
              <RepositoryDetailRow
                label="Assigned profile"
                value={automation.name || automation.profile_id}
              />
              <RepositoryDetailRow
                label="Profile identity"
                value={`${automation.profile_id} · snapshot v${automation.profile_version}`}
                mono
              />
              <RepositoryDetailRow
                label="Branch"
                value={repositoryBranchLabel(automation)}
              />
              <RepositoryDetailRow
                label="Updated"
                value={formatTimestamp(automation.updated_at)}
              />
            </dl>
          </section>
          <section aria-labelledby="repository-history-heading">
            <h2
              id="repository-history-heading"
              className="mb-2 text-sm font-semibold"
            >
              Local review history
            </h2>
            <p className="text-muted-foreground mb-3 text-sm">
              These records are deleted by Purge review history or Remove.
              Existing GitHub issues are never changed or deleted.
            </p>
            {purgeSummary ? (
              <HistoryDeletionCounts summary={purgeSummary} />
            ) : (
              <div
                role="status"
                className="border-border text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-sm"
              >
                History deletion counts are unavailable. Refresh the repository
                details before deleting history.
              </div>
            )}
            {purgeBlockers.length > 0 && (
              <div
                role="status"
                className="border-border bg-muted/30 mt-3 rounded-lg border px-3 py-2 text-sm"
              >
                <p className="font-medium">Review history status</p>
                <ul className="text-muted-foreground mt-1 list-disc space-y-1 pl-5">
                  {purgeBlockers.map((blocker) => (
                    <li key={blocker.code}>
                      {blocker.message}
                      {blocker.count > 1
                        ? ` (${blocker.count.toLocaleString()})`
                        : ""}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </section>
        </div>
      )}
    </CollectionDetailShell>
  )
}

function RepositoryHistoryDestructiveDialog({
  action,
  repository,
  summary,
  disabled,
  disabledReason,
  pending,
  onConfirm,
}: {
  action: "purge" | "remove"
  repository: string
  summary?: RepositoryReviewPurgeSummary
  disabled: boolean
  disabledReason: string
  pending: boolean
  onConfirm: () => void
}) {
  const [open, setOpen] = useState(false)
  const [confirmation, setConfirmation] = useState("")
  const purge = action === "purge"
  const inputID = `repository-history-${action}-confirmation`
  const triggerLabel = purge ? "Purge review history" : "Remove"
  const actionLabel = purge ? "Purge review history" : "Remove repository"
  const pendingLabel = purge ? "Purging…" : "Removing…"

  const updateOpen = (nextOpen: boolean) => {
    setOpen(nextOpen)
    if (!nextOpen) setConfirmation("")
  }

  return (
    <AlertDialog open={open} onOpenChange={updateOpen}>
      <AlertDialogTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant={purge ? "outline" : "destructive"}
          disabled={disabled}
          title={disabled ? disabledReason : undefined}
        >
          <IconTrash /> {triggerLabel}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {purge
              ? `Purge review history for ${repository}?`
              : `Remove ${repository}?`}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {purge
              ? "This permanently deletes the repository-review ledger while keeping the repository configuration and run controls. Existing GitHub issues are not changed or deleted. Discussion threads and generic workflow records remain under their own retention policies."
              : "This permanently deletes the repository configuration, run controls, and repository-review ledger. Existing GitHub issues are not changed or deleted. Discussion threads and generic workflow records remain under their own retention policies."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {summary && <HistoryDeletionCounts summary={summary} />}
        <div className="space-y-2">
          <Label htmlFor={inputID}>
            Type <span className="font-mono">{repository}</span> to confirm
          </Label>
          <Input
            id={inputID}
            value={confirmation}
            autoComplete="off"
            spellCheck={false}
            aria-invalid={
              confirmation.length > 0 && confirmation !== repository
            }
            onChange={(event) => setConfirmation(event.target.value)}
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={pending || confirmation !== repository}
            onClick={onConfirm}
          >
            {pending ? pendingLabel : actionLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function HistoryDeletionCounts({
  summary,
}: {
  summary: RepositoryReviewPurgeSummary
}) {
  const counts = [
    ["Raw findings", summary.raw_findings],
    ["Deduplicated findings", summary.deduplicated_findings],
    ["Repository findings", summary.repository_findings],
    ["Issue previews", summary.issue_previews],
    ["External issue associations", summary.external_issue_associations],
  ] as const

  return (
    <dl className="border-border divide-border divide-y rounded-lg border text-sm">
      {counts.map(([label, value]) => (
        <div
          key={label}
          className="flex items-center justify-between gap-4 px-3 py-2"
        >
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="font-mono tabular-nums">{value.toLocaleString()}</dd>
        </div>
      ))}
    </dl>
  )
}

function historyMutationInput(
  detail: RepositoryReviewAutomationDetail | undefined,
) {
  if (!detail?.capabilities.purge_summary) {
    throw new Error("History deletion details are unavailable.")
  }
  return {
    expected_version: detail.automation.version,
    expected_repository_version:
      detail.capabilities.purge_summary.repository_version,
    expected_ledger_fence: detail.capabilities.purge_summary.ledger_fence,
    confirm_repository: detail.automation.repository,
  }
}

export function RepositoryReviewRepositoryEditorPage({
  automationID,
  onBack,
  onSaved,
}: {
  automationID?: string
  onBack: () => void
  onSaved: (automationID: string) => void
}) {
  const detailQuery = useQuery({
    queryKey: repositoryDetailKey(automationID ?? "new"),
    queryFn: ({ signal }) =>
      getRepositoryReviewAutomation(automationID ?? "", signal),
    enabled: Boolean(automationID),
    retry: false,
  })
  const contextQuery = useQuery({
    queryKey: editorContextKey,
    queryFn: async ({ signal }) => {
      const [profilePage, repositoryPage] = await Promise.all([
        listRepositoryReviewProfiles(signal),
        listRepositoryReviewAutomations(signal),
      ])
      return {
        profiles: profilePage.profiles,
        repositories: repositoryPage.automations,
      }
    },
    retry: false,
  })
  const notFound =
    detailQuery.error instanceof RepositoryReviewAPIError &&
    detailQuery.error.status === 404
  const loading =
    contextQuery.isLoading || (Boolean(automationID) && detailQuery.isLoading)
  const loadError =
    (!notFound ? errorMessage(detailQuery.error) : "") ||
    errorMessage(contextQuery.error)

  return (
    <CollectionDetailShell
      title={automationID ? "Edit repository" : "Add repository"}
      identity={
        automationID ? (
          <span className="font-mono text-xs">{automationID}</span>
        ) : undefined
      }
      status={
        detailQuery.data ? (
          <Badge variant="secondary">{detailQuery.data.status}</Badge>
        ) : undefined
      }
      loading={loading}
      error={loadError || undefined}
      notFound={notFound}
      onBack={onBack}
      onRetry={() =>
        void (automationID
          ? Promise.all([detailQuery.refetch(), contextQuery.refetch()])
          : contextQuery.refetch())
      }
      backLabel={
        automationID ? "Repository details" : "All review repositories"
      }
    >
      {contextQuery.data &&
        (!automationID || detailQuery.data) &&
        (contextQuery.data.profiles.length === 0 ? (
          <div
            role="status"
            className="border-border text-muted-foreground rounded-lg border border-dashed px-6 py-12 text-center text-sm"
          >
            Create a review profile before adding a repository.
          </div>
        ) : (
          <RepositoryReviewRepositoryForm
            initial={detailQuery.data}
            profiles={contextQuery.data.profiles}
            repositories={contextQuery.data.repositories}
            onCancel={onBack}
            onSaved={onSaved}
          />
        ))}
    </CollectionDetailShell>
  )
}

function RepositoryReviewRepositoryForm({
  initial,
  profiles,
  repositories,
  onCancel,
  onSaved,
}: {
  initial?: RepositoryReviewAutomation
  profiles: RepositoryReviewProfile[]
  repositories: RepositoryReviewAutomation[]
  onCancel: () => void
  onSaved: (automationID: string) => void
}) {
  const queryClient = useQueryClient()
  const [value, setValue] = useState<RepositoryEditorValue>(() => ({
    repository: initial?.repository ?? "",
    profileID: initial?.profile_id ?? profiles[0]?.id ?? "",
    branch: initial?.branch || initial?.ref || "",
  }))
  const [actionError, setActionError] = useState("")
  const duplicate = repositories.some(
    (item) =>
      item.id !== initial?.id &&
      item.repository.trim().toLowerCase() ===
        value.repository.trim().toLowerCase(),
  )
  const branchError = repositoryBranchError(value.branch)
  const configurationBusy = initial
    ? repositoryConfigurationBusy(initial)
    : false
  const saveMutation = useMutation({
    mutationFn: async () => {
      const input = {
        repository: value.repository.trim(),
        profile_id: value.profileID,
        branch: value.branch.trim(),
      }
      return initial
        ? updateRepositoryReviewAutomation(initial.id, {
            ...input,
            expected_version: initial.version,
          })
        : createRepositoryReviewAutomation(input)
    },
    onSuccess: async (saved) => {
      queryClient.setQueryData(repositoryDetailKey(saved.id), saved)
      await queryClient.invalidateQueries({ queryKey: repositoriesKey })
      onSaved(saved.id)
    },
    onError: (error) => setActionError(errorMessage(error)),
  })

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (
      saveMutation.isPending ||
      configurationBusy ||
      !value.repository.trim() ||
      !value.profileID ||
      duplicate ||
      branchError
    ) {
      return
    }
    saveMutation.mutate()
  }

  return (
    <form className="mx-auto max-w-3xl space-y-6" onSubmit={submit}>
      <div>
        <h2 className="text-base font-semibold">Repository configuration</h2>
        <p className="text-muted-foreground mt-1 text-sm">
          Assign exactly one review profile. Leave branch blank to follow the
          repository&apos;s default branch.
        </p>
      </div>
      {configurationBusy && (
        <div
          role="alert"
          className="border-border bg-muted/30 rounded-lg border px-3 py-2 text-sm"
        >
          This configuration cannot be edited while its review is{" "}
          {initial?.status}.
        </div>
      )}
      {actionError && (
        <div
          role="alert"
          className="text-destructive flex items-center gap-2 text-sm"
        >
          <IconAlertTriangle className="size-4" /> {actionError}
        </div>
      )}
      <div className="space-y-2">
        <Label htmlFor="review-repository">Repository</Label>
        <Input
          id="review-repository"
          value={value.repository}
          placeholder="owner/repository or safe Git URL"
          disabled={configurationBusy}
          onChange={(event) => {
            setActionError("")
            setValue({ ...value, repository: event.target.value })
          }}
        />
        {duplicate && (
          <p role="alert" className="text-destructive text-xs">
            This repository already has a review configuration.
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label htmlFor="review-profile">Assigned profile</Label>
        <select
          id="review-profile"
          className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
          value={value.profileID}
          disabled={configurationBusy}
          onChange={(event) => {
            setActionError("")
            setValue({ ...value, profileID: event.target.value })
          }}
        >
          <option value="">Select profile</option>
          {profiles.map((profile) => (
            <option key={profile.id} value={profile.id}>
              {profile.name} · {profile.reviewer_model}
            </option>
          ))}
        </select>
      </div>
      <ReviewAdvancedSection description="optional branch override">
        <div className="space-y-2">
          <Label htmlFor="review-branch">Branch override</Label>
          <Input
            id="review-branch"
            value={value.branch}
            placeholder="Blank uses the repository default branch"
            disabled={configurationBusy}
            onChange={(event) => {
              setActionError("")
              setValue({ ...value, branch: event.target.value })
            }}
          />
          <p className="text-muted-foreground text-xs">
            Branch names only. Repository reviews do not accept arbitrary
            targets, URLs, tags, or commit hashes here.
          </p>
          {branchError && (
            <p role="alert" className="text-destructive text-xs">
              {branchError}
            </p>
          )}
        </div>
      </ReviewAdvancedSection>
      <div className="border-border flex justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          type="submit"
          disabled={
            saveMutation.isPending ||
            configurationBusy ||
            !value.repository.trim() ||
            !value.profileID ||
            duplicate ||
            Boolean(branchError)
          }
        >
          Save repository
        </Button>
      </div>
    </form>
  )
}

function RepositoryDetailRow({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="grid gap-1 px-3 py-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:gap-4">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={mono ? "font-mono text-xs break-all" : "break-words"}>
        {value}
      </dd>
    </div>
  )
}

function repositoryBranchLabel(item: RepositoryReviewAutomation): string {
  return item.branch || item.ref || "Default repository branch"
}

function repositoryConfigurationBusy(
  item: RepositoryReviewAutomation,
): boolean {
  return item.status === "running" || item.status === "stopping"
}

function formatTimestamp(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.valueOf())
    ? value || "Not reported"
    : date.toLocaleString()
}

function errorMessage(error: unknown): string {
  if (!error) return ""
  return error instanceof Error
    ? error.message
    : "Repository configuration request failed."
}

function repositoryBranchError(value: string): string {
  if (!value.trim()) return ""
  const branch = value
  const lower = branch.toLowerCase()
  if (
    branch !== branch.trim() ||
    new TextEncoder().encode(branch).length > 255 ||
    [...branch].some(
      (character) =>
        /\s/u.test(character) ||
        character.charCodeAt(0) < 32 ||
        character.charCodeAt(0) === 127,
    ) ||
    lower === "head" ||
    lower === "@" ||
    lower.startsWith("refs/") ||
    lower.startsWith("tags/") ||
    lower.includes("://") ||
    (/^[0-9a-f]+$/i.test(branch) &&
      branch.length >= 7 &&
      branch.length <= 64) ||
    /[~^:?#*\\]/.test(branch) ||
    branch.includes("[") ||
    branch.includes("..") ||
    branch.includes("@{") ||
    branch.includes("//") ||
    branch.startsWith("-") ||
    branch.startsWith("/") ||
    branch.endsWith("/") ||
    branch.split("/").some((component) => {
      const componentLower = component.toLowerCase()
      return (
        !component ||
        component.startsWith(".") ||
        component.endsWith(".") ||
        componentLower.endsWith(".lock")
      )
    })
  ) {
    return "Enter a branch name, or leave blank for the repository default branch."
  }
  return ""
}
