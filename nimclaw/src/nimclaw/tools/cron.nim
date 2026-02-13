import std/[asyncdispatch, json, tables, strutils, times, locks, options]
import types
import ../services/cron as cron_service
import ../bus
import ../bus_types
import ../utils

type
  JobExecutor* = proc (content, sessionKey, channel, chatID: string): Future[string] {.async.}

  CronTool* = ref object of ContextualTool
    cronService*: CronService
    executor*: JobExecutor
    msgBus*: MessageBus
    channel*: string
    chatID*: string
    lock*: Lock

proc newCronTool*(cronService: CronService, executor: JobExecutor, msgBus: MessageBus): CronTool =
  var ct = CronTool(
    cronService: cronService,
    executor: executor,
    msgBus: msgBus
  )
  initLock(ct.lock)
  return ct

method name*(t: CronTool): string = "cron"
method description*(t: CronTool): string = "Schedule reminders and tasks. IMPORTANT: When user asks to be reminded or scheduled, you MUST call this tool. Use 'at_seconds' for one-time reminders (e.g., 'remind me in 10 minutes' → at_seconds=600). Use 'every_seconds' ONLY for recurring tasks (e.g., 'every 2 hours' → every_seconds=7200). Use 'cron_expr' for complex recurring schedules (e.g., '0 9 * * *' for daily at 9am)."
method parameters*(t: CronTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "action": {
        "type": "string",
        "enum": ["add", "list", "remove", "enable", "disable"],
        "description": "Action to perform. Use 'add' when user wants to schedule a reminder or task."
      },
      "message": {
        "type": "string",
        "description": "The reminder/task message to display when triggered (required for add)"
      },
      "at_seconds": {
        "type": "integer",
        "description": "One-time reminder: seconds from now when to trigger (e.g., 600 for 10 minutes later). Use this for one-time reminders like 'remind me in 10 minutes'."
      },
      "every_seconds": {
        "type": "integer",
        "description": "Recurring interval in seconds (e.g., 3600 for every hour). Use this ONLY for recurring tasks like 'every 2 hours' or 'daily reminder'."
      },
      "cron_expr": {
        "type": "string",
        "description": "Cron expression for complex recurring schedules (e.g., '0 9 * * *' for daily at 9am). Use this for complex recurring schedules."
      },
      "job_id": {
        "type": "string",
        "description": "Job ID (for remove/enable/disable)"
      },
      "deliver": {
        "type": "boolean",
        "description": "If true, send message directly to channel. If false, let agent process the message (for complex tasks). Default: true"
      }
    },
    "required": %["action"]
  }.toTable

method setContext*(t: CronTool, channel, chatID: string) =
  acquire(t.lock)
  t.channel = channel
  t.chatID = chatID
  release(t.lock)

proc addJob(t: CronTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  acquire(t.lock)
  let channel = t.channel
  let chatID = t.chatID
  release(t.lock)

  if channel == "" or chatID == "":
    return "Error: no session context (channel/chat_id not set). Use this tool in an active conversation."

  if not args.hasKey("message"): return "Error: message is required for add"
  let message = args["message"].getStr()

  var schedule: CronSchedule
  if args.hasKey("at_seconds"):
    let atSeconds = args["at_seconds"].getInt()
    let atMS = (getTime().toUnix * 1000) + (atSeconds * 1000)
    schedule = CronSchedule(kind: "at", atMs: some(atMS))
  elif args.hasKey("every_seconds"):
    let everySeconds = args["every_seconds"].getInt()
    let everyMS = everySeconds * 1000
    schedule = CronSchedule(kind: "every", everyMs: some(everyMS.int64))
  elif args.hasKey("cron_expr"):
    schedule = CronSchedule(kind: "cron", expr: args["cron_expr"].getStr())
  else:
    return "Error: one of at_seconds, every_seconds, or cron_expr is required"

  let deliver = if args.hasKey("deliver"): args["deliver"].getBool() else: true
  let messagePreview = truncate(message, 30)

  try:
    let job = await t.cronService.addJob(messagePreview, schedule, message, deliver, channel, chatID)
    return strutils.format("Created job '$1' (id: $2)", job.name, job.id)
  except Exception as e:
    return "Error adding job: " & e.msg

method execute*(t: CronTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("action"): return "Error: action is required"
  let action = args["action"].getStr()

  case action:
  of "add": return await t.addJob(args)
  of "list":
    let jobs = t.cronService.listJobs(false)
    if jobs.len == 0: return "No scheduled jobs."
    var res = "Scheduled jobs:\n"
    for j in jobs:
      var schedInfo = "unknown"
      if j.schedule.kind == "every" and j.schedule.everyMs.isSome:
        schedInfo = "every " & $(j.schedule.everyMs.get div 1000) & "s"
      elif j.schedule.kind == "cron":
        schedInfo = j.schedule.expr
      elif j.schedule.kind == "at":
        schedInfo = "one-time"
      res.add(strutils.format("- $1 (id: $2, $3)\n", j.name, j.id, schedInfo))
    return res
  of "remove":
    if not args.hasKey("job_id"): return "Error: job_id is required"
    let jobID = args["job_id"].getStr()
    if t.cronService.removeJob(jobID):
      return "Removed job " & jobID
    else:
      return "Job " & jobID & " not found"
  of "enable", "disable":
    if not args.hasKey("job_id"): return "Error: job_id is required"
    let jobID = args["job_id"].getStr()
    let enabled = action == "enable"
    let job = t.cronService.enableJob(jobID, enabled)
    if job == nil: return "Job " & jobID & " not found"
    let status = if enabled: "enabled" else: "disabled"
    return strutils.format("Job '$1' $2", job.name, status)
  else:
    return "Error: unknown action: " & action
