import {
  Outlet,
  createRootRoute,
  useRouterState,
} from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"
import { useEffect } from "react"

import { AppLayout } from "@/components/app-layout"
import { initializeChatStore } from "@/features/chat/controller"
import { isLauncherLoginPathname } from "@/lib/launcher-login-path"

const RootLayout = () => {
  // Prefer the real address bar path: stale embedded bundles may not register
  // /launcher-login in the route tree, which would otherwise keep AppLayout +
  // gateway polling → 401 → launcherFetch redirect loop.
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

  const isLauncherLogin =
    isLauncherLoginPathname(windowPath) ||
    isLauncherLoginPathname(routerState.pathname) ||
    routerState.matches.some((m) => m.routeId === "/launcher-login")

  useEffect(() => {
    if (isLauncherLogin) {
      return
    }
    initializeChatStore()
  }, [isLauncherLogin])

  if (isLauncherLogin) {
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
