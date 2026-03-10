import {
  IconKey,
  IconLoader2,
  IconPlayerStopFilled,
  IconSparkles,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthProviderStatus } from "@/api/oauth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

import { CredentialCard } from "./credential-card"

interface AnthropicCredentialCardProps {
  status?: OAuthProviderStatus
  activeAction: string
  token: string
  onTokenChange: (value: string) => void
  onStopLoading: () => void
  onSaveToken: () => void
  onAskLogout: () => void
}

export function AnthropicCredentialCard({
  status,
  activeAction,
  token,
  onTokenChange,
  onStopLoading,
  onSaveToken,
  onAskLogout,
}: AnthropicCredentialCardProps) {
  const { t } = useTranslation()
  const actionBusy = activeAction !== ""
  const tokenLoading = activeAction === "anthropic:token"
  const stopLabel = t("credentials.actions.stopLoading")

  return (
    <CredentialCard
      title={
        <span className="inline-flex items-center gap-2">
          <span className="border-muted inline-flex size-6 items-center justify-center rounded-full border">
            <IconSparkles className="size-3.5" />
          </span>
          <span>Anthropic</span>
        </span>
      }
      description={t("credentials.providers.anthropic.description")}
      status={status?.status ?? "not_logged_in"}
      authMethod={status?.auth_method}
      actions={
        <div className="border-muted flex h-[120px] flex-col justify-center rounded-lg border p-3">
          <div className="flex h-full flex-col gap-3">
            <div className="flex h-full items-center gap-2">
              <Input
                value={token}
                onChange={(e) => onTokenChange(e.target.value)}
                type="password"
                placeholder={t("credentials.fields.anthropicToken")}
              />
              <Button
                size="sm"
                className="w-fit"
                disabled={actionBusy || !token.trim()}
                onClick={onSaveToken}
              >
                {tokenLoading && (
                  <IconLoader2 className="size-4 animate-spin" />
                )}
                <IconKey className="size-4" />
                {t("credentials.actions.saveToken")}
              </Button>
              {tokenLoading && (
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={onStopLoading}
                  aria-label={stopLabel}
                  title={stopLabel}
                  className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                >
                  <IconPlayerStopFilled className="size-4" />
                </Button>
              )}
            </div>
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
            {activeAction === "anthropic:logout" && (
              <IconLoader2 className="size-4 animate-spin" />
            )}
            {t("credentials.actions.logout")}
          </Button>
        ) : null
      }
    />
  )
}
