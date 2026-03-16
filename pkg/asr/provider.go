// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package asr

import (
	"context"
	"errors"
	"fmt"
)

// AudioFormat describes the audio wire format a provider expects from the device.
// The gateway uses this to populate helloReply.audio_params; devices must send accordingly.
type AudioFormat struct {
	Codec      string // "pcm" | "opus"
	SampleRate int    // Hz, e.g. 16000
	Channels   int    // 1 = mono
}

// MergeFrames concatenates audio frames into a single byte slice.
// Useful for batch-mode providers that process all audio at once.
func MergeFrames(frames [][]byte) []byte {
	total := 0
	for _, f := range frames {
		total += len(f)
	}
	merged := make([]byte, 0, total)
	for _, f := range frames {
		merged = append(merged, f...)
	}
	return merged
}

// Provider is the ASR interface.
type Provider interface {
	Name() string
	// AudioFormat returns the audio format this provider expects from the device.
	// The gateway advertises this via helloReply so the device sends the correct format.
	AudioFormat() AudioFormat
	// Transcribe converts audio frames to text.
	// frames: raw audio bytes in the format declared by AudioFormat().
	Transcribe(ctx context.Context, frames [][]byte) (string, error)
}

// ResultCallback receives incremental ASR results.
// final=true 表示识别完成（definite），后续不再有回调。
type ResultCallback func(text string, final bool)

// StreamingProvider extends Provider with batch streaming ASR support.
// The callback is invoked each time the recognized text changes, and once more
// with final=true when recognition is complete.
type StreamingProvider interface {
	Provider
	TranscribeStream(ctx context.Context, frames [][]byte, callback ResultCallback) error
}

// ErrSessionClosed is returned by StreamingSession.Wait when the session
// was closed before a final result was available.
var ErrSessionClosed = errors.New("asr: session closed")

// StreamingSession is an active real-time ASR session.
// Open with RealtimeProvider.OpenSession; feed audio with SendAudio;
// signal end-of-audio by passing isLast=true; then call Wait for the result.
type StreamingSession interface {
	// SendAudio pushes a raw audio frame. Set isLast=true on the final frame.
	// frame: raw audio bytes in the format declared by the provider's AudioFormat().
	SendAudio(frame []byte, isLast bool) error
	// Wait blocks until the final recognition result is available.
	Wait(ctx context.Context) (string, error)
	// Close aborts the session and releases all resources.
	Close()
}

// RealtimeProvider extends Provider with live frame-by-frame ASR.
// Audio is fed incrementally as it arrives, reducing latency.
type RealtimeProvider interface {
	Provider
	// OpenSession starts a new live ASR session.
	// callback receives each incremental result (final=true on completion).
	OpenSession(ctx context.Context, callback ResultCallback) (StreamingSession, error)
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
		return nil, fmt.Errorf("asr: provider %q not registered", name)
	}
	return f(cfg)
}
