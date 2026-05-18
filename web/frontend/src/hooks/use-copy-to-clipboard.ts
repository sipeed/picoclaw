import { useEffect, useRef, useState } from "react"

import { copyText } from "@/lib/clipboard"

const DEFAULT_RESET_DELAY_MS = 2000

export function useCopyToClipboard(
  resetDelayMs: number = DEFAULT_RESET_DELAY_MS,
) {
  const [isCopied, setIsCopied] = useState(false)
  const resetTimerRef = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (resetTimerRef.current !== null) {
        window.clearTimeout(resetTimerRef.current)
      }
    }
  }, [])

  const markCopied = () => {
    if (resetTimerRef.current !== null) {
      window.clearTimeout(resetTimerRef.current)
    }

    setIsCopied(true)
    resetTimerRef.current = window.setTimeout(() => {
      setIsCopied(false)
      resetTimerRef.current = null
    }, resetDelayMs)
  }

  const copy = async (text: string) => {
    const didCopy = await copyText(text)
    if (didCopy) {
      markCopied()
    }
    return didCopy
  }

  return {
    copy,
    isCopied,
  }
}
