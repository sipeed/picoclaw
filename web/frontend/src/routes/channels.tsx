import { createFileRoute } from "@tanstack/react-router"

import { ChannelsPage } from "@/components/channels/channels-page"

export const Route = createFileRoute("/channels")({
  component: ChannelsPage,
})
