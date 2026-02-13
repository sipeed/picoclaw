import std/[os, json, strutils, asyncdispatch, tables, syncio, times, locks]
import ../bus, ../bus_types, ../config, ../logger, ../providers/types as providers_types, ../session, ../utils
import context as agent_context
import ../tools/registry as tools_registry
import ../tools/base as tools_base
import ../tools/[filesystem, edit, shell, spawn, subagent, web, cron as cron_tool, message]

type
  ProcessOptions* = object
    sessionKey*: string
    channel*: string
    chatID*: string
    userMessage*: string
    defaultResponse*: string
    enableSummary*: bool
    sendResponse*: bool

  AgentLoop* = ref object
    bus*: MessageBus
    provider*: LLMProvider
    workspace*: string
    model*: string
    contextWindow*: int
    maxIterations*: int
    sessions*: SessionManager
    contextBuilder*: ContextBuilder
    tools*: ToolRegistry
    running*: bool
    summarizing*: Table[string, bool]
    summarizingLock*: Lock

proc newAgentLoop*(cfg: Config, msgBus: MessageBus, provider: LLMProvider): AgentLoop =
  let workspace = cfg.workspacePath()
  if not dirExists(workspace):
    createDir(workspace)

  let toolsRegistry = newToolRegistry()

  # Register all tools faithfully as in Go
  toolsRegistry.register(ReadFileTool())
  toolsRegistry.register(WriteFileTool())
  toolsRegistry.register(ListDirTool())
  toolsRegistry.register(newExecTool(workspace))

  toolsRegistry.register(newWebSearchTool(cfg.tools.web.search.api_key, cfg.tools.web.search.max_results))
  toolsRegistry.register(newWebFetchTool(50000))

  let msgTool = newMessageTool()
  msgTool.setSendCallback(proc(channel, chatID, content: string): Future[void] {.async.} =
    msgBus.publishOutbound(OutboundMessage(channel: channel, chat_id: chatID, content: content))
  )
  toolsRegistry.register(msgTool)

  let subagentManager = newSubagentManager(provider, workspace, msgBus)
  toolsRegistry.register(newSpawnTool(subagentManager))

  toolsRegistry.register(newEditFileTool(workspace))
  toolsRegistry.register(newAppendFileTool())

  let sessionsManager = newSessionManager(workspace / "sessions")
  let contextBuilder = newContextBuilder(workspace)
  contextBuilder.setToolsRegistry(toolsRegistry)

  var al = AgentLoop(
    bus: msgBus,
    provider: provider,
    workspace: workspace,
    model: cfg.agents.defaults.model,
    contextWindow: cfg.agents.defaults.max_tokens,
    maxIterations: cfg.agents.defaults.max_tool_iterations,
    sessions: sessionsManager,
    contextBuilder: contextBuilder,
    tools: toolsRegistry,
    running: false,
    summarizing: initTable[string, bool]()
  )
  initLock(al.summarizingLock)
  return al

proc stop*(al: AgentLoop) =
  al.running = false

proc registerTool*(al: AgentLoop, tool: Tool) =
  al.tools.register(tool)

proc estimateTokens(messages: seq[providers_types.Message]): int =
  var total = 0
  for m in messages:
    total += m.content.len div 4
  return total

proc summarizeBatch(al: AgentLoop, batch: seq[providers_types.Message], existingSummary: string): Future[string] {.async.} =
  var prompt = "Provide a concise summary of this conversation segment, preserving core context and key points.\n"
  if existingSummary != "":
    prompt.add("Existing context: " & existingSummary & "\n")
  prompt.add("\nCONVERSATION:\n")
  for m in batch:
    prompt.add(m.role & ": " & m.content & "\n")

  let response = await al.provider.chat(@[providers_types.Message(role: "user", content: prompt)], @[], al.model, initTable[string, JsonNode]())
  return response.content

proc summarizeSession(al: AgentLoop, sessionKey: string) {.async.} =
  let history = al.sessions.getHistory(sessionKey)
  let summary = al.sessions.getSummary(sessionKey)

  if history.len <= 4: return
  let toSummarize = history[0 .. ^5]

  # Oversized Message Guard
  let maxMessageTokens = al.contextWindow div 2
  var validMessages: seq[providers_types.Message] = @[]
  for m in toSummarize:
    if m.role == "user" or m.role == "assistant":
      if (m.content.len div 4) < maxMessageTokens:
        validMessages.add(m)

  if validMessages.len == 0: return

  let finalSummary = await al.summarizeBatch(validMessages, summary)

  if finalSummary != "":
    al.sessions.setSummary(sessionKey, finalSummary)
    al.sessions.truncateHistory(sessionKey, 4)
    al.sessions.save(al.sessions.getOrCreate(sessionKey))

proc maybeSummarize(al: AgentLoop, sessionKey: string) =
  acquire(al.summarizingLock)
  if al.summarizing.hasKey(sessionKey) and al.summarizing[sessionKey]:
    release(al.summarizingLock)
    return

  let history = al.sessions.getHistory(sessionKey)
  let tokenEstimate = estimateTokens(history)
  let threshold = (al.contextWindow * 75) div 100

  if history.len > 20 or tokenEstimate > threshold:
    al.summarizing[sessionKey] = true
    release(al.summarizingLock)
    discard (proc() {.async.} =
      await summarizeSession(al, sessionKey)
      acquire(al.summarizingLock)
      al.summarizing[sessionKey] = false
      release(al.summarizingLock)
    )()
  else:
    release(al.summarizingLock)

