import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"
import {
  listSkills,
  deleteSkill,
} from "@/api/skills"

export function useCockpitSkills() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const skillsQuery = useQuery({
    queryKey: ["skills"],
    queryFn: listSkills,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteSkill(name),
    onSuccess: () => {
      toast.success(t("pages.agent.skills.delete_success", "Skill deleted"))
      void queryClient.invalidateQueries({ queryKey: ["skills"] })
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to delete skill")
    },
  })

  return {
    skills: skillsQuery.data?.skills ?? [],
    isLoading: skillsQuery.isLoading,
    isError: skillsQuery.isError,
    deleteSkill: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  }
}