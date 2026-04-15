import {
  IconBrain,
  IconCheck,
  IconCopy,
  IconDownload,
  IconFileText,
} from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import ReactMarkdown from "react-markdown"
import rehypeHighlight from "rehype-highlight"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { Button } from "@/components/ui/button"
import { formatMessageTime } from "@/hooks/use-pico-chat"
import { cn } from "@/lib/utils"
import type { ChatAttachment } from "@/store/chat"

interface AssistantMessageProps {
  content: string
  attachments?: ChatAttachment[]
  isThought?: boolean
  timestamp?: string | number
}

export function AssistantMessage({
  content,
  attachments = [],
  isThought = false,
  timestamp = "",
}: AssistantMessageProps) {
  const { t } = useTranslation()
  const [isCopied, setIsCopied] = useState(false)
  const hasText = content.trim().length > 0
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )
  const fileAttachments = attachments.filter(
    (attachment) => attachment.type !== "image",
  )
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""

  const handleCopy = () => {
    navigator.clipboard.writeText(content).then(() => {
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    })
  }

  return (
    <div className="group flex w-full flex-col gap-1.5">
      <div className="text-muted-foreground flex items-center justify-between gap-2 px-1 text-xs opacity-70">
        <div className="flex items-center gap-2">
          <span>PicoClaw</span>
          {isThought && (
            <span className="inline-flex items-center gap-1 rounded-full border border-amber-300/80 bg-amber-100/80 px-2 py-0.5 text-[11px] font-medium text-amber-800 dark:border-amber-500/40 dark:bg-amber-500/15 dark:text-amber-200">
              <IconBrain className="size-3" />
              <span>{t("chat.reasoningLabel")}</span>
            </span>
          )}
          {formattedTimestamp && (
            <>
              <span className="opacity-50">•</span>
              <span>{formattedTimestamp}</span>
            </>
          )}
        </div>
      </div>

      <div
        className={cn(
          "relative overflow-hidden rounded-xl border",
          isThought
            ? "border-amber-200/90 bg-amber-50/70 text-amber-950 dark:border-amber-500/35 dark:bg-amber-500/10 dark:text-amber-100"
            : "bg-card text-card-foreground",
        )}
      >
        {hasText && (
          <div
            className={cn(
              "prose dark:prose-invert prose-pre:my-2 prose-pre:overflow-x-auto prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-100 prose-pre:p-0 dark:prose-pre:bg-zinc-950 max-w-none [overflow-wrap:anywhere] break-words",
              isThought
                ? "prose-p:my-1.5 p-3 text-[13px] leading-relaxed opacity-90"
                : "prose-p:my-2 p-4 text-[15px] leading-relaxed",
            )}
          >
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw, rehypeSanitize, rehypeHighlight]}
            >
              {content}
            </ReactMarkdown>
          </div>
        )}

        {(imageAttachments.length > 0 || fileAttachments.length > 0) && (
          <div
            className={cn(
              "flex flex-col gap-3",
              hasText ? "px-4 pb-4" : "p-4",
            )}
          >
            {imageAttachments.length > 0 && (
              <div className="flex flex-wrap gap-3">
                {imageAttachments.map((attachment, index) => (
                  <a
                    key={`${attachment.url}-${index}`}
                    href={attachment.url}
                    target="_blank"
                    rel="noreferrer"
                    className="overflow-hidden rounded-xl border"
                  >
                    <img
                      src={attachment.url}
                      alt={attachment.filename || "Attachment"}
                      className="max-h-72 max-w-full object-cover"
                    />
                  </a>
                ))}
              </div>
            )}

            {fileAttachments.length > 0 && (
              <div className="flex flex-col gap-2">
                {fileAttachments.map((attachment, index) => (
                  <a
                    key={`${attachment.url}-${index}`}
                    href={attachment.url}
                    download={attachment.filename}
                    className="bg-background/70 hover:bg-background/90 flex items-center justify-between gap-3 rounded-xl border px-3 py-2 transition-colors"
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <IconFileText className="text-muted-foreground size-4 shrink-0" />
                      <span className="truncate text-sm">
                        {attachment.filename || "Download attachment"}
                      </span>
                    </span>
                    <IconDownload className="text-muted-foreground size-4 shrink-0" />
                  </a>
                ))}
              </div>
            )}
          </div>
        )}

        {hasText && (
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "absolute top-2 right-2 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100",
              isThought
                ? "bg-amber-100/70 hover:bg-amber-200/80 dark:bg-amber-500/20 dark:hover:bg-amber-400/30"
                : "bg-background/50 hover:bg-background/80",
            )}
            onClick={handleCopy}
          >
            {isCopied ? (
              <IconCheck className="h-4 w-4 text-green-500" />
            ) : (
              <IconCopy className="text-muted-foreground h-4 w-4" />
            )}
          </Button>
        )}
      </div>
    </div>
  )
}
