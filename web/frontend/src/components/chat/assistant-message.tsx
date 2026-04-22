import {
  IconBrain,
  IconCheck,
  IconChevronDown,
  IconCopy,
  IconDownload,
  IconFileText,
} from "@tabler/icons-react"
import { useAtom } from "jotai"
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
import { type ChatAttachment, showThoughtsAtom } from "@/store/chat"

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
  const [isExpanded, setIsExpanded] = useAtom(showThoughtsAtom)
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
      {!isThought && (
        <div className="text-muted-foreground/60 flex items-center justify-between gap-2 px-1 text-xs opacity-70">
          <div className="flex items-center gap-2">
            <span>PicoClaw</span>
            {formattedTimestamp && (
              <>
                <span className="opacity-50">•</span>
                <span>{formattedTimestamp}</span>
              </>
            )}
          </div>
        </div>
      )}

      <div
        className={cn(
          "relative overflow-hidden rounded-xl border",
          isThought
            ? "border-border/30 bg-muted/20 text-muted-foreground dark:border-border/20 dark:bg-muted/10"
            : "bg-card text-card-foreground border-border/60",
        )}
      >
        {isThought && (
          <div
            className="text-muted-foreground/60 hover:text-muted-foreground/80 flex cursor-pointer items-center justify-between px-3 py-2 text-[12px] font-medium transition-colors select-none"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            <div className="flex items-center gap-1.5">
              <IconBrain className="size-3.5" />
              <span>{t("chat.reasoningLabel")}</span>
            </div>
            <IconChevronDown
              className={cn(
                "size-3.5 opacity-0 transition-all duration-200 group-hover:opacity-100",
                isExpanded ? "rotate-180" : "",
              )}
            />
          </div>
        )}
        {(!isThought || isExpanded) && hasText && (
          <div
            className={cn(
              "prose dark:prose-invert prose-pre:my-2 prose-pre:overflow-x-auto prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-100 prose-pre:p-0 prose-pre:text-zinc-900 dark:prose-pre:bg-zinc-950 dark:prose-pre:text-zinc-100 max-w-none [overflow-wrap:anywhere] break-words",
              isThought
                ? "prose-p:my-1.5 prose-p:whitespace-pre-wrap px-3 pt-0 pb-3 text-[13px] leading-relaxed opacity-70"
                : "prose-p:my-2 prose-p:whitespace-pre-wrap p-4 text-[15px] leading-relaxed",
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
            className={cn("flex flex-col gap-3", hasText ? "px-4 pb-4" : "p-4")}
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

        {!isThought && hasText && (
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "bg-background/50 hover:bg-background/80 absolute top-2 right-2 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100",
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
