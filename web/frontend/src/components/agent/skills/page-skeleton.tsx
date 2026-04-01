import { Skeleton } from "@/components/ui/skeleton"

export function PageSkeleton() {
  return (
    <div className="mt-4 space-y-8">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[1, 2, 3, 4].map((index) => (
          <Skeleton
            key={index}
            className="border-border/40 h-24 w-full rounded-xl border"
          />
        ))}
      </div>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-48" />
        </div>
        <Skeleton className="h-14 w-full rounded-xl" />
        <div className="grid gap-4 pt-4 lg:grid-cols-2">
          {[1, 2, 3, 4].map((index) => (
            <Skeleton key={index} className="h-36 w-full rounded-xl" />
          ))}
        </div>
      </div>
    </div>
  )
}
