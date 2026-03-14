import { createFileRoute } from "@tanstack/react-router"

import { LogSettingsPanel } from "@/components/logs/log-settings-panel"

export const Route = createFileRoute("/config/logs")({
  component: LogSettingsPanel,
})
