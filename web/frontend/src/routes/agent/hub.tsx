import { createFileRoute } from "@tanstack/react-router"

import { HubPage } from "@/components/agent/hub/hub-page"

export const Route = createFileRoute("/agent/hub")({
  component: AgentHubRoute,
})

function AgentHubRoute() {
  return <HubPage />
}
