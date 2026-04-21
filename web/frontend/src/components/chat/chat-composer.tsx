import { IconArrowUp, IconChecklist, IconMessageCircleQuestion, IconPhotoPlus, IconSparkles, IconX } from "@tabler/icons-react"
import type { KeyboardEvent } from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { ChatAttachment, ChatInteractionMode } from "@/store/chat"

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
  mode: ChatInteractionMode
  onInputChange: (value: string) => void
  onModeChange: (mode: ChatInteractionMode) => void
  onAddImages: () => void
  onRemoveAttachment: (index: number) => void
  onSend: () => void
  inputDisabledReason: ChatInputDisabledReason | null
  canSend: boolean
}

export function ChatComposer({
  input,
  attachments,
  mode,
  onInputChange,
  onModeChange,
  onAddImages,
  onRemoveAttachment,
  onSend,
  inputDisabledReason,
  canSend,
}: ChatComposerProps) {
  const { t } = useTranslation()
  const canInput = inputDisabledReason === null
  const disabledMessage =
    inputDisabledReason === null
      ? null
      : t(`chat.disabledPlaceholder.${inputDisabledReason}`)
  const placeholder = disabledMessage ?? t("chat.placeholder")

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      onSend()
    }
  }

	const modeOptions: Array<{
		value: ChatInteractionMode
		label: string
		icon: typeof IconSparkles
	}> = [
		{ value: "agent", label: t("chat.mode.agent"), icon: IconSparkles },
		{ value: "ask", label: t("chat.mode.ask"), icon: IconMessageCircleQuestion },
		{ value: "plan", label: t("chat.mode.plan"), icon: IconChecklist },
	]

  return (
    <div className="shrink-0 px-3 pt-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] md:px-5 md:pb-5">
      <div className="bg-background mx-auto flex max-w-[880px] flex-col gap-3 rounded-xl border border-border/70 px-3 py-3 shadow-sm">
        <div className="flex flex-wrap items-center gap-1.5 px-1">
          {modeOptions.map((option) => {
            const Icon = option.icon
            const active = mode === option.value
            return (
              <Button
                key={option.value}
                type="button"
                variant={active ? "secondary" : "ghost"}
                size="sm"
                className={cn(
                  "h-8 gap-2 rounded-md px-3 text-xs",
                  active && "bg-foreground text-background hover:bg-foreground/90 hover:text-background",
                )}
                onClick={() => onModeChange(option.value)}
                disabled={!canInput}
              >
                <Icon className="size-3.5" />
                <span>{option.label}</span>
              </Button>
            )
          })}
          <span className="text-muted-foreground ml-auto hidden text-xs md:inline">
            {t(`chat.modeHint.${mode}`)}
          </span>
        </div>

        {attachments.length > 0 && (
          <div className="flex flex-wrap gap-2 px-1">
            {attachments.map((attachment, index) => (
              <div
                key={`${attachment.url}-${index}`}
                className="bg-muted/30 relative h-20 w-20 overflow-hidden rounded-lg border border-border/70"
              >
                <img
                  src={attachment.url}
                  alt={attachment.filename || t("chat.uploadedImage")}
                  className="h-full w-full object-cover"
                />
                <button
                  type="button"
                  onClick={() => onRemoveAttachment(index)}
                  className="bg-background/90 text-foreground absolute top-1 right-1 inline-flex h-6 w-6 items-center justify-center rounded-md border border-border/70 shadow-sm transition hover:bg-accent"
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
          placeholder={placeholder}
          disabled={!canInput}
          title={disabledMessage || undefined}
          className={cn(
            "placeholder:text-muted-foreground/70 max-h-[220px] min-h-[72px] resize-none border-0 bg-transparent px-1 py-1 text-[14px] leading-6 shadow-none transition-colors focus-visible:ring-0 focus-visible:outline-none dark:bg-transparent",
            !canInput && "cursor-not-allowed",
          )}
          minRows={1}
          maxRows={8}
        />
        {!canInput && disabledMessage && (
          <div className="text-muted-foreground px-1 py-1 text-xs">
            {disabledMessage}
          </div>
        )}

        <div className="flex items-center justify-between border-t border-border/60 pt-2">
          <div className="flex items-center gap-1.5">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="text-muted-foreground hover:text-foreground h-8 w-8 rounded-md"
              onClick={onAddImages}
              disabled={!canInput}
              aria-label={t("chat.attachImage")}
              title={t("chat.attachImage")}
            >
              <IconPhotoPlus className="size-4" />
            </Button>
            <span className="text-muted-foreground hidden text-xs md:inline">
              Enter to send, Shift+Enter for newline
            </span>
          </div>

          {canInput ? (
            <Button
              type="button"
              size="sm"
              className="h-8 gap-2 rounded-md bg-foreground px-3 text-background transition-colors hover:bg-foreground/85"
              onClick={onSend}
              disabled={!canSend}
            >
              <IconArrowUp className="size-4" />
              <span>{t("chat.send")}</span>
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  )
}
