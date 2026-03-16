// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package doubao implements the Doubao (火山引擎) streaming ASR provider.
// Protocol: binary WebSocket with a 4-byte header + gzip-compressed payloads.
// Docs: https://www.volcengine.com/docs/6561/80818
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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/asr"
)

func init() {
	asr.Register("doubao", func(cfg map[string]any) (asr.Provider, error) {
		return newProvider(cfg)
	})
}

const defaultASRURL = "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel"

// 时长包资源 ID 以控制台实际购买的为准：
// 模型1.0: volc.bigasr.sauc.duration  模型2.0(Seed): volc.seedasr.sauc.duration
const defaultResourceID = "volc.bigasr.sauc.duration"

// header byte layout (4 bytes):
//
//	[0]: (version=0x01 << 4) | header_size=0x01
//	[1]: (message_type << 4) | message_type_specific_flags
//	[2]: (serial_method << 4) | compression_type
//	[3]: reserved=0x00
const (
	msgTypeFullClientRequest = 0x01 // JSON init frame
	msgTypeAudioOnly         = 0x02 // audio frame
	msgTypeServerError       = 0x0F

	flagNormal    = 0x00
	flagLastFrame = 0x02

	serialJSON  = 0x01
	compressGZP = 0x01
)

type provider struct {
	appID      string
	token      string
	cluster    string
	resourceID string
	wsURL      string
	dialer     *websocket.Dialer
}

func newProvider(cfg map[string]any) (*provider, error) {
	appID, _ := cfg["appid"].(string)
	token, _ := cfg["access_token"].(string)
	cluster, _ := cfg["cluster"].(string)
	rid, _ := cfg["resource_id"].(string)
	wsURL, _ := cfg["ws_url"].(string)

	if rid == "" {
		rid = defaultResourceID
	}
	if wsURL == "" {
		wsURL = defaultASRURL
	}
	// 快捷API接入只需要 token（API Key）和 cluster；传统模式还需要 appid。
	if token == "" || cluster == "" {
		return nil, fmt.Errorf("doubao asr: access_token and cluster required")
	}

	// 不走代理直连火山引擎 ASR，避免本地 http_proxy 拦截 WebSocket 连接。
	dialer := &websocket.Dialer{
		NetDialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		HandshakeTimeout: 10 * time.Second,
	}

	return &provider{appID: appID, token: token, cluster: cluster, resourceID: rid, wsURL: wsURL, dialer: dialer}, nil
}

func (p *provider) Name() string { return "doubao" }

// AudioFormat 声明 doubao ASR 期望收到原始 PCM 字节（网关透传，无需 Opus 解码）。
func (p *provider) AudioFormat() asr.AudioFormat {
	return asr.AudioFormat{Codec: "pcm", SampleRate: 16000, Channels: 1}
}

// Transcribe sends PCM frames to Doubao streaming ASR and returns the final text.
func (p *provider) Transcribe(ctx context.Context, frames [][]byte) (string, error) {
	var final string
	err := p.transcribeInternal(ctx, asr.MergeFrames(frames), func(text string, isDef bool) {
		if isDef {
			final = text
		}
	})
	return final, err
}

// TranscribeStream sends PCM frames to Doubao streaming ASR and calls callback for each
// incremental result. callback is invoked only when recognized text changes, and
// final=true when recognition is complete.
func (p *provider) TranscribeStream(ctx context.Context, frames [][]byte, callback asr.ResultCallback) error {
	return p.transcribeInternal(ctx, asr.MergeFrames(frames), callback)
}

