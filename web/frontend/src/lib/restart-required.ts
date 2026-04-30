import type { TFunction } from "i18next"
import { toast } from "sonner"

export function showRestartRequiredToast(t: TFunction, name: string) {
  toast.warning(t("common.restartRequiredTitle"), {
    description: t("common.restartRequiredDesc", { name }),
  })
}

export function showSaveSuccessOrRestartToast(
  t: TFunction,
  savedMessage: string,
  name: string,
  restartRequired: boolean,
) {
  if (restartRequired) {
    showRestartRequiredToast(t, name)
    return
  }
  toast.success(savedMessage)
}
