import { IconLanguage } from "@tabler/icons-react"
import { createFileRoute } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import { postLauncherDashboardSetup } from "@/api/launcher-auth"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

function LauncherSetupPage() {
  const { t, i18n } = useTranslation()
  const [password, setPassword] = React.useState("")
  const [confirm, setConfirm] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)
  const [error, setError] = React.useState("")

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError("")
    if (password !== confirm) {
      setError(t("launcherSetup.errorMismatch"))
      return
    }
    setSubmitting(true)
    try {
      const result = await postLauncherDashboardSetup(password, confirm)
      if (result.ok) {
        globalThis.location.assign("/launcher-login")
        return
      }
      setError(result.error)
    } catch {
      setError(t("launcherSetup.errorNetwork"))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-dvh flex-col bg-[#0a0e17] text-[#e0f2fe] hex-grid-bg scan-line-overlay">
      <header className="flex h-14 shrink-0 items-center justify-end gap-2 border-b border-[rgba(0,212,255,0.15)] px-4">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="text-[#64748b] hover:bg-[rgba(0,212,255,0.05)] hover:text-[#00d4ff]"
              aria-label="Language"
            >
              <IconLanguage className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="glass-panel-strong border-[rgba(0,212,255,0.2)]">
            <DropdownMenuItem
              onClick={() => i18n.changeLanguage("en")}
              className="text-[#e0f2fe] focus:bg-[rgba(0,212,255,0.1)] focus:text-[#00d4ff]"
            >
              English
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => i18n.changeLanguage("zh")}
              className="text-[#e0f2fe] focus:bg-[rgba(0,212,255,0.1)] focus:text-[#00d4ff]"
            >
              简体中文
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </header>

      <div className="flex flex-1 items-center justify-center p-4">
        <div className="glass-panel hud-corners w-full max-w-md rounded-2xl p-8">
          {/* Orb & Title */}
          <div className="mb-8 flex flex-col items-center">
            <div className="relative flex h-24 w-24 items-center justify-center">
              <svg className="absolute inset-0 h-full w-full" viewBox="0 0 96 96">
                <circle
                  cx="48" cy="48" r="44"
                  fill="none"
                  stroke="rgba(0, 212, 255, 0.15)"
                  strokeWidth="1"
                  strokeDasharray="6 4"
                  style={{ animation: "orb-rotate 12s linear infinite", transformOrigin: "center" }}
                />
                <circle
                  cx="48" cy="48" r="36"
                  fill="none"
                  stroke="rgba(0, 212, 255, 0.25)"
                  strokeWidth="1"
                  style={{ animation: "ring-pulse 3s ease-in-out infinite" }}
                />
              </svg>
              <div className="animate-breathe flex h-8 w-8 items-center justify-center rounded-full bg-[rgba(0,212,255,0.08)] shadow-[0_0_15px_rgba(0,212,255,0.3)]">
                <div className="h-3 w-3 rounded-full bg-[#00d4ff] shadow-[0_0_10px_rgba(0,212,255,0.8)]" />
              </div>
            </div>
            <h1 className="glow-text-pulse mt-4 text-xl font-bold tracking-[0.3em] text-[#00d4ff]">
              J.A.R.V.I.S.
            </h1>
            <p className="mt-2 text-sm tracking-wider text-[rgba(0,212,255,0.4)]">
              {t("launcherSetup.description")}
            </p>
          </div>

          <form className="flex flex-col gap-5" onSubmit={onSubmit}>
            <div className="flex flex-col gap-2">
              <Label
                htmlFor="setup-password"
                className="text-xs font-medium tracking-wider text-[rgba(0,212,255,0.5)] uppercase"
              >
                {t("launcherSetup.passwordLabel")}
              </Label>
              <Input
                id="setup-password"
                name="password"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t("launcherSetup.passwordPlaceholder")}
                className="border-[rgba(0,212,255,0.2)] bg-[rgba(0,212,255,0.03)] text-[#e0f2fe] placeholder:text-[#475569] focus:border-[rgba(0,212,255,0.4)] focus:ring-[rgba(0,212,255,0.2)]"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label
                htmlFor="setup-confirm"
                className="text-xs font-medium tracking-wider text-[rgba(0,212,255,0.5)] uppercase"
              >
                {t("launcherSetup.confirmLabel")}
              </Label>
              <Input
                id="setup-confirm"
                name="confirm"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                placeholder={t("launcherSetup.confirmPlaceholder")}
                className="border-[rgba(0,212,255,0.2)] bg-[rgba(0,212,255,0.03)] text-[#e0f2fe] placeholder:text-[#475569] focus:border-[rgba(0,212,255,0.4)] focus:ring-[rgba(0,212,255,0.2)]"
              />
            </div>
            <Button
              type="submit"
              disabled={submitting}
              className="h-11 border-none bg-[#00d4ff] text-[#0a0e17] font-semibold tracking-wider shadow-[0_0_15px_rgba(0,212,255,0.3)] transition-all hover:bg-[#00d4ff]/90 hover:shadow-[0_0_25px_rgba(0,212,255,0.5)] active:scale-[0.98]"
            >
              {submitting ? (
                <div className="flex items-center gap-2">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-[#0a0e17] border-t-transparent" />
                  <span>{t("labels.loading")}</span>
                </div>
              ) : (
                <span className="uppercase">{t("launcherSetup.submit")}</span>
              )}
            </Button>
            {error && (
              <div className="rounded-lg border border-[rgba(239,68,68,0.3)] bg-[rgba(239,68,68,0.05)] px-4 py-2.5 text-center text-sm text-[#ef4444]" role="alert">
                {error}
              </div>
            )}
          </form>

          {/* Bottom decoration */}
          <div className="mt-8 flex items-center justify-center gap-2 text-[10px] tracking-[0.2em] text-[rgba(0,212,255,0.2)]">
            <div className="h-1 w-1 rounded-full bg-[rgba(0,212,255,0.3)]" />
            <span>INITIALIZE SYSTEM</span>
            <div className="h-1 w-1 rounded-full bg-[rgba(0,212,255,0.3)]" />
          </div>
        </div>
      </div>

      {/* Bottom glow line */}
      <div className="h-px bg-gradient-to-r from-transparent via-[rgba(0,212,255,0.3)] to-transparent" />
    </div>
  )
}

export const Route = createFileRoute("/launcher-setup")({
  component: LauncherSetupPage,
})