import {
  IconFileCode,
  IconSparkles,
  IconWorld,
  IconX,
} from "@tabler/icons-react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import type { SkillDetailResponse, SkillSupportItem } from "@/api/skills"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

import { OriginBadge } from "./origin-badge"
import { getOriginLabel, getSkillOriginKind } from "./origin-utils"
import type { SkillDetailView } from "./types"

const DETAIL_VIEWS = [
  "preview",
  "raw",
  "meta",
] as const satisfies SkillDetailView[]

interface DetailSheetProps {
  open: boolean
  selectedSkill: SkillSupportItem | null
  selectedSkillDetail?: SkillDetailResponse
  isLoading: boolean
  error: unknown
  detailView: SkillDetailView
  onDetailViewChange: (view: SkillDetailView) => void
  onOpenChange: (open: boolean) => void
}

export function DetailSheet({
  open,
  selectedSkill,
  selectedSkillDetail,
  isLoading,
  error,
  detailView,
  onDetailViewChange,
  onOpenChange,
}: DetailSheetProps) {
  const { t } = useTranslation()

  const activeSkillDetail = selectedSkillDetail ?? selectedSkill
  const activeSkillOrigin = activeSkillDetail
    ? getSkillOriginKind(activeSkillDetail)
    : null
  const detailLineCount = selectedSkillDetail
    ? selectedSkillDetail.content.split("\n").length
    : 0
  const detailCharacterCount = selectedSkillDetail?.content.length ?? 0

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-0 p-0 shadow-2xl data-[side=right]:!w-full data-[side=right]:sm:!w-[720px] data-[side=right]:sm:!max-w-[720px]"
      >
        <SheetHeader className="bg-muted/10 border-b px-6 py-6">
          <div className="flex items-center gap-3">
            <div className="bg-primary/10 string-1 ring-primary/20 text-primary flex size-10 items-center justify-center rounded-xl">
              {activeSkillDetail?.origin_kind === "builtin" ? (
                <IconSparkles className="size-5" />
              ) : activeSkillDetail?.registry_name ? (
                <IconWorld className="size-5" />
              ) : (
                <IconFileCode className="size-5" />
              )}
            </div>
            <div className="min-w-0 flex-1 space-y-1 text-left">
              <SheetTitle className="truncate text-xl font-bold tracking-tight">
                {activeSkillDetail?.name ||
                  t("pages.agent.skills.viewer_title")}
              </SheetTitle>
              <SheetDescription className="line-clamp-2">
                {activeSkillDetail?.description ||
                  t("pages.agent.skills.viewer_description")}
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-x-hidden overflow-y-scroll px-6 py-6">
          {isLoading ? (
            <div className="space-y-6">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-24 w-full rounded-xl" />
              <Skeleton className="h-[400px] w-full rounded-xl" />
            </div>
          ) : error ? (
            <div className="text-destructive border-destructive/20 bg-destructive/5 flex h-40 flex-col items-center justify-center gap-3 rounded-xl border">
              <IconX className="size-6 opacity-80" />
              <span className="text-sm font-medium">
                {t("pages.agent.skills.load_detail_error")}
              </span>
            </div>
          ) : selectedSkillDetail ? (
            <div className="space-y-6">
              {activeSkillOrigin === "third_party" ? (
                <div className="border-border/40 bg-card/40 space-y-4 rounded-xl border p-4 shadow-sm">
                  <div className="flex flex-wrap items-center gap-2 px-1">
                    <OriginBadge
                      origin={activeSkillOrigin}
                      label={getOriginLabel(activeSkillOrigin, t)}
                    />
                  </div>

                  <div className="grid gap-3 sm:grid-cols-2">
                    {selectedSkillDetail.registry_name ? (
                      <MetadataItem
                        label={t("pages.agent.skills.metadata.registry")}
                        value={selectedSkillDetail.registry_name}
                      />
                    ) : null}
                    {selectedSkillDetail.installed_version ? (
                      <MetadataItem
                        label={t("pages.agent.skills.metadata.version")}
                        value={selectedSkillDetail.installed_version}
                      />
                    ) : null}
                    {selectedSkillDetail.registry_url ? (
                      <MetadataItem
                        label={t("pages.agent.skills.metadata.url")}
                        value={
                          <a
                            href={selectedSkillDetail.registry_url}
                            target="_blank"
                            rel="noreferrer"
                            className="text-primary hover:text-primary/80 inline break-all underline-offset-4 hover:underline"
                          >
                            {selectedSkillDetail.registry_url}
                          </a>
                        }
                        mono
                      />
                    ) : null}
                  </div>
                </div>
              ) : null}

              <div className="border-border/70 bg-muted/20 inline-flex rounded-lg border p-1 shadow-sm">
                {DETAIL_VIEWS.map((view) => (
                  <button
                    key={view}
                    type="button"
                    className={cn(
                      "rounded-md px-4 py-1.5 text-xs font-medium transition-all duration-200",
                      detailView === view
                        ? "bg-background text-foreground ring-border/30 shadow-[0_1px_3px_rgba(0,0,0,0.1)] ring-1"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                    )}
                    onClick={() => onDetailViewChange(view)}
                  >
                    {t(`pages.agent.skills.detail_tabs.${view}`)}
                  </button>
                ))}
              </div>

              {detailView === "preview" ? (
                <div className="prose prose-zinc dark:prose-invert prose-sm sm:prose-base prose-pre:rounded-xl prose-pre:border prose-pre:border-border/40 prose-pre:bg-zinc-950/90 prose-pre:shadow-sm prose-headings:tracking-tight prose-a:text-primary prose-a:no-underline hover:prose-a:underline max-w-none">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={[rehypeRaw, rehypeSanitize]}
                  >
                    {selectedSkillDetail.content}
                  </ReactMarkdown>
                </div>
              ) : null}

              {detailView === "raw" ? (
                <div className="border-border/50 overflow-x-auto rounded-xl border bg-zinc-950 p-5 shadow-sm">
                  <pre className="font-mono text-[13px] leading-relaxed break-words whitespace-pre-wrap text-zinc-100/90">
                    <code>{selectedSkillDetail.content}</code>
                  </pre>
                </div>
              ) : null}

              {detailView === "meta" ? (
                <div className="grid gap-4 sm:grid-cols-2">
                  <MetadataItem
                    label={t("pages.agent.skills.metadata.name")}
                    value={selectedSkillDetail.name}
                  />
                  <MetadataItem
                    label={t("pages.agent.skills.metadata.description")}
                    value={
                      selectedSkillDetail.description ||
                      t("pages.agent.skills.no_description")
                    }
                  />
                  <MetadataItem
                    label={t("pages.agent.skills.metadata.lines")}
                    value={String(detailLineCount)}
                  />
                  <MetadataItem
                    label={t("pages.agent.skills.metadata.characters")}
                    value={String(detailCharacterCount)}
                  />
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function MetadataItem({
  label,
  value,
  mono = false,
}: {
  label: string
  value: ReactNode
  mono?: boolean
}) {
  return (
    <div className="border-border/70 bg-muted/20 rounded-xl border px-4 py-3">
      <div className="text-muted-foreground text-[11px] font-semibold tracking-[0.18em] uppercase">
        {label}
      </div>
      <div
        className={cn(
          "text-foreground mt-2 text-sm leading-6 break-all",
          mono && "font-mono text-xs",
        )}
      >
        {value}
      </div>
    </div>
  )
}
