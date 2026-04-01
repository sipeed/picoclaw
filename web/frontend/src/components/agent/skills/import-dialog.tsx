import { IconLoader2, IconUpload, IconX } from "@tabler/icons-react"
import type { DragEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"

interface ImportDialogProps {
  open: boolean
  isImportPending: boolean
  isDragActive: boolean
  onOpenChange: (open: boolean) => void
  onImportClick: () => void
  onDragEnter: (event: DragEvent<HTMLDivElement>) => void
  onDragLeave: (event: DragEvent<HTMLDivElement>) => void
  onDrop: (event: DragEvent<HTMLDivElement>) => void
}

export function ImportDialog({
  open,
  isImportPending,
  isDragActive,
  onOpenChange,
  onImportClick,
  onDragEnter,
  onDragLeave,
  onDrop,
}: ImportDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!isImportPending) {
          onOpenChange(nextOpen)
        }
      }}
    >
      <DialogContent
        showCloseButton={false}
        className="border-border/40 bg-card/95 max-w-[420px] gap-6 p-6 text-center shadow-lg backdrop-blur-sm focus:outline-none sm:rounded-2xl"
      >
        <div className="relative space-y-1 px-8">
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground hover:text-foreground absolute top-0 right-0"
            onClick={() => onOpenChange(false)}
            disabled={isImportPending}
            aria-label={t("common.cancel")}
            title={t("common.cancel")}
          >
            <IconX className="size-4" />
          </Button>

          <DialogHeader className="space-y-1 text-center">
            <DialogTitle className="text-lg font-semibold tracking-tight">
              {t("pages.agent.skills.dropzone_title")}
            </DialogTitle>
            <DialogDescription className="text-muted-foreground text-sm">
              {t("pages.agent.skills.dropzone_description")}
            </DialogDescription>
          </DialogHeader>
        </div>

        <SkillImportPanel
          isDragActive={isDragActive}
          isImportPending={isImportPending}
          onDragEnter={onDragEnter}
          onDragLeave={onDragLeave}
          onDrop={onDrop}
          onImportClick={onImportClick}
        />
      </DialogContent>
    </Dialog>
  )
}

function SkillImportPanel({
  isDragActive,
  isImportPending,
  onDragEnter,
  onDragLeave,
  onDrop,
  onImportClick,
}: {
  isDragActive: boolean
  isImportPending: boolean
  onDragEnter: (event: DragEvent<HTMLDivElement>) => void
  onDragLeave: (event: DragEvent<HTMLDivElement>) => void
  onDrop: (event: DragEvent<HTMLDivElement>) => void
  onImportClick: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4">
      <div
        className={cn(
          "flex min-h-48 cursor-pointer flex-col items-center justify-center gap-3 rounded-xl border-2 border-dashed px-4 py-6 text-center transition-all duration-300",
          isDragActive
            ? "border-primary bg-primary/10 scale-[1.02]"
            : "border-border/60 bg-muted/30 hover:bg-muted/50 hover:border-primary/50",
          isImportPending && "pointer-events-none opacity-50",
        )}
        onClick={() => {
          if (!isImportPending) {
            onImportClick()
          }
        }}
        onDragEnter={onDragEnter}
        onDragLeave={onDragLeave}
        onDragOver={(event) => event.preventDefault()}
        onDrop={onDrop}
      >
        <div
          className={cn(
            "mb-2 rounded-full p-3 transition-colors duration-300",
            isDragActive
              ? "bg-primary text-primary-foreground shadow-sm"
              : "bg-background text-muted-foreground ring-border/50 shadow-sm ring-1",
          )}
        >
          <IconUpload className="size-6" />
        </div>
        <div className="space-y-1">
          <div className="text-foreground text-sm font-semibold tracking-tight">
            {isDragActive
              ? t("pages.agent.skills.dropzone_active")
              : t("pages.agent.skills.dropzone_label")}
          </div>
          <p className="text-muted-foreground mx-auto hidden max-w-[270px] text-xs leading-relaxed sm:block">
            {isDragActive
              ? t("pages.agent.skills.dropzone_release")
              : t("pages.agent.skills.import_constraints")}
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          className="pointer-events-none mt-2 h-8 shadow-sm"
          disabled={isImportPending}
        >
          {isImportPending ? (
            <IconLoader2 className="mr-1.5 size-3.5 animate-spin" />
          ) : null}
          {t("pages.agent.skills.import")}
        </Button>
      </div>
    </div>
  )
}
