import { useQuery } from "@tanstack/react-query"
import {
  IconAlertTriangle,
  IconRefresh,
  IconSubtask,
} from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { getTasks, type TaskRecord } from "@/api/tasks"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

const taskKindOptions = ["all", "delegate", "spawn"]

function formatTaskTime(ms?: number) {
  if (!ms) {
    return "n/a"
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "medium",
  }).format(new Date(ms))
}

function truncate(text: string | undefined, max: number) {
  const value = (text ?? "").trim()
  if (value.length <= max) {
    return value
  }
  return `${value.slice(0, max)}...`
}

function statusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "succeeded":
      return "default"
    case "failed":
    case "timed_out":
    case "cancelled":
    case "lost":
      return "destructive"
    case "running":
    case "queued":
    case "planned":
      return "secondary"
    default:
      return "outline"
  }
}

function taskScope(task: TaskRecord) {
  const parts = [task.channel, task.chat_id].filter(Boolean).join("/")
  if (task.topic_id) {
    return `${parts} topic=${task.topic_id}`
  }
  return parts || "unscoped"
}

function TaskCard({ task }: { task: TaskRecord }) {
  const result = task.error || task.terminal_summary || task.progress_summary
  const deliverable = task.deliverable
  const artifactCount = deliverable?.artifacts?.length ?? 0
  const legacyCompletion = task.completion

  return (
    <Card size="sm" className="border-border/70 bg-card/70">
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle className="font-mono text-sm">{task.task_id}</CardTitle>
          <Badge variant={statusVariant(task.status)}>{task.status}</Badge>
          <Badge variant="outline">
            {task.runtime}
            {task.task_kind ? `/${task.task_kind}` : ""}
          </Badge>
          {task.delivery_mode && (
            <Badge variant="secondary">{task.delivery_mode}</Badge>
          )}
        </div>
        <CardDescription>
          {task.agent_id ? `agent=${task.agent_id} · ` : ""}
          {taskScope(task)}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {task.task && (
          <p className="text-foreground/90 text-sm">{truncate(task.task, 280)}</p>
        )}
        {result && (
          <div className="bg-muted/40 rounded-lg p-3 text-sm">
            <div className="text-muted-foreground mb-1 text-xs font-medium">
              Result
            </div>
            <p className="whitespace-pre-wrap">{truncate(result, 500)}</p>
          </div>
        )}
        <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
          <span>created {formatTaskTime(task.created_at)}</span>
          {task.ended_at ? <span>ended {formatTaskTime(task.ended_at)}</span> : null}
          <span>delivery {task.delivery_status}</span>
          {deliverable?.text ? <span>deliverable text</span> : null}
          {artifactCount > 0 ? <span>{artifactCount} artifacts</span> : null}
          {!deliverable && legacyCompletion?.text ? (
            <span>legacy completion text</span>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

export function TasksPage() {
  const { t } = useTranslation()
  const [taskKind, setTaskKind] = useState("all")
  const tasksQuery = useQuery({
    queryKey: ["tasks", taskKind],
    queryFn: () =>
      getTasks({
        limit: 100,
        taskKind: taskKind === "all" ? undefined : taskKind,
      }),
    refetchInterval: 5000,
  })

  const tasks = tasksQuery.data?.tasks ?? []
  const counts = tasksQuery.data?.counts ?? {}

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title={t("navigation.tasks", "Tasks")}>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void tasksQuery.refetch()}
          disabled={tasksQuery.isFetching}
        >
          <IconRefresh className={cn("size-4", tasksQuery.isFetching && "animate-spin")} />
          Refresh
        </Button>
      </PageHeader>

      <div className="flex-1 overflow-auto px-6 py-6 pb-20">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-5">
          <div className="flex flex-col gap-3 rounded-xl border border-border/70 bg-card/60 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div className="flex items-center gap-2 text-sm font-medium">
                <IconSubtask className="size-4" />
                Durable task registry
              </div>
              <p className="text-muted-foreground mt-1 text-sm">
                Recent spawn/delegate task records from the current workspace.
              </p>
              {tasksQuery.data?.store_path ? (
                <p className="text-muted-foreground mt-1 font-mono text-xs">
                  {tasksQuery.data.store_path}
                </p>
              ) : null}
            </div>
            <div className="flex items-center gap-3">
              <Select value={taskKind} onValueChange={setTaskKind}>
                <SelectTrigger className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {taskKindOptions.map((option) => (
                    <SelectItem key={option} value={option}>
                      {option === "all" ? "All tasks" : option}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-5">
            {["running", "queued", "planned", "succeeded", "failed"].map((status) => (
              <Card key={status} size="sm" className="bg-card/60">
                <CardContent>
                  <div className="text-muted-foreground text-xs uppercase tracking-wide">
                    {status}
                  </div>
                  <div className="mt-1 text-2xl font-semibold">
                    {counts[status] ?? 0}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {tasksQuery.isError ? (
            <Card className="border-destructive/30 bg-destructive/5">
              <CardContent className="flex items-center gap-3">
                <IconAlertTriangle className="text-destructive size-5" />
                <span>
                  {tasksQuery.error instanceof Error
                    ? tasksQuery.error.message
                    : "Failed to load tasks"}
                </span>
              </CardContent>
            </Card>
          ) : null}

          {!tasksQuery.isLoading && tasks.length === 0 ? (
            <Card>
              <CardContent className="text-muted-foreground py-10 text-center">
                No task records found.
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-3">
              {tasks.map((task) => (
                <TaskCard key={task.task_id} task={task} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
