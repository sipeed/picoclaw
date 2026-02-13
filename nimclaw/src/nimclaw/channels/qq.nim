import std/[asyncdispatch, httpclient, json, strutils, tables, locks, times]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils
import ws

type
  QQChannel* = ref object of BaseChannel
    appID: string
    appSecret: string
    token: string
    ws: WebSocket
    processedIDs: Table[string, bool]
    lock: Lock

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

proc getAccessToken(c: QQChannel) {.async.} =
  let client = newAsyncHttpClient()
  let url = "https://bots.qq.com/app/getAppAccessToken"
  let payload = %*{"appId": c.appID, "clientSecret": c.appSecret}
  try:
    let response = await client.post(url, $payload)
    let body = await response.body
    let res = parseJson(body)
    if res.hasKey("access_token"):
      c.token = res["access_token"].getStr()
      infoC("qq", "Obtained QQ access token")
    else:
      errorCF("qq", "Failed to get access token", {"response": body}.toTable)
  except Exception as e:
    errorCF("qq", "Auth error", {"error": e.msg}.toTable)
  finally:
    client.close()

proc qqGatewayLoop(c: QQChannel) {.async.} =
  while c.running:
    try:
      let data = await c.ws.receiveStrPacket()
      if data == "": break
      let msg = parseJson(data)
      let op = msg["op"].getInt()

      if op == 10: # Hello
        let interval = msg["d"]["heartbeat_interval"].getInt()
        discard (proc() {.async.} =
          while c.running:
            await sleepAsync(interval)
            if c.ws != nil: await c.ws.send($ %*{"op": 1, "d": nil})
        )()
        # Identify
        await c.ws.send($ %*{
          "op": 2,
          "d": {
            "token": "QQBot " & c.token,
            "intents": 1 shl 30, # Intent for C2C and Group messages
            "properties": {"os": "linux", "browser": "nimclaw", "device": "nimclaw"}
          }
        })

      elif op == 0: # Dispatch
        let t = msg["t"].getStr()
        if t == "C2C_MESSAGE_CREATE" or t == "GROUP_AT_MESSAGE_CREATE":
          let d = msg["d"]
          let msgID = d["id"].getStr()

          acquire(c.lock)
          if c.processedIDs.hasKey(msgID):
            release(c.lock)
            continue
          c.processedIDs[msgID] = true
          release(c.lock)

          let senderID = if d.hasKey("author"): d["author"]["id"].getStr() else: "unknown"
          let content = d["content"].getStr()
          let chatID = if t == "C2C_MESSAGE_CREATE": senderID else: d["group_id"].getStr()

          infoCF("qq", "Received message", {"type": t, "sender": senderID}.toTable)
          c.handleMessage(senderID, chatID, content)

    except Exception as e:
      errorCF("qq", "Gateway error", {"error": e.msg}.toTable)
      await sleepAsync(5000)

method name*(c: QQChannel): string = "qq"

method start*(c: QQChannel) {.async.} =
  if c.appID == "" or c.appSecret == "": return
  infoC("qq", "Starting QQ Bot channel...")
  await c.getAccessToken()

  let client = newAsyncHttpClient()
  client.headers["Authorization"] = "QQBot " & c.token
  try:
    let response = await client.get("https://api.sgroup.qq.com/gateway/bot")
    let body = await response.body
    let res = parseJson(body)
    if res.hasKey("url"):
      let url = res["url"].getStr()
      c.ws = await newWebSocket(url)
      c.running = true
      discard qqGatewayLoop(c)
      infoC("qq", "QQ bot connected")
  except Exception as e:
    errorCF("qq", "Connection failed", {"error": e.msg}.toTable)
  finally:
    client.close()

method stop*(c: QQChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: QQChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  let client = newAsyncHttpClient()
  client.headers["Authorization"] = "QQBot " & c.token
  client.headers["Content-Type"] = "application/json"
  let url = "https://api.sgroup.qq.com/v2/users/$1/messages".format(msg.chat_id)
  let payload = %*{"content": msg.content, "msg_type": 0}
  try:
    let resp = await client.post(url, $payload)
    if not resp.status.startsWith("200"):
      let body = await resp.body
      errorCF("qq", "Send failed", {"status": resp.status, "response": body}.toTable)
  except Exception as e:
    errorCF("qq", "Send error", {"error": e.msg}.toTable)
  finally:
    client.close()

method isRunning*(c: QQChannel): bool = c.running
