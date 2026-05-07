import { IconSearch } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { SkillSupportItem } from "@/api/skills"

import { OriginBadge } from "./origin-badge"
import { getOriginLabel } from "./origin-utils"
import { SkillCard } from "./skill-card"
import type { SkillGroupSection, SkillLayoutMode } from "./types"

interface SkillsListProps {
  sortedSkills: SkillSupportItem[]
  groupedSkills: SkillGroupSection[]
  layoutMode: SkillLayoutMode
  sourceFilter: string
  hasActiveFilters: boolean
  onViewSkill: (skill: SkillSupportItem) => void
  onDeleteSkill: (skill: SkillSupportItem) => void
}

export function SkillsList({
  sortedSkills,
  groupedSkills,
  layoutMode,
  sourceFilter,
  hasActiveFilters,
  onViewSkill,
  onDeleteSkill,
}: SkillsListProps) {
  const { t } = useTranslation()

  if (!sortedSkills.length) {
    return (
      <div className="border-border/40 bg-muted/5 flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed py-16 text-center shadow-sm">
        <div className="bg-muted mb-2 rounded-full p-4">
          <IconSearch className="text-muted-foreground size-6" />
        </div>
        <h3 className="text-lg font-semibold tracking-tight">
          {hasActiveFilters
            ? t("pages.agent.skills.no_results")
            : t("pages.agent.skills.empty")}
        </h3>
      </div>
    )
  }

  if (layoutMode === "grouped" && sourceFilter === "all") {
    return (
      <div className="space-y-6">
        {groupedSkills.map((section) => (
          <div key={section.origin} className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <OriginBadge
                origin={section.origin}
                label={getOriginLabel(section.origin, t)}
              />
            </div>
            <div className="grid gap-4 lg:grid-cols-2">
              {section.skills.map((skill) => (
                <SkillCard
                  key={`${skill.source}:${skill.name}`}
                  skill={skill}
                  onView={() => onViewSkill(skill)}
                  onDelete={() => onDeleteSkill(skill)}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {sortedSkills.map((skill) => (
        <SkillCard
          key={`${skill.source}:${skill.name}`}
          skill={skill}
          onView={() => onViewSkill(skill)}
          onDelete={() => onDeleteSkill(skill)}
        />
      ))}
    </div>
  )
}
