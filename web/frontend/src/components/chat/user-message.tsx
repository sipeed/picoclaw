import { formatMessageTime } from "@/hooks/use-pico-chat"
import type { ChatAttachment } from "@/store/chat"

interface UserMessageProps {
  content: string
  attachments?: ChatAttachment[]
  timestamp?: string | number
}

export function UserMessage({
  content,
  attachments = [],
  timestamp,
}: UserMessageProps) {
  const hasText = content.trim().length > 0
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )
  const formattedTimestamp = timestamp ? formatMessageTime(timestamp) : ""

  return (
    <div className="ml-auto flex w-full max-w-[820px] justify-end gap-3">
      <div className="flex min-w-0 max-w-[80%] flex-col items-end gap-2">
        <div className="text-muted-foreground flex items-center gap-2 text-[11px] uppercase tracking-[0.14em]">
          <span>You</span>
          {formattedTimestamp ? <span className="opacity-60">{formattedTimestamp}</span> : null}
        </div>
      {imageAttachments.length > 0 && (
        <div className="flex flex-wrap justify-end gap-2">
          {imageAttachments.map((attachment, index) => (
            <div
              key={`${attachment.url}-${index}`}
              className="overflow-hidden rounded-xl border border-border/70 bg-card shadow-sm"
            >
              <img
                src={attachment.url}
                alt={attachment.filename || "Uploaded image"}
                className="max-h-72 max-w-full object-cover"
              />
            </div>
          ))}
        </div>
      )}

      {hasText && (
        <div className="w-full rounded-xl border border-blue-200/70 bg-blue-50/70 px-4 py-3 text-[14px] leading-6 wrap-break-word whitespace-pre-wrap text-slate-900 shadow-sm dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-slate-100">
          {content}
        </div>
      )}
      </div>

      <div className="bg-muted text-muted-foreground mt-5 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-border/70 text-[11px] font-semibold uppercase">
        You
      </div>
    </div>
  )
}
