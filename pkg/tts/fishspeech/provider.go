// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package fishspeech 对接 Fish Speech v1.5.x HTTP TTS 服务。
// POST /v1/tts 输出 WAV（streaming=true），流式读取 PCM 帧后直接下发客户端。
// 无需 CGO，完全纯 Go。
//
// 部署：
//
//	docker compose -f docker/docker-compose.asr-tts.yml up -d fishspeech
//
// 配置（环境变量）：
//
//	PICOCLAW_VOICE_TTS_API_URL       服务地址，默认 http://127.0.0.1:8080
//	PICOCLAW_VOICE_TTS_API_KEY       Bearer Token，本地部署通常无需填写
//	PICOCLAW_VOICE_TTS_REFERENCE_ID  参考音色 ID（留空使用服务默认音色）
//	PICOCLAW_VOICE_TTS_SAMPLE_RATE   输出采样率，默认 44100（Fish Speech v1.5.x 默认值）
package fishspeech

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/tts"
)

func init() {
	tts.Register("fishspeech", func(cfg map[string]any) (tts.Provider, error) {
		return newProvider(cfg)
	})
}

const (
	defaultAPIBase    = "http://127.0.0.1:8080"
	defaultSampleRate = 44100
	// 每次回调约 46ms 的 PCM（@44100Hz mono s16le）
	pcmChunkBytes = 4096
	// 用于生成参考音频的固定短句（中性内容，音色稳定）
	refGenText = "你好，我是你的语音助手。"
)

// refAudio 缓存：首次合成时用 seed 生成一段参考 WAV，后续请求带上它以固定音色。
type refAudio struct {
	once  sync.Once
	wav   []byte // RIFF WAV 文件（streaming=false, format=wav）
	ready bool
}

type provider struct {
	apiBase     string
	apiKey      string
	referenceID string
	sampleRate  int
	seed        int // 0 = 随机音色；非 0 = 首次合成后锁定参考音频
	ref         refAudio
	client      *http.Client
}

func newProvider(cfg map[string]any) (*provider, error) {
	apiBase, _ := cfg["api_url"].(string)
	apiKey, _ := cfg["api_key"].(string)
	referenceID, _ := cfg["reference_id"].(string)
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	sampleRate := defaultSampleRate
	if sr, ok := cfg["sample_rate"].(int); ok && sr > 0 {
		sampleRate = sr
	}
	seed, _ := cfg["seed"].(int)
	return &provider{
		apiBase:     apiBase,
		apiKey:      apiKey,
		referenceID: referenceID,
		sampleRate:  sampleRate,
		seed:        seed,
		// 仅设置连接建立超时；response body 读取时间由 ctx 控制（streaming=true 时 body 无期限流式输出）
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			},
		},
	}, nil
}

func (p *provider) Name() string { return "fishspeech" }

// AudioFormat 声明输出格式为 PCM s16le，采样率由配置指定（默认 44100Hz mono）。
// 网关通过 tts_params 下发给客户端，客户端据此初始化播放设备，无需 Opus 解码。
func (p *provider) AudioFormat() tts.AudioFormat {
	return tts.AudioFormat{Codec: "pcm", SampleRate: p.sampleRate, Channels: 1}
}

// SynthesizeFrames 向 Fish Speech HTTP API 发请求，流式读取原始 PCM s16le 数据分块回调。
// 若配置了 seed，首次调用会先用固定短句生成参考音频并缓存，后续每次合成都带上该参考，
// 从而保证不同句子的音色一致。
func (p *provider) SynthesizeFrames(ctx context.Context, text, voice string, onFrame func([]byte)) error {
	refID := voice
	if refID == "" {
		refID = p.referenceID
	}

	// seed 非 0 且没有外部 reference_id 时，使用内部参考音频固定音色
	var refWAV []byte
	if p.seed != 0 && refID == "" {
		p.ref.once.Do(func() {
			wav, err := p.generateRefWAV()
			if err != nil {
				log.Printf("tts/fishspeech: generate ref audio: %v (voice may vary)", err)
				return
			}
			p.ref.wav = wav
			p.ref.ready = true
		})
		if p.ref.ready {
			refWAV = p.ref.wav
		}
	}

	payload := map[string]any{
		"text":         text,
		"streaming":    true,
		"chunk_length": 100,
	}
	if refID != "" {
		payload["reference_id"] = refID
	} else if len(refWAV) > 0 {
		// 将缓存的参考 WAV 编码为 base64，作为 in-context speaker 固定音色
		payload["references"] = []map[string]any{
			{
				"audio": base64.StdEncoding.EncodeToString(refWAV),
				"text":  refGenText,
			},
		}
	} else if p.seed != 0 {
		// 参考音频未就绪（生成失败）时退化为 seed 模式
		payload["seed"] = p.seed
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("tts/fishspeech: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/v1/tts", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("tts/fishspeech: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("tts/fishspeech: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("tts/fishspeech: http %d: %s", resp.StatusCode, b)
	}

	// Fish Speech streaming=true 直接返回裸 PCM s16le，无 RIFF 头
	buf := make([]byte, pcmChunkBytes)
	for {
		n, err := io.ReadFull(resp.Body, buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			onFrame(chunk)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("tts/fishspeech: read: %w", err)
		}
	}
	return nil
}

// generateRefWAV 用 refGenText + seed 生成一段参考 WAV，streaming=false 以获取完整文件。
func (p *provider) generateRefWAV() ([]byte, error) {
	payload := map[string]any{
		"text":      refGenText,
		"streaming": false,
		"format":    "wav",
		"seed":      p.seed,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, p.apiBase+"/v1/tts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, b)
	}
	return io.ReadAll(resp.Body)
}


