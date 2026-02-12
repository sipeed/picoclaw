import std/[os, strutils, json, asyncdispatch, tables]
import cligen
import nimclaw/[config, logger, bus, bus_types, session, agent/loop, providers/http, providers/types]
import nimclaw/channels/[manager as channel_manager, base as channel_base]
import nimclaw/services/[heartbeat, cron as cron_service, voice]
import nimclaw/skills/[loader as skills_loader, installer as skills_installer]

const version = "0.1.0"
const logo = "🦞"

proc getConfigPath(): string =
  getHomeDir() / ".picoclaw" / "config.json"

proc createWorkspaceTemplates(workspace: string) =
  let templates = {
    "AGENTS.md": "# Agent Instructions\n",
    "SOUL.md": "# Soul\n",
    "USER.md": "# User\n",
    "IDENTITY.md": "# Identity\n"
  }.toTable
  for filename, content in templates:
    let filePath = workspace / filename
    if not fileExists(filePath): writeFile(filePath, content)

proc onboard() =
  let configPath = getConfigPath()
  if fileExists(configPath):
    stdout.write "Overwrite? (y/n): "
    if stdin.readLine() != "y": return
  let cfg = defaultConfig()
  saveConfig(configPath, cfg)
  let workspace = cfg.workspacePath()
  if not dirExists(workspace): createDir(workspace)
  createWorkspaceTemplates(workspace)
  echo logo, " picoclaw is ready!"

proc agent(message = "", session = "cli:default", debug = false) =
  if debug: setLevel(DEBUG)
  let cfg = loadConfig(getConfigPath())
  let agentLoop = newAgentLoop(cfg, newMessageBus(), createProvider(cfg))
  if message != "": echo logo, " ", waitFor agentLoop.processDirect(message, session)
  else:
    while true:
      stdout.write logo & " You: "; let input = stdin.readLine().strip()
      if input in ["exit", "quit"]: break
      if input == "": continue
      echo "\n", logo, " ", waitFor agentLoop.processDirect(input, session), "\n"

proc status() =
  let configPath = getConfigPath()
  echo logo, " picoclaw Status\nConfig: ", configPath, if fileExists(configPath): " ✓" else: " ✗"

proc gateway(debug = false) =
  if debug: setLevel(DEBUG)
  let cfg = loadConfig(getConfigPath())
  let msgBus = newMessageBus()
  let agentLoop = newAgentLoop(cfg, msgBus, createProvider(cfg))
  let chanManager = newManager(cfg, msgBus); chanManager.initChannels()

  if cfg.providers.groq.api_key != "":
    let transcriber = newGroqTranscriber(cfg.providers.groq.api_key)
    for name in ["telegram", "discord"]:
      let (ch, ok) = chanManager.getChannel(name)
      if ok: ch.setTranscriber(transcriber)
    infoC("voice", "Groq voice transcription enabled")

  echo logo, " Starting Gateway..."
  let hbService = newHeartbeatService(cfg.workspacePath(), proc(p: string): Future[void] {.async.} =
    discard await agentLoop.processDirect(p, "system:heartbeat")
  , 1800, true)

  waitFor chanManager.startAll()
  waitFor hbService.start()
  echo logo, " Gateway started. Press Ctrl+C to stop."
  while true: poll()

proc skills(list = false, install = "", remove = "", installBuiltin = false) =
  let cfg = loadConfig(getConfigPath())
  let workspace = cfg.workspacePath()
  if list:
    let loader = newSkillsLoader(workspace, "", "")
    for s in loader.listSkills(): echo "✓ ", s.name
  elif install != "":
    let installer = newSkillInstaller(workspace)
    waitFor installer.installFromGitHub(install)
    echo "Installed ", install
  elif remove != "":
    let installer = newSkillInstaller(workspace)
    installer.uninstall(remove)
    echo "Removed ", remove
  elif installBuiltin:
    echo "Copying builtin skills to workspace..."
    # Copy logic...

proc cron(list = false) =
  let cfg = loadConfig(getConfigPath())
  let cs = newCronService(cfg.workspacePath() / "cron" / "jobs.json", nil)
  if list:
    for j in cs.listJobs(true): echo j.id, ": ", j.name

when isMainModule:
  dispatchMulti([onboard], [agent], [gateway], [status], [skills], [cron])
