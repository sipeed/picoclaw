import type {
  ChatActionItem,
  ChatCardBlock,
  ChatFieldItem,
  ChatFormField,
  ChatListItem,
  ChatProgressStep,
  ChatTodoItem,
  ChatStructuredContent,
  ChatStructuredValue,
} from "@/store/chat"

const planHeadingRe = /^\s*#{1,6}\s+(.+?)\s*$/
const planCheckboxRe = /^\s*(?:[-*+]|\d+[.)])\s*\[\s*(x|X)?\s*\]\s*(.+?)\s*$/
const planBulletRe = /^\s*[-*+]\s+(.+?)\s*$/
const planNumberRe = /^\s*\d+[.)]\s+(.+?)\s*$/

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined
}

function firstString(record: Record<string, unknown>, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = asString(record[key])
    if (value) {
      return value
    }
  }
  return undefined
}

function inferStructuredType(record: Record<string, unknown>): string | undefined {
  const explicitType = asString(record.type)
  if (explicitType) {
    return explicitType
  }
  if (Array.isArray(record.options)) {
    return "options"
  }
  if (Array.isArray(record.items)) {
    return "todo"
  }
  if (Array.isArray(record.steps) || record.progress !== undefined) {
    return "progress"
  }
  if (firstString(record, ["level", "severity", "statusLevel"])) {
    return "alert"
  }
  if (Array.isArray(record.fields)) {
    return "form"
  }
  if (Array.isArray(record.blocks) || Array.isArray(record.actions)) {
    return "card"
  }
  return undefined
}

function inferOptionsMode(record: Record<string, unknown>): "single" | "multiple" {
  const explicitMode = firstString(record, [
    "mode",
    "selectionMode",
    "selection_mode",
  ])
  if (explicitMode === "multiple") {
    return "multiple"
  }
  if (
    asBoolean(record.multiple) === true ||
    asBoolean(record.multi) === true ||
    asBoolean(record.multiSelect) === true ||
    asBoolean(record.multi_select) === true
  ) {
    return "multiple"
  }
  return "single"
}

function parseActionItem(value: unknown): ChatActionItem | null {
  const record = asRecord(value)
  if (!record) {
    return null
  }

  const label = asString(record.label)
  if (!label) {
    return null
  }

  const variant = asString(record.variant)
  return {
    label,
    type: asString(record.type),
    action: asString(record.action),
    value: asString(record.value),
    url: asString(record.url),
    variant:
      variant === "default" ||
      variant === "outline" ||
      variant === "secondary" ||
      variant === "ghost"
        ? variant
        : undefined,
  }
}

function parseFieldItem(value: unknown): ChatFieldItem | null {
  const record = asRecord(value)
  if (!record) {
    return null
  }
  const label = asString(record.label)
  const fieldValue = asString(record.value)
  if (!label || !fieldValue) {
    return null
  }
  return { label, value: fieldValue }
}

function parseListItem(value: unknown): ChatListItem | null {
  if (typeof value === "string" && value.trim()) {
    return { text: value }
  }
  const record = asRecord(value)
  if (!record) {
    return null
  }
  const text = asString(record.text)
  if (!text) {
    return null
  }
  return { text, label: asString(record.label) }
}

function parseProgressStep(value: unknown): ChatProgressStep | null {
  const record = asRecord(value)
  if (!record) {
    return null
  }
  const label = asString(record.label)
  if (!label) {
    return null
  }
  return {
    label,
    status: asString(record.status),
    detail: asString(record.detail),
  }
}

function parseTodoItem(value: unknown): ChatTodoItem | null {
  const record = asRecord(value)
  if (!record) {
    return null
  }

  const title = firstString(record, ["title", "label", "text", "step"])
  if (!title) {
    return null
  }

  const status = asString(record.status)
  return {
    id: asString(record.id),
    title,
    detail: firstString(record, ["detail", "description", "message"]),
    status:
      status === "not-started" ||
      status === "in-progress" ||
      status === "completed"
        ? status
        : undefined,
  }
}

function parseFormField(value: unknown): ChatFormField | null {
  const record = asRecord(value)
  if (!record) {
    return null
  }

  const name = asString(record.name)
  const label = asString(record.label)
  if (!name || !label) {
    return null
  }

  return {
    name,
    label,
    fieldType: asString(record.fieldType) ?? asString(record.type),
    value: asString(record.value),
    placeholder: asString(record.placeholder),
    required: record.required === true,
  }
}

