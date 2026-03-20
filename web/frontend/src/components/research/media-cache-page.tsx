import {
  IconFileText,
  IconPhoto,
  IconTrash,
} from "@tabler/icons-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import * as React from "react"

import {
  type MediaCacheContent,
  type MediaCacheEntry,
  deleteAllMediaCache,
  deleteMediaCacheEntry,
  getMediaCacheContent,
  getMediaCacheEntries,
} from "@/api/media-cache"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export function MediaCachePage() {
  const [typeFilter, setTypeFilter] = React.useState<string>("")
  const [expandedHash, setExpandedHash] = React.useState<string | null>(null)
  const queryClient = useQueryClient()

  const { data: entries, isLoading, error } = useQuery({
    queryKey: ["media-cache", typeFilter],
    queryFn: () => getMediaCacheEntries(typeFilter || undefined),
    refetchInterval: 30000,
  })

  const handleDeleteAll = async () => {
    if (!confirm("Delete all cached media?")) return
    await deleteAllMediaCache()
    setExpandedHash(null)
    queryClient.invalidateQueries({ queryKey: ["media-cache"] })
  }

  const handleDeleteEntry = async (hash: string) => {
    await deleteMediaCacheEntry(hash)
    if (expandedHash === hash) setExpandedHash(null)
    queryClient.invalidateQueries({ queryKey: ["media-cache"] })
  }

  return (
    <div className="flex-1 overflow-auto px-6 py-3">
      <div className="w-full max-w-6xl space-y-4">
        {/* Type filter + clear all */}
        <div className="flex items-center gap-2">
          <FilterButton
            active={typeFilter === ""}
            onClick={() => setTypeFilter("")}
          >
            All
          </FilterButton>
          <FilterButton
            active={typeFilter === "image_desc"}
            onClick={() => setTypeFilter("image_desc")}
          >
            <IconPhoto className="size-3.5" />
            Images
          </FilterButton>
          <FilterButton
            active={typeFilter === "pdf_ocr"}
            onClick={() => setTypeFilter("pdf_ocr")}
          >
            <IconFileText className="size-3.5" />
            PDF
          </FilterButton>
          {entries && entries.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive ml-auto gap-1"
              onClick={handleDeleteAll}
            >
              <IconTrash className="size-3.5" />
              Clear All
            </Button>
          )}
        </div>

        {isLoading ? (
          <div className="text-muted-foreground py-6 text-sm">Loading...</div>
        ) : error ? (
          <div className="text-destructive py-6 text-sm">
            Failed to load media cache.
          </div>
        ) : !entries?.length ? (
          <Card className="border-dashed">
            <CardContent className="text-muted-foreground py-10 text-center text-sm">
              No cached media yet. Send an image or PDF to get started.
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {entries.map((entry) => (
              <MediaEntry
                key={`${entry.hash}-${entry.type}`}
                entry={entry}
                expanded={expandedHash === entry.hash}
                onToggle={() =>
                  setExpandedHash(
                    expandedHash === entry.hash ? null : entry.hash,
                  )
                }
                onDelete={() => handleDeleteEntry(entry.hash)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function FilterButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Button
      variant={active ? "default" : "outline"}
      size="sm"
      onClick={onClick}
      className="gap-1"
    >
      {children}
    </Button>
  )
}

function MediaEntry({
  entry,
  expanded,
  onToggle,
  onDelete,
}: {
  entry: MediaCacheEntry
  expanded: boolean
  onToggle: () => void
  onDelete: () => void
}) {
  const isImage = entry.type === "image_desc"
  const Icon = isImage ? IconPhoto : IconFileText
  const typeLabel = isImage ? "Image" : "PDF"
  const typeColor = isImage
    ? "text-blue-600 bg-blue-50"
    : "text-orange-600 bg-orange-50"

  const accessed = new Date(entry.accessed_at)
  const timeStr = accessed.toLocaleString()

  return (
    <Card className="gap-0">
      <CardHeader
        className="cursor-pointer select-none"
        onClick={onToggle}
      >
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Icon className="text-muted-foreground size-4 shrink-0" />
              <span className="truncate font-mono text-xs">{entry.hash}</span>
            </CardTitle>
            <CardDescription className="mt-1 line-clamp-2">
              {entry.result}
            </CardDescription>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-1">
            <span
              className={cn(
                "rounded-md px-2 py-0.5 text-[11px] font-semibold",
                typeColor,
              )}
            >
              {typeLabel}
              {entry.pages ? ` (${entry.pages}p)` : ""}
            </span>
            <span className="text-muted-foreground text-[10px]">{timeStr}</span>
          </div>
        </div>
      </CardHeader>
      {expanded && (
        <CardContent className="border-t pt-3">
          <ExpandedContent entry={entry} onDelete={onDelete} />
        </CardContent>
      )}
    </Card>
  )
}

function ExpandedContent({
  entry,
  onDelete,
}: {
  entry: MediaCacheEntry
  onDelete: () => void
}) {
  const isPDF = entry.type === "pdf_ocr"

  const { data, isLoading } = useQuery({
    queryKey: ["media-cache-content", entry.hash],
    queryFn: () => getMediaCacheContent(entry.hash),
    enabled: isPDF, // only fetch full content for PDFs
  })

  return (
    <div className="space-y-3">
      {!isPDF ? (
        <div className="space-y-2">
          <div className="text-muted-foreground text-xs font-medium">
            Description
          </div>
          <div className="bg-muted rounded-md p-3 text-sm whitespace-pre-wrap">
            {entry.result}
          </div>
        </div>
      ) : (
        <>
          <div className="space-y-1">
            <div className="text-muted-foreground text-xs font-medium">Preview</div>
            <div className="bg-muted rounded-md p-3 text-sm whitespace-pre-wrap">
              {entry.result}
            </div>
          </div>
          {entry.file_path && (
            <div className="text-muted-foreground flex items-center gap-1 text-xs">
              <IconFileText className="size-3" />
              <span className="font-mono">{entry.file_path}</span>
            </div>
          )}
          {isLoading ? (
            <div className="text-muted-foreground py-2 text-sm">
              Loading full content...
            </div>
          ) : data?.content ? (
            <div className="space-y-1">
              <div className="text-muted-foreground text-xs font-medium">
                Full OCR Content
              </div>
              <div className="bg-muted max-h-96 overflow-auto rounded-md p-3 text-sm whitespace-pre-wrap">
                {data.content}
              </div>
            </div>
          ) : null}
        </>
      )}
      <div className="flex justify-end">
        <Button
          variant="ghost"
          size="sm"
          className="text-destructive gap-1"
          onClick={onDelete}
        >
          <IconTrash className="size-3.5" />
          Delete
        </Button>
      </div>
    </div>
  )
}
