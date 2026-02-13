import std/[os, json, times, strutils, tables, locks, asyncdispatch, options, sequtils]
import jsony

type
  CronSchedule* = object
    kind*: string
    atMs*: Option[int64]
    everyMs*: Option[int64]
    expr*: string
    tz*: string

  CronPayload* = object
    kind*: string
    message*: string
    deliver*: bool
    channel*: string
    to*: string

  CronJobState* = object
    nextRunAtMs*: Option[int64]
    lastRunAtMs*: Option[int64]
    lastStatus*: string
    lastError*: string

  CronJob* = ref object
    id*: string
    name*: string
    enabled*: bool
    schedule*: CronSchedule
    payload*: CronPayload
    state*: CronJobState
    createdAtMs*: int64
    updatedAtMs*: int64
    deleteAfterRun*: bool

  CronStore* = object
    version*: int
    jobs*: seq[CronJob]

  JobHandler* = proc (job: CronJob): Future[(string, string)] {.async.}

  CronService* = ref object
    storePath*: string
    store*: CronStore
    onJob*: JobHandler
    lock*: Lock
    running*: bool

proc computeNextRun(cs: CronService, schedule: CronSchedule, nowMS: int64): Option[int64] =
  if schedule.kind == "at":
    if schedule.atMs.isSome and schedule.atMs.get > nowMS:
      return schedule.atMs
    return none(int64)

  if schedule.kind == "every":
    if schedule.everyMs.isNone or schedule.everyMs.get <= 0:
      return none(int64)
    return some(nowMS + schedule.everyMs.get)

  if schedule.kind == "cron":
    # Very simple placeholder for cron expression parsing
    # In a full implementation, we'd use a cron parser lib
    return some(nowMS + 3600000) # Default to 1 hour if expr is set but unparsed

  return none(int64)

proc saveStoreUnsafe(cs: CronService) =
  let dir = parentDir(cs.storePath)
  if dir != "" and not dirExists(dir):
    createDir(dir)
  writeFile(cs.storePath, cs.store.toJson())

proc loadStore(cs: CronService) =
  cs.store = CronStore(version: 1, jobs: @[])
  if fileExists(cs.storePath):
    try:
      let data = readFile(cs.storePath)
      cs.store = data.fromJson(CronStore)
    except:
      discard

proc newCronService*(storePath: string, onJob: JobHandler): CronService =
  var cs = CronService(
    storePath: storePath,
    onJob: onJob,
    running: false
  )
  initLock(cs.lock)
  cs.loadStore()
  return cs

proc addJob*(cs: CronService, name: string, schedule: CronSchedule, message: string, deliver: bool, channel, to: string): Future[CronJob] {.async.} =
  acquire(cs.lock)
  defer: release(cs.lock)

  let nowMS = getTime().toUnix * 1000
  let jobID = $nowMS # Simple ID

  var job = CronJob(
    id: jobID,
    name: name,
    enabled: true,
    schedule: schedule,
    payload: CronPayload(
      kind: "agent_turn",
      message: message,
      deliver: deliver,
      channel: channel,
      to: to
    ),
    state: CronJobState(
      nextRunAtMs: cs.computeNextRun(schedule, nowMS)
    ),
    createdAtMs: nowMS,
    updatedAtMs: nowMS,
    deleteAfterRun: (schedule.kind == "at")
  )

  cs.store.jobs.add(job)
  cs.saveStoreUnsafe()
  return job

proc listJobs*(cs: CronService, includeDisabled: bool): seq[CronJob] =
  acquire(cs.lock)
  defer: release(cs.lock)
  if includeDisabled: return cs.store.jobs
  var res: seq[CronJob] = @[]
  for j in cs.store.jobs:
    if j.enabled: res.add(j)
  return res

proc removeJob*(cs: CronService, jobID: string): bool =
  acquire(cs.lock)
  defer: release(cs.lock)
  let before = cs.store.jobs.len
  cs.store.jobs.keepIf(proc(j: CronJob): bool = j.id != jobID)
  let removed = cs.store.jobs.len < before
  if removed: cs.saveStoreUnsafe()
  return removed

proc enableJob*(cs: CronService, jobID: string, enabled: bool): CronJob =
  acquire(cs.lock)
  defer: release(cs.lock)
  for i in 0 ..< cs.store.jobs.len:
    if cs.store.jobs[i].id == jobID:
      cs.store.jobs[i].enabled = enabled
      cs.store.jobs[i].updatedAtMs = getTime().toUnix * 1000
      if enabled:
        cs.store.jobs[i].state.nextRunAtMs = cs.computeNextRun(cs.store.jobs[i].schedule, getTime().toUnix * 1000)
      else:
        cs.store.jobs[i].state.nextRunAtMs = none(int64)
      cs.saveStoreUnsafe()
      return cs.store.jobs[i]
  # Should really return option or throw
  return CronJob()

proc checkJobs(cs: CronService) {.async.} =
  while cs.running:
    let nowMS = getTime().toUnix * 1000
    var dueJobs: seq[CronJob] = @[]

    acquire(cs.lock)
    for i in 0 ..< cs.store.jobs.len:
      let job = cs.store.jobs[i]
      if job.enabled and job.state.nextRunAtMs.isSome and job.state.nextRunAtMs.get <= nowMS:
        dueJobs.add(job)
        cs.store.jobs[i].state.nextRunAtMs = none(int64)
    release(cs.lock)

    for job in dueJobs:
      if cs.onJob != nil:
        discard await cs.onJob(job)

      acquire(cs.lock)
      for i in 0 ..< cs.store.jobs.len:
        if cs.store.jobs[i].id == job.id:
          cs.store.jobs[i].state.lastRunAtMs = some(nowMS)
          if cs.store.jobs[i].schedule.kind == "at":
            if cs.store.jobs[i].deleteAfterRun:
              cs.store.jobs.delete(i)
              break
            else:
              cs.store.jobs[i].enabled = false
          else:
            cs.store.jobs[i].state.nextRunAtMs = cs.computeNextRun(cs.store.jobs[i].schedule, getTime().toUnix * 1000)
          break
      cs.saveStoreUnsafe()
      release(cs.lock)

    await sleepAsync(1000)

proc start*(cs: CronService) {.async.} =
  cs.running = true
  discard checkJobs(cs)

proc stop*(cs: CronService) =
  cs.running = false
