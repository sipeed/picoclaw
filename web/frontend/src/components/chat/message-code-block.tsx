import {
  IconCheck,
  IconChevronDown,
  IconCopy,
} from "@tabler/icons-react"
import { type ComponentProps, type ReactNode, useState } from "react"
import { useTranslation } from "react-i18next"

import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard"
import { cn } from "@/lib/utils"

import {
  extractCodeBlockFromPreNode,
  type MarkdownNode,
} from "./message-code-block.utils"

import { Button } from "@/components/ui/button"

interface MessageCodeBlockProps {
  code: string
  language?: string | null
  label?: string
  children?: ReactNode
  className?: string
  bodyClassName?: string
  wrapLongLines?: boolean
}

interface MarkdownCodeBlockProps extends ComponentProps<"pre"> {
  node?: MarkdownNode
}

export function MessageCodeBlock({
  code,
  language = null,
  label,
  children,
  className,
  bodyClassName,
  wrapLongLines = false,
}: MessageCodeBlockProps) {
  const { t } = useTranslation()
  const { copy, isCopied } = useCopyToClipboard()
  const [isExpanded, setIsExpanded] = useState(true)
  const blockLabel = label ?? (language ? language.toUpperCase() : t("chat.codeLabel"))
  const copyLabel = isCopied ? t("chat.copiedLabel") : t("chat.copyCode")
  const expandLabel = isExpanded ? t("chat.collapseCode") : t("chat.expandCode")

  return (
    <div
      data-picoclaw-code-block=""
      className={cn(
        "not-prose my-4 overflow-hidden rounded-lg border border-[#d0d7de] bg-[#f6f8fa] text-[#24292f] shadow-xs dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#c9d1d9]",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2 border-b border-[#d0d7de] bg-black/[0.03] px-3 py-2 dark:border-[#30363d] dark:bg-white/[0.03]">
        <span className="text-[11px] font-medium tracking-[0.16em] text-zinc-600 uppercase dark:text-zinc-400">
          {blockLabel}
        </span>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="h-7 text-zinc-600 hover:bg-zinc-300/70 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
            onClick={() => void copy(code)}
            aria-label={copyLabel}
            title={copyLabel}
          >
            {isCopied ? (
              <IconCheck className="text-green-500" />
            ) : (
              <IconCopy />
            )}
            <span className="hidden sm:inline">{copyLabel}</span>
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="h-7 text-zinc-600 hover:bg-zinc-300/70 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
            onClick={() => setIsExpanded((expanded) => !expanded)}
            aria-expanded={isExpanded}
            aria-label={expandLabel}
            title={expandLabel}
          >
            <IconChevronDown
              className={cn("transition-transform duration-200", isExpanded && "rotate-180")}
            />
            <span className="hidden sm:inline">{expandLabel}</span>
          </Button>
        </div>
      </div>

      {isExpanded && (
        <pre
          className={cn(
            "m-0 overflow-x-auto bg-transparent px-4 py-3 font-mono text-[13px] leading-6 [&_code]:block [&_code]:bg-transparent [&_code]:p-0 [&_code]:text-inherit",
            wrapLongLines ? "break-words whitespace-pre-wrap" : "whitespace-pre",
            bodyClassName,
          )}
        >
          {children ?? (
            <code className={language ? `language-${language}` : undefined}>
              {code}
            </code>
          )}
        </pre>
      )}
    </div>
  )
}

export function MarkdownCodeBlock({
  children,
  className,
  node,
}: MarkdownCodeBlockProps) {
  const { code, language } = extractCodeBlockFromPreNode(node)

  return (
    <MessageCodeBlock
      code={code}
      language={language}
      bodyClassName={className}
    >
      {children}
    </MessageCodeBlock>
  )
}
