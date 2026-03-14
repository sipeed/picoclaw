import { IconHistory, IconPencil, IconTrash } from "@tabler/icons-react"
import dayjs from "dayjs"
import { type KeyboardEvent, type RefObject, useState } from "react"
import { useTranslation } from "react-i18next"

import type { SessionSummary } from "@/api/sessions"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ScrollArea } from "@/components/ui/scroll-area"

interface SessionHistoryMenuProps {
  sessions: SessionSummary[]
  activeSessionId: string
  hasMore: boolean
  loadError: boolean
  loadErrorMessage: string
  observerRef: RefObject<HTMLDivElement | null>
  onOpenChange: (open: boolean) => void
  onSwitchSession: (sessionId: string) => void
  onDeleteSession: (sessionId: string) => void
  onRenameSession: (sessionId: string, title: string) => void
}

export function SessionHistoryMenu({
  sessions,
  activeSessionId,
  hasMore,
  loadError,
  loadErrorMessage,
  observerRef,
  onOpenChange,
  onSwitchSession,
  onDeleteSession,
  onRenameSession,
}: SessionHistoryMenuProps) {
  const { t } = useTranslation()
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState("")

  const startRename = (session: SessionSummary, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setRenamingId(session.id)
    setRenameValue(session.title || session.preview)
  }

  const commitRename = (sessionId: string) => {
    const trimmed = renameValue.trim()
    if (trimmed) {
      onRenameSession(sessionId, trimmed)
    }
    setRenamingId(null)
  }

  const handleRenameKeyDown = (
    e: KeyboardEvent<HTMLInputElement>,
    sessionId: string,
  ) => {
    if (e.key === "Enter") {
      e.preventDefault()
      e.stopPropagation()
      commitRename(sessionId)
    } else if (e.key === "Escape") {
      e.preventDefault()
      e.stopPropagation()
      setRenamingId(null)
    }
  }

  return (
    <DropdownMenu onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button variant="secondary" size="sm" className="h-9 gap-2">
          <IconHistory className="size-4" />
          <span className="hidden sm:inline">{t("chat.history")}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-72"
        onInteractOutside={(e) => {
          // Prevent dropdown from closing while user is typing in the rename input
          if (renamingId) e.preventDefault()
        }}
        onEscapeKeyDown={(e) => {
          if (renamingId) {
            e.preventDefault()
            setRenamingId(null)
          }
        }}
      >
        <ScrollArea className="max-h-[300px]">
          {loadError && (
            <DropdownMenuItem disabled>
              <span className="text-destructive text-xs">
                {loadErrorMessage}
              </span>
            </DropdownMenuItem>
          )}
          {sessions.length === 0 && !loadError ? (
            <DropdownMenuItem disabled>
              <span className="text-muted-foreground text-xs">
                {t("chat.noHistory")}
              </span>
            </DropdownMenuItem>
          ) : (
            sessions.map((session) => (
              <DropdownMenuItem
                key={session.id}
                className={`group relative my-0.5 flex flex-col items-start gap-0.5 pr-16 ${
                  session.id === activeSessionId ? "bg-accent" : ""
                }`}
                onClick={() => {
                  if (renamingId === session.id) return
                  onSwitchSession(session.id)
                }}
              >
                {renamingId === session.id ? (
                  <input
                    autoFocus
                    className="bg-background border-input focus:ring-ring w-full rounded border px-1 py-0.5 text-sm font-medium outline-none focus:ring-1"
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => handleRenameKeyDown(e, session.id)}
                    onBlur={() => commitRename(session.id)}
                    onClick={(e) => e.stopPropagation()}
                  />
                ) : (
                  <span className="line-clamp-1 text-sm font-medium">
                    {session.title || session.preview}
                  </span>
                )}
                <span className="text-muted-foreground text-xs">
                  {t("chat.messagesCount", {
                    count: session.message_count,
                  })}{" "}
                  · {dayjs(session.updated).fromNow()}
                </span>

                {/* Rename button */}
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t("chat.renameSession")}
                  className="text-muted-foreground hover:bg-accent absolute top-1/2 right-8 h-6 w-6 -translate-y-1/2 opacity-0 transition-opacity group-hover:opacity-100"
                  onClick={(e) => startRename(session, e)}
                >
                  <IconPencil className="h-4 w-4" />
                </Button>

                {/* Delete button */}
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t("chat.deleteSession")}
                  className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive absolute top-1/2 right-2 h-6 w-6 -translate-y-1/2 opacity-0 transition-opacity group-hover:opacity-100"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    onDeleteSession(session.id)
                  }}
                >
                  <IconTrash className="h-4 w-4" />
                </Button>
              </DropdownMenuItem>
            ))
          )}
          {hasMore && sessions.length > 0 && (
            <div ref={observerRef} className="py-2 text-center">
              <span className="text-muted-foreground animate-pulse text-xs">
                {t("chat.loadingMore")}
              </span>
            </div>
          )}
        </ScrollArea>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
