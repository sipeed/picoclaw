import {
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router"

import { ResearchPage } from "@/components/research/research-page"

export const Route = createFileRoute("/research")({
  component: ResearchRouteLayout,
})

function ResearchRouteLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  if (pathname === "/research") {
    return <ResearchPage />
  }

  return <Outlet />
}
