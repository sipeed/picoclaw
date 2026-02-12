import std/[os, json, asyncdispatch, httpclient, tables, strutils, uri, times]
import regex
import types

const userAgent = "Mozilla/5.0 (compatible; nimclaw/1.0)"

type
  WebSearchTool* = ref object of Tool
    apiKey*: string
    maxResults*: int

proc newWebSearchTool*(apiKey: string, maxResults: int): WebSearchTool =
  let count = if maxResults <= 0 or maxResults > 10: 5 else: maxResults
  WebSearchTool(apiKey: apiKey, maxResults: count)

method name*(t: WebSearchTool): string = "web_search"
method description*(t: WebSearchTool): string = "Search the web for current information. Returns titles, URLs, and snippets from search results."
method parameters*(t: WebSearchTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "query": {
        "type": "string",
        "description": "Search query"
      },
      "count": {
        "type": "integer",
        "description": "Number of results (1-10)",
        "minimum": 1,
        "maximum": 10
      }
    },
    "required": %["query"]
  }.toTable

method execute*(t: WebSearchTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if t.apiKey == "": return "Error: BRAVE_API_KEY not configured"
  if not args.hasKey("query"): return "Error: query is required"
  let query = args["query"].getStr()
  var count = t.maxResults
  if args.hasKey("count"):
    count = args["count"].getInt()
    if count <= 0 or count > 10: count = t.maxResults

  let searchURL = "https://api.search.brave.com/res/v1/web/search?q=$1&count=$2".format(encodeUrl(query), count)

  let client = newAsyncHttpClient(userAgent = userAgent)
  client.headers["Accept"] = "application/json"
  client.headers["X-Subscription-Token"] = t.apiKey

  try:
    let response = await client.get(searchURL)
    let body = await response.body
    let jsonResp = parseJson(body)

    if not jsonResp.hasKey("web") or not jsonResp["web"].hasKey("results"):
      return "No results for: " & query

    let results = jsonResp["web"]["results"]
    if results.len == 0:
      return "No results for: " & query

    var lines: seq[string] = @[]
    lines.add("Results for: " & query)
    for i in 0 ..< min(results.len, count):
      let item = results[i]
      lines.add("$1. $2\n   $3".format(i + 1, item["title"].getStr(), item["url"].getStr()))
      if item.hasKey("description"):
        lines.add("   " & item["description"].getStr())

    return lines.join("\n")
  except Exception as e:
    return "Error: search failed: " & e.msg
  finally:
    client.close()

type
  WebFetchTool* = ref object of Tool
    maxChars*: int

proc newWebFetchTool*(maxChars: int): WebFetchTool =
  let count = if maxChars <= 0: 50000 else: maxChars
  WebFetchTool(maxChars: count)

method name*(t: WebFetchTool): string = "web_fetch"
method description*(t: WebFetchTool): string = "Fetch a URL and extract readable content (HTML to text). Use this to get weather info, news, articles, or any web content."
method parameters*(t: WebFetchTool): Table[string, JsonNode] =
  {
    "type": %"object",
    "properties": %*{
      "url": {
        "type": "string",
        "description": "URL to fetch"
      },
      "maxChars": {
        "type": "integer",
        "description": "Maximum characters to extract",
        "minimum": 100
      }
    },
    "required": %["url"]
  }.toTable

proc extractText(html: string): string =
  var result = html
  result = result.replace(re"(?s)<script[\s\S]*?<\/script>", "")
  result = result.replace(re"(?s)<style[\s\S]*?<\/style>", "")
  result = result.replace(re"<[^>]+>", "")
  result = result.replace(re"\s+", " ")
  return result.strip()

method execute*(t: WebFetchTool, args: Table[string, JsonNode]): Future[string] {.async.} =
  if not args.hasKey("url"): return "Error: url is required"
  let urlStr = args["url"].getStr()

  let u = parseUri(urlStr)
  if u.scheme != "http" and u.scheme != "https":
    return "Error: only http/https URLs are allowed"

  var maxChars = t.maxChars
  if args.hasKey("maxChars"):
    let mc = args["maxChars"].getInt()
    if mc > 100: maxChars = mc

  let client = newAsyncHttpClient(userAgent = userAgent)
  try:
    let response = await client.get(urlStr)
    let body = await response.body
    let contentType = response.headers.getOrDefault("Content-Type")

    var text = ""
    var extractor = ""

    if contentType.contains("application/json"):
      text = body # Could format it if we wanted
      extractor = "json"
    elif contentType.contains("text/html") or body.startsWith("<!DOCTYPE") or body.toLowerAscii.startsWith("<html"):
      text = extractText(body)
      extractor = "text"
    else:
      text = body
      extractor = "raw"

    let truncated = text.len > maxChars
    if truncated:
      text = text[0 ..< maxChars]

    let resObj = %*{
      "url": urlStr,
      "status": response.status,
      "extractor": extractor,
      "truncated": truncated,
      "length": text.len,
      "text": text
    }
    return resObj.pretty()
  except Exception as e:
    return "Error: fetch failed: " & e.msg
  finally:
    client.close()
