import {
  Navigate,
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router"

export const Route = createFileRoute("/channels")({
  component: ChannelsLayout,
})

function ChannelsLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  if (pathname === "/channels") {
    return <Navigate to="/channels/$name" params={{ name: "pico" }} />
  }

  return <Outlet />
}
