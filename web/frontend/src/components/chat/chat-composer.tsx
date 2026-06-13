import { IconArrowUp, IconPhotoPlus, IconX } from "@tabler/icons-react"
import {
  type ClipboardEvent as ReactClipboardEvent,
  type DragEvent as ReactDragEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  useRef,
} from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { ContextUsageRing } from "@/components/chat/context-usage-ring"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { ChatAttachment, ContextUsage } from "@/store/chat"

export type ChatInputDisabledReason =
  | "gatewayUnknown"
  | "gatewayStarting"
  | "gatewayRestarting"
  | "gatewayStopping"
  | "gatewayStopped"
  | "gatewayError"
  | "websocketConnecting"
  | "websocketDisconnected"
  | "websocketError"
  | "noDefaultModel"

interface ChatComposerProps {
  input: string
  attachments: ChatAttachment[]
  onInputChange: (value: string) => void
  onAddImages: () => void
  onPaste: (event: ReactClipboardEvent<HTMLTextAreaElement>) => void
  onDragEnter: (event: ReactDragEvent<HTMLDivElement>) => void
  onDragLeave: (event: ReactDragEvent<HTMLDivElement>) => void
  onDragOver: (event: ReactDragEvent<HTMLDivElement>) => void
  onDrop: (event: ReactDragEvent<HTMLDivElement>) => void
  onRemoveAttachment: (index: number) => void
  onSend: () => void
  onContextDetail?: () => void
  inputDisabledReason: ChatInputDisabledReason | null
  canSend: boolean
  isDragActive: boolean
  contextUsage?: ContextUsage
}

export function ChatComposer({
  input,
  attachments,
  onInputChange,
  onAddImages,
  onPaste,
  onDragEnter,
  onDragLeave,
  onDragOver,
  onDrop,
  onRemoveAttachment,
  onSend,
  onContextDetail,
  inputDisabledReason,
  canSend,
  isDragActive,
  contextUsage,
}: ChatComposerProps) {
  const { t } = useTranslation()
  const canInput = inputDisabledReason === null
  const composingRef = useRef(false)
  const hasInput = input.trim().length > 0
  const disabledMessage =
    inputDisabledReason === null
      ? null
      : t(`chat.disabledPlaceholder.${inputDisabledReason}`)
  const placeholder = disabledMessage ?? t("chat.placeholder")

  const handleKeyDown = (e: ReactKeyboardEvent<HTMLTextAreaElement>) => {
    const nativeEvent = e.nativeEvent as Event & {
      isComposing?: boolean
      keyCode?: number
    }
    if (
      composingRef.current ||
      nativeEvent.isComposing ||
      nativeEvent.keyCode === 229
    ) {
      return
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      onSend()
    }
  }

  return (
    <div className="before:bg-background pointer-events-none relative z-10 -mt-[24px] shrink-0 [scrollbar-gutter:stable] overflow-y-auto px-4 pb-[calc(1rem+env(safe-area-inset-bottom))] before:pointer-events-none before:absolute before:inset-x-0 before:top-[24px] before:bottom-0 before:content-[''] md:px-8 md:pb-8 lg:px-24 xl:px-48">
      <div className="pointer-events-auto mx-auto flex max-w-[1000px] flex-col items-end">
        <div
          className={cn(
            "bg-card border-border/60 relative flex w-full flex-col rounded-2xl border p-3 shadow-sm transition-colors",
            isDragActive && "border-violet-400/70 bg-violet-500/5",
          )}
          onDragEnter={onDragEnter}
          onDragLeave={onDragLeave}
          onDragOver={onDragOver}
          onDrop={onDrop}
        >
          {isDragActive && (
            <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-2xl border-2 border-dashed border-violet-400/70 bg-violet-500/10">
              <div className="bg-background/95 text-foreground rounded-full px-4 py-2 text-sm font-medium shadow-sm">
                {t("chat.dropImagesActive")}
              </div>
            </div>
          )}

          {attachments.length > 0 && (
            <div className="mb-3 flex flex-wrap gap-2 px-2">
              {attachments.map((attachment, index) => (
                <div
                  key={`${attachment.url}-${index}`}
                  className="bg-background relative h-20 w-20 overflow-hidden rounded-xl border"
                >
                  <img
                    src={attachment.url}
                    alt={attachment.filename || t("chat.uploadedImage")}
                    className="h-full w-full object-cover"
                  />
                  <button
                    type="button"
                    onClick={() => onRemoveAttachment(index)}
                    className="bg-background/85 text-foreground absolute top-1 right-1 inline-flex h-6 w-6 items-center justify-center rounded-full border shadow-sm transition hover:bg-white"
                    aria-label={t("chat.removeImage")}
                    title={t("chat.removeImage")}
                  >
                    <IconX className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}

          <TextareaAutosize
            value={input}
            onChange={(e) => onInputChange(e.target.value)}
            onCompositionStart={() => {
              composingRef.current = true
            }}
            onCompositionEnd={() => {
              composingRef.current = false
            }}
            onPaste={onPaste}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={!canInput}
            title={disabledMessage || undefined}
            className={cn(
              "placeholder:text-muted-foreground/50 max-h-[200px] min-h-[64px] resize-none border-0 bg-transparent px-2 py-1 text-[15px] shadow-none transition-colors focus-visible:ring-0 focus-visible:outline-none dark:bg-transparent",
              !canInput && "cursor-not-allowed",
            )}
            minRows={1}
            maxRows={8}
          />

          <div className="mt-2 flex items-center justify-between px-1">
            <div className="flex items-center gap-1">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="text-muted-foreground hover:text-foreground h-8 w-8 rounded-full"
                onClick={onAddImages}
                disabled={!canInput}
                aria-label={t("chat.attachImage")}
                title={t("chat.attachImage")}
              >
                <IconPhotoPlus className="size-4" />
              </Button>
            </div>

            <div className="flex items-center gap-1.5">
              {contextUsage && (
                <ContextUsageRing
                  usage={contextUsage}
                  onDetailClick={onContextDetail}
                />
              )}
              {canInput ? (
                <span tabIndex={!canSend ? 0 : undefined}>
                  <Button
                    type="button"
                    size="icon"
                    className="size-8 rounded-full bg-violet-500 text-white transition-transform hover:bg-violet-600 active:scale-95"
                    onClick={onSend}
                    disabled={!canSend}
                    aria-label={t("chat.sendMessage")}
                  >
                    <IconArrowUp className="size-4" />
                  </Button>
                </span>
              ) : null}
            </div>
          </div>
        </div>

        <div
          aria-hidden={!hasInput}
          className={cn(
            "border-border/50 bg-muted/55 text-muted-foreground mt-2 inline-flex items-center rounded-md border px-3 py-1 text-[11px] shadow-sm transition-all duration-200 dark:bg-muted/45",
            hasInput
              ? "translate-y-0 opacity-100"
              : "pointer-events-none -translate-y-1 opacity-0",
          )}
        >
          {t("chat.composeHint")}
        </div>
      </div>
    </div>
  )
}
