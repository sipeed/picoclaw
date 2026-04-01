import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  type ChangeEvent,
  type DragEvent,
  startTransition,
  useDeferredValue,
  useMemo,
  useRef,
  useState,
} from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type SkillSupportItem,
  deleteSkill,
  getSkill,
  getSkills,
  importSkill,
} from "@/api/skills"

import {
  compareSkills,
  getOriginLabel,
  getSkillOriginKind,
  sortOrigins,
} from "./origin-utils"
import type {
  SkillDetailView,
  SkillGroupSection,
  SkillLayoutMode,
  SkillSortOption,
  SkillStatItem,
} from "./types"

const MAX_IMPORT_FILE_SIZE = 1 << 20

export function useSkillsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const importInputRef = useRef<HTMLInputElement | null>(null)
  const dragDepthRef = useRef(0)

  const [searchQuery, setSearchQuery] = useState("")
  const deferredSearchQuery = useDeferredValue(searchQuery)
  const [sourceFilter, setSourceFilter] = useState("all")
  const [sortOrder, setSortOrder] = useState<SkillSortOption>("name-asc")
  const [layoutMode, setLayoutMode] = useState<SkillLayoutMode>("grouped")
  const [detailView, setDetailView] = useState<SkillDetailView>("preview")
  const [isDragActive, setIsDragActive] = useState(false)
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false)
  const [selectedSkill, setSelectedSkill] = useState<SkillSupportItem | null>(
    null,
  )
  const [skillPendingDelete, setSkillPendingDelete] =
    useState<SkillSupportItem | null>(null)

  const skillsQuery = useQuery({
    queryKey: ["skills"],
    queryFn: getSkills,
  })

  const skillDetailQuery = useQuery({
    queryKey: ["skills", selectedSkill?.name],
    queryFn: () => getSkill(selectedSkill!.name),
    enabled: selectedSkill !== null,
  })

  const importMutation = useMutation({
    mutationFn: async (file: File) => importSkill(file),
    onSuccess: (importedSkill) => {
      toast.success(t("pages.agent.skills.import_success"))
      startTransition(() => {
        setIsImportDialogOpen(false)
        setDetailView("preview")
        if (importedSkill.name) {
          setSelectedSkill({
            name: importedSkill.name,
            path: importedSkill.path ?? "",
            source: importedSkill.source ?? "workspace",
            description: importedSkill.description ?? "",
            origin_kind: importedSkill.origin_kind ?? "manual",
            registry_name: importedSkill.registry_name,
            registry_url: importedSkill.registry_url,
            installed_version: importedSkill.installed_version,
            installed_at: importedSkill.installed_at,
          })
        }
      })
      void queryClient.invalidateQueries({ queryKey: ["skills"] })
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.skills.import_error"),
      )
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (name: string) => deleteSkill(name),
    onSuccess: (_, deletedName) => {
      toast.success(t("pages.agent.skills.delete_success"))
      setSkillPendingDelete(null)
      if (
        selectedSkill?.name === deletedName &&
        selectedSkill.source === "workspace"
      ) {
        setSelectedSkill(null)
      }
      void queryClient.invalidateQueries({ queryKey: ["skills"] })
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.skills.delete_error"),
      )
    },
  })

  const allSkills = useMemo(
    () => skillsQuery.data?.skills ?? [],
    [skillsQuery.data?.skills],
  )
  const normalizedSearchQuery = deferredSearchQuery.trim().toLowerCase()

  const availableOrigins = useMemo(
    () =>
      sortOrigins([
        ...new Set(allSkills.map((skill) => getSkillOriginKind(skill))),
      ]),
    [allSkills],
  )

  const filteredSkills = useMemo(() => {
    return allSkills.filter((skill) => {
      const matchesSource =
        sourceFilter === "all"
          ? true
          : getSkillOriginKind(skill) === sourceFilter
      if (!matchesSource) return false
      if (normalizedSearchQuery === "") return true

      const searchTarget =
        `${skill.name} ${skill.description} ${skill.registry_name ?? ""}`.toLowerCase()
      return searchTarget.includes(normalizedSearchQuery)
    })
  }, [allSkills, normalizedSearchQuery, sourceFilter])

  const sortedSkills = useMemo(
    () => [...filteredSkills].sort((left, right) => compareSkills(left, right, sortOrder)),
    [filteredSkills, sortOrder],
  )

  const groupedSkills = useMemo<SkillGroupSection[]>(
    () =>
      availableOrigins
        .map((origin) => ({
          origin,
          skills: sortedSkills.filter(
            (skill) => getSkillOriginKind(skill) === origin,
          ),
        }))
        .filter((section) => section.skills.length > 0),
    [availableOrigins, sortedSkills],
  )

  const stats = useMemo<SkillStatItem[]>(
    () => [
      {
        key: "all",
        origin: "all",
        label: t("pages.agent.skills.summary.total"),
        count: allSkills.length,
      },
      ...availableOrigins.map((origin) => ({
        key: origin,
        origin,
        label: getOriginLabel(origin, t),
        count: allSkills.filter((skill) => getSkillOriginKind(skill) === origin)
          .length,
      })),
    ],
    [allSkills, availableOrigins, t],
  )

  const hasActiveFilters =
    normalizedSearchQuery !== "" || sourceFilter !== "all"

  const handleImportClick = () => {
    importInputRef.current?.click()
  }

  const handleViewSkill = (skill: SkillSupportItem) => {
    setDetailView("preview")
    setSelectedSkill(skill)
  }

  const handleRequestDelete = (skill: SkillSupportItem) => {
    setSkillPendingDelete(skill)
  }

  const handleConfirmDelete = () => {
    if (skillPendingDelete) {
      deleteMutation.mutate(skillPendingDelete.name)
    }
  }

  const handleDetailSheetOpenChange = (open: boolean) => {
    if (!open) {
      setSelectedSkill(null)
    }
  }

  const handleImportDialogOpenChange = (open: boolean) => {
    if (!importMutation.isPending) {
      setIsImportDialogOpen(open)
    }
  }

  const handleDeleteDialogOpenChange = (open: boolean) => {
    if (!open) {
      setSkillPendingDelete(null)
    }
  }

  const validateImportFile = (file: File) => {
    const fileName = file.name.toLowerCase()
    const isMarkdownFile =
      fileName.endsWith(".md") ||
      file.type === "text/markdown" ||
      file.type === "text/plain" ||
      file.type === ""
    const isZipFile =
      fileName.endsWith(".zip") ||
      file.type === "application/zip" ||
      file.type === "application/x-zip-compressed"

    if (!isMarkdownFile && !isZipFile) {
      return t("pages.agent.skills.import_invalid_type")
    }

    if (file.size > MAX_IMPORT_FILE_SIZE) {
      return t("pages.agent.skills.import_invalid_size")
    }

    return null
  }

  const handleImportFile = (file: File) => {
    const validationMessage = validateImportFile(file)
    if (validationMessage) {
      toast.error(validationMessage)
      return
    }
    importMutation.mutate(file)
  }

  const handleImportFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return
    handleImportFile(file)
    event.target.value = ""
  }

  const resetDragState = () => {
    dragDepthRef.current = 0
    setIsDragActive(false)
  }

  const handleDropZoneDragEnter = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    dragDepthRef.current += 1
    setIsDragActive(true)
  }

  const handleDropZoneDragLeave = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    dragDepthRef.current = Math.max(0, dragDepthRef.current - 1)
    if (dragDepthRef.current === 0) {
      setIsDragActive(false)
    }
  }

  const handleDropZoneDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    const file = event.dataTransfer.files?.[0]
    resetDragState()
    if (!file) return
    handleImportFile(file)
  }

  return {
    searchQuery,
    sourceFilter,
    sortOrder,
    layoutMode,
    detailView,
    isDragActive,
    isImportDialogOpen,
    selectedSkill,
    skillPendingDelete,
    availableOrigins,
    groupedSkills,
    stats,
    sortedSkills,
    hasActiveFilters,
    importInputRef,
    selectedSkillDetail: skillDetailQuery.data,
    skillsError: skillsQuery.error,
    skillDetailError: skillDetailQuery.error,
    isLoading: skillsQuery.isLoading,
    isSkillDetailLoading: skillDetailQuery.isLoading,
    isImportPending: importMutation.isPending,
    isDeletePending: deleteMutation.isPending,
    setSearchQuery,
    setSourceFilter,
    setSortOrder,
    setLayoutMode,
    setDetailView,
    openImportDialog: () => setIsImportDialogOpen(true),
    handleViewSkill,
    handleRequestDelete,
    handleConfirmDelete,
    handleImportClick,
    handleImportFileChange,
    handleDropZoneDragEnter,
    handleDropZoneDragLeave,
    handleDropZoneDrop,
    handleDetailSheetOpenChange,
    handleImportDialogOpenChange,
    handleDeleteDialogOpenChange,
  }
}
