import std/[asyncdispatch, tables, strutils, json, locks, os, httpclient]
import ws
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils

type
  DingTalkChannel* = ref object of BaseChannel
    clientID*: string
    clientSecret*: string
    sessionWebhooks*: Table[string, string]
    lock*: Lock

proc newDingTalkChannel*(cfg: DingTalkConfig, bus: MessageBus): DingTalkChannel =
  let base = newBaseChannel("dingtalk", bus, cfg.allow_from)
  var dc = DingTalkChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    clientID: cfg.client_id,
    clientSecret: cfg.client_secret,
    sessionWebhooks: initTable[string, string]()
  )
  initLock(dc.lock)
  return dc

method name*(c: DingTalkChannel): string = "dingtalk"

method start*(c: DingTalkChannel) {.async.} =
  infoC("dingtalk", "Starting DingTalk channel (Stream Mode)...")
  # Implementation would require DingTalk stream protocol handling
  c.running = true
  warnC("dingtalk", "DingTalk stream protocol not fully implemented in Nim yet.")

method stop*(c: DingTalkChannel) {.async.} =
  c.running = false

method send*(c: DingTalkChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return

  acquire(c.lock)
  let hasWebhook = c.sessionWebhooks.hasKey(msg.chat_id)
  let webhook = if hasWebhook: c.sessionWebhooks[msg.chat_id] else: ""
  release(c.lock)

  if webhook == "":
    errorCF("dingtalk", "No session webhook found for chat", {"chat_id": msg.chat_id}.toTable)
    return

  let client = newAsyncHttpClient()
  client.headers["Content-Type"] = "application/json"

  let payload = %*{
    "msgtype": "markdown",
    "markdown": {
      "title": "PicoClaw",
      "text": msg.content
    }
  }

  try:
    discard await client.post(webhook, $payload)
  except Exception as e:
    errorCF("dingtalk", "Failed to send DingTalk message", {"error": e.msg}.toTable)
  finally:
    client.close()

method isRunning*(c: DingTalkChannel): bool = c.running
