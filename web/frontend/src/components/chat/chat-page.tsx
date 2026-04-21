import { IconPlus } from "@tabler/icons-react"
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { AssistantMessage } from "@/components/chat/assistant-message"
import {
  ChatComposer,
  type ChatInputDisabledReason,
} from "@/components/chat/chat-composer"
import { ChatEmptyState } from "@/components/chat/chat-empty-state"
import { ModelSelector } from "@/components/chat/model-selector"
import { SessionHistoryMenu } from "@/components/chat/session-history-menu"
import { TypingIndicator } from "@/components/chat/typing-indicator"
import { UserMessage } from "@/components/chat/user-message"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { useChatModels } from "@/hooks/use-chat-models"
import { useGateway } from "@/hooks/use-gateway"
import { usePicoChat } from "@/hooks/use-pico-chat"
import { useSessionHistory } from "@/hooks/use-session-history"
import type {
  ChatAttachment,
  ChatInteractionMode,
  ChatMessage,
  ChatStructuredContent,
  ChatStructuredProgress,
  ChatStructuredValue,
  ConnectionState,
} from "@/store/chat"
import type { GatewayState } from "@/store/gateway"

const MAX_IMAGE_SIZE_BYTES = 7 * 1024 * 1024
const MAX_IMAGE_SIZE_LABEL = "7 MB"
const ALLOWED_IMAGE_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
  "image/bmp",
])

function flattenStructuredParts(
  structured: ChatStructuredValue | undefined,
): ChatStructuredContent[] {
  if (!structured) {
    return []
  }
  return Array.isArray(structured) ? structured : [structured]
}

function isToolProgressPart(
  part: ChatStructuredContent,
): part is ChatStructuredProgress {
  return part.type === "progress" && part.kind === "agent/tool-exec"
}

function isProgressOnlyAssistantMessage(message: ChatMessage): boolean {
  if (message.role !== "assistant" || message.kind === "thought") {
    return false
  }

  if (message.content.trim() || (message.attachments?.length ?? 0) > 0) {
    return false
  }

  const parts = flattenStructuredParts(message.structured)
  return parts.length > 0 && parts.every(isToolProgressPart)
}

function isMergeableAssistantResultMessage(message: ChatMessage): boolean {
  return message.role === "assistant" && message.kind !== "thought"
}

function mergeStructuredValues(
  left: ChatStructuredValue | undefined,
  right: ChatStructuredValue | undefined,
): ChatStructuredValue | undefined {
  const merged = [...flattenStructuredParts(left), ...flattenStructuredParts(right)]
  if (merged.length === 0) {
    return undefined
  }
  return merged
}

function aggregateRenderableMessages(messages: ChatMessage[]): ChatMessage[] {
  const aggregated: ChatMessage[] = []

  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index]

    if (!isProgressOnlyAssistantMessage(message)) {
      aggregated.push(message)
      continue
    }

    let mergedMessage = message
    let cursor = index + 1

    while (cursor < messages.length && isProgressOnlyAssistantMessage(messages[cursor])) {
      const progressMessage = messages[cursor]
      mergedMessage = {
        ...mergedMessage,
        id: `${mergedMessage.id}__${progressMessage.id}`,
        structured: mergeStructuredValues(mergedMessage.structured, progressMessage.structured),
        timestamp: progressMessage.timestamp,
      }
      cursor += 1
    }

    const nextMessage = messages[cursor]
    if (nextMessage && isMergeableAssistantResultMessage(nextMessage)) {
      mergedMessage = {
        ...nextMessage,
        id: `${mergedMessage.id}__${nextMessage.id}`,
        structured: mergeStructuredValues(mergedMessage.structured, nextMessage.structured),
      }
      index = cursor
    } else {
      index = cursor - 1
    }

    aggregated.push(mergedMessage)
  }

  return aggregated
}

function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === "string") {
        resolve(reader.result)
        return
      }
      reject(new Error("Failed to read file"))
    }
    reader.onerror = () =>
      reject(reader.error || new Error("Failed to read file"))
    reader.readAsDataURL(file)
  })
}

