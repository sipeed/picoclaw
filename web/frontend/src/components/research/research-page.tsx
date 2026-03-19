import {
  IconCircleCheck,
  IconCircleDashed,
  IconCircleX,
  IconLoader2,
  IconPlayerPlay,
  IconPlus,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ResearchTask,
  createResearchTask,
  getResearchTasks,
} from "@/api/research"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

const statusConfig: Record<
  ResearchTask["status"],
  { icon: React.ComponentType<{ className?: string }>; color: string }
> = {
  pending: { icon: IconCircleDashed, color: "text-yellow-600 bg-yellow-50" },
  active: { icon: IconPlayerPlay, color: "text-blue-600 bg-blue-50" },
  completed: { icon: IconCircleCheck, color: "text-green-600 bg-green-50" },
  failed: { icon: IconCircleX, color: "text-red-600 bg-red-50" },
  canceled: { icon: IconCircleX, color: "text-gray-500 bg-gray-50" },
}

export function ResearchPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [sheetOpen, setSheetOpen] = React.useState(false)
  const [title, setTitle] = React.useState("")
  const [description, setDescription] = React.useState("")

  const { data: tasks, isLoading, error } = useQuery({
    queryKey: ["research-tasks"],
    queryFn: () => getResearchTasks(),
    refetchInterval: 30000,
  })

  const createMutation = useMutation({
    mutationFn: () => createResearchTask(title.trim(), description.trim()),
    onSuccess: () => {
      toast.success(t("pages.research.create_success"))
      setSheetOpen(false)
      setTitle("")
      setDescription("")
      void queryClient.invalidateQueries({ queryKey: ["research-tasks"] })
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.research.create_error"),
      )
    },
  })

  return (
    <div className="flex h-full flex-col">
      <div className="flex justify-end px-6 pt-2">
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button size="sm">
              <IconPlus className="size-4" />
              {t("pages.research.new_task")}
            </Button>
          </SheetTrigger>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>{t("pages.research.new_task")}</SheetTitle>
              <SheetDescription>
                {t("pages.research.new_task_description")}
              </SheetDescription>
            </SheetHeader>
            <form
              className="mt-6 space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                if (title.trim()) createMutation.mutate()
              }}
            >
              <div className="space-y-2">
                <Label htmlFor="research-title">
                  {t("pages.research.field_title")}
                </Label>
                <Input
                  id="research-title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t("pages.research.field_title_placeholder")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="research-desc">
                  {t("pages.research.field_description")}
                </Label>
                <Textarea
                  id="research-desc"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder={t(
                    "pages.research.field_description_placeholder",
                  )}
                  rows={4}
                />
              </div>
              <Button
                type="submit"
                disabled={!title.trim() || createMutation.isPending}
                className="w-full"
              >
                {createMutation.isPending ? (
                  <IconLoader2 className="size-4 animate-spin" />
                ) : null}
                {t("pages.research.create")}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-4">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.research.load_error")}
            </div>
          ) : !tasks?.length ? (
            <Card className="border-dashed">
              <CardContent className="text-muted-foreground py-10 text-center text-sm">
                {t("pages.research.empty")}
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 lg:grid-cols-2">
              {tasks.map((task) => (
                <TaskCard key={task.id} task={task} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function TaskCard({ task }: { task: ResearchTask }) {
  const { t } = useTranslation()
  const config = statusConfig[task.status]
  const StatusIcon = config.icon

  return (
    <Link to="/research/$taskId" params={{ taskId: task.id }}>
      <Card
        className={cn(
          "cursor-pointer gap-3 border transition-colors hover:shadow-sm",
          task.status === "active" && "border-blue-200/70",
          task.status === "completed" && "border-emerald-200/70",
          task.status === "failed" && "border-red-200/70",
        )}
        size="sm"
      >
        <CardHeader>
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              <CardTitle className="text-sm">{task.title}</CardTitle>
              {task.description ? (
                <CardDescription className="mt-1 line-clamp-2">
                  {task.description}
                </CardDescription>
              ) : null}
            </div>
            <span
              className={cn(
                "flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-[11px] font-semibold",
                config.color,
              )}
            >
              <StatusIcon className="size-3.5" />
              {t(`pages.research.status.${task.status}`)}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-muted-foreground flex items-center gap-3 text-xs">
            <span>
              {t("pages.research.documents_count", {
                count: task.document_count,
              })}
            </span>
            <span>⏱ {task.interval || "24h"}</span>
            {task.last_researched_at ? (
              <span>
                {t("pages.research.last_researched")}:{" "}
                {new Date(task.last_researched_at).toLocaleDateString()}
              </span>
            ) : null}
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
