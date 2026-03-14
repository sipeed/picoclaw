import { IconKey, IconLock, IconLockOpen, IconLoader2 } from "@tabler/icons-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { getPassphraseStatus, setPassphrase } from "@/api/passphrase"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export function PassphraseCard() {
  const { t } = useTranslation()
  const [value, setValue] = useState("")
  const [isSet, setIsSet] = useState<boolean | null>(null)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ text: string; error: boolean } | null>(null)

  useEffect(() => {
    getPassphraseStatus()
      .then((res) => setIsSet(res.passphrase_set))
      .catch(() => setIsSet(false))
  }, [])

  async function handleSave() {
    if (!value.trim()) {
      setMessage({ text: t("credentials.passphrase.errorEmpty"), error: true })
      return
    }
    setSaving(true)
    setMessage(null)
    try {
      await setPassphrase(value.trim())
      setIsSet(true)
      setValue("")
      setMessage({ text: t("credentials.passphrase.successMessage"), error: false })
    } catch {
      setMessage({ text: t("credentials.passphrase.errorSave"), error: true })
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="bg-card flex h-full flex-col rounded-xl border p-4">
      <div className="min-h-16">
        <h3 className="text-base font-semibold inline-flex items-center gap-2">
          <span className="border-muted inline-flex size-6 items-center justify-center rounded-full border">
            <IconKey className="size-3.5" />
          </span>
          {t("credentials.passphrase.title")}
        </h3>
        <p className="text-muted-foreground mt-1 text-xs">
          {t("credentials.passphrase.description")}
        </p>
      </div>

      <div className="mt-3 flex items-center gap-2 text-xs">
        {isSet === null ? (
          <IconLoader2 className="size-3.5 animate-spin text-muted-foreground" />
        ) : isSet ? (
          <>
            <IconLockOpen className="size-3.5 text-green-500" />
            <span className="text-green-600 dark:text-green-400">
              {t("credentials.passphrase.statusSet")}
            </span>
          </>
        ) : (
          <>
            <IconLock className="size-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">
              {t("credentials.passphrase.statusNotSet")}
            </span>
          </>
        )}
      </div>

      <div className="mt-auto flex flex-col gap-4 pt-4">
        <div className="border-muted flex h-[120px] flex-col justify-center rounded-lg border p-3">
          <div className="flex h-full flex-col gap-3">
            <div className="flex h-full items-center gap-2">
              <Input
                value={value}
                onChange={(e) => setValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void handleSave()
                }}
                type="password"
                placeholder={t("credentials.passphrase.placeholder")}
                disabled={saving}
              />
              <Button
                size="sm"
                className="w-fit"
                disabled={saving || !value.trim()}
                onClick={() => void handleSave()}
              >
                {saving && <IconLoader2 className="size-4 animate-spin" />}
                {saving
                  ? t("credentials.passphrase.saving")
                  : t("credentials.passphrase.save")}
              </Button>
            </div>
            {message && (
              <p
                className={`text-xs ${message.error ? "text-destructive" : "text-green-600 dark:text-green-400"}`}
              >
                {message.text}
              </p>
            )}
          </div>
        </div>
        <div className="min-h-8" />
      </div>
    </section>
  )
}
