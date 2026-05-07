import type { CSSProperties } from "react"
import wrapAnsi from "wrap-ansi"

export type AnsiSegment = {
  style: CSSProperties
  text: string
}

type AnsiState = {
  background?: string
  bold?: boolean
  dim?: boolean
  foreground?: string
  italic?: boolean
  strikethrough?: boolean
  underline?: boolean
  underlineColor?: string
}

const ANSI_PATTERN = new RegExp(String.raw`\u001B\[([0-9;]*)m`, "g")

const ANSI_COLORS = [
  "#4b5563",
  "#f87171",
  "#4ade80",
  "#facc15",
  "#60a5fa",
  "#c084fc",
  "#22d3ee",
  "#f3f4f6",
]

const ANSI_BRIGHT_COLORS = [
  "#6b7280",
  "#fb7185",
  "#86efac",
  "#fde047",
  "#93c5fd",
  "#e879f9",
  "#67e8f9",
  "#ffffff",
]

function cloneAnsiState(state: AnsiState): AnsiState {
  return { ...state }
}

function ansi256ToHex(code: number): string {
  if (code < 0 || code > 255) {
    return "inherit"
  }

  if (code < 8) {
    return ANSI_COLORS[code]
  }

  if (code < 16) {
    return ANSI_BRIGHT_COLORS[code - 8]
  }

  if (code < 232) {
    const index = code - 16
    const red = Math.floor(index / 36)
    const green = Math.floor((index % 36) / 6)
    const blue = index % 6
    const scale = [0, 95, 135, 175, 215, 255]
    return `rgb(${scale[red]}, ${scale[green]}, ${scale[blue]})`
  }

  const gray = 8 + (code - 232) * 10
  return `rgb(${gray}, ${gray}, ${gray})`
}

function codeToColor(code: number): string | undefined {
  if (code >= 30 && code <= 37) {
    return ANSI_COLORS[code - 30]
  }

  if (code >= 40 && code <= 47) {
    return ANSI_COLORS[code - 40]
  }

  if (code >= 90 && code <= 97) {
    return ANSI_BRIGHT_COLORS[code - 90]
  }

  if (code >= 100 && code <= 107) {
    return ANSI_BRIGHT_COLORS[code - 100]
  }

  if (code === 39 || code === 49) {
    return undefined
  }
}

function applyExtendedColor(
  state: AnsiState,
  codes: number[],
  index: number,
  target: "foreground" | "background" | "underlineColor",
): number {
  const mode = codes[index + 1]

  if (mode === 5) {
    const colorCode = codes[index + 2]
    if (colorCode !== undefined) {
      state[target] = ansi256ToHex(colorCode)
      return index + 2
    }
  }

  if (mode === 2) {
    const red = codes[index + 2]
    const green = codes[index + 3]
    const blue = codes[index + 4]
    if (red !== undefined && green !== undefined && blue !== undefined) {
      state[target] = `rgb(${red}, ${green}, ${blue})`
      return index + 4
    }
  }

  return index
}

function styleToCss(style: AnsiState): CSSProperties {
  return {
    backgroundColor: style.background,
    color: style.foreground,
    fontStyle: style.italic ? "italic" : undefined,
    fontWeight: style.bold ? 700 : undefined,
    opacity: style.dim ? 0.7 : undefined,
    textDecorationColor: style.underlineColor,
    textDecorationLine:
      [
        style.underline ? "underline" : "",
        style.strikethrough ? "line-through" : "",
      ]
        .filter(Boolean)
        .join(" ") || undefined,
  }
}

export function parseAnsiSegments(input: string): AnsiSegment[] {
  const segments: AnsiSegment[] = []
  const state: AnsiState = {}
  let lastIndex = 0
  let match: RegExpExecArray | null

  const pushText = (text: string) => {
    if (!text) {
      return
    }

    segments.push({
      style: styleToCss(cloneAnsiState(state)),
      text,
    })
  }

  ANSI_PATTERN.lastIndex = 0

  while ((match = ANSI_PATTERN.exec(input)) !== null) {
    pushText(input.slice(lastIndex, match.index))

    const codes = (match[1] || "0")
      .split(";")
      .map((value) => (value === "" ? 0 : Number.parseInt(value, 10)))
      .filter((value) => Number.isFinite(value))

    for (let index = 0; index < codes.length; index += 1) {
      const code = codes[index]

      if (code === 0) {
        Object.keys(state).forEach((key) => {
          delete state[key as keyof AnsiState]
        })
        continue
      }

      if (code === 1) {
        state.bold = true
        continue
      }

      if (code === 2) {
        state.dim = true
        continue
      }

      if (code === 3) {
        state.italic = true
        continue
      }

      if (code === 4) {
        state.underline = true
        continue
      }

      if (code === 9) {
        state.strikethrough = true
        continue
      }

      if (code === 21 || code === 22) {
        delete state.bold
        delete state.dim
        continue
      }

      if (code === 23) {
        delete state.italic
        continue
      }

      if (code === 24) {
        delete state.underline
        continue
      }

      if (code === 29) {
        delete state.strikethrough
        continue
      }

      if (code === 39) {
        delete state.foreground
        continue
      }

      if (code === 49) {
        delete state.background
        continue
      }

      if (code === 59) {
        delete state.underlineColor
        continue
      }

      if (code === 38) {
        index = applyExtendedColor(state, codes, index, "foreground")
        continue
      }

      if (code === 48) {
        index = applyExtendedColor(state, codes, index, "background")
        continue
      }

      if (code === 58) {
        index = applyExtendedColor(state, codes, index, "underlineColor")
        continue
      }

      if ((code >= 30 && code <= 37) || (code >= 90 && code <= 97)) {
        state.foreground = codeToColor(code)
        continue
      }

      if ((code >= 40 && code <= 47) || (code >= 100 && code <= 107)) {
        state.background = codeToColor(code)
      }
    }

    lastIndex = ANSI_PATTERN.lastIndex
  }

  pushText(input.slice(lastIndex))

  if (segments.length === 0) {
    return [{ style: {}, text: input }]
  }

  return segments
}

export function wrapLogLine(line: string, columns: number): string {
  const normalized = line.replaceAll("\r\n", "\n").replaceAll("\r", "\n")

  if (columns < 20) {
    return normalized
  }

  return wrapAnsi(normalized, columns, {
    hard: true,
    trim: false,
    wordWrap: false,
  })
}
