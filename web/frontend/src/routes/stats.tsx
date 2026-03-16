import { createFileRoute } from "@tanstack/react-router"

import { StatsPage } from "@/components/stats/stats-page"

export const Route = createFileRoute("/stats")({
  component: StatsPage,
})
