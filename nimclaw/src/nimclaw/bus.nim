import std/[asyncdispatch, asyncfutures, tables, locks]
import bus_types

type
  MessageBus* = ref object
    inboundQueue: seq[InboundMessage]
    outboundQueue: seq[OutboundMessage]
    inboundWaiters: seq[Future[InboundMessage]]
    outboundWaiters: seq[Future[OutboundMessage]]
    handlers: Table[string, MessageHandler]
    lock: Lock

proc newMessageBus*(): MessageBus =
  var bus = MessageBus()
  bus.inboundQueue = @[]
  bus.outboundQueue = @[]
  bus.inboundWaiters = @[]
  bus.outboundWaiters = @[]
  bus.handlers = initTable[string, MessageHandler]()
  initLock(bus.lock)
  return bus

proc publishInbound*(bus: MessageBus, msg: InboundMessage) =
  acquire(bus.lock)
  if bus.inboundWaiters.len > 0:
    let waiter = bus.inboundWaiters[0]
    bus.inboundWaiters.delete(0)
    release(bus.lock)
    waiter.complete(msg)
  else:
    bus.inboundQueue.add(msg)
    release(bus.lock)

proc consumeInbound*(bus: MessageBus): Future[InboundMessage] {.async.} =
  acquire(bus.lock)
  if bus.inboundQueue.len > 0:
    let msg = bus.inboundQueue[0]
    bus.inboundQueue.delete(0)
    release(bus.lock)
    return msg
  else:
    let fut = newFuture[InboundMessage]("consumeInbound")
    bus.inboundWaiters.add(fut)
    release(bus.lock)
    return await fut

proc publishOutbound*(bus: MessageBus, msg: OutboundMessage) =
  acquire(bus.lock)
  if bus.outboundWaiters.len > 0:
    let waiter = bus.outboundWaiters[0]
    bus.outboundWaiters.delete(0)
    release(bus.lock)
    waiter.complete(msg)
  else:
    bus.outboundQueue.add(msg)
    release(bus.lock)

proc subscribeOutbound*(bus: MessageBus): Future[OutboundMessage] {.async.} =
  acquire(bus.lock)
  if bus.outboundQueue.len > 0:
    let msg = bus.outboundQueue[0]
    bus.outboundQueue.delete(0)
    release(bus.lock)
    return msg
  else:
    let fut = newFuture[OutboundMessage]("subscribeOutbound")
    bus.outboundWaiters.add(fut)
    release(bus.lock)
    return await fut

proc registerHandler*(bus: MessageBus, channel: string, handler: MessageHandler) =
  acquire(bus.lock)
  bus.handlers[channel] = handler
  release(bus.lock)

proc getHandler*(bus: MessageBus, channel: string): (MessageHandler, bool) =
  acquire(bus.lock)
  defer: release(bus.lock)
  if bus.handlers.hasKey(channel):
    return (bus.handlers[channel], true)
  else:
    return (nil, false)

proc close*(bus: MessageBus) =
  # In a real implementation we'd probably fail all pending waiters
  discard
