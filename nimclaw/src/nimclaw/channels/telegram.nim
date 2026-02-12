import std/[asyncdispatch, tables, strutils, json, re, locks, os, httpclient, options]
import telebot
import base
import ../bus, ../bus_types, ../config, ../logger, ../utils, ../services/voice

type
  TelegramChannel* = ref object of BaseChannel
    bot*: TeleBot
    token*: string
    chatIDs*: Table[string, int64]
    transcriber*: GroqTranscriber

proc markdownToTelegramHTML(text: string): string =
  var res = text
  res = res.replace(re"&", "&amp;").replace(re"<", "&lt;").replace(re">", "&gt;")
  res = res.replace(re"\[([^\]]+)\]\(([^)]+)\)", "<a href=\"$2\">$1</a>")
  res = res.replace(re"\*\*(.+?)\*\*", "<b>$1</b>")
  res = res.replace(re"__(.+?)__", "<b>$1</b>")
  res = res.replace(re"_([^_]+)_", "<i>$1</i>")
  res = res.replace(re"~~(.+?)~~", "<s>$1</s>")
  res = res.replace(re"(?m)^[-*]\s+", "• ")
  return res

proc newTelegramChannel*(cfg: TelegramConfig, bus: MessageBus): TelegramChannel =
  let base = newBaseChannel("telegram", bus, cfg.allow_from)
  TelegramChannel(
    bus: base.bus,
    name: base.name,
    allowList: base.allowList,
    running: false,
    bot: newTeleBot(cfg.token),
    token: cfg.token,
    chatIDs: initTable[string, int64]()
  )

method name*(c: TelegramChannel): string = "telegram"

method start*(c: TelegramChannel) {.async.} =
  infoC("telegram", "Starting Telegram bot...")
  c.running = true

  proc updateHandler(bot: Telebot, update: Update): Future[bool] {.async.} =
    if not update.message.isNil:
      let msg = update.message
      if not msg.fromUser.isNil:
        let user = msg.fromUser
        var senderID = $user.id
        if user.username != "":
          senderID = $user.id & "|" & user.username

        let chatID = msg.chat.id
        c.chatIDs[senderID] = chatID

        var content = msg.text
        if msg.caption != "":
          if content != "": content.add("\n")
          content.add(msg.caption)

        if content == "": content = "[empty message]"

        c.handleMessage(senderID, $chatID, content)
    return true

  c.bot.onUpdate(updateHandler)
  discard c.bot.pollAsync(timeout = 30)

method stop*(c: TelegramChannel) {.async.} =
  c.running = false

method send*(c: TelegramChannel, msg: OutboundMessage) {.async.} =
  if not c.running: return

  let chatID = msg.chat_id.parseBiggestInt()
  let htmlContent = markdownToTelegramHTML(msg.content)

  try:
    discard await c.bot.sendMessage(chatID, htmlContent, parseMode = "HTML")
  except Exception as e:
    warnCF("telegram", "HTML parse failed, falling back to plain text", {"error": e.msg}.toTable)
    discard await c.bot.sendMessage(chatID, msg.content)

method setTranscriber*(c: TelegramChannel, transcriber: GroqTranscriber) =
  c.transcriber = transcriber

method isRunning*(c: TelegramChannel): bool = c.running
