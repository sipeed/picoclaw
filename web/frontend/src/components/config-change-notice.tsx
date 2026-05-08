import {
  IconAlertCircle,
  IconDeviceFloppy,
  IconRefresh,
} from "@tabler/icons-react"

import { cn } from "@/lib/utils"

interface ConfigChangeNoticeProps {
  kind: "save" | "restart"
  title: string
  description?: string
  className?: string
}

export function ConfigChangeNotice({
  kind,
  title,
  description,
  className,
}: ConfigChangeNoticeProps) {
  const Icon =
    kind === "restart"
      ? IconRefresh
      : kind === "save"
        ? IconDeviceFloppy
        : IconAlertCircle

  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-lg border px-3 py-2 text-sm",
        kind === "restart"
          ? "border-amber-200 bg-amber-50 text-amber-900"
          : "border-yellow-200 bg-yellow-50 text-yellow-900",
        className,
      )}
    >
      <Icon className="mt-0.5 size-4 shrink-0" />
      <div className="min-w-0">
        <p className="font-medium">{title}</p>
        {description && (
          <p className="mt-0.5 text-xs/5 opacity-85">{description}</p>
        )}
      </div>
    </div>
  )
}
