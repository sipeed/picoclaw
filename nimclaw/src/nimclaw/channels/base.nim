import std/[asyncdispatch, strutils, tables]
import ../bus, ../bus_types
import ../services/voice

type
  Channel* = ref object of RootObj

method name*(c: Channel): string {.base.} = ""
method start*(c: Channel): Future[void] {.base, async.} = discard
method stop*(c: Channel): Future[void] {.base, async.} = discard
method send*(c: Channel, msg: OutboundMessage): Future[void] {.base, async.} = discard
method isRunning*(c: Channel): bool {.base.} = false
method isAllowed*(c: Channel, senderID: string): bool {.base.} = true
method setTranscriber*(c: Channel, transcriber: GroqTranscriber) {.base.} = discard

type
  BaseChannel* = ref object of Channel
    bus*: MessageBus
    running*: bool
    name*: string
    allowList*: seq[string]

proc newBaseChannel*(name: string, bus: MessageBus, allowList: seq[string]): BaseChannel =
  BaseChannel(
    bus: bus,
    name: name,
    allowList: allowList,
    running: false
  )

method name*(c: BaseChannel): string = c.name
method isRunning*(c: BaseChannel): bool = c.running

method isAllowed*(c: BaseChannel, senderID: string): bool =
  if c.allowList.len == 0: return true
  for allowed in c.allowList:
    if senderID == allowed: return true
  return false

proc handleMessage*(c: BaseChannel, senderID, chatID, content: string, media: seq[string] = @[], metadata: Table[string, string] = initTable[string, string]()) =
  if not c.isAllowed(senderID): return

  let sessionKey = c.name & ":" & chatID
  let msg = InboundMessage(
    channel: c.name,
    sender_id: senderID,
    chat_id: chatID,
    content: content,
    media: media,
    session_key: sessionKey,
    metadata: metadata
  )
  c.bus.publishInbound(msg)
