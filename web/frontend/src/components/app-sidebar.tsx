import { IconChevronRight } from "@tabler/icons-react"
import {
  IconAtom,
  IconChevronsDown,
  IconChevronsUp,
  IconCpu,
  IconKey,
  IconListDetails,
  IconMessageCircle,
  IconSearch,
  IconSettings,
  IconSparkles,
  IconTools,
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
  useSidebar,
} from "@/components/ui/sidebar"
import { useSidebarChannels } from "@/hooks/use-sidebar-channels"

interface NavItem {
  title: string
  url: string
  icon: React.ComponentType<{ className?: string }>
  translateTitle?: boolean
}

interface NavGroup {
  label: string
  defaultOpen: boolean
  items: NavItem[]
  isChannelsGroup?: boolean
}

const baseNavGroups: Omit<NavGroup, "items">[] = [
  { label: "navigation.chat", defaultOpen: true },
  { label: "navigation.model_group", defaultOpen: true },
  { label: "navigation.agent_group", defaultOpen: true },
  { label: "navigation.services", defaultOpen: true },
]

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const routerState = useRouterState()
  const { i18n, t } = useTranslation()
  const { isMobile, setOpenMobile } = useSidebar()
  const currentPath = routerState.location.pathname
  const {
    channelItems,
    hasMoreChannels,
    showAllChannels,
    toggleShowAllChannels,
  } = useSidebarChannels({
    language: (i18n.resolvedLanguage ?? i18n.language ?? "").toLowerCase(),
    t,
  })

  const handleNavItemClick = React.useCallback(() => {
    if (isMobile) {
      setOpenMobile(false)
    }
  }, [isMobile, setOpenMobile])

  const navGroups: NavGroup[] = React.useMemo(() => {
    return [
      {
        ...baseNavGroups[0],
        items: [
          { title: "navigation.chat", url: "/", icon: IconMessageCircle, translateTitle: true },
        ],
      },
      {
        ...baseNavGroups[1],
        items: [
          { title: "navigation.models", url: "/models", icon: IconAtom, translateTitle: true },
          { title: "navigation.credentials", url: "/credentials", icon: IconKey, translateTitle: true },
        ],
      },
      {
        label: "navigation.channels_group",
        defaultOpen: true,
        items: channelItems.map((item) => ({
          title: item.title,
          url: item.url,
          icon: item.icon,
          translateTitle: false,
        })),
        isChannelsGroup: true,
      },
      {
        ...baseNavGroups[2],
        items: [
          { title: "Agent Cockpit", url: "/agent/cockpit", icon: IconCpu, translateTitle: false },
          { title: "navigation.hub", url: "/agent/hub", icon: IconSearch, translateTitle: true },
          { title: "navigation.skills", url: "/agent/skills", icon: IconSparkles, translateTitle: true },
          { title: "navigation.tools", url: "/agent/tools", icon: IconTools, translateTitle: true },
        ],
      },
      {
        ...baseNavGroups[3],
        items: [
          { title: "navigation.config", url: "/config", icon: IconSettings, translateTitle: true },
          { title: "navigation.logs", url: "/logs", icon: IconListDetails, translateTitle: true },
        ],
      },
    ]
  }, [channelItems])

  return (
    <Sidebar
      {...props}
      className="border-r-[rgba(0,212,255,0.1)] bg-[#0f172a] pt-3"
    >
      <SidebarContent className="bg-[#0f172a]">
        {navGroups.map((group) => (
          <Collapsible
            key={group.label}
            defaultOpen={group.defaultOpen}
            className="group/collapsible mb-1"
          >
            <SidebarGroup className="px-2 py-0">
              <SidebarGroupLabel asChild>
                <CollapsibleTrigger className="flex w-full cursor-pointer items-center justify-between rounded-md px-2 py-1.5 text-xs font-medium tracking-wider text-[rgba(0,212,255,0.4)] transition-colors hover:bg-[rgba(0,212,255,0.05)] hover:text-[rgba(0,212,255,0.6)]">
                  <span className="uppercase">{t(group.label)}</span>
                  <IconChevronRight className="size-3 opacity-40 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                </CollapsibleTrigger>
              </SidebarGroupLabel>
              <CollapsibleContent>
                <SidebarGroupContent className="pt-1">
                  <SidebarMenu>
                    {group.items.map((item) => {
                      const isActive =
                        currentPath === item.url ||
                        (item.url !== "/" && currentPath.startsWith(`${item.url}/`))
                      return (
                        <SidebarMenuItem key={item.title}>
                          <SidebarMenuButton
                            asChild
                            isActive={isActive}
                            onClick={handleNavItemClick}
                            data-tour={item.url === "/models" ? "models-nav" : undefined}
                            className={`h-9 px-3 rounded-md transition-all duration-200 ${
                              isActive
                                ? "bg-[rgba(0,212,255,0.1)] text-[#00d4ff] font-medium border-l-2 border-[#00d4ff] shadow-[inset_0_0_10px_rgba(0,212,255,0.03)]"
                                : "text-[#64748b] hover:bg-[rgba(0,212,255,0.05)] hover:text-[#94a3b8]"
                            }`}
                          >
                            <Link to={item.url}>
                              <item.icon
                                className={`size-4 ${isActive ? "text-[#00d4ff]" : "opacity-50"}`}
                              />
                              <span className={isActive ? "opacity-100" : "opacity-80"}>
                                {item.translateTitle === false ? item.title : t(item.title)}
                              </span>
                            </Link>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      )
                    })}
                    {group.isChannelsGroup && hasMoreChannels && (
                      <SidebarMenuItem key="channels-more-toggle">
                        <SidebarMenuButton
                          onClick={toggleShowAllChannels}
                          className="h-9 px-3 text-[#64748b] hover:bg-[rgba(0,212,255,0.05)] hover:text-[#94a3b8]"
                        >
                          {showAllChannels ? (
                            <IconChevronsUp className="size-4 opacity-50" />
                          ) : (
                            <IconChevronsDown className="size-4 opacity-50" />
                          )}
                          <span className="opacity-70">
                            {showAllChannels
                              ? t("navigation.show_less_channels")
                              : t("navigation.show_more_channels")}
                          </span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    )}
                  </SidebarMenu>
                </SidebarGroupContent>
              </CollapsibleContent>
            </SidebarGroup>
          </Collapsible>
        ))}
      </SidebarContent>

      {/* Bottom system info */}
      <div className="border-t-[rgba(0,212,255,0.1)] border-t px-3 py-3">
        <div className="flex items-center gap-2 text-[10px] tracking-wider text-[rgba(0,212,255,0.3)]">
          <div className="h-1.5 w-1.5 rounded-full bg-[#10b981] shadow-[0_0_4px_rgba(16,185,129,0.5)]" />
          <span className="font-mono uppercase">AFRICA v1.0</span>
        </div>
      </div>

      <SidebarRail className="bg-[rgba(0,212,255,0.1)]" />
    </Sidebar>
  )
}