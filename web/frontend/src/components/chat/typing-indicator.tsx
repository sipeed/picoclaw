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
    <div className="flex w-full flex-col gap-1.5">
      <div className="text-muted-foreground flex items-center gap-2 px-1 text-xs opacity-70">
        <span>PicoClaw</span>
      </div>
      <div className="bg-card inline-flex w-fit max-w-xs flex-col gap-3 rounded-xl border px-5 py-4">
        <div className="flex items-center gap-1.5">
          <span className="size-2 animate-bounce rounded-full bg-violet-400/70 [animation-delay:-0.3s]" />
          <span className="size-2 animate-bounce rounded-full bg-violet-400/70 [animation-delay:-0.15s]" />
          <span className="size-2 animate-bounce rounded-full bg-violet-400/70" />
        </div>

        <div className="bg-muted relative h-1 w-36 overflow-hidden rounded-full">
          <div className="absolute inset-0 animate-[shimmer_2s_infinite] rounded-full bg-gradient-to-r from-violet-500/60 via-violet-400/80 to-violet-500/60 bg-[length:200%_100%]" />
        </div>

        <p
          key={stepIndex}
          className="text-muted-foreground animate-[fadeSlideIn_0.4s_ease-out] text-xs"
        >
          {thinkingSteps[stepIndex]}
        </p>
      </div>
    </div>
  )
}
