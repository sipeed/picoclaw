import { useTranslation } from "react-i18next"

import type { WebSearchConfigResponse } from "@/api/tools"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"

import type { WebSearchDraftUpdater } from "./types"
import { WebSearchGeneralSettings } from "./web-search-general-settings"
import { WebSearchProviderSettings } from "./web-search-provider-settings"

interface WebSearchTabProps {
  draft: WebSearchConfigResponse | null
  currentProviderLabel: string
  providerLabelMap: Map<string, string>
  expandedProvider: string | null
  isLoading: boolean
  hasError: boolean
  isSaving: boolean
  onSave: () => void
  onToggleProviderExpand: (providerId: string) => void
  onUpdateDraft: WebSearchDraftUpdater
}

export function WebSearchTab({
  draft,
  currentProviderLabel,
  providerLabelMap,
  expandedProvider,
  isLoading,
  hasError,
  isSaving,
  onSave,
  onToggleProviderExpand,
  onUpdateDraft,
}: WebSearchTabProps) {
  const { t } = useTranslation()

  return (
    <div className="animate-in fade-in slide-in-from-bottom-2 space-y-12 pt-2 duration-500">
      {hasError ? (
        <div className="py-20 text-center">
          <p className="text-destructive font-medium">
            {t(
              "pages.agent.tools.web_search.load_error",
              "Failed to load web search configuration",
            )}
          </p>
        </div>
      ) : isLoading || !draft ? (
        <LoadingState />
      ) : (
        <>
          <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
            <div className="max-w-xl space-y-3">
              <div className="flex items-center gap-3">
                <h1 className="text-foreground/90 text-2xl font-semibold tracking-tight">
                  {t(
                    "pages.agent.tools.web_search.title",
                    "Web Search Configuration",
                  )}
                </h1>
                <div className="rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-[11px] font-semibold tracking-wide text-emerald-600 uppercase dark:text-emerald-400">
                  {currentProviderLabel}
                </div>
              </div>
              <p className="text-muted-foreground/80 text-[14px] leading-relaxed">
                {t(
                  "pages.agent.tools.web_search.description",
                  "Provide web search capability for agents to find the latest real-world info. Automatically routes to the optimal active provider.",
                )}
              </p>
            </div>

            <Button
              onClick={onSave}
              disabled={isSaving}
              className="h-10 shrink-0 rounded-xl px-6 shadow-sm transition-all active:scale-95"
            >
              {t("pages.agent.tools.web_search.save", "Save Changes")}
            </Button>
          </div>

          <div className="space-y-10">
            <WebSearchGeneralSettings
              draft={draft}
              onUpdateDraft={onUpdateDraft}
            />
            <WebSearchProviderSettings
              providerLabelMap={providerLabelMap}
              settings={draft.settings}
              expandedProvider={expandedProvider}
              onToggleProviderExpand={onToggleProviderExpand}
              onUpdateDraft={onUpdateDraft}
            />
          </div>
        </>
      )}
    </div>
  )
}

function LoadingState() {
  return (
    <div className="space-y-8">
      <Skeleton className="h-24 rounded-2xl" />
      <Skeleton className="h-64 rounded-2xl" />
    </div>
  )
}
