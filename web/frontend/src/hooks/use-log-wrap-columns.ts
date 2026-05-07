import { useEffect, useRef, useState } from "react"

const DEFAULT_WRAP_COLUMNS = 120
const MIN_WRAP_COLUMNS = 20

export function useLogWrapColumns() {
  const [wrapColumns, setWrapColumns] = useState(DEFAULT_WRAP_COLUMNS)
  const contentRef = useRef<HTMLDivElement>(null)
  const measureRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    const content = contentRef.current
    const measure = measureRef.current

    if (!content || !measure) {
      return
    }

    const updateWrapColumns = () => {
      const contentWidth = content.clientWidth
      const charWidth = measure.getBoundingClientRect().width

      if (!contentWidth || !charWidth) {
        return
      }

      const nextColumns = Math.max(
        Math.floor(contentWidth / charWidth) - 1,
        MIN_WRAP_COLUMNS,
      )

      setWrapColumns((current) =>
        current === nextColumns ? current : nextColumns,
      )
    }

    updateWrapColumns()

    const observer = new ResizeObserver(updateWrapColumns)
    observer.observe(content)

    return () => {
      observer.disconnect()
    }
  }, [])

  return {
    contentRef,
    measureRef,
    wrapColumns,
  }
}
