package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AliyunTTSConfig 阿里云 TTS 配置
type AliyunTTSConfig struct {
	APIKey     string `json:"api_key"    env:"PICOCLAW_VOICE_TTS_API_KEY"`     // DashScope API Key
	Voice      string `json:"voice"      env:"PICOCLAW_VOICE_TTS_VOICE"`        // 音色选择
	Speed      int    `json:"speed"      env:"PICOCLAW_VOICE_TTS_SPEED"`        // 语速 -500~500
	Volume     int    `json:"volume"     env:"PICOCLAW_VOICE_TTS_VOLUME"`       // 音量 -500~500
	Pitch      int    `json:"pitch"      env:"PICOCLAW_VOICE_TTS_PITCH"`        // 音调 -500~500
	Format     string `json:"format"     env:"PICOCLAW_VOICE_TTS_FORMAT"`       // 音频格式 mp3/wav
	SampleRate int    `json:"sample_rate" env:"PICOCLAW_VOICE_TTS_SAMPLE_RATE"`  // 采样率
}

// AliyunTTSSynthesizer 阿里云语音合成器
type AliyunTTSSynthesizer struct {
	config     AliyunTTSConfig
	httpClient *http.Client
	apiBase    string
}

// TTSResponse 阿里云 TTS 响应
type TTSResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RequestID string `json:"request_id"`
	Data    struct {
		Audio string `json:"audio"` // base64 编码的音频
		Format string `json:"format"`
		SampleRate string `json:"sample_rate"`
		Duration float64 `json:"duration"`
	} `json:"data"`
}

// AvailableVoices 可用的音色列表
var AvailableVoices = []struct {
	ID          string
	Name        string
	Description string
}{
	{"xiaoyun", "xiaoyun", "云小希 - 阳光女声"},
	{"xiaogang", "xiaogang", "云小刚 - 活力男声"},
	{"xiaomei", "xiaomei", "云小美 - 温柔女声"},
	{"xiaobai", "xiaobai", "云小白 - 清亮女声"},
	{"xiaoyong", "xiaoyong", "云小勇 - 自信男声"},
	{"ruoxi", "ruoxi", "若兮 - 温柔女声"},
	{"ruobing", "ruobing", "若冰 - 沉稳女声"},
	{"aixia", "艾夏", "艾夏 - 活泼女声"},
	{"aiyu", "艾雨", "艾雨 - 知性女声"},
	{"aibin", "艾彬", "艾彬 - 成熟男声"},
	{"aijia", "艾佳", "艾佳 - 甜美女声"},
	{"aichuan", "艾川", "艾川 - 浑厚男声"},
	{"zhili", "知丽", "知丽 - 清晰女声"},
	{"yujie", "云剑", "云剑 - 慷慨男声"},
}

