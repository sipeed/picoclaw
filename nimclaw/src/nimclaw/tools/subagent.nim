import std/[asyncdispatch, tables, locks, times, json, strutils]
import types
import ../providers/types as providers_types
import ../bus
import ../bus_types

type
  SubagentTask* = ref object
    id*: string
    task*: string
    label*: string
    originChannel*: string
    originChatID*: string
    status*: string
    result*: string
    created*: int64

  SubagentManager* = ref object
    tasks*: Table[string, SubagentTask]
    lock*: Lock
    provider*: providers_types.LLMProvider
    bus*: MessageBus
    workspace*: string
    nextID*: int

proc newSubagentManager*(provider: providers_types.LLMProvider, workspace: string, bus: MessageBus): SubagentManager =
  var sm = SubagentManager(
    tasks: initTable[string, SubagentTask](),
    provider: provider,
    bus: bus,
    workspace: workspace,
    nextID: 1
  )
  initLock(sm.lock)
  return sm

proc runTask*(sm: SubagentManager, task: SubagentTask) {.async.} =
  task.status = "running"
  task.created = getTime().toUnix * 1000

  let messages = @[
    providers_types.Message(role: "system", content: "You are a subagent. Complete the given task independently and report the result."),
    providers_types.Message(role: "user", content: task.task)
  ]

  try:
    let response = await sm.provider.chat(messages, @[], sm.provider.getDefaultModel(), initTable[string, JsonNode]())
    acquire(sm.lock)
    task.status = "completed"
    task.result = response.content
    release(sm.lock)
  except Exception as e:
    acquire(sm.lock)
    task.status = "failed"
    task.result = "Error: " & e.msg
    release(sm.lock)

  if sm.bus != nil:
    let announceContent = strutils.format("Task '$1' completed.\n\nResult:\n$2", task.label, task.result)
    sm.bus.publishInbound(InboundMessage(
      channel: "system",
      sender_id: "subagent:" & task.id,
      chat_id: task.originChannel & ":" & task.originChatID,
      content: announceContent
    ))

proc spawn*(sm: SubagentManager, task, label, originChannel, originChatID: string): string =
  acquire(sm.lock)
  let taskID = "subagent-" & $sm.nextID
  sm.nextID += 1

  let subagentTask = SubagentTask(
    id: taskID,
    task: task,
    label: label,
    originChannel: originChannel,
    originChatID: originChatID,
    status: "running",
    created: getTime().toUnix * 1000
  )
  sm.tasks[taskID] = subagentTask
  release(sm.lock)

  discard sm.runTask(subagentTask)

  if label != "":
    return "Spawned subagent '$1' for task: $2".format(label, task)
  return "Spawned subagent for task: $1".format(task)
