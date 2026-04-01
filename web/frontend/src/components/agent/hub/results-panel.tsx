import { IconLoader2, IconSearch, IconX } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import {
  type SkillRegistrySearchResult,
  type SkillSupportItem,
} from "@/api/skills"

import { MarketSkillCard } from "./market-skill-card"

export function ResultsPanel({
  canSearchMarketplace,
  hasSubmittedQuery,
  submittedQuery,
  marketResults,
  marketSearchError,
  isMarketSearchInitialLoading,
  isMarketSearchLoadingMore,
  canInstallFromMarketplace,
  getInstalledSkill,
  isInstallPending,
  onInstall,
  onViewInstalled,
}: {
  canSearchMarketplace: boolean
  hasSubmittedQuery: boolean
  submittedQuery: string
  marketResults: SkillRegistrySearchResult[]
  marketSearchError: unknown
  isMarketSearchInitialLoading: boolean
  isMarketSearchLoadingMore: boolean
  canInstallFromMarketplace: boolean
  getInstalledSkill: (installedName?: string) => SkillSupportItem | null
  isInstallPending: (result: SkillRegistrySearchResult) => boolean
  onInstall: (result: SkillRegistrySearchResult) => void
  onViewInstalled: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className="mx-auto flex w-full max-w-[1000px] justify-center">
      <div className="w-full">
        {canSearchMarketplace && hasSubmittedQuery ? (
          <div className="space-y-6">
            <div className="rounded-xl border border-amber-200/80 bg-amber-50/70 px-4 py-3 text-sm text-amber-900">
              <div className="font-semibold">
                {t("pages.agent.skills.marketplace_notice_title")}
              </div>
              <div className="mt-1 leading-6">
                {t("pages.agent.skills.marketplace_notice_body")}
              </div>
            </div>

            {isMarketSearchInitialLoading ? (
              <div className="border-border/40 bg-muted/10 flex min-h-[200px] flex-col items-center justify-center gap-4 rounded-xl border border-dashed">
                <IconLoader2 className="text-muted-foreground/60 size-6 animate-spin" />
                <span className="text-muted-foreground text-sm font-medium">
                  {t("pages.agent.skills.marketplace_loading_results")}
                </span>
              </div>
            ) : marketSearchError ? (
              <div className="border-destructive/20 bg-destructive/5 rounded-xl border px-6 py-5">
                <div className="text-destructive flex items-center gap-3">
                  <IconX className="size-5" />
                  <span className="text-sm font-medium">
                    {marketSearchError instanceof Error
                      ? marketSearchError.message
                      : t("pages.agent.skills.marketplace_search_error")}
                  </span>
                </div>
              </div>
            ) : marketResults.length ? (
              <div className="space-y-4">
                <div className="border-border/40 flex items-center justify-between border-b pb-4">
                  <h3 className="text-foreground/85 text-base font-semibold">
                    {t("pages.agent.skills.marketplace_results_title", {
                      query: submittedQuery,
                      count: marketResults.length,
                    })}
                  </h3>
                  <span className="text-muted-foreground text-xs font-medium">
                    {t("pages.agent.skills.marketplace_results_hint")}
                  </span>
                </div>
                <div className="grid gap-4 lg:grid-cols-2">
                  {marketResults.map((result) => (
                    <MarketSkillCard
                      key={`${result.registry_name}:${result.slug}`}
                      result={result}
                      canInstall={canInstallFromMarketplace}
                      installPending={isInstallPending(result)}
                      installedSkill={getInstalledSkill(result.installed_name)}
                      onInstall={() => onInstall(result)}
                      onViewInstalled={onViewInstalled}
                    />
                  ))}
                </div>
                {isMarketSearchLoadingMore ? (
                  <div className="text-muted-foreground flex items-center justify-center gap-2 pt-2 text-sm">
                    <IconLoader2 className="size-4 animate-spin" />
                    <span>
                      {t("pages.agent.skills.marketplace_loading_more")}
                    </span>
                  </div>
                ) : null}
              </div>
            ) : (
              <div className="border-border/40 bg-muted/10 flex min-h-[200px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed">
                <IconSearch className="text-muted-foreground/50 size-6" />
                <span className="text-muted-foreground text-sm font-medium">
                  {t("pages.agent.skills.marketplace_empty_results", {
                    query: submittedQuery,
                  })}
                </span>
              </div>
            )}
          </div>
        ) : !canSearchMarketplace ? (
          <div className="border-border/40 bg-muted/10 flex min-h-[200px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed">
            <span className="text-muted-foreground text-sm font-medium">
              {t("pages.agent.skills.marketplace_unavailable")}
            </span>
          </div>
        ) : (
          <div className="border-border/40 bg-muted/10 flex min-h-[200px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed">
            <IconSearch className="text-muted-foreground/50 size-6" />
            <span className="text-muted-foreground text-sm font-medium">
              {t("pages.agent.skills.marketplace_idle")}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
