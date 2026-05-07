import { createFileRoute } from "@tanstack/react-router"

import { ToolsPage } from "@/components/agent/tools/tools-page"

export const Route = createFileRoute("/agent/tools")({
  component: AgentToolsRoute,
})

function AgentToolsRoute() {
  return <ToolsPage />
}
