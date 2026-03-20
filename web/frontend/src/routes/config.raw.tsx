import { createFileRoute } from "@tanstack/react-router"

import { RawConfigPage } from "@/components/config/raw-config-page"

export const Route = createFileRoute("/config/raw")({
  component: RawConfigPage,
})
