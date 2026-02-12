import std/[tables, json, asyncdispatch]

type
  ToolFunctionCall* = object
    name*: string
    arguments*: string

  ToolCall* = object
    id*: string
    `type`*: string
    function*: ToolFunctionCall
    name*: string
    arguments*: Table[string, JsonNode]

  UsageInfo* = object
    prompt_tokens*: int
    completion_tokens*: int
    total_tokens*: int

  LLMResponse* = object
    content*: string
    tool_calls*: seq[ToolCall]
    finish_reason*: string
    usage*: UsageInfo

  Message* = object
    role*: string
    content*: string
    tool_calls*: seq[ToolCall]
    tool_call_id*: string

  ToolFunctionDefinition* = object
    name*: string
    description*: string
    parameters*: Table[string, JsonNode]

  ToolDefinition* = object
    `type`*: string
    function*: ToolFunctionDefinition

  LLMProvider* = ref object of RootObj

method chat*(p: LLMProvider, messages: seq[Message], tools: seq[ToolDefinition], model: string, options: Table[string, JsonNode]): Future[LLMResponse] {.base, async.} =
  discard

method getDefaultModel*(p: LLMProvider): string {.base.} =
  return ""
