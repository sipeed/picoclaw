import std/[os, json, asyncdispatch, tables, strutils]
import types

type
  ReadFileTool* = ref object of Tool
  WriteFileTool* = ref object of Tool
  ListDirTool* = ref object of Tool

# ReadFileTool
method name*(t: ReadFileTool): string = "read_file"
method description*(t: ReadFileTool): string = "Read the contents of a file"
method parameters*(t: ReadFileTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "path": {
        "type": "string",
        "description": "Path to the file to read"
      }
    },
    "required": %["path"]
  }.toTable

method execute*(t: ReadFileTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("path"): return "Error: path is required"
  let path = args["path"].getStr()
  try:
    return readFile(path)
  except Exception as e:
    return "Error: failed to read file: " & e.msg

# WriteFileTool
method name*(t: WriteFileTool): string = "write_file"
method description*(t: WriteFileTool): string = "Write content to a file"
method parameters*(t: WriteFileTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "path": {
        "type": "string",
        "description": "Path to the file to write"
      },
      "content": {
        "type": "string",
        "description": "Content to write to the file"
      }
    },
    "required": %["path", "content"]
  }.toTable

method execute*(t: WriteFileTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("path"): return "Error: path is required"
  if not args.hasKey("content"): return "Error: content is required"
  let path = args["path"].getStr()
  let content = args["content"].getStr()
  let dir = parentDir(path)
  try:
    if dir != "" and not dirExists(dir):
      createDir(dir)
    writeFile(path, content)
    return "File written successfully"
  except Exception as e:
    return "Error: failed to write file: " & e.msg

# ListDirTool
method name*(t: ListDirTool): string = "list_dir"
method description*(t: ListDirTool): string = "List files and directories in a path"
method parameters*(t: ListDirTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "path": {
        "type": "string",
        "description": "Path to list"
      }
    },
    "required": %["path"]
  }.toTable

method execute*(t: ListDirTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  let path = if args.hasKey("path"): args["path"].getStr() else: "."
  try:
    var result = ""
    for kind, entry in walkDir(path):
      if kind == pcDir or kind == pcLinkToDir:
        result.add("DIR:  " & lastPathPart(entry) & "\n")
      else:
        result.add("FILE: " & lastPathPart(entry) & "\n")
    return result
  except Exception as e:
    return "Error: failed to read directory: " & e.msg
