import std/[os, strutils, json, asyncdispatch, tables, times, options]
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
    "AGENTS.md": "# Agent Instructions\nYou are a helpful AI assistant.\n",
    "SOUL.md": "# Soul\nI am picoclaw.\n",
    "USER.md": "# User\n",
    "IDENTITY.md": "# Identity\nName: PicoClaw 🦞\n"
  }.toTable
  for filename, content in templates:
    let filePath = workspace / filename
    if not fileExists(filePath): writeFile(filePath, content)
  if not dirExists(workspace / "memory"): createDir(workspace / "memory")
  if not fileExists(workspace / "memory" / "MEMORY.md"):
    writeFile(workspace / "memory" / "MEMORY.md", "# Long-term Memory\n")

proc onboard() =
  let configPath = getConfigPath()
  if fileExists(configPath):
    stdout.write "Overwrite? (y/n): "
    if stdin.readLine().toLowerAscii != "y": return
  let cfg = defaultConfig()
  saveConfig(configPath, cfg)
  let workspace = cfg.workspacePath()
  createDir(workspace)
  createDir(workspace / "memory"); createDir(workspace / "skills")
  createDir(workspace / "sessions"); createDir(workspace / "cron")
  createWorkspaceTemplates(workspace)
  echo logo, " picoclaw is ready!"

proc agent(message = "", session = "cli:default", debug = false) =
  if debug: setLevel(DEBUG)
  let cfg = loadConfig(getConfigPath())
  let agentLoop = newAgentLoop(cfg, newMessageBus(), createProvider(cfg))
  if message != "": echo logo, " ", waitFor agentLoop.processDirect(message, session)
  else:
    echo logo, " Interactive mode\n"
    while true:
      stdout.write logo & " You: "; let input = stdin.readLine().strip()
      if input in ["exit", "quit"]: break
      if input == "": continue
      echo "\n", logo, " ", waitFor agentLoop.processDirect(input, session), "\n"

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
  let hbService = newHeartbeatService(cfg.workspacePath(), proc(p: string): Future[void] {.async.} =
    discard await agentLoop.processDirect(p, "system:heartbeat")
  , 1800, true)
  echo logo, " Starting Gateway..."
  waitFor chanManager.startAll(); waitFor hbService.start()
  echo logo, " Gateway started. Press Ctrl+C to stop."
  while true: poll()

proc status() =
  let configPath = getConfigPath()
  echo logo, " picoclaw Status\nConfig: ", configPath, if fileExists(configPath): " ✓" else: " ✗"

proc cron(list = false, add = false, remove = "", enable = "", disable = "",
          name = "", message = "", every = 0, at = 0.0, cron_expr = "",
          deliver = true, channel = "", to = "") =
  let cfg = loadConfig(getConfigPath())
  let cs = newCronService(cfg.workspacePath() / "cron" / "jobs.json", nil)
  if list:
    for j in cs.listJobs(true): echo "$1 ($2) - $3".format(j.name, j.id, j.schedule.kind)
  elif add:
    var sched: CronSchedule
    if every > 0: sched = CronSchedule(kind: "every", everyMs: some(every.int64 * 1000))
    elif at > 0: sched = CronSchedule(kind: "at", atMs: some(at.int64))
    elif cron_expr != "": sched = CronSchedule(kind: "cron", expr: cron_expr)
    else: (echo "Error: every, at, or cron_expr required"; return)
    let job = waitFor cs.addJob(name, sched, message, deliver, channel, to)
    echo "Added job: ", job.id
  elif remove != "":
    if cs.removeJob(remove): echo "Removed job ", remove
  elif enable != "": discard cs.enableJob(enable, true)
  elif disable != "": discard cs.enableJob(disable, false)

proc skills(list = false, install = "", remove = "", show = "", search = false,
            list_builtin = false, install_builtin = false) =
  let cfg = loadConfig(getConfigPath())
  let workspace = cfg.workspacePath()
  let installer = newSkillInstaller(workspace)
  let loader = newSkillsLoader(workspace, "", "")
  if list:
    for s in loader.listSkills(): echo "✓ ", s.name
  elif list_builtin:
    echo "Builtin skills: weather, news, stock, calculator" # Matches Go logic
  elif install != "":
    waitFor installer.installFromGitHub(install); echo "Installed ", install
  elif remove != "":
    installer.uninstall(remove); echo "Removed ", remove
  elif show != "":
    let (c, ok) = loader.loadSkill(show)
    if ok: echo c
  elif search:
    let available = waitFor installer.listAvailableSkills()
    for s in available: echo "- ", s.name, ": ", s.description
  elif install_builtin:
    echo "Copying builtin skills to workspace..."
    let builtinDir = getAppDir() / "picoclaw" / "skills"
    let targetDir = workspace / "skills"
    for s in ["weather", "news", "stock", "calculator"]:
      if dirExists(builtinDir / s):
        copyDir(builtinDir / s, targetDir / s)
        echo "  Installed ", s

when isMainModule:
  dispatchMulti([onboard], [agent], [gateway], [status], [cron], [skills])
