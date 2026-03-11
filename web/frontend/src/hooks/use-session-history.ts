import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { type SessionSummary, deleteSession, getSessions } from "@/api/sessions"

const LIMIT = 20

interface UseSessionHistoryOptions {
  activeSessionId: string
  onDeletedActiveSession: () => void
}

export function useSessionHistory({
  activeSessionId,
  onDeletedActiveSession,
}: UseSessionHistoryOptions) {
  const { t } = useTranslation()
  const observerRef = useRef<HTMLDivElement>(null)
  const [sessions, setSessions] = useState<SessionSummary[]>([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  const [loadError, setLoadError] = useState(false)

  const loadSessions = useCallback(
    async (reset = true) => {
      try {
        const currentOffset = reset ? 0 : offset
        if (reset) {
          setLoadError(false)
          setHasMore(true)
          setOffset(0)
        }

        const data = await getSessions(currentOffset, LIMIT)
        setLoadError(false)

        if (data.length < LIMIT) {
          setHasMore(false)
        }

        if (reset) {
          setSessions(data)
        } else {
          setSessions((prev) => {
            const existingIds = new Set(prev.map((s) => s.id))
            const newItems = data.filter((s) => !existingIds.has(s.id))
            return [...prev, ...newItems]
          })
        }

        setOffset(currentOffset + data.length)
      } catch (err) {
        console.error("Failed to fetch session history:", err)
        setLoadError(true)
        if (!reset) {
          setHasMore(false)
        }
      } finally {
        setIsLoadingMore(false)
      }
    },
    [offset],
  )

  useEffect(() => {
    if (!observerRef.current || !hasMore || isLoadingMore || loadError) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (
          entries[0].isIntersecting &&
          hasMore &&
          !isLoadingMore &&
          !loadError
        ) {
          setIsLoadingMore(true)
          void loadSessions(false)
        }
      },
      { threshold: 0.1 },
    )

    observer.observe(observerRef.current)
    return () => observer.disconnect()
  }, [hasMore, isLoadingMore, loadError, loadSessions])

  const handleDeleteSession = useCallback(
    async (id: string) => {
      try {
        await deleteSession(id)
        setSessions((prev) => prev.filter((s) => s.id !== id))
        if (id === activeSessionId) {
          onDeletedActiveSession()
        }
      } catch (err) {
        console.error("Failed to delete session:", err)
      }
    },
    [activeSessionId, onDeletedActiveSession],
  )

  return {
    sessions,
    hasMore,
    loadError,
    loadErrorMessage: t("chat.historyLoadFailed"),
    observerRef,
    loadSessions,
    handleDeleteSession,
  }
}
