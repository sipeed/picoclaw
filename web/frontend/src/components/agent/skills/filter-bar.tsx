import { IconLayoutGrid, IconLayoutList, IconSearch } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

import { getOriginLabel } from "./origin-utils"
import type { SkillLayoutMode, SkillSortOption } from "./types"

interface FilterBarProps {
  searchQuery: string
  sourceFilter: string
  availableOrigins: string[]
  sortOrder: SkillSortOption
  layoutMode: SkillLayoutMode
  onSearchQueryChange: (value: string) => void
  onSourceFilterChange: (value: string) => void
  onSortOrderChange: (value: SkillSortOption) => void
  onLayoutModeChange: (value: SkillLayoutMode) => void
}

export function FilterBar({
  searchQuery,
  sourceFilter,
  availableOrigins,
  sortOrder,
  layoutMode,
  onSearchQueryChange,
  onSourceFilterChange,
  onSortOrderChange,
  onLayoutModeChange,
}: FilterBarProps) {
  const { t } = useTranslation()

  return (
    <div className="border-border/40 bg-muted/20 flex flex-wrap items-center gap-3 rounded-xl border p-2 shadow-sm">
      <div className="relative min-w-[200px] flex-1">
        <IconSearch className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
        <Input
          value={searchQuery}
          onChange={(event) => onSearchQueryChange(event.target.value)}
          placeholder={t("pages.agent.skills.search_placeholder")}
          className="hover:bg-background/50 focus-visible:bg-background h-9 border-transparent bg-transparent pl-9 shadow-none focus-visible:ring-1"
        />
      </div>

      <div className="bg-border/60 hidden h-6 w-px sm:block" />

      <Select value={sourceFilter} onValueChange={onSourceFilterChange}>
        <SelectTrigger className="hover:bg-background/50 focus:bg-background h-9 w-[140px] border-transparent bg-transparent shadow-none hover:ring-1 focus:ring-1">
          <SelectValue placeholder={t("pages.agent.skills.source_label")} />
        </SelectTrigger>
        <SelectContent align="end">
          <SelectItem value="all">
            {t("pages.agent.skills.origin.all")}
          </SelectItem>
          {availableOrigins.map((origin) => (
            <SelectItem key={origin} value={origin}>
              {getOriginLabel(origin, t)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="bg-border/60 hidden h-6 w-px sm:block" />

      <Select
        value={sortOrder}
        onValueChange={(value) => onSortOrderChange(value as SkillSortOption)}
      >
        <SelectTrigger className="hover:bg-background/50 focus:bg-background h-9 w-[160px] border-transparent bg-transparent shadow-none hover:ring-1 focus:ring-1">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-xs">
              {t("pages.agent.skills.sort_label", {
                defaultValue: "Sort by",
              })}
              :
            </span>
            <SelectValue />
          </div>
        </SelectTrigger>
        <SelectContent align="end">
          <SelectItem value="name-asc">
            {t("pages.agent.skills.sort.name_asc")}
          </SelectItem>
          <SelectItem value="name-desc">
            {t("pages.agent.skills.sort.name_desc")}
          </SelectItem>
          <SelectItem value="source">
            {t("pages.agent.skills.sort.source")}
          </SelectItem>
        </SelectContent>
      </Select>

      <div className="bg-border/60 hidden h-6 w-px sm:block" />

      <div className="bg-background/50 ring-border/20 inline-flex items-center rounded-lg p-0.5 shadow-sm ring-1">
        <button
          type="button"
          className={cn(
            "rounded-md px-2.5 py-1.5 text-xs font-medium transition-all",
            layoutMode === "grouped"
              ? "bg-background text-foreground ring-border/30 shadow-sm ring-1"
              : "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => onLayoutModeChange("grouped")}
        >
          <IconLayoutList className="size-4" />
        </button>
        <button
          type="button"
          className={cn(
            "rounded-md px-2.5 py-1.5 text-xs font-medium transition-all",
            layoutMode === "grid"
              ? "bg-background text-foreground ring-border/30 shadow-sm ring-1"
              : "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => onLayoutModeChange("grid")}
        >
          <IconLayoutGrid className="size-4" />
        </button>
      </div>
    </div>
  )
}
