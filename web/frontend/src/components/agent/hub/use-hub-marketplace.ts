import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { useNavigate } from "@tanstack/react-router"
import { useEffect, useRef, useState, type UIEvent } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  getSkills,
  installSkill,
  searchSkills,
  type SkillSearchResponse,
  type SkillRegistrySearchResult,
  type SkillSupportItem,
} from "@/api/skills"
import { getTools } from "@/api/tools"

import { buildUnavailableToolMessages } from "./tool-support"

const MARKET_SEARCH_LIMIT = 20

export function useHubMarketplace() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isLoadMoreLockedRef = useRef(false)

  const [marketQuery, setMarketQuery] = useState("")
  const [submittedMarketQuery, setSubmittedMarketQuery] = useState("")

  const { data: skillsData } = useQuery({
    queryKey: ["skills"],
    queryFn: getSkills,
  })
  const { data: toolsData } = useQuery({
    queryKey: ["tools"],
    queryFn: getTools,
  })

  const findSkillsTool = toolsData?.tools.find(
    (tool) => tool.name === "find_skills",
  )
  const installSkillTool = toolsData?.tools.find(
    (tool) => tool.name === "install_skill",
  )
  const canSearchMarketplace = findSkillsTool?.status === "enabled"
  const canInstallFromMarketplace = installSkillTool?.status === "enabled"
  const hasSubmittedQuery = submittedMarketQuery.trim() !== ""
  const isMarketSearchActive = canSearchMarketplace && hasSubmittedQuery

  const {
    data: marketSearchData,
    isPending: isMarketSearchPending,
    isFetching: isMarketSearchFetching,
    isFetchingNextPage,
    error: marketSearchError,
    hasNextPage,
    fetchNextPage,
    refetch: refetchMarketSearch,
  } = useInfiniteQuery({
    queryKey: ["skills-marketplace", submittedMarketQuery],
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      searchSkills(
        submittedMarketQuery,
        MARKET_SEARCH_LIMIT,
        Number(pageParam) || 0,
      ),
    getNextPageParam: (lastPage: SkillSearchResponse) =>
      lastPage.has_more ? lastPage.next_offset ?? undefined : undefined,
    enabled: isMarketSearchActive,
    staleTime: 5 * 60 * 1000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  })

  const installMutation = useMutation({
    mutationFn: installSkill,
    onSuccess: (response) => {
      toast.success(
        t("pages.agent.skills.install_success", {
          name: response.skill?.name ?? response.slug,
        }),
      )
      void queryClient.invalidateQueries({ queryKey: ["skills"] })
      void queryClient.invalidateQueries({ queryKey: ["skills-marketplace"] })
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.skills.install_error"),
      )
    },
  })

  const allSkills = skillsData?.skills ?? []
  const workspaceSkillsByName = new Map(
    allSkills
      .filter((skill) => skill.source === "workspace")
      .map((skill) => [skill.name, skill] as const),
  )
  const marketResults =
    marketSearchData?.pages.flatMap((page) => page.results) ?? []
  const hasMoreMarketResults = hasNextPage ?? false
  const isMarketSearchInitialLoading =
    isMarketSearchActive &&
    !marketSearchData &&
    (isMarketSearchPending || isMarketSearchFetching)
  const isMarketSearchLoadingMore =
    isMarketSearchActive &&
    Boolean(marketSearchData) &&
    isFetchingNextPage
  const installPendingKey =
    installMutation.isPending && installMutation.variables
      ? `${installMutation.variables.registry}:${installMutation.variables.slug}`
      : null

  const unavailableToolMessages = buildUnavailableToolMessages({
    searchTool: findSkillsTool,
    installTool: installSkillTool,
    t,
  })

  useEffect(() => {
    if (!isFetchingNextPage) {
      isLoadMoreLockedRef.current = false
    }
  }, [isFetchingNextPage])

  const handleSearchSubmit = () => {
    const nextQuery = marketQuery.trim()
    if (!canSearchMarketplace || nextQuery === "") {
      return
    }

    isLoadMoreLockedRef.current = false
    if (nextQuery === submittedMarketQuery) {
      void refetchMarketSearch()
      return
    }

    setSubmittedMarketQuery(nextQuery)
  }

  const handleInstall = (result: SkillRegistrySearchResult) => {
    installMutation.mutate({
      slug: result.slug,
      registry: result.registry_name,
      version: result.version || undefined,
    })
  }

  const handleViewInstalled = () => {
    void navigate({ to: "/agent/skills" })
  }

  const handleScroll = (event: UIEvent<HTMLDivElement>) => {
    if (
      !isMarketSearchActive ||
      !hasMoreMarketResults ||
      isFetchingNextPage ||
      isLoadMoreLockedRef.current
    ) {
      return
    }

    const node = event.currentTarget
    const remaining = node.scrollHeight - node.scrollTop - node.clientHeight
    if (remaining > 240) {
      return
    }

    isLoadMoreLockedRef.current = true
    void fetchNextPage()
  }

  const getInstalledSkill = (installedName?: string): SkillSupportItem | null => {
    if (!installedName) {
      return null
    }
    return workspaceSkillsByName.get(installedName) ?? null
  }

  const isInstallPending = (result: SkillRegistrySearchResult) =>
    installPendingKey === `${result.registry_name}:${result.slug}`

  return {
    marketQuery,
    submittedMarketQuery,
    canSearchMarketplace,
    canInstallFromMarketplace,
    marketResults,
    marketSearchError,
    unavailableToolMessages,
    hasSubmittedQuery,
    isMarketSearchInitialLoading,
    isMarketSearchLoadingMore,
    setMarketQuery,
    handleSearchSubmit,
    handleInstall,
    handleViewInstalled,
    handleScroll,
    getInstalledSkill,
    isInstallPending,
  }
}
