import { createFileRoute } from "@tanstack/react-router"

import { ResearchPage } from "@/components/agent/research/research-page"

// Use type assertion to bypass route tree registration issue
// The route will be properly registered when routeTree is regenerated
export const Route = createFileRoute("/agent/research" as any)({
  component: AgentResearchRoute,
})

function AgentResearchRoute() {
  return <ResearchPage />
}