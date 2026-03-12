import { IconLoader2 } from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ToolSupportItem, getTools, setToolEnabled } from "@/api/tools"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { cn } from "@/lib/utils"

export function ToolsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ["tools"],
    queryFn: getTools,
  })

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
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.tools.toggle_error"),
      )
    },
  })

  const groupedTools = (() => {
    if (!data) return [] as Array<[string, ToolSupportItem[]]>
    const buckets = new Map<string, ToolSupportItem[]>()
    for (const item of data.tools) {
      const list = buckets.get(item.category) ?? []
      list.push(item)
      buckets.set(item.category, list)
    }
    return Array.from(buckets.entries())
  })()

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.tools")} />

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.agent.load_error")}
            </div>
          ) : (
            <section className="space-y-5">
              <p className="text-muted-foreground mt-1 text-sm">
                {t("pages.agent.tools.description")}
              </p>

              {data?.tools.length ? (
                groupedTools.map(([category, items]) => (
                  <div key={category} className="space-y-3">
                    <div className="text-foreground/85 text-sm font-semibold tracking-wide">
                      {t(`pages.agent.tools.categories.${category}`)}
                    </div>
                    <div className="grid gap-4 lg:grid-cols-2">
                      {items.map((tool) => {
                        const reasonText = tool.reason_code
                          ? t(`pages.agent.tools.reasons.${tool.reason_code}`)
                          : ""
                        const isPending =
                          toggleMutation.isPending &&
                          toggleMutation.variables?.name === tool.name
                        const nextEnabled = tool.status !== "enabled"

                        return (
                          <Card
                            key={tool.name}
                            className={cn(
                              "gap-4 border transition-colors",
                              tool.status === "enabled" &&
                                "border-emerald-200/70 bg-emerald-50/50",
                              tool.status === "blocked" &&
                                "border-amber-200/80 bg-amber-50/60",
                              tool.status === "disabled" &&
                                "border-border/60 bg-card/70",
                            )}
                            size="sm"
                          >
                            <CardHeader>
                              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                                <div className="min-w-0 flex-1">
                                  <CardTitle className="font-mono text-sm break-all">
                                    {tool.name}
                                  </CardTitle>
                                  <CardDescription className="mt-1 break-words">
                                    {tool.description}
                                  </CardDescription>
                                </div>
                                <div className="flex shrink-0 items-center gap-2 self-start">
                                  <ToolStatusBadge status={tool.status} />
                                  <Button
                                    variant={
                                      nextEnabled ? "default" : "outline"
                                    }
                                    size="sm"
                                    disabled={isPending}
                                    onClick={() =>
                                      toggleMutation.mutate({
                                        name: tool.name,
                                        enabled: nextEnabled,
                                      })
                                    }
                                  >
                                    {isPending ? (
                                      <IconLoader2 className="size-4 animate-spin" />
                                    ) : null}
                                    {nextEnabled
                                      ? t("pages.agent.tools.enable")
                                      : t("pages.agent.tools.disable")}
                                  </Button>
                                </div>
                              </div>
                            </CardHeader>
                            <CardContent className="space-y-2">
                              <div className="text-muted-foreground text-xs">
                                {t("pages.agent.tools.config_key", {
                                  key: tool.config_key,
                                })}
                              </div>
                              {reasonText ? (
                                <div className="text-sm text-amber-800">
                                  {reasonText}
                                </div>
                              ) : null}
                            </CardContent>
                          </Card>
                        )
                      })}
                    </div>
                  </div>
                ))
              ) : (
                <Card className="border-dashed">
                  <CardContent className="text-muted-foreground py-10 text-center text-sm">
                    {t("pages.agent.tools.empty")}
                  </CardContent>
                </Card>
              )}
            </section>
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
        "shrink-0 rounded-md px-2 py-1 text-[11px] font-semibold",
        status === "enabled" && "bg-emerald-100 text-emerald-700",
        status === "blocked" && "bg-amber-100 text-amber-700",
        status === "disabled" && "bg-muted text-muted-foreground",
      )}
    >
      {t(`pages.agent.tools.status.${status}`)}
    </span>
  )
}
