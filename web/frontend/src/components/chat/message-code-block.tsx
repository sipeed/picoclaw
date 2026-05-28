import {
  IconCheck,
  IconChevronDown,
  IconCopy,
} from "@tabler/icons-react"
import { useAtom } from "jotai"
import hljs from "highlight.js/lib/core"
import json from "highlight.js/lib/languages/json"
import {
  type ComponentProps,
  type CSSProperties,
  type ReactNode,
  useState,
} from "react"
import { useTranslation } from "react-i18next"

import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard"
import { cn } from "@/lib/utils"
import { codeBlockWrapAtom } from "@/store/code-block"

import {
  extractCodeBlockFromPreNode,
  extractCodeBlockRenderState,
  type MarkdownNode,
  splitCodeIntoLines,
  splitHighlightedHtmlIntoLines,
  splitRenderedCodeContentIntoLines,
  trimTrailingEmptyRenderedCodeLine,
  trimTrailingEmptyStringLine,
} from "./message-code-block.utils"

import { Button } from "@/components/ui/button"

const CODE_LABEL_FONT_FAMILY =
  'ui-monospace, "SFMono-Regular", Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei UI", "Microsoft YaHei", monospace'

hljs.registerLanguage("json", json)

interface MessageCodeBlockProps {
  code: string
  language?: string | null
  label?: string
  className?: string
  bodyClassName?: string
  children?: ReactNode
  trimTrailingEmptyLine?: boolean
}

interface MarkdownCodeBlockProps extends ComponentProps<"pre"> {
  node?: MarkdownNode
}

function getHighlightedHtml(code: string, language?: string | null) {
  if (!language) {
    return null
  }

  try {
    return hljs.highlight(code, { language }).value
  } catch {
    return null
  }
}

export function MessageCodeBlock({
  code,
  language = null,
  label,
  className,
  bodyClassName,
  children,
  trimTrailingEmptyLine = false,
}: MessageCodeBlockProps) {
  const { t } = useTranslation()
  const { copy, isCopied } = useCopyToClipboard()
  const [wrapLongLines, setWrapLongLines] = useAtom(codeBlockWrapAtom)
  const [isExpanded, setIsExpanded] = useState(true)
  const blockLabel =
    label ??
    (language
      ? language.toLocaleLowerCase()
      : t("chat.codeLabel").toLocaleLowerCase())
  const copyLabel = isCopied ? t("chat.copiedLabel") : t("chat.copyCode")
  const expandLabel = isExpanded ? t("chat.collapseCode") : t("chat.expandCode")
  const wrapLabel = wrapLongLines
    ? t("chat.disableCodeWrap")
    : t("chat.enableCodeWrap")
  const renderedCodeState = children
    ? extractCodeBlockRenderState(children)
    : {
        renderedContent: null,
        className: undefined,
      }
  const highlightedHtml = !children ? getHighlightedHtml(code, language) : null
  const highlightedLines = highlightedHtml
    ? splitHighlightedHtmlIntoLines(highlightedHtml)
    : null
  const codeLines = children
    ? (trimTrailingEmptyLine
        ? trimTrailingEmptyRenderedCodeLine(
            splitRenderedCodeContentIntoLines(renderedCodeState.renderedContent),
          )
        : splitRenderedCodeContentIntoLines(renderedCodeState.renderedContent))
    : (trimTrailingEmptyLine
        ? trimTrailingEmptyStringLine(
            highlightedLines ?? splitCodeIntoLines(code),
          )
        : (highlightedLines ?? splitCodeIntoLines(code)))
  const lineNumberWidth = `${String(codeLines.length).length + 1}ch`

  return (
    <div
      data-picoclaw-code-block=""
      className={cn(
        "not-prose my-4 overflow-hidden rounded-lg border border-[#d0d7de] bg-[#f6f8fa] text-[#24292f] shadow-xs dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#c9d1d9]",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2 border-b border-[#d0d7de] bg-black/[0.03] px-3 py-2 dark:border-[#30363d] dark:bg-white/[0.03]">
        <span
          className="text-[11px] font-medium text-zinc-600 dark:text-zinc-400"
          style={{ fontFamily: CODE_LABEL_FONT_FAMILY }}
        >
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
            className="h-7 px-2 text-[11px] text-zinc-600 hover:bg-zinc-300/70 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
            onClick={() => setWrapLongLines((current) => !current)}
            aria-pressed={wrapLongLines}
            aria-label={wrapLabel}
            title={wrapLabel}
          >
            {wrapLabel}
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
            "m-0 overflow-x-auto bg-transparent px-4 py-3 font-mono text-[13px] leading-6",
            bodyClassName,
          )}
        >
          <code
            className={cn(
              "block bg-transparent p-0 text-inherit",
              children
                ? renderedCodeState.className
                : cn(highlightedHtml && "hljs", language && `language-${language}`),
            )}
          >
            {codeLines.map((line, index) => (
              <span
                key={`${index}-${line.length}`}
                className="grid grid-cols-[var(--code-line-number-width)_minmax(0,1fr)] items-start gap-x-3"
                style={
                  {
                    "--code-line-number-width": lineNumberWidth,
                  } as CSSProperties
                }
              >
                <span className="sticky left-0 z-1 select-none bg-[#f6f8fa] text-right text-zinc-500/80 dark:bg-[#0d1117] dark:text-zinc-500">
                  {index + 1}
                </span>
                {!children && highlightedLines ? (
                  <span
                    className={cn(
                      "min-w-0",
                      wrapLongLines
                        ? "break-words whitespace-pre-wrap"
                        : "whitespace-pre",
                    )}
                    dangerouslySetInnerHTML={{ __html: line }}
                  />
                ) : (
                  <span
                    className={cn(
                      "min-w-0",
                      wrapLongLines
                        ? "break-words whitespace-pre-wrap"
                        : "whitespace-pre",
                    )}
                  >
                    {line}
                  </span>
                )}
              </span>
            ))}
          </code>
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
      trimTrailingEmptyLine
    >
      {children}
    </MessageCodeBlock>
  )
}
