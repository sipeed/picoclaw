import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"

import { OriginIcon } from "./origin-badge"
import { getOriginAccentClasses } from "./origin-utils"
import type { SkillStatItem } from "./types"

export function Stats({ stats }: { stats: SkillStatItem[] }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {stats.map((stat) => (
        <Card
          key={stat.key}
          size="sm"
          className="border-border/40 bg-card/40 hover:bg-card gap-3 shadow-sm transition-all hover:shadow-md"
        >
          <CardContent className="flex items-center justify-between pt-4">
            <div className="space-y-1">
              <div className="text-muted-foreground text-[11px] font-semibold tracking-wider uppercase">
                {stat.label}
              </div>
              <div className="text-2xl font-bold tracking-tight">
                {stat.count}
              </div>
            </div>
            <div
              className={cn(
                "rounded-xl p-2.5 shadow-sm ring-1 ring-white/10 ring-inset",
                getOriginAccentClasses(stat.origin),
              )}
            >
              <OriginIcon origin={stat.origin} />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
