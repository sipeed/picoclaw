import { useEffect, useMemo, useState } from "react"
import ReactMarkdown from "react-markdown"
import rehypeHighlight from "rehype-highlight"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { cn } from "@/lib/utils"

interface StreamingMarkdownProps {
  content: string
  animate: boolean
  className?: string
}

function nextChunkSize(remaining: number) {
  if (remaining > 480) return 6
  if (remaining > 240) return 4
  if (remaining > 120) return 3
  return 1
}

function nextFrameDelay(nextChunk: string, remaining: number) {
  if (/[,.!?;:\n]$/.test(nextChunk)) {
    return 90
  }
  if (remaining > 480) return 32
  if (remaining > 240) return 38
  if (remaining > 120) return 44
  return 52
}

export function StreamingMarkdown({
  content,
  animate,
  className,
}: StreamingMarkdownProps) {
  const [displayedContent, setDisplayedContent] = useState(() =>
    animate ? "" : content,
  )

  useEffect(() => {
    if (!animate) {
      setDisplayedContent(content)
      return
    }

    setDisplayedContent((current) => {
      if (current === "" && content.length > 0) {
        return ""
      }
      if (content.length < current.length) {
        return content
      }
      return current
    })
  }, [animate, content])

  useEffect(() => {
    if (!animate || displayedContent === content) {
      return
    }

    const remaining = content.length - displayedContent.length
    const chunkSize = nextChunkSize(remaining)
    const nextChunk = content.slice(
      displayedContent.length,
      displayedContent.length + chunkSize,
    )

    const timer = window.setTimeout(() => {
      setDisplayedContent((current) => {
        if (current === content) {
          return current
        }

        const nextLength = Math.min(
          content.length,
          current.length + chunkSize,
        )
        return content.slice(0, nextLength)
      })
    }, nextFrameDelay(nextChunk, remaining))

    return () => window.clearTimeout(timer)
  }, [animate, content, displayedContent])

  const markdown = useMemo(
    () => (animate ? displayedContent : content),
    [animate, content, displayedContent],
  )

  return (
    <div className={cn("relative", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, rehypeSanitize, rehypeHighlight]}
      >
        {markdown}
      </ReactMarkdown>
      {animate && (
        <span
          aria-hidden="true"
          className="bg-foreground/70 ml-0.5 inline-block h-[1.05em] w-0.5 animate-pulse align-[-0.12em]"
        />
      )}
    </div>
  )
}
