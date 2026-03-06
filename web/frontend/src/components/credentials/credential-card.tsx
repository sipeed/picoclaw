import type { ReactNode } from "react"

import type { OAuthProviderStatus } from "@/api/oauth"

import { ProviderStatusLine } from "./provider-status-line"

interface CredentialCardProps {
  title: ReactNode
  description: string
  status: OAuthProviderStatus["status"]
  authMethod?: string
  details?: ReactNode
  actions: ReactNode
  footer?: ReactNode
}

export function CredentialCard({
  title,
  description,
  status,
  authMethod,
  details,
  actions,
  footer,
}: CredentialCardProps) {
  return (
    <section className="bg-card flex h-full flex-col rounded-xl border p-4">
      <div className="min-h-16">
        <h3 className="text-base font-semibold">{title}</h3>
        <p className="text-muted-foreground mt-1 text-xs">{description}</p>
      </div>

      <ProviderStatusLine status={status} authMethod={authMethod} />
      <div className="text-muted-foreground mt-3 min-h-11 text-xs leading-5">
        {details}
      </div>

      <div className="mt-auto flex flex-col gap-4 pt-4">
        <div className="min-h-[112px]">{actions}</div>
        <div className="min-h-8">{footer}</div>
      </div>
    </section>
  )
}
