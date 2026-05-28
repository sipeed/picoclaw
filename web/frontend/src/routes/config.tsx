import { Outlet, createFileRoute, useRouterState } from "@tanstack/react-router"

import { ConfigPage } from "@/components/config/config-page"

export const Route = createFileRoute("/config")({
  component: ConfigRouteLayout,
})

function ConfigRouteLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  if (pathname === "/config") {
    return <ConfigPage />
  }

  return <Outlet />
}
