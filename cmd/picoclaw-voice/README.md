# picoclaw-voice

xiaozhi WebSocket 语音网关，将 xiaozhi-esp32 协议设备（ESP32、桌面客户端等）接入 PicoClaw AI 引擎，提供完整的 ASR → LLM → TTS 语音对话流水线。

## 功能特性

- **WebSocket 服务端**：实现 [xiaozhi-esp32 协议](../../docs/channels/xiaozhi/README.zh.md) v3，路径 `/xiaozhi/v1/`
- **音频格式协商**：服务端在 hello 握手阶段通过 `asr_params` / `tts_params` 下发上下行格式（PCM 或 Opus），客户端自动适配
- **三段并发流水线**：实时 ASR（接收音频帧即转写）→ LLM 推理（流式断句）→ TTS 合成推流，端到端首字延迟最优
- **多 provider 支持**：ASR 支持豆包（云端）和 FunASR（本地），TTS 支持豆包（云端）和 Fish Speech（本地），可自由混搭
- **设备管理**：`device_id` 注册表，同一设备重连时自动驱逐旧会话
- **多轮记忆**：通过 `PICOCLAW_VOICE_OWNER_ID` 控制跨设备/跨频道共享同一对话记忆
- **LLM thinking 通知**：支持推理模型 `<think>` 块，向客户端发送 `llm.thinking_start/end` 事件

## 快速开始

### 直接运行（推荐）

```bash
# 1. 构建
cd cmd/picoclaw-voice
go build -o picoclaw-voice .

# 2. 配置
cp .env.example .env
$EDITOR .env   # 至少填写 LLM 配置和 ASR/TTS 凭证

# 3. 启动
./picoclaw-voice
```

### Docker

```bash
# 从 repo 根目录构建（Dockerfile context 需要整个 repo）
docker build -f cmd/picoclaw-voice/Dockerfile -t picoclaw-voice .

docker run -d --name picoclaw-voice \
  --env-file cmd/picoclaw-voice/.env \
  -v ~/.picoclaw:/root/.picoclaw \
  -p 8765:8765 \
  picoclaw-voice
```

### 连接到 picoclaw

picoclaw-voice 读取 `~/.picoclaw/config.json`（与 picoclaw 主进程共享同一份配置）以获取 LLM provider、记忆后端和 MCP 工具配置。

