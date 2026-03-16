import {
  IconArrowLeft,
  IconChevronDown,
  IconChevronRight,
  IconCircleCheck,
  IconCircleDashed,
  IconCircleX,
  IconLoader2,
  IconPlayerPlay,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ResearchDocument,
  getResearchDocContent,
  getResearchTask,
  researchTaskAction,
} from "@/api/research"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { cn } from "@/lib/utils"

const statusConfig: Record<
  string,
  { icon: React.ComponentType<{ className?: string }>; color: string }
> = {
  pending: { icon: IconCircleDashed, color: "text-yellow-600 bg-yellow-50" },
  active: { icon: IconPlayerPlay, color: "text-blue-600 bg-blue-50" },
  completed: { icon: IconCircleCheck, color: "text-green-600 bg-green-50" },
  failed: { icon: IconCircleX, color: "text-red-600 bg-red-50" },
  canceled: { icon: IconCircleX, color: "text-gray-500 bg-gray-50" },
}

export function TaskDetailPage({ taskId }: { taskId: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const {
    data: task,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["research-task", taskId],
    queryFn: () => getResearchTask(taskId),
    refetchInterval: 15000,
  })

  const actionMutation = useMutation({
    mutationFn: (action: "cancel" | "reopen") =>
      researchTaskAction(taskId, action),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["research-task", taskId],
      })
      void queryClient.invalidateQueries({ queryKey: ["research-tasks"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Action failed")
    },
  })

  const canCancel =
    task?.status === "pending" || task?.status === "active"
  const canReopen =
    task?.status === "completed" || task?.status === "failed"

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={task?.title ?? t("labels.loading")}
        titleExtra={
          task ? (
            <StatusBadge status={task.status} />
          ) : null
        }
      >
        <Link to="/research">
          <Button variant="ghost" size="sm">
            <IconArrowLeft className="size-4" />
            {t("pages.research.back")}
          </Button>
        </Link>
        {canCancel ? (
          <Button
            variant="outline"
            size="sm"
            disabled={actionMutation.isPending}
            onClick={() => actionMutation.mutate("cancel")}
          >
            {actionMutation.isPending ? (
              <IconLoader2 className="size-4 animate-spin" />
            ) : null}
            {t("pages.research.action_cancel")}
          </Button>
        ) : null}
        {canReopen ? (
          <Button
            variant="outline"
            size="sm"
            disabled={actionMutation.isPending}
            onClick={() => actionMutation.mutate("reopen")}
          >
            {actionMutation.isPending ? (
              <IconLoader2 className="size-4 animate-spin" />
            ) : null}
            {t("pages.research.action_reopen")}
          </Button>
        ) : null}
      </PageHeader>

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-4xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.research.load_error")}
            </div>
          ) : task ? (
            <>
              {task.description ? (
                <Card size="sm">
                  <CardHeader>
                    <CardTitle className="text-sm">
                      {t("pages.research.description")}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-muted-foreground whitespace-pre-wrap text-sm">
                      {task.description}
                    </p>
                  </CardContent>
                </Card>
              ) : null}

              <div className="space-y-3">
                <h3 className="text-foreground/85 text-sm font-semibold">
                  {t("pages.research.documents_title", {
                    count: task.documents.length,
                  })}
                </h3>
                {task.documents.length === 0 ? (
                  <Card className="border-dashed">
                    <CardContent className="text-muted-foreground py-8 text-center text-sm">
                      {t("pages.research.no_documents")}
                    </CardContent>
                  </Card>
                ) : (
                  <div className="space-y-2">
                    {task.documents.map((doc) => (
                      <DocumentAccordion
                        key={doc.id}
                        doc={doc}
                        taskId={taskId}
                      />
                    ))}
                  </div>
                )}
              </div>

              <div className="text-muted-foreground space-y-1 text-xs">
                <div>
                  {t("pages.research.interval")}: {task.interval || "24h"}
                  {task.last_researched_at ? (
                    <> · {t("pages.research.last_researched")}:{" "}
                    {new Date(task.last_researched_at).toLocaleString()}</>
                  ) : null}
                </div>
                <div>
                  {t("pages.research.created_at")}:{" "}
                  {new Date(task.created_at).toLocaleString()}
                </div>
                {task.completed_at ? (
                  <div>
                    {t("pages.research.completed_at")}:{" "}
                    {new Date(task.completed_at).toLocaleString()}
                  </div>
                ) : null}
                <div>
                  {t("pages.research.output_dir")}: {task.output_dir}
                </div>
              </div>
            </>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  const config = statusConfig[status] ?? statusConfig.pending
  const Icon = config.icon
  return (
    <span
      className={cn(
        "flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-semibold",
        config.color,
      )}
    >
      <Icon className="size-3.5" />
      {t(`pages.research.status.${status}`)}
    </span>
  )
}

function DocumentAccordion({
  doc,
  taskId,
}: {
  doc: ResearchDocument
  taskId: string
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = React.useState(false)

  const { data: content, isLoading } = useQuery({
    queryKey: ["research-doc", taskId, doc.id],
    queryFn: () => getResearchDocContent(taskId, doc.id),
    enabled: expanded,
  })

  return (
    <Card size="sm">
      <button
        type="button"
        className="flex w-full items-center gap-3 px-4 py-3 text-left"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? (
          <IconChevronDown className="text-muted-foreground size-4 shrink-0" />
        ) : (
          <IconChevronRight className="text-muted-foreground size-4 shrink-0" />
        )}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-xs font-mono">
              #{doc.seq}
            </span>
            <span className="text-sm font-medium">{doc.title}</span>
            <span className="text-muted-foreground rounded bg-gray-100 px-1.5 py-0.5 text-[10px]">
              {doc.doc_type}
            </span>
          </div>
          {doc.summary ? (
            <p className="text-muted-foreground mt-0.5 text-xs line-clamp-1">
              {doc.summary}
            </p>
          ) : null}
        </div>
      </button>
      {expanded ? (
        <CardContent className="border-t pt-3">
          {isLoading ? (
            <div className="text-muted-foreground py-4 text-center text-sm">
              {t("labels.loading")}
            </div>
          ) : content ? (
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap text-sm">
              {content.content}
            </pre>
          ) : null}
        </CardContent>
      ) : null}
    </Card>
  )
}
