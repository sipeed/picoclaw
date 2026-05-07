import type { ReactNode } from "react"

import { useHighlightTheme } from "./hooks/use-highlight-theme"

interface AppProvidersProps {
  children: ReactNode
}

export function AppProviders({ children }: AppProvidersProps) {
  useHighlightTheme()

  return <>{children}</>
}
