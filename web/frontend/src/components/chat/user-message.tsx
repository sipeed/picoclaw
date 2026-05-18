import { IconCheck, IconCopy } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard"
import { cn } from "@/lib/utils"
import type { ChatAttachment } from "@/store/chat"
import { useTranslation } from "react-i18next"

interface UserMessageProps {
  content: string
  attachments?: ChatAttachment[]
}

export function UserMessage({ content, attachments = [] }: UserMessageProps) {
  const { t } = useTranslation()
  const { copy, isCopied } = useCopyToClipboard()
  const hasText = content.trim().length > 0
  const isCommand = content.trim().startsWith("/")
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )
  const copyMessageLabel = isCopied
    ? t("chat.copiedLabel")
    : t("chat.copyMessage")

  return (
    <div className="group flex w-full flex-col items-end gap-1.5">
      {imageAttachments.length > 0 && (
        <div className="flex max-w-[70%] flex-wrap justify-end gap-2">
          {imageAttachments.map((attachment, index) => (
            <img
              key={`${attachment.url}-${index}`}
              src={attachment.url}
              alt={attachment.filename || "Uploaded image"}
              className="max-h-72 max-w-full object-cover"
            />
          ))}
        </div>
      )}

      {hasText && (
        <div className="relative max-w-[70%]">
          <div
            className={cn(
              "wrap-break-word whitespace-pre-wrap",
              isCommand
                ? "rounded-xl border border-zinc-200 bg-transparent px-4 py-3 font-mono text-[14px] text-zinc-800 dark:border-zinc-800/60 dark:bg-[#121212] dark:text-zinc-200 dark:shadow-sm"
                : "rounded-2xl rounded-tr-sm bg-violet-500 px-5 py-3 text-[15px] leading-relaxed text-white shadow-sm",
            )}
          >
            {isCommand ? (
              <div className="flex items-start gap-2.5">
                <span className="font-bold text-emerald-600 select-none dark:text-emerald-400">
                  ❯
                </span>
                <span className="mt-[1px]">{content}</span>
              </div>
            ) : (
              content
            )}
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className={cn(
              "bg-background/75 hover:bg-background absolute top-2 right-2 h-7 w-7 opacity-0 shadow-xs transition-opacity group-hover:opacity-100",
              isCommand
                ? "text-zinc-700 dark:text-zinc-200"
                : "text-violet-700 dark:text-violet-100",
            )}
            onClick={() => void copy(content)}
            aria-label={copyMessageLabel}
            title={copyMessageLabel}
          >
            {isCopied ? (
              <IconCheck className="h-4 w-4 text-green-500" />
            ) : (
              <IconCopy className="h-4 w-4" />
            )}
          </Button>
        </div>
      )}
    </div>
  )
}
