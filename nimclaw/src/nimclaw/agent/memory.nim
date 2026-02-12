import std/[os, times, strutils]

type
  MemoryStore* = ref object
    workspace*: string
    memoryDir*: string
    memoryFile*: string

proc newMemoryStore*(workspace: string): MemoryStore =
  let memoryDir = workspace / "memory"
  let memoryFile = memoryDir / "MEMORY.md"
  if not dirExists(memoryDir):
    createDir(memoryDir)
  MemoryStore(
    workspace: workspace,
    memoryDir: memoryDir,
    memoryFile: memoryFile
  )

proc getTodayFile(ms: MemoryStore): string =
  let today = now().format("yyyyMMdd")
  let monthDir = today[0..5]
  return ms.memoryDir / monthDir / (today & ".md")

proc readLongTerm*(ms: MemoryStore): string =
  if fileExists(ms.memoryFile):
    return readFile(ms.memoryFile)
  return ""

proc writeLongTerm*(ms: MemoryStore, content: string) =
  writeFile(ms.memoryFile, content)

proc readToday*(ms: MemoryStore): string =
  let todayFile = ms.getTodayFile()
  if fileExists(todayFile):
    return readFile(todayFile)
  return ""

proc appendToday*(ms: MemoryStore, content: string) =
  let todayFile = ms.getTodayFile()
  let monthDir = parentDir(todayFile)
  if not dirExists(monthDir):
    createDir(monthDir)

  var existingContent = ""
  if fileExists(todayFile):
    existingContent = readFile(todayFile)

  var newContent = ""
  if existingContent == "":
    let header = "# " & now().format("yyyy-MM-dd") & "\n\n"
    newContent = header & content
  else:
    newContent = existingContent & "\n" & content

  writeFile(todayFile, newContent)

proc getRecentDailyNotes*(ms: MemoryStore, days: int): string =
  var notes: seq[string] = @[]
  for i in 0 ..< days:
    let date = now() - i.days
    let dateStr = date.format("yyyyMMdd")
    let monthDir = dateStr[0..5]
    let filePath = ms.memoryDir / monthDir / (dateStr & ".md")
    if fileExists(filePath):
      notes.add(readFile(filePath))

  if notes.len == 0: return ""
  return notes.join("\n\n---\n\n")

proc getMemoryContext*(ms: MemoryStore): string =
  var parts: seq[string] = @[]
  let longTerm = ms.readLongTerm()
  if longTerm != "":
    parts.add("## Long-term Memory\n\n" & longTerm)

  let recentNotes = ms.getRecentDailyNotes(3)
  if recentNotes != "":
    parts.add("## Recent Daily Notes\n\n" & recentNotes)

  if parts.len == 0: return ""
  return "# Memory\n\n" & parts.join("\n\n---\n\n")
