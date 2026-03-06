import {
  IconBrandGoogle,
  IconLoader2,
  IconLockOpen,
  IconPlayerStopFilled,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthProviderStatus } from "@/api/oauth"
import { Button } from "@/components/ui/button"

import { CredentialCard } from "./credential-card"

interface AntigravityCredentialCardProps {
  status?: OAuthProviderStatus
  activeAction: string
  onStopLoading: () => void
  onStartBrowserOAuth: () => void
  onAskLogout: () => void
}

export function AntigravityCredentialCard({
  status,
  activeAction,
  onStopLoading,
  onStartBrowserOAuth,
  onAskLogout,
}: AntigravityCredentialCardProps) {
  const { t } = useTranslation()
  const actionBusy = activeAction !== ""
  const browserLoading = activeAction === "google-antigravity:browser"

  return (
    <CredentialCard
      title={
        <span className="inline-flex items-center gap-2">
          <span className="border-muted inline-flex size-6 items-center justify-center rounded-full border">
            <IconBrandGoogle className="size-3.5" />
          </span>
          <span>Google Antigravity</span>
        </span>
      }
      description={t(
        "credentials.providers.antigravity.description",
        "Uses browser OAuth for Google Cloud Code Assist.",
      )}
      status={status?.status ?? "not_logged_in"}
      authMethod={status?.auth_method}
      details={
        <div className="space-y-1">
          {status?.email && (
            <p>
              {t("credentials.labels.email", "Email")}: {status.email}
            </p>
          )}
          {status?.project_id && (
            <p>
              {t("credentials.labels.project", "Project")}: {status.project_id}
            </p>
          )}
        </div>
      }
      actions={
        <div className="border-muted flex h-[120px] flex-col justify-center rounded-lg border p-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={actionBusy}
              onClick={onStartBrowserOAuth}
            >
              {browserLoading && (
                <IconLoader2 className="size-4 animate-spin" />
              )}
              <IconLockOpen className="size-4" />
              {t("credentials.actions.browser", "Browser OAuth")}
            </Button>
            {browserLoading && (
              <Button
                size="icon-xs"
                variant="secondary"
                onClick={onStopLoading}
                className="text-destructive hover:bg-destructive/10 hover:text-destructive"
              >
                <IconPlayerStopFilled className="size-3" />
              </Button>
            )}
          </div>
        </div>
      }
      footer={
        status?.logged_in ? (
          <Button
            variant="ghost"
            size="sm"
            disabled={actionBusy}
            onClick={onAskLogout}
            className="text-destructive hover:bg-destructive/10 hover:text-destructive"
          >
            {activeAction === "google-antigravity:logout" && (
              <IconLoader2 className="size-4 animate-spin" />
            )}
            {t("credentials.actions.logout", "Logout")}
          </Button>
        ) : null
      }
    />
  )
}
