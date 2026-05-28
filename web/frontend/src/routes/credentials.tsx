import { createFileRoute } from "@tanstack/react-router"

import { CredentialsPage } from "@/components/credentials/credentials-page"

export const Route = createFileRoute("/credentials")({
  component: CredentialsPage,
})