function resolveChatInputDisabledReason({
  hasDefaultModel,
  connectionState,
  gatewayState,
}: {
  hasDefaultModel: boolean
  connectionState: ConnectionState
  gatewayState: GatewayState
}): ChatInputDisabledReason | null {
  if (gatewayState === "unknown") {
    return "gatewayUnknown"
  }

  if (gatewayState === "starting") {
    return "gatewayStarting"
  }

  if (gatewayState === "restarting") {
    return "gatewayRestarting"
  }

  if (gatewayState === "stopping") {
    return "gatewayStopping"
  }

  if (gatewayState === "stopped") {
    return "gatewayStopped"
  }

  if (gatewayState === "error") {
    return "gatewayError"
  }

  if (connectionState === "connecting") {
    return "websocketConnecting"
  }

  if (connectionState === "error") {
    return "websocketError"
  }

  if (connectionState === "disconnected") {
    return "websocketDisconnected"
  }

  if (!hasDefaultModel) {
    return "noDefaultModel"
  }

  return null
}

export function ChatPage() {
  const { t } = useTranslation()
  const scrollRef = useRef<HTMLDivElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [hasScrolled, setHasScrolled] = useState(false)
  const [input, setInput] = useState("")
  const [attachments, setAttachments] = useState<ChatAttachment[]>([])
  const [mode, setMode] = useState<ChatInteractionMode>("agent")

  const {
    messages,
    connectionState,
    isTyping,
    activeSessionId,
    sendMessage,
    switchSession,
    newChat,
  } = usePicoChat()

  const { state: gwState } = useGateway()
  const isGatewayRunning = gwState === "running"

  const {
    defaultModelName,
    hasAvailableModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
  } = useChatModels({ isConnected: isGatewayRunning })
  const hasDefaultModel = Boolean(defaultModelName)
  const inputDisabledReason = resolveChatInputDisabledReason({
    hasDefaultModel,
    connectionState,
    gatewayState: gwState,
  })
  const canInput = inputDisabledReason === null

  const {
    sessions,
    hasMore,
    loadError,
    loadErrorMessage,
    observerRef,
    loadSessions,
    handleDeleteSession,
  } = useSessionHistory({
    activeSessionId,
    onDeletedActiveSession: newChat,
  })

  const syncScrollState = (element: HTMLDivElement) => {
    const { scrollTop, scrollHeight, clientHeight } = element
    setHasScrolled(scrollTop > 0)
    setIsAtBottom(scrollHeight - scrollTop <= clientHeight + 10)
  }

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    syncScrollState(e.currentTarget)
  }

  useEffect(() => {
    if (scrollRef.current) {
      if (isAtBottom) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight
      }
      syncScrollState(scrollRef.current)
    }
  }, [messages, isTyping, isAtBottom])

  const handleSend = () => {
    if ((!input.trim() && attachments.length === 0) || !canInput) return
    if (
      sendMessage({
        content: input,
        attachments,
        mode,
      })
    ) {
      setInput("")
      setAttachments([])
    }
  }

  const handleSelectOption = (value: string) => {
    if (!canInput) return
    sendMessage({ content: value, mode })
  }

  const handleAddImages = () => {
    if (!canInput) return
    fileInputRef.current?.click()
  }

  const handleRemoveAttachment = (index: number) => {
    setAttachments((prev) => prev.filter((_, itemIndex) => itemIndex !== index))
  }

  const handleImageSelection = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? [])
    event.target.value = ""

    if (files.length === 0) {
      return
    }

    const nextAttachments: ChatAttachment[] = []
    for (const file of files) {
      if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
        toast.error(
          t("chat.invalidImage", {
            name: file.name,
          }),
        )
        continue
      }

      if (file.size > MAX_IMAGE_SIZE_BYTES) {
        toast.error(
          t("chat.imageTooLarge", {
            name: file.name,
            size: MAX_IMAGE_SIZE_LABEL,
          }),
        )
        continue
      }

      try {
        nextAttachments.push({
          type: "image",
          filename: file.name,
          url: await readFileAsDataUrl(file),
        })
      } catch {
        toast.error(
          t("chat.imageReadFailed", {
            name: file.name,
          }),
        )
      }
    }

    if (nextAttachments.length > 0) {
      setAttachments(nextAttachments.slice(0, 1))
    }
  }

  const canSubmit =
    canInput && (Boolean(input.trim()) || attachments.length > 0)
  const renderMessages = useMemo(
    () => aggregateRenderableMessages(messages),
    [messages],
  )

  return (
    <div className="bg-muted/35 flex h-full min-h-0 flex-col">
      <PageHeader
        title={t("navigation.chat")}
        className={`transition-shadow ${
          hasScrolled ? "border-border/60 bg-background/90 shadow-xs backdrop-blur" : "border-transparent bg-transparent shadow-none"
        }`}
        titleExtra={
          hasAvailableModels && (
            <ModelSelector
              defaultModelName={defaultModelName}
              apiKeyModels={apiKeyModels}
              oauthModels={oauthModels}
              localModels={localModels}
              onValueChange={handleSetDefault}
            />
          )
        }
      >
        <Button
          variant="secondary"
          size="sm"
          onClick={newChat}
          className="h-9 gap-2"
        >
          <IconPlus className="size-4" />
          <span className="hidden sm:inline">{t("chat.newChat")}</span>
        </Button>

        <SessionHistoryMenu
          sessions={sessions}
          activeSessionId={activeSessionId}
          hasMore={hasMore}
          loadError={loadError}
          loadErrorMessage={loadErrorMessage}
          observerRef={observerRef}
          onOpenChange={(open) => {
            if (open) {
              void loadSessions(true)
            }
          }}
          onSwitchSession={switchSession}
          onDeleteSession={handleDeleteSession}
        />
      </PageHeader>

      <div className="min-h-0 flex-1 px-2 pb-2 md:px-4 md:pb-4">
        <div className="bg-background/92 ring-border/60 mx-auto flex h-full w-full max-w-[1380px] min-h-0 flex-col overflow-hidden rounded-2xl border shadow-sm ring-1">
          <div
            ref={scrollRef}
            onScroll={handleScroll}
            className="min-h-0 flex-1 overflow-y-auto px-4 py-6 md:px-8 lg:px-10"
          >
            <div className="mx-auto flex w-full max-w-[880px] flex-col gap-5 pb-12">
          {renderMessages.length === 0 && !isTyping && (
            <ChatEmptyState
              hasAvailableModels={hasAvailableModels}
              defaultModelName={defaultModelName}
              isConnected={isGatewayRunning}
            />
          )}

          {renderMessages.map((msg) => (
            <div
              key={msg.id}
              className="flex w-full"
            >
              {msg.role === "assistant" ? (
                <AssistantMessage
                  content={msg.content}
                  isThought={msg.kind === "thought"}
                  timestamp={msg.timestamp}
                  structured={msg.structured}
                  onSelectOption={handleSelectOption}
                />
              ) : (
                <UserMessage
                  content={msg.content}
                  attachments={msg.attachments}
                  timestamp={msg.timestamp}
                />
              )}
            </div>
          ))}

          {isTyping && <TypingIndicator />}
            </div>
          </div>

          <div className="border-border/70 bg-background/96 border-t">
            <ChatComposer
              input={input}
              attachments={attachments}
              mode={mode}
              onInputChange={setInput}
              onModeChange={setMode}
              onAddImages={handleAddImages}
              onRemoveAttachment={handleRemoveAttachment}
              onSend={handleSend}
              inputDisabledReason={inputDisabledReason}
              canSend={canSubmit}
            />
          </div>
        </div>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
        className="hidden"
        onChange={handleImageSelection}
      />
    </div>
  )
}
