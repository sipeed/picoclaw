import { IconAdjustments } from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { launcherFetch } from "@/api/http"
import { ConfigChangeNotice } from "@/components/config-change-notice"
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
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { showSaveSuccessOrRestartToast } from "@/lib/restart-required"
import { refreshGatewayState } from "@/store/gateway"

export function RawConfigPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const { data: config, isLoading } = useQuery({
    queryKey: ["config"],
    queryFn: async () => {
      const res = await launcherFetch("/api/config")
      if (!res.ok) {
        throw new Error("Failed to fetch config")
      }
      return res.json()
    },
  })

  const mutation = useMutation({
    mutationFn: async (newConfig: string) => {
      const res = await launcherFetch("/api/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: newConfig,
      })
      if (!res.ok) {
        throw new Error("Failed to save config")
      }
    },
    onSuccess: (_, submittedConfig) => {
      try {
        const savedConfig = JSON.parse(submittedConfig)
        setLastSavedConfig(savedConfig)
        setIsDirty(false)
        queryClient.invalidateQueries({ queryKey: ["config"] })
      } catch {
        queryClient.invalidateQueries({ queryKey: ["config"] })
      }
      void refreshGatewayState({ force: true }).then((gateway) => {
        showSaveSuccessOrRestartToast(
          t,
          t("pages.config.save_success"),
          t("navigation.config"),
          gateway?.restartRequired === true,
        )
      })
    },
    onError: () => {
      toast.error(t("pages.config.save_error"))
    },
  })

  const [editorValue, setEditorValue] = useState("")
  const [isDirty, setIsDirty] = useState(false)
  const [lastSavedConfig, setLastSavedConfig] = useState<Record<
    string,
    unknown
  > | null>(null)

  const effectiveEditorValue =
    editorValue || (config ? JSON.stringify(config, null, 2) : "")

  const handleSave = () => {
    try {
      JSON.parse(effectiveEditorValue)
      mutation.mutate(effectiveEditorValue)
    } catch (error) {
      toast.error(
        t(
          "pages.config.invalid_json",
          error instanceof Error ? error.message : "Invalid JSON format.",
        ),
      )
    }
  }

  const handleFormat = () => {
    try {
      const formatted = JSON.stringify(
        JSON.parse(effectiveEditorValue),
        null,
        2,
      )
      setEditorValue(formatted)
      toast.success(t("pages.config.format_success"))
    } catch (error) {
      toast.error(
        t(
          "pages.config.format_error",
          error instanceof Error ? error.message : "Invalid JSON format.",
        ),
      )
    }
  }

  const [showResetDialog, setShowResetDialog] = useState(false)

  const confirmReset = () => {
    if (lastSavedConfig) {
      setEditorValue(JSON.stringify(lastSavedConfig, null, 2))
    } else if (config) {
      setEditorValue(JSON.stringify(config, null, 2))
    }
    setIsDirty(false)
    toast.info(t("pages.config.reset_success"))
    setShowResetDialog(false)
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("pages.config.raw_json_title")}>
        <Button variant="outline" asChild>
          <Link to="/config">
            <IconAdjustments className="size-4" />
            {t("pages.config.back_to_visual")}
          </Link>
        </Button>
      </PageHeader>

      <div className="flex min-h-0 flex-1 flex-col p-1 lg:p-3 lg:p-6">
        <div className="mx-auto flex h-full min-h-0 w-full max-w-[1000px] flex-col">
          {isLoading ? (
            <div className="flex flex-1 items-center justify-center">
              <p>{t("labels.loading")}</p>
            </div>
          ) : (
            <div className="flex min-h-0 flex-1 flex-col gap-3">
              {isDirty && (
                <ConfigChangeNotice
                  kind="save"
                  title={t("common.saveChangesTitle")}
                  description={t("pages.config.unsaved_changes")}
                  className="shrink-0"
                />
              )}
              <div className="relative min-h-0 flex-1 overflow-hidden rounded-lg border shadow-sm">
                <Textarea
                  value={effectiveEditorValue}
                  onChange={(e) => {
                    setEditorValue(e.target.value)
                    setIsDirty(true)
                  }}
                  wrap="off"
                  className="h-full min-h-0 resize-none overflow-auto border-0 bg-transparent px-4 py-3 font-mono text-sm [overflow-wrap:normal] whitespace-pre shadow-none focus-visible:ring-0"
                  placeholder={t("pages.config.json_placeholder")}
                />
              </div>
              <div className="flex shrink-0 justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={handleFormat}
                  disabled={mutation.isPending}
                >
                  {t("pages.config.format")}
                </Button>
                <AlertDialog
                  open={showResetDialog}
                  onOpenChange={setShowResetDialog}
                >
                  <AlertDialogTrigger asChild>
                    <Button
                      variant="outline"
                      disabled={!isDirty}
                      onClick={() => setShowResetDialog(true)}
                    >
                      {t("common.reset")}
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>
                        {t("pages.config.reset_confirm_title")}
                      </AlertDialogTitle>
                      <AlertDialogDescription>
                        {t("pages.config.reset_confirm_desc")}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>
                        {t("common.cancel")}
                      </AlertDialogCancel>
                      <AlertDialogAction onClick={confirmReset}>
                        {t("common.confirm")}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
                <Button onClick={handleSave} disabled={mutation.isPending}>
                  {mutation.isPending ? t("common.saving") : t("common.save")}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
