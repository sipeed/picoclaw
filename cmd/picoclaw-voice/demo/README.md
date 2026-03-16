# picoclaw-voice demo

演示如何通过 xiaozhi 协议连接 picoclaw-voice，完整体验 ASR → LLM → TTS 流水线。

支持两种 TTS 下行音频格式（由服务端协商决定，客户端自动适配）：
- **opus**：doubao TTS 输出，需要 `opuslib` 解码
- **pcm**：Fish Speech TTS 输出，直接播放 s16le PCM

## 系统依赖

`opuslib` 仅在 doubao TTS（Opus 格式）时需要：

```bash
# Debian/Ubuntu
sudo apt install portaudio19-dev libopus-dev

# macOS
brew install portaudio opus
```

使用 Fish Speech TTS（PCM 格式）时无需 opus 系统库。

## 安装 Python 依赖

```bash
pip install -r requirements.txt
```

`opuslib` 为可选依赖，仅 doubao Opus 格式时使用。

## 运行

```bash
# 麦克风模式：按空格开始录音，再按空格停止
python client.py

# 指定服务地址
python client.py --url ws://192.168.1.100:8765/xiaozhi/v1/

# 文件模式：自动发送音频文件，TTS 播放完毕后退出
python client.py --audio-file /tmp/input.wav
```

## 输出示例

```
[连接] ws://localhost:8765/xiaozhi/v1/
[握手] session_id=a1b2c3d4-...
  ↑ ASR上行: pcm 16000Hz 1ch
  ↓ TTS下行: opus 24000Hz 1ch
[就绪] 按空格开始/停止录音，Ctrl+C 退出

────────────────────────────────────────
[●] 音频数据发送中...  (再按空格停止)
[○] 音频数据发送已停止...
  [流式识别结果] 你好
  [你] 你好，今天天气怎么样？
  [LLM] 您好！今天北京的天气是晴天，气温约 25 度...
  [合成中]
  [合成完毕]
```

