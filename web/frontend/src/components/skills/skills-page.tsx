import {
  IconFileInfo,
  IconLoader2,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ChangeEvent, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"
import { toast } from "sonner"

import {
  type SkillSupportItem,
  deleteSkill,
  getSkill,
  getSkills,
  importSkill,
} from "@/api/skills"
import { PageHeader } from "@/components/page-header"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"

export function SkillsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const importInputRef = useRef<HTMLInputElement | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<SkillSupportItem | null>(
    null,
  )
  const [skillPendingDelete, setSkillPendingDelete] =
    useState<SkillSupportItem | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ["skills"],
    queryFn: getSkills,
  })
  const {
    data: selectedSkillDetail,
    isLoading: isSkillDetailLoading,
    error: skillDetailError,
  } = useQuery({
    queryKey: ["skills", selectedSkill?.name],
    queryFn: () => getSkill(selectedSkill!.name),
    enabled: selectedSkill !== null,
  })

  const importMutation = useMutation({
    mutationFn: async (file: File) => importSkill(file),
    onSuccess: () => {
      toast.success(t("pages.agent.skills.import_success"))
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

  const handleImportClick = () => {
    importInputRef.current?.click()
  }

  const handleImportFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return
    importMutation.mutate(file)
    event.target.value = ""
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={t("navigation.skills")}
        children={
          <>
            <input
              ref={importInputRef}
              type="file"
              accept=".md,text/markdown,text/plain"
              className="hidden"
              onChange={handleImportFileChange}
            />
            <Button
              variant="outline"
              onClick={handleImportClick}
              disabled={importMutation.isPending}
            >
              {importMutation.isPending ? (
                <IconLoader2 className="size-4 animate-spin" />
              ) : (
                <IconPlus className="size-4" />
              )}
              {t("pages.agent.skills.import")}
            </Button>
          </>
        }
      />

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.agent.load_error")}
            </div>
          ) : (
            <section className="space-y-5">
              <p className="text-muted-foreground text-sm">
                {t("pages.agent.skills.description")}
              </p>

              {data?.skills.length ? (
                <div className="grid gap-4 lg:grid-cols-2">
                  {data.skills.map((skill) => (
                    <Card
                      key={`${skill.source}:${skill.name}`}
                      className="border-border/60 gap-4"
                      size="sm"
                    >
                      <CardHeader>
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <CardTitle className="font-semibold">
                              {skill.name}
                            </CardTitle>
                            <CardDescription className="mt-3">
                              {skill.description ||
                                t("pages.agent.skills.no_description")}
                            </CardDescription>
                          </div>
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              className="text-muted-foreground hover:text-foreground"
                              onClick={() => setSelectedSkill(skill)}
                              title={t("pages.agent.skills.view")}
                            >
                              <IconFileInfo className="size-4" />
                            </Button>
                            {skill.source === "workspace" ? (
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                className="text-muted-foreground hover:text-destructive"
                                onClick={() => setSkillPendingDelete(skill)}
                                title={t("pages.agent.skills.delete")}
                              >
                                <IconTrash className="size-4" />
                              </Button>
                            ) : null}
                          </div>
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-2">
                        <div className="text-muted-foreground text-[11px] tracking-[0.18em] uppercase">
                          {t("pages.agent.skills.path")}
                        </div>
                        <div className="bg-muted text-foreground overflow-x-auto rounded-lg px-3 py-2 font-mono text-xs leading-relaxed">
                          {skill.path}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <Card className="border-dashed">
                  <CardContent className="text-muted-foreground py-10 text-center text-sm">
                    {t("pages.agent.skills.empty")}
                  </CardContent>
                </Card>
              )}
            </section>
          )}
        </div>
      </div>

      <Sheet
        open={selectedSkill !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedSkill(null)
        }}
      >
        <SheetContent
          side="right"
          className="w-full gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[560px] data-[side=right]:sm:!max-w-[560px]"
        >
          <SheetHeader className="border-b px-6 py-5">
            <SheetTitle>
              {selectedSkill?.name || t("pages.agent.skills.viewer_title")}
            </SheetTitle>
            <SheetDescription>
              {selectedSkill?.description ||
                t("pages.agent.skills.viewer_description")}
            </SheetDescription>
          </SheetHeader>

          <div className="flex-1 overflow-auto px-6 py-5">
            {isSkillDetailLoading ? (
              <div className="text-muted-foreground text-sm">
                {t("pages.agent.skills.loading_detail")}
              </div>
            ) : skillDetailError ? (
              <div className="text-destructive text-sm">
                {t("pages.agent.skills.load_detail_error")}
              </div>
            ) : selectedSkillDetail ? (
              <div className="space-y-5">
                <div className="prose prose-sm dark:prose-invert prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-950 prose-pre:p-3 max-w-none">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={[rehypeRaw, rehypeSanitize]}
                  >
                    {selectedSkillDetail.content}
                  </ReactMarkdown>
                </div>
              </div>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

      <AlertDialog
        open={skillPendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setSkillPendingDelete(null)
        }}
      >
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("pages.agent.skills.delete_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("pages.agent.skills.delete_description", {
                name: skillPendingDelete?.name,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              {t("common.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleteMutation.isPending || !skillPendingDelete}
              onClick={() => {
                if (skillPendingDelete)
                  deleteMutation.mutate(skillPendingDelete.name)
              }}
            >
              {deleteMutation.isPending ? (
                <IconLoader2 className="size-4 animate-spin" />
              ) : (
                <IconTrash className="size-4" />
              )}
              {t("pages.agent.skills.delete_confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
