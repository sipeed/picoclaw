import { IconLoader2 } from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { type ModelInfo, deleteModel } from "@/api/models"
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

interface DeleteModelDialogProps {
  model: ModelInfo | null
  onClose: () => void
  onDeleted: () => void
}

export function DeleteModelDialog({
  model,
  onClose,
  onDeleted,
}: DeleteModelDialogProps) {
  const { t } = useTranslation()
  const [deleting, setDeleting] = useState(false)

  const handleConfirm = async () => {
    if (!model) return
    if (model.is_default) {
      onClose()
      return
    }
    setDeleting(true)
    try {
      await deleteModel(model.index)
      onDeleted()
    } catch {
      // ignore, user can retry from list
    } finally {
      setDeleting(false)
      onClose()
    }
  }

  return (
    <AlertDialog open={model !== null} onOpenChange={(v) => !v && onClose()}>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("models.delete.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("models.delete.description", { name: model?.model_name })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose} disabled={deleting}>
            {t("common.cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={handleConfirm}
            disabled={deleting}
          >
            {deleting && <IconLoader2 className="size-4 animate-spin" />}
            {t("models.delete.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
