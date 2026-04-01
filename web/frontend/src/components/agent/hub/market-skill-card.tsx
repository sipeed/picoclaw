import {
  IconCheck,
  IconFileInfo,
  IconLoader2,
  IconPlus,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import {
  type SkillRegistrySearchResult,
  type SkillSupportItem,
} from "@/api/skills"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

export function MarketSkillCard({
  result,
  canInstall,
  installPending,
  installedSkill,
  onInstall,
  onViewInstalled,
}: {
  result: SkillRegistrySearchResult
  canInstall: boolean
  installPending: boolean
  installedSkill: SkillSupportItem | null
  onInstall: () => void
  onViewInstalled: () => void
}) {
  const { t } = useTranslation()

  return (
    <Card
      className="group relative overflow-hidden border-border/40 bg-card/40 transition-all hover:border-border/80 hover:bg-card hover:shadow-md"
      size="sm"
    >
      {result.installed && (
        <div className="absolute inset-x-0 top-0 h-1 bg-emerald-500/20" />
      )}
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1 space-y-2">
            <div className="mb-1 flex flex-wrap items-center gap-2">
              <CardTitle className="text-base font-semibold tracking-tight">
                {result.display_name || result.slug}
              </CardTitle>
              <span className="inline-flex items-center rounded-md bg-muted/60 px-2 py-0.5 text-[10px] font-semibold tracking-wider text-muted-foreground uppercase ring-1 ring-inset ring-border/50">
                {result.registry_name}
              </span>
              {result.installed ? (
                <span className="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-600 ring-1 ring-inset ring-emerald-500/20">
                  {t("pages.agent.skills.marketplace_installed")}
                </span>
              ) : null}
            </div>
            <div className="font-mono text-xs text-muted-foreground opacity-80">
              {result.slug}
              {result.version ? (
                <span className="text-muted-foreground/60">
                  {" "}
                  · v{result.version}
                </span>
              ) : null}
            </div>
            <CardDescription className="mt-2 line-clamp-2 text-sm leading-relaxed">
              {result.summary}
            </CardDescription>
            {result.url ? (
              <div className="pt-1">
                <a
                  href={result.url}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex text-xs text-primary/80 transition-colors hover:text-primary hover:underline hover:underline-offset-4"
                >
                  {result.url}
                </a>
              </div>
            ) : null}
          </div>
          <div className="flex shrink-0 flex-col items-end gap-2">
            <Button
              size="sm"
              variant={result.installed ? "secondary" : "default"}
              className="shadow-sm transition-all"
              disabled={!canInstall || result.installed || installPending}
              onClick={onInstall}
            >
              {installPending ? (
                <IconLoader2 className="size-4 animate-spin" />
              ) : result.installed ? (
                <IconCheck className="size-4" />
              ) : (
                <IconPlus className="size-4" />
              )}
              {result.installed
                ? t("pages.agent.skills.marketplace_installed")
                : t("pages.agent.skills.marketplace_install_action")}
            </Button>
            {result.installed && installedSkill ? (
              <Button
                variant="outline"
                size="xs"
                onClick={onViewInstalled}
                className="w-full shadow-sm hover:bg-muted"
              >
                <IconFileInfo className="mr-1 size-3.5" />
                {t("pages.agent.skills.marketplace_view_installed")}
              </Button>
            ) : null}
          </div>
        </div>
      </CardHeader>
      {result.installed_name ? (
        <CardContent className="pt-0 pb-4">
          <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-xs text-emerald-700 dark:text-emerald-400">
            {t("pages.agent.skills.marketplace_installed_hint", {
              name: result.installed_name,
            })}
          </div>
        </CardContent>
      ) : null}
    </Card>
  )
}
