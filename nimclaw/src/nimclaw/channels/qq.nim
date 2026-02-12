import std/[asyncdispatch, httpclient, json, strutils, tables, locks, times]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils
import ws

type
  QQChannel* = ref object of BaseChannel
    appID: string
    appSecret: string
    lock: Lock
    ws: WebSocket
    processedIDs: Table[string, bool]

proc newQQChannel*(cfg: QQConfig, bus: MessageBus): QQChannel =
  let base = newBaseChannel("qq", bus, cfg.allow_from)
  var qc = QQChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    appID: cfg.app_id,
    appSecret: cfg.app_secret,
    processedIDs: initTable[string, bool]()
  )
  initLock(qc.lock)
  return qc

method name*(c: QQChannel): string = "qq"

method start*(c: QQChannel) {.async.} =
  infoC("qq", "Starting QQ Bot channel (WebSocket mode)...")
  c.running = true
  warnC("qq", "QQ Bot WebSocket implementation requires OpenAPI access tokens.")

method stop*(c: QQChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: QQChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  infoCF("qq", "Sending QQ message", {"chat_id": msg.chat_id}.toTable)

method isRunning*(c: QQChannel): bool = c.running
