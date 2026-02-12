import std/[asyncdispatch, tables, strutils, json, locks]
import ws
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils

type
  WhatsAppChannel* = ref object of BaseChannel
    conn*: WebSocket
    url*: string
    lock*: Lock

proc newWhatsAppChannel*(cfg: WhatsAppConfig, bus: MessageBus): WhatsAppChannel =
  let base = newBaseChannel("whatsapp", bus, cfg.allow_from)
  var wc = WhatsAppChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    url: cfg.bridge_url
  )
  initLock(wc.lock)
  return wc

method name*(c: WhatsAppChannel): string = "whatsapp"

proc listen(c: WhatsAppChannel) {.async.} =
  while c.running:
    try:
      let data = await c.conn.receiveStrPacket()
      let msg = parseJson(data)
      if msg.hasKey("type") and msg["type"].getStr() == "message":
        let senderID = msg["from"].getStr()
        let chatID = msg.getOrDefault("chat").getStr(senderID)
        let content = msg.getOrDefault("content").getStr("")
        c.handleMessage(senderID, chatID, content)
    except Exception as e:
      errorCF("whatsapp", "WhatsApp read error", {"error": e.msg}.toTable)
      await sleepAsync(2000)

method start*(c: WhatsAppChannel) {.async.} =
  infoCF("whatsapp", "Starting WhatsApp channel connecting to $1...", {"url": c.url}.toTable)
  try:
    c.conn = await newWebSocket(c.url)
    c.running = true
    discard listen(c)
    infoC("whatsapp", "WhatsApp channel connected")
  except Exception as e:
    errorCF("whatsapp", "Failed to connect to WhatsApp bridge", {"error": e.msg}.toTable)

method stop*(c: WhatsAppChannel) {.async.} =
  c.running = false
  if c.conn != nil:
    c.conn.close()

method send*(c: WhatsAppChannel, msg: OutboundMessage) {.async.} =
  if c.conn == nil: return

  let payload = %*{
    "type": "message",
    "to": msg.chat_id,
    "content": msg.content
  }

  try:
    await c.conn.send($payload)
  except Exception as e:
    errorCF("whatsapp", "Failed to send WhatsApp message", {"error": e.msg}.toTable)

method isRunning*(c: WhatsAppChannel): bool = c.running
