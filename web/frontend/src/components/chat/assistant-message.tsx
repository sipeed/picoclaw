import { IconAlertCircle, IconBrain, IconCheck, IconChevronDown, IconChevronRight, IconClockHour4, IconCopy, IconLoader2 } from "@tabler/icons-react"
import { type ReactNode, useState } from "react"
import { useTranslation } from "react-i18next"
import ReactMarkdown from "react-markdown"
import rehypeHighlight from "rehype-highlight"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { inferStructuredContentFromText } from "@/features/chat/structured"
import { formatMessageTime } from "@/hooks/use-pico-chat"
import { cn } from "@/lib/utils"
import type {
  ChatActionItem,
  ChatCardBlock,
  ChatStructuredAlert,
  ChatStructuredProgress,
  ChatStructuredCard,
  ChatTodoStatus,
  ChatTodoItem,
  ChatStructuredForm,
  ChatStructuredOptions,
  ChatStructuredTodo,
  ChatUnknownBlock,
  ChatStructuredContent,
  ChatStructuredValue,
} from "@/store/chat"

interface AssistantMessageProps {
  content: string
  isThought?: boolean
  timestamp?: string | number
  structured?: ChatStructuredValue
  onSelectOption?: (value: string) => void
}

const STRUCTURED_PANEL_CLASS =
  "space-y-3 rounded-xl border border-border/60 bg-muted/20 p-4 shadow-sm"

const STRUCTURED_SUBSECTION_CLASS =
  "rounded-lg border border-border/60 bg-background/70 p-3"

const MARKDOWN_BODY_CLASS =
  "prose dark:prose-invert prose-pre:my-2 prose-pre:overflow-x-auto prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-100 prose-pre:p-0 dark:prose-pre:bg-zinc-950 max-w-none [overflow-wrap:anywhere] break-words"

const MESSAGE_CONTENT_CLASS =
  "min-w-0 flex-1 space-y-2"

function renderMarkdown(text: string) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeRaw, rehypeSanitize, rehypeHighlight]}
    >
      {text}
    </ReactMarkdown>
  )
}

function renderStructuredText(text?: string) {
  if (!text?.trim()) {
    return null
  }

  return (
    <div className="prose dark:prose-invert max-w-none text-sm [overflow-wrap:anywhere] break-words">
      {renderMarkdown(text)}
    </div>
  )
}

function StructuredMetaPill({
  children,
  tone = "default",
}: {
  children: ReactNode
  tone?: "default" | "muted" | "accent"
}) {
  return (
    <span
      className={cn(
        "rounded-full border px-2 py-0.5 text-[11px] font-medium",
        tone === "accent"
          ? "border-foreground/10 bg-foreground/5"
          : tone === "muted"
            ? "text-muted-foreground border-border/70"
            : "border-border/70",
      )}
    >
      {children}
    </span>
  )
}

function StructuredHeader({
  title,
  kind,
  badge,
  eyebrow,
}: {
  title?: string
  kind?: string
  badge?: string
  eyebrow?: string
}) {
  if (!title && !kind && !badge && !eyebrow) {
    return null
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        {eyebrow && (
          <StructuredMetaPill tone="accent">{eyebrow}</StructuredMetaPill>
        )}
        {badge && <StructuredMetaPill>{badge}</StructuredMetaPill>}
        {kind && <StructuredMetaPill tone="muted">{kind}</StructuredMetaPill>}
      </div>
      {title && <div className="text-sm font-semibold tracking-tight">{title}</div>}
    </div>
  )
}

function actionVariant(action: ChatActionItem) {
  return action.variant ?? (action.url ? "outline" : "secondary")
}

function renderStructuredRawFallback(
  raw: Record<string, unknown> | undefined,
  summary: string,
) {
  if (!raw) {
    return null
  }

  return (
    <details className="rounded-xl border border-dashed border-border/70 bg-muted/20 p-3 shadow-sm">
      <summary className="cursor-pointer text-sm font-medium">{summary}</summary>
      <pre className="mt-2 overflow-x-auto text-xs">{JSON.stringify(raw, null, 2)}</pre>
    </details>
  )
}

function StructuredActions({
  actions,
  onSelectOption,
}: {
  actions: ChatActionItem[]
  onSelectOption?: (value: string) => void
}) {
  if (actions.length === 0) {
    return null
  }

  return (
    <div className="flex flex-wrap gap-2">
      {actions.map((action) => {
        const key = `${action.label}-${action.value ?? action.url ?? action.action ?? "action"}`
        const commonClassName =
          "h-auto max-w-full items-start justify-start whitespace-normal text-left"

        if (action.url) {
          return (
            <Button
              key={key}
              asChild
              type="button"
              variant={actionVariant(action)}
              size="sm"
              className={commonClassName}
            >
              <a href={action.url} target="_blank" rel="noreferrer">
                {action.label}
              </a>
            </Button>
          )
        }

        return (
          <Button
            key={key}
            type="button"
            variant={actionVariant(action)}
            size="sm"
            className={commonClassName}
            onClick={() => onSelectOption?.(action.value ?? action.label)}
          >
            {action.label}
          </Button>
        )
      })}
    </div>
  )
}

