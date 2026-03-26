import {
  IconCheck,
  IconLoader2,
  IconQrcode,
  IconRefresh,
  IconX,
} from "@tabler/icons-react"
import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { patchAppConfig, pollWecomFlow, startWecomFlow } from "@/api/channels"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"

type BindingState =
  | "idle"
  | "loading"
  | "waiting"
  | "scaned"
  | "confirmed"
  | "expired"
  | "error"

interface WecomFormProps {
  config: ChannelConfig
  isEdit: boolean
  onBindSuccess?: () => void
  onEnabledChange?: (enabled: boolean) => void
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

export function WecomForm({
  config,
  isEdit,
  onBindSuccess,
  onEnabledChange,
}: WecomFormProps) {
  const { t } = useTranslation()

  const [bindState, setBindState] = useState<BindingState>("idle")
  const [qrDataURI, setQrDataURI] = useState<string | null>(null)
  const [botID, setBotID] = useState<string | null>(null)
  const [errorMsg, setErrorMsg] = useState("")
  const [enabled, setEnabled] = useState(config.enabled === true)
  const [toggleSaving, setToggleSaving] = useState(false)
  const [toggleError, setToggleError] = useState("")

  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollGenerationRef = useRef(0)
  const existingBotID = asString(config.bot_id)
  const isBound = isEdit && existingBotID !== ""

  const stopPolling = useCallback(() => {
    pollGenerationRef.current += 1
    if (pollTimerRef.current !== null) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }, [])

  useEffect(() => () => stopPolling(), [stopPolling])

  useEffect(() => {
    setEnabled(config.enabled === true)
  }, [config.enabled])

  useEffect(() => {
    if (!existingBotID) return
    stopPolling()
    setBotID(existingBotID)
    setBindState("confirmed")
    setErrorMsg("")
  }, [existingBotID, stopPolling])

  const startPolling = useCallback(
    (id: string) => {
      stopPolling()
      const generation = pollGenerationRef.current
      let inFlight = false
      pollTimerRef.current = setInterval(async () => {
        if (inFlight) return
        inFlight = true
        try {
          const resp = await pollWecomFlow(id)
          if (generation !== pollGenerationRef.current) {
            return
          }
          if (resp.status === "scaned") {
            setBindState("scaned")
          } else if (resp.status === "confirmed") {
            stopPolling()
            setBotID(resp.bot_id ?? existingBotID ?? null)
            setBindState("confirmed")
            onBindSuccess?.()
          } else if (resp.status === "expired") {
            stopPolling()
            setBindState("expired")
          } else if (resp.status === "error") {
            stopPolling()
            setBindState("error")
            setErrorMsg(resp.error ?? t("channels.wecom.errorGeneric"))
          }
        } catch {
          // transient network error — keep polling
        } finally {
          inFlight = false
        }
      }, 2000)
    },
    [existingBotID, onBindSuccess, stopPolling, t],
  )

  const handleEnabledChange = useCallback(
    async (checked: boolean) => {
      if (!existingBotID || toggleSaving) {
        return
      }
      setToggleSaving(true)
      setToggleError("")
      try {
        await patchAppConfig({
          channels: {
            wecom: {
              enabled: checked,
            },
          },
        })
        setEnabled(checked)
        onEnabledChange?.(checked)
      } catch (e) {
        setToggleError(
          e instanceof Error ? e.message : t("channels.wecom.errorGeneric"),
        )
      } finally {
        setToggleSaving(false)
      }
    },
    [existingBotID, onEnabledChange, t, toggleSaving],
  )

  const handleBind = async () => {
    setBindState("loading")
    setErrorMsg("")
    setToggleError("")
    setQrDataURI(null)
    stopPolling()
    try {
      const resp = await startWecomFlow()
      setQrDataURI(resp.qr_data_uri ?? null)
      setBindState("waiting")
      startPolling(resp.flow_id)
    } catch (e) {
      setBindState("error")
      setErrorMsg(
        e instanceof Error ? e.message : t("channels.wecom.errorGeneric"),
      )
    }
  }

  const handleRebind = () => {
    stopPolling()
    setBindState("idle")
    setQrDataURI(null)
    setBotID(null)
    setErrorMsg("")
    void handleBind()
  }

  const renderBindSection = () => {
    if (bindState === "idle") {
      if (isBound) {
        return (
          <div className="flex flex-col items-center gap-3 py-6">
            <div className="flex items-center gap-2 rounded-full bg-emerald-500/10 px-4 py-2 text-sm font-medium text-emerald-600 dark:text-emerald-400">
              <IconCheck size={16} />
              {t("channels.wecom.bound")}
            </div>
            {existingBotID && (
              <p className="text-muted-foreground font-mono text-xs">
                {existingBotID}
              </p>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={handleRebind}
              className="mt-1 gap-2"
            >
              <IconRefresh size={14} />
              {t("channels.wecom.rebind")}
            </Button>
          </div>
        )
      }
      return (
        <div className="flex flex-col items-center gap-4 py-6">
          <p className="text-muted-foreground text-sm">
            {t("channels.wecom.notBound")}
          </p>
          <Button onClick={handleBind} className="gap-2">
            <IconQrcode size={16} />
            {t("channels.wecom.bind")}
          </Button>
        </div>
      )
    }

    if (bindState === "loading") {
      return (
        <div className="flex flex-col items-center gap-3 py-8">
          <IconLoader2
            className="text-muted-foreground animate-spin"
            size={32}
          />
          <p className="text-muted-foreground text-sm">
            {t("channels.wecom.generating")}
          </p>
        </div>
      )
    }

    if (bindState === "waiting" || bindState === "scaned") {
      return (
        <div className="flex flex-col items-center gap-4 py-4">
          {qrDataURI ? (
            <img
              src={qrDataURI}
              alt="WeCom QR Code"
              className="border-border/60 h-48 w-48 rounded-xl border bg-white p-2 shadow-sm"
            />
          ) : (
            <div className="border-border/60 bg-muted flex h-48 w-48 items-center justify-center rounded-xl border">
              <IconLoader2
                className="text-muted-foreground animate-spin"
                size={32}
              />
            </div>
          )}
          {bindState === "scaned" ? (
            <div className="flex items-center gap-2 rounded-full bg-amber-500/10 px-4 py-2 text-sm font-medium text-amber-600 dark:text-amber-400">
              <IconLoader2 size={14} className="animate-spin" />
              {t("channels.wecom.scanned")}
            </div>
          ) : (
            <p className="text-muted-foreground text-sm">
              {t("channels.wecom.scanHint")}
            </p>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRebind}
            className="text-muted-foreground"
          >
            <IconRefresh size={14} className="mr-1" />
            {t("channels.wecom.refresh")}
          </Button>
        </div>
      )
    }

    if (bindState === "confirmed") {
      return (
        <div className="flex flex-col items-center gap-3 py-6">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500/10">
            <IconCheck
              size={28}
              className="text-emerald-600 dark:text-emerald-400"
            />
          </div>
          <p className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
            {t("channels.wecom.bound")}
          </p>
          {botID && (
            <p className="text-muted-foreground font-mono text-xs">{botID}</p>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRebind}
            className="mt-1 gap-2"
          >
            <IconRefresh size={14} />
            {t("channels.wecom.rebind")}
          </Button>
        </div>
      )
    }

    if (bindState === "expired") {
      return (
        <div className="flex flex-col items-center gap-4 py-6">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-amber-500/10">
            <IconX size={28} className="text-amber-600 dark:text-amber-400" />
          </div>
          <p className="text-sm text-amber-600 dark:text-amber-400">
            {t("channels.wecom.expired")}
          </p>
          <Button onClick={handleRebind} className="gap-2">
            <IconRefresh size={14} />
            {t("channels.wecom.retry")}
          </Button>
        </div>
      )
    }

    if (bindState === "error") {
      return (
        <div className="flex flex-col items-center gap-4 py-6">
          <div className="bg-destructive/10 flex h-14 w-14 items-center justify-center rounded-full">
            <IconX size={28} className="text-destructive" />
          </div>
          <p className="text-destructive text-sm">
            {errorMsg || t("channels.wecom.errorGeneric")}
          </p>
          <Button variant="outline" onClick={handleRebind} className="gap-2">
            <IconRefresh size={14} />
            {t("channels.wecom.retry")}
          </Button>
        </div>
      )
    }

    return null
  }

  return (
    <div className="space-y-5">
      <div className="border-border/60 bg-background rounded-lg border px-4 py-3">
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-sm font-medium">
              {t("channels.page.enableLabel")}
            </p>
            <p className="text-muted-foreground mt-0.5 text-xs">
              {isBound
                ? t("channels.wecom.enableDesc")
                : t("channels.wecom.enableBindFirst")}
            </p>
          </div>
          <Switch
            checked={enabled}
            disabled={!isBound || toggleSaving}
            onCheckedChange={(checked) => void handleEnabledChange(checked)}
          />
        </div>
        {toggleError && (
          <p className="text-destructive mt-2 text-sm">{toggleError}</p>
        )}
      </div>

      <div className="border-border/60 bg-muted/30 rounded-xl border">
        <div className="border-border/60 border-b px-4 py-3">
          <p className="text-sm font-medium">{t("channels.wecom.bindTitle")}</p>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {t("channels.wecom.bindDesc")}
          </p>
        </div>
        {renderBindSection()}
      </div>
    </div>
  )
}
