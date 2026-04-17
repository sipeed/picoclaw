# TTS（文本转语音）

这个目录负责 PicoClaw 的语音合成能力。

如果你是第一次配置 TTS，可以参照下面这个流程：

1. 在 `model_list` 里添加一个支持 TTS 的模型。
2. 用 `voice.tts_model_name` 指向这个模型。
3. 在 `.security.yml` 里配置对应的 API Key。

## 快速推荐

对于大多数用户，建议优先从下面两种开始：

| 提供商 | 推荐理由 |
| --- | --- |
| [OpenAI](https://platform.openai.com/docs/guides/text-to-speech) | 这是 PicoClaw 当前最稳定、最直接的 TTS 路径。当前实现就是围绕 OpenAI 兼容的 `/audio/speech` 接口格式构建的，所以 OpenAI 是最稳妥的默认选择。 |
| [Xiaomi MiMo](https://platform.xiaomimimo.com) | 由于响应速度和语音音色对于中国用户更友好，MiMo 是一个不错的第二选择。 |

## TTS 配置是如何工作的

PicoClaw 不会把 TTS 的 API Key 放在 `voice` 配置里。

推荐方式是：

- `voice.tts_model_name` 用来选择 `model_list` 里的某个命名模型。
- 对应的 `model_list` 条目提供真实的 provider、model ID、`api_base` 和代理配置。
- `.security.yml` 负责保存该模型条目的 API Key。

这是当前推荐且受支持的配置方式。

## 推荐配置方式

### 方案 A：OpenAI

`config.json`

```json
{
  "voice": {
    "tts_model_name": "openai-tts"
  },
  "model_list": [
    {
      "model_name": "openai-tts",
      "model": "openai/tts-1"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  openai-tts:
    api_keys:
      - "sk-openai-your-key"
```

### 方案 B：Xiaomi MiMo

`config.json`

```json
{
  "voice": {
    "tts_model_name": "mimo-tts"
  },
  "model_list": [
    {
      "model_name": "mimo-tts",
      "model": "mimo/mimo-v2-tts"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  mimo-tts:
    api_keys:
      - "your-mimo-key"
```

如果你使用自定义的 MiMo 接口地址，也可以显式设置 `api_base`。如果不设置，PicoClaw 会自动使用该 provider 的默认地址。

## PicoClaw 当前实际发送的 TTS 请求

当前 TTS 运行时使用的是 OpenAI 兼容的语音合成请求，并带有以下默认值：

- Endpoint：`/audio/speech`
- 返回格式：`opus`
- Voice：`alloy`
- Model：来自你所选中的 `model_list` 条目

这意味着：

- `openai/tts-1` 可以自然工作。
- 其他 OpenAI 兼容 provider 也可能可用，前提是它们接受相同的请求格式。
- PicoClaw 目前还没有对用户暴露一个配置项来修改 TTS voice，当前固定为 `alloy`。

## PicoClaw 如何选择 TTS Provider

`DetectTTS` 会按下面顺序选择 TTS：

1. **首选路径**：根据 `voice.tts_model_name` 在 `model_list` 中找到对应模型。
2. 如果找到了匹配条目，并且它有 API Key，PicoClaw 就会使用这个模型条目的配置创建一个 OpenAI 兼容的 TTS provider。
3. **回退路径**：如果没有设置 `voice.tts_model_name`，或者该名字无法解析，PicoClaw 会扫描 `model_list`，选中第一个模型字符串里包含 `tts` 且带有 API Key 的条目。

回退扫描只是为了兼容旧行为。新配置建议始终显式设置 `voice.tts_model_name`。

## 关于 API Base 的处理方式

PicoClaw 会对 TTS 的 `api_base` 做规范化处理：

- 对 OpenAI 来说，像 `https://api.openai.com` 或 `https://api.openai.com/v1` 这样的地址，会自动变成 `https://api.openai.com/v1/audio/speech`。
- 对其他 OpenAI 兼容 provider，PicoClaw 会尽量保留你提供的基础路径，只确保它最终以 `/audio/speech` 结尾。
- 如果没有设置 `api_base`，并且模型前缀是已知 provider，PicoClaw 会自动使用该 provider 的默认地址。

## 常见错误

- `voice.tts_model_name` 指向了一个不存在的 `model_list` 名称。
- 在 `model_list` 里定义了 TTS 模型，但忘了在 `.security.yml` 中配置对应 API Key。
- 误以为 PicoClaw 会自动支持 provider 自定义 voice 参数。
- 使用了不兼容 OpenAI `/audio/speech` 请求格式的接口地址。

## 最小检查清单

在测试 `send_tts` 之前，请确认：

- `voice.tts_model_name` 能正确匹配某个 `model_list[].model_name`。
- `.security.yml` 中对应条目已经配置了有效 API Key。
- 你所选的 provider 支持 OpenAI 兼容的语音合成接口。
- 你选择的模型本身确实支持 TTS。
