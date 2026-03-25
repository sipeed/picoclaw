import { IconLoader2, IconRefresh, IconCheck, IconX, IconQrcode } from "@tabler/icons-react"
import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { pollWeixinFlow, startWeixinFlow } from "@/api/channels"
import { Field } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

type BindingState = "idle" | "loading" | "waiting" | "scaned" | "confirmed" | "expired" | "error"

interface WeixinFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  isEdit: boolean
  onBindSuccess?: () => void
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === "string")
}

export function WeixinForm({ config, onChange, isEdit, onBindSuccess }: WeixinFormProps) {
  const { t } = useTranslation()

  const [bindState, setBindState] = useState<BindingState>("idle")
  const [qrDataURI, setQrDataURI] = useState<string | null>(null)
  const [accountID, setAccountID] = useState<string | null>(null)
  const [errorMsg, setErrorMsg] = useState("")

  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const isBound = isEdit && asString(config.account_id) !== ""
  const existingAccountID = asString(config.account_id)

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current !== null) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }, [])

  useEffect(() => () => stopPolling(), [stopPolling])

  const startPolling = useCallback(
    (id: string) => {
      stopPolling()
      pollTimerRef.current = setInterval(async () => {
        try {
          const resp = await pollWeixinFlow(id)
          if (resp.status === "scaned") {
            setBindState("scaned")
          } else if (resp.status === "confirmed") {
            stopPolling()
            setAccountID(resp.account_id ?? null)
            setBindState("confirmed")
            onBindSuccess?.()
          } else if (resp.status === "expired") {
            stopPolling()
            setBindState("expired")
          } else if (resp.status === "error") {
            stopPolling()
            setBindState("error")
            setErrorMsg(resp.error ?? t("channels.weixin.errorGeneric"))
          }
        } catch {
          // transient network error — keep polling
        }
      }, 2000)
    },
    [stopPolling, onBindSuccess, t],
  )

  const handleBind = async () => {
    setBindState("loading")
    setErrorMsg("")
    setQrDataURI(null)
    stopPolling()
    try {
      const resp = await startWeixinFlow()
      setQrDataURI(resp.qr_data_uri ?? null)
      setBindState("waiting")
      startPolling(resp.flow_id)
    } catch (e) {
      setBindState("error")
      setErrorMsg(e instanceof Error ? e.message : t("channels.weixin.errorGeneric"))
    }
  }

  const handleRebind = () => {
    stopPolling()
    setBindState("idle")
    setQrDataURI(null)
    setAccountID(null)
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
              {t("channels.weixin.bound")}
            </div>
            {existingAccountID && (
              <p className="text-xs text-muted-foreground font-mono">{existingAccountID}</p>
            )}
            <Button variant="outline" size="sm" onClick={handleRebind} className="mt-1 gap-2">
              <IconRefresh size={14} />
              {t("channels.weixin.rebind")}
            </Button>
          </div>
        )
      }
      return (
        <div className="flex flex-col items-center gap-4 py-6">
          <p className="text-sm text-muted-foreground">{t("channels.weixin.notBound")}</p>
          <Button onClick={handleBind} className="gap-2">
            <IconQrcode size={16} />
            {t("channels.weixin.bind")}
          </Button>
        </div>
      )
    }

    if (bindState === "loading") {
      return (
        <div className="flex flex-col items-center gap-3 py-8">
          <IconLoader2 className="animate-spin text-muted-foreground" size={32} />
          <p className="text-sm text-muted-foreground">{t("channels.weixin.generating")}</p>
        </div>
      )
    }

    if (bindState === "waiting" || bindState === "scaned") {
      return (
        <div className="flex flex-col items-center gap-4 py-4">
          {qrDataURI ? (
            <img
              src={qrDataURI}
              alt="WeChat QR Code"
              className="h-48 w-48 rounded-xl border border-border/60 bg-white p-2 shadow-sm"
            />
          ) : (
            <div className="flex h-48 w-48 items-center justify-center rounded-xl border border-border/60 bg-muted">
              <IconLoader2 className="animate-spin text-muted-foreground" size={32} />
            </div>
          )}
          {bindState === "scaned" ? (
            <div className="flex items-center gap-2 rounded-full bg-amber-500/10 px-4 py-2 text-sm font-medium text-amber-600 dark:text-amber-400">
              <IconLoader2 size={14} className="animate-spin" />
              {t("channels.weixin.scanned")}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{t("channels.weixin.scanHint")}</p>
          )}
          <Button variant="ghost" size="sm" onClick={handleRebind} className="text-muted-foreground">
            <IconRefresh size={14} className="mr-1" />
            {t("channels.weixin.refresh")}
          </Button>
        </div>
      )
    }

    if (bindState === "confirmed") {
      return (
        <div className="flex flex-col items-center gap-3 py-6">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500/10">
            <IconCheck size={28} className="text-emerald-600 dark:text-emerald-400" />
          </div>
          <p className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
            {t("channels.weixin.bound")}
          </p>
          {accountID && (
            <p className="text-xs text-muted-foreground font-mono">{accountID}</p>
          )}
          <Button variant="outline" size="sm" onClick={handleRebind} className="mt-1 gap-2">
            <IconRefresh size={14} />
            {t("channels.weixin.rebind")}
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
          <p className="text-sm text-amber-600 dark:text-amber-400">{t("channels.weixin.expired")}</p>
          <Button onClick={handleRebind} className="gap-2">
            <IconRefresh size={14} />
            {t("channels.weixin.retry")}
          </Button>
        </div>
      )
    }

    if (bindState === "error") {
      return (
        <div className="flex flex-col items-center gap-4 py-6">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
            <IconX size={28} className="text-destructive" />
          </div>
          <p className="text-sm text-destructive">{errorMsg || t("channels.weixin.errorGeneric")}</p>
          <Button variant="outline" onClick={handleRebind} className="gap-2">
            <IconRefresh size={14} />
            {t("channels.weixin.retry")}
          </Button>
        </div>
      )
    }

    return null
  }

  return (
    <div className="space-y-5">
      {/* QR Bind Section */}
      <div className="rounded-xl border border-border/60 bg-muted/30">
        <div className="border-b border-border/60 px-4 py-3">
          <p className="text-sm font-medium">{t("channels.weixin.bindTitle")}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">{t("channels.weixin.bindDesc")}</p>
        </div>
        {renderBindSection()}
      </div>

      {/* allow_from */}
      <Field
        label={t("channels.field.allowFrom")}
        hint={t("channels.form.desc.allowFrom")}
      >
        <Input
          value={asStringArray(config.allow_from).join(", ")}
          onChange={(e) =>
            onChange(
              "allow_from",
              e.target.value
                .split(",")
                .map((s: string) => s.trim())
                .filter(Boolean),
            )
          }
          placeholder={t("channels.field.allowFromPlaceholder")}
        />
      </Field>

      {/* proxy */}
      <Field
        label={t("channels.field.proxy")}
        hint={t("channels.form.desc.proxy")}
      >
        <Input
          value={asString(config.proxy)}
          onChange={(e) => onChange("proxy", e.target.value)}
          placeholder="http://localhost:7890"
        />
      </Field>
    </div>
  )
}
