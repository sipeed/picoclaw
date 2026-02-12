import std/[os, times, strutils, json, syncio, tables]
import jsony

type
  LogLevel* = enum
    DEBUG, INFO, WARN, ERROR, FATAL

const
  logLevelNames: Table[LogLevel, string] = {
    DEBUG: "DEBUG",
    INFO:  "INFO",
    WARN:  "WARN",
    ERROR: "ERROR",
    FATAL: "FATAL"
  }.toTable

var
  currentLevel = INFO
  logFile: File
  fileLoggingEnabled = false

type
  LogEntry* = object
    level*: string
    timestamp*: string
    component*: string
    message*: string
    fields*: Table[string, string]
    caller*: string

proc setLevel*(level: LogLevel) =
  currentLevel = level

proc getLevel*(): LogLevel =
  currentLevel

proc enableFileLogging*(filePath: string): bool =
  try:
    if fileLoggingEnabled:
      logFile.close()
    logFile = open(filePath, fmAppend)
    fileLoggingEnabled = true
    echo "File logging enabled: ", filePath
    return true
  except:
    echo "Failed to open log file: ", filePath
    return false

proc disableFileLogging*() =
  if fileLoggingEnabled:
    logFile.close()
    fileLoggingEnabled = false
    echo "File logging disabled"

proc formatFields(fields: Table[string, string]): string =
  if fields.len == 0: return ""
  var parts: seq[string] = @[]
  for k, v in fields:
    parts.add(k & "=" & v)
  return " {" & parts.join(", ") & "}"

proc logMessage(level: LogLevel, component: string, message: string, fields: Table[string, string] = initTable[string, string]()) =
  if level < currentLevel:
    return

  let now = now().utc
  let timestamp = now.format("yyyy-MM-dd'T'HH:mm:ss'Z'")

  var entry = LogEntry(
    level: logLevelNames[level],
    timestamp: timestamp,
    component: component,
    message: message,
    fields: fields
  )

  # In Nim, getting caller info is a bit different, we can use getStackTrace() or similar if needed
  # but for now let's keep it simple.

  if fileLoggingEnabled:
    try:
      logFile.writeLine(entry.toJson() & "\n")
      logFile.flushFile()
    except:
      discard

  let componentStr = if component != "": " " & component & ":" else: ""
  let fieldStr = formatFields(fields)

  echo "[$1] [$2]$3 $4$5".format(timestamp, logLevelNames[level], componentStr, message, fieldStr)

  if level == FATAL:
    quit(1)

proc debug*(message: string) = logMessage(DEBUG, "", message)
proc debugC*(component, message: string) = logMessage(DEBUG, component, message)
proc debugF*(message: string, fields: Table[string, string]) = logMessage(DEBUG, "", message, fields)
proc debugCF*(component, message: string, fields: Table[string, string]) = logMessage(DEBUG, component, message, fields)

proc info*(message: string) = logMessage(INFO, "", message)
proc infoC*(component, message: string) = logMessage(INFO, component, message)
proc infoF*(message: string, fields: Table[string, string]) = logMessage(INFO, "", message, fields)
proc infoCF*(component, message: string, fields: Table[string, string]) = logMessage(INFO, component, message, fields)

proc warn*(message: string) = logMessage(WARN, "", message)
proc warnC*(component, message: string) = logMessage(WARN, component, message)
proc warnF*(message: string, fields: Table[string, string]) = logMessage(WARN, "", message, fields)
proc warnCF*(component, message: string, fields: Table[string, string]) = logMessage(WARN, component, message, fields)

proc error*(message: string) = logMessage(ERROR, "", message)
proc errorC*(component, message: string) = logMessage(ERROR, component, message)
proc errorF*(message: string, fields: Table[string, string]) = logMessage(ERROR, "", message, fields)
proc errorCF*(component, message: string, fields: Table[string, string]) = logMessage(ERROR, component, message, fields)

proc fatal*(message: string) = logMessage(FATAL, "", message)
proc fatalC*(component, message: string) = logMessage(FATAL, component, message)
proc fatalF*(message: string, fields: Table[string, string]) = logMessage(FATAL, "", message, fields)
proc fatalCF*(component, message: string, fields: Table[string, string]) = logMessage(FATAL, component, message, fields)
