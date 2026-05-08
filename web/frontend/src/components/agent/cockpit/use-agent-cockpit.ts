import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { getPicoMemoryGraph, getPicoSubagents } from "@/api/pico"
import { getTools, getWebSearchConfig, setToolEnabled } from "@/api/tools"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"
import { useCockpitSkills } from "@/hooks/use-cockpit-skills"
import { listAgents, Agent } from "@/api/agents"

type ToolStatusFilter = "all" | "enabled" | "disabled" | "blocked"

export function useAgentCockpit(sessionId: string) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [searchQuery, setSearchQuery] = useState("")
  const [statusFilter, setStatusFilter] = useState<ToolStatusFilter>("all")
  const [activeTab, setActiveTab] = useState<"tools" | "skills" | "agents" | "research">("tools")
  const deferredSearchQuery = useDeferredValue(searchQuery)

  const toolsQuery = useQuery({
    queryKey: ["tools"],
    queryFn: getTools,
  })
  const webSearchQuery = useQuery({
    queryKey: ["tools", "web-search-config"],
    queryFn: getWebSearchConfig,
  })
  const subagentsQuery = useQuery({
    queryKey: ["pico", "subagents", sessionId],
    queryFn: () => getPicoSubagents(sessionId),
    enabled: Boolean(sessionId),
    refetchInterval: 3000,
  })
  const memoryGraphQuery = useQuery({
    queryKey: ["pico", "memory-graph", sessionId],
    queryFn: () => getPicoMemoryGraph(sessionId),
    enabled: Boolean(sessionId),
    refetchInterval: 10000,
  })

  // -- Agents query for integration into cockpit
  const agentsQuery = useQuery({
    queryKey: ["agents"],
    queryFn: listAgents,
  })

  const {
    skills,
    isLoading: skillsLoading,
    isError: skillsError,
  } = useCockpitSkills()

  const toggleToolMutation = useMutation({
    mutationFn: async ({ name, enabled }: { name: string; enabled: boolean }) =>
      setToolEnabled(name, enabled),
    onSuccess: async (_, variables) => {
      const gateway = await refreshGatewayState({ force: true })
      showSaveSuccessOrRestartToast(
        t,
        variables.enabled
          ? t("pages.agent.tools.enable_success", "Tool enabled successfully")
          : t(
              "pages.agent.tools.disable_success",
              "Tool disabled successfully",
            ),
        "Agent Cockpit",
        gateway?.restartRequired === true,
      )
      void queryClient.invalidateQueries({ queryKey: ["tools"] })
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t("pages.agent.tools.toggle_error", "Failed to toggle tool"),
      )
    },
  })

  const tools = toolsQuery.data?.tools ?? []
  const agents = agentsQuery.data?.agents ?? []
  const normalizedSearchQuery = deferredSearchQuery.trim().toLowerCase()

  const groupedTools = useMemo(() => {
    const groups = new Map<string, typeof tools>()

    for (const tool of tools) {
      if (statusFilter !== "all" && tool.status !== statusFilter) {
        continue
      }

      if (normalizedSearchQuery) {
        const haystack = `${tool.name} ${tool.description}`.toLowerCase()
        if (!haystack.includes(normalizedSearchQuery)) {
          continue
        }
      }

      const items = groups.get(tool.category) ?? []
      items.push(tool)
      groups.set(tool.category, items)
    }

    return Array.from(groups.entries())
  }, [normalizedSearchQuery, statusFilter, tools])

  const categoryCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const tool of tools) {
      counts.set(tool.category, (counts.get(tool.category) ?? 0) + 1)
    }
    return Array.from(counts.entries())
  }, [tools])

  const statusCounts = useMemo(() => {
    return {
      all: tools.length,
      enabled: tools.filter((tool) => tool.status === "enabled").length,
      disabled: tools.filter((tool) => tool.status === "disabled").length,
      blocked: tools.filter((tool) => tool.status === "blocked").length,
    }
  }, [tools])

    return {
      categoryCounts,
      groupedTools,
      pendingToolName: toggleToolMutation.isPending
        ? (toggleToolMutation.variables?.name ?? null)
        : null,
      searchQuery,
      sessionMemoryGraph: memoryGraphQuery.data ?? null,
      sessionSubagents: subagentsQuery.data?.tasks ?? [],
      statusCounts,
      statusFilter,
      tools,
      activeTab,
      setActiveTab,
      hasMemoryGraphError: memoryGraphQuery.error != null,
      webSearchConfig: webSearchQuery.data ?? null,
      hasSubagentsError: subagentsQuery.error != null,
      hasToolsError: toolsQuery.error != null,
      isMemoryGraphLoading: memoryGraphQuery.isLoading,
      isSubagentsLoading: subagentsQuery.isLoading,
      isToolsLoading: toolsQuery.isLoading,
      isWebSearchLoading: webSearchQuery.isLoading,
      skills,
      skillsLoading,
      skillsError,
      setSearchQuery,
      setStatusFilter,
      toggleTool: (name: string, enabled: boolean) =>
        toggleToolMutation.mutate({ name, enabled }),
      // -- Agents integration
      agents,
      agentsLoading: agentsQuery.isLoading,
      agentsError: agentsQuery.isError,
    }
}
