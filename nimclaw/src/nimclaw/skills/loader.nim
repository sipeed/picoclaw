import std/[os, strutils, json, re, tables]

type
  SkillMetadata* = object
    name*: string
    description*: string

  SkillInfo* = object
    name*: string
    path*: string
    source*: string
    description*: string

  SkillsLoader* = ref object
    workspace*: string
    workspaceSkills*: string
    globalSkills*: string
    builtinSkills*: string

proc newSkillsLoader*(workspace, globalSkills, builtinSkills: string): SkillsLoader =
  SkillsLoader(
    workspace: workspace,
    workspaceSkills: workspace / "skills",
    globalSkills: globalSkills,
    builtinSkills: builtinSkills
  )

proc getSkillMetadata(sl: SkillsLoader, skillPath: string): SkillMetadata =
  # Minimal implementation for now
  SkillMetadata(name: lastPathPart(parentDir(skillPath)))

proc listSkills*(sl: SkillsLoader): seq[SkillInfo] =
  # Minimal implementation
  result = @[]
  if dirExists(sl.workspaceSkills):
    for kind, path in walkDir(sl.workspaceSkills):
      if kind == pcDir:
        let skillFile = path / "SKILL.md"
        if fileExists(skillFile):
          result.add(SkillInfo(name: lastPathPart(path), path: skillFile, source: "workspace"))

proc loadSkill*(sl: SkillsLoader, name: string): (string, bool) =
  let skillFile = sl.workspaceSkills / name / "SKILL.md"
  if fileExists(skillFile):
    return (readFile(skillFile), true)
  return ("", false)

proc loadSkillsForContext*(sl: SkillsLoader, skillNames: seq[string]): string =
  if skillNames.len == 0: return ""
  var parts: seq[string] = @[]
  for name in skillNames:
    let (content, ok) = sl.loadSkill(name)
    if ok:
      parts.add("### Skill: " & name & "\n\n" & content)
  return parts.join("\n\n---\n\n")

proc escapeXML(s: string): string =
  s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

proc stripFrontmatter(content: string): string =
  # Simple version: if it starts with ---, find next ---
  if content.startsWith("---\n"):
    let nextIdx = content.find("\n---\n", 4)
    if nextIdx != -1:
      return content[nextIdx + 5 .. ^1]
  return content

proc buildSkillsSummary*(sl: SkillsLoader): string =
  let skills = sl.listSkills()
  if skills.len == 0: return ""
  var lines = @["<skills>"]
  for s in skills:
    lines.add("  <skill>")
    lines.add("    <name>" & escapeXML(s.name) & "</name>")
    lines.add("    <description>" & escapeXML(s.description) & "</description>")
    lines.add("    <location>" & escapeXML(s.path) & "</location>")
    lines.add("    <source>" & s.source & "</source>")
    lines.add("  </skill>")
  lines.add("</skills>")
  return lines.join("\n")
