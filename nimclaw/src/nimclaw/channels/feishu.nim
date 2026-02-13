import std/[asyncdispatch, httpclient, json, strutils, tables, locks, times]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils
import ws

type
  FeishuChannel* = ref object of BaseChannel
    appID: string
    appSecret: string
    token: string
    ws: WebSocket

proc newFeishuChannel*(cfg: FeishuConfig, bus: MessageBus): FeishuChannel =
  let base = newBaseChannel("feishu", bus, cfg.allow_from)
  return FeishuChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    appID: cfg.app_id,
    appSecret: cfg.app_secret
  )

proc getTenantAccessToken(c: FeishuChannel) {.async.} =
  let client = newAsyncHttpClient()
  let url = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
  let payload = %*{"app_id": c.appID, "app_secret": c.appSecret}
  try:
    let response = await client.post(url, $payload)
    let body = await response.body
    let res = parseJson(body)
    if res.hasKey("tenant_access_token"):
      c.token = res["tenant_access_token"].getStr()
      infoC("feishu", "Obtained Feishu tenant access token")
    else:
      errorCF("feishu", "Failed to get token", {"response": body}.toTable)
  except Exception as e:
    errorCF("feishu", "Auth error", {"error": e.msg}.toTable)
  finally:
    client.close()

proc feishuGatewayLoop(c: FeishuChannel) {.async.} =
  while c.running:
    try:
      if c.ws == nil:
        await sleepAsync(5000)
        continue
      let data = await c.ws.receiveStrPacket()
      if data == "": break
      let msg = parseJson(data)

      # Handle Feishu WebSocket events
      if msg.hasKey("header") and msg["header"].hasKey("event_type"):
        let eventType = msg["header"]["event_type"].getStr()
        if eventType == "im.message.receive_v1":
          let event = msg["event"]
          let sender = event["sender"]
          let message = event["message"]

          let chatID = message["chat_id"].getStr()
          let senderID = if sender.hasKey("sender_id"):
                           sender["sender_id"].getOrDefault("open_id").getStr()
                         else: "unknown"

          var content = ""
          if message["msg_type"].getStr() == "text":
            let contentJson = parseJson(message["content"].getStr())
            content = contentJson["text"].getStr()
          else:
            content = "[Non-text message]"

          infoCF("feishu", "Received message", {"sender": senderID}.toTable)
          c.handleMessage(senderID, chatID, content)

    except Exception as e:
      errorCF("feishu", "Gateway error", {"error": e.msg}.toTable)
      await sleepAsync(5000)

method name*(c: FeishuChannel): string = "feishu"

method start*(c: FeishuChannel) {.async.} =
  if c.appID == "" or c.appSecret == "": return
  infoC("feishu", "Starting Feishu channel (WS mode)...")
  await c.getTenantAccessToken()

  let client = newAsyncHttpClient()
  client.headers["Authorization"] = "Bearer " & c.token
  try:
    # Simplified Lark WS handshake
    let url = "https://open.feishu.cn/open-apis/ws/v1/endpoint"
    let response = await client.post(url, "")
    let body = await response.body
    let res = parseJson(body)
    if res.hasKey("data") and res["data"].hasKey("url"):
      let wsUrl = res["data"]["url"].getStr()
      c.ws = await newWebSocket(wsUrl)
      c.running = true
      discard feishuGatewayLoop(c)
      infoC("feishu", "Feishu connected via WebSocket")
    else:
      c.running = true
      infoC("feishu", "Feishu started in send-only mode (WS failed)")
  except Exception as e:
    errorCF("feishu", "WS handshake failed", {"error": e.msg}.toTable)
    c.running = true
  finally:
    client.close()

method stop*(c: FeishuChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: FeishuChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  let client = newAsyncHttpClient()
  client.headers["Authorization"] = "Bearer " & c.token
  client.headers["Content-Type"] = "application/json"
  let url = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
  let payload = %*{
    "receive_id": msg.chat_id,
    "msg_type": "text",
    "content": $ %*{"text": msg.content}
  }
  try:
    let resp = await client.post(url, $payload)
    if not resp.status.startsWith("200"):
      let body = await resp.body
      errorCF("feishu", "Send failed", {"status": resp.status, "response": body}.toTable)
  except Exception as e:
    errorCF("feishu", "Send error", {"error": e.msg}.toTable)
  finally:
    client.close()

method isRunning*(c: FeishuChannel): bool = c.running
