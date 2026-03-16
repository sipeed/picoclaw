// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// picoclaw-voice: xiaozhi WebSocket gateway — Opus↔ASR→LLM→TTS→Opus pipeline.
// 通过 xiaozhi-esp32 协议与客户端通信，内部调用 ASR/LLM/TTS 三方服务。
//
// 配置通过环境变量注入（见 Config 结构体）:
// PICOCLAW_VOICE_LISTEN      监听地址，默认 :8765
// PICOCLAW_VOICE_OWNER_ID    固定 owner_id，多设备共享同一份记忆，默认空（per-device 独立记忆）
// PICOCLAW_CONFIG            picoclaw config.json 路径，默认 ~/.picoclaw/config.json
// PICOCLAW_HOME              picoclaw home 目录，默认 ~/.picoclaw
//
// 通用供应商选择：
// PICOCLAW_VOICE_ASR_PROVIDER  ASR 供应商，可选 doubao | funasr，默认 doubao
// PICOCLAW_VOICE_TTS_PROVIDER  TTS 供应商，可选 doubao | fishspeech，默认 doubao
//
// FunASR（本地免费 ASR，推荐中文）：
// PICOCLAW_VOICE_ASR_WS_URL    FunASR WebSocket 地址，默认 wss://127.0.0.1:10095（内置自签名证书，跳过校验）
// PICOCLAW_VOICE_ASR_MODE      识别模式：2pass（默认）| online | offline
//
// Fish Speech（本地免费 TTS，推荐中文，需 GPU）：
// PICOCLAW_VOICE_TTS_API_URL      Fish Speech HTTP 地址，默认 http://127.0.0.1:8080
// PICOCLAW_VOICE_TTS_REFERENCE_ID 参考音色 ID，留空使用服务默认
// PICOCLAW_VOICE_TTS_SAMPLE_RATE  输出采样率，默认 44100（Fish Speech v2 默认）
//
// 共享火山引擎凭证（doubao 供应商 ASR/TTS 使用同一账号时，只需填这两项）：
// PICOCLAW_VOICE_APPID  火山引擎 AppID（ASR/TTS 共用兜底）
// PICOCLAW_VOICE_TOKEN  火山引擎 Access Token（ASR/TTS 共用兜底）
//
// doubao ASR 单独覆盖（可选）：
// PICOCLAW_VOICE_ASR_APPID       覆盖 PICOCLAW_VOICE_APPID
// PICOCLAW_VOICE_ASR_TOKEN       覆盖 PICOCLAW_VOICE_TOKEN
// PICOCLAW_VOICE_ASR_CLUSTER     默认 bigmodel_transcribe
// PICOCLAW_VOICE_ASR_RESOURCE_ID 默认 volc.bigasr.sauc.duration
//
// doubao TTS 单独覆盖（可选）：
// PICOCLAW_VOICE_TTS_APPID    覆盖 PICOCLAW_VOICE_APPID
// PICOCLAW_VOICE_TTS_TOKEN    覆盖 PICOCLAW_VOICE_TOKEN
// PICOCLAW_VOICE_TTS_CLUSTER  默认 volcano_tts
// PICOCLAW_VOICE_TTS_VOICE    音色
package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/asr"
	_ "github.com/sipeed/picoclaw/pkg/asr/doubao"
	_ "github.com/sipeed/picoclaw/pkg/asr/funasr"
	"github.com/sipeed/picoclaw/pkg/bus"
	picoconfig "github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tts"
	_ "github.com/sipeed/picoclaw/pkg/tts/doubao"
	_ "github.com/sipeed/picoclaw/pkg/tts/fishspeech"
)

type config struct {
	Listen  string `env:"PICOCLAW_VOICE_LISTEN"    envDefault:":8765"`
	OwnerID string `env:"PICOCLAW_VOICE_OWNER_ID"`

	// 共享凭证：ASR 和 TTS 可复用同一套火山引擎账号
	AppID string `env:"PICOCLAW_VOICE_APPID"`
	Token string `env:"PICOCLAW_VOICE_TOKEN"`

	// ASR 供应商选择 (doubao | funasr)
	ASRProvider   string `env:"PICOCLAW_VOICE_ASR_PROVIDER"    envDefault:"doubao"`
	ASRAppID      string `env:"PICOCLAW_VOICE_ASR_APPID"` // doubao：覆盖 PICOCLAW_VOICE_APPID
	ASRToken      string `env:"PICOCLAW_VOICE_ASR_TOKEN"` // doubao：覆盖 PICOCLAW_VOICE_TOKEN
	ASRCluster    string `env:"PICOCLAW_VOICE_ASR_CLUSTER"     envDefault:"bigmodel_transcribe"`
	ASRResourceID string `env:"PICOCLAW_VOICE_ASR_RESOURCE_ID"` // doubao：资源 ID
	ASRWsURL      string `env:"PICOCLAW_VOICE_ASR_WS_URL"`      // funasr：WebSocket 地址
	ASRMode       string `env:"PICOCLAW_VOICE_ASR_MODE"`        // funasr：2pass | online | offline

	// TTS 供应商选择 (doubao | fishspeech)
	TTSProvider    string `env:"PICOCLAW_VOICE_TTS_PROVIDER"      envDefault:"doubao"`
	TTSAPIURL      string `env:"PICOCLAW_VOICE_TTS_API_URL"`                     // fishspeech：HTTP 服务地址
	TTSAPIKey      string `env:"PICOCLAW_VOICE_TTS_API_KEY"`                     // fishspeech：Bearer Token
	TTSReferenceID string `env:"PICOCLAW_VOICE_TTS_REFERENCE_ID"`                // fishspeech：参考音色 ID
	TTSSampleRate  int    `env:"PICOCLAW_VOICE_TTS_SAMPLE_RATE"  envDefault:"0"` // fishspeech：输出采样率，0=provider 默认
	TTSSeed        int    `env:"PICOCLAW_VOICE_TTS_SEED"         envDefault:"0"` // fishspeech：固定随机种子，0=随机音色
	TTSAppID       string `env:"PICOCLAW_VOICE_TTS_APPID"`                       // doubao：覆盖 PICOCLAW_VOICE_APPID
	TTSToken       string `env:"PICOCLAW_VOICE_TTS_TOKEN"`                       // doubao：覆盖 PICOCLAW_VOICE_TOKEN
	TTSCluster     string `env:"PICOCLAW_VOICE_TTS_CLUSTER"       envDefault:"volcano_tts"`
	TTSVoice       string `env:"PICOCLAW_VOICE_TTS_VOICE"`
}

