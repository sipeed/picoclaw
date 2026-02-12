import std/[asyncdispatch, tables, locks, strutils]
import base as channel_base
import telegram, discord, whatsapp, dingtalk, maixcam, feishu, qq
import ../bus, ../bus_types, ../config, ../logger

type
  Manager* = ref object
    channels*: Table[string, channel_base.Channel]
    bus*: MessageBus
    config*: Config
    lock*: Lock
    running*: bool

proc newManager*(cfg: Config, messageBus: MessageBus): Manager =
  Manager(
    channels: initTable[string, channel_base.Channel](),
    bus: messageBus,
    config: cfg
  )

proc initChannels*(m: Manager) =
  infoC("channels", "Initializing channel manager")

  if m.config.channels.telegram.enabled and m.config.channels.telegram.token != "":
    m.channels["telegram"] = newTelegramChannel(m.config.channels.telegram, m.bus)

  if m.config.channels.discord.enabled and m.config.channels.discord.token != "":
    m.channels["discord"] = newDiscordChannel(m.config.channels.discord, m.bus)

  if m.config.channels.whatsapp.enabled and m.config.channels.whatsapp.bridge_url != "":
    m.channels["whatsapp"] = newWhatsAppChannel(m.config.channels.whatsapp, m.bus)

  if m.config.channels.dingtalk.enabled:
    m.channels["dingtalk"] = newDingTalkChannel(m.config.channels.dingtalk, m.bus)

  if m.config.channels.maixcam.enabled:
    m.channels["maixcam"] = newMaixCamChannel(m.config.channels.maixcam, m.bus)

  if m.config.channels.feishu.enabled:
    m.channels["feishu"] = newFeishuChannel(m.config.channels.feishu, m.bus)

  if m.config.channels.qq.enabled:
    m.channels["qq"] = newQQChannel(m.config.channels.qq, m.bus)

  infoCF("channels", "Channel initialization completed", {"enabled_channels": $m.channels.len}.toTable)

proc dispatchOutbound(m: Manager) {.async.} =
  infoC("channels", "Outbound dispatcher started")
  while m.running:
    let msg = await m.bus.subscribeOutbound()
    if m.channels.hasKey(msg.channel):
      let channel = m.channels[msg.channel]
      try:
        await channel.send(msg)
      except Exception as e:
        errorCF("channels", "Error sending message to channel", {"channel": msg.channel, "error": e.msg}.toTable)
    else:
      warnCF("channels", "Unknown channel for outbound message", {"channel": msg.channel}.toTable)

proc startAll*(m: Manager) {.async.} =
  if m.channels.len == 0:
    warnC("channels", "No channels enabled")
    return

  m.running = true
  discard dispatchOutbound(m)

  for name, channel in m.channels:
    infoCF("channels", "Starting channel", {"channel": name}.toTable)
    try:
      await channel.start()
    except Exception as e:
      errorCF("channels", "Failed to start channel", {"channel": name, "error": e.msg}.toTable)

proc stopAll*(m: Manager) {.async.} =
  m.running = false
  for name, channel in m.channels:
    infoCF("channels", "Stopping channel", {"channel": name}.toTable)
    try:
      await channel.stop()
    except Exception as e:
      errorCF("channels", "Error stopping channel", {"channel": name, "error": e.msg}.toTable)

proc getEnabledChannels*(m: Manager): seq[string] =
  for k in m.channels.keys: result.add(k)

proc getChannel*(m: Manager, name: string): (channel_base.Channel, bool) =
  if m.channels.hasKey(name): (m.channels[name], true) else: (nil, false)
