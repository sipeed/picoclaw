import { createFileRoute } from "@tanstack/react-router"

import { RequestLogViewer } from "@/components/logs/request-log-viewer"

export const Route = createFileRoute("/logs/requests")({
  component: RequestLogViewer,
})
