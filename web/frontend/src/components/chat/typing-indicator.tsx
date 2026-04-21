import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

export function TypingIndicator() {
  const { t } = useTranslation()
  const thinkingSteps = [
    t("chat.thinking.step1"),
    t("chat.thinking.step2"),
    t("chat.thinking.step3"),
    t("chat.thinking.step4"),
  ]
  const [stepIndex, setStepIndex] = useState(0)

  useEffect(() => {
    const stepsCount = thinkingSteps.length
    const interval = setInterval(() => {
      setStepIndex((prev) => (prev + 1) % stepsCount)
    }, 3000)
    return () => clearInterval(interval)
  }, [thinkingSteps.length])

  return (
    <div className="flex w-full max-w-[820px] gap-3">
      <div className="bg-muted text-muted-foreground mt-5 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-border/70 text-[11px] font-semibold uppercase">
        AI
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="text-muted-foreground flex items-center gap-2 text-[11px] uppercase tracking-[0.14em]">
          <span>PicoClaw</span>
          <span className="rounded-full border border-border/70 px-2 py-0.5 text-[10px] tracking-normal normal-case">
            Thinking
          </span>
        </div>
        <div className="bg-card inline-flex w-fit min-w-56 max-w-md flex-col gap-3 rounded-xl border border-border/70 px-4 py-3 shadow-sm">
          <div className="flex items-center gap-1.5">
            <span className="size-2 animate-bounce rounded-full bg-foreground/55 [animation-delay:-0.3s]" />
            <span className="size-2 animate-bounce rounded-full bg-foreground/55 [animation-delay:-0.15s]" />
            <span className="size-2 animate-bounce rounded-full bg-foreground/55" />
          </div>

          <div className="bg-muted relative h-1 w-40 overflow-hidden rounded-full">
            <div className="absolute inset-0 animate-[shimmer_2s_infinite] rounded-full bg-gradient-to-r from-foreground/35 via-foreground/65 to-foreground/35 bg-[length:200%_100%]" />
          </div>

          <p
            key={stepIndex}
            className="text-muted-foreground animate-[fadeSlideIn_0.4s_ease-out] text-xs leading-5"
          >
            {thinkingSteps[stepIndex]}
          </p>
        </div>
      </div>
    </div>
  )
}
