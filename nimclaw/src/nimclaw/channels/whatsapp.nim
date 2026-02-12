import std/[asyncdispatch, tables, strutils, json, locks]
import ws
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils

type
  WhatsAppChannel* = ref object of BaseChannel
    conn: WebSocket
    url: string

proc newWhatsAppChannel*(cfg: WhatsAppConfig, bus: MessageBus): WhatsAppChannel =
  let base = newBaseChannel("whatsapp", bus, cfg.allow_from)
  WhatsAppChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    url: cfg.bridge_url
  )

method name*(c: WhatsAppChannel): string = "whatsapp"

proc listen(c: WhatsAppChannel) {.async.} =
  while c.running:
    try:
      let data = await c.conn.receiveStrPacket()
      if data == "": break
      let msg = parseJson(data)
      if msg.getOrDefault("type").getStr() == "message":
        let senderID = msg["from"].getStr()
        let chatID = msg.getOrDefault("chat").getStr(senderID)
        let content = msg.getOrDefault("content").getStr("")

        var metadata = initTable[string, string]()
        if msg.hasKey("id"): metadata["message_id"] = msg["id"].getStr()
        if msg.hasKey("from_name"): metadata["user_name"] = msg["from_name"].getStr()

        c.handleMessage(senderID, chatID, content, @[], metadata)
    except Exception as e:
      errorCF("whatsapp", "WhatsApp read error", {"error": e.msg}.toTable)
      await sleepAsync(2000)

method start*(c: WhatsAppChannel) {.async.} =
  infoC("whatsapp", "Starting WhatsApp channel connecting to " & c.url)
  try:
    c.conn = await newWebSocket(c.url)
    c.running = true
    discard listen(c)
    infoC("whatsapp", "WhatsApp channel connected")
  except Exception as e:
    errorCF("whatsapp", "Failed to connect to WhatsApp bridge", {"error": e.msg}.toTable)

method stop*(c: WhatsAppChannel) {.async.} =
  c.running = false
  if c.conn != nil: c.conn.close()

method send*(c: WhatsAppChannel, msg: OutboundMessage) {.async.} =
  if c.conn == nil: return
  let payload = %*{"type": "message", "to": msg.chat_id, "content": msg.content}
  try:
    await c.conn.send($payload)
  except Exception as e:
    errorCF("whatsapp", "Failed to send WhatsApp message", {"error": e.msg}.toTable)

method isRunning*(c: WhatsAppChannel): bool = c.running
