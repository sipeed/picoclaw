import { useTranslation } from "react-i18next"

import type { OAuthProviderStatus } from "@/api/oauth"

interface ProviderStatusLineProps {
  status: OAuthProviderStatus["status"]
  authMethod?: string
}

export function ProviderStatusLine({
  status,
  authMethod,
}: ProviderStatusLineProps) {
  const { t } = useTranslation()

  const style =
    status === "connected"
      ? "bg-green-500/10 text-green-700 dark:text-green-300"
      : status === "needs_refresh"
        ? "bg-amber-500/10 text-amber-700 dark:text-amber-300"
        : status === "expired"
          ? "bg-red-500/10 text-red-700 dark:text-red-300"
          : "bg-muted text-muted-foreground"

  return (
    <div className="flex items-center justify-between gap-2">
      <span className={`rounded px-2 py-1 text-xs font-medium ${style}`}>
        {status === "connected"
          ? t("credentials.status.connected")
          : status === "needs_refresh"
            ? t("credentials.status.needsRefresh")
            : status === "expired"
              ? t("credentials.status.expired")
              : t("credentials.status.notLoggedIn")}
      </span>
      {authMethod && (
        <span className="text-muted-foreground text-xs uppercase">
          {authMethod}
        </span>
      )}
    </div>
  )
}
