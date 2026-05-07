import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import type { SkillSupportItem } from "@/api/skills"
import { useCockpitSkills } from "@/hooks/use-cockpit-skills"
import { PageHeader } from "@/components/page-header"
import { Input } from "@/components/ui/input"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { SkillCard } from "./skill-card"

interface SkillsPageProps {
  embedded?: boolean
}

export function SkillsPage({ embedded = false }: SkillsPageProps) {
  const { t } = useTranslation()
  const {
    skills,
    isLoading,
    isError,
    deleteSkill,
  } = useCockpitSkills()

  const [searchQuery, setSearchQuery] = useState("")
  const deferredSearchQuery = useDeferredValue(searchQuery)
  const [skillToDelete, setSkillToDelete] = useState<SkillSupportItem | null>(null)

  const filteredSkills = useMemo(() => {
    const query = deferredSearchQuery.trim().toLowerCase()
    if (!query) return skills
    return skills.filter(
      (skill) =>
        skill.name.toLowerCase().includes(query) ||
        skill.description.toLowerCase().includes(query) ||
        skill.origin_kind.toLowerCase().includes(query)
    )
  }, [skills, deferredSearchQuery])

  const handleDelete = async () => {
    if (!skillToDelete) return
    try {
      await deleteSkill(skillToDelete.name)
    } catch {
      // Error handled by hook
    } finally {
      setSkillToDelete(null)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-muted-foreground">Loading skills...</p>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-destructive">Failed to load skills. Please try again.</p>
      </div>
    )
  }

  const mainContent = (
    <div className="mx-auto w-full max-w-6xl">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="max-w-md flex-1">
          <Input
            placeholder={t("common.search", "Search skills...")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="bg-background"
          />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {filteredSkills.length === 0 ? (
          <div className="col-span-full rounded-lg border border-dashed border-white/10 p-12 text-center">
            <p className="text-muted-foreground">
              {deferredSearchQuery
                ? t("pages.agent.skills.no_results", "No skills found")
                : t("pages.agent.skills.no_skills", "No skills installed")}
            </p>
            <p className="mt-2 text-sm text-muted-foreground/60">
              {deferredSearchQuery
                ? t("pages.agent.skills.no_results_hint", "Try a different search")
                : t("pages.agent.skills.no_skills_hint", "Install skills via the CLI or import them")}
            </p>
          </div>
        ) : (
          filteredSkills.map((skill) => (
            <SkillCard
              key={skill.name}
              skill={skill}
              onDelete={() => setSkillToDelete(skill)}
            />
          ))
        )}
      </div>
    </div>
  )

  const modals = (
    <AlertDialog
      open={skillToDelete !== null}
      onOpenChange={(open) => { if (!open) setSkillToDelete(null) }}
    >
      <AlertDialogContent>
        <AlertDialogTitle>
          {t("pages.agent.skills.confirm_delete", "Delete Skill?")}
        </AlertDialogTitle>
        <p className="text-muted-foreground">
          {t(
            "pages.agent.skills.confirm_delete_message",
            `Are you sure you want to delete "${skillToDelete?.name}"? This action cannot be undone.`,
          )}
        </p>
        <AlertDialogFooter>
          <AlertDialogCancel>
            {t("common.cancel", "Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {t("common.delete", "Delete")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )

  if (embedded) {
    return (
      <>
        {mainContent}
        {modals}
      </>
    )
  }

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title={t("navigation.skills", "Skills")} />
      <div className="flex-1 overflow-auto px-6 py-6 pb-20">
        {mainContent}
      </div>
      {modals}
    </div>
  )
}
