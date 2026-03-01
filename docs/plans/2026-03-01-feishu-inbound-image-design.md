# Feishu Inbound Image Download

## Goal

Enable Feishu channel to receive image messages from users, download them via SDK, and pass them through the Agent pipeline so vision-capable LLMs can process the images.

## Scope

- Only `MsgTypeImage`. No files, audio, video, or rich text.
- Inbound only. Outbound image sending is covered by PR #902.

## Implementation

### Changed file: `pkg/channels/feishu_64.go`

**`handleMessageReceive()`** — add image detection before the existing text extraction:

1. Check `message.MessageType == larkim.MsgTypeImage`
2. Parse `message.Content` JSON to extract `image_key`
3. Call `downloadImage(ctx, imageKey)` to get a local temp file path
4. Register with MediaStore via `store.Store(localPath, meta, scope)` to get `media://uuid` ref
5. Append ref to `mediaPaths []string`
6. Append `[image: photo]` to content text (matches Telegram convention)
7. Pass `mediaPaths` to `HandleMessage()` (currently hardcoded to `nil`)

**New method: `downloadImage(ctx, imageKey) (string, error)`**

1. Call `c.client.Im.V1.Image.Get(ctx, req)` with the image_key
2. Create temp file under `os.TempDir()/picoclaw_media/`
3. Copy response body to temp file
4. Return file path

### Pattern reference: Telegram channel

```go
// telegram.go:384-420 — the pattern we follow
scope := channels.BuildMediaScope("telegram", chatIDStr, messageIDStr)
storeMedia := func(localPath, filename string) string {
    if store := c.GetMediaStore(); store != nil {
        ref, _ := store.Store(localPath, media.MediaMeta{Filename: filename, Source: "telegram"}, scope)
        return ref
    }
    return localPath
}
// ...
photoPath := c.downloadPhoto(ctx, photo.FileID)
mediaPaths = append(mediaPaths, storeMedia(photoPath, "photo.jpg"))
content += "[image: photo]"
```

## Testing

### New file: `pkg/channels/feishu_64_test.go`

1. **`TestExtractFeishuImageKey`** — parse `{"image_key":"img_v2_xxx"}` correctly
2. **`TestDownloadImage`** — mock HTTP server returns PNG bytes, verify temp file written
3. **`TestHandleImageMessage`** — full flow: construct P2MessageReceiveV1 event with MsgTypeImage, verify mediaPaths populated and content contains `[image: photo]`
4. **`TestTextMessageUnchanged`** — existing text handling not broken

Mock approach: `httptest.NewServer` to intercept Feishu SDK HTTP calls.

## Dependencies

| PR | What it provides | Required for |
|----|-----------------|--------------|
| #555 | Agent pipeline `Media` field + `serializeMessages()` | LLM actually seeing images |
| #902 | Feishu outbound media + rich text | Sending images back |
| Ours | Feishu inbound image download | Receiving images |

All three needed for end-to-end: user sends image -> LLM sees it -> LLM responds.

## Deployment

1. PR our change to `sipeed/picoclaw:main` from `feat/feishu-inbound-image`
2. On `dev`, merge all three PRs + ours
3. Cross-compile arm64, deploy to ClawBox
4. Test: send cat photo via Feishu, verify LLM describes it
5. If issues found in PR #555 or #902, file comments on those PRs

## Future: Full File Type Inbound Support

Not in scope for this PR. After image support is stable and merged, a future effort should extend `handleMessageReceive()` to support all Feishu inbound file types:

| Message Type | SDK API | Content JSON | Priority |
|---|---|---|---|
| `image` | `Image.Get(image_key)` | `{"image_key":"..."}` | **This PR** |
| `file` | `File.Get(file_key)` | `{"file_key":"...","file_name":"..."}` | Next |
| `audio` | `File.Get(file_key)` | `{"file_key":"...","duration":...}` | Later |
| `media` (video) | `File.Get(file_key)` | `{"file_key":"...","image_key":"..."}` | Later |
| `post` (rich text w/ inline images) | Parse post JSON, `Image.Get()` per image | Nested JSON array | Much later |

This is tracked as a long-term goal, not planned for any near-term milestone.
