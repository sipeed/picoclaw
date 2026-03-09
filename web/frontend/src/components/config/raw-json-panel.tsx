import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

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

export function RawJsonPanel() {
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
      toast.success(t("pages.config.save_success"))
      try {
        const savedConfig = JSON.parse(submittedConfig)
        setLastSavedConfig(savedConfig)
        setIsDirty(false)
        queryClient.invalidateQueries({ queryKey: ["config"] })
      } catch {
        queryClient.invalidateQueries({ queryKey: ["config"] })
      }
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
    <Card>
      <CardHeader>
        <CardTitle>{t("pages.config.raw_json_title")}</CardTitle>
        <CardDescription>{t("pages.config.raw_json_desc")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <p>{t("labels.loading")}</p>
          </div>
        ) : (
          <div className="space-y-3">
            {isDirty && (
              <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-2 text-sm text-yellow-700">
                {t("pages.config.unsaved_changes")}
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
                  placeholder={t("pages.config.json_placeholder")}
                />
              </ScrollArea>
            </div>
            <div className="flex justify-end space-x-2">
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
                    <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
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
      </CardContent>
    </Card>
  )
}
