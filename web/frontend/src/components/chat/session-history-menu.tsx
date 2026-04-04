import { IconBrandWhatsapp, IconCheck, IconCopy, IconHistory, IconLink, IconPencil, IconTrash, IconX } from "@tabler/icons-react"
import dayjs from "dayjs"
import { type RefObject, useState } from "react"
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
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState("")
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [copiedWaId, setCopiedWaId] = useState<string | null>(null)

  const startRename = (e: React.MouseEvent, session: SessionSummary) => {
    e.preventDefault()
    e.stopPropagation()
    setEditingId(session.id)
    setEditValue(session.title || session.preview || "")
  }

  const confirmRename = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (editingId && editValue.trim()) {
      onRenameSession(editingId, editValue.trim())
    }
    setEditingId(null)
  }

  const cancelRename = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setEditingId(null)
  }

  const copyLink = (e: React.MouseEvent, sessionId: string) => {
    e.preventDefault()
    e.stopPropagation()
    const url = `${window.location.origin}/?session=${sessionId}`
    navigator.clipboard.writeText(url)
    setCopiedId(sessionId)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const copyWhatsApp = (e: React.MouseEvent, peerName: string, sessionId: string) => {
    e.preventDefault()
    e.stopPropagation()
    const phone = peerName.replace(/\+/g, "")
    navigator.clipboard.writeText(`wa.me/${phone}`)
    setCopiedWaId(sessionId)
    setTimeout(() => setCopiedWaId(null), 2000)
  }

  return (
    <DropdownMenu onOpenChange={(open) => { if (!open) setEditingId(null); onOpenChange(open) }}>
      <DropdownMenuTrigger asChild>
        <Button variant="secondary" size="sm" className="h-9 gap-2">
          <IconHistory className="size-4" />
          <span className="hidden sm:inline">{t("chat.history")}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
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
                className={`group relative my-0.5 flex flex-col items-start gap-0.5 pr-24 ${
                  session.id === activeSessionId ? "bg-accent" : ""
                }`}
                onClick={() => { if (!editingId) onSwitchSession(session.id) }}
              >
                {editingId === session.id ? (
                  <div className="flex w-full items-center gap-1" onClick={(e) => e.stopPropagation()}>
                    <input
                      className="bg-background border-input h-6 flex-1 rounded border px-1 text-sm"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") confirmRename(e as unknown as React.MouseEvent)
                        if (e.key === "Escape") setEditingId(null)
                      }}
                      autoFocus
                    />
                    <Button variant="ghost" size="icon" className="h-5 w-5 text-green-500" onClick={confirmRename}>
                      <IconCheck className="h-3 w-3" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-5 w-5" onClick={cancelRename}>
                      <IconX className="h-3 w-3" />
                    </Button>
                  </div>
                ) : (
                  <>
                    <span className="line-clamp-1 text-sm font-medium">
                      {session.title || session.preview}
                    </span>
                    <span className="text-muted-foreground text-xs">
                      {t("chat.messagesCount", { count: session.message_count })}{" "}
                      · {dayjs(session.updated).fromNow()}
                    </span>
                  </>
                )}
                {editingId !== session.id && (
                  <div className="absolute top-1/2 right-2 flex -translate-y-1/2 gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("chat.renameSession")}
                      className="text-muted-foreground hover:text-foreground h-6 w-6"
                      onClick={(e) => startRename(e, session)}
                    >
                      <IconPencil className="h-3.5 w-3.5" />
                    </Button>
                    {session.peer_name && (
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label="wa.me"
                        className="text-muted-foreground hover:text-green-600 h-6 w-6"
                        onClick={(e) => copyWhatsApp(e, session.peer_name!, session.id)}
                      >
                        {copiedWaId === session.id ? (
                          <IconCheck className="h-3.5 w-3.5 text-green-500" />
                        ) : (
                          <IconBrandWhatsapp className="h-3.5 w-3.5" />
                        )}
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("chat.copyLink")}
                      className="text-muted-foreground hover:text-foreground h-6 w-6"
                      onClick={(e) => copyLink(e, session.id)}
                    >
                      {copiedId === session.id ? (
                        <IconCheck className="h-3.5 w-3.5 text-green-500" />
                      ) : (
                        <IconLink className="h-3.5 w-3.5" />
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("chat.deleteSession")}
                      className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive h-6 w-6"
                      onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                        onDeleteSession(session.id)
                      }}
                    >
                      <IconTrash className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                )}
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
