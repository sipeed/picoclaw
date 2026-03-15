import { createFileRoute } from "@tanstack/react-router"

import { TaskDetailPage } from "@/components/research/task-detail-page"

export const Route = createFileRoute("/research/$taskId")({
  component: ResearchTaskRoute,
})

function ResearchTaskRoute() {
  const { taskId } = Route.useParams()
  return <TaskDetailPage taskId={taskId} />
}
