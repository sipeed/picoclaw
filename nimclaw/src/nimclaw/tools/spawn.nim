import std/[asyncdispatch, json, tables, strutils]
import types
import subagent

type
  SpawnTool* = ref object of ContextualTool
    manager*: SubagentManager
    originChannel*: string
    originChatID*: string

proc newSpawnTool*(manager: SubagentManager): SpawnTool =
  SpawnTool(
    manager: manager,
    originChannel: "cli",
    originChatID: "direct"
  )

method name*(t: SpawnTool): string = "spawn"
method description*(t: SpawnTool): string = "Spawn a subagent to handle a task in the background. Use this for complex or time-consuming tasks that can run independently. The subagent will complete the task and report back when done."
method parameters*(t: SpawnTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "task": {
        "type": "string",
        "description": "The task for subagent to complete"
      },
      "label": {
        "type": "string",
        "description": "Optional short label for the task (for display)"
      }
    },
    "required": %["task"]
  }.toTable

method setContext*(t: SpawnTool, channel, chatID: string) =
  t.originChannel = channel
  t.originChatID = chatID

method execute*(t: SpawnTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("task"): return "Error: task is required"
  let task = args["task"].getStr()
  let label = if args.hasKey("label"): args["label"].getStr() else: ""

  if t.manager == nil:
    return "Error: Subagent manager not configured"

  return t.manager.spawn(task, label, t.originChannel, t.originChatID)
