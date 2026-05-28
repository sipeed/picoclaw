import type { TFunction } from "i18next"

import type { SkillSupportItem } from "@/api/skills"

import type { SkillSortOption } from "./types"

const KNOWN_ORIGIN_ORDER = ["builtin", "third_party", "manual"]

export function compareSkills(
  left: SkillSupportItem,
  right: SkillSupportItem,
  sortOrder: SkillSortOption,
) {
  if (sortOrder === "source") {
    const sourceDelta = compareOriginOrder(
      getSkillOriginKind(left),
      getSkillOriginKind(right),
    )
    if (sourceDelta !== 0) return sourceDelta
    return left.name.localeCompare(right.name)
  }

  if (sortOrder === "name-desc") {
    return right.name.localeCompare(left.name)
  }

  return left.name.localeCompare(right.name)
}

export function sortOrigins(origins: string[]) {
  return [...origins].sort(compareOriginOrder)
}

export function getSkillOriginKind(skill: SkillSupportItem) {
  const origin = skill.origin_kind || skill.source
  return origin === "global" ? "builtin" : origin
}

export function getOriginLabel(origin: string, t: TFunction) {
  if (origin === "builtin" || origin === "third_party" || origin === "manual") {
    return t(`pages.agent.skills.origin.${origin}`)
  }
  if (origin === "all") {
    return t("pages.agent.skills.origin.all")
  }
  return origin
}

export function getOriginAccentClasses(origin: string) {
  if (origin === "manual") {
    return "bg-emerald-100 text-emerald-700"
  }
  if (origin === "third_party") {
    return "bg-sky-100 text-sky-700"
  }
  if (origin === "builtin") {
    return "bg-amber-100 text-amber-700"
  }
  return "bg-muted text-muted-foreground"
}

export function getOriginBadgeClasses(origin: string) {
  if (origin === "manual") {
    return "bg-emerald-100 text-emerald-700"
  }
  if (origin === "third_party") {
    return "bg-sky-100 text-sky-700"
  }
  if (origin === "builtin") {
    return "bg-amber-100 text-amber-700"
  }
  return "bg-muted text-muted-foreground"
}

function compareOriginOrder(left: string, right: string) {
  const leftIndex = KNOWN_ORIGIN_ORDER.indexOf(left)
  const rightIndex = KNOWN_ORIGIN_ORDER.indexOf(right)

  if (leftIndex !== -1 || rightIndex !== -1) {
    if (leftIndex === -1) return 1
    if (rightIndex === -1) return -1
    return leftIndex - rightIndex
  }

  return left.localeCompare(right)
}
