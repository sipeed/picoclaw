import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

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
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"

export const Route = createFileRoute("/config")({
  component: ConfigPage,
})

function ConfigPage() {
  const { t } = useTranslation()
  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.config", "Config")} />
      <div className="flex-1 overflow-auto p-3 lg:p-6">
        <div className="mx-auto max-w-4xl">
          <RawJsonPanel />
        </div>
      </div>
    </div>
  )
}

function RawJsonPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const { data: config, isLoading } = useQuery({
    queryKey: ["config"],
    queryFn: async () => {
      const res = await fetch("/api/config")
      if (!res.ok) {
        throw new Error("Failed to fetch config")
      }
      return res.json()
    },
  })

  const mutation = useMutation({
    mutationFn: async (newConfig: string) => {
      const res = await fetch("/api/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: newConfig,
      })
      if (!res.ok) {
        throw new Error("Failed to save config")
      }
    },
    onSuccess: (_, submittedConfig) => {
      toast.success(
        t("pages.config.save_success", "Configuration saved successfully."),
      )
      // Update last saved config and reset dirty state
      try {
        const savedConfig = JSON.parse(submittedConfig)
        setLastSavedConfig(savedConfig)
        setIsDirty(false)
        // Important: Invalidate the query to refresh the cached data
        queryClient.invalidateQueries({ queryKey: ["config"] })
      } catch {
        // If JSON parsing fails, invalidate to get fresh data
        queryClient.invalidateQueries({ queryKey: ["config"] })
      }
    },
    onError: () => {
      toast.error(t("pages.config.save_error", "Failed to save configuration."))
    },
  })

  const [editorValue, setEditorValue] = useState("")
  const [isDirty, setIsDirty] = useState(false)

  // Store the last saved config to detect changes
  const [lastSavedConfig, setLastSavedConfig] = useState<Record<
    string,
    unknown
  > | null>(null)

  const effectiveEditorValue =
    editorValue || (config ? JSON.stringify(config, null, 2) : "")

  const handleSave = () => {
    try {
      // Validate JSON before saving
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
      toast.success(
        t("pages.config.format_success", "JSON formatted successfully."),
      )
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
    // Reset editor content to the last saved configuration
    if (lastSavedConfig) {
      setEditorValue(JSON.stringify(lastSavedConfig, null, 2))
    } else if (config) {
      // Fallback to current config if no last saved config
      setEditorValue(JSON.stringify(config, null, 2))
    }
    setIsDirty(false)
    toast.info(
      t(
        "pages.config.reset_success",
        "Changes have been reset to the last saved state.",
      ),
    )
    setShowResetDialog(false)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {t("pages.config.raw_json_title", "Raw JSON Configuration")}
        </CardTitle>
        <CardDescription>
          {t(
            "pages.config.raw_json_desc",
            "Advanced users can directly edit the raw JSON configuration below.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <p>{t("labels.loading", "Loading...")}</p>
          </div>
        ) : (
          <div className="space-y-3">
            {isDirty && (
              <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-2 text-sm text-yellow-700">
                {t("pages.config.unsaved_changes", "You have unsaved changes.")}
              </div>
            )}
            <div className="bg-muted/30 relative rounded-lg border">
              <ScrollArea className="h-[calc(100vh-20rem)] min-h-[200px]">
                <Textarea
                  value={effectiveEditorValue}
                  onChange={(e) => {
                    setEditorValue(e.target.value)
                    setIsDirty(true)
                  }}
                  className="min-h-[200px] resize-none border-0 bg-transparent px-4 py-3 font-mono text-sm shadow-none focus-visible:ring-0"
                  placeholder={t(
                    "pages.config.json_placeholder",
                    "Enter valid JSON configuration...",
                  )}
                />
              </ScrollArea>
            </div>
            <div className="flex justify-end space-x-2">
              <Button
                variant="outline"
                onClick={handleFormat}
                disabled={mutation.isPending}
              >
                {t("pages.config.format", "Format")}
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
                    {t("common.reset", "Reset")}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t("pages.config.reset_confirm_title", "Reset Changes")}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t(
                        "pages.config.reset_confirm_desc",
                        "Are you sure you want to reset your unsaved changes back to the last saved state?",
                      )}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>
                      {t("common.cancel", "Cancel")}
                    </AlertDialogCancel>
                    <AlertDialogAction onClick={confirmReset}>
                      {t("common.confirm", "Confirm")}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
              <Button onClick={handleSave} disabled={mutation.isPending}>
                {mutation.isPending
                  ? t("common.saving", "Saving...")
                  : t("common.save", "Save")}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
