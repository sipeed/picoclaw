import { useLayoutEffect, useRef } from "react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/page-header"

import { ToolLibraryTab } from "./tool-library-tab"
import { ToolsTabs } from "./tools-tabs"
import { useToolsPage } from "./use-tools-page"
import { WebSearchTab } from "./web-search-tab"

export function ToolsPage() {
  const { t } = useTranslation()
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  const {
    activeTab,
    expandedProvider,
    groupedTools,
    pendingToolName,
    providerLabelMap,
    searchQuery,
    statusFilter,
    tools,
    totalFilteredCount,
    webSearchDraft,
    hasToolsError,
    hasWebSearchError,
    isToolsLoading,
    isWebSearchLoading,
    isWebSearchSaving,
    isWebSearchDirty,
    setActiveTab,
    setSearchQuery,
    setStatusFilter,
    saveWebSearchConfig,
    toggleExpandedProvider,
    toggleTool,
    updateWebSearchDraft,
  } = useToolsPage()

  useLayoutEffect(() => {
    scrollContainerRef.current?.scrollTo({ top: 0 })
  }, [activeTab])

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title={t("navigation.tools", "Tools")} />
      <ToolsTabs activeTab={activeTab} onChange={setActiveTab} />

      <div
        ref={scrollContainerRef}
        className="flex-1 overflow-auto px-6 py-6 pb-20"
      >
        <div className="mx-auto w-full max-w-6xl">
          {activeTab === "library" ? (
            <ToolLibraryTab
              allTools={tools}
              groupedTools={groupedTools}
              totalFilteredCount={totalFilteredCount}
              searchQuery={searchQuery}
              statusFilter={statusFilter}
              isLoading={isToolsLoading}
              hasError={hasToolsError}
              pendingToolName={pendingToolName}
              onSearchQueryChange={setSearchQuery}
              onStatusFilterChange={setStatusFilter}
              onOpenWebSearchSettings={() => setActiveTab("web-search")}
              onToggleTool={toggleTool}
            />
          ) : (
            <WebSearchTab
              draft={webSearchDraft}
              providerLabelMap={providerLabelMap}
              expandedProvider={expandedProvider}
              isLoading={isWebSearchLoading}
              hasError={hasWebSearchError}
              isSaving={isWebSearchSaving}
              isDirty={isWebSearchDirty}
              onSave={saveWebSearchConfig}
              onToggleProviderExpand={toggleExpandedProvider}
              onUpdateDraft={updateWebSearchDraft}
            />
          )}
        </div>
      </div>
    </div>
  )
}