确保 picoclaw 已完成 [初始化配置](https://github.com/sipeed/picoclaw#configuration) 后再启动 picoclaw-voice。

## 环境变量

> 加载优先级（高→低）：Shell 进程环境 → `.env` 文件 → 代码默认值

### 通用

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PICOCLAW_VOICE_LISTEN` | `:8765` | WebSocket 监听地址 |
| `PICOCLAW_VOICE_OWNER_ID` | *(空)* | 固定 owner_id；空时每台设备独立记忆 |
| `PICOCLAW_CONFIG` | `~/.picoclaw/config.json` | picoclaw 配置文件路径 |
| `PICOCLAW_HOME` | `~/.picoclaw` | picoclaw home 目录（次优先） |

### ASR 配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PICOCLAW_VOICE_ASR_PROVIDER` | `doubao` | ASR 供应商：`doubao` \| `funasr` |
| `PICOCLAW_VOICE_APPID` | | 火山引擎 AppID（`doubao` provider 兜底） |
| `PICOCLAW_VOICE_TOKEN` | | 火山引擎 Token（`doubao` provider 兜底） |
| `PICOCLAW_VOICE_ASR_APPID` | | 单独覆盖 ASR AppID |
| `PICOCLAW_VOICE_ASR_TOKEN` | | 单独覆盖 ASR Token |
| `PICOCLAW_VOICE_ASR_CLUSTER` | `bigmodel_transcribe` | doubao ASR 集群 |
| `PICOCLAW_VOICE_ASR_RESOURCE_ID` | `volc.bigasr.sauc.duration` | doubao ASR 资源 ID（见下方模型列表） |
| `PICOCLAW_VOICE_ASR_WS_URL` | `wss://127.0.0.1:10095` | funasr WebSocket 地址（内置自签名证书，自动跳过校验） |
| `PICOCLAW_VOICE_ASR_MODE` | `2pass` | funasr 识别模式：`2pass` \| `online` \| `offline` |

### TTS 配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PICOCLAW_VOICE_TTS_PROVIDER` | `doubao` | TTS 供应商：`doubao` \| `fishspeech` |
| `PICOCLAW_VOICE_TTS_APPID` | | 单独覆盖 TTS AppID（doubao） |
| `PICOCLAW_VOICE_TTS_TOKEN` | | 单独覆盖 TTS Token（doubao） |
| `PICOCLAW_VOICE_TTS_CLUSTER` | `volcano_tts` | doubao TTS 集群 |
| `PICOCLAW_VOICE_TTS_VOICE` | | doubao 音色 ID，见[火山引擎音色列表](https://www.volcengine.com/docs/6561/97465) |
| `PICOCLAW_VOICE_TTS_API_URL` | `http://127.0.0.1:8080` | fishspeech HTTP 服务地址 |
| `PICOCLAW_VOICE_TTS_API_KEY` | | fishspeech Bearer Token（不需要鉴权时留空） |
| `PICOCLAW_VOICE_TTS_REFERENCE_ID` | | fishspeech 参考音色 ID（留空用服务默认） |
| `PICOCLAW_VOICE_TTS_SAMPLE_RATE` | `0` | fishspeech 输出采样率（0 = provider 默认，通常为 44100 Hz） |

## ASR Provider 说明

### doubao（豆包，默认）

调用[火山引擎大模型语音识别](https://console.volcengine.com/speech/service/10)，流式实时转写。

**凭证填写方式（两选一）：**
- **API Key 模式**（推荐）：只填 `PICOCLAW_VOICE_TOKEN`（UUID 格式），`APPID` 留空
- **App Key 模式**：填数字格式 `PICOCLAW_VOICE_APPID` + `PICOCLAW_VOICE_TOKEN`

**模型选择（`PICOCLAW_VOICE_ASR_RESOURCE_ID`）：**
- `volc.bigasr.sauc.duration` — 豆包流式 ASR 1.0（默认，稳定）
- `volc.seedasr.sauc.duration` — 豆包流式 ASR 2.0 Seed（更高精度）

### funasr（本地免费）

连接本机运行的 [FunASR](https://github.com/modelscope/FunASR) 推理服务，免费且支持中文。

**快速启动 FunASR 容器：**

```bash
# 从 repo 根目录执行
docker compose -f docker/docker-compose.asr-tts.yml up -d funasr
```

模型首次启动时自动从 ModelScope 下载（约 1 GB），后续启动使用缓存（`~/models/funasr/`）。

## TTS Provider 说明

### doubao（豆包，默认）

调用[火山引擎语音合成](https://console.volcengine.com/speech/service/8)，输出 Opus 音频流。凭证配置同 doubao ASR。

### fishspeech（本地免费，推荐 GPU）

连接本机运行的 [Fish Speech](https://github.com/fishaudio/fish-speech) v1.5.x 服务，输出 PCM 音频流（44100 Hz），效果自然、中文优秀。

**快速启动 Fish Speech 容器（需 NVIDIA GPU）：**

```bash
# 安装 NVIDIA Container Toolkit（首次）
# 参见：https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html

# 从 repo 根目录执行
docker compose -f docker/docker-compose.asr-tts.yml up -d fishspeech
```

模型已内置于 `fishaudio/fish-speech:v1.5.1` 镜像，无需单独挂载。

## 本地免费方案（FunASR + Fish Speech）

一键启动完整本地推理栈：

```bash
docker compose -f docker/docker-compose.asr-tts.yml up -d
```

然后在 `.env` 中切换 provider：

```dotenv
PICOCLAW_VOICE_ASR_PROVIDER=funasr
PICOCLAW_VOICE_ASR_WS_URL=wss://127.0.0.1:10095

PICOCLAW_VOICE_TTS_PROVIDER=fishspeech
PICOCLAW_VOICE_TTS_API_URL=http://127.0.0.1:8080
```

也可以混搭，例如云端 ASR + 本地 TTS，只需分别设置对应的 `PROVIDER` 变量即可。

## 协议说明

picoclaw-voice 实现 xiaozhi-esp32 协议 v3，并在此基础上做了少量扩展。完整协议文档见 [docs/channels/xiaozhi/README.zh.md](../../docs/channels/xiaozhi/README.zh.md)。

**音频格式协商流程：**

1. 客户端发送 `hello`，携带上行 `audio_params`（建议 PCM 16kHz mono）
2. 服务端回 `hello`，携带：
   - `asr_params`：服务端期望的**上行**音频格式（客户端按此格式发送语音帧）
   - `tts_params`：服务端实际的**下行** TTS 格式（doubao 输出 Opus，fishspeech 输出 PCM）
3. 后续二进制帧格式以协商结果为准

**picoclaw 扩展字段：**

| 方向 | 字段 | 说明 |
|------|------|------|
| 客户端→服务端 | `listen.memory_id` | 指定 LLM 多轮记忆 key，相同 key 共享对话上下文（跨设备/跨会话） |
| 服务端→客户端 | `hello.session_id` | 连接级 UUID，用于日志关联 |
| 服务端→客户端 | `llm`（新消息类型） | LLM 推理通知：`{"type":"llm","text":"..."}` 断句文本；`thinking_start` / `thinking_end` 思考链事件 |

## Demo

Python 演示客户端，完整演示握手→录音→流式 ASR→LLM→TTS 播放全流程。

**系统依赖（仅 doubao TTS/ASR 的 Opus 模式需要）：**

```bash
# Debian/Ubuntu
sudo apt install portaudio19-dev libopus-dev

# macOS
brew install portaudio opus
```

**运行：**

```bash
cd cmd/picoclaw-voice/demo
pip install -r requirements.txt

# 麦克风模式：按空格开始/停止录音
python client.py --url ws://localhost:8765/xiaozhi/v1/

# 文件模式：自动发送音频文件后退出
python client.py --audio-file /tmp/input.wav
```

详见 [demo/README.md](demo/README.md)。
