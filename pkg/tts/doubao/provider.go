// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package doubao 通过豆包（火山引擎）TTS WebSocket 流式接口合成语音。
// 协议参考：wss://openspeech.bytedance.com/api/v1/tts/ws_binary
//
// 服务端帧类型（data[1]>>4）：
//
//	0x0B（音频帧）：4字节头 + 4字节seq + 4字节载荷长度 + Ogg Opus 数据
//	0x0C（合成结束）/0x0F（错误）：4字节头 + 4字节载荷长度 + gzip JSON
//
// 鉴权：Authorization: Bearer;{token}（火山引擎非标准分号格式）
// 无 CGO，纯 Go 实现。
package doubao

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/tts"
)

const (
	defaultWSURL   = "wss://openspeech.bytedance.com/api/v1/tts/ws_binary"
	defaultCluster = "volcano_tts"
	defaultVoice   = "zh_female_wanwanxiaohe_moon_bigtts"
)

// 二进制帧协议常量
const (
	msgTypeFullClientRequest = byte(0x01)
	msgTypeServerAudio       = byte(0x0B) // 音频帧，含 seq + payload_size
	msgTypeServerError       = byte(0x0F) // 错误帧，含 payload_size（无 seq）

	flagNormal  = byte(0x00)
	serialJSON  = byte(0x01)
	compressGZP = byte(0x01)
)

func init() {
	tts.Register("doubao", func(cfg map[string]any) (tts.Provider, error) {
		return newProvider(cfg)
	})
}

type provider struct {
	appid        string
	accessToken  string
	cluster      string
	defaultVoice string
	wsURL        string
	dialer       *websocket.Dialer
}

func newProvider(cfg map[string]any) (*provider, error) {
	get := func(key string) string {
		v, _ := cfg[key].(string)
		return v
	}
	p := &provider{
		appid:        get("appid"),
		accessToken:  get("access_token"),
		cluster:      get("cluster"),
		defaultVoice: get("voice"),
		wsURL:        get("ws_url"),
	}
	if p.accessToken == "" {
		return nil, fmt.Errorf("tts/doubao: access_token is required")
	}
	if p.cluster == "" {
		p.cluster = defaultCluster
	}
	if p.defaultVoice == "" {
		p.defaultVoice = defaultVoice
	}
	if p.wsURL == "" {
		p.wsURL = defaultWSURL
	}
	// 不走代理直连火山引擎，避免本地 http_proxy 拦截 WebSocket。
	p.dialer = &websocket.Dialer{
		NetDialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		HandshakeTimeout: 10 * time.Second,
	}
	return p, nil
}

func (p *provider) Name() string { return "doubao" }

func (p *provider) AudioFormat() tts.AudioFormat {
	return tts.AudioFormat{Codec: "opus", SampleRate: 16000, Channels: 1}
}

// SynthesizeFrames 通过豆包 TTS WebSocket 流式接口合成语音。
// 连接建立后立即发送合成请求（含文字），服务端边合成边推送原始 Opus 帧，
// 每帧到达即触发 onFrame 回调，无需等待整句合成完成。
func (p *provider) SynthesizeFrames(ctx context.Context, text, voice string, onFrame func([]byte)) error {
	if voice == "" {
		voice = p.defaultVoice
	}

	// 火山引擎 TTS WS 接口要求 Authorization: Bearer;{token}（分号分隔，非标准空格）。
	// appid 在 JSON 请求体 app.appid 里传，Bearer 头仅携带 token。
	headers := http.Header{
		"Authorization":    {"Bearer;" + p.accessToken},
		"X-Api-Connect-Id": {uuid.New().String()},
	}

	conn, resp, err := p.dialer.DialContext(ctx, p.wsURL, headers)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			return fmt.Errorf("tts/doubao: dial: %w (HTTP %d: %s)", err, resp.StatusCode, bytes.TrimSpace(body))
		}
		return fmt.Errorf("tts/doubao: dial: %w", err)
	}
	defer conn.Close()

	// ctx 取消时关闭连接，使 ReadMessage 立即解除阻塞。
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// 发送合成请求帧（包含完整文字；operation="submit" 触发流式推送）
	// 注意：火山引擎 TTS 二进制 WS 协议中 app.token 需加 "access_" 前缀，
	// 而 HTTP Authorization Bearer 使用原始 token。
	reqPayload := map[string]any{
		"app": map[string]any{
			"appid":   p.appid,
			"token":   "access_" + p.accessToken,
			"cluster": p.cluster,
		},
		"user": map[string]any{"uid": "picoclaw"},
		"audio": map[string]any{
			"voice_type": voice,
			// ogg_opus：服务端推送 Ogg 容器包装的 Opus 数据；
			// 服务端 API encoding 与内部 AudioFormat codec 解耦，
			// 此处通过 io.Pipe + ParseOggOpusPackets 流式解包后回调原始 Opus 帧。
			"encoding": "ogg_opus",
			"rate":     16000,
			"channel":  1,
		},
		"request": map[string]any{
			"reqid":         uuid.New().String(),
			"text":          text,
			"text_type":     "plain",
			"operation":     "submit", // submit = 流式推送；query = HTTP 一次性返回
			"with_frontend": 1,
			"frontend_type": "unitTson",
		},
	}
	initFrame, err := buildTTSFrame(reqPayload)
	if err != nil {
		return fmt.Errorf("tts/doubao: build request frame: %w", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, initFrame); err != nil {
		return fmt.Errorf("tts/doubao: send request: %w", err)
	}

	// 用 io.Pipe 将流式 Ogg 载荷接入 ParseOggOpusPackets：
	// 主循环写 → 解析协程读，边收 WS 帧边解包 Opus，实现真正流式回调。
	pr, pw := io.Pipe()
	parseErrCh := make(chan error, 1)
	go func() {
		parseErrCh <- tts.ParseOggOpusPackets(pr, onFrame)
	}()

	frameCount := 0
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				pw.CloseWithError(ctx.Err())
				return ctx.Err()
			}
			pw.CloseWithError(err)
			break
		}
		audio, isLast, parseErr := parseTTSFrame(frameCount, msg)
		if parseErr != nil {
			pw.CloseWithError(parseErr)
			return parseErr
		}
		if len(audio) > 0 {
			frameCount++
			if _, werr := pw.Write(audio); werr != nil {
				return werr
			}
		}
		if isLast {
			pw.Close()
			break
		}
	}

	if err := <-parseErrCh; err != nil && ctx.Err() == nil {
		return err
	}
	log.Printf("tts/doubao: streamed %d ogg pages for %q", frameCount, truncate(text, 20))
	return nil
}

// parseTTSFrame 解析豆包 TTS WebSocket 服务端响应帧。
//
// 帧类型由 data[1]>>4 决定，不同类型结构不同：
//
//	msgType=0x0B（音频帧）：[4:8]=seq int32（负=末帧）, [8:12]=载荷长度, [12:]=Ogg数据
//	msgType=其他（0x0C合成结束/0x0F错误）：[4:8]=载荷长度, [8:]=gzip JSON，无 seq 字段
func parseTTSFrame(idx int, data []byte) (audio []byte, isLast bool, err error) {
	if len(data) < 4 {
		return nil, false, fmt.Errorf("tts/doubao: frame too short (%d bytes)", len(data))
	}

	msgType := data[1] >> 4
	compressType := data[2] & 0x0F

	if msgType != 0x0B {
		// 非音频帧（0x0C=合成结束，0x0F=服务端错误）：[4:8]=载荷长度，[8:]=gzip JSON
		if len(data) >= 8 {
			payloadSize := binary.BigEndian.Uint32(data[4:8])
			end := 8 + int(payloadSize)
			if end <= len(data) {
				payload := data[8:end]
				if compressType == compressGZP {
					if decoded, e := gunzipBytes(payload); e == nil {
						payload = decoded
					}
				}
				if len(payload) > 2 && payload[0] == '{' {
					var resp struct {
						Code int `json:"code"`
					}
					if json.Unmarshal(payload, &resp) == nil && resp.Code != 0 && resp.Code != 200 {
						s := string(payload)
						if len(s) > 256 {
							s = s[:256]
						}
						log.Printf("tts/doubao: frame[%d] server error: %s", idx, s)
						return nil, true, fmt.Errorf("tts/doubao: server error: %s", s)
					}
				}
			}
		}
		return nil, true, nil // 非音频帧均视为流结束信号
	}

	// msgType=0x0B：音频帧，[4:8]=seq，[8:12]=载荷长度，[12:]=Ogg数据
	if len(data) == 8 {
		seq := int32(binary.BigEndian.Uint32(data[4:8]))
		return nil, seq < 0, nil
	}
	if len(data) < 12 {
		return nil, false, fmt.Errorf("tts/doubao: audio frame too short (%d bytes)", len(data))
	}
	seq := int32(binary.BigEndian.Uint32(data[4:8]))
	payloadSize := binary.BigEndian.Uint32(data[8:12])
	end := 12 + int(payloadSize)
	if end > len(data) {
		return nil, false, fmt.Errorf("tts/doubao: payload size %d exceeds frame length %d", payloadSize, len(data))
	}
	return data[12:end], seq < 0, nil
}

// buildTTSFrame 将 JSON payload gzip 压缩后封装为豆包二进制协议帧（客户端 → 服务端）。
func buildTTSFrame(payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	compressed, err := gzipBytes(data)
	if err != nil {
		return nil, err
	}
	hdr := [4]byte{
		(0x01 << 4) | 0x01,
		(msgTypeFullClientRequest << 4) | flagNormal,
		(serialJSON << 4) | compressGZP,
		0x00,
	}
	frame := make([]byte, 0, 8+len(compressed))
	frame = append(frame, hdr[:]...)
	frame = binary.BigEndian.AppendUint32(frame, uint32(len(compressed)))
	frame = append(frame, compressed...)
	return frame, nil
}

func gunzipBytes(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
