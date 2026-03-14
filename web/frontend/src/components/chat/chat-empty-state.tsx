import {
  IconKey,
  IconLoader2,
  IconLock,
  IconPlugConnectedX,
  IconRobot,
  IconRobotOff,
  IconStar,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { setPassphrase } from "@/api/passphrase"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

interface ChatEmptyStateProps {
  hasConfiguredModels: boolean
  defaultModelName: string
  isConnected: boolean
  gatewayStartReason?: string
  passphraseState?: "" | "pending" | "failed"
}

export function ChatEmptyState({
  hasConfiguredModels,
  defaultModelName,
  isConnected,
  gatewayStartReason = "",
  passphraseState = "",
}: ChatEmptyStateProps) {
  const { t } = useTranslation()
  const needsPassphrase = gatewayStartReason.toLowerCase().includes("passphrase")
    || passphraseState === "failed"
    || passphraseState === "pending"

  const [passphrase, setPassphraseValue] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [saved, setSaved] = useState(false)

  // When backend signals failure, reset the saved flag and show error.
  useEffect(() => {
    if (passphraseState === "failed") {
      setSaved(false)
      setError(t("credentials.passphrase.errorWrongPassphrase"))
    }
  }, [passphraseState, t])

  async function handleUnlock() {
    if (!passphrase.trim()) return
    setSaving(true)
    setError("")
    try {
      await setPassphrase(passphrase.trim())
      setSaved(true)
      setPassphraseValue("")
    } catch {
      setError(t("credentials.passphrase.errorSave"))
    } finally {
      setSaving(false)
    }
  }

  // Passphrase unlock takes priority — models/config can't load without it
  if (!isConnected && needsPassphrase) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-blue-500/10 text-blue-500">
          <IconLock className="h-8 w-8" />
        </div>
        <h3 className="mb-2 text-xl font-medium">
          {t("credentials.passphrase.title")}
        </h3>
        <p className="text-muted-foreground mb-6 text-center text-sm">
          {t("credentials.passphrase.description")}
        </p>
        {saved ? (
          <p className="text-green-600 dark:text-green-400 text-sm">
            {t("credentials.passphrase.successMessage")}
          </p>
        ) : (
          <div className="flex w-full max-w-sm flex-col gap-2">
            <div className="flex gap-2">
              <Input
                type="password"
                placeholder={t("credentials.passphrase.placeholder")}
                value={passphrase}
                onChange={(e) => setPassphraseValue(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") void handleUnlock() }}
                disabled={saving}
                autoFocus
              />
              <Button
                size="sm"
                disabled={saving || !passphrase.trim()}
                onClick={() => void handleUnlock()}
              >
                {saving
                  ? <IconLoader2 className="size-4 animate-spin" />
                  : <IconKey className="size-4" />}
                {saving
                  ? t("credentials.passphrase.saving")
                  : t("credentials.passphrase.save")}
              </Button>
            </div>
            {error && (
              <p className="text-destructive text-xs">{error}</p>
            )}
          </div>
        )}
      </div>
    )
  }

  if (!hasConfiguredModels) {
    return (
      <div className="flex flex-col items-center justify-center py-20 opacity-70">
        <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500">
          <IconRobotOff className="h-8 w-8" />
        </div>
        <h3 className="mb-2 text-xl font-medium">
          {t("chat.empty.noConfiguredModel")}
        </h3>
        <p className="text-muted-foreground mb-4 text-center text-sm">
          {t("chat.empty.noConfiguredModelDescription")}
        </p>
        <Button asChild variant="secondary" size="sm" className="px-4">
          <Link to="/models">{t("chat.empty.goToModels")}</Link>
        </Button>
      </div>
    )
  }

  if (!defaultModelName) {
    return (
      <div className="flex flex-col items-center justify-center py-20 opacity-70">
        <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500">
          <IconStar className="h-8 w-8" />
        </div>
        <h3 className="mb-2 text-xl font-medium">
          {t("chat.empty.noSelectedModel")}
        </h3>
        <p className="text-muted-foreground mb-4 text-center text-sm">
          {t("chat.empty.noSelectedModelDescription")}
        </p>
      </div>
    )
  }

  if (!isConnected) {
    return (
      <div className="flex flex-col items-center justify-center py-20 opacity-70">
        <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500">
          <IconPlugConnectedX className="h-8 w-8" />
        </div>
        <h3 className="mb-2 text-xl font-medium">
          {t("chat.empty.notRunning")}
        </h3>
        <p className="text-muted-foreground mb-4 text-center text-sm">
          {t("chat.empty.notRunningDescription")}
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col items-center justify-center py-20 opacity-70">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-violet-500/10 text-violet-500">
        <IconRobot className="h-8 w-8" />
      </div>
      <h3 className="mb-2 text-xl font-medium">{t("chat.welcome")}</h3>
      <p className="text-muted-foreground text-center text-sm">
        {t("chat.welcomeDesc")}
      </p>
    </div>
  )
}

