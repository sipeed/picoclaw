import { IconPlus } from "@tabler/icons-react"
import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { AssistantMessage } from "@/components/chat/assistant-message"
import { ChatComposer } from "@/components/chat/chat-composer"
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

export function ChatPage() {
  const { t } = useTranslation()
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [input, setInput] = useState("")

  const {
    messages,
    isTyping,
    activeSessionId,
    sendMessage,
    switchSession,
    newChat,
  } = usePicoChat()

  const { state: gwState } = useGateway()
  const isConnected = gwState === "running"

  const {
    defaultModelName,
    hasConfiguredModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
  } = useChatModels({ isConnected })

  const { sessions, hasMore, observerRef, loadSessions, handleDeleteSession } =
    useSessionHistory({
      activeSessionId,
      onDeletedActiveSession: newChat,
    })

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget
    setIsAtBottom(scrollHeight - scrollTop <= clientHeight + 10)
  }

  useEffect(() => {
    if (isAtBottom && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isTyping, isAtBottom])

  const handleSend = () => {
    if (!input.trim() || !isConnected) return
    sendMessage(input.trim())
    setInput("")
  }

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader
        title={t("navigation.chat")}
        titleExtra={
          hasConfiguredModels && (
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
          variant="outline"
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
        className="min-h-0 flex-1 overflow-y-auto px-4 py-6 md:px-8 lg:px-24 xl:px-48"
      >
        <div className="mx-auto flex w-full max-w-250 flex-col gap-8 pb-8">
          {messages.length === 0 && !isTyping && (
            <ChatEmptyState
              hasConfiguredModels={hasConfiguredModels}
              defaultModelName={defaultModelName}
              isConnected={isConnected}
            />
          )}

          {messages.map((msg) => (
            <div key={msg.id} className="flex w-full">
              {msg.role === "assistant" ? (
                <AssistantMessage
                  content={msg.content}
                  timestamp={msg.timestamp}
                />
              ) : (
                <UserMessage content={msg.content} />
              )}
            </div>
          ))}

          {isTyping && <TypingIndicator />}
        </div>
      </div>

      <ChatComposer
        input={input}
        onInputChange={setInput}
        onSend={handleSend}
        isConnected={isConnected}
        hasDefaultModel={Boolean(defaultModelName)}
      />
    </div>
  )
}
