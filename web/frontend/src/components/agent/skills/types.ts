import type { SkillSupportItem } from "@/api/skills"

export type SkillSortOption = "name-asc" | "name-desc" | "source"
export type SkillLayoutMode = "grouped" | "grid"
export type SkillDetailView = "preview" | "raw" | "meta"

export interface SkillGroupSection {
  origin: string
  skills: SkillSupportItem[]
}

export interface SkillStatItem {
  key: string
  origin: string
  label: string
  count: number
}
