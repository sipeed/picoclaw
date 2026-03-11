import { IconLoader2 } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

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

interface LogoutConfirmDialogProps {
  open: boolean
  providerLabel: string
  isSubmitting: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void | Promise<void>
}

export function LogoutConfirmDialog({
  open,
  providerLabel,
  isSubmitting,
  onOpenChange,
  onConfirm,
}: LogoutConfirmDialogProps) {
  const { t } = useTranslation()

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t("credentials.logoutDialog.title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "credentials.logoutDialog.description",
              "This will remove your saved credential for {{provider}}.",
              { provider: providerLabel },
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} variant="destructive">
            {isSubmitting && <IconLoader2 className="size-4 animate-spin" />}
            {t("credentials.actions.logout")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
