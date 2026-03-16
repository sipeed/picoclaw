// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tts

import (
	"context"
	"fmt"
)

// AudioFormat 描述 TTS provider 的输出音频格式，由 provider 自身声明。
// 网关通过 helloReply.tts_params 下发给客户端，客户端据此初始化解码器。
type AudioFormat struct {
	Codec      string // 编码格式，如 "opus"、"pcm"
	SampleRate int    // 采样率，如 16000
	Channels   int    // 声道数，1 = 单声道
}

// Provider is the TTS interface.
// AudioFormat 由 provider 自身声明；SynthesizeFrames 输出与之对应格式的帧。
type Provider interface {
	Name() string
	// AudioFormat 返回本 provider 的输出格式，用于向客户端协商解码参数。
	AudioFormat() AudioFormat
	// SynthesizeFrames 将文本合成为音频，每帧通过 onFrame 回调返回。
	// 帧格式由 AudioFormat() 声明；voice 为空时使用 provider 默认音色。
	SynthesizeFrames(ctx context.Context, text, voice string, onFrame func([]byte)) error
}

// Factory creates a Provider from a config map.
type Factory func(cfg map[string]any) (Provider, error)

var factories = map[string]Factory{}

// Register adds a provider factory. Called from provider init() functions.
func Register(name string, f Factory) {
	factories[name] = f
}

// New creates a Provider by name.
func New(name string, cfg map[string]any) (Provider, error) {
	f, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("tts: provider %q not registered", name)
	}
	return f(cfg)
}