// connect 建立到豆包 ASR 服务的 WebSocket 连接并完成握手。
// 返回已握手的 conn，调用方负责关闭。
// ctx 取消时会异步关闭连接，使阻塞的 ReadMessage 立即返回。
func (p *provider) connect(ctx context.Context) (*websocket.Conn, error) {
	var headers http.Header
	if p.appID == "" {
		headers = http.Header{
			"Authorization":     {"Bearer " + p.token},
			"X-Api-Resource-Id": {p.resourceID},
			"X-Api-Connect-Id":  {uuid.New().String()},
		}
	} else {
		headers = http.Header{
			"X-Api-App-Key":     {p.appID},
			"X-Api-Access-Key":  {p.token},
			"X-Api-Resource-Id": {p.resourceID},
			"X-Api-Connect-Id":  {uuid.New().String()},
		}
	}

	conn, resp, err := p.dialer.DialContext(ctx, p.wsURL, headers)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			return nil, fmt.Errorf("doubao asr: dial: %w (HTTP %d: %s)", err, resp.StatusCode, bytes.TrimSpace(body))
		}
		return nil, fmt.Errorf("doubao asr: dial: %w", err)
	}

	// ctx 取消时关闭连接，使 ReadMessage 立即解除阻塞。
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// 发送初始化帧（协议握手）
	// Model 2.0（seedasr）：鉴权纯靠 HTTP Header，body 里不需要 app 节。
	initReq := map[string]any{
		"user": map[string]any{"uid": "picoclaw"},
		"request": map[string]any{
			"model_name":      "bigmodel",
			"show_utterances": true,
			"result_type":     "stream",
			"end_window_size": 200,
		},
		"audio": map[string]any{
			"format":      "pcm",
			"codec":       "pcm",
			"rate":        16000,
			"bits":        16,
			"channel":     1,
			"sample_rate": 16000,
		},
	}
	frame, err := buildJSONFrame(msgTypeFullClientRequest, flagNormal, initReq)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("doubao asr: build init frame: %w", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		conn.Close()
		return nil, fmt.Errorf("doubao asr: send init: %w", err)
	}
	_, initResp, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("doubao asr: read init response: %w", err)
	}
	if err := checkErrorResponse(initResp); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// transcribeInternal is the core implementation shared by Transcribe and TranscribeStream.
// pcmBytes: 16kHz 16-bit mono PCM, little-endian, raw bytes（由 MergeFrames 合并后传入）。
func (p *provider) transcribeInternal(ctx context.Context, pcmBytes []byte, callback asr.ResultCallback) error {
	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send PCM in 100ms chunks（3200 bytes = 1600 samples × 2 bytes），last chunk has flagLastFrame
	const chunkBytes = 3200
	for i := 0; i < len(pcmBytes); {
		end := i + chunkBytes
		if end > len(pcmBytes) {
			end = len(pcmBytes)
		}
		isLast := end >= len(pcmBytes)
		flags := byte(flagNormal)
		if isLast {
			flags = flagLastFrame
		}
		audioFrame, err := buildAudioFrame(flags, pcmBytes[i:end])
		if err != nil {
			return fmt.Errorf("doubao asr: build audio frame: %w", err)
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, audioFrame); err != nil {
			return fmt.Errorf("doubao asr: send audio: %w", err)
		}
		i = end

		// Check for context cancellation mid-stream
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Read responses until we get a definite utterance or connection closes.
	var lastText string
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			break // server closed normally
		}
		text, definite, done := parseASRResult(msg)
		// 只在文本有变化时回调，消除重复片段
		if text != "" && text != lastText {
			lastText = text
			label := "partial"
			if definite {
				label = "final"
			}
			log.Printf("doubao asr: %s text=%q", label, text)
			if callback != nil {
				callback(text, definite)
			}
		}
		if done {
			break
		}
	}

	if lastText == "" {
		log.Printf("doubao asr: no final text (silent or unrecognized)")
	}
	return nil
}

// asrResult carries the final ASR recognition result or an error.
type asrResult struct {
	text string
	err  error
}

// streamingSession is an active live doubao ASR session.
type streamingSession struct {
	conn      *websocket.Conn
	sendMu    sync.Mutex
	resultCh  chan asrResult // exactly one value written by the read goroutine
	closedCh  chan struct{}
	closeOnce sync.Once
}

// OpenSession establishes a new real-time ASR session.
// The passed context controls the session lifetime: cancellation closes the connection.
func (p *provider) OpenSession(ctx context.Context, callback asr.ResultCallback) (asr.StreamingSession, error) {
	conn, err := p.connect(ctx)
	if err != nil {
		return nil, err
	}

	sess := &streamingSession{
		conn:     conn,
		resultCh: make(chan asrResult, 1),
		closedCh: make(chan struct{}),
	}

	// 读取协程：持续接收 ASR 结果，将最终结果写入 resultCh
	go func() {
		var lastText string
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				sess.resultCh <- asrResult{err: err}
				return
			}
			text, definite, done := parseASRResult(msg)
			if text != "" && text != lastText {
				lastText = text
				label := "partial"
				if definite {
					label = "final"
				}
				log.Printf("doubao asr: %s text=%q", label, text)
				if callback != nil {
					callback(text, definite)
				}
			}
			if done {
				if lastText == "" {
					log.Printf("doubao asr: no final text (silent or unrecognized)")
				}
				sess.resultCh <- asrResult{text: lastText}
				return
			}
		}
	}()

	return sess, nil
}