func main() {
	// 优先从可执行文件所在目录加载 .env，回退到当前工作目录。
	// 已有同名环境变量时跳过（不覆盖），即 Docker environment: 覆盖 > .env。
	if exe, err := os.Executable(); err == nil {
		loadDotEnv(filepath.Join(filepath.Dir(exe), ".env"))
	} else {
		loadDotEnv(".env")
	}

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("picoclaw-voice: parse config: %v", err)
	}

	// 加载 picoclaw config 并初始化 AgentLoop (LLM)
	pcCfgPath := picoclawConfigPath()
	pcCfg, err := picoconfig.LoadConfig(pcCfgPath)
	if err != nil {
		log.Fatalf("picoclaw-voice: load picoclaw config from %s: %v", pcCfgPath, err)
	}
	llmProvider, modelID, err := providers.CreateProvider(pcCfg)
	if err != nil {
		log.Fatalf("picoclaw-voice: create LLM provider: %v", err)
	}
	if modelID != "" {
		pcCfg.Agents.Defaults.ModelName = modelID
	}
	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(pcCfg, msgBus, llmProvider)

	log.Printf("picoclaw-voice: ASR config: provider=%s ws_url=%s mode=%s", cfg.ASRProvider, cfg.ASRWsURL, cfg.ASRMode)
	log.Printf("picoclaw-voice: TTS config: provider=%s api_url=%s seed=%d", cfg.TTSProvider, cfg.TTSAPIURL, cfg.TTSSeed)

	asrProvider, err := asr.New(cfg.ASRProvider, map[string]any{
		// doubao 字段
		"appid":        orStr(cfg.ASRAppID, cfg.AppID),
		"access_token": orStr(cfg.ASRToken, cfg.Token),
		"cluster":      cfg.ASRCluster,
		"resource_id":  orStr(cfg.ASRResourceID, "volc.bigasr.sauc.duration"),
		// funasr 字段
		"ws_url": cfg.ASRWsURL,
		"mode":   cfg.ASRMode,
	})
	if err != nil {
		log.Fatalf("picoclaw-voice: init ASR: %v", err)
	}

	ttsProvider, err := tts.New(cfg.TTSProvider, map[string]any{
		// fishspeech 字段
		"api_url":      cfg.TTSAPIURL,
		"api_key":      cfg.TTSAPIKey,
		"reference_id": cfg.TTSReferenceID,
		"sample_rate":  cfg.TTSSampleRate,
		"seed":         cfg.TTSSeed,
		// doubao 字段
		"appid":        orStr(cfg.TTSAppID, cfg.AppID),
		"access_token": orStr(cfg.TTSToken, cfg.Token),
		"cluster":      cfg.TTSCluster,
		"voice":        cfg.TTSVoice,
	})
	if err != nil {
		log.Fatalf("picoclaw-voice: init TTS: %v", err)
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	reg := newDeviceRegistry()

	http.HandleFunc("/xiaozhi/v1/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("picoclaw-voice: ws upgrade: %v", err)
			return
		}
		s := newSession(conn, asrProvider, ttsProvider, agentLoop, cfg.OwnerID, reg)
		s.run()
	})

	log.Printf("picoclaw-voice: listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, nil); err != nil {
		log.Fatalf("picoclaw-voice: %v", err)
	}
}

// picoclawConfigPath 返回 picoclaw config.json 路径。
// 优先级: $PICOCLAW_CONFIG > $PICOCLAW_HOME/config.json > ~/.picoclaw/config.json
func picoclawConfigPath() string {
	if p := os.Getenv("PICOCLAW_CONFIG"); p != "" {
		return p
	}
	if h := os.Getenv("PICOCLAW_HOME"); h != "" {
		return filepath.Join(h, "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

// orStr 返回 a（若非空），否则返回 b。
func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// loadDotEnv 从指定路径加载 .env 文件。
// 格式：KEY=VALUE，忽略空行和 # 注释行。
// .env 的值总是生效（覆盖 stale shell export），因此本地开发无需手动 unset 旧变量。
// Docker 容器内不存在此文件（Dockerfile 未 COPY），故 Docker 环境不受影响。
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // 文件不存在时静默跳过（Docker 容器内正常触发此分支）
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}
		os.Setenv(key, val) // 总是覆盖，防止 stale shell export 干扰本地调试
	}
}
