import std/[asyncdispatch, json, tables, strutils]
import types

type
  SendCallback* = proc (channel, chatID, content: string): Future[void] {.async.}

  MessageTool* = ref object of ContextualTool
    sendCallback*: SendCallback
    defaultChannel*: string
    defaultChatID*: string

proc newMessageTool*(): MessageTool =
  MessageTool()

method name*(t: MessageTool): string = "message"
method description*(t: MessageTool): string = "Send a message to user on a chat channel. Use this when you want to communicate something."
method parameters*(t: MessageTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "content": {
        "type": "string",
        "description": "The message content to send"
      },
      "channel": {
        "type": "string",
        "description": "Optional: target channel (telegram, whatsapp, etc.)"
      },
      "chat_id": {
        "type": "string",
        "description": "Optional: target chat/user ID"
      }
    },
    "required": %["content"]
  }.toTable

method setContext*(t: MessageTool, channel, chatID: string) =
  t.defaultChannel = channel
  t.defaultChatID = chatID

proc setSendCallback*(t: MessageTool, callback: SendCallback) =
  t.sendCallback = callback

method execute*(t: MessageTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("content"): return "Error: content is required"
  let content = args["content"].getStr()

  var channel = if args.hasKey("channel"): args["channel"].getStr() else: ""
  var chatID = if args.hasKey("chat_id"): args["chat_id"].getStr() else: ""

  if channel == "": channel = t.defaultChannel
  if chatID == "": chatID = t.defaultChatID

  if channel == "" or chatID == "":
    return "Error: No target channel/chat specified"

  if t.sendCallback == nil:
    return "Error: Message sending not configured"

  try:
    await t.sendCallback(channel, chatID, content)
    return "Message sent to $1:$2".format(channel, chatID)
  except Exception as e:
    return "Error sending message: " & e.msg
