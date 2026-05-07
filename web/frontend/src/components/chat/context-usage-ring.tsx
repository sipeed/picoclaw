import { IconArrowRight } from "@tabler/icons-react"
import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ContextUsage } from "@/store/chat"

interface ContextUsageRingProps {
  usage: ContextUsage
  onDetailClick?: () => void
}

function formatTokens(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

export function ContextUsageRing({
  usage,
  onDetailClick,
}: ContextUsageRingProps) {
  const { t } = useTranslation()
  const [intent, setIntent] = useState(false) // user wants open
  const [visible, setVisible] = useState(false) // DOM mounted
  const [animated, setAnimated] = useState(false) // CSS target state
  const [cooldown, setCooldown] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(null)
  const hoverIntent = useRef<ReturnType<typeof setTimeout>>(null)
  const closeTimer = useRef<ReturnType<typeof setTimeout>>(null)

  useEffect(() => {
    if (intent) {
      // Mount first, animate in on next frame
      if (closeTimer.current) clearTimeout(closeTimer.current)
      setVisible(true)
      requestAnimationFrame(() => {
        requestAnimationFrame(() => setAnimated(true))
      })
    } else if (visible) {
      // Animate out, then unmount
      setAnimated(false)
      closeTimer.current = setTimeout(() => setVisible(false), 150)
    }
  }, [intent, visible])

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
      if (hoverIntent.current) clearTimeout(hoverIntent.current)
      if (closeTimer.current) clearTimeout(closeTimer.current)
    }
  }, [])

  const percent = Math.min(usage.used_percent, 100)
  const radius = 8
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (percent / 100) * circumference
  const barPercent = Math.min(percent, 100)

  const handleDetail = () => {
    if (cooldown || !onDetailClick) return
    setCooldown(true)
    onDetailClick()
    setIntent(false)
    timerRef.current = setTimeout(() => setCooldown(false), 1000)
  }

  // Desktop: hover to open, mouse leave to close (with small delay)
  const handleMouseEnter = () => {
    if (hoverIntent.current) clearTimeout(hoverIntent.current)
    setIntent(true)
  }

  const handleMouseLeave = () => {
    hoverIntent.current = setTimeout(() => setIntent(false), 150)
  }

  // Mobile: tap to toggle (preventDefault suppresses synthetic mouseenter)
  const handleTouchStart = (e: React.TouchEvent) => {
    e.preventDefault()
    setIntent((v) => !v)
  }

  return (
    <div
      ref={containerRef}
      className="relative"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <button
        type="button"
        onTouchStart={handleTouchStart}
        className="relative flex h-6 w-6 cursor-pointer items-center justify-center transition-opacity hover:opacity-70"
      >
        <svg className="h-6 w-6 -rotate-90" viewBox="0 0 20 20">
          <circle
            cx="10"
            cy="10"
            r={radius}
            fill="none"
            className="stroke-muted-foreground/30"
            strokeWidth="2"
          />
          <circle
            cx="10"
            cy="10"
            r={radius}
            fill="none"
            className="stroke-muted-foreground"
            strokeWidth="2"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={offset}
          />
        </svg>
        <span className="text-muted-foreground absolute text-[8px] font-medium tabular-nums">
          {percent}
        </span>
      </button>

      {visible && (
        <div
          className={`bg-popover text-popover-foreground absolute right-0 bottom-full z-50 mb-3 w-[220px] rounded-xl border p-4 shadow-lg transition-all duration-150 ${
            animated
              ? "scale-100 opacity-100"
              : "pointer-events-none scale-95 opacity-0"
          }`}
        >
          <div className="bg-popover absolute -bottom-1.5 right-3 h-3 w-3 rotate-45 border-r border-b" />

          <div className="flex items-center justify-between">
            <span className="text-muted-foreground text-xs">
              {t("chat.contextTitle")}
            </span>
            <span className="text-xs font-medium">
              {formatTokens(usage.used_tokens)} /{" "}
              {formatTokens(usage.compress_at_tokens)}
            </span>
          </div>
          <div className="bg-muted mt-1.5 h-1.5 w-full overflow-hidden rounded-full">
            <div
              className="h-full rounded-full bg-violet-500 transition-all"
              style={{ width: `${barPercent}%` }}
            />
          </div>

          <button
            type="button"
            onClick={handleDetail}
            disabled={cooldown}
            className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-violet-600 transition-opacity hover:opacity-70 disabled:opacity-40 dark:text-violet-400"
          >
            {t("chat.contextDetail")}
            <IconArrowRight className="h-3 w-3" />
          </button>
        </div>
      )}
    </div>
  )
}
