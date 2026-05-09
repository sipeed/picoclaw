import { IconLanguage } from "@tabler/icons-react"
import { createFileRoute } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  getLauncherAuthStatus,
  postLauncherDashboardLogin,
} from "@/api/launcher-auth"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

function AfricaLoginOrb() {
  return (
    <div className="relative flex h-32 w-32 items-center justify-center">
      <svg className="absolute inset-0 h-full w-full" viewBox="0 0 128 128">
        <circle
          cx="64" cy="64" r="60"
          fill="none"
          stroke="rgba(0, 212, 255, 0.15)"
          strokeWidth="1"
          strokeDasharray="6 4"
          style={{ animation: "orb-rotate 12s linear infinite", transformOrigin: "center" }}
        />
        <circle
          cx="64" cy="64" r="50"
          fill="none"
          stroke="rgba(0, 212, 255, 0.1)"
          strokeWidth="1.5"
          strokeDasharray="3 6"
          style={{ animation: "orb-rotate-reverse 8s linear infinite", transformOrigin: "center" }}
        />
        <circle
          cx="64" cy="64" r="40"
          fill="none"
          stroke="rgba(0, 212, 255, 0.25)"
          strokeWidth="1"
          style={{ animation: "ring-pulse 3s ease-in-out infinite" }}
        />
      </svg>
      <div className="animate-breathe flex h-10 w-10 items-center justify-center rounded-full bg-[rgba(0,212,255,0.08)] shadow-[0_0_20px_rgba(0,212,255,0.3),0_0_40px_rgba(0,212,255,0.15)]">
        <div className="h-4 w-4 rounded-full bg-[#00d4ff] shadow-[0_0_12px_rgba(0,212,255,0.8),0_0_24px_rgba(0,212,255,0.4)]" />
      </div>
    </div>
  )
}

function LauncherLoginPage() {
  const { t, i18n } = useTranslation()
  const [password, setPassword] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)
  const [error, setError] = React.useState("")

  React.useEffect(() => {
    void getLauncherAuthStatus()
      .then((s) => {
        if (!s.initialized) {
          globalThis.location.assign("/launcher-setup")
        }
      })
      .catch(() => {
        /* network error — stay on login page */
      })
  }, [])

  const loginWithPassword = React.useCallback(
    async (passwordValue: string) => {
      setError("")
      setSubmitting(true)
      try {
        const result = await postLauncherDashboardLogin(passwordValue)
        if (result.ok) {
          globalThis.location.assign("/")
          return
        }
        if (result.status === 409) {
          globalThis.location.assign("/launcher-setup")
          return
        }
        if (result.status === 401) {
          setError(t("launcherLogin.errorInvalid"))
          return
        }
        setError(result.error)
      } catch {
        setError(t("launcherLogin.errorNetwork"))
      } finally {
        setSubmitting(false)
      }
    },
    [t],
  )

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    await loginWithPassword(password)
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
            <AfricaLoginOrb />
            <h1 className="glow-text-pulse mt-5 text-2xl font-bold tracking-[0.3em] text-[#00d4ff]">
              J.A.R.V.I.S.
            </h1>
            <p className="mt-2 text-sm tracking-wider text-[rgba(0,212,255,0.4)]">
              {t("launcherLogin.description")}
            </p>
          </div>

          <form className="flex flex-col gap-5" onSubmit={onSubmit}>
            <div className="flex flex-col gap-2">
              <Label
                htmlFor="launcher-password"
                className="text-xs font-medium tracking-wider text-[rgba(0,212,255,0.5)] uppercase"
              >
                {t("launcherLogin.passwordLabel")}
              </Label>
              <Input
                id="launcher-password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t("launcherLogin.passwordPlaceholder")}
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
                <span className="uppercase">{t("launcherLogin.submit")}</span>
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
            <span>SECURE ACCESS</span>
            <div className="h-1 w-1 rounded-full bg-[rgba(0,212,255,0.3)]" />
          </div>
        </div>
      </div>

      {/* Bottom glow line */}
      <div className="h-px bg-gradient-to-r from-transparent via-[rgba(0,212,255,0.3)] to-transparent" />
    </div>
  )
}

export const Route = createFileRoute("/launcher-login")({
  component: LauncherLoginPage,
})