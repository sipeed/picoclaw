import {
  IconBrain,
  IconCheck,
  IconChevronDown,
  IconCopy,
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
import { showThoughtsAtom } from "@/store/chat"

interface AssistantMessageProps {
  content: string
  isThought?: boolean
  timestamp?: string | number
}

export function AssistantMessage({
  content,
  isThought = false,
  timestamp = "",
}: AssistantMessageProps) {
  const { t } = useTranslation()
  const [isCopied, setIsCopied] = useState(false)
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
        {(!isThought || isExpanded) && (
          <div
            className={cn(
              "prose dark:prose-invert prose-pre:my-2 prose-pre:overflow-x-auto prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-100 prose-pre:p-0 prose-pre:text-zinc-900 dark:prose-pre:bg-zinc-950 dark:prose-pre:text-zinc-100 max-w-none [overflow-wrap:anywhere] break-words",
              isThought
                ? "prose-p:my-1.5 px-3 pt-0 pb-3 text-[13px] leading-relaxed opacity-70"
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
        {!isThought && (
          <Button
            variant="ghost"
            size="icon"
            className="bg-background/50 hover:bg-background/80 absolute top-2 right-2 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100"
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
