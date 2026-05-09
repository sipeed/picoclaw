import { IconTrash, IconWorld, IconFolder } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"
import type { SkillSupportItem } from "@/api/skills"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

interface SkillCardProps {
  skill: SkillSupportItem
  onDelete: () => void
  onView?: () => void
}

export function SkillCard({ skill, onDelete, onView: _onView }: SkillCardProps) {
  void _onView // avoid unused warning
  const { t } = useTranslation()

  function originKindLabel(kind: string): string {
    switch (kind) {
      case "builtin": return "Built-in"
      case "third_party": return "Third Party"
      case "manual": return "Manual"
      default: return kind
    }
  }

  const kindColor = skill.origin_kind === "builtin"
    ? "bg-blue-500/20 text-blue-400"
    : skill.origin_kind === "third_party"
    ? "bg-purple-500/20 text-purple-400"
    : "bg-gray-500/20 text-gray-400"

  return (
    <Card size="sm">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <CardTitle className="text-base font-semibold tracking-tight">
                {skill.name}
              </CardTitle>
              <Badge className={cn("text-[8px] uppercase tracking-widest font-black px-1.5 py-0.5", kindColor)}>
                {originKindLabel(skill.origin_kind)}
              </Badge>
              {skill.installed_version && (
                <Badge className="bg-white/10 text-white/60 text-[8px] uppercase tracking-widest font-black px-1.5 py-0.5">
                  v{skill.installed_version}
                </Badge>
              )}
            </div>
            <CardDescription className="line-clamp-2 text-sm leading-relaxed">
              {skill.description}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 text-xs text-white/60">
            {skill.registry_name && (
              <span className="flex items-center gap-1">
                <IconWorld className="size-3.5" />
                {skill.registry_name}
              </span>
            )}
            {skill.source && (
              <span className="flex items-center gap-1">
                <IconFolder className="size-3.5" />
                {skill.source}
              </span>
            )}
          </div>
          {skill.origin_kind === "manual" && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={onDelete}
              className="text-white/60 hover:text-destructive hover:bg-destructive/10"
              title={t("common.delete")}
            >
              <IconTrash className="size-3.5" />
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}