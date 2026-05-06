import {
  IconArrowRight,
  IconBrain,
  IconMicrophone,
  IconMicrophoneOff,
  IconPhoto,
  IconSearch,
  IconSettings,
  IconUpload,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import dayjs from "dayjs"
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from "react"
import { toast } from "sonner"

import type { ChatAttachment } from "@/store/chat"
import { usePicoChat } from "@/hooks/use-pico-chat"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"

import { MemoryGraph } from "./memory-graph"
import { useAgentCockpit } from "./use-agent-cockpit"

const MAX_IMAGE_SIZE_BYTES = 7 * 1024 * 1024
const ALLOWED_IMAGE_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
  "image/bmp",
])

declare global {
  interface Window {
    SpeechRecognition?: new () => SpeechRecognitionLike
    webkitSpeechRecognition?: new () => SpeechRecognitionLike
  }
}

interface SpeechRecognitionLike {
  continuous: boolean
  interimResults: boolean
  lang: string
  onresult: ((event: SpeechRecognitionEventLike) => void) | null
  onend: (() => void) | null
  onerror: ((event: { error: string }) => void) | null
  start(): void
  stop(): void
}

interface SpeechRecognitionEventLike {
  results: ArrayLike<ArrayLike<{ transcript: string }>>
}

function statusBadgeVariant(status: string) {
  switch (status) {
    case "enabled":
    case "completed":
      return "default" as const
    case "blocked":
    case "failed":
      return "destructive" as const
    case "running":
      return "secondary" as const
    default:
      return "outline" as const
  }
}

function reasonLabel(reasonCode?: string) {
  switch (reasonCode) {
    case "requires_subagent":
      return "Requires subagent runtime"
    case "requires_skills":
      return "Requires skills support"
    case "requires_mcp_discovery":
      return "Requires MCP discovery"
    case "requires_linux":
      return "Linux only"
    case "requires_serial_platform":
      return "Unsupported serial platform"
    default:
      return reasonCode ?? ""
  }
}

function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === "string") {
        resolve(reader.result)
        return
      }
      reject(new Error("Failed to read file"))
    }
    reader.onerror = () =>
      reject(reader.error || new Error("Failed to read file"))
    reader.readAsDataURL(file)
  })
}

export function CockpitPage() {
  const { activeSessionId, connectionState, sendMessage } = usePicoChat()
  const {
    categoryCounts,
    groupedTools,
    pendingToolName,
    searchQuery,
    sessionSubagents,
    sessionMemoryGraph,
    statusCounts,
    statusFilter,
    webSearchConfig,
    hasMemoryGraphError,
    hasSubagentsError,
    hasToolsError,
    isMemoryGraphLoading,
    isSubagentsLoading,
    isToolsLoading,
    isWebSearchLoading,
    setSearchQuery,
    setStatusFilter,
    toggleTool,
  } = useAgentCockpit(activeSessionId)

  const [prompt, setPrompt] = useState("")
  const [attachments, setAttachments] = useState<ChatAttachment[]>([])
  const [isListening, setIsListening] = useState(false)
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null)

  useEffect(() => {
    const Recognition =
      window.SpeechRecognition ?? window.webkitSpeechRecognition
    if (!Recognition) {
      return
    }
    const recognition = new Recognition()
    recognition.continuous = false
    recognition.interimResults = false
    recognition.lang = "en-US"
    recognition.onresult = (event) => {
      const transcript = event.results[0]?.[0]?.transcript?.trim() ?? ""
      setPrompt(transcript)
      if (!transcript) {
        return
      }
      const sent = sendMessage({ content: transcript })
      if (!sent) {
        toast.error("Voice capture worked, but chat is not ready to send.")
      }
    }
    recognition.onend = () => setIsListening(false)
    recognition.onerror = (event) => {
      setIsListening(false)
      toast.error(`Voice capture error: ${event.error}`)
    }
    recognitionRef.current = recognition
  }, [sendMessage])

  const filteredToolCount = useMemo(
    () => groupedTools.reduce((total, [, items]) => total + items.length, 0),
    [groupedTools],
  )

  const currentProviderLabel = useMemo(() => {
    const current = webSearchConfig?.providers.find((provider) => provider.current)
    return current?.label ?? webSearchConfig?.provider ?? "Auto"
  }, [webSearchConfig])

  const handleImageSelection = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? [])
    event.target.value = ""
    if (files.length === 0) {
      return
    }

    const nextAttachments: ChatAttachment[] = []
    for (const file of files) {
      if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
        toast.error(`Unsupported image type: ${file.name}`)
        continue
      }
      if (file.size > MAX_IMAGE_SIZE_BYTES) {
        toast.error(`${file.name} exceeds 7 MB.`)
        continue
      }
      try {
        const url = await readFileAsDataUrl(file)
        nextAttachments.push({
          type: "image",
          url,
          filename: file.name,
          contentType: file.type,
        })
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : `Failed to read ${file.name}`,
        )
      }
    }

    setAttachments((current) => [...current, ...nextAttachments])
  }

  const handleSendBridgeMessage = () => {
    const sent = sendMessage({ content: prompt, attachments })
    if (!sent) {
      toast.error("Chat connection is not ready yet.")
      return
    }
    setPrompt("")
    setAttachments([])
  }

  const toggleVoice = () => {
    if (!recognitionRef.current) {
      toast.error("Voice capture is not supported in this browser.")
      return
    }
    if (isListening) {
      recognitionRef.current.stop()
      setIsListening(false)
      return
    }
    try {
      recognitionRef.current.start()
      setIsListening(true)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Unable to start voice capture.",
      )
      setIsListening(false)
    }
  }

  return (
    <div className="flex h-full flex-col overflow-hidden bg-[#07110a] text-[#d7f9df]">
      <PageHeader title="Agent Cockpit" />

      <div className="flex-1 overflow-auto px-4 py-4 sm:px-6 sm:py-6">
        <div className="mx-auto grid w-full max-w-[1500px] gap-4 xl:grid-cols-[260px_minmax(0,1fr)_360px]">
          <aside className="space-y-4">
            <Card className="border-[#1f4f31] bg-[#09150c] text-[#d7f9df] shadow-none">
              <CardHeader>
                <CardTitle className="font-mono text-sm uppercase tracking-[0.22em]">
                  Systems
                </CardTitle>
                <CardDescription className="text-[#8bb39a]">
                  Filter the active tool surface.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="relative">
                  <IconSearch className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-[#6aa37b]" />
                  <Input
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    placeholder="Search tools"
                    className="border-[#1f4f31] bg-[#050d08] pl-9 text-[#d7f9df] placeholder:text-[#5c8168]"
                  />
                </div>

                <div className="space-y-2">
                  {(["all", "enabled", "disabled", "blocked"] as const).map(
                    (status) => (
                      <button
                        key={status}
                        type="button"
                        onClick={() => setStatusFilter(status)}
                        className={cn(
                          "flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm transition-colors",
                          statusFilter === status
                            ? "border-[#4fc267] bg-[#10351b] text-[#effff3]"
                            : "border-[#173621] bg-[#08100b] text-[#8bb39a] hover:border-[#2c6a3e] hover:text-[#d7f9df]",
                        )}
                      >
                        <span className="capitalize">{status}</span>
                        <span className="font-mono text-xs">
                          {statusCounts[status]}
                        </span>
                      </button>
                    ),
                  )}
                </div>

                <div className="space-y-2 border-t border-[#173621] pt-4">
                  <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-[#7ca88a]">
                    Categories
                  </p>
                  {categoryCounts.map(([category, count]) => (
                    <div
                      key={category}
                      className="flex items-center justify-between text-sm text-[#9cc8a8]"
                    >
                      <span className="capitalize">{category}</span>
                      <span className="font-mono text-xs">{count}</span>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </aside>

          <section className="space-y-4">
            <Card className="border-[#215d36] bg-[linear-gradient(180deg,#0a150c_0%,#0a1c10_100%)] text-[#effff3] shadow-none">
              <CardHeader>
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <CardTitle className="font-mono text-sm uppercase tracking-[0.24em]">
                      Tool Grid
                    </CardTitle>
                    <CardDescription className="text-[#98c7a5]">
                      Real launcher tools, live from PicoClaw.
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="border-[#2a6b3d] text-[#9fd8ae]">
                      {filteredToolCount} visible
                    </Badge>
                    <Button asChild variant="outline" className="border-[#2a6b3d] bg-transparent text-[#c8f5d2] hover:bg-[#12301a]">
                      <Link to="/agent/tools">
                        <IconSettings className="size-4" />
                        Full Tools
                      </Link>
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {hasToolsError ? (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
                    Failed to load tools.
                  </div>
                ) : isToolsLoading ? (
                  <div className="grid gap-3 md:grid-cols-2">
                    {Array.from({ length: 6 }).map((_, index) => (
                      <Skeleton key={index} className="h-36 rounded-xl bg-[#112117]" />
                    ))}
                  </div>
                ) : (
                  <div className="space-y-5">
                    {groupedTools.map(([category, items]) => (
                      <div key={category} className="space-y-3">
                        <div className="flex items-center justify-between border-b border-[#173621] pb-2">
                          <h2 className="font-mono text-xs uppercase tracking-[0.24em] text-[#8ec49c]">
                            {category}
                          </h2>
                          <span className="font-mono text-xs text-[#5f8f6f]">
                            {items.length}
                          </span>
                        </div>
                        <div className="grid gap-3 md:grid-cols-2">
                          {items.map((tool) => (
                            <Card
                              key={tool.name}
                              size="sm"
                              className="border-[#173621] bg-[#08100b] text-[#d7f9df] shadow-none"
                            >
                              <CardHeader className="gap-2">
                                <div className="flex items-start justify-between gap-3">
                                  <div className="space-y-2">
                                    <CardTitle className="font-mono text-sm">
                                      {tool.name}
                                    </CardTitle>
                                    <Badge
                                      variant={statusBadgeVariant(tool.status)}
                                      className="capitalize"
                                    >
                                      {tool.status}
                                    </Badge>
                                  </div>
                                  <Switch
                                    checked={tool.status !== "disabled"}
                                    disabled={pendingToolName === tool.name}
                                    onCheckedChange={(checked) =>
                                      toggleTool(tool.name, checked)
                                    }
                                  />
                                </div>
                                <CardDescription className="text-[#8fb59b]">
                                  {tool.description}
                                </CardDescription>
                              </CardHeader>
                              <CardContent className="space-y-3">
                                <div className="flex items-center justify-between text-xs text-[#739981]">
                                  <span className="capitalize">{tool.category}</span>
                                  <span className="font-mono">{tool.config_key}</span>
                                </div>
                                {tool.reason_code ? (
                                  <div className="rounded-lg border border-amber-400/20 bg-amber-400/10 px-3 py-2 text-xs text-amber-100">
                                    {reasonLabel(tool.reason_code)}
                                  </div>
                                ) : null}
                              </CardContent>
                            </Card>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>

            <Card className="border-[#215d36] bg-[linear-gradient(180deg,#09130b_0%,#071109_100%)] text-[#effff3] shadow-none">
              <CardHeader>
                <CardTitle className="font-mono text-sm uppercase tracking-[0.24em]">
                  Memory Network
                </CardTitle>
                <CardDescription className="text-[#98c7a5]">
                  Obsidian-style graph built from PicoClaw workspace memory and the active session trail.
                </CardDescription>
              </CardHeader>
              <CardContent>
                {hasMemoryGraphError ? (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
                    Failed to load memory graph.
                  </div>
                ) : isMemoryGraphLoading ? (
                  <Skeleton className="h-[420px] rounded-xl bg-[#112117]" />
                ) : sessionMemoryGraph && sessionMemoryGraph.nodes.length > 0 ? (
                  <MemoryGraph
                    nodes={sessionMemoryGraph.nodes}
                    edges={sessionMemoryGraph.edges}
                  />
                ) : (
                  <div className="rounded-lg border border-[#173621] bg-[#050d08] px-4 py-3 text-sm text-[#8bb39a]">
                    Memory graph will appear when the session and workspace memory have visible context.
                  </div>
                )}
              </CardContent>
            </Card>
          </section>

          <aside className="space-y-4">
            <Card className="border-[#1f4f31] bg-[#09150c] text-[#d7f9df] shadow-none">
              <CardHeader>
                <CardTitle className="font-mono text-sm uppercase tracking-[0.22em]">
                  Session Bridge
                </CardTitle>
                <CardDescription className="text-[#8bb39a]">
                  Voice and images route into the active Pico session.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="rounded-lg border border-[#173621] bg-[#050d08] px-3 py-2 text-xs text-[#9fc6aa]">
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-mono uppercase tracking-[0.18em]">
                      Session
                    </span>
                    <Badge variant="outline" className="border-[#2a6b3d] text-[#9fd8ae]">
                      {connectionState}
                    </Badge>
                  </div>
                  <p className="mt-2 break-all font-mono text-[11px] text-[#d7f9df]">
                    {activeSessionId}
                  </p>
                </div>

                <Input
                  value={prompt}
                  onChange={(event) => setPrompt(event.target.value)}
                  placeholder="Send a message to the active agent session"
                  className="border-[#1f4f31] bg-[#050d08] text-[#d7f9df] placeholder:text-[#5c8168]"
                />

                {attachments.length > 0 ? (
                  <div className="space-y-2">
                    {attachments.map((attachment, index) => (
                      <div
                        key={`${attachment.filename ?? "attachment"}-${index}`}
                        className="flex items-center justify-between rounded-lg border border-[#173621] bg-[#050d08] px-3 py-2 text-xs text-[#bfe7c8]"
                      >
                        <span className="truncate">{attachment.filename ?? "Image"}</span>
                        <button
                          type="button"
                          className="text-[#7ca88a] hover:text-[#effff3]"
                          onClick={() =>
                            setAttachments((current) =>
                              current.filter((_, itemIndex) => itemIndex !== index),
                            )
                          }
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>
                ) : null}

                <div className="grid grid-cols-2 gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    className="border-[#2a6b3d] bg-transparent text-[#d7f9df] hover:bg-[#12301a]"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <IconPhoto className="size-4" />
                    Add Image
                  </Button>
                  <Button
                    type="button"
                    variant={isListening ? "destructive" : "outline"}
                    className={cn(
                      "border-[#2a6b3d] text-[#d7f9df] hover:bg-[#12301a]",
                      isListening && "border-red-400/40 bg-red-500/10 text-red-100",
                    )}
                    onClick={toggleVoice}
                  >
                    {isListening ? (
                      <IconMicrophoneOff className="size-4" />
                    ) : (
                      <IconMicrophone className="size-4" />
                    )}
                    Voice
                  </Button>
                </div>

                <Button
                  type="button"
                  className="w-full bg-[#4fc267] text-[#08120b] hover:bg-[#77e58e]"
                  onClick={handleSendBridgeMessage}
                >
                  <IconUpload className="size-4" />
                  Send To Active Session
                </Button>

                <div className="grid grid-cols-2 gap-2">
                  <Button asChild variant="outline" className="border-[#2a6b3d] bg-transparent text-[#d7f9df] hover:bg-[#12301a]">
                    <Link to="/">
                      Open Chat
                      <IconArrowRight className="size-4" />
                    </Link>
                  </Button>
                  <Button asChild variant="outline" className="border-[#2a6b3d] bg-transparent text-[#d7f9df] hover:bg-[#12301a]">
                    <Link to="/agent/hub">
                      Skills Hub
                      <IconArrowRight className="size-4" />
                    </Link>
                  </Button>
                </div>

                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
                  className="hidden"
                  multiple
                  onChange={handleImageSelection}
                />
              </CardContent>
            </Card>

            <Card className="border-[#1f4f31] bg-[#09150c] text-[#d7f9df] shadow-none">
              <CardHeader>
                <CardTitle className="font-mono text-sm uppercase tracking-[0.22em]">
                  Main Agent Subagents
                </CardTitle>
                <CardDescription className="text-[#8bb39a]">
                  Live status for the current Pico session only.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                {hasSubagentsError ? (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                    Failed to load subagent status.
                  </div>
                ) : isSubagentsLoading ? (
                  Array.from({ length: 3 }).map((_, index) => (
                    <Skeleton key={index} className="h-20 rounded-xl bg-[#112117]" />
                  ))
                ) : sessionSubagents.length === 0 ? (
                  <div className="rounded-lg border border-[#173621] bg-[#050d08] px-3 py-3 text-sm text-[#8bb39a]">
                    No subagents have been created in this session yet.
                  </div>
                ) : (
                  sessionSubagents.map((task) => (
                    <div
                      key={task.id}
                      className="rounded-xl border border-[#173621] bg-[#050d08] px-3 py-3"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <p className="font-mono text-sm text-[#effff3]">
                            {task.label || task.id}
                          </p>
                          <p className="text-[11px] text-[#7ca88a]">
                            {dayjs(task.created).format("MMM D, HH:mm")}
                          </p>
                        </div>
                        <Badge variant={statusBadgeVariant(task.status)} className="capitalize">
                          {task.status}
                        </Badge>
                      </div>
                      {task.result ? (
                        <p className="mt-3 text-xs leading-relaxed text-[#9dc2a7]">
                          {task.result}
                        </p>
                      ) : null}
                    </div>
                  ))
                )}
              </CardContent>
            </Card>

            <Card className="border-[#1f4f31] bg-[#09150c] text-[#d7f9df] shadow-none">
              <CardHeader>
                <CardTitle className="font-mono text-sm uppercase tracking-[0.22em]">
                  Tool Runtime
                </CardTitle>
                <CardDescription className="text-[#8bb39a]">
                  Quick view of the current search/tool setup.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-sm text-[#a8cfb3]">
                <div className="flex items-center justify-between rounded-lg border border-[#173621] bg-[#050d08] px-3 py-3">
                  <span className="flex items-center gap-2">
                    <IconBrain className="size-4 text-[#71d888]" />
                    Web search provider
                  </span>
                  <span className="font-mono text-xs uppercase text-[#effff3]">
                    {isWebSearchLoading ? "Loading" : currentProviderLabel}
                  </span>
                </div>
                <p className="text-xs text-[#7ca88a]">
                  The cockpit reuses PicoClaw’s existing Pico session, tool
                  toggles, and media pipeline instead of the source app’s
                  Firebase and Gemini-specific wiring.
                </p>
              </CardContent>
            </Card>
          </aside>
        </div>
      </div>
    </div>
  )
}
