import std/[os, json, strutils, tables]
import jsony

type
  AgentDefaults* = object
    workspace*: string
    model*: string
    max_tokens*: int
    temperature*: float64
    max_tool_iterations*: int

  AgentsConfig* = object
    defaults*: AgentDefaults

  WhatsAppConfig* = object
    enabled*: bool
    bridge_url*: string
    allow_from*: seq[string]

  TelegramConfig* = object
    enabled*: bool
    token*: string
    allow_from*: seq[string]

  FeishuConfig* = object
    enabled*: bool
    app_id*: string
    app_secret*: string
    encrypt_key*: string
    verification_token*: string
    allow_from*: seq[string]

  DiscordConfig* = object
    enabled*: bool
    token*: string
    allow_from*: seq[string]

  MaixCamConfig* = object
    enabled*: bool
    host*: string
    port*: int
    allow_from*: seq[string]

  QQConfig* = object
    enabled*: bool
    app_id*: string
    app_secret*: string
    allow_from*: seq[string]

  DingTalkConfig* = object
    enabled*: bool
    client_id*: string
    client_secret*: string
    allow_from*: seq[string]

  ChannelsConfig* = object
    whatsapp*: WhatsAppConfig
    telegram*: TelegramConfig
    feishu*: FeishuConfig
    discord*: DiscordConfig
    maixcam*: MaixCamConfig
    qq*: QQConfig
    dingtalk*: DingTalkConfig

  ProviderConfig* = object
    api_key*: string
    api_base*: string

  ProvidersConfig* = object
    anthropic*: ProviderConfig
    openai*: ProviderConfig
    openrouter*: ProviderConfig
    groq*: ProviderConfig
    zhipu*: ProviderConfig
    vllm*: ProviderConfig
    gemini*: ProviderConfig

  GatewayConfig* = object
    host*: string
    port*: int

  WebSearchConfig* = object
    api_key*: string
    max_results*: int

  WebToolsConfig* = object
    search*: WebSearchConfig

  ToolsConfig* = object
    web*: WebToolsConfig

  Config* = object
    agents*: AgentsConfig
    channels*: ChannelsConfig
    providers*: ProvidersConfig
    gateway*: GatewayConfig
    tools*: ToolsConfig

proc expandHome*(path: string): string =
  if path == "": return path
  if path[0] == '~':
    let home = getHomeDir()
    if path.len > 1 and path[1] == '/':
      return home / path[2..^1]
    return home
  return path

proc defaultConfig*(): Config =
  result = Config(
    agents: AgentsConfig(
      defaults: AgentDefaults(
        workspace: "~/.picoclaw/workspace",
        model: "glm-4.7",
        max_tokens: 8192,
        temperature: 0.7,
        max_tool_iterations: 20
      )
    ),
    channels: ChannelsConfig(
      whatsapp: WhatsAppConfig(enabled: false, bridge_url: "ws://localhost:3001"),
      telegram: TelegramConfig(enabled: false),
      feishu: FeishuConfig(enabled: false),
      discord: DiscordConfig(enabled: false),
      maixcam: MaixCamConfig(enabled: false, host: "0.0.0.0", port: 18790),
      qq: QQConfig(enabled: false),
      dingtalk: DingTalkConfig(enabled: false)
    ),
    gateway: GatewayConfig(host: "0.0.0.0", port: 18790),
    tools: ToolsConfig(
      web: WebToolsConfig(
        search: WebSearchConfig(max_results: 5)
      )
    )
  )

proc parseEnv*(cfg: var Config) =
  # Simple manual environment variable parsing to match Go's env library
  if existsEnv("PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"): cfg.agents.defaults.workspace = getEnv("PICOCLAW_AGENTS_DEFAULTS_WORKSPACE")
  if existsEnv("PICOCLAW_AGENTS_DEFAULTS_MODEL"): cfg.agents.defaults.model = getEnv("PICOCLAW_AGENTS_DEFAULTS_MODEL")
  # Add more as needed, but for now we focus on core features

proc loadConfig*(path: string): Config =
  result = defaultConfig()
  if fileExists(path):
    try:
      let data = readFile(path)
      result = data.fromJson(Config)
    except:
      discard # Log error maybe

  parseEnv(result)

proc saveConfig*(path: string, cfg: Config) =
  let dir = parentDir(path)
  if not dirExists(dir):
    createDir(dir)
  writeFile(path, cfg.toJson())

proc workspacePath*(cfg: Config): string =
  expandHome(cfg.agents.defaults.workspace)

proc getAPIKey*(cfg: Config): string =
  if cfg.providers.openrouter.api_key != "": return cfg.providers.openrouter.api_key
  if cfg.providers.anthropic.api_key != "": return cfg.providers.anthropic.api_key
  if cfg.providers.openai.api_key != "": return cfg.providers.openai.api_key
  if cfg.providers.gemini.api_key != "": return cfg.providers.gemini.api_key
  if cfg.providers.zhipu.api_key != "": return cfg.providers.zhipu.api_key
  if cfg.providers.groq.api_key != "": return cfg.providers.groq.api_key
  if cfg.providers.vllm.api_key != "": return cfg.providers.vllm.api_key
  return ""

proc getAPIBase*(cfg: Config): string =
  if cfg.providers.openrouter.api_key != "":
    if cfg.providers.openrouter.api_base != "": return cfg.providers.openrouter.api_base
    return "https://openrouter.ai/api/v1"
  if cfg.providers.zhipu.api_key != "": return cfg.providers.zhipu.api_base
  if cfg.providers.vllm.api_key != "" and cfg.providers.vllm.api_base != "": return cfg.providers.vllm.api_base
  return ""
