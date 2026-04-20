import { useTranslation } from "react-i18next"

import { cn } from "@/lib/utils"

import type { ToolsPageTab } from "./types"

interface ToolsTabsProps {
  activeTab: ToolsPageTab
  onChange: (tab: ToolsPageTab) => void
}

const tabs: Array<{
  defaultLabel: string
  key: ToolsPageTab
  translationKey: string
}> = [
  {
    key: "library",
    translationKey: "pages.agent.tools.library_title",
    defaultLabel: "Tool Library",
  },
  {
    key: "web-search",
    translationKey: "pages.agent.tools.web_search.title",
    defaultLabel: "Web Search",
  },
]

export function ToolsTabs({ activeTab, onChange }: ToolsTabsProps) {
  const { t } = useTranslation()

  return (
    <div className="border-border/60 border-b px-6 pt-2">
      <div className="flex gap-8">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => onChange(tab.key)}
            className={cn(
              "hover:text-foreground relative cursor-pointer pb-4 text-[14px] font-medium transition-colors outline-none",
              activeTab === tab.key
                ? "text-foreground"
                : "text-muted-foreground",
            )}
          >
            {t(tab.translationKey, tab.defaultLabel)}
            {activeTab === tab.key && (
              <span className="bg-primary absolute inset-x-0 bottom-0 h-[2px] rounded-t-full" />
            )}
          </button>
        ))}
      </div>
    </div>
  )
}
