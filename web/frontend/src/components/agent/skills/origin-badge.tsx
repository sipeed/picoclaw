import {
  IconFileCode,
  IconFolder,
  IconSparkles,
  IconWorld,
} from "@tabler/icons-react"

import { cn } from "@/lib/utils"

import { getOriginBadgeClasses } from "./origin-utils"

export function OriginBadge({
  origin,
  label,
}: {
  origin: string
  label: string
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-1 text-[11px] font-semibold",
        getOriginBadgeClasses(origin),
      )}
    >
      <OriginIcon origin={origin} />
      {label}
    </span>
  )
}

export function OriginIcon({ origin }: { origin: string }) {
  if (origin === "builtin") {
    return <IconSparkles className="size-3.5" />
  }
  if (origin === "third_party") {
    return <IconWorld className="size-3.5" />
  }
  if (origin === "manual") {
    return <IconFolder className="size-3.5" />
  }
  if (origin === "all") {
    return <IconFileCode className="size-4" />
  }
  return <IconFileCode className="size-3.5" />
}
