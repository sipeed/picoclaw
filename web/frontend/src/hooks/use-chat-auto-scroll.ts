import {
  type UIEvent,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react"

const BOTTOM_FOLLOW_THRESHOLD = 32

function getBottomDistance(element: HTMLDivElement) {
  return element.scrollHeight - element.scrollTop - element.clientHeight
}

function getBottomScrollTop(element: HTMLDivElement) {
  return Math.max(0, element.scrollHeight - element.clientHeight)
}

function nextScrollStep(distance: number, streaming: boolean) {
  const magnitude = Math.abs(distance)
  const factor = streaming ? 0.34 : 0.22
  const min = streaming ? 12 : 10
  const max = streaming ? 72 : 56
  return Math.sign(distance) * Math.min(max, Math.max(min, magnitude * factor))
}

interface UseChatAutoScrollOptions {
  deps: readonly unknown[]
  streaming: boolean
}

export function useChatAutoScroll({
  deps,
  streaming,
}: UseChatAutoScrollOptions) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const stickyRef = useRef(true)
  const targetScrollTopRef = useRef(0)
  const animationFrameRef = useRef<number | null>(null)
  const programmaticScrollRef = useRef(false)

  const [isAtBottom, setIsAtBottom] = useState(true)
  const [hasScrolled, setHasScrolled] = useState(false)

  const cancelAnimation = useCallback(() => {
    if (animationFrameRef.current !== null) {
      window.cancelAnimationFrame(animationFrameRef.current)
      animationFrameRef.current = null
    }
    programmaticScrollRef.current = false
  }, [])

  const syncStateFromElement = useCallback(
    (element: HTMLDivElement, options?: { programmatic?: boolean }) => {
      const distanceToBottom = getBottomDistance(element)
      setHasScrolled(element.scrollTop > 0)

      if (options?.programmatic && stickyRef.current) {
        setIsAtBottom(true)
        return
      }

      const nextIsAtBottom = distanceToBottom <= BOTTOM_FOLLOW_THRESHOLD
      stickyRef.current = nextIsAtBottom
      setIsAtBottom(nextIsAtBottom)
      if (!nextIsAtBottom) {
        cancelAnimation()
      }
    },
    [cancelAnimation],
  )

  const animateToBottom = useCallback(() => {
    const element = scrollRef.current
    if (!element || !stickyRef.current) {
      return
    }

    targetScrollTopRef.current = getBottomScrollTop(element)

    if (animationFrameRef.current !== null) {
      return
    }

    const step = () => {
      const currentElement = scrollRef.current
      if (!currentElement || !stickyRef.current) {
        cancelAnimation()
        return
      }

      const target = Math.max(
        targetScrollTopRef.current,
        getBottomScrollTop(currentElement),
      )
      targetScrollTopRef.current = target

      const distance = target - currentElement.scrollTop
      if (Math.abs(distance) <= 1) {
        currentElement.scrollTop = target
        animationFrameRef.current = null
        programmaticScrollRef.current = false
        syncStateFromElement(currentElement, { programmatic: true })
        return
      }

      programmaticScrollRef.current = true
      currentElement.scrollTop += nextScrollStep(distance, streaming)
      syncStateFromElement(currentElement, { programmatic: true })
      animationFrameRef.current = window.requestAnimationFrame(step)
    }

    animationFrameRef.current = window.requestAnimationFrame(step)
  }, [cancelAnimation, streaming, syncStateFromElement])

  const handleScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      const element = event.currentTarget
      const isProgrammatic = programmaticScrollRef.current
      if (!isProgrammatic) {
        const nextIsAtBottom =
          getBottomDistance(element) <= BOTTOM_FOLLOW_THRESHOLD
        stickyRef.current = nextIsAtBottom
      }
      syncStateFromElement(element, { programmatic: isProgrammatic })
    },
    [syncStateFromElement],
  )

  const handleManualScrollIntent = useCallback(() => {
    if (!stickyRef.current) {
      return
    }
    stickyRef.current = false
    setIsAtBottom(false)
    cancelAnimation()
  }, [cancelAnimation])

  const scrollToBottom = useCallback(
    (options?: { immediate?: boolean }) => {
      const element = scrollRef.current
      if (!element) {
        return
      }

      stickyRef.current = true
      setIsAtBottom(true)
      targetScrollTopRef.current = getBottomScrollTop(element)

      if (options?.immediate) {
        cancelAnimation()
        element.scrollTop = targetScrollTopRef.current
        syncStateFromElement(element, { programmatic: true })
        return
      }

      animateToBottom()
    },
    [animateToBottom, cancelAnimation, syncStateFromElement],
  )

  useLayoutEffect(() => {
    const element = scrollRef.current
    if (!element) {
      return
    }

    if (!hasScrolled && element.scrollHeight > 0) {
      element.scrollTop = getBottomScrollTop(element)
      stickyRef.current = true
      setIsAtBottom(true)
      setHasScrolled(element.scrollTop > 0)
      return
    }

    if (stickyRef.current) {
      animateToBottom()
    }
  }, [animateToBottom, hasScrolled, ...deps])

  useEffect(() => cancelAnimation, [cancelAnimation])

  return {
    scrollRef,
    isAtBottom,
    hasScrolled,
    handleScroll,
    handleManualScrollIntent,
    scrollToBottom,
  }
}