// SendAudio pushes a raw PCM frame to the live ASR session.
// frame: 16kHz 16-bit mono PCM bytes（与 AudioFormat 声明一致）。
func (ss *streamingSession) SendAudio(frame []byte, isLast bool) error {
	select {
	case <-ss.closedCh:
		return asr.ErrSessionClosed
	default:
	}
	flags := byte(flagNormal)
	if isLast {
		flags = flagLastFrame
	}
	audioFrame, err := buildAudioFrame(flags, frame)
	if err != nil {
		return err
	}
	ss.sendMu.Lock()
	defer ss.sendMu.Unlock()
	return ss.conn.WriteMessage(websocket.BinaryMessage, audioFrame)
}

// Wait blocks until the final ASR result is available, ctx is cancelled,
// or the session is closed.
func (ss *streamingSession) Wait(ctx context.Context) (string, error) {
	select {
	case r := <-ss.resultCh:
		return r.text, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-ss.closedCh:
		return "", asr.ErrSessionClosed
	}
}

// Close aborts the session and releases all resources.
func (ss *streamingSession) Close() {
	ss.closeOnce.Do(func() {
		close(ss.closedCh)
		ss.conn.Close()
	})
}

// buildJSONFrame wraps a JSON payload in the doubao binary frame format.
func buildJSONFrame(msgType, flags byte, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	compressed, err := gzipBytes(data)
	if err != nil {
		return nil, err
	}
	return buildFrame(msgType, flags, serialJSON, compressGZP, compressed), nil
}

// buildAudioFrame wraps raw PCM bytes in the doubao binary frame format.
func buildAudioFrame(flags byte, pcmBytes []byte) ([]byte, error) {
	compressed, err := gzipBytes(pcmBytes)
	if err != nil {
		return nil, err
	}
	return buildFrame(msgTypeAudioOnly, flags, serialJSON, compressGZP, compressed), nil
}

func buildFrame(msgType, flags, serial, compress byte, payload []byte) []byte {
	hdr := [4]byte{
		(0x01 << 4) | 0x01, // version=1, header_size=1
		(msgType << 4) | flags,
		(serial << 4) | compress,
		0x00,
	}
	frame := make([]byte, 0, 8+len(payload))
	frame = append(frame, hdr[:]...)
	frame = binary.BigEndian.AppendUint32(frame, uint32(len(payload)))
	frame = append(frame, payload...)
	return frame
}

func checkErrorResponse(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("doubao asr: response too short (%d bytes)", len(data))
	}
	if (data[1] >> 4) == msgTypeServerError {
		if len(data) >= 8 {
			code := binary.BigEndian.Uint32(data[4:8])
			return fmt.Errorf("doubao asr: server error code=%d", code)
		}
		return fmt.Errorf("doubao asr: server error")
	}
	return nil
}

type asrPayload struct {
	Code   int `json:"code"`
	Result struct {
		Utterances []struct {
			Text     string `json:"text"`
			Definite bool   `json:"definite"`
		} `json:"utterances"`
	} `json:"result"`
}

// parseASRResult parses a server response frame, handling optional gzip compression.
// Returns (text, definite, done).
//   - definite=false → 中间结果，text 可能非空（继续等待后续帧）
//   - definite=true  → 最终结果，text 非空，done=true
func parseASRResult(data []byte) (text string, definite bool, done bool) {
	if len(data) < 12 {
		return "", false, false
	}
	if (data[1] >> 4) == msgTypeServerError {
		return "", false, true
	}
	// byte 2 lower nibble = compression type: 0x01 = gzip
	payload := data[12:]
	if data[2]&0x0F == compressGZP {
		uncompressed, err := gunzipBytes(payload)
		if err != nil {
			log.Printf("doubao asr: decompress response: %v", err)
			return "", false, false
		}
		payload = uncompressed
	}
	var p asrPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("doubao asr: parse response JSON: %v", err)
		return "", false, false
	}
	if p.Code == 1013 { // no speech detected
		return "", false, true
	}
	if p.Code != 0 && p.Code != 1000 {
		log.Printf("doubao asr: server code=%d", p.Code)
	}
	for _, u := range p.Result.Utterances {
		if u.Text != "" {
			if u.Definite {
				return u.Text, true, true
			}
			// 返回中间结果，不打 log（由调用方在文本变化时记录）
			return u.Text, false, false
		}
	}
	return "", false, false
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