// NewAliyunTTSSynthesizer 创建阿里云 TTS 合成器
func NewAliyunTTSSynthesizer(config AliyunTTSConfig) *AliyunTTSSynthesizer {
	// 设置默认值
	if config.Format == "" {
		config.Format = "mp3"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 24000
	}
	if config.Voice == "" {
		config.Voice = "xiaoyun" // 默认音色
	}
	if config.Speed == 0 {
		config.Speed = 0 // 正常语速
	}
	if config.Volume == 0 {
		config.Volume = 0 // 正常音量
	}
	if config.Pitch == 0 {
		config.Pitch = 0 // 正常音调
	}

	logger.DebugCF("voice", "Creating Aliyun TTS synthesizer", map[string]any{
		"voice":      config.Voice,
		"speed":      config.Speed,
		"volume":     config.Volume,
		"pitch":      config.Pitch,
		"format":     config.Format,
		"sample_rate": config.SampleRate,
		"has_api_key": config.APIKey != "",
	})

	return &AliyunTTSSynthesizer{
		config:     config,
		apiBase:    "https://dashscope.aliyuncs.com/api/v1/services/audio/t2a",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Synthesize 将文字转换为语音
func (t *AliyunTTSSynthesizer) Synthesize(ctx context.Context, text string) ([]byte, error) {
	if t.config.APIKey == "" {
		return nil, fmt.Errorf("aliyun TTS API key is not configured")
	}

	logger.InfoCF("voice", "Starting TTS synthesis", map[string]any{
		"text_length": len(text),
		"voice":      t.config.Voice,
	})

	// 构建请求
	requestBody := map[string]interface{}{
		"model": "sambert-emotion-tts-v1",
		"input": map[string]string{
			"text": text,
		},
		"parameters": map[string]interface{}{
			"voice": t.config.Voice,
			"format": t.config.Format,
			"sample_rate": t.config.SampleRate,
			"speech_rate": t.config.Speed,
			"volume": t.config.Volume,
			"pitch": t.config.Pitch,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.ErrorCF("voice", "Failed to marshal TTS request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiBase, bytes.NewReader(jsonBody))
	if err != nil {
		logger.ErrorCF("voice", "Failed to create TTS request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.config.APIKey)
	req.Header.Set("X-DashScope-Api-Version", "2024-09-01")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("voice", "Failed to send TTS request", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("voice", "Failed to read TTS response", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("voice", "TTS API error", map[string]any{
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result TTSResponse
	if err := json.Unmarshal(body, &result); err != nil {
		logger.ErrorCF("voice", "Failed to unmarshal TTS response", map[string]any{"error": err, "response": string(body)})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Code != "200" && result.Code != "" {
		logger.ErrorCF("voice", "TTS synthesis failed", map[string]any{
			"code":    result.Code,
			"message": result.Message,
		})
		return nil, fmt.Errorf("TTS error: %s - %s", result.Code, result.Message)
	}

	// 解码 base64 音频
	audioData, err := base64.StdEncoding.DecodeString(result.Data.Audio)
	if err != nil {
		logger.ErrorCF("voice", "Failed to decode audio data", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to decode audio: %w", err)
	}

	logger.InfoCF("voice", "TTS synthesis completed", map[string]any{
		"audio_size":  len(audioData),
		"format":     result.Data.Format,
		"duration":   result.Data.Duration,
	})

	return audioData, nil
}

// SynthesizeToFile 将文字转换为语音并保存到文件
func (t *AliyunTTSSynthesizer) SynthesizeToFile(ctx context.Context, text string, outputPath string) (string, error) {
	audioData, err := t.Synthesize(ctx, text)
	if err != nil {
		return "", err
	}

	// 确保目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.ErrorCF("voice", "Failed to create output directory", map[string]any{"error": err, "dir": dir})
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 保存文件
	if err := os.WriteFile(outputPath, audioData, 0644); err != nil {
		logger.ErrorCF("voice", "Failed to write audio file", map[string]any{"error": err, "path": outputPath})
		return "", fmt.Errorf("failed to write audio file: %w", err)
	}

	logger.InfoCF("voice", "Audio file saved", map[string]any{
		"path": outputPath,
		"size": len(audioData),
	})

	return outputPath, nil
}

// IsAvailable 检查 TTS 是否可用
func (t *AliyunTTSSynthesizer) IsAvailable() bool {
	available := t.config.APIKey != ""
	logger.DebugCF("voice", "Checking TTS availability", map[string]any{"available": available})
	return available
}

// GetVoice 获取当前配置的音色
func (t *AliyunTTSSynthesizer) GetVoice() string {
	return t.config.Voice
}

// SetVoice 设置音色
func (t *AliyunTTSSynthesizer) SetVoice(voice string) {
	t.config.Voice = voice
}

// GetConfig 获取配置
func (t *AliyunTTSSynthesizer) GetConfig() AliyunTTSConfig {
	return t.config
}

// GetVoiceName 获取音色名称
func GetVoiceName(voiceID string) string {
	for _, v := range AvailableVoices {
		if v.ID == voiceID {
			return v.Name
		}
	}
	return voiceID
}

// GetAllVoices 获取所有可用音色
func GetAllVoices() []struct {
	ID          string
	Name        string
	Description string
} {
	return AvailableVoices
}

// CleanText 清理待合成文本
func CleanText(text string) string {
	// 移除多余空白
	text = strings.TrimSpace(text)
	// 移除特殊字符（保留中文、英文、数字、常见标点）
	var result []rune
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' {
			result = append(result, ' ')
		} else if r >= 0x4E00 && r <= 0x9FFF || // 中文
			r >= 0x0030 && r <= 0x0039 || // 数字
			r >= 0x0041 && r <= 0x005A || // 大写英文
			r >= 0x0061 && r <= 0x007A || // 小写英文
			r == ' ' || r == '.' || r == ',' || r == '?' || r == '!' ||
			r == ':' || r == ';' || r == '"' || r == '\'' ||
			r == '(' || r == ')' || r == '[' || r == ']' ||
			r == '-' || r == '_' || r == '/' || r == '\\' {
			result = append(result, r)
		}
	}
	return strings.TrimSpace(string(result))
}

// TruncateText 截断文本（阿里云 TTS 有字符限制）
func TruncateText(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "。"
}
