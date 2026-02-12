import std/[os, times, strutils, json, tables, locks]
import jsony
import providers/types as providers_types

type
  Session* = ref object
    key*: string
    messages*: seq[providers_types.Message]
    summary*: string
    created*: float64
    updated*: float64

  SessionManager* = ref object
    sessions*: Table[string, Session]
    lock*: Lock
    storage*: string

proc newSessionManager*(storage: string): SessionManager =
  result = SessionManager(
    sessions: initTable[string, Session](),
    storage: storage
  )
  initLock(result.lock)
  if storage != "":
    if not dirExists(storage):
      createDir(storage)
    # loadSessions would be here
    for file in walkFiles(storage / "*.json"):
      try:
        let data = readFile(file)
        let session = data.fromJson(Session)
        result.sessions[session.key] = session
      except:
        discard

proc getOrCreate*(sm: SessionManager, key: string): Session =
  acquire(sm.lock)
  defer: release(sm.lock)
  if sm.sessions.hasKey(key):
    return sm.sessions[key]
  else:
    let session = Session(
      key: key,
      messages: @[],
      created: getTime().toUnixFloat(),
      updated: getTime().toUnixFloat()
    )
    sm.sessions[key] = session
    return session

proc addFullMessage*(sm: SessionManager, sessionKey: string, msg: providers_types.Message) =
  acquire(sm.lock)
  defer: release(sm.lock)
  if not sm.sessions.hasKey(sessionKey):
    sm.sessions[sessionKey] = Session(
      key: sessionKey,
      messages: @[],
      created: getTime().toUnixFloat()
    )
  let session = sm.sessions[sessionKey]
  session.messages.add(msg)
  session.updated = getTime().toUnixFloat()

proc addMessage*(sm: SessionManager, sessionKey, role, content: string) =
  sm.addFullMessage(sessionKey, providers_types.Message(role: role, content: content))

proc getHistory*(sm: SessionManager, key: string): seq[providers_types.Message] =
  acquire(sm.lock)
  defer: release(sm.lock)
  if not sm.sessions.hasKey(key):
    return @[]
  return sm.sessions[key].messages

proc getSummary*(sm: SessionManager, key: string): string =
  acquire(sm.lock)
  defer: release(sm.lock)
  if not sm.sessions.hasKey(key):
    return ""
  return sm.sessions[key].summary

proc setSummary*(sm: SessionManager, key, summary: string) =
  acquire(sm.lock)
  defer: release(sm.lock)
  if sm.sessions.hasKey(key):
    sm.sessions[key].summary = summary
    sm.sessions[key].updated = getTime().toUnixFloat()

proc truncateHistory*(sm: SessionManager, key: string, keepLast: int) =
  acquire(sm.lock)
  defer: release(sm.lock)
  if not sm.sessions.hasKey(key): return
  let session = sm.sessions[key]
  if session.messages.len <= keepLast: return
  session.messages = session.messages[session.messages.len - keepLast .. ^1]
  session.updated = getTime().toUnixFloat()

proc save*(sm: SessionManager, session: Session) =
  if sm.storage == "": return
  acquire(sm.lock)
  defer: release(sm.lock)
  let path = sm.storage / (session.key & ".json")
  try:
    writeFile(path, session.toJson())
  except:
    discard
