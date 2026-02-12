import std/[asyncdispatch, httpclient, os, json, strutils, tables]

type
  AvailableSkill* = object
    name*: string
    repository*: string
    description*: string
    author*: string
    tags*: seq[string]

  BuiltinSkill* = object
    name*: string
    path*: string
    enabled*: bool

  SkillInstaller* = ref object
    workspace*: string

proc newSkillInstaller*(workspace: string): SkillInstaller =
  SkillInstaller(workspace: workspace)

proc installFromGitHub*(si: SkillInstaller, repo: string): Future[void] {.async.} =
  let skillName = lastPathPart(repo)
  let skillDir = si.workspace / "skills" / skillName

  if dirExists(skillDir):
    raise newException(IOError, "Skill '$1' already exists".format(skillName))

  let url = "https://raw.githubusercontent.com/$1/main/SKILL.md".format(repo)
  let client = newAsyncHttpClient()

  try:
    let response = await client.get(url)
    if response.status != $Http200:
      raise newException(IOError, "Failed to fetch skill: " & response.status)

    let body = await response.body
    if not dirExists(si.workspace / "skills"):
      createDir(si.workspace / "skills")
    createDir(skillDir)
    writeFile(skillDir / "SKILL.md", body)
  finally:
    client.close()

proc uninstall*(si: SkillInstaller, skillName: string) =
  let skillDir = si.workspace / "skills" / skillName
  if not dirExists(skillDir):
    raise newException(IOError, "Skill '$1' not found".format(skillName))
  removeDir(skillDir)

proc listAvailableSkills*(si: SkillInstaller): Future[seq[AvailableSkill]] {.async.} =
  let url = "https://raw.githubusercontent.com/sipeed/picoclaw-skills/main/skills.json"
  let client = newAsyncHttpClient()

  try:
    let response = await client.get(url)
    if response.status != $Http200:
      raise newException(IOError, "Failed to fetch skills list: " & response.status)

    let body = await response.body
    return parseJson(body).to(seq[AvailableSkill])
  finally:
    client.close()