proc runLLMIteration(al: AgentLoop, messages: seq[providers_types.Message], opts: ProcessOptions): Future[(string, int, seq[providers_types.Message])] {.async.} =
  var iteration = 0
  var finalContent = ""
  var currentMessages = messages

  while iteration < al.maxIterations:
    iteration += 1
    debugCF("agent", "LLM iteration", {"iteration": $iteration, "max": $al.maxIterations}.toTable)

    let toolDefs = al.tools.getDefinitions()
    let response = await al.provider.chat(currentMessages, toolDefs, al.model, initTable[string, JsonNode]())

    if response.tool_calls.len == 0:
      finalContent = response.content
      infoCF("agent", "LLM response without tool calls", {"iteration": $iteration}.toTable)
      break

    var assistantMsg = providers_types.Message(role: "assistant", content: response.content, tool_calls: response.tool_calls)
    currentMessages.add(assistantMsg)
    al.sessions.addFullMessage(opts.sessionKey, assistantMsg)

    for tc in response.tool_calls:
      infoCF("agent", "Tool call: " & tc.name, {"tool": tc.name, "iteration": $iteration}.toTable)
      let result = await al.tools.executeWithContext(tc.name, tc.arguments, opts.channel, opts.chatID)
      let toolResultMsg = providers_types.Message(role: "tool", content: result, tool_call_id: tc.id)
      currentMessages.add(toolResultMsg)
      al.sessions.addFullMessage(opts.sessionKey, toolResultMsg)

  return (finalContent, iteration, currentMessages)

proc runAgentLoop*(al: AgentLoop, opts: ProcessOptions): Future[string] {.async.} =
  let history = al.sessions.getHistory(opts.sessionKey)
  let summary = al.sessions.getSummary(opts.sessionKey)
  var messages = al.contextBuilder.buildMessages(history, summary, opts.userMessage, opts.channel, opts.chatID)

  al.sessions.addMessage(opts.sessionKey, "user", opts.userMessage)

  let (finalContentRaw, iteration, _) = await al.runLLMIteration(messages, opts)
  var finalContent = finalContentRaw

  if finalContent == "":
    finalContent = opts.defaultResponse

  al.sessions.addMessage(opts.sessionKey, "assistant", finalContent)
  al.sessions.save(al.sessions.getOrCreate(opts.sessionKey))

  if opts.enableSummary:
    al.maybeSummarize(opts.sessionKey)

  if opts.sendResponse:
    al.bus.publishOutbound(OutboundMessage(channel: opts.channel, chat_id: opts.chatID, content: finalContent))

  infoCF("agent", "Response: " & truncate(finalContent, 120), {"session_key": opts.sessionKey, "iterations": $iteration}.toTable)
  return finalContent

proc processMessage*(al: AgentLoop, msg: InboundMessage): Future[string] {.async.} =
  infoCF("agent", "Processing message from " & msg.channel & ":" & msg.sender_id, {"session_key": msg.session_key}.toTable)

  # update tool contexts
  let (toolMsg, okMsg) = al.tools.get("message")
  if okMsg:
    if toolMsg of MessageTool: cast[MessageTool](toolMsg).setContext(msg.channel, msg.chat_id)
  let (toolSpawn, okSpawn) = al.tools.get("spawn")
  if okSpawn:
    if toolSpawn of SpawnTool: cast[SpawnTool](toolSpawn).setContext(msg.channel, msg.chat_id)
  let (toolCron, okCron) = al.tools.get("cron")
  if okCron:
    if toolCron of CronTool: cast[CronTool](toolCron).setContext(msg.channel, msg.chat_id)

  if msg.channel == "system":
    # logic for system messages...
    return ""

  return await al.runAgentLoop(ProcessOptions(
    sessionKey: msg.session_key,
    channel: msg.channel,
    chatID: msg.chat_id,
    userMessage: msg.content,
    defaultResponse: "I've completed processing but have no response to give.",
    enableSummary: true,
    sendResponse: false
  ))

proc processDirect*(al: AgentLoop, content, sessionKey: string): Future[string] {.async.} =
  let msg = InboundMessage(channel: "cli", sender_id: "user", chat_id: "direct", content: content, session_key: sessionKey)
  return await al.processMessage(msg)

proc run*(al: AgentLoop) {.async.} =
  al.running = true
  while al.running:
    let msg = await al.bus.consumeInbound()
    let response = await al.processMessage(msg)
    if response != "":
      al.bus.publishOutbound(OutboundMessage(channel: msg.channel, chat_id: msg.chat_id, content: response))

proc getStartupInfo*(al: AgentLoop): Table[string, JsonNode] =
  var info = initTable[string, JsonNode]()
  info["tools"] = %*{"count": al.tools.list().len, "names": al.tools.list()}
  info["skills"] = %al.contextBuilder.getSkillsInfo()
  return info
