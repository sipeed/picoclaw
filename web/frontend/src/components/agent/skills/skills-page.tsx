import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { motion } from "motion/react"
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
        <div className="flex items-center gap-2 text-cyan-100/50">
          <motion.div
            className="w-2 h-2 rounded-full bg-[#00bcff]"
            animate={{ opacity: [0.3, 1, 0.3] }}
            transition={{ duration: 1.5, repeat: Infinity }}
          />
          Loading skills...
        </div>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-red-400">Failed to load skills. Please try again.</p>
      </div>
    )
  }

  const mainContent = (
    <div className="w-full">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="max-w-md flex-1">
          <div className="relative">
            <Input
              placeholder={t("common.search", "Search skills...")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="bg-black/40 border-[#00bcff]/20 text-cyan-100 placeholder:text-cyan-100/30 focus:border-[#00bcff] focus:ring-[#00bcff]/20 backdrop-blur-sm"
            />
          </div>
        </div>
        <div className="text-[9px] uppercase tracking-widest text-cyan-100/40 font-bold">
          {filteredSkills.length} Skills Loaded
        </div>
      </div>

      <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
        {filteredSkills.length === 0 ? (
          <div className="col-span-full rounded-xl border border-dashed border-[#00bcff]/15 p-12 text-center bg-black/20 backdrop-blur-sm">
            <p className="text-cyan-100/50">
              {deferredSearchQuery
                ? t("pages.agent.skills.no_results", "No skills found")
                : t("pages.agent.skills.no_skills", "No skills installed")}
            </p>
            <p className="mt-2 text-[11px] text-cyan-100/30">
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
      <AlertDialogContent className="bg-[#0a0e27] border-[#00bcff]/20 text-cyan-100 shadow-[0_0_40px_#00bcff1a]">
        <AlertDialogTitle>
          {t("pages.agent.skills.confirm_delete", "Delete Skill?")}
        </AlertDialogTitle>
        <p className="text-cyan-100/50">
          {t(
            "pages.agent.skills.confirm_delete_message",
            `Are you sure you want to delete "${skillToDelete?.name}"? This action cannot be undone.`,
          )}
        </p>
        <AlertDialogFooter>
          <AlertDialogCancel className="border-[#00bcff]/20 text-cyan-100/60 hover:text-cyan-100 hover:bg-[#00bcff]/5">
            {t("common.cancel", "Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            className="bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30"
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
    <div className="bg-[#0a0e27] flex h-full flex-col">
      <PageHeader title={t("navigation.skills", "Skills")} />
      <div className="flex-1 overflow-auto px-6 py-6 pb-20">
        {mainContent}
      </div>
      {modals}
    </div>
  )
}