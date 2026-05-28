import { createFileRoute } from "@tanstack/react-router"

import { ChannelConfigPage } from "@/components/channels/channel-config-page"

export const Route = createFileRoute("/channels/$name")({
  component: ChannelsByNameRoute,
})

function ChannelsByNameRoute() {
  const { name } = Route.useParams()

  return <ChannelConfigPage channelName={name} />
}
