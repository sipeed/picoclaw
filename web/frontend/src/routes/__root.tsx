import { Outlet, createRootRoute, useNavigate, useRouterState } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"
import { useEffect } from "react"

import { AppLayout } from "@/components/app-layout"
import { initializeChatStore } from "@/features/chat/controller"
import { useAuth } from "@/features/auth"

const RootLayout = () => {
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const { needsSetup, needsLogin, isAuthenticated } = useAuth()

  useEffect(() => {
    initializeChatStore()
  }, [])

  useEffect(() => {
    if ((needsSetup || needsLogin) && pathname !== "/login") {
      navigate({ to: "/login" })
    }
  }, [needsSetup, needsLogin, pathname, navigate])

  if (!isAuthenticated && pathname !== "/login") {
    return null
  }

  return (
    <AppLayout>
      <Outlet />
      <TanStackRouterDevtools />
    </AppLayout>
  )
}

export const Route = createRootRoute({ component: RootLayout })
