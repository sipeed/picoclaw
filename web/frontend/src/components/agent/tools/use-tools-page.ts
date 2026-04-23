import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  getTools,
  getWebSearchConfig,
  setToolEnabled,
  updateWebSearchConfig,
  type WebSearchConfigResponse,
} from "@/api/tools"
import { refreshGatewayState } from "@/store/gateway"

import type { GroupedTools, ToolStatusFilter, ToolsPageTab } from "./types"

export function useToolsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [activeTab, setActiveTab] = useState<ToolsPageTab>("library")
  const [searchQuery, setSearchQuery] = useState("")
  const deferredSearchQuery = useDeferredValue(searchQuery)
  const [statusFilter, setStatusFilter] = useState<ToolStatusFilter>("all")
  const [expandedProvider, setExpandedProvider] = useState<string | null>(null)
  const [webSearchDraftOverride, setWebSearchDraftOverride] =
    useState<WebSearchConfigResponse | null>(null)

  const toolsQuery = useQuery({
    queryKey: ["tools"],
    queryFn: getTools,
  })
  const webSearchQuery = useQuery({
    queryKey: ["tools", "web-search-config"],
    queryFn: getWebSearchConfig,
  })

  const tools = useMemo(() => toolsQuery.data?.tools ?? [], [toolsQuery.data?.tools])
  const normalizedSearchQuery = deferredSearchQuery.trim().toLowerCase()
  const webSearchDraft = webSearchDraftOverride ?? webSearchQuery.data ?? null

  const toggleToolMutation = useMutation({
    mutationFn: async ({ name, enabled }: { name: string; enabled: boolean }) =>
      setToolEnabled(name, enabled),
    onSuccess: (_, variables) => {
      toast.success(
        variables.enabled
          ? t("pages.agent.tools.enable_success", "Tool enabled successfully")
          : t(
              "pages.agent.tools.disable_success",
              "Tool disabled successfully",
            ),
      )
      void queryClient.invalidateQueries({ queryKey: ["tools"] })
      void refreshGatewayState({ force: true })
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t("pages.agent.tools.toggle_error", "Failed to toggle tool"),
      )
    },
  })

  const saveWebSearchMutation = useMutation({
    mutationFn: updateWebSearchConfig,
    onSuccess: (updatedConfig) => {
      queryClient.setQueryData(["tools", "web-search-config"], updatedConfig)
      setWebSearchDraftOverride(null)
      toast.success(
        t(
          "pages.agent.tools.web_search.save_success",
          "Settings saved successfully",
        ),
      )
      void queryClient.invalidateQueries({
        queryKey: ["tools", "web-search-config"],
      })
      void queryClient.invalidateQueries({ queryKey: ["tools"] })
      void refreshGatewayState({ force: true })
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t(
              "pages.agent.tools.web_search.save_error",
              "Failed to save settings",
            ),
      )
    },
  })

  const groupedTools = useMemo<{
    groupedTools: GroupedTools
    totalFilteredCount: number
  }>(() => {
    let totalFilteredCount = 0
    const grouped = new Map<string, typeof tools>()

    for (const tool of tools) {
      if (statusFilter !== "all" && tool.status !== statusFilter) {
        continue
      }

      if (normalizedSearchQuery) {
        const matchesName = tool.name.toLowerCase().includes(normalizedSearchQuery)
        const matchesDescription = (tool.description || "")
          .toLowerCase()
          .includes(normalizedSearchQuery)

        if (!matchesName && !matchesDescription) {
          continue
        }
      }

      totalFilteredCount += 1
      const items = grouped.get(tool.category) ?? []
      items.push(tool)
      grouped.set(tool.category, items)
    }

    return {
      groupedTools: Array.from(grouped.entries()),
      totalFilteredCount,
    }
  }, [normalizedSearchQuery, statusFilter, tools])

  const providerLabelMap = useMemo(() => {
    const providers = webSearchDraft?.providers ?? []
    return new Map(providers.map((provider) => [provider.id, provider.label]))
  }, [webSearchDraft])

  const pendingToolName = toggleToolMutation.isPending
    ? (toggleToolMutation.variables?.name ?? null)
    : null

  const updateWebSearchDraft = (
    updater: (current: WebSearchConfigResponse) => WebSearchConfigResponse,
  ) => {
    setWebSearchDraftOverride((current) => {
      const draft = current ?? webSearchQuery.data
      return draft ? updater(draft) : current
    })
  }

  const toggleTool = (name: string, enabled: boolean) => {
    toggleToolMutation.mutate({ name, enabled })
  }

  const saveWebSearchConfig = () => {
    if (webSearchDraft) {
      saveWebSearchMutation.mutate(webSearchDraft)
    }
  }

  const toggleExpandedProvider = (providerId: string) => {
    setExpandedProvider((current) =>
      current === providerId ? null : providerId,
    )
  }

  return {
    activeTab,
    expandedProvider,
    groupedTools: groupedTools.groupedTools,
    pendingToolName,
    providerLabelMap,
    searchQuery,
    statusFilter,
    tools,
    totalFilteredCount: groupedTools.totalFilteredCount,
    webSearchDraft,
    hasToolsError: toolsQuery.error != null,
    hasWebSearchError: webSearchQuery.error != null,
    isToolsLoading: toolsQuery.isLoading,
    isWebSearchLoading: webSearchQuery.isLoading,
    isWebSearchSaving: saveWebSearchMutation.isPending,
    setActiveTab,
    setSearchQuery,
    setStatusFilter,
    saveWebSearchConfig,
    toggleExpandedProvider,
    toggleTool,
    updateWebSearchDraft,
  }
}
