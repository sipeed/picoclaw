import std/[tables, asyncdispatch]

type
  InboundMessage* = object
    channel*: string
    sender_id*: string
    chat_id*: string
    content*: string
    media*: seq[string]
    session_key*: string
    metadata*: Table[string, string]

  OutboundMessage* = object
    channel*: string
    chat_id*: string
    content*: string

  MessageHandler* = proc (msg: InboundMessage): Future[void] {.async.}
