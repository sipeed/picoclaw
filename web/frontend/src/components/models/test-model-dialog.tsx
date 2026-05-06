import { IconLoader2, IconPlugConnected, IconX } from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { type ModelInfo, testModel } from "@/api/models"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface TestModelDialogProps {
  model: ModelInfo | null
  open: boolean
  onClose: () => void
}

interface TestResult {
  success: boolean
  latency_ms: number
  status: string
  error?: string
}

export function TestModelDialog({
  model,
  open,
  onClose,
}: TestModelDialogProps) {
  const { t } = useTranslation()
  const [testing, setTesting] = useState(false)
  const [result, setResult] = useState<TestResult | null>(null)

  const handleTest = async () => {
    if (!model) return
    setTesting(true)
    setResult(null)
    try {
      const res = await testModel(model.index)
      setResult(res)
    } catch (e) {
      setResult({
        success: false,
        latency_ms: 0,
        status: "error",
        error: e instanceof Error ? e.message : t("models.test.testFailed"),
      })
    } finally {
      setTesting(false)
    }
  }

  const handleClose = () => {
    setResult(null)
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <IconPlugConnected className="size-5" />
            {t("models.test.title")}
          </DialogTitle>
          <DialogDescription>
            {t("models.test.description")}
          </DialogDescription>
        </DialogHeader>

        {model && (
          <div className="space-y-3">
            <div className="bg-muted/50 rounded-lg p-3 text-sm">
              <div>
                <span className="text-muted-foreground">{t("models.test.modelLabel")} </span>
                <span className="font-mono">{model.model_name}</span>
              </div>
              <div>
                <span className="text-muted-foreground">{t("models.test.identifierLabel")} </span>
                <span className="font-mono">{model.model}</span>
              </div>
              {model.api_base && (
                <div>
                  <span className="text-muted-foreground">{t("models.test.endpointLabel")} </span>
                  <span className="font-mono text-xs">{model.api_base}</span>
                </div>
              )}
            </div>

            {!result && !testing && (
              <Button onClick={handleTest} className="w-full">
                <IconPlugConnected className="size-4" />
                {t("models.test.testConnection")}
              </Button>
            )}

            {testing && (
              <div className="flex items-center justify-center gap-2 py-6 text-muted-foreground">
                <IconLoader2 className="size-5 animate-spin" />
                <span>{t("models.test.testing")}</span>
              </div>
            )}

            {result && (
              <div
                className={`rounded-lg p-4 text-sm ${
                  result.success
                    ? "bg-green-500/10 text-green-700 dark:text-green-400"
                    : "bg-destructive/10 text-destructive"
                }`}
              >
                {result.success ? (
                  <div className="space-y-1">
                    <div className="font-medium">{t("models.test.success")}</div>
                    <div className="text-xs opacity-80">
                      {t("models.test.responseTime", { ms: result.latency_ms })}
                    </div>
                  </div>
                ) : (
                  <div className="space-y-1">
                    <div className="font-medium flex items-center gap-1">
                      <IconX className="size-4" />
                      {t("models.test.failed")}
                    </div>
                    <div className="text-xs opacity-80">
                      {result.error || t("models.test.status", { status: result.status })}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          <Button variant="ghost" onClick={handleClose}>
            {t("common.cancel")}
          </Button>
          {result && (
            <Button variant="outline" onClick={handleTest}>
              {t("models.test.testAgain")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
