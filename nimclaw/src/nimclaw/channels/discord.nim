import std/[asyncdispatch, tables, strutils, json, os, httpclient]
import dimscord
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils, ../services/voice

type
  DiscordChannel* = ref object of BaseChannel
    discord*: DiscordClient
    token*: string
    transcriber*: GroqTranscriber

proc newDiscordChannel*(cfg: DiscordConfig, bus: MessageBus): DiscordChannel =
  let base = newBaseChannel("discord", bus, cfg.allow_from)
  DiscordChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    token: cfg.token,
    discord: newDiscordClient(cfg.token)
  )

method name*(c: DiscordChannel): string = "discord"

method start*(c: DiscordChannel) {.async.} =
  infoC("discord", "Starting Discord bot...")

  c.discord.events.on_ready = proc (s: Shard, r: Ready) {.async.} =
    infoCF("discord", "Discord bot connected", {"username": r.user.username, "user_id": r.user.id}.toTable)
    c.running = true

  c.discord.events.message_create = proc (s: Shard, m: Message) {.async.} =
    if m.author.bot: return

    let senderID = m.author.id
    let chatID = m.channel_id
    let content = m.content

    c.handleMessage(senderID, chatID, content)

  await c.discord.startSession(gateway_intents = {giGuildMessages, giDirectMessages, giMessageContent})

method stop*(c: DiscordChannel) {.async.} =
  c.running = false
  await c.discord.endSession()

method send*(c: DiscordChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return

  try:
    discard await c.discord.api.sendMessage(msg.chat_id, msg.content)
  except Exception as e:
    errorCF("discord", "Failed to send discord message", {"error": e.msg}.toTable)

method setTranscriber*(c: DiscordChannel, transcriber: GroqTranscriber) =
  c.transcriber = transcriber

method isRunning*(c: DiscordChannel): bool = c.running
