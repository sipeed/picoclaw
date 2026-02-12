import std/[asyncdispatch, httpclient, json, strutils, os, tables, times]
import ../logger, ../utils

type
  TranscriptionResponse* = object
    text*: string
    language*: string
    duration*: float64

  GroqTranscriber* = ref object
    apiKey*: string
    apiBase*: string
    client*: AsyncHttpClient

proc newGroqTranscriber*(apiKey: string): GroqTranscriber =
  GroqTranscriber(
    apiKey: apiKey,
    apiBase: "https://api.groq.com/openai/v1",
    client: newAsyncHttpClient()
  )

proc isAvailable*(t: GroqTranscriber): bool =
  t.apiKey != ""

proc transcribe*(t: GroqTranscriber, audioFilePath: string): Future[TranscriptionResponse] {.async.} =
  infoCF("voice", "Starting transcription", {"audio_file": audioFilePath}.toTable)

  if not fileExists(audioFilePath):
    raise newException(IOError, "Audio file not found")

  let boundary = "----NimClawBoundary" & $getTime().toUnix
  var body = ""

  # Simplified multipart body construction
  body.add("--" & boundary & "\r\n")
  body.add("Content-Disposition: form-data; name=\"file\"; filename=\"" & lastPathPart(audioFilePath) & "\"\r\n")
  body.add("Content-Type: audio/mpeg\r\n\r\n")
  body.add(readFile(audioFilePath))
  body.add("\r\n")

  body.add("--" & boundary & "\r\n")
  body.add("Content-Disposition: form-data; name=\"model\"\r\n\r\n")
  body.add("whisper-large-v3\r\n")

  body.add("--" & boundary & "--\r\n")

  t.client.headers = newHttpHeaders({
    "Content-Type": "multipart/form-data; boundary=" & boundary,
    "Authorization": "Bearer " & t.apiKey
  })

  let url = t.apiBase & "/audio/transcriptions"
  let response = await t.client.post(url, body)
  let respBody = await response.body

  if response.status != $Http200:
    errorCF("voice", "API error", {"status": response.status, "response": respBody}.toTable)
    raise newException(IOError, "API error: " & respBody)

  let result = parseJson(respBody).to(TranscriptionResponse)
  infoCF("voice", "Transcription completed successfully", {"text_length": $result.text.len}.toTable)
  return result
