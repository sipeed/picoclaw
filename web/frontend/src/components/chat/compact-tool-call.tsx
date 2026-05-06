import { useState } from "react"

import type { ChatToolCall } from "@/store/chat"

interface CompactToolCallProps {
  toolCall: ChatToolCall
  isRequestPermission?: boolean
}

export function CompactToolCall({
  toolCall,
  isRequestPermission = false,
}: CompactToolCallProps) {
  const [isExpanded, setIsExpanded] = useState(isRequestPermission)
  const toolName = toolCall.function?.name?.trim() ?? ""
  const toolArguments = toolCall.function?.arguments?.trim() ?? ""
  const explanation =
    toolCall.extraContent?.toolFeedbackExplanation?.trim() ?? ""

  // Try to parse args for display
  let parsedArgs: Record<string, unknown> | null = null
  if (toolArguments) {
    try {
      parsedArgs = JSON.parse(toolArguments)
    } catch {
      // ignore
    }
  }

  // Status icon
  const isError = explanation.toLowerCase().includes("error") || explanation.toLowerCase().includes("denied")

  return (
    <div
      className={`flex items-start gap-2 py-1.5 text-[13px] border-b border-border/20 last:border-0 ${
        isRequestPermission ? "bg-yellow-500/5 -mx-2 px-2 rounded-md" : ""
      }`}
    >
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-1.5 flex-1 text-left hover:opacity-80 transition-opacity"
      >
        <div
          className={`flex items-center justify-center w-5 h-5 rounded shrink-0 mt-0.5 text-xs ${
            isRequestPermission
              ? "bg-yellow-500/20 text-yellow-600"
              : isError
                ? "bg-red-500/20 text-red-600"
                : "bg-green-500/20 text-green-600"
          }`}
        >
          {isExpanded ? "▼" : "🔧"}
        </div>
        <span className="font-medium text-foreground/90">{toolName}</span>
        {!isExpanded && parsedArgs && (
          <span className="text-muted-foreground/70 truncate">
            {String(parsedArgs.path ?? parsedArgs.command ?? "")}
          </span>
        )}
        {!isExpanded && !parsedArgs && explanation && (
          <span className="text-muted-foreground/70 truncate max-w-[200px]">
            {explanation}
          </span>
        )}
        {!isExpanded && (
          <span className={`shrink-0 ml-auto text-xs ${isError ? "text-red-500" : "text-green-500"}`}>
            {isError ? "✗" : "✓"}
          </span>
        )}
      </button>

      {isExpanded && (explanation || toolArguments) && (
        <div className="mt-1 space-y-2 w-full">
          {explanation && (
            <div className="text-muted-foreground/80 text-[12px] leading-relaxed whitespace-pre-wrap">
              {explanation}
            </div>
          )}
          {toolArguments && (
            <div className="bg-muted/30 rounded-md p-2 font-mono text-[11px] overflow-x-auto">
              <pre className="whitespace-pre-wrap break-all">{toolArguments}</pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
