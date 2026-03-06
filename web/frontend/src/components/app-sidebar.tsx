import { IconChevronRight } from "@tabler/icons-react"
import {
  IconAtom,
  IconKey,
  IconListDetails,
  IconMessageCircle,
  IconSettings,
} from "@tabler/icons-react"
import { Link, useRouterState } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar"

// Navigation data with real routes
const navGroups = [
  {
    label: "navigation.chat",
    defaultOpen: true,
    items: [{ title: "navigation.chat", url: "/", icon: IconMessageCircle }],
  },
  {
    label: "navigation.model_group",
    defaultOpen: true,
    items: [
      { title: "navigation.models", url: "/models", icon: IconAtom },
      { title: "navigation.credentials", url: "/credentials", icon: IconKey },
    ],
  },
  {
    label: "navigation.services",
    defaultOpen: true,
    items: [
      { title: "navigation.config", url: "/config", icon: IconSettings },
      { title: "navigation.logs", url: "/logs", icon: IconListDetails },
    ],
  },
]

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const routerState = useRouterState()
  const { t } = useTranslation()
  const currentPath = routerState.location.pathname

  return (
    <Sidebar
      {...props}
      className="bg-background border-r-border/20 border-r pt-3"
    >
      <SidebarContent className="bg-background">
        {navGroups.map((group) => (
          <Collapsible
            key={group.label}
            defaultOpen={group.defaultOpen}
            className="group/collapsible mb-1"
          >
            <SidebarGroup className="px-2 py-0">
              <SidebarGroupLabel asChild>
                <CollapsibleTrigger className="hover:bg-muted/60 flex w-full cursor-pointer items-center justify-between rounded-md px-2 py-1.5 transition-colors">
                  <span>{t(group.label)}</span>
                  <IconChevronRight className="size-3.5 opacity-50 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                </CollapsibleTrigger>
              </SidebarGroupLabel>
              <CollapsibleContent>
                <SidebarGroupContent className="pt-1">
                  <SidebarMenu>
                    {group.items.map((item) => {
                      const isActive = currentPath === item.url
                      return (
                        <SidebarMenuItem key={item.title}>
                          <SidebarMenuButton
                            asChild
                            isActive={isActive}
                            className={`h-9 px-3 ${isActive ? "bg-accent/80 text-foreground font-medium" : "text-muted-foreground hover:bg-muted/60"}`}
                          >
                            <Link to={item.url}>
                              <item.icon
                                className={`size-4 ${isActive ? "opacity-100" : "opacity-60"}`}
                              />
                              <span
                                className={
                                  isActive ? "opacity-100" : "opacity-80"
                                }
                              >
                                {t(item.title)}
                              </span>
                            </Link>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      )
                    })}
                  </SidebarMenu>
                </SidebarGroupContent>
              </CollapsibleContent>
            </SidebarGroup>
          </Collapsible>
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}
