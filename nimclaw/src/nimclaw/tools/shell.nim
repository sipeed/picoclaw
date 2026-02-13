import std/[os, osproc, json, asyncdispatch, tables, strutils, times, streams]
import regex
import types

type
  ExecTool* = ref object of Tool
    workingDir*: string
    timeout*: Duration
    denyPatterns*: seq[Regex]
    allowPatterns*: seq[Regex]
    restrictToWorkspace*: bool

proc newExecTool*(workingDir: string): ExecTool =
  let denyPatternsStrings = [
    r"\brm\s+-[rf]{1,2}\b",
    r"\bdel\s+/[fq]\b",
    r"\brmdir\s+/s\b",
    r"\b(format|mkfs|diskpart)\b\s",
    r"\bdd\s+if=",
    r">\s*/dev/sd[a-z]\b",
    r"\b(shutdown|reboot|poweroff)\b",
    r":\(\)\s*\{.*\};\s*:"
  ]
  var denyPatterns: seq[Regex] = @[]
  for p in denyPatternsStrings:
    denyPatterns.add(re(p))

  ExecTool(
    workingDir: workingDir,
    timeout: initDuration(seconds = 60),
    denyPatterns: denyPatterns,
    allowPatterns: @[],
    restrictToWorkspace: false
  )

method name*(t: ExecTool): string = "exec"
method description*(t: ExecTool): string = "Execute a shell command and return its output. Use with caution."
method parameters*(t: ExecTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "command": {
        "type": "string",
        "description": "The shell command to execute"
      },
      "working_dir": {
        "type": "string",
        "description": "Optional working directory for the command"
      }
    },
    "required": %["command"]
  }.toTable

proc guardCommand(t: ExecTool, command, cwd: string): string =
  let lower = command.toLowerAscii
  for pattern in t.denyPatterns:
    if lower.contains(pattern):
      return "Command blocked by safety guard (dangerous pattern detected)"

  if t.allowPatterns.len > 0:
    var allowed = false
    for pattern in t.allowPatterns:
      if lower.contains(pattern):
        allowed = true
        break
    if not allowed:
      return "Command blocked by safety guard (not in allowlist)"

  if t.restrictToWorkspace:
    if command.contains("..\\") or command.contains("../"):
      return "Command blocked by safety guard (path traversal detected)"
    # More strict path check could be added here

  return ""

method execute*(t: ExecTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("command"): return "Error: command is required"
  let command = args["command"].getStr()
  var cwd = t.workingDir
  if args.hasKey("working_dir") and args["working_dir"].getStr() != "":
    cwd = args["working_dir"].getStr()

  if cwd == "":
    cwd = getCurrentDir()

  let guardErr = t.guardCommand(command, cwd)
  if guardErr != "":
    return "Error: " & guardErr

  # Nim's asyncdispatch doesn't have a direct async process execution with timeout easily available in stdlib
  # but we can use execProcess or similar, or just run it in a thread if needed.
  # For now, let's use a simple synchronous execProcess as a placeholder if we're in a single-threaded async loop,
  # or better, use a thread-pool.

  # Actually, std/osproc has startProcess and we can poll it.

  var p = startProcess("sh", workingDir = cwd, args = ["-c", command], options = {poStdErrToStdOut})
  let startTime = now()
  var output = ""

  # Simple polling for timeout and reading output without blocking
  while p.running:
    if (now() - startTime) > t.timeout:
      p.terminate()
      return "Error: Command timed out after " & $t.timeout

    # Read available data from stream
    let data = p.outputStream.readStr(1024)
    if data != "":
      output.add(data)

    await sleepAsync(50)

  # Final read
  output.add(p.outputStream.readAll())
  let exitCode = p.peekExitCode()
  p.close()

  if exitCode != 0:
    output.add("\nExit code: " & $exitCode)

  if output == "":
    output = "(no output)"

  let maxLen = 10000
  if output.len > maxLen:
    output = output[0 ..< maxLen] & "\n... (truncated, " & $(output.len - maxLen) & " more chars)"

  return output
