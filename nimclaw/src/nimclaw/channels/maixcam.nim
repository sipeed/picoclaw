import std/[asyncdispatch, asyncnet, json, tables, strutils, locks]
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils

type
  MaixCamChannel* = ref object of BaseChannel
    server: AsyncSocket
    clients: seq[AsyncSocket]
    lock: Lock
    host: string
    port: int

proc newMaixCamChannel*(cfg: MaixCamConfig, bus: MessageBus): MaixCamChannel =
  let base = newBaseChannel("maixcam", bus, cfg.allow_from)
  var mc = MaixCamChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    clients: @[],
    host: cfg.host,
    port: cfg.port
  )
  initLock(mc.lock)
  return mc

method name*(c: MaixCamChannel): string = "maixcam"

proc handleClient(c: MaixCamChannel, client: AsyncSocket) {.async.} =
  while c.running:
    try:
      let line = await client.recvLine()
      if line == "": break
      let msg = parseJson(line)
      let msgType = msg.getOrDefault("type").getStr()

      case msgType:
      of "person_detected":
        let data = msg["data"]
        let score = data.getOrDefault("score").getFloat()
        let x = data.getOrDefault("x").getFloat()
        let y = data.getOrDefault("y").getFloat()
        let w = data.getOrDefault("w").getFloat()
        let h = data.getOrDefault("h").getFloat()
        let className = data.getOrDefault("class_name").getStr("person")

        let content = "📷 Person detected!\nClass: $1\nConfidence: $2%\nPosition: ($3, $4)\nSize: $5x$6".format(
          className, (score * 100).formatFloat(ffDecimal, 2), x, y, w, h
        )

        var metadata = initTable[string, string]()
        metadata["timestamp"] = $msg.getOrDefault("timestamp").getFloat()
        metadata["score"] = $score

        c.handleMessage("maixcam", "default", content, @[], metadata)

      of "heartbeat":
        debugC("maixcam", "Received heartbeat")
      of "status":
        infoCF("maixcam", "Status update from MaixCam", {"status": $msg["data"]}.toTable)
      else:
        warnCF("maixcam", "Unknown message type", {"type": msgType}.toTable)

    except Exception as e:
      errorCF("maixcam", "Failed to handle client", {"error": e.msg}.toTable)
      break

  acquire(c.lock)
  let idx = c.clients.find(client)
  if idx != -1: c.clients.delete(idx)
  release(c.lock)
  client.close()

method start*(c: MaixCamChannel) {.async.} =
  infoC("maixcam", "Starting MaixCam channel server")
  c.server = newAsyncSocket()
  c.server.setSockOpt(OptReuseAddr, true)
  try:
    c.server.bindAddr(Port(c.port), c.host)
    c.server.listen()
    c.running = true
    infoCF("maixcam", "MaixCam server listening", {"host": c.host, "port": $c.port}.toTable)

    discard (proc() {.async.} =
      while c.running:
        let client = await c.server.accept()
        acquire(c.lock)
        c.clients.add(client)
        release(c.lock)
        discard handleClient(c, client)
    )()
  except Exception as e:
    errorCF("maixcam", "Failed to start MaixCam server", {"error": e.msg}.toTable)

method stop*(c: MaixCamChannel) {.async.} =
  c.running = false
  c.server.close()
  acquire(c.lock)
  for client in c.clients: client.close()
  c.clients = @[]
  release(c.lock)

method send*(c: MaixCamChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return
  let payload = %*{"type": "command", "timestamp": 0.0, "message": msg.content, "chat_id": msg.chat_id}
  let data = $payload & "\n"
  acquire(c.lock)
  for client in c.clients:
    try: await client.send(data)
    except: discard
  release(c.lock)

method isRunning*(c: MaixCamChannel): bool = c.running
