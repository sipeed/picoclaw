import std/[os, times, strutils, tables, locks, asyncdispatch]

type
  HeartbeatService* = ref object
    workspace*: string
    onHeartbeat*: proc (prompt: string): Future[void] {.async.}
    interval*: Duration
    enabled*: bool
    lock*: Lock
    running*: bool

proc newHeartbeatService*(workspace: string, onHeartbeat: proc (prompt: string): Future[void] {.async.}, intervalS: int, enabled: bool): HeartbeatService =
  var hs = HeartbeatService(
    workspace: workspace,
    onHeartbeat: onHeartbeat,
    interval: initDuration(seconds = intervalS),
    enabled: enabled,
    running: false
  )
  initLock(hs.lock)
  return hs

proc buildPrompt(hs: HeartbeatService): string =
  let notesFile = hs.workspace / "memory" / "HEARTBEAT.md"
  var notes = ""
  if fileExists(notesFile):
    notes = readFile(notesFile)

  let now = now().format("yyyy-MM-dd HH:mm")

  return """# Heartbeat Check

Current time: $1

Check if there are any tasks I should be aware of or actions I should take.
Review the memory file for any important updates or changes.
Be proactive in identifying potential issues or improvements.

$2
""".format(now, notes)

proc log(hs: HeartbeatService, message: string) =
  let logFile = hs.workspace / "memory" / "heartbeat.log"
  let timestamp = now().format("yyyy-MM-dd HH:mm:ss")
  try:
    let f = open(logFile, fmAppend)
    f.writeLine("[$1] $2".format(timestamp, message))
    f.close()
  except:
    discard

proc runLoop(hs: HeartbeatService) {.async.} =
  while hs.running:
    await sleepAsync(hs.interval.inMilliseconds.int)
    if not hs.enabled or not hs.running: continue

    let prompt = hs.buildPrompt()
    if hs.onHeartbeat != nil:
      try:
        await hs.onHeartbeat(prompt)
      except Exception as e:
        hs.log("Heartbeat error: " & e.msg)

proc start*(hs: HeartbeatService) {.async.} =
  if hs.running: return
  if not hs.enabled: return
  hs.running = true
  discard runLoop(hs)

proc stop*(hs: HeartbeatService) =
  hs.running = false
