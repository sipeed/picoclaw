import {
  IconDatabase,
  IconFileSearch,
} from "@tabler/icons-react"
import {
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router"
import * as React from "react"

import { MediaCachePage } from "@/components/research/media-cache-page"
import { ResearchPage } from "@/components/research/research-page"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export const Route = createFileRoute("/research")({
  component: ResearchRouteLayout,
})

function ResearchRouteLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [tab, setTab] = React.useState<"research" | "media">("research")

  // If on a detail sub-route, show Outlet
  if (pathname !== "/research") {
    return <Outlet />
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={tab === "research" ? "Research" : "Media"}>
        <div className="flex gap-1 rounded-lg bg-muted p-1">
          <TabButton
            active={tab === "research"}
            onClick={() => setTab("research")}
          >
            <IconFileSearch className="size-3.5" />
            Research
          </TabButton>
          <TabButton
            active={tab === "media"}
            onClick={() => setTab("media")}
          >
            <IconDatabase className="size-3.5" />
            Media
          </TabButton>
        </div>
      </PageHeader>

      {tab === "research" ? <ResearchPage /> : <MediaCachePage />}
    </div>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={onClick}
      className={cn(
        "gap-1.5 text-xs",
        active && "bg-background shadow-sm",
      )}
    >
      {children}
    </Button>
  )
}
