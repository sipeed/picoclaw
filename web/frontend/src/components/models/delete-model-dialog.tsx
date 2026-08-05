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
  isDefaultModel: boolean
  onClose: () => void
  onDeleted: () => void
  onMutationStarted: () => void
  onMutationSettled: () => void
}

export function DeleteModelDialog({
  model,
  isDefaultModel,
  onClose,
  onDeleted,
  onMutationStarted,
  onMutationSettled,
}: DeleteModelDialogProps) {
  const { t } = useTranslation()
  const [deleting, setDeleting] = useState(false)

  const handleConfirm = async () => {
    if (!model) return
    if (isDefaultModel) {
      onClose()
      return
    }
    setDeleting(true)
    onMutationStarted()
    try {
      await deleteModel(model.index)
      onDeleted()
    } catch {
      // ignore, user can retry from list
    } finally {
      onMutationSettled()
      setDeleting(false)
      onClose()
    }
  }

  return (
    <AlertDialog
      open={model !== null}
      onOpenChange={(v) => !v && !deleting && onClose()}
    >
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
            disabled={deleting || isDefaultModel}
          >
            {deleting && <IconLoader2 className="size-4 animate-spin" />}
            {t("models.delete.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