function parseCardBlock(value: unknown): ChatCardBlock | null {
  const record = asRecord(value)
  if (!record) {
    return null
  }

  const type = asString(record.type)
  if (!type) {
    return null
  }

  switch (type) {
    case "text":
    case "markdown": {
      const text = asString(record.text)
      return text ? { type, text } : null
    }
    case "fields": {
      const items = Array.isArray(record.fields)
        ? record.fields
            .map((item) => parseFieldItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : []
      return items.length > 0 ? { type, fields: items } : null
    }
    case "badge": {
      const label = asString(record.label)
      return label ? { type, label, status: asString(record.status) } : null
    }
    case "actions": {
      const actions = Array.isArray(record.actions)
        ? record.actions
            .map((item) => parseActionItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : []
      return actions.length > 0 ? { type, actions } : null
    }
    case "list": {
      const items = Array.isArray(record.items)
        ? record.items
            .map((item) => parseListItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : []
      return items.length > 0 ? { type, items } : null
    }
    case "table": {
      const rows = Array.isArray(record.rows)
        ? record.rows
            .map((row) =>
              Array.isArray(row)
                ? row
                    .map((cell) => (typeof cell === "string" ? cell : String(cell)))
                    .filter((cell) => cell.length > 0)
                : [],
            )
            .filter((row) => row.length > 0)
        : []
      if (rows.length === 0) {
        return null
      }
      const headers = Array.isArray(record.headers)
        ? record.headers.map((item) => String(item))
        : undefined
      return { type, headers, rows }
    }
    case "image": {
      const url = asString(record.url)
      return url ? { type, url, alt: asString(record.alt) } : null
    }
    case "divider":
      return { type }
    case "json":
      return { type, data: record.data }
    default:
      return { type: "unknown", blockType: type, raw: record }
  }
}

function parseSingleStructuredContent(
  structured: unknown,
): ChatStructuredContent | undefined {
  const record = asRecord(structured)
  if (!record) {
    return undefined
  }

  const type = inferStructuredType(record)
  if (!type) {
    return undefined
  }

  switch (type) {
    case "options": {
      const options = Array.isArray(record.options)
        ? record.options
            .map((item) => {
              const entry = asRecord(item)
              if (!entry) {
                return null
              }
              const label = asString(entry.label)
              const value = asString(entry.value)
              if (!label || !value) {
                return null
              }
              return {
                label,
                value,
                description: asString(entry.description),
              }
            })
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : []
      return options.length > 0
        ? {
            type,
            options,
            mode: inferOptionsMode(record),
            allowCustom:
              asBoolean(record.allowCustom) ??
              asBoolean(record.allow_custom) ??
              asBoolean(record.customInputEnabled) ??
              asBoolean(record.custom_input_enabled) ??
              false,
            customPlaceholder:
              firstString(record, [
                "customPlaceholder",
                "custom_placeholder",
                "inputPlaceholder",
                "input_placeholder",
              ]),
            submitLabel:
              firstString(record, [
                "submitLabel",
                "submit_label",
                "buttonLabel",
                "button_label",
              ]),
            raw: record,
          }
        : undefined
    }
    case "card": {
      const blocks = Array.isArray(record.blocks)
        ? record.blocks
            .map((item) => parseCardBlock(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      const actions = Array.isArray(record.actions)
        ? record.actions
            .map((item) => parseActionItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      return {
        type,
        kind: asString(record.kind),
        version: asString(record.version),
        title: asString(record.title),
        blocks,
        actions,
        raw: record,
      }
    }
    case "form": {
      const fields = Array.isArray(record.fields)
        ? record.fields
            .map((item) => parseFormField(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      const actions = Array.isArray(record.actions)
        ? record.actions
            .map((item) => parseActionItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      return {
        type,
        kind: asString(record.kind),
        version: asString(record.version),
        title: asString(record.title),
        content: firstString(record, ["content", "description", "message"]),
        fields,
        actions,
        raw: record,
      }
    }
    case "progress": {
      const steps = Array.isArray(record.steps)
        ? record.steps
            .map((item) => parseProgressStep(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      return {
        type,
        kind: asString(record.kind),
        version: asString(record.version),
        title: asString(record.title),
        content: firstString(record, ["content", "description", "message", "detail"]),
        status: asString(record.status),
        steps,
        raw: record,
      }
    }
    case "todo": {
      const items = Array.isArray(record.items)
        ? record.items
            .map((item) => parseTodoItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      return {
        type,
        kind: asString(record.kind),
        version: asString(record.version),
        title: asString(record.title),
        content: firstString(record, ["content", "description", "message"]),
        items,
        raw: record,
      }
    }
    case "alert": {
      const actions = Array.isArray(record.actions)
        ? record.actions
            .map((item) => parseActionItem(item))
            .filter((item): item is NonNullable<typeof item> => item !== null)
        : undefined
      return {
        type,
        kind: asString(record.kind),
        version: asString(record.version),
        title: asString(record.title),
        level: firstString(record, ["level", "severity", "statusLevel"]),
        content: firstString(record, ["content", "description", "message", "detail"]),
        actions,
        raw: record,
      }
    }
    default:
      return {
        type: "unknown",
        kind: asString(record.kind) ?? type,
        raw: record,
      }
  }
}

export function parseStructuredContent(
  structured: unknown,
): ChatStructuredValue | undefined {
  if (Array.isArray(structured)) {
    const parts = structured
      .map((item) => parseSingleStructuredContent(item))
      .filter((item): item is ChatStructuredContent => item !== undefined)
    return parts.length > 0 ? parts : undefined
  }

  return parseSingleStructuredContent(structured)
}

function cleanPlanText(value: string): string {
  return value
    .trim()
    .replace(/^[:\-\s]+/, "")
    .replace(/[*_`#]/g, "")
    .trim()
}

function looksLikePlanLine(line: string): boolean {
  return (
    planCheckboxRe.test(line) ||
    planBulletRe.test(line) ||
    planNumberRe.test(line)
  )
}

function isPlanHeading(text: string): boolean {
  return /规划|计划|plan|阶段|任务|step|phase|milestone|测试|验证|优化/i.test(text)
}

function inferPlanStatus(text: string): ChatTodoItem["status"] {
  if (/completed|done|已完成/i.test(text)) {
    return "completed"
  }
  if (/in-progress|running|进行中/i.test(text)) {
    return "in-progress"
  }
  return "not-started"
}

export function inferStructuredContentFromText(
  content: string,
): ChatStructuredValue | undefined {
  const normalized = content
    .replace(/\r\n/g, "\n")
    .replace(/[\u00A0\u3000]/g, " ")
    .trim()
  if (!normalized) {
    return undefined
  }

  const lines = normalized.split("\n")
  let title: string | undefined
  let summary: string | undefined
  let hasChecklist = false
  const headingItems: ChatTodoItem[] = []
  const listItems: ChatTodoItem[] = []

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line || line === "---") {
      continue
    }

    const headingMatch = line.match(planHeadingRe)
    if (headingMatch) {
      const headingText = cleanPlanText(headingMatch[1] ?? "")
      if (!headingText) {
        continue
      }
      if (!title) {
        title = headingText
        continue
      }
      if (isPlanHeading(headingText)) {
        headingItems.push({
          title: headingText,
          status: "not-started",
        })
      }
      continue
    }

    if (!summary && !looksLikePlanLine(line)) {
      const candidate = cleanPlanText(line)
      if (candidate) {
        summary = candidate
      }
    }

    const checkboxMatch = line.match(planCheckboxRe)
    if (checkboxMatch) {
      hasChecklist = true
      listItems.push({
        title: cleanPlanText(checkboxMatch[2] ?? ""),
        status: checkboxMatch[1]?.toLowerCase() === "x" ? "completed" : "not-started",
      })
      continue
    }

    const bulletMatch = line.match(planBulletRe)
    if (bulletMatch) {
      const itemText = cleanPlanText(bulletMatch[1] ?? "")
      if (itemText) {
        listItems.push({ title: itemText, status: inferPlanStatus(itemText) })
      }
      continue
    }

    const numberMatch = line.match(planNumberRe)
    if (numberMatch) {
      const itemText = cleanPlanText(numberMatch[1] ?? "")
      if (itemText) {
        listItems.push({ title: itemText, status: inferPlanStatus(itemText) })
      }
    }
  }

  const items = headingItems.length > 0 ? headingItems : listItems
  if (items.length < 2) {
    return undefined
  }

  const signalText = `${title ?? ""}\n${normalized}`
  if (!hasChecklist && !/规划|计划|plan|阶段|任务|todo|step|phase/i.test(signalText)) {
    return undefined
  }

  if (!items.some((item) => item.status === "completed" || item.status === "in-progress")) {
    items[0] = { ...items[0], status: "in-progress" }
  }

  return {
    type: "todo",
    title: title ?? "Plan",
    content: summary,
    items: items.slice(0, 8),
  }
}