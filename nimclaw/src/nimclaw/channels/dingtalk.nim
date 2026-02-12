import std/[asyncdispatch, httpclient, json, strutils, tables, locks, times]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils
import ws

type
  DingTalkChannel* = ref object of BaseChannel
    clientID: string
    clientSecret: string
    sessionWebhooks: Table[string, string]
    lock: Lock
    ws: WebSocket

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

proc dingtalkGatewayLoop(c: DingTalkChannel) {.async.} =
  while c.running:
    try:
      let data = await c.ws.receiveStrPacket()
      if data == "": break
      let msg = parseJson(data)
      # DingTalk stream protocol handling simplified
      if msg.hasKey("specversion") and msg.hasKey("type") and msg["type"].getStr() == "chat.chatbot.message":
        let dataModel = msg["data"]
        let content = dataModel["text"]["content"].getStr()
        let senderID = dataModel["senderStaffId"].getStr()
        let chatID = if dataModel["conversationType"].getStr() == "1": senderID else: dataModel["conversationId"].getStr()

        acquire(c.lock)
        c.sessionWebhooks[chatID] = dataModel["sessionWebhook"].getStr()
        release(c.lock)

        c.handleMessage(senderID, chatID, content)

    except Exception as e:
      errorCF("dingtalk", "Gateway error", {"error": e.msg}.toTable)
      await sleepAsync(5000)

method name*(c: DingTalkChannel): string = "dingtalk"

method start*(c: DingTalkChannel) {.async.} =
  infoC("dingtalk", "Starting DingTalk channel (Stream Mode)...")
  # To implement DingTalk stream properly we'd need to get a gateway URL first
  # For now we'll simulate the connection if we have a valid mock/known URL or just log a warning
  c.running = true
  warnC("dingtalk", "DingTalk stream protocol requires specific gateway URL discovery.")

method stop*(c: DingTalkChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: DingTalkChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  acquire(c.lock)
  let hasWebhook = c.sessionWebhooks.hasKey(msg.chat_id)
  let webhook = if hasWebhook: c.sessionWebhooks[msg.chat_id] else: ""
  release(c.lock)

  if webhook == "": return

  let client = newAsyncHttpClient()
  client.headers["Content-Type"] = "application/json"
  let payload = %*{"msgtype": "markdown", "markdown": {"title": "PicoClaw", "text": msg.content}}
  try: discard await client.post(webhook, $payload)
  except: discard
  finally: client.close()

method isRunning*(c: DingTalkChannel): bool = c.running
