#!/usr/bin/env python3
"""
picoclaw-voice 演示客户端

完整演示 ASR → LLM → TTS 流水线及 picoclaw 协议（音频格式协商、llm thinking 事件）。

支持两种 TTS 下行格式（由服务端协商）：
  - opus：接收 Opus 包，用 opuslib 解码后播放（doubao TTS）
  - pcm：接收原始 PCM s16le 数据，直接播放（Fish Speech TTS）

协议流程：
  1. 客户端发 hello
  2. 服务端回 hello，携带 asr_params / tts_params（协商音频格式）
  3. 按空格 → 发 listen.start，流式发送麦克风音频帧
  4. 再按空格 → 发 listen.end，停止发送
  5. 服务端推送事件：stt / llm / tts
  6. 服务端在 tts 期间推送二进制 Opus 帧

用法：
    python client.py --url ws://HOST:18765/xiaozhi/v1/
    python client.py --audio-file /tmp/input.wav   # 文件模式，自动发送后退出

依赖：pip install -r requirements.txt
"""

import argparse
import json
import os
import platform
import queue
import signal
import sys
import threading
import time
import uuid

try:
    import miniaudio
    import pyaudio
    import websocket
except ImportError as e:
    print(f"缺少依赖：{e}")
    print("请先执行：pip install -r requirements.txt")
    sys.exit(1)

try:
    import opuslib
    _HAS_OPUSLIB = True
except ImportError:
    _HAS_OPUSLIB = False

SAMPLE_RATE = 16000
CHANNELS = 1
FRAME_DURATION_MS = 60
FRAME_SAMPLES = SAMPLE_RATE * FRAME_DURATION_MS // 1000  # 960 samples = 60ms
PYAUDIO_FORMAT = pyaudio.paInt16


