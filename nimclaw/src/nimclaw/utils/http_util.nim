import std/[asyncdispatch, httpclient, json, strutils, tables]

type
  HTTPRequest* = object
    url*: string
    method*: string
    headers*: Table[string, string]
    body*: string

proc request*(req: HTTPRequest): Future[string] {.async.} =
  let client = newAsyncHttpClient()
  for k, v in req.headers:
    client.headers[k] = v

  try:
    let meth = case req.method.toUpperAscii:
      of "GET": HttpGet
      of "POST": HttpPost
      of "PUT": HttpPut
      of "DELETE": HttpDelete
      else: HttpPost

    let response = await client.request(req.url, meth, req.body)
    let body = await response.body
    if not response.status.startsWith("200"):
      raise newException(IOError, "HTTP error ($1): $2".format(response.status, body))
    return body
  finally:
    client.close()

proc get*(url: string, headers: Table[string, string] = initTable[string, string]()): Future[string] {.async.} =
  return await request(HTTPRequest(url: url, method: "GET", headers: headers))

proc post*(url: string, body: string, headers: Table[string, string] = initTable[string, string]()): Future[string] {.async.} =
  var h = headers
  if not h.hasKey("Content-Type"):
    h["Content-Type"] = "application/json"
  return await request(HTTPRequest(url: url, method: "POST", headers: h, body: body))
