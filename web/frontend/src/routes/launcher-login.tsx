import { IconLanguage, IconMoon, IconSun } from "@tabler/icons-react"
import { createFileRoute } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  getLauncherAuthStatus,
  postLauncherDashboardLogin,
  type LauncherAuthTokenHelp,
} from "@/api/launcher-auth"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useTheme } from "@/hooks/use-theme"

function LauncherLoginPage() {
  const { t, i18n } = useTranslation()
  const { theme, toggleTheme } = useTheme()
  const [token, setToken] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)
  const [error, setError] = React.useState("")
  const [tokenHelp, setTokenHelp] = React.useState<LauncherAuthTokenHelp | null>(
    null,
  )

  React.useEffect(() => {
    let cancelled = false
    void getLauncherAuthStatus()
      .then((s) => {
        if (cancelled || s.authenticated || !s.token_help) {
          return
        }
        setTokenHelp(s.token_help)
      })
      .catch(() => {
        /* ignore; login form still usable */
      })
    return () => {
      cancelled = true
    }
  }, [])

  const loginWithToken = React.useCallback(
    async (tokenValue: string) => {
      setError("")
      setSubmitting(true)
      try {
        const ok = await postLauncherDashboardLogin(tokenValue)
        if (ok) {
          globalThis.location.assign("/")
          return
        }
        setError(t("launcherLogin.errorInvalid"))
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
    await loginWithToken(token)
  }

  return (
    <div className="bg-background text-foreground flex min-h-dvh flex-col">
      <header className="border-border/50 flex h-14 shrink-0 items-center justify-end gap-2 border-b px-4">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="icon" aria-label="Language">
              <IconLanguage className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => i18n.changeLanguage("en")}>
              English
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => i18n.changeLanguage("zh")}>
              简体中文
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          variant="outline"
          size="icon"
          type="button"
          onClick={() => toggleTheme()}
          aria-label={theme === "dark" ? "Light mode" : "Dark mode"}
        >
          {theme === "dark" ? (
            <IconSun className="size-4" />
          ) : (
            <IconMoon className="size-4" />
          )}
        </Button>
      </header>

      <div className="flex flex-1 items-center justify-center p-4">
        <Card className="w-full max-w-md" size="sm">
          <CardHeader>
            <CardTitle>{t("launcherLogin.title")}</CardTitle>
            <CardDescription>{t("launcherLogin.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="flex flex-col gap-4" onSubmit={onSubmit}>
              <div className="flex flex-col gap-2">
                <Label htmlFor="launcher-token">
                  {t("launcherLogin.tokenLabel")}
                </Label>
                <Input
                  id="launcher-token"
                  name="token"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder={t("launcherLogin.tokenPlaceholder")}
                />
              </div>
              <Button type="submit" disabled={submitting}>
                {submitting ? t("labels.loading") : t("launcherLogin.submit")}
              </Button>
              {error ? (
                <p className="text-destructive text-sm" role="alert">
                  {error}
                </p>
              ) : null}
            </form>
            {tokenHelp ? (
              <div className="border-border/60 mt-6 border-t pt-4">
                <p className="text-muted-foreground mb-2 text-sm font-medium">
                  {t("launcherLogin.helpTitle")}
                </p>
                <ul className="text-muted-foreground list-inside list-disc space-y-1.5 text-sm">
                  {tokenHelp.console_stdout ? (
                    <li>{t("launcherLogin.helpConsole")}</li>
                  ) : null}
                  {tokenHelp.tray_copy_menu ? (
                    <li>{t("launcherLogin.helpTray")}</li>
                  ) : null}
                  {tokenHelp.log_file ? (
                    <li>
                      {t("launcherLogin.helpLogFile", {
                        path: tokenHelp.log_file,
                      })}
                    </li>
                  ) : null}
                  {tokenHelp.env_var_name ? (
                    <li>
                      {t("launcherLogin.helpEnv", {
                        env: tokenHelp.env_var_name,
                      })}
                    </li>
                  ) : null}
                </ul>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export const Route = createFileRoute("/launcher-login")({
  component: LauncherLoginPage,
})
