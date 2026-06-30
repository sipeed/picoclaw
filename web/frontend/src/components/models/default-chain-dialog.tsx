import {
  IconArrowBarDown,
  IconArrowBarToUp,
  IconArrowDown,
  IconArrowUp,
  IconGripVertical,
  IconTrash,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface DefaultChainDialogProps {
  open: boolean
  onClose: () => void
  models: ModelInfo[]
  defaultModelName: string
  fallbackChain: string[]
  onMoveUp: (modelName: string) => void
  onMoveDown: (modelName: string) => void
  onMoveTop: (modelName: string) => void
  onMoveBottom: (modelName: string) => void
  onRemove: (modelName: string) => void
}

export function DefaultChainDialog({
  open,
  onClose,
  models,
  defaultModelName,
  fallbackChain,
  onMoveUp,
  onMoveDown,
  onMoveTop,
  onMoveBottom,
  onRemove,
}: DefaultChainDialogProps) {
  const { t } = useTranslation()
  const modelsByName = new Map(models.map((model) => [model.model_name, model]))
  const defaultModel = defaultModelName
    ? modelsByName.get(defaultModelName)
    : undefined

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("models.defaultChain.dialogTitle")}</DialogTitle>
          <DialogDescription>
            {t("models.defaultChain.dialogDescription")}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <section className="space-y-2">
            <p className="text-sm font-medium">
              {t("models.defaultChain.defaultModel")}
            </p>
            <div className="bg-muted/40 rounded-lg border px-4 py-3">
              {defaultModel ? (
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">
                      {defaultModel.model_name}
                    </p>
                    <p className="text-muted-foreground truncate font-mono text-xs">
                      {defaultModel.model}
                    </p>
                  </div>
                  <span className="bg-primary/10 text-primary shrink-0 rounded px-2 py-1 text-[11px] font-medium">
                    {t("models.badge.default")}
                  </span>
                </div>
              ) : (
                <p className="text-muted-foreground text-sm">
                  {t("models.defaultChain.noDefault")}
                </p>
              )}
            </div>
          </section>

          <section className="space-y-2">
            <p className="text-sm font-medium">
              {t("models.defaultChain.fallbackChain")}
            </p>
            {fallbackChain.length === 0 ? (
              <div className="text-muted-foreground bg-muted/20 rounded-lg border border-dashed px-4 py-6 text-sm">
                {t("models.defaultChain.empty")}
              </div>
            ) : (
              <div className="space-y-2">
                {fallbackChain.map((modelName, index) => {
                  const model = modelsByName.get(modelName)
                  const isFirst = index === 0
                  const isLast = index === fallbackChain.length - 1

                  return (
                    <div
                      key={modelName}
                      className="bg-background flex items-center gap-3 rounded-lg border px-3 py-3"
                    >
                      <span className="text-muted-foreground shrink-0">
                        <IconGripVertical className="size-4" />
                      </span>

                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">
                          {model?.model_name ?? modelName}
                        </p>
                        <p className="text-muted-foreground truncate font-mono text-xs">
                          {model?.model ?? modelName}
                        </p>
                      </div>

                      <div className="flex shrink-0 items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => onMoveTop(modelName)}
                          disabled={isFirst}
                          aria-label={t("models.defaultChain.moveTop")}
                          title={t("models.defaultChain.moveTop")}
                        >
                          <IconArrowBarToUp className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => onMoveUp(modelName)}
                          disabled={isFirst}
                          aria-label={t("models.defaultChain.moveUp")}
                          title={t("models.defaultChain.moveUp")}
                        >
                          <IconArrowUp className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => onMoveDown(modelName)}
                          disabled={isLast}
                          aria-label={t("models.defaultChain.moveDown")}
                          title={t("models.defaultChain.moveDown")}
                        >
                          <IconArrowDown className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => onMoveBottom(modelName)}
                          disabled={isLast}
                          aria-label={t("models.defaultChain.moveBottom")}
                          title={t("models.defaultChain.moveBottom")}
                        >
                          <IconArrowBarDown className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => onRemove(modelName)}
                          aria-label={t("models.defaultChain.remove")}
                          title={t("models.defaultChain.remove")}
                          className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                        >
                          <IconTrash className="size-3.5" />
                        </Button>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </section>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("common.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
