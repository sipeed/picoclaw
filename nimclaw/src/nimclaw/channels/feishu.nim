import std/[asyncdispatch, httpclient, json, strutils, tables, locks, times]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils
import ws

type
  FeishuChannel* = ref object of BaseChannel
    appID: string
    appSecret: string
    lock: Lock
    ws: WebSocket

proc newFeishuChannel*(cfg: FeishuConfig, bus: MessageBus): FeishuChannel =
  let base = newBaseChannel("feishu", bus, cfg.allow_from)
  var fc = FeishuChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    appID: cfg.app_id,
    appSecret: cfg.app_secret
  )
  initLock(fc.lock)
  return fc

method name*(c: FeishuChannel): string = "feishu"

method start*(c: FeishuChannel) {.async.} =
  infoC("feishu", "Starting Feishu channel (Long Connection Mode)...")
  c.running = true
  warnC("feishu", "Feishu WebSocket implementation requires tenant_access_token and gateway discovery.")

method stop*(c: FeishuChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: FeishuChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  infoCF("feishu", "Sending Feishu message", {"chat_id": msg.chat_id}.toTable)
  # Feishu requires complex auth (tenant_access_token)

method isRunning*(c: FeishuChannel): bool = c.running
