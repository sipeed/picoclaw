import { Outlet, createRootRoute, useRouterState } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"
import { useEffect } from "react"

import { getLauncherAuthStatus } from "@/api/launcher-auth"
import { AppLayout } from "@/components/app-layout"
import { initializeChatStore } from "@/features/chat/controller"
import { isLauncherAuthPathname } from "@/lib/launcher-login-path"

const RootLayout = () => {
  // Prefer the real address bar path: stale embedded bundles may not register
  // /launcher-login or /launcher-setup in the route tree, which would otherwise
  // keep AppLayout + gateway polling → 401 → launcherFetch redirect loop.
  const routerState = useRouterState({
    select: (s) => ({
      pathname: s.location.pathname,
      matches: s.matches,
    }),
  })

  const windowPath =
    typeof globalThis.location !== "undefined"
      ? globalThis.location.pathname || "/"
      : routerState.pathname

  const isAuthPage =
    isLauncherAuthPathname(windowPath) ||
    isLauncherAuthPathname(routerState.pathname) ||
    routerState.matches.some(
      (m) => m.routeId === "/launcher-login" || m.routeId === "/launcher-setup",
    )

  // Session guard: proactively check auth status on every page load.
  // This catches the case where ?token= auto-login bypassed the login/setup UI.
  useEffect(() => {
    if (isAuthPage) return
    void getLauncherAuthStatus()
      .then((s) => {
        if (!s.initialized) {
          globalThis.location.assign("/launcher-setup")
        } else if (!s.authenticated) {
          globalThis.location.assign("/launcher-login")
        }
      })
      .catch(() => {
        // Network error or 401 — launcherFetch will handle redirect on real API calls.
      })
  }, [isAuthPage])

  useEffect(() => {
    if (isAuthPage) {
      return
    }
    initializeChatStore()
  }, [isAuthPage])

  if (isAuthPage) {
    return (
      <>
        <Outlet />
        {import.meta.env.DEV ? <TanStackRouterDevtools /> : null}
      </>
    )
  }

  return (
    <AppLayout>
      <Outlet />
      {import.meta.env.DEV ? <TanStackRouterDevtools /> : null}
    </AppLayout>
  )
}

export const Route = createRootRoute({ component: RootLayout })
