import {
  IconBrain,
  IconCheck,
  IconChevronDown,
  IconCopy,
  IconDownload,
  IconFileText,
  IconTool,
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
import {
  type AssistantMessageKind,
  type ChatAttachment,
  type ChatToolCall,
} from "@/store/chat"

interface AssistantMessageProps {
  content: string
  attachments?: ChatAttachment[]
  kind?: AssistantMessageKind
  toolCalls?: ChatToolCall[]
  timestamp?: string | number
}

export function AssistantMessage({
  content,
  attachments = [],
  kind = "normal",
  toolCalls = [],
  timestamp = "",
}: AssistantMessageProps) {
  const { t } = useTranslation()
  const [isCopied, setIsCopied] = useState(false)
  const isThought = kind === "thought"
  const isToolCalls = kind === "tool_calls"
  const isCollapsedBlock = isThought || isToolCalls
  const hasText = content.trim().length > 0
  const hasToolCalls = toolCalls.length > 0
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )
  const fileAttachments = attachments.filter(
    (attachment) => attachment.type !== "image",
  )
  const [isExpanded, setIsExpanded] = useState(true)
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""

  const handleCopy = async () => {
    const markCopied = () => {
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    }

    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(content)
        markCopied()
        return
      }
    } catch {
      // HTTP 或受限环境下可能不支持 Clipboard API，继续走降级方案
    }

    const textArea = document.createElement("textarea")
    textArea.value = content
    textArea.setAttribute("readonly", "")
    textArea.style.position = "fixed"
    textArea.style.left = "-9999px"
    document.body.appendChild(textArea)
    textArea.select()

    try {
      const copied = document.execCommand("copy")
      if (copied) {
        markCopied()
      }
    } finally {
      document.body.removeChild(textArea)
    }
  }

  const collapsedLabel = isThought
    ? t("chat.reasoningLabel")
    : t("chat.toolCallsLabel")

  return (
    <div className="group flex w-full flex-col gap-1.5">
      {!isCollapsedBlock && (
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

      {(hasText || isCollapsedBlock || hasToolCalls) && (
        <div
          className={cn(
            "relative overflow-hidden rounded-xl border",
            isCollapsedBlock
              ? "border-border/30 bg-muted/20 text-muted-foreground dark:border-border/20 dark:bg-muted/10"
              : "bg-card text-card-foreground border-border/60",
          )}
        >
          {isCollapsedBlock && (
            <div
              className="text-muted-foreground/60 hover:text-muted-foreground/80 flex cursor-pointer items-center justify-between px-3 py-2 text-[12px] font-medium transition-colors select-none"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              <div className="flex items-center gap-1.5">
                {isThought ? (
                  <IconBrain className="size-3.5" />
                ) : (
                  <IconTool className="size-3.5" />
                )}
                <span>{collapsedLabel}</span>
              </div>
              <div className="flex items-center gap-2">
                {formattedTimestamp && (
                  <span className="opacity-50">{formattedTimestamp}</span>
                )}
                <IconChevronDown
                  className={cn(
                    "size-3.5 opacity-0 transition-all duration-200 group-hover:opacity-100",
                    isExpanded ? "rotate-180" : "",
                  )}
                />
              </div>
            </div>
          )}
          {(!isCollapsedBlock || isExpanded) && isToolCalls && hasToolCalls && (
            <div className="space-y-3 px-3 pt-0 pb-3">
              {toolCalls.map((toolCall, index) => {
                const explanation =
                  toolCall.extraContent?.toolFeedbackExplanation?.trim() ?? ""
                const toolName = toolCall.function?.name?.trim() ?? ""
                const toolArguments = toolCall.function?.arguments?.trim() ?? ""
                const hasFunctionSummary = toolName || toolArguments

                if (!explanation && !hasFunctionSummary) {
                  return null
                }

                return (
                  <div
                    key={toolCall.id ?? `${toolName}-${index}`}
                    className={cn(
                      "space-y-3",
                      index > 0 && "border-border/20 border-t pt-3",
                    )}
                  >
                    {explanation && (
                      <div className="space-y-1.5">
                        <div className="text-muted-foreground/55 text-[11px] font-medium tracking-wide uppercase">
                          {t("chat.toolCallExplanationLabel")}
                        </div>
                        <div className="prose dark:prose-invert prose-p:my-1.5 prose-p:whitespace-pre-wrap max-w-none text-[13px] leading-relaxed [overflow-wrap:anywhere] break-words opacity-75">
                          <ReactMarkdown
                            remarkPlugins={[remarkGfm]}
                            rehypePlugins={[
                              rehypeRaw,
                              rehypeSanitize,
                              rehypeHighlight,
                            ]}
                          >
                            {explanation}
                          </ReactMarkdown>
                        </div>
                      </div>
                    )}

                    {hasFunctionSummary && (
                      <div
                        className={cn(
                          "space-y-1.5",
                          explanation && "border-border/20 border-t pt-3",
                        )}
                      >
                        <div className="text-muted-foreground/55 text-[11px] font-medium tracking-wide uppercase">
                          {t("chat.toolCallFunctionLabel")}
                        </div>
                        <div className="bg-background/55 border-border/25 space-y-2 rounded-lg border px-3 py-2.5">
                          {toolName && (
                            <div className="text-foreground/75 font-mono text-[12px] font-semibold">
                              {toolName}
                            </div>
                          )}
                          {toolArguments && (
                            <pre className="text-muted-foreground/75 overflow-x-auto font-mono text-[12px] leading-relaxed break-words whitespace-pre-wrap">
                              {toolArguments}
                            </pre>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
          {(!isCollapsedBlock || isExpanded) && !isToolCalls && hasText && (
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

          {!isCollapsedBlock && hasText && (
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
      )}

      {imageAttachments.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-2">
          {imageAttachments.map((attachment, index) => (
            <a
              key={`${attachment.url}-${index}`}
              href={attachment.url}
              target="_blank"
              rel="noreferrer"
              className="group/img border-border/50 bg-muted/30 hover:border-border/80 relative overflow-hidden rounded-xl border shadow-sm transition-colors"
            >
              <img
                src={attachment.url}
                alt={attachment.filename || "Attached image"}
                className="max-h-80 max-w-[280px] object-contain transition-transform duration-300 group-hover/img:scale-[1.02]"
              />
              <div className="absolute inset-0 bg-black/0 transition-colors group-hover/img:bg-black/10 dark:group-hover/img:bg-black/20" />
            </a>
          ))}
        </div>
      )}

      {fileAttachments.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-3">
          {fileAttachments.map((attachment, index) => (
            <a
              key={`${attachment.url}-${index}`}
              href={attachment.url}
              download={attachment.filename}
              className="group/file border-border/60 bg-card flex w-fit max-w-sm min-w-[220px] items-center gap-3.5 rounded-xl border px-4 py-3 transition-all duration-300 hover:-translate-y-0.5 hover:border-violet-500/30 hover:shadow-sm dark:hover:border-violet-500/40"
            >
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-violet-400 ring-1 ring-violet-500/10 dark:bg-violet-500/10 dark:text-violet-400 dark:ring-violet-500/30">
                <IconFileText className="h-5 w-5" />
              </div>
              <div className="flex min-w-0 flex-1 flex-col pr-1">
                <span className="text-foreground/90 truncate text-[14px] leading-tight font-medium transition-colors group-hover/file:text-violet-600 dark:group-hover/file:text-violet-400">
                  {attachment.filename || "Download file"}
                </span>
                <span className="text-muted-foreground/70 mt-1 text-[12px] font-medium">
                  {attachment.filename?.split(".").pop()?.toUpperCase() ||
                    "FILE"}
                </span>
              </div>
              <div className="bg-muted/60 text-muted-foreground/50 dark:bg-muted/20 flex h-8 w-8 shrink-0 items-center justify-center rounded-full transition-all duration-300 group-hover/file:bg-violet-400 group-hover/file:text-white group-hover/file:shadow-sm dark:group-hover/file:bg-violet-400">
                <IconDownload className="h-4 w-4 transition-transform duration-300 group-hover/file:-translate-y-[1px]" />
              </div>
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
