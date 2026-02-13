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
      if c.ws == nil:
        await sleepAsync(5000)
        continue

      let data = await c.ws.receiveStrPacket()
      if data == "": break
      let msg = parseJson(data)

      # Simplified DingTalk Stream Mode handling
      if msg.getOrDefault("type").getStr() == "chat.chatbot.message":
        let dataModel = msg["data"]
        let content = dataModel["text"]["content"].getStr()
        let senderID = dataModel["senderStaffId"].getStr()
        let chatID = if dataModel["conversationType"].getStr() == "1": senderID else: dataModel["conversationId"].getStr()

        acquire(c.lock)
        c.sessionWebhooks[chatID] = dataModel["sessionWebhook"].getStr()
        release(c.lock)

        infoCF("dingtalk", "Received message", {"sender": senderID}.toTable)
        c.handleMessage(senderID, chatID, content)
    except Exception as e:
      errorCF("dingtalk", "Gateway error", {"error": e.msg}.toTable)
      await sleepAsync(5000)

method name*(c: DingTalkChannel): string = "dingtalk"

method start*(c: DingTalkChannel) {.async.} =
  if c.clientID == "" or c.clientSecret == "": return
  infoC("dingtalk", "Starting DingTalk channel...")

  # In a real implementation, we would perform OAuth and then connect to DingTalk's Stream Gateway.
  # Here we provide the structure to support it.
  c.running = true
  discard dingtalkGatewayLoop(c)
  infoC("dingtalk", "DingTalk channel started")

method stop*(c: DingTalkChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: DingTalkChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  acquire(c.lock)
  let hasWebhook = c.sessionWebhooks.hasKey(msg.chat_id)
  let webhook = if hasWebhook: c.sessionWebhooks[msg.chat_id] else: ""
  release(c.lock)

  if webhook == "":
    # Fallback to general DingTalk Bot API if session webhook is missing
    errorCF("dingtalk", "No session webhook for chat", {"chat_id": msg.chat_id}.toTable)
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
    let resp = await client.post(webhook, $payload)
    if not resp.status.startsWith("200"):
      let body = await resp.body
      errorCF("dingtalk", "Send failed", {"status": resp.status, "response": body}.toTable)
  except Exception as e:
    errorCF("dingtalk", "Send error", {"error": e.msg}.toTable)
  finally:
    client.close()

method isRunning*(c: DingTalkChannel): bool = c.running
