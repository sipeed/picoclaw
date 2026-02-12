import std/[json, tables, asyncdispatch]

type
  Tool* = ref object of RootObj

method name*(t: Tool): string {.base.} = ""
method description*(t: Tool): string {.base.} = ""
method parameters*(t: Tool): Table[string, JsonNode] {.base.} = initTable[string, JsonNode]()
method execute*(t: Tool, args: Table[string, JsonNode]): Future[string] {.base, async.} = return ""

type
  ContextualTool* = ref object of Tool

method setContext*(t: ContextualTool, channel, chatID: string) {.base.} = discard