class PicoclawVoiceClient:

    def __init__(self, url: str):
        self.url = url
        self.device_id = "demo:" + uuid.uuid4().hex[:8]
        self.ws = None

        # 握手后由服务端 hello 的 audio_params / tts_params 更新
        self.audio_fmt = "pcm"       # 上行 ASR 格式
        self.tts_sample_rate = SAMPLE_RATE  # 下行 TTS 采样率
        self.tts_channels = CHANNELS        # 下行 TTS 声道数
        self.tts_format = "opus"            # 下行 TTS 编码格式
        self.handshake_done = threading.Event()
        self._stop = threading.Event()  # 通知播放线程退出
        self._is_listening = False      # 是否正在发送音频帧给服务端
        self._pushing = False           # 空格是否正被按住
        self._file_mode = False         # 文件输入模式

        self.pa = pyaudio.PyAudio()
        self._audio_device = self._find_pulse_device()
        self.audio_out_q: queue.Queue = queue.Queue()
        self.dec = None  # 据 tts_format 延迟初始化
        self.enc = None  # 据 audio_fmt 延迟初始化

    def _log(self, msg: str = ""):
        # raw tty 模式（Linux）下 \n 不回行首，必须用 \r\n
        end = "\n" if platform.system() == "Windows" else "\r\n"
        sys.stdout.write(msg + end)
        sys.stdout.flush()

    def _find_pulse_device(self):
        """返回 PulseAudio 设备索引，找不到时返回 None（让 PyAudio 用系统默认）。"""
        for i in range(self.pa.get_device_count()):
            info = self.pa.get_device_info_by_index(i)
            if "pulse" in info["name"].lower():
                return i
        return None

    # ── WebSocket 回调 ────────────────────────────────────────────────────────

    def on_open(self, ws):
        self._log(f"[连接] {self.url}")
        ws.send(json.dumps({
            "type": "hello",
            "version": 3,
            "transport": "websocket",
            "device_id": self.device_id,
            "audio_params": {
                "format": "pcm",
                "sample_rate": SAMPLE_RATE,
                "channels": CHANNELS,
                "frame_duration": FRAME_DURATION_MS,
            },
        }))

    def on_message(self, ws, message):
        if isinstance(message, bytes):
            # 下行 Opus 音频帧（tts.sentence_start/end 之间）
            self.audio_out_q.put(message)
            return
        try:
            msg = json.loads(message)
        except json.JSONDecodeError:
            return

        mtype = msg.get("type", "")

        if mtype == "hello":
            # 上行：ASR 期望格式（客户端发送音频给服务端）
            asr_params = msg.get("asr_params", {})
            self.audio_fmt = asr_params.get("format", "pcm")
            # 下行：TTS 输出格式（服务端发送 Opus 音频给客户端）
            tts_params = msg.get("tts_params", {})
            self.tts_sample_rate = tts_params.get("sample_rate", SAMPLE_RATE)
            self.tts_channels = tts_params.get("channels", CHANNELS)
            self.tts_format = tts_params.get("format", "opus")
            if self.tts_format == "opus":
                if not _HAS_OPUSLIB:
                    raise RuntimeError("TTS 格式为 opus，但 opuslib 未安装：pip install opuslib")
                self.dec = opuslib.Decoder(self.tts_sample_rate, self.tts_channels)
            if self.audio_fmt == "opus":
                if not _HAS_OPUSLIB:
                    raise RuntimeError("ASR 上行格式为 opus，但 opuslib 未安装：pip install opuslib")
                self.enc = opuslib.Encoder(SAMPLE_RATE, CHANNELS, opuslib.APPLICATION_VOIP)
            self._log(f"[握手] session_id={msg.get('session_id', '')}")
            self._log(f"  ↑ ASR上行: {self.audio_fmt} {asr_params.get('sample_rate')}Hz {asr_params.get('channels')}ch")
            self._log(f"  ↓ TTS下行: {self.tts_format} {self.tts_sample_rate}Hz {self.tts_channels}ch")
            self.handshake_done.set()
            if self._file_mode:
                self._log("[就绪] 文件模式，正在自动发送音频...")
            else:
                self._log("[就绪] 按空格开始/停止录音，Ctrl+C 退出")

        elif mtype == "stt":
            state = msg.get("state", "")
            text = msg.get("text", "")
            if state == "recognizing" and text:
                self._log(f"  [流式识别结果] {text}")
            elif state == "stop":
                if text:
                    self._log(f"  [你] {text}")
                else:
                    self._log("  [你] （静音/未识别）")

        elif mtype == "llm":
            state = msg.get("state", "")
            text = msg.get("text", "")
            if state == "thinking_start":
                self._log("  ⏳ 思考中...")
            elif state == "thinking_end":
                self._log(f"  ✓ 思考完成（{msg.get('duration_ms', 0)}ms）")
            elif text:
                self._log(f"  [LLM] {text}")

        elif mtype == "tts":
            state = msg.get("state", "")
            if state == "start":
                self._log("  [合成中]")
            elif state == "stop":
                self._log("  [合成完毕]")
                if self._file_mode:
                    threading.Thread(target=lambda: (time.sleep(0.5), self.ws.close()), daemon=True).start()
                else:
                    self._log("")
                    self._log("[就绪] 按空格开始/停止录音，Ctrl+C 退出")
            elif state == "abort":
                self._log("  [合成中断]")
                if not self._file_mode:
                    self._log("")
                    self._log("[就绪] 按空格开始/停止录音，Ctrl+C 退出")

    def on_error(self, ws, error):
        self._log(f"[错误] {error}")

    def on_close(self, ws, code, msg):
        self._log(f"[断开] {code} {msg}")
        self.handshake_done.set()

    # ── PTT 按键控制 ───────────────────────────────────────────────────────────

    def _key_thread(self):
        """跨平台按键读取：空格切换 PTT，Ctrl+C 退出。"""
        self.handshake_done.wait()
        if platform.system() == "Windows":
            import msvcrt
            while not self._stop.is_set():
                if msvcrt.kbhit():
                    ch = msvcrt.getwch()
                    if ch == ' ':
                        self._push_start() if not self._pushing else self._push_end()
                    elif ch == '\x03':  # Ctrl+C
                        self._stop.set()
                        if self.ws:
                            self.ws.close()
                        os.kill(os.getpid(), signal.SIGINT)
                        break
                else:
                    threading.Event().wait(0.05)
        else:
            import select, termios, tty
            fd = sys.stdin.fileno()
            old = termios.tcgetattr(fd)
            try:
                tty.setraw(fd)
                while not self._stop.is_set():
                    r, _, _ = select.select([fd], [], [], 0.1)
                    if not r:
                        continue
                    ch = os.read(fd, 1)
                    if ch == b' ':
                        self._push_start() if not self._pushing else self._push_end()
                    elif ch in (b'\x03', b'\x1c'):
                        self._stop.set()
                        if self.ws:
                            self.ws.close()
                        os.kill(os.getpid(), signal.SIGINT)
                        break
            finally:
                termios.tcsetattr(fd, termios.TCSADRAIN, old)

    def _push_start(self):
        """空格按下：发 listen.start，开始流式发送音频。"""
        if self._pushing or self._stop.is_set():
            return
        self._pushing = True
        sid = str(uuid.uuid4())
        self._is_listening = True
        try:
            self.ws.send(json.dumps({"type": "listen", "state": "start", "session_id": sid}))
            self._log("")
            self._log("─" * 40)
            self._log("[●] 音频数据发送中...  (再按空格停止)")
        except Exception:
            self._is_listening = False
            self._pushing = False

    def _push_end(self):
        """空格松开：停止发送音频，发 listen.end。"""
        if not self._pushing:
            return
        self._pushing = False
        self._is_listening = False
        try:
            self.ws.send(json.dumps({"type": "listen", "state": "end"}))
            self._log("[○] 音频数据发送已停止...")
        except Exception:
            pass

    # ── 下行音频播放（兼容 opus / pcm）────────────────────────────────────────

    def _playback_thread(self):
        self.handshake_done.wait()
        max_frame_size = self.tts_sample_rate * 120 // 1000
        bytes_per_frame = self.tts_channels * 2  # s16le

        _buf  = bytearray()
        _lock = threading.Lock()

        def _feed_loop():
            while not self._stop.is_set():
                try:
                    frame = self.audio_out_q.get(timeout=0.1)
                    if self.tts_format == "opus":
                        pcm = self.dec.decode(frame, max_frame_size)
                    else:
                        pcm = frame  # PCM 直接使用
                    with _lock:
                        _buf.extend(pcm)
                except queue.Empty:
                    pass
                except Exception as e:
                    self._log(f'[解码错误] {e}')

        threading.Thread(target=_feed_loop, daemon=True).start()

        def _pcm_stream():
            num_frames = yield b""  # 预激，接收首次 send(num_frames)
            while True:
                if self._stop.is_set():
                    return  # 生成器结束，miniaudio 停止播放并退出 dev.start()
                needed = num_frames * bytes_per_frame
                with _lock:
                    if len(_buf) >= needed:
                        chunk = bytes(_buf[:needed])
                        del _buf[:needed]
                    else:
                        chunk = bytes(_buf) + b'\x00' * (needed - len(_buf))
                        _buf.clear()
                num_frames = yield chunk

        gen = _pcm_stream()
        next(gen)
        with miniaudio.PlaybackDevice(
            output_format=miniaudio.SampleFormat.SIGNED16,
            nchannels=self.tts_channels,
            sample_rate=self.tts_sample_rate,
        ) as dev:
            dev.start(gen)
            self._stop.wait()

    # ── 文件模式：从音频文件读取并发送 ASR 帧 ──────────────────────────────────

    def _file_record_thread(self, audio_file: str):
        """读取音频文件（自动重采样到 16kHz mono），以实时速率流式发送给 ASR。"""
        self.handshake_done.wait()
        try:
            decoded = miniaudio.decode_file(
                audio_file,
                output_format=miniaudio.SampleFormat.SIGNED16,
                nchannels=CHANNELS,
                sample_rate=SAMPLE_RATE,
            )
        except Exception as e:
            self._log(f"[文件读取失败] {e}")
            self._stop.set()
            return

        pcm_data = bytes(decoded.samples)
        frame_bytes = FRAME_SAMPLES * CHANNELS * 2  # s16le，每帧字节数
        duration_s = len(pcm_data) / 2 / SAMPLE_RATE

        sid = str(uuid.uuid4())
        self.ws.send(json.dumps({"type": "listen", "state": "start", "session_id": sid}))
        self._log("─" * 40)
        self._log(f"[文件模式] 发送: {os.path.basename(audio_file)}  ({duration_s:.1f}s)")

        offset = 0
        while offset < len(pcm_data) and not self._stop.is_set():
            chunk = pcm_data[offset:offset + frame_bytes]
            if len(chunk) < frame_bytes:
                chunk = chunk + b'\x00' * (frame_bytes - len(chunk))
            if self.audio_fmt == "opus":
                chunk = bytes(self.enc.encode(chunk, FRAME_SAMPLES))
            self.ws.send(chunk, opcode=websocket.ABNF.OPCODE_BINARY)
            offset += frame_bytes
            time.sleep(FRAME_DURATION_MS / 1000.0)  # 模拟实时节奏

        self.ws.send(json.dumps({"type": "listen", "state": "end"}))
        self._log("[文件模式] 发送完毕，等待识别...")

    # ── 麦克风录音：持续采集，_is_listening 控制是否发送 ─────────────────────

    def _record_thread(self):
        """持续录制麦克风音频，仅在 _is_listening=True 时向服务端发送帧。"""
        self.handshake_done.wait()

        stream = self.pa.open(
            format=PYAUDIO_FORMAT, channels=CHANNELS,
            rate=SAMPLE_RATE, input=True,
            input_device_index=self._audio_device,
            frames_per_buffer=FRAME_SAMPLES,
        )
        try:
            while not self._stop.is_set():
                try:
                    pcm_bytes = stream.read(FRAME_SAMPLES, exception_on_overflow=False)
                except OSError:
                    break
                if not self._is_listening:
                    continue
                if self.audio_fmt == "opus":
                    frame = bytes(self.enc.encode(pcm_bytes, FRAME_SAMPLES))
                else:
                    frame = pcm_bytes
                try:
                    self.ws.send(frame, opcode=websocket.ABNF.OPCODE_BINARY)
                except Exception:
                    break
        finally:
            stream.stop_stream()
            stream.close()

    # ── 主入口 ────────────────────────────────────────────────────────────────

    def run(self, audio_file: str = ""):
        self._file_mode = bool(audio_file)
        ws = websocket.WebSocketApp(
            self.url,
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close,
        )
        self.ws = ws

        playback_t = threading.Thread(target=self._playback_thread, daemon=True)
        playback_t.start()
        if audio_file:
            threading.Thread(target=self._file_record_thread, args=(audio_file,), daemon=True).start()
        else:
            threading.Thread(target=self._record_thread, daemon=True).start()
            threading.Thread(target=self._key_thread, daemon=True).start()

        try:
            ws.run_forever()
        except KeyboardInterrupt:
            pass
        self._stop.set()
        playback_t.join(timeout=1.0)
        self.pa.terminate()


def main():
    if platform.system() != "Windows":
        # 压制 ALSA/PortAudio C 层噪声（不影响 Python stderr）
        _devnull = os.open(os.devnull, os.O_WRONLY)
        os.dup2(_devnull, 2)
        os.close(_devnull)

    parser = argparse.ArgumentParser(description="picoclaw-voice 演示客户端")
    parser.add_argument("--url", default="ws://127.0.0.1:18765/xiaozhi/v1/",
                        help="服务端 WebSocket 地址")
    parser.add_argument("--audio-file", default="",
                        help="音频文件路径（代替麦克风，自动发送后退出）")
    args = parser.parse_args()

    PicoclawVoiceClient(args.url).run(audio_file=args.audio_file)


if __name__ == "__main__":
    main()
