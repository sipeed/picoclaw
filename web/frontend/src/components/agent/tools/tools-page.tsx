import { IconSearch } from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ToolSupportItem, getTools, setToolEnabled } from "@/api/tools"
import { PageHeader } from "@/components/page-header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import { refreshGatewayState } from "@/store/gateway"

export function ToolsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ["tools"],
    queryFn: getTools,
  })

  const [searchQuery, setSearchQuery] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")

  const toggleMutation = useMutation({
    mutationFn: async ({ name, enabled }: { name: string; enabled: boolean }) =>
      setToolEnabled(name, enabled),
    onSuccess: (_, variables) => {
      toast.success(
        variables.enabled
          ? t("pages.agent.tools.enable_success")
          : t("pages.agent.tools.disable_success"),
      )
      void queryClient.invalidateQueries({ queryKey: ["tools"] })
      void refreshGatewayState({ force: true })
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.tools.toggle_error"),
      )
    },
  })

  // Filter and group tools
  const { groupedTools, totalFilteredCount } = useMemo(() => {
    if (!data) return { groupedTools: [], totalFilteredCount: 0 }

    let count = 0
    const buckets = new Map<string, ToolSupportItem[]>()

    for (const item of data.tools) {
      // Apply status filter
      if (statusFilter !== "all" && item.status !== statusFilter) continue

      // Apply search query
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase()
        const matchesName = item.name.toLowerCase().includes(query)
        const matchesDesc = (item.description || "")
          .toLowerCase()
          .includes(query)
        if (!matchesName && !matchesDesc) continue
      }

      count++
      const list = buckets.get(item.category) ?? []
      list.push(item)
      buckets.set(item.category, list)
    }

    return {
      groupedTools: Array.from(buckets.entries()),
      totalFilteredCount: count,
    }
  }, [data, searchQuery, statusFilter])

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title={t("navigation.tools")} />

      <div className="flex-1 overflow-auto px-6 py-6">
        <div className="mx-auto w-full max-w-6xl space-y-8">
          {/* Header & Description */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-end">
            {/* Filters Toolbar */}
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
              <div className="relative">
                <IconSearch className="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                <Input
                  type="text"
                  placeholder={t("pages.agent.tools.search_placeholder")}
                  className="w-full pl-9 sm:w-64"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-full sm:w-40">
                  <SelectValue
                    placeholder={t("pages.agent.tools.filter.all")}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">
                    {t("pages.agent.tools.filter.all")}
                  </SelectItem>
                  <SelectItem value="enabled">
                    {t("pages.agent.tools.filter.enabled")}
                  </SelectItem>
                  <SelectItem value="disabled">
                    {t("pages.agent.tools.filter.disabled")}
                  </SelectItem>
                  <SelectItem value="blocked">
                    {t("pages.agent.tools.filter.blocked")}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Content Area */}
          {error ? (
            <Card className="border-destructive/50 bg-destructive/10 cursor-default">
              <CardContent className="py-10 text-center">
                <p className="text-destructive font-medium">
                  {t("pages.agent.load_error")}
                </p>
              </CardContent>
            </Card>
          ) : isLoading ? (
            // Skeleton Loading State
            <div className="space-y-8">
              {[1, 2].map((groupIndex) => (
                <div key={groupIndex} className="space-y-4">
                  <Skeleton className="h-5 w-32" />
                  <div className="grid gap-4 lg:grid-cols-2">
                    {[1, 2, 3, 4].map((itemIndex) => (
                      <Card
                        key={itemIndex}
                        className="border-border/60 shadow-none"
                      >
                        <CardHeader className="pb-3">
                          <Skeleton className="mb-2 h-5 w-48" />
                          <Skeleton className="h-4 w-full" />
                          <Skeleton className="h-4 w-3/4" />
                        </CardHeader>
                        <CardContent>
                          <Skeleton className="mt-2 h-8 w-full rounded-md" />
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          ) : totalFilteredCount === 0 ? (
            // Empty State
            <Card className="bg-muted/30 cursor-default border-dashed">
              <CardContent className="flex flex-col items-center justify-center py-16 text-center text-sm">
                <div className="bg-muted mb-4 rounded-full p-4">
                  <IconSearch className="text-muted-foreground size-8" />
                </div>
                <h3 className="mb-1 text-lg font-medium">
                  {data?.tools.length === 0
                    ? t("pages.agent.tools.empty")
                    : t("pages.agent.tools.no_results")}
                </h3>
                {data?.tools.length !== 0 && (
                  <p className="text-muted-foreground">
                    Try adjusting your search criteria or status filters.
                  </p>
                )}
              </CardContent>
            </Card>
          ) : (
            // Tool Categories list
            <div className="space-y-8">
              {groupedTools.map(([category, items]) => (
                <div key={category} className="space-y-4">
                  <h3 className="text-foreground text-sm font-semibold tracking-wide uppercase">
                    {t(`pages.agent.tools.categories.${category}`)}
                  </h3>
                  <div className="grid gap-4 lg:grid-cols-2">
                    {items.map((tool) => {
                      const reasonText = tool.reason_code
                        ? t(`pages.agent.tools.reasons.${tool.reason_code}`)
                        : ""
                      const isPending =
                        toggleMutation.isPending &&
                        toggleMutation.variables?.name === tool.name
                      const isEnabled = tool.status === "enabled"
                      const isDisabled = tool.status === "disabled"
                      const isBlocked = tool.status === "blocked"

                      return (
                        <Card
                          key={tool.name}
                          className={cn(
                            "group cursor-default transition-colors",
                            isBlocked
                              ? "border-amber-200/80 bg-amber-50/60 dark:border-amber-900/50 dark:bg-amber-950/20"
                              : "border-border/60",
                            isDisabled && "opacity-80",
                          )}
                        >
                          <CardHeader className="pb-3">
                            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-2">
                                  <CardTitle className="font-mono text-sm font-semibold break-all">
                                    {tool.name}
                                  </CardTitle>
                                  <ToolStatusBadge status={tool.status} />
                                </div>
                                <CardDescription className="text-muted-foreground/80 mt-2 text-xs leading-relaxed break-words sm:text-sm">
                                  {tool.description}
                                </CardDescription>
                              </div>
                              <div className="flex shrink-0 items-center pt-1 pl-2 sm:pt-0">
                                <Switch
                                  checked={isEnabled}
                                  disabled={isPending}
                                  onCheckedChange={(checked) =>
                                    toggleMutation.mutate({
                                      name: tool.name,
                                      enabled: checked,
                                    })
                                  }
                                />
                              </div>
                            </div>
                          </CardHeader>
                          {reasonText && (
                            <CardContent className="pt-0 pb-4">
                              <div className="text-xs font-medium text-amber-700 dark:text-amber-400">
                                {reasonText}
                              </div>
                            </CardContent>
                          )}
                        </Card>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function ToolStatusBadge({ status }: { status: ToolSupportItem["status"] }) {
  const { t } = useTranslation()

  return (
    <span
      className={cn(
        "shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium tracking-wide sm:text-[11px]",
        status === "enabled" &&
          "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
        status === "blocked" &&
          "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400",
        status === "disabled" && "bg-muted text-muted-foreground",
      )}
    >
      {t(`pages.agent.tools.status.${status}`)}
    </span>
  )
}
