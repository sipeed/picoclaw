import {
  Navigate,
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router"

export const Route = createFileRoute("/agent")({
  component: AgentLayout,
})

function AgentLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  if (pathname === "/agent") {
    return <Navigate to="/agent/hub" />
  }

  return <Outlet />
}
