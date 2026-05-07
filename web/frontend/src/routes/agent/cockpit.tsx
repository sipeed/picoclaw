import { createFileRoute } from "@tanstack/react-router"

import { CockpitPage } from "@/components/agent/cockpit/cockpit-page"

export const Route = createFileRoute("/agent/cockpit")({
  component: AgentCockpitRoute,
})

function AgentCockpitRoute() {
  return <CockpitPage />
}
