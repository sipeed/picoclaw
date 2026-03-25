import { ReactNode } from "react"

import { cn } from "@/lib/utils"

interface ControlItemProps {
  label: string
  hint?: string
  control: ReactNode
  className?: string
}

export function ControlItem({
  label,
  hint,
  control,
  className,
}: ControlItemProps) {
  return (
    <div
      className={cn(
        "flex flex-row items-center justify-between rounded-lg border p-4 shadow-sm",
        className,
      )}
    >
      <div className="space-y-0.5">
        <label className="text-sm leading-none font-medium peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
          {label}
        </label>
        {hint && <p className="text-muted-foreground text-sm">{hint}</p>}
      </div>
      <div>{control}</div>
    </div>
  )
}
