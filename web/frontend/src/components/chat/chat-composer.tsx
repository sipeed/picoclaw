import { IconArrowUp, IconPhotoPlus, IconX } from "@tabler/icons-react"
import type { KeyboardEvent } from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { ChatAttachment } from "@/store/chat"

interface ChatComposerProps {
  input: string
  attachments: ChatAttachment[]
  onInputChange: (value: string) => void
  onAddImages: () => void
  onRemoveAttachment: (index: number) => void
  onSend: () => void
  isConnected: boolean
  hasDefaultModel: boolean
  canSend: boolean
}

export function ChatComposer({
  input,
  attachments,
  onInputChange,
  onAddImages,
  onRemoveAttachment,
  onSend,
  isConnected,
  hasDefaultModel,
  canSend,
}: ChatComposerProps) {
  const { t } = useTranslation()
  const canInput = isConnected && hasDefaultModel

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      onSend()
    }
  }

  return (
    <div className="bg-background shrink-0 px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))] md:px-8 md:pb-8 lg:px-24 xl:px-48">
      <div className="bg-card border-border/80 mx-auto flex max-w-[1000px] flex-col rounded-2xl border p-3 shadow-md">
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
          onKeyDown={handleKeyDown}
          placeholder={t("chat.placeholder")}
          disabled={!canInput}
          className={cn(
            "placeholder:text-muted-foreground/50 max-h-[200px] min-h-[60px] resize-none border-0 bg-transparent px-2 py-1 text-[15px] shadow-none transition-colors focus-visible:ring-0 focus-visible:outline-none dark:bg-transparent",
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

          <Button
            type="button"
            size="icon"
            className="size-8 rounded-full bg-violet-500 text-white transition-transform hover:bg-violet-600 active:scale-95"
            onClick={onSend}
            disabled={!canSend}
          >
            <IconArrowUp className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
