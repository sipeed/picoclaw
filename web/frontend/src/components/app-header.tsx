import {
  IconBook,
  IconLanguage,
  IconLoader2,
  IconLogout,
  IconMenu2,
  IconPlayerPlay,
  IconPower,
  IconRefresh,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import { postLauncherDashboardLogout } from "@/api/launcher-auth"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog.tsx"
import { Button } from "@/components/ui/button.tsx"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx"
import { Separator } from "@/components/ui/separator.tsx"
import { SidebarTrigger } from "@/components/ui/sidebar"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useGateway } from "@/hooks/use-gateway.ts"

export function AppHeader() {
  const { i18n, t } = useTranslation()
  const {
    state: gwState,
    loading: gwLoading,
    canStart,
    startReason,
    restartRequired,
    start,
    restart,
    stop,
    error: gwError,
  } = useGateway()

  const isRunning = gwState === "running"
  const isStarting = gwState === "starting"
  const isRestarting = gwState === "restarting"
  const isStopping = gwState === "stopping"
  const isStopped = gwState === "stopped" || gwState === "unknown"
  const showNotConnectedHint =
    !isRestarting &&
    !isStopping &&
    canStart &&
    (gwState === "stopped" || gwState === "error")

  const [showStopDialog, setShowStopDialog] = React.useState(false)
  const [showLogoutDialog, setShowLogoutDialog] = React.useState(false)
  const [currentTime, setCurrentTime] = React.useState("")

  // Live clock
  React.useEffect(() => {
    const update = () => {
      const now = new Date()
      setCurrentTime(
        now.toLocaleTimeString("en-US", {
          hour12: false,
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })
      )
    }
    update()
    const interval = setInterval(update, 1000)
    return () => clearInterval(interval)
  }, [])

  const handleLogout = async () => {
    await postLauncherDashboardLogout()
    globalThis.location.assign("/launcher-login")
  }

  const handleGatewayToggle = () => {
    if (gwLoading || isRestarting || isStopping || (!isRunning && !canStart)) {
      return
    }
    if (isRunning) {
      setShowStopDialog(true)
    } else {
      void start()
    }
  }

  const handleGatewayRestart = () => {
    if (gwLoading || isRestarting || !restartRequired || !canStart) return
    void restart()
  }

  const confirmStop = () => {
    setShowStopDialog(false)
    stop()
  }

  return (
    <header className="glass-panel-strong sticky top-0 z-50 flex h-14 shrink-0 items-center justify-between border-b border-[rgba(0,212,255,0.15)] px-4">
      <div className="flex items-center gap-2">
        <SidebarTrigger className="text-jarvis-muted hover:bg-jarvis-glow hover:text-jarvis-cyan flex h-9 w-9 items-center justify-center rounded-lg sm:hidden [&>svg]:size-5">
          <IconMenu2 />
        </SidebarTrigger>
        <div className="hidden shrink-0 items-center sm:flex">
          <Link to="/" className="flex items-center gap-2.5">
            <div className="relative flex h-8 w-8 items-center justify-center">
              {/* Mini Africa orb in header */}
              <div className="absolute inset-0 rounded-full border border-[rgba(0,212,255,0.4)] animate-breathe" />
              <div className="absolute inset-1.5 rounded-full border border-dashed border-[rgba(0,212,255,0.3)]" style={{ animation: "orb-rotate 8s linear infinite" }} />
              <div className="h-2 w-2 rounded-full bg-[#00d4ff] shadow-[0_0_8px_rgba(0,212,255,0.6)]" />
            </div>
            <span className="glow-text-pulse text-lg font-bold tracking-[0.25em] text-[#00d4ff]">
              A.F.R.I.C.A
            </span>
          </Link>
        </div>
      </div>

      {/* Center status */}
      <div className="pointer-events-none absolute left-1/2 hidden h-full -translate-x-1/2 items-center justify-center lg:flex">
        {showNotConnectedHint ? (
          <div className="flex items-center gap-2 rounded-full border border-[rgba(239,68,68,0.3)] bg-[rgba(239,68,68,0.05)] px-4 py-1.5 text-xs backdrop-blur-md">
            <span className="relative flex size-2 shrink-0 items-center justify-center rounded-full bg-[rgba(239,68,68,0.5)]">
              <span className="absolute inline-flex size-full animate-ping rounded-full bg-[rgba(239,68,68,0.4)] opacity-75"></span>
            </span>
            <span className="text-[rgba(239,68,68,0.8)]">{t("chat.notConnected")}</span>
          </div>
        ) : isRunning ? (
          <div className="flex items-center gap-2 text-xs">
            <span className="relative flex size-2 shrink-0 items-center justify-center rounded-full bg-[#10b981]">
              <span className="absolute inline-flex size-full animate-ping rounded-full bg-[#10b981] opacity-50"></span>
            </span>
            <span className="font-medium tracking-wider text-[rgba(0,212,255,0.6)]">SYSTEM ONLINE</span>
          </div>
        ) : null}
      </div>

      <AlertDialog open={showStopDialog} onOpenChange={setShowStopDialog}>
        <AlertDialogContent className="glass-panel-strong border-[rgba(0,212,255,0.2)]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#00d4ff]">
              {t("header.gateway.stopDialog.title")}
            </AlertDialogTitle>
            <AlertDialogDescription className="text-[#94a3b8]">
              {t("header.gateway.stopDialog.description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="border-[rgba(0,212,255,0.15)] bg-transparent text-[#94a3b8] hover:bg-[rgba(0,212,255,0.05)]">CANCEL</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmStop}
              className="bg-[#ef4444] text-white hover:bg-[#ef4444]/90 border-none"
            >
              SHUTDOWN
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={showLogoutDialog} onOpenChange={setShowLogoutDialog}>
        <AlertDialogContent className="glass-panel-strong border-[rgba(0,212,255,0.2)]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#00d4ff]">{t("header.logout.tooltip")}</AlertDialogTitle>
            <AlertDialogDescription className="text-[#94a3b8]">
              {t("header.logout.description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="border-[rgba(0,212,255,0.15)] bg-transparent text-[#94a3b8] hover:bg-[rgba(0,212,255,0.05)]">CANCEL</AlertDialogCancel>
            <AlertDialogAction onClick={() => void handleLogout()} className="bg-[#00d4ff] text-[#0a0e17] hover:bg-[#00d4ff]/90 border-none">
              LOGOUT
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <div className="flex items-center gap-1 text-sm md:gap-2">
        {/* Live Clock */}
        <span className="mr-2 hidden font-mono text-[rgba(0,212,255,0.4)] text-xs tracking-wider lg:block">
          {currentTime}
        </span>

        {restartRequired && (
          <Tooltip delayDuration={700}>
            <TooltipTrigger asChild>
              <Button
                variant="secondary"
                size="icon-sm"
                className="border-[rgba(245,158,11,0.3)] bg-[rgba(245,158,11,0.1)] text-[#f59e0b] hover:bg-[rgba(245,158,11,0.2)]"
                onClick={handleGatewayRestart}
                disabled={gwLoading || isRestarting || isStopping || !canStart}
                aria-label={t("header.gateway.action.restart")}
              >
                <IconRefresh className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent className="glass-panel-strong border-[rgba(0,212,255,0.2)] text-[#e0f2fe]">
              {t("header.gateway.restartRequired")}
            </TooltipContent>
          </Tooltip>
        )}

        {/* Gateway Start/Stop */}
        {isRunning ? (
          <Tooltip delayDuration={700}>
            <TooltipTrigger asChild>
              <Button
                variant="destructive"
                size="icon-sm"
                className="size-8 border-[rgba(239,68,68,0.3)] bg-[rgba(239,68,68,0.15)] text-[#ef4444] hover:bg-[rgba(239,68,68,0.25)]"
                data-tour="gateway-button"
                onClick={handleGatewayToggle}
                disabled={gwLoading}
                aria-label={t("header.gateway.action.stop")}
              >
                <IconPower className="h-4 w-4 opacity-80" />
              </Button>
            </TooltipTrigger>
            <TooltipContent className="glass-panel-strong border-[rgba(0,212,255,0.2)] text-[#e0f2fe]">
              {gwError ?? t("header.gateway.action.stop")}
            </TooltipContent>
          </Tooltip>
        ) : (
          <Tooltip delayDuration={gwError || (!canStart && startReason) ? 0 : 700}>
            <TooltipTrigger asChild>
              <span
                className={!canStart && startReason ? "cursor-not-allowed" : undefined}
                tabIndex={!canStart && startReason ? 0 : undefined}
              >
                <Button
                  variant={isStarting || isRestarting || isStopping ? "secondary" : "default"}
                  size="sm"
                  data-tour="gateway-button"
                  className={`h-8 gap-2 border-[rgba(16,185,129,0.3)] px-3 ${
                    isStopped
                      ? "bg-[rgba(16,185,129,0.15)] text-[#10b981] hover:bg-[rgba(16,185,129,0.25)]"
                      : "bg-transparent text-[#94a3b8]"
                  } ${!canStart ? "pointer-events-none" : ""}`}
                  onClick={handleGatewayToggle}
                  disabled={gwLoading || isStarting || isRestarting || isStopping || !canStart}
                >
                  {gwLoading || isStarting || isRestarting || isStopping ? (
                    <IconLoader2 className="h-4 w-4 animate-spin opacity-70" />
                  ) : (
                    <IconPlayerPlay className="h-4 w-4 opacity-80" />
                  )}
                  <span className="text-xs font-semibold tracking-wider">
                    {isStopping
                      ? t("header.gateway.status.stopping")
                      : isRestarting
                        ? t("header.gateway.status.restarting")
                        : isStarting
                          ? t("header.gateway.status.starting")
                          : "INITIALIZE"}
                  </span>
                </Button>
              </span>
            </TooltipTrigger>
            {gwError || (!canStart && startReason) ? (
              <TooltipContent className="glass-panel-strong border-[rgba(0,212,255,0.2)] text-[#e0f2fe]">{gwError ?? startReason}</TooltipContent>
            ) : null}
          </Tooltip>
        )}

        <Separator className="mx-3 my-2 hidden md:block" orientation="vertical" />

        {/* Docs Link */}
        <Button
          variant="ghost"
          size="icon"
          className="size-8 text-[#64748b] hover:bg-[rgba(0,212,255,0.05)] hover:text-[#00d4ff]"
          data-tour="docs-button"
          asChild
        >
          <a href="https://docs.picoclaw.io" target="_blank" rel="noreferrer">
            <IconBook className="size-4.5" />
          </a>
        </Button>

        {/* Language Switcher */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="size-8 text-[#64748b] hover:bg-[rgba(0,212,255,0.05)] hover:text-[#00d4ff]">
              <IconLanguage className="size-4.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="glass-panel-strong border-[rgba(0,212,255,0.2)]">
            <DropdownMenuItem onClick={() => i18n.changeLanguage("en")} className="text-[#e0f2fe] focus:bg-[rgba(0,212,255,0.1)] focus:text-[#00d4ff]">
              English
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => i18n.changeLanguage("zh")} className="text-[#e0f2fe] focus:bg-[rgba(0,212,255,0.1)] focus:text-[#00d4ff]">
              简体中文
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <Separator className="mx-2 my-2" orientation="vertical" />

        {/* Logout */}
        <Tooltip delayDuration={700}>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-8 text-[#64748b] hover:bg-[rgba(0,212,255,0.05)] hover:text-[#00d4ff]"
              onClick={() => setShowLogoutDialog(true)}
              aria-label={t("header.logout.tooltip")}
            >
              <IconLogout className="size-4.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent className="glass-panel-strong border-[rgba(0,212,255,0.2)] text-[#e0f2fe]">{t("header.logout.tooltip")}</TooltipContent>
        </Tooltip>
      </div>

      {/* Bottom glow line */}
      <div className="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[rgba(0,212,255,0.3)] to-transparent" />
    </header>
  )
}