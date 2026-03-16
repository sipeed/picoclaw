// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package funasr 对接 FunASR WebSocket 服务（iic/SenseVoiceSmall 等模型）。
// 支持实时流式 ASR（RealtimeProvider），中文识别效果优秀，完全本地部署，无需付费 API。
//
// 部署：
//
//	docker compose -f docker/docker-compose.asr-tts.yml up -d funasr
//
// 配置（环境变量）：
//
//	PICOCLAW_VOICE_ASR_WS_URL  WebSocket 地址，默认 wss://127.0.0.1:10095
//	PICOCLAW_VOICE_ASR_MODE    识别模式：2pass（默认）| online | offline
package funasr

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/asr"
)

func init() {
	asr.Register("funasr", func(cfg map[string]any) (asr.Provider, error) {
		return newProvider(cfg)
	})
}

// 默认使用 WSS，FunASR SDK 镜像内置自签名证书
const defaultWSURL = "wss://127.0.0.1:10095"

type provider struct {
	wsURL  string
	mode   string // "2pass" | "online" | "offline"
	dialer *websocket.Dialer
}

func newProvider(cfg map[string]any) (*provider, error) {
	wsURL, _ := cfg["ws_url"].(string)
	mode, _ := cfg["mode"].(string)
	if wsURL == "" {
		wsURL = defaultWSURL
	}
	if mode == "" {
		mode = "2pass"
	}
	return &provider{
		wsURL: wsURL,
		mode:  mode,
		dialer: &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			// FunASR 使用自签名证书，跳过校验
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}, nil
}

func (p *provider) Name() string { return "funasr" }

func (p *provider) AudioFormat() asr.AudioFormat {
	return asr.AudioFormat{Codec: "pcm", SampleRate: 16000, Channels: 1}
}

// Transcribe 批量模式：将所有帧合并后一次性送识别。
// 使用 OpenSession 实现，以复用实时 ASR 连接逻辑。
func (p *provider) Transcribe(ctx context.Context, frames [][]byte) (string, error) {
	sess, err := p.OpenSession(ctx, func(_ string, _ bool) {})
	if err != nil {
		return "", err
	}
	defer sess.Close()

	pcm := asr.MergeFrames(frames)
	// 每次推送 3200 字节（100ms @ 16kHz 16-bit mono），匹配 FunASR 推荐块大小
	const chunkSize = 3200
	for i := 0; i < len(pcm); i += chunkSize {
		end := i + chunkSize
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := sess.SendAudio(pcm[i:end], end >= len(pcm)); err != nil {
			return "", err
		}
	}
	return sess.Wait(ctx)
}

// OpenSession 实现 RealtimeProvider：建立 WebSocket 连接，启动收包 goroutine。
func (p *provider) OpenSession(ctx context.Context, callback asr.ResultCallback) (asr.StreamingSession, error) {
	// FunASR WebSocket 服务要求 Sec-WebSocket-Protocol: binary
	header := http.Header{"Sec-WebSocket-Protocol": {"binary"}}
	conn, _, err := p.dialer.DialContext(ctx, p.wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("funasr: dial %s: %w", p.wsURL, err)
	}

	// ctx 取消时关闭连接，使 ReadMessage 立即返回
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	cfg := map[string]any{
		"mode":           p.mode,
		"chunk_size":     []int{5, 10, 5},
		"chunk_interval": 10,
		"wav_name":       "picoclaw",
		"is_speaking":    true,
		"wav_format":     "pcm",
		"itn":            true,
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := conn.WriteMessage(websocket.TextMessage, cfgBytes); err != nil {
		conn.Close()
		return nil, fmt.Errorf("funasr: send config: %w", err)
	}

	sess := &streamSession{
		conn:     conn,
		callback: callback,
		mode:     p.mode,
		resultCh: make(chan string, 1),
		errCh:    make(chan error, 1),
	}
	go sess.readLoop()
	return sess, nil
}

// ── session ──────────────────────────────────────────────────────────────────

type streamSession struct {
	conn     *websocket.Conn
	callback asr.ResultCallback
	mode     string
	mu       sync.Mutex
	closed   bool
	resultCh chan string
	errCh    chan error
}

type funasrResult struct {
	Mode    string `json:"mode"`
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

// readLoop 持续读取服务端推送的 JSON 识别结果。
// 2pass 模式：2pass-online 为中间结果，2pass-offline 为最终高质量结果。
// online/offline 模式：is_final=true 表示识别完成。
func (s *streamSession) readLoop() {
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			// ctx 取消时 conn.Close() 触发此处，属于正常退出路径
			select {
			case s.errCh <- fmt.Errorf("funasr: read: %w", err):
			default:
			}
			return
		}

		var r funasrResult
		if err := json.Unmarshal(data, &r); err != nil {
			log.Printf("funasr: parse result: %v (raw: %s)", err, data)
			continue
		}

		// 判断是否为最终结果：
		// - 2pass-offline：FunASR 双路模式的最终离线结果（最高精度）
		// - offline：纯离线单路模式，收到结果即终态
		// - is_final=true：在线模式显式标记
		isFinal := r.Mode == "2pass-offline" || r.Mode == "offline" || (r.Mode == "online" && r.IsFinal)

		s.callback(r.Text, isFinal)

		if isFinal {
			select {
			case s.resultCh <- r.Text:
			default:
			}
			return
		}
	}
}

func (s *streamSession) SendAudio(frame []byte, isLast bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return asr.ErrSessionClosed
	}
	if len(frame) > 0 {
		if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			return fmt.Errorf("funasr: send audio: %w", err)
		}
	}
	if isLast {
		end, _ := json.Marshal(map[string]any{"is_speaking": false})
		if err := s.conn.WriteMessage(websocket.TextMessage, end); err != nil {
			return fmt.Errorf("funasr: send end: %w", err)
		}
	}
	return nil
}

func (s *streamSession) Wait(ctx context.Context) (string, error) {
	select {
	case text := <-s.resultCh:
		return text, nil
	case err := <-s.errCh:
		if ctx.Err() != nil {
			return "", asr.ErrSessionClosed
		}
		return "", err
	case <-ctx.Done():
		return "", asr.ErrSessionClosed
	}
}

func (s *streamSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		s.conn.Close()
	}
}
