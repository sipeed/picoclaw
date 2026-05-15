import { IconPlus } from "@tabler/icons-react"
import { useAtom } from "jotai"
import { type ChangeEvent, useEffect, useRef, useState } from "react"
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
import { Switch } from "@/components/ui/switch"
import { useChatModels } from "@/hooks/use-chat-models"
import { useGateway } from "@/hooks/use-gateway"
import { usePicoChat } from "@/hooks/use-pico-chat"
import { useSessionHistory } from "@/hooks/use-session-history"
import type { ConnectionState } from "@/store/chat"
import type { ChatAttachment } from "@/store/chat"
import { showAssistantDetailsAtom } from "@/store/chat"
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
  const [showAssistantDetails, setShowAssistantDetails] = useAtom(
    showAssistantDetailsAtom,
  )

  const {
    messages,
    connectionState,
    isTyping,
    activeSessionId,
    contextUsage,
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
    const { clientHeight, scrollHeight, scrollTop } = element
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
      })
    ) {
      setInput("")
      setAttachments([])
    }
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

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader
        title={t("navigation.chat")}
        className={`transition-shadow ${
          hasScrolled ? "shadow-xs" : "shadow-none"
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
        <div className="border-border/60 hidden items-center gap-2 rounded-lg border px-3 py-1.5 sm:flex">
          <span className="text-muted-foreground text-sm">
            {t("chat.showAssistantDetails")}
          </span>
          <Switch
            checked={showAssistantDetails}
            onCheckedChange={setShowAssistantDetails}
            aria-label={t("chat.showAssistantDetails")}
            size="sm"
          />
        </div>

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

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="min-h-0 flex-1 overflow-y-auto px-4 py-6 [scrollbar-gutter:stable] md:px-8 lg:px-24 xl:px-48"
      >
        <div className="mx-auto flex w-full max-w-250 flex-col gap-8 pb-8">
          {messages.length === 0 && !isTyping && (
            <ChatEmptyState
              hasAvailableModels={hasAvailableModels}
              defaultModelName={defaultModelName}
              isConnected={isGatewayRunning}
            />
          )}

          {messages.map((msg) => {
            if (
              !showAssistantDetails &&
              (msg.kind === "thought" || msg.kind === "tool_calls")
            ) {
              return null
            }

            return (
              <div key={msg.id} className="flex w-full">
                {msg.role === "assistant" ? (
                  <AssistantMessage
                    content={msg.content}
                    attachments={msg.attachments}
                    kind={msg.kind}
                    toolCalls={msg.toolCalls}
                    timestamp={msg.timestamp}
                  />
                ) : (
                  <UserMessage
                    content={msg.content}
                    attachments={msg.attachments}
                    timestamp={msg.timestamp}
                  />
                )}
              </div>
            )
          })}

          {isTyping && <TypingIndicator />}
        </div>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
        className="hidden"
        onChange={handleImageSelection}
      />

      <ChatComposer
        input={input}
        attachments={attachments}
        onInputChange={setInput}
        onAddImages={handleAddImages}
        onRemoveAttachment={handleRemoveAttachment}
        onSend={handleSend}
        onContextDetail={() => {
          if (sendMessage({ content: "/context", attachments: [] })) {
            setInput("")
          }
        }}
        inputDisabledReason={inputDisabledReason}
        canSend={canSubmit}
        contextUsage={contextUsage}
      />
    </div>
  )
}
