import { useTranslation } from "react-i18next"

import type { ToolSupportItem } from "@/api/tools"
import { cn } from "@/lib/utils"

interface ToolStatusBadgeProps {
  status: ToolSupportItem["status"]
}

export function ToolStatusBadge({ status }: ToolStatusBadgeProps) {
  const { t } = useTranslation()

  return (
    <span
      className={cn(
        "shrink-0 rounded-full px-2.5 py-0.5 text-[11px] font-medium tracking-wide sm:text-[11px]",
        status === "enabled" &&
          "bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400",
        status === "blocked" &&
          "bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-400",
        status === "disabled" &&
          "bg-muted text-muted-foreground/80 dark:bg-muted-foreground/20 dark:text-muted-foreground",
      )}
    >
      {t(`pages.agent.tools.status.${status}`, status)}
    </span>
  )
}