function StructuredCardBlockView({
  block,
  onSelectOption,
}: {
  block: ChatCardBlock
  onSelectOption?: (value: string) => void
}) {
  switch (block.type) {
    case "text": {
      const textBlock = block
      return <p className="text-sm leading-6 whitespace-pre-wrap">{textBlock.text}</p>
    }
    case "markdown": {
      const markdownBlock = block
      return <div className="prose dark:prose-invert max-w-none text-sm">{renderMarkdown(markdownBlock.text)}</div>
    }
    case "fields":
      {
        const fieldsBlock = block
      return (
        <div className="grid gap-2 sm:grid-cols-2">
          {fieldsBlock.fields.map((field) => (
            <div key={`${field.label}-${field.value}`} className={STRUCTURED_SUBSECTION_CLASS}>
              <div className="text-muted-foreground text-xs">{field.label}</div>
              <div className="mt-1 text-sm font-medium break-words">{field.value}</div>
            </div>
          ))}
        </div>
      )
      }
    case "badge":
      {
        const badgeBlock = block
      return (
        <span className="inline-flex rounded-full border border-border/70 bg-background/70 px-2.5 py-1 text-xs font-medium">
          {badgeBlock.label}
        </span>
      )
      }
    case "actions": {
      const actionsBlock = block
      return <StructuredActions actions={actionsBlock.actions} onSelectOption={onSelectOption} />
    }
    case "list":
      {
        const listBlock = block
      return (
        <ul className="list-disc space-y-1 pl-5 text-sm">
          {listBlock.items.map((item, index) => (
            <li key={`${item.label ?? item.text}-${index}`}>
              {item.label ? `${item.label}: ${item.text}` : item.text}
            </li>
          ))}
        </ul>
      )
      }
    case "table":
      {
        const tableBlock = block
      return (
        <div className="overflow-x-auto rounded-xl border border-border/60 bg-background/70">
          <table className="w-full min-w-80 text-sm">
            {tableBlock.headers && tableBlock.headers.length > 0 && (
              <thead className="bg-muted/60">
                <tr>
                  {tableBlock.headers.map((header, index) => (
                    <th key={`${header}-${index}`} className="px-3 py-2 text-left font-medium">
                      {header}
                    </th>
                  ))}
                </tr>
              </thead>
            )}
            <tbody>
              {tableBlock.rows.map((row, rowIndex) => (
                <tr key={`row-${rowIndex}`} className="border-t border-border/60">
                  {row.map((cell, cellIndex) => (
                    <td key={`cell-${rowIndex}-${cellIndex}`} className="px-3 py-2 align-top break-words">
                      {cell}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )
      }
    case "image":
      {
        const imageBlock = block
      return (
        <img
          src={imageBlock.url}
          alt={imageBlock.alt ?? "structured image"}
          className="max-h-80 rounded-xl border border-border/60 bg-background/70 object-contain"
        />
      )
      }
    case "divider":
      return <div className="border-t border-border/60" />
    case "json": {
      const jsonBlock = block
      return (
        <details className="rounded-xl border border-border/60 bg-background/60 p-3 shadow-sm">
          <summary className="cursor-pointer text-sm font-medium">JSON</summary>
          <pre className="mt-2 overflow-x-auto text-xs">{JSON.stringify(jsonBlock.data, null, 2)}</pre>
        </details>
      )
    }
    default: {
      const unknownBlock = block as ChatUnknownBlock
      return (
        <details className="rounded-xl border border-dashed border-border/70 bg-muted/20 p-3 shadow-sm">
          <summary className="cursor-pointer text-sm font-medium">Unsupported block: {unknownBlock.blockType}</summary>
          <pre className="mt-2 overflow-x-auto text-xs">{JSON.stringify(unknownBlock.raw, null, 2)}</pre>
        </details>
      )
    }
  }
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined
  }
  return value as Record<string, unknown>
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined
}

function cardToFormContent(card: ChatStructuredCard): ChatStructuredForm | undefined {
  const fields = Array.isArray(card.raw?.fields)
    ? card.raw.fields
        .map((item) => {
          const entry = asRecord(item)
          const name = asString(entry?.name)
          const label = asString(entry?.label)
          if (!entry || !name || !label) {
            return null
          }
          return {
            name,
            label,
            fieldType: asString(entry.type) ?? asString(entry.fieldType),
            value: asString(entry.value),
            required: asBoolean(entry.required),
            placeholder: asString(entry.placeholder),
          }
        })
        .filter((item): item is NonNullable<typeof item> => item !== null)
    : undefined

  return {
    type: "form",
    kind: card.kind,
    version: card.version,
    title: card.title,
    content: asString(card.raw?.content) ?? asString(card.raw?.description),
    fields,
    actions: card.actions,
    raw: card.raw,
  }
}

function cardToProgressContent(card: ChatStructuredCard): ChatStructuredProgress | undefined {
  const steps = Array.isArray(card.raw?.steps)
    ? card.raw.steps
        .map((item) => {
          const entry = asRecord(item)
          const label = asString(entry?.label)
          if (!entry || !label) {
            return null
          }
          return {
            label,
            status: asString(entry.status),
            detail:
              asString(entry.detail) ??
              asString(entry.description) ??
              asString(entry.message),
          }
        })
        .filter((item): item is NonNullable<typeof item> => item !== null)
    : undefined

  return {
    type: "progress",
    kind: card.kind,
    version: card.version,
    title: card.title,
    content:
      asString(card.raw?.content) ??
      asString(card.raw?.description) ??
      asString(card.raw?.message),
    status: asString(card.raw?.status),
    steps,
    raw: card.raw,
  }
}

function cardToAlertContent(card: ChatStructuredCard): ChatStructuredAlert | undefined {
  return {
    type: "alert",
    kind: card.kind,
    version: card.version,
    title: card.title,
    level:
      asString(card.raw?.level) ??
      asString(card.raw?.severity) ??
      asString(card.raw?.statusLevel),
    content:
      asString(card.raw?.content) ??
      asString(card.raw?.description) ??
      asString(card.raw?.message),
    actions: card.actions,
    raw: card.raw,
  }
}

function cardToTodoContent(card: ChatStructuredCard): ChatStructuredTodo | undefined {
  const items = Array.isArray(card.raw?.items)
    ? card.raw.items
        .map((item) => {
          const entry = asRecord(item)
          const title = asString(entry?.title)
          if (!entry || !title) {
            return null
          }
          return {
            id: asString(entry.id),
            title,
            status: (
              entry.status === "not-started" ||
              entry.status === "in-progress" ||
              entry.status === "completed"
                ? entry.status
                : undefined) as ChatTodoStatus | undefined,
            detail:
              asString(entry.detail) ??
              asString(entry.description) ??
              asString(entry.message),
          }
        })
        .filter((item): item is NonNullable<typeof item> => item !== null)
    : undefined

  return {
    type: "todo",
    kind: card.kind,
    version: card.version,
    title: card.title,
    content:
      asString(card.raw?.content) ??
      asString(card.raw?.description) ??
      asString(card.raw?.message),
    items,
    raw: card.raw,
  }
}

function OptionsStructuredContent({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredOptions
  onSelectOption?: (value: string) => void
}) {
  const [selectedOptions, setSelectedOptions] = useState<string[]>([])
  const [customValue, setCustomValue] = useState("")
  const mode = structured.mode ?? "single"
  const submitLabel = structured.submitLabel ?? "Send"
  const customPlaceholder =
    structured.customPlaceholder ?? "Enter a custom value"
  const canSubmitMultiple =
    selectedOptions.length > 0 || customValue.trim().length > 0
  const selectedLabels = structured.options
    .filter((option) => selectedOptions.includes(option.value))
    .map((option) => option.label)

  const toggleOption = (value: string) => {
    if (mode === "single") {
      onSelectOption?.(value)
      return
    }

    setSelectedOptions((prev) =>
      prev.includes(value)
        ? prev.filter((item) => item !== value)
        : [...prev, value],
    )
  }

  const submitSelection = () => {
    const values = [...selectedOptions]
    if (customValue.trim()) {
      values.push(customValue.trim())
    }
    if (values.length === 0) {
      return
    }
    onSelectOption?.(mode === "multiple" ? values.join("\n") : values[0])
    setSelectedOptions([])
    setCustomValue("")
  }

  return (
    <div className={STRUCTURED_PANEL_CLASS}>
      <StructuredHeader
        eyebrow="Options"
        badge={mode === "multiple" ? "Multiple" : "Single"}
      />
      {mode === "multiple" && (
        <div className="text-muted-foreground text-xs">
          Select one or more options, then press {submitLabel}.
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        {structured.options.map((option) => {
          const isSelected = selectedOptions.includes(option.value)
          return (
            <Button
              key={`${option.label}-${option.value}`}
              type="button"
              variant={mode === "multiple" && isSelected ? "secondary" : "outline"}
              size="sm"
              className="h-auto max-w-full items-start justify-start whitespace-normal text-left"
              onClick={() => toggleOption(option.value)}
            >
              <span className="flex w-full flex-col items-start gap-0.5">
                <span>{option.label}</span>
                {option.description && (
                  <span className="text-muted-foreground text-xs font-normal">
                    {option.description}
                  </span>
                )}
                {mode === "multiple" && isSelected && (
                  <span className="text-xs font-medium">Selected</span>
                )}
              </span>
            </Button>
          )
        })}
      </div>
      {mode === "multiple" && selectedOptions.length > 0 && (
        <div className="text-muted-foreground rounded-lg border border-border/60 bg-background/70 px-3 py-2 text-xs">
          Selected: {selectedLabels.join(", ")}
        </div>
      )}
      {structured.allowCustom && (
        <div className="flex gap-2">
          <Input
            value={customValue}
            placeholder={customPlaceholder}
            onChange={(event) => setCustomValue(event.target.value)}
            onKeyDown={(event) => {
              if (event.key !== "Enter") {
                return
              }
              event.preventDefault()
              submitSelection()
            }}
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="shrink-0"
            onClick={submitSelection}
            disabled={!customValue.trim() && mode !== "multiple"}
          >
            {submitLabel}
          </Button>
        </div>
      )}
      {mode === "multiple" && (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          className="h-auto whitespace-normal"
          onClick={submitSelection}
          disabled={!canSubmitMultiple}
        >
          {submitLabel}
        </Button>
      )}
      {structured.options.length === 0 &&
        renderStructuredRawFallback(structured.raw, "Unsupported options payload")}
    </div>
  )
}

function GenericCardStructuredContent({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredCard
  onSelectOption?: (value: string) => void
}) {
  const hasRenderableContent =
    Boolean(structured.blocks?.length) || Boolean(structured.actions?.length)

  return (
    <div className={STRUCTURED_PANEL_CLASS}>
      <StructuredHeader title={structured.title} kind={structured.kind} eyebrow="Card" />
      {structured.blocks?.map((block, index) => (
        <StructuredCardBlockView key={`${block.type}-${index}`} block={block} onSelectOption={onSelectOption} />
      ))}
      {structured.actions && structured.actions.length > 0 && (
        <StructuredActions actions={structured.actions} onSelectOption={onSelectOption} />
      )}
      {!hasRenderableContent &&
        renderStructuredRawFallback(
          structured.raw,
          structured.kind ? `Custom card payload: ${structured.kind}` : "Unsupported card payload",
        )}
    </div>
  )
}

function FormStructuredContent({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredForm
  onSelectOption?: (value: string) => void
}) {
  const hasRenderableContent =
    Boolean(structured.content?.trim()) ||
    Boolean(structured.fields?.length) ||
    Boolean(structured.actions?.length)

  return (
    <div className={STRUCTURED_PANEL_CLASS}>
      <StructuredHeader title={structured.title} kind={structured.kind} eyebrow="Form" />
      {renderStructuredText(structured.content)}
      {structured.fields?.map((field) => (
        <div key={field.name} className={STRUCTURED_SUBSECTION_CLASS}>
          <div className="flex items-center gap-2">
            <div className="text-sm font-medium">{field.label}</div>
            {field.required && <span className="text-destructive text-xs">*</span>}
          </div>
          <div className="text-muted-foreground mt-1 text-sm">
            {field.value || field.placeholder || field.fieldType || field.name}
          </div>
        </div>
      ))}
      {structured.actions && structured.actions.length > 0 && (
        <StructuredActions actions={structured.actions} onSelectOption={onSelectOption} />
      )}
      {!hasRenderableContent &&
        renderStructuredRawFallback(
          structured.raw,
          structured.kind ? `Custom form payload: ${structured.kind}` : "Unsupported form payload",
        )}
    </div>
  )
}

function ProgressStructuredContent({ structured }: { structured: ChatStructuredProgress }) {
  const isToolProcess = structured.kind === "agent/tool-exec"
  const hasRenderableContent =
    Boolean(structured.content?.trim()) ||
    Boolean(structured.steps?.length) ||
    Boolean(structured.status?.trim())

  if (isToolProcess) {
    return <ToolProgressPanel items={[structured]} />
  }

  return (
    <div className="space-y-3 rounded-xl border border-sky-200/70 bg-sky-50/50 p-4 shadow-sm dark:border-sky-500/20 dark:bg-sky-500/5">
      <StructuredHeader
        title={structured.title}
        kind={structured.kind}
        badge={structured.status}
        eyebrow="Progress"
      />
      {renderStructuredText(structured.content)}
      <div className="space-y-2">
        {structured.steps?.map((step, index) => (
          <div key={`${step.label}-${index}`} className={STRUCTURED_SUBSECTION_CLASS}>
            <div className="flex flex-wrap items-center gap-2">
              <div className="text-sm font-medium">{step.label}</div>
              {step.status && (
                <StructuredMetaPill tone="muted">{step.status}</StructuredMetaPill>
              )}
            </div>
            {step.detail && (
              <div className="text-muted-foreground mt-1 text-sm">{step.detail}</div>
            )}
          </div>
        ))}
      </div>
      {!hasRenderableContent &&
        renderStructuredRawFallback(
          structured.raw,
          structured.kind
            ? `Custom progress payload: ${structured.kind}`
            : "Unsupported progress payload",
        )}
    </div>
  )
}

function AlertStructuredContent({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredAlert
  onSelectOption?: (value: string) => void
}) {
  const levelClassName =
    structured.level === "error"
      ? "border-red-300/80 bg-red-50/70 text-red-900 dark:border-red-500/35 dark:bg-red-500/10 dark:text-red-100"
      : structured.level === "warning"
        ? "border-amber-300/80 bg-amber-50/70 text-amber-900 dark:border-amber-500/35 dark:bg-amber-500/10 dark:text-amber-100"
        : "border-blue-300/80 bg-blue-50/70 text-blue-900 dark:border-blue-500/35 dark:bg-blue-500/10 dark:text-blue-100"
  const hasRenderableContent =
    Boolean(structured.title?.trim()) ||
    Boolean(structured.level?.trim()) ||
    Boolean(structured.content?.trim()) ||
    Boolean(structured.actions?.length)

  return (
    <div className={cn("space-y-3 rounded-xl border p-4 shadow-sm", levelClassName)}>
      <StructuredHeader
        title={structured.title}
        kind={structured.kind}
        badge={structured.level?.toUpperCase()}
        eyebrow="Alert"
      />
      {renderStructuredText(structured.content)}
      {structured.actions && structured.actions.length > 0 && (
        <StructuredActions actions={structured.actions} onSelectOption={onSelectOption} />
      )}
      {!hasRenderableContent &&
        renderStructuredRawFallback(
          structured.raw,
          structured.kind ? `Custom alert payload: ${structured.kind}` : "Unsupported alert payload",
        )}
    </div>
  )
}

function TodoStructuredContent({ structured }: { structured: ChatStructuredTodo }) {
  const hasRenderableContent =
    Boolean(structured.title?.trim()) ||
    Boolean(structured.content?.trim()) ||
    Boolean(structured.items?.length)

  return (
    <div className="space-y-3">
      {structured.items && structured.items.length > 0 && (
        <TodoListPanel
          title={structured.title}
          content={structured.content}
          items={structured.items}
        />
      )}
      {!hasRenderableContent &&
        renderStructuredRawFallback(
          structured.raw,
          structured.kind ? `Custom todo payload: ${structured.kind}` : "Unsupported todo payload",
        )}
    </div>
  )
}

function UnknownStructuredContent({ structured }: { structured: ChatStructuredContent }) {
  const kind = "kind" in structured ? structured.kind : undefined
  return (
    <details className="rounded-xl border border-dashed border-border/70 bg-muted/20 p-3 shadow-sm">
      <summary className="cursor-pointer text-sm font-medium">
        Unsupported card{kind ? `: ${kind}` : ""}
      </summary>
      <pre className="mt-2 overflow-x-auto text-xs">{JSON.stringify(structured.raw, null, 2)}</pre>
    </details>
  )
}

function CardKindFormStructuredContent({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredCard
  onSelectOption?: (value: string) => void
}) {
  const content = cardToFormContent(structured)
  return content ? (
    <FormStructuredContent structured={content} onSelectOption={onSelectOption} />
  ) : (
    <GenericCardStructuredContent structured={structured} onSelectOption={onSelectOption} />
  )
}

function CardKindProgressStructuredContent({
  structured,
}: {
  structured: ChatStructuredCard
}) {
  const content = cardToProgressContent(structured)
  return content ? <ProgressStructuredContent structured={content} /> : <GenericCardStructuredContent structured={structured} />
}

function CardKindAlertStructuredContent({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredCard
  onSelectOption?: (value: string) => void
}) {
  const content = cardToAlertContent(structured)
  return content ? (
    <AlertStructuredContent structured={content} onSelectOption={onSelectOption} />
  ) : (
    <GenericCardStructuredContent structured={structured} onSelectOption={onSelectOption} />
  )
}

function CardKindTodoStructuredContent({ structured }: { structured: ChatStructuredCard }) {
  const content = cardToTodoContent(structured)
  return content ? <TodoStructuredContent structured={content} /> : <GenericCardStructuredContent structured={structured} />
}

// VS Code style: one key function determines the part kind, one registry maps it to a renderer.
// Adding a new kind = adding one entry to PART_RENDERERS. No nested if/else anywhere.
// Mirrors how VS Code's ResponsePartKind drives ChatResponsePart dispatch.
function getStructuredPartKind(structured: ChatStructuredContent): string {
  // card+kind is the canonical form for semantic variants (form, progress, todo, alert).
  // The backend normalizes all alias input types to card+kind before sending.
  if (structured.type === "card" && structured.kind) {
    return `card:${structured.kind}`
  }
  return structured.type
}

type StructuredPartRenderer = React.ComponentType<{
  structured: ChatStructuredContent
  onSelectOption?: (value: string) => void
}>

// Single registry keyed by part kind.
// Canonical protocol types sit at the top; card kinds follow.
// Legacy flat types (form/progress/todo/alert) are kept as a safety net for
// clients that haven't been updated — they route to the same renderer components.
const PART_RENDERERS: Record<string, StructuredPartRenderer> = {
  // Canonical types
  options: OptionsStructuredContent as StructuredPartRenderer,
  card: GenericCardStructuredContent as StructuredPartRenderer,
  // Canonical card kinds — backend normalizes aliases here
  "card:form": CardKindFormStructuredContent as StructuredPartRenderer,
  "card:builtin/form": CardKindFormStructuredContent as StructuredPartRenderer,
  "card:progress": CardKindProgressStructuredContent as StructuredPartRenderer,
  "card:builtin/progress": CardKindProgressStructuredContent as StructuredPartRenderer,
  "card:alert": CardKindAlertStructuredContent as StructuredPartRenderer,
  "card:builtin/alert": CardKindAlertStructuredContent as StructuredPartRenderer,
  "card:todo": CardKindTodoStructuredContent as StructuredPartRenderer,
  "card:builtin/todo": CardKindTodoStructuredContent as StructuredPartRenderer,
  // Legacy flat alias types
  form: FormStructuredContent as StructuredPartRenderer,
  progress: ProgressStructuredContent as StructuredPartRenderer,
  alert: AlertStructuredContent as StructuredPartRenderer,
  todo: TodoStructuredContent as StructuredPartRenderer,
}

function StructuredContentView({
  structured,
  onSelectOption,
}: {
  structured: ChatStructuredContent
  onSelectOption?: (value: string) => void
}) {
  const partKind = getStructuredPartKind(structured)
  const Renderer = PART_RENDERERS[partKind] ?? UnknownStructuredContent
  return <Renderer structured={structured as never} onSelectOption={onSelectOption} />
}

function isToolProgressStructured(
  structured: ChatStructuredContent,
): structured is ChatStructuredProgress {
  return structured.type === "progress" && structured.kind === "agent/tool-exec"
}

function splitStructuredParts(structured: ChatStructuredValue | undefined): {
  toolProgressParts: ChatStructuredProgress[]
  otherParts: ChatStructuredContent[]
} {
  if (!structured) {
    return { toolProgressParts: [], otherParts: [] }
  }

  const parts = Array.isArray(structured) ? structured : [structured]
  const toolProgressParts: ChatStructuredProgress[] = []
  const otherParts: ChatStructuredContent[] = []

  for (const part of parts) {
    if (isToolProgressStructured(part)) {
      toolProgressParts.push(part)
      continue
    }
    otherParts.push(part)
  }

  return { toolProgressParts, otherParts }
}

function TodoStatusIcon({ status }: { status?: ChatTodoItem["status"] }) {
  if (status === "completed") {
    return (
      <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-emerald-500/60 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300">
        <IconCheck className="size-3.5" />
      </span>
    )
  }
  if (status === "in-progress") {
    return (
      <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-sky-500/40 bg-sky-500/10 text-sky-600 dark:text-sky-300">
        <IconLoader2 className="size-3.5 animate-spin" />
      </span>
    )
  }

  return (
    <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-border/70 bg-background text-muted-foreground">
      <IconClockHour4 className="size-3" />
    </span>
  )
}

function isPlanLikeTodo(title?: string, content?: string, items: ChatTodoItem[] = []): boolean {
  const normalized = [title, content, ...items.map((item) => item.title)]
    .filter((value): value is string => Boolean(value?.trim()))
    .join("\n")
    .toLowerCase()

  return /计划|规划|待办|任务|执行|步骤|拆解|plan|todo|task|tasks|step|steps|phase|milestone|implement|fix|review|verify|ship|refactor/.test(
    normalized,
  )
}

function hasInformationalTitleShape(title: string): boolean {
  if (/[:：]/.test(title)) {
    return true
  }

  return /km|公里|m\b|米|预算|时间|海拔|难度|装备|温差|距离|费用|日期|人数|路线|目录|文件|路径|版本|状态|配置|说明|摘要|总结|结果/i.test(
    title,
  )
}

function hasInformationalPanelTitle(title?: string): boolean {
  if (!title?.trim()) {
    return false
  }

  return /概述|概览|总览|摘要|总结|路线|信息|说明|结果|清单|一览|概况|overview|summary|outline|brief/i.test(
    title,
  )
}

function isInformationalTodo(
  title: string | undefined,
  content: string | undefined,
  items: ChatTodoItem[],
): boolean {
  if (items.length === 0 || items.some((item) => item.status)) {
    return false
  }

  if (hasInformationalPanelTitle(title)) {
    return true
  }

  if (isPlanLikeTodo(title, content, items)) {
    return false
  }

  const informationalCount = items.filter((item) => hasInformationalTitleShape(item.title)).length
  return informationalCount >= Math.max(1, Math.ceil(items.length / 2))
}

function splitInformationalTitle(title: string): {
  label: string
  value?: string
} {
  const match = title.match(/^([^:：]+)[:：]\s*(.+)$/)
  if (!match) {
    return { label: title }
  }

  return {
    label: match[1]?.trim() ?? title,
    value: match[2]?.trim() || undefined,
  }
}

function InformationalListPanel({
  title,
  content,
  items,
}: {
  title?: string
  content?: string
  items: ChatTodoItem[]
}) {
  const [isOpen, setIsOpen] = useState(true)
  const panelTitle = title?.trim() || "信息摘要"

  return (
    <div className="overflow-hidden rounded-lg border border-border/70 bg-background shadow-sm">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="flex w-full items-center gap-2 border-b border-border/60 px-3 py-2 text-left hover:bg-muted/20"
      >
        {isOpen ? (
          <IconChevronDown className="size-4 shrink-0 text-muted-foreground" />
        ) : (
          <IconChevronRight className="size-4 shrink-0 text-muted-foreground" />
        )}
        <div className="min-w-0 text-sm font-medium text-foreground">{panelTitle}</div>
      </button>
      {isOpen && (
        <div>
          {content?.trim() && (
            <div className="border-b border-border/60 px-4 py-2 text-sm leading-6 text-muted-foreground">
              {content}
            </div>
          )}
          <div className="px-3 py-2">
            {items.map((item, index) => {
              const { label, value } = splitInformationalTitle(item.title)

              return (
                <div
                  key={`${item.id ?? item.title}-${index}`}
                  className="flex items-start gap-3 rounded-md px-1.5 py-2 hover:bg-muted/20"
                >
                  <span className="mt-2 size-2 shrink-0 rounded-full bg-muted-foreground/45" />
                  <div className="min-w-0 flex-1 space-y-1">
                    {value ? (
                      <div className="text-[14px] leading-6 text-foreground">
                        <span className="font-medium">{label}:</span>{" "}
                        <span>{value}</span>
                      </div>
                    ) : (
                      <div className="break-words text-[14px] leading-6 text-foreground">
                        {label}
                      </div>
                    )}
                    {item.detail && (
                      <div className="text-xs leading-5 text-muted-foreground">
                        {item.detail}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

function ToolProgressStatusIcon({ status }: { status?: string }) {
  const normalized = (status ?? "").toLowerCase()

  if (normalized === "completed") {
    return (
      <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-emerald-500/60 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300">
        <IconCheck className="size-3.5" />
      </span>
    )
  }
  if (normalized === "error") {
    return (
      <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-red-500/50 bg-red-500/10 text-red-600 dark:text-red-300">
        <IconAlertCircle className="size-3.5" />
      </span>
    )
  }
  if (normalized === "running" || normalized === "in-progress") {
    return (
      <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-sky-500/40 bg-sky-500/10 text-sky-600 dark:text-sky-300">
        <IconLoader2 className="size-3.5 animate-spin" />
      </span>
    )
  }

  return (
    <span className="mt-0.5 flex size-5 items-center justify-center rounded-full border border-border/70 bg-background text-muted-foreground">
      <IconClockHour4 className="size-3" />
    </span>
  )
}

function ToolProgressPanel({
  items,
  title,
}: {
  items: ChatStructuredProgress[]
  title?: string
}) {
  const [isOpen, setIsOpen] = useState(true)
  const completedCount = items.filter(
    (item) => (item.status ?? "").toLowerCase() === "completed",
  ).length
  const panelTitle = title?.trim() || "执行过程"

  if (items.length === 0) {
    return null
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border/70 bg-background shadow-sm">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="flex w-full items-center gap-2 border-b border-border/60 px-3 py-2 text-left hover:bg-muted/20"
      >
        {isOpen ? (
          <IconChevronDown className="size-4 shrink-0 text-muted-foreground" />
        ) : (
          <IconChevronRight className="size-4 shrink-0 text-muted-foreground" />
        )}
        <div className="min-w-0 text-sm font-medium text-foreground">
          {panelTitle}
          <span className="ml-1 text-muted-foreground">({completedCount}/{items.length})</span>
        </div>
      </button>
      {isOpen && (
        <div className="px-3 py-2">
          {items.map((item, index) => {
            const normalized = (item.status ?? "").toLowerCase()
            const itemTitle = item.title?.trim() || `执行步骤 ${index + 1}`

            return (
              <div
                key={`${itemTitle}-${index}`}
                className="flex items-start gap-3 rounded-md px-1.5 py-2 hover:bg-muted/20"
              >
                <ToolProgressStatusIcon status={item.status} />
                <div className="min-w-0 flex-1 space-y-1">
                  <div
                    className={cn(
                      "break-words text-[14px] leading-5 text-foreground",
                      normalized === "completed" && "text-muted-foreground line-through",
                    )}
                  >
                    {itemTitle}
                  </div>
                  {item.content?.trim() && (
                    <div className="text-xs leading-5 text-muted-foreground">{item.content}</div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

function TodoListPanel({
  title,
  content,
  items,
}: {
  title?: string
  content?: string
  items: ChatTodoItem[]
}) {
  const [isOpen, setIsOpen] = useState(true)

  if (items.length === 0) {
    return null
  }
  if (isInformationalTodo(title, content, items)) {
    return <InformationalListPanel title={title} content={content} items={items} />
  }
  const completedCount = items.filter((item) => item.status === "completed").length
  const panelTitle = title?.trim() || "待办事项"

  return (
    <div className="overflow-hidden rounded-lg border border-border/70 bg-background shadow-sm">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="flex w-full items-center gap-2 border-b border-border/60 px-3 py-2 text-left hover:bg-muted/20"
      >
        {isOpen ? (
          <IconChevronDown className="size-4 shrink-0 text-muted-foreground" />
        ) : (
          <IconChevronRight className="size-4 shrink-0 text-muted-foreground" />
        )}
        <div className="min-w-0 text-sm font-medium text-foreground">
          {panelTitle}
          <span className="ml-1 text-muted-foreground">({completedCount}/{items.length})</span>
        </div>
      </button>
      {isOpen && (
        <div>
          {content?.trim() && (
            <div className="border-b border-border/60 px-4 py-2 text-sm leading-6 text-muted-foreground">
              {content}
            </div>
          )}
          <div className="px-3 py-2">
            {items.map((item, index) => (
              <div
                key={`${item.id ?? item.title}-${index}`}
                className="flex items-start gap-3 rounded-md px-1.5 py-2 hover:bg-muted/20"
              >
                <TodoStatusIcon status={item.status} />
                <div className="min-w-0 flex-1 space-y-1">
                  <div
                    className={cn(
                      "break-words text-[14px] leading-5 text-foreground",
                      item.status === "completed" && "text-muted-foreground line-through",
                    )}
                  >
                    {item.title}
                  </div>
                  {item.detail && (
                    <div className="text-xs leading-5 text-muted-foreground">
                      {item.detail}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function extractPlanPreviewItems(content: string): ChatTodoItem[] {
  const lines = content.replace(/\r\n/g, "\n").split("\n")
  const items: ChatTodoItem[] = []

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) {
      continue
    }

    const headingMatch = line.match(/^#{2,6}\s+(.+)$/)
    if (headingMatch) {
      const title = headingMatch[1]?.replace(/[*_`#]/g, "").trim() ?? ""
      if (/阶段|phase|milestone|任务/i.test(title) && !/项目目标|核心特性|任务分解/i.test(title)) {
        items.push({ title, status: "not-started" })
      }
      continue
    }

    const bulletMatch = line.match(/^[-*+]\s*(?:\[(?: |x|X)\]\s*)?(.+)$/)
    if (bulletMatch && items.length < 8) {
      const title = bulletMatch[1]?.replace(/[*_`]/g, "").trim() ?? ""
      if (title && !/^T\d/.test(title)) {
        items.push({ title, status: /done|completed|已完成/i.test(title) ? "completed" : "not-started" })
      }
    }
  }

  const deduped: ChatTodoItem[] = []
  const seen = new Set<string>()
  for (const item of items) {
    if (item.title && !seen.has(item.title)) {
      seen.add(item.title)
      deduped.push(item)
    }
  }
  if (deduped.length > 0 && !deduped.some((item) => item.status === "in-progress" || item.status === "completed")) {
    deduped[0] = { ...deduped[0], status: "in-progress" }
  }
  return deduped.slice(0, 6)
}

function InlinePlanPreview({ content }: { content: string }) {
  const items = extractPlanPreviewItems(content)
  if (items.length < 2) {
    return null
  }

  return <TodoListPanel title="Plan" items={items} />
}

function shouldPreferStructuredPanel(
  structured: ChatStructuredValue | undefined,
): boolean {
  if (!structured) {
    return false
  }

  const parts = Array.isArray(structured) ? structured : [structured]
  return parts.some(
    (part) =>
      part.type === "todo" ||
      (part.type === "progress" && part.kind === "agent/tool-exec"),
  )
}

export function AssistantMessage({
  content,
  isThought = false,
  timestamp = "",
  structured,
  onSelectOption,
}: AssistantMessageProps) {
  const { t } = useTranslation()
  const [isCopied, setIsCopied] = useState(false)
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""
  const hasBody = Boolean(content.trim())
  const shouldCollapseThought = isThought && content.trim().length > 160
  const inferredStructured =
    !structured && !isThought ? inferStructuredContentFromText(content) : undefined
  const effectiveStructured = structured ?? inferredStructured
  const { toolProgressParts, otherParts } = splitStructuredParts(effectiveStructured)
  const hasToolProgress = toolProgressParts.length > 0
  const preferStructuredPanel = shouldPreferStructuredPanel(effectiveStructured)
  const shouldShowInlinePlanPreview =
    !effectiveStructured && /规划|计划|阶段|任务分解|项目目标|任务|plan/i.test(content)

  const handleCopy = () => {
    navigator.clipboard.writeText(content).then(() => {
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    })
  }

  return (
    <div className="group flex w-full max-w-[820px] gap-3">
      <div className="bg-muted text-muted-foreground mt-5 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-border/70 text-[11px] font-semibold uppercase">
        AI
      </div>

      <div className={MESSAGE_CONTENT_CLASS}>
        <div className="text-muted-foreground flex items-center gap-2 text-[11px] uppercase tracking-[0.14em]">
          <span>PicoClaw</span>
          {isThought && (
            <span className="inline-flex items-center gap-1 rounded-full border border-amber-300/80 bg-amber-100/80 px-2 py-0.5 text-[10px] font-medium tracking-normal normal-case text-amber-800 dark:border-amber-500/40 dark:bg-amber-500/15 dark:text-amber-200">
              <IconBrain className="size-3" />
              <span>{t("chat.reasoningLabel")}</span>
            </span>
          )}
          {formattedTimestamp ? <span className="opacity-60">{formattedTimestamp}</span> : null}
        </div>

        {hasToolProgress && (
          <div className={cn(!hasBody && "pt-1")}>
            <ToolProgressPanel items={toolProgressParts} />
          </div>
        )}

        {hasBody && !isThought && (!preferStructuredPanel || hasToolProgress) && (
          <div className="relative rounded-lg px-1 py-0.5">
            {shouldShowInlinePlanPreview && (
              <div className="mb-3">
                <InlinePlanPreview content={content} />
              </div>
            )}
            <div className={cn(MARKDOWN_BODY_CLASS, "prose-p:my-2 text-[14px] leading-6 text-foreground")}>
              {renderMarkdown(content)}
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="absolute top-0 right-0 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100"
              onClick={handleCopy}
            >
              {isCopied ? (
                <IconCheck className="h-4 w-4 text-green-500" />
              ) : (
                <IconCopy className="text-muted-foreground h-4 w-4" />
              )}
            </Button>
          </div>
        )}

        {hasBody && isThought && !shouldCollapseThought && (
          <div className="rounded-xl border border-amber-200/80 bg-amber-50/60 p-4 text-amber-950 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/8 dark:text-amber-100">
            <div className={cn(MARKDOWN_BODY_CLASS, "prose-p:my-1.5 text-[13px] leading-relaxed opacity-90")}>
              {renderMarkdown(content)}
            </div>
          </div>
        )}

        {hasBody && isThought && shouldCollapseThought && (
          <details className="rounded-xl border border-amber-200/80 bg-amber-50/60 p-4 text-amber-950 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/8 dark:text-amber-100">
            <summary className="cursor-pointer text-sm font-medium opacity-90">
              {t("chat.reasoningLabel")}
            </summary>
            <div className={cn(MARKDOWN_BODY_CLASS, "mt-3 text-[13px] leading-relaxed opacity-90")}>
              {renderMarkdown(content)}
            </div>
          </details>
        )}

        {effectiveStructured && (
          <div className={cn(!hasBody && "pt-1")}>
            {otherParts.length > 0 ? (
              <div className="space-y-3">
                {otherParts.map((part, index) => (
                  <StructuredContentView
                    key={`${part.type}-${index}`}
                    structured={part}
                    onSelectOption={onSelectOption}
                  />
                ))}
              </div>
            ) : null}
          </div>
        )}
      </div>
    </div>
  )
}
