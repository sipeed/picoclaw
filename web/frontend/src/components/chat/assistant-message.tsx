import { IconCheck, IconCopy } from "@tabler/icons-react"
import { useEffect, useRef, useState } from "react"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { Button } from "@/components/ui/button"
import { formatMessageTime } from "@/hooks/use-pico-chat"

const STREAM_FRAME_MS = 18
const STREAM_STEPS = 60

interface AssistantMessageProps {
  content: string
  timestamp?: string | number
  isStreaming?: boolean
}

export function AssistantMessage({
  content,
  timestamp = "",
  isStreaming = false,
}: AssistantMessageProps) {
  const [isCopied, setIsCopied] = useState(false)
  const [displayedContent, setDisplayedContent] = useState(content)
  const timerRef = useRef<number | null>(null)
  const displayedRef = useRef(displayedContent)
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""

  const syncDisplayedContent = (nextContent: string) => {
    setDisplayedContent((previous) =>
      previous === nextContent ? previous : nextContent,
    )
  }

  useEffect(() => {
    displayedRef.current = displayedContent
  }, [displayedContent])

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (timerRef.current !== null) {
      window.clearInterval(timerRef.current)
      timerRef.current = null
    }

    if (!isStreaming || content.trim() === "") {
      queueMicrotask(() => syncDisplayedContent(content))
      return
    }

    const currentDisplayed = displayedRef.current
    const targetRunes = Array.from(content)
    const start = content.startsWith(currentDisplayed) ? currentDisplayed : ""
    const startLen = Array.from(start).length

    if (startLen >= targetRunes.length) {
      queueMicrotask(() => syncDisplayedContent(content))
      return
    }

    let cursor = startLen
    const chunkSize = Math.max(
      1,
      Math.ceil((targetRunes.length - startLen) / STREAM_STEPS),
    )

    queueMicrotask(() => syncDisplayedContent(start))
    timerRef.current = window.setInterval(() => {
      cursor = Math.min(targetRunes.length, cursor + chunkSize)
      setDisplayedContent(targetRunes.slice(0, cursor).join(""))
      if (cursor >= targetRunes.length && timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
    }, STREAM_FRAME_MS)
  }, [content, isStreaming])

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
          {formattedTimestamp && (
            <>
              <span className="opacity-50">•</span>
              <span>{formattedTimestamp}</span>
            </>
          )}
        </div>
      </div>

      <div className="bg-card text-card-foreground relative overflow-hidden rounded-xl border">
        <div className="prose dark:prose-invert prose-p:my-2 prose-pre:my-2 prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-950 prose-pre:p-3 max-w-none p-4 text-[15px] leading-relaxed">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeRaw, rehypeSanitize]}
          >
            {displayedContent}
          </ReactMarkdown>
        </div>
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
      </div>
    </div>
  )
}
