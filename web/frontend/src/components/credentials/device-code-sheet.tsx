import { IconRefresh } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthFlowState } from "@/api/oauth"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"

interface DeviceCodeSheetProps {
  open: boolean
  flow: OAuthFlowState | null
  flowHint: string
  onOpenChange: (open: boolean) => void
}

export function DeviceCodeSheet({
  open,
  flow,
  flowHint,
  onOpenChange,
}: DeviceCodeSheetProps) {
  const { t } = useTranslation()

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="data-[side=right]:!w-full data-[side=right]:sm:!w-[480px] data-[side=right]:sm:!max-w-[480px]"
      >
        <SheetHeader className="border-b-muted border-b px-6 py-5">
          <SheetTitle>
            {t("credentials.device.title", "OpenAI Device Login")}
          </SheetTitle>
          <SheetDescription>
            {t(
              "credentials.device.description",
              "Open the verification page and enter the code below. This page will refresh automatically.",
            )}
          </SheetDescription>
        </SheetHeader>

        <div className="space-y-4 px-6 py-5">
          <div>
            <p className="text-muted-foreground text-xs uppercase">
              {t("credentials.device.code", "User Code")}
            </p>
            <p className="mt-1 rounded-md border px-3 py-2 font-mono text-lg font-semibold tracking-wide">
              {flow?.user_code || "-"}
            </p>
          </div>

          <div>
            <p className="text-muted-foreground text-xs uppercase">
              {t("credentials.device.url", "Verification URL")}
            </p>
            <a
              href={flow?.verify_url || "#"}
              target="_blank"
              rel="noreferrer"
              className="text-primary mt-1 block text-sm break-all underline"
            >
              {flow?.verify_url || "-"}
            </a>
          </div>

          <div className="text-muted-foreground flex items-center gap-2 text-sm">
            <IconRefresh className="size-4" />
            {t("credentials.device.polling", "Polling login status...")}
          </div>

          {flow && (
            <div className="bg-muted rounded-md border px-3 py-2 text-sm">
              {flowHint}
            </div>
          )}
        </div>

        <SheetFooter className="border-t-muted border-t px-6 py-4">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("common.cancel", "Close")}
          </Button>
          <Button asChild disabled={!flow?.verify_url}>
            <a href={flow?.verify_url || "#"} target="_blank" rel="noreferrer">
              {t("credentials.device.open", "Open Verification Page")}
            </a>
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
