import std/[asyncdispatch, httpclient, json, strutils, tables, os, re, times, options]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils, ../services/voice
import ws

type
  DiscordChannel* = ref object of BaseChannel
    token*: string
    ws*: WebSocket
    transcriber*: GroqTranscriber

proc newDiscordChannel*(cfg: DiscordConfig, bus: MessageBus): DiscordChannel =
  let base = newBaseChannel("discord", bus, cfg.allow_from)
  DiscordChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    token: cfg.token
  )

method setTranscriber*(c: DiscordChannel, transcriber: GroqTranscriber) =
  c.transcriber = transcriber

proc apiCall(c: DiscordChannel, method_name: string, url_part: string, payload: JsonNode = nil, meth: string = "POST"): Future[JsonNode] {.async.} =
  let client = newAsyncHttpClient()
  client.headers["Authorization"] = "Bot " & c.token
  client.headers["Content-Type"] = "application/json"
  let url = "https://discord.com/api/v10/" & url_part
  try:
    let response = if meth == "POST": await client.post(url, if payload != nil: $payload else: "")
                   elif meth == "GET": await client.get(url)
                   else: await client.post(url, "")
    let body = await response.body
    if body == "": return %*{}
    return parseJson(body)
  finally:
    client.close()

proc gatewayLoop(c: DiscordChannel) {.async.} =
  while c.running:
    try:
      let data = await c.ws.receiveStrPacket()
      if data == "": break
      let msg = parseJson(data)
      let op = msg["op"].getInt()

      if op == 10: # Hello
        let interval = msg["d"]["heartbeat_interval"].getInt()
        # Start heartbeating (simplified)
        discard (proc() {.async.} =
          while c.running:
            await sleepAsync(interval)
            if c.ws != nil: await c.ws.send($ %*{"op": 1, "d": nil})
        )()
        # Identify
        await c.ws.send($ %*{
          "op": 2,
          "d": {
            "token": c.token,
            "intents": 33280, # GuildMessages | DirectMessages | MessageContent
            "properties": {"os": "linux", "browser": "nimclaw", "device": "nimclaw"}
          }
        })

      elif op == 0: # Dispatch
        let t = msg["t"].getStr()
        if t == "MESSAGE_CREATE":
          let d = msg["d"]
          if d.getOrDefault("author").getOrDefault("bot").getBool(): continue
          let senderID = d["author"]["id"].getStr()
          let chatID = d["channel_id"].getStr()
          let content = d["content"].getStr()
          c.handleMessage(senderID, chatID, content)

    except Exception as e:
      errorCF("discord", "Gateway error", {"error": e.msg}.toTable)
      await sleepAsync(5000)

method name*(c: DiscordChannel): string = "discord"

method start*(c: DiscordChannel) {.async.} =
  infoC("discord", "Starting Discord bot (Gateway mode)...")
  try:
    let gatewayRes = await c.apiCall("GET", "gateway/bot", meth="GET")
    let url = gatewayRes["url"].getStr() & "/?v=10&encoding=json"
    c.ws = await newWebSocket(url)
    c.running = true
    discard gatewayLoop(c)
  except Exception as e:
    errorCF("discord", "Failed to start Discord bot", {"error": e.msg}.toTable)

method stop*(c: DiscordChannel) {.async.} =
  c.running = false
  if c.ws != nil: c.ws.close()

method send*(c: DiscordChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  discard await c.apiCall("POST", "channels/$1/messages".format(msg.chat_id), %*{"content": msg.content})

method isRunning*(c: DiscordChannel): bool = c.running
