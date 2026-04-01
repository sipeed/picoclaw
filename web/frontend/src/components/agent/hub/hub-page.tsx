import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/page-header"

import { ResultsPanel } from "./results-panel"
import { SearchPanel } from "./search-panel"
import { useHubMarketplace } from "./use-hub-marketplace"

export function HubPage() {
  const { t } = useTranslation()
  const hub = useHubMarketplace()

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.hub")} />

      <div
        className="flex-1 overflow-auto px-6 py-6"
        onScroll={hub.handleScroll}
      >
        <div className="mx-auto w-full max-w-[1000px] space-y-8">
          <section className="animate-in fade-in mx-auto flex w-full flex-col items-center space-y-8 duration-300 md:duration-500">
            <SearchPanel
              marketQuery={hub.marketQuery}
              canSearchMarketplace={hub.canSearchMarketplace}
              isMarketSearchInitialLoading={hub.isMarketSearchInitialLoading}
              unavailableToolMessages={hub.unavailableToolMessages}
              onMarketQueryChange={hub.setMarketQuery}
              onSearchSubmit={hub.handleSearchSubmit}
            />

            <ResultsPanel
              canSearchMarketplace={hub.canSearchMarketplace}
              hasSubmittedQuery={hub.hasSubmittedQuery}
              submittedQuery={hub.submittedMarketQuery}
              marketResults={hub.marketResults}
              marketSearchError={hub.marketSearchError}
              isMarketSearchInitialLoading={hub.isMarketSearchInitialLoading}
              isMarketSearchLoadingMore={hub.isMarketSearchLoadingMore}
              canInstallFromMarketplace={hub.canInstallFromMarketplace}
              getInstalledSkill={hub.getInstalledSkill}
              isInstallPending={hub.isInstallPending}
              onInstall={hub.handleInstall}
              onViewInstalled={hub.handleViewInstalled}
            />
          </section>
        </div>
      </div>
    </div>
  )
}
