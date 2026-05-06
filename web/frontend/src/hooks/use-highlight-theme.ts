import githubDarkCss from "highlight.js/styles/github-dark.css?inline"
import githubLightCss from "highlight.js/styles/github.css?inline"
import { useEffect } from "react"

const THEME_STYLE_ID = "hljs-theme-style"
const THEME_STYLE_OWNER_ATTR = "data-picoclaw-highlight-theme"
const THEME_STYLE_OWNER_VALUE = "true"
const MANAGED_THEME_STYLE_SELECTOR = `style[${THEME_STYLE_OWNER_ATTR}="${THEME_STYLE_OWNER_VALUE}"]`
const ID_THEME_STYLE_SELECTOR = `style#${THEME_STYLE_ID}`

function getOrCreateThemeStyleElement(): HTMLStyleElement {
  const managedStyleElement = document.head.querySelector<HTMLStyleElement>(
    MANAGED_THEME_STYLE_SELECTOR,
  )
  if (managedStyleElement) {
    return managedStyleElement
  }

  const existingStyleElement = document.querySelector<HTMLStyleElement>(
    ID_THEME_STYLE_SELECTOR,
  )
  if (existingStyleElement) {
    existingStyleElement.setAttribute(
      THEME_STYLE_OWNER_ATTR,
      THEME_STYLE_OWNER_VALUE,
    )
    return existingStyleElement
  }

  const conflictingElement = document.getElementById(THEME_STYLE_ID)
  const styleElement = document.createElement("style")
  if (!conflictingElement) {
    styleElement.id = THEME_STYLE_ID
  }

  // Leave conflicting non-style nodes untouched and track the injected style explicitly.
  styleElement.setAttribute(THEME_STYLE_OWNER_ATTR, THEME_STYLE_OWNER_VALUE)
  document.head.appendChild(styleElement)

  return styleElement
}

export function useHighlightTheme() {
  useEffect(() => {
    const root = document.documentElement
    const styleElement = getOrCreateThemeStyleElement()

    const applyTheme = () => {
      const nextThemeCss = root.classList.contains("dark")
        ? githubDarkCss
        : githubLightCss
      styleElement.textContent = nextThemeCss
    }

    applyTheme()

    const observer = new MutationObserver(() => {
      applyTheme()
    })

    observer.observe(root, {
      attributes: true,
      attributeFilter: ["class"],
    })

    return () => {
      observer.disconnect()
    }
  }, [])
}
