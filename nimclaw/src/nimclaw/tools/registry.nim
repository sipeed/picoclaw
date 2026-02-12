import std/[asyncdispatch, tables, json, locks, times, strutils]
import types
import ../logger
import ../providers/types as providers_types

type
  ToolRegistry* = ref object
    tools: Table[string, Tool]
    lock: Lock

proc newToolRegistry*(): ToolRegistry =
  var tr = ToolRegistry(tools: initTable[string, Tool]())
  initLock(tr.lock)
  return tr

proc register*(r: ToolRegistry, tool: Tool) =
  acquire(r.lock)
  defer: release(r.lock)
  r.tools[tool.name()] = tool

proc get*(r: ToolRegistry, name: string): (Tool, bool) =
  acquire(r.lock)
  defer: release(r.lock)
  if r.tools.hasKey(name):
    return (r.tools[name], true)
  else:
    return (nil, false)

proc list*(r: ToolRegistry): seq[string] =
  acquire(r.lock)
  defer: release(r.lock)
  for k in r.tools.keys:
    result.add(k)

proc count*(r: ToolRegistry): int =
  acquire(r.lock)
  defer: release(r.lock)
  r.tools.len

proc getSummaries*(r: ToolRegistry): seq[string] =
  acquire(r.lock)
  defer: release(r.lock)
  for tool in r.tools.values:
    result.add("- `" & tool.name() & "` - " & tool.description())

proc toolToSchema*(tool: Tool): ToolDefinition =
  ToolDefinition(
    `type`: "function",
    function: ToolFunctionDefinition(
      name: tool.name(),
      description: tool.description(),
      parameters: tool.parameters()
    )
  )

proc getDefinitions*(r: ToolRegistry): seq[ToolDefinition] =
  acquire(r.lock)
  defer: release(r.lock)
  for tool in r.tools.values:
    result.add(toolToSchema(tool))

proc executeWithContext*(r: ToolRegistry, name: string, args: Table[string, JsonNode], channel, chatID: string): Future[string] {.async.} =
  infoCF("tool", "Tool execution started", {"tool": name, "args": $args}.toTable)

  let (tool, ok) = r.get(name)
  if not ok:
    errorCF("tool", "Tool not found", {"tool": name}.toTable)
    return "Error: tool '" & name & "' not found"

  if tool of ContextualTool and channel != "" and chatID != "":
    (cast[ContextualTool](tool)).setContext(channel, chatID)

  let start = now()
  var result = ""
  try:
    result = await tool.execute(args)
  except Exception as e:
    let duration = (now() - start).inMilliseconds
    errorCF("tool", "Tool execution failed", {"tool": name, "duration": $duration, "error": e.msg}.toTable)
    return "Error: " & e.msg

  let duration = (now() - start).inMilliseconds
  infoCF("tool", "Tool execution completed", {"tool": name, "duration_ms": $duration, "result_length": $result.len}.toTable)
  return result
