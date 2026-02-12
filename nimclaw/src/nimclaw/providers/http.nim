import std/[asyncdispatch, json, strutils, tables, options], httpclient
import types
import ../config as claw_config

type
  HTTPProvider* = ref object of LLMProvider
    apiKey*: string
    apiBase*: string
    client*: AsyncHttpClient

proc newHTTPProvider*(apiKey, apiBase: string): HTTPProvider =
  HTTPProvider(
    apiKey: apiKey,
    apiBase: apiBase,
    client: newAsyncHttpClient()
  )

method getDefaultModel*(p: HTTPProvider): string =
  return ""

method chat*(p: HTTPProvider, messages: seq[Message], tools: seq[ToolDefinition], model: string, options: Table[string, JsonNode]): Future[LLMResponse] {.async.} =
  if p.apiBase == "":
    raise newException(ValueError, "API base not configured")

  var requestBody = %*{
    "model": model,
    "messages": messages
  }

  if tools.len > 0:
    requestBody["tools"] = %tools
    requestBody["tool_choice"] = %"auto"

  if options.hasKey("max_tokens"):
    let lowerModel = model.toLowerAscii
    if lowerModel.contains("glm") or lowerModel.contains("o1"):
      requestBody["max_completion_tokens"] = options["max_tokens"]
    else:
      requestBody["max_tokens"] = options["max_tokens"]

  if options.hasKey("temperature"):
    requestBody["temperature"] = options["temperature"]

  p.client.headers = newHttpHeaders({
    "Content-Type": "application/json"
  })
  if p.apiKey != "":
    p.client.headers["Authorization"] = "Bearer " & p.apiKey

  let url = p.apiBase & "/chat/completions"
  let response = await p.client.post(url, $requestBody)
  let body = await response.body

  if not response.status.startsWith("200"):
    raise newException(IOError, "API error ($1): $2".format(response.status, body))

  let jsonResp = parseJson(body)

  var llmResp = LLMResponse()
  if jsonResp.hasKey("choices") and jsonResp["choices"].len > 0:
    let choice = jsonResp["choices"][0]
    let msg = choice["message"]
    if msg.hasKey("content") and msg["content"].kind != JNull:
      llmResp.content = msg["content"].getStr()

    if msg.hasKey("tool_calls"):
      for tc in msg["tool_calls"]:
        var toolCall = ToolCall(
          id: tc["id"].getStr(),
          `type`: tc.getOrDefault("type").getStr("function")
        )
        if tc.hasKey("function"):
          let fn = tc["function"]
          toolCall.name = fn["name"].getStr()
          let argsStr = fn["arguments"].getStr()
          try:
            let argsJson = parseJson(argsStr)
            for k, v in argsJson.fields:
              toolCall.arguments[k] = v
          except:
            toolCall.arguments["raw"] = %argsStr
        llmResp.tool_calls.add(toolCall)

    llmResp.finish_reason = choice.getOrDefault("finish_reason").getStr("stop")

  if jsonResp.hasKey("usage"):
    let usage = jsonResp["usage"]
    llmResp.usage = UsageInfo(
      prompt_tokens: usage.getOrDefault("prompt_tokens").getInt(),
      completion_tokens: usage.getOrDefault("completion_tokens").getInt(),
      total_tokens: usage.getOrDefault("total_tokens").getInt()
    )

  return llmResp

proc createProvider*(cfg: Config): LLMProvider =
  let model = cfg.agents.defaults.model
  var apiKey, apiBase: string
  let lowerModel = model.toLowerAscii

  case model:
  of "":
    discard # Should not happen
  else:
    if model.startsWith("openrouter/") or model.startsWith("anthropic/") or model.startsWith("openai/") or
       model.startsWith("meta-llama/") or model.startsWith("deepseek/") or model.startsWith("google/"):
      apiKey = cfg.providers.openrouter.api_key
      apiBase = if cfg.providers.openrouter.api_base != "": cfg.providers.openrouter.api_base else: "https://openrouter.ai/api/v1"
    elif (lowerModel.contains("claude") or model.startsWith("anthropic/")) and cfg.providers.anthropic.api_key != "":
      apiKey = cfg.providers.anthropic.api_key
      apiBase = if cfg.providers.anthropic.api_base != "": cfg.providers.anthropic.api_base else: "https://api.anthropic.com/v1"
    elif (lowerModel.contains("gpt") or model.startsWith("openai/")) and cfg.providers.openai.api_key != "":
      apiKey = cfg.providers.openai.api_key
      apiBase = if cfg.providers.openai.api_base != "": cfg.providers.openai.api_base else: "https://api.openai.com/v1"
    elif (lowerModel.contains("gemini") or model.startsWith("google/")) and cfg.providers.gemini.api_key != "":
      apiKey = cfg.providers.gemini.api_key
      apiBase = if cfg.providers.gemini.api_base != "": cfg.providers.gemini.api_base else: "https://generativelanguage.googleapis.com/v1beta"
    elif (lowerModel.contains("glm") or lowerModel.contains("zhipu")) and cfg.providers.zhipu.api_key != "":
      apiKey = cfg.providers.zhipu.api_key
      apiBase = if cfg.providers.zhipu.api_base != "": cfg.providers.zhipu.api_base else: "https://open.bigmodel.cn/api/paas/v4"
    elif (lowerModel.contains("groq") or model.startsWith("groq/")) and cfg.providers.groq.api_key != "":
      apiKey = cfg.providers.groq.api_key
      apiBase = if cfg.providers.groq.api_base != "": cfg.providers.groq.api_base else: "https://api.groq.com/openai/v1"
    elif cfg.providers.vllm.api_base != "":
      apiKey = cfg.providers.vllm.api_key
      apiBase = cfg.providers.vllm.api_base
    else:
      if cfg.providers.openrouter.api_key != "":
        apiKey = cfg.providers.openrouter.api_key
        apiBase = if cfg.providers.openrouter.api_base != "": cfg.providers.openrouter.api_base else: "https://openrouter.ai/api/v1"
      else:
        raise newException(ValueError, "no API key configured for model: " & model)

  if apiKey == "" and not model.startsWith("bedrock/"):
     raise newException(ValueError, "no API key configured for provider (model: " & model & ")")

  if apiBase == "":
     raise newException(ValueError, "no API base configured for provider (model: " & model & ")")

  return newHTTPProvider(apiKey, apiBase)
