// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package doubao

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"
)

// ---- buildFrame ----

func TestBuildFrame_HeaderLayout(t *testing.T) {
	frame := buildFrame(msgTypeFullClientRequest, flagNormal, serialJSON, compressGZP, []byte("payload"))

	if len(frame) < 8 {
		t.Fatalf("frame too short: %d bytes", len(frame))
	}
	// byte[0]: (version=0x01 << 4) | header_size=0x01 = 0x11
	if frame[0] != 0x11 {
		t.Errorf("byte[0] = 0x%02x, want 0x11", frame[0])
	}
	// byte[1]: (msgType << 4) | flags
	wantByte1 := byte((msgTypeFullClientRequest << 4) | flagNormal)
	if frame[1] != wantByte1 {
		t.Errorf("byte[1] = 0x%02x, want 0x%02x", frame[1], wantByte1)
	}
	// byte[2]: (serial << 4) | compress
	wantByte2 := byte((serialJSON << 4) | compressGZP)
	if frame[2] != wantByte2 {
		t.Errorf("byte[2] = 0x%02x, want 0x%02x", frame[2], wantByte2)
	}
	// byte[3]: reserved = 0x00
	if frame[3] != 0x00 {
		t.Errorf("byte[3] = 0x%02x, want 0x00", frame[3])
	}
}

func TestBuildFrame_PayloadLength(t *testing.T) {
	payload := []byte("hello world")
	frame := buildFrame(msgTypeAudioOnly, flagLastFrame, serialJSON, compressGZP, payload)

	// bytes[4:8] = big-endian uint32 of len(payload)
	gotLen := binary.BigEndian.Uint32(frame[4:8])
	if gotLen != uint32(len(payload)) {
		t.Errorf("payload length = %d, want %d", gotLen, len(payload))
	}
	if !bytes.Equal(frame[8:], payload) {
		t.Errorf("payload bytes mismatch")
	}
}

func TestBuildFrame_AudioFlags(t *testing.T) {
	frame := buildFrame(msgTypeAudioOnly, flagLastFrame, serialJSON, compressGZP, []byte{})
	// byte[1] should carry flagLastFrame in lower nibble
	if frame[1]&0x0F != flagLastFrame {
		t.Errorf("flags nibble = 0x%x, want 0x%x", frame[1]&0x0F, flagLastFrame)
	}
}

// ---- buildJSONFrame ----

func TestBuildJSONFrame_GzipPayload(t *testing.T) {
	payload := map[string]any{"key": "value"}
	frame, err := buildJSONFrame(msgTypeFullClientRequest, flagNormal, payload)
	if err != nil {
		t.Fatalf("buildJSONFrame: %v", err)
	}
	if len(frame) < 8 {
		t.Fatalf("frame too short: %d", len(frame))
	}
	// bytes[4:8] = payload length
	compressedLen := binary.BigEndian.Uint32(frame[4:8])
	if int(compressedLen) != len(frame)-8 {
		t.Errorf("compressed length field %d != actual %d", compressedLen, len(frame)-8)
	}
	// Verify gzip decompresses to valid JSON containing "key"
	r, err := gzip.NewReader(bytes.NewReader(frame[8:]))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if out["key"] != "value" {
		t.Errorf("got key=%v, want 'value'", out["key"])
	}
}

// ---- buildAudioFrame ----

func TestBuildAudioFrame_NormalFlag(t *testing.T) {
	pcm := make([]byte, 3200) // 100ms of silence
	frame, err := buildAudioFrame(flagNormal, pcm)
	if err != nil {
		t.Fatalf("buildAudioFrame: %v", err)
	}
	if (frame[1] >> 4) != msgTypeAudioOnly {
		t.Errorf("msgType = 0x%x, want 0x%x", frame[1]>>4, msgTypeAudioOnly)
	}
	if frame[1]&0x0F != flagNormal {
		t.Errorf("flags = 0x%x, want 0x%x", frame[1]&0x0F, flagNormal)
	}
}

func TestBuildAudioFrame_LastFlag(t *testing.T) {
	frame, err := buildAudioFrame(flagLastFrame, []byte{})
	if err != nil {
		t.Fatalf("buildAudioFrame: %v", err)
	}
	if frame[1]&0x0F != flagLastFrame {
		t.Errorf("flags = 0x%x, want 0x%x", frame[1]&0x0F, flagLastFrame)
	}
}

// ---- parseASRResult ----

func buildMockResponse(code int, text string, definite bool) []byte {
	payload, _ := json.Marshal(map[string]any{
		"code": code,
		"result": map[string]any{
			"utterances": []map[string]any{
				{"text": text, "definite": definite},
			},
		},
	})
	// 响应格式：4 字节 header + 8 字节跳过 + JSON
	var buf []byte
	buf = append(buf, 0x11, (0x0B<<4)|0x00, 0x00, 0x00) // header
	buf = append(buf, 0, 0, 0, 0, 0, 0, 0, 0)           // 8 bytes skipped
	buf = append(buf, payload...)
	return buf
}

func TestParseASRResult_DefiniteUtterance(t *testing.T) {
	data := buildMockResponse(1000, "你好世界", true)
	text, definite, done := parseASRResult(data)
	if text != "你好世界" {
		t.Errorf("text = %q, want '你好世界'", text)
	}
	if !definite {
		t.Error("definite = false, want true")
	}
	if !done {
		t.Error("done = false, want true")
	}
}

func TestParseASRResult_IndefiniteUtterance(t *testing.T) {
	data := buildMockResponse(1000, "你好", false)
	text, definite, done := parseASRResult(data)
	if text != "你好" {
		t.Errorf("text = %q, want '你好' (intermediate result)", text)
	}
	if definite {
		t.Error("definite = true, want false for non-definite")
	}
	if done {
		t.Error("done = true, want false for non-definite")
	}
}

func TestParseASRResult_NoSpeechCode(t *testing.T) {
	data := buildMockResponse(1013, "", false)
	text, _, done := parseASRResult(data)
	if text != "" || !done {
		t.Errorf("got text=%q done=%v, want empty/true for code 1013 (silent, session ends)", text, done)
	}
}

func TestParseASRResult_TooShort(t *testing.T) {
	_, _, done := parseASRResult([]byte{0x11, 0x00})
	if done {
		t.Error("expected done=false for short frame")
	}
}

func TestParseASRResult_ServerError(t *testing.T) {
	// msgType 0x0F in byte[1] upper nibble
	data := []byte{0x11, (msgTypeServerError << 4), 0x00, 0x00, 0, 0, 0, 42, 0, 0, 0, 0}
	_, _, done := parseASRResult(data)
	if !done {
		t.Error("expected done=true for server error frame")
	}
}

// ---- checkErrorResponse ----

func TestCheckErrorResponse_OK(t *testing.T) {
	frame := buildFrame(0x0B, flagNormal, serialJSON, compressGZP, []byte("{}"))
	if err := checkErrorResponse(frame); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckErrorResponse_ServerError(t *testing.T) {
	frame := []byte{0x11, (msgTypeServerError << 4) | 0x00, 0x00, 0x00, 0, 0, 0, 99, 0, 0, 0, 4}
	if err := checkErrorResponse(frame); err == nil {
		t.Error("expected error for server error frame")
	}
}

func TestCheckErrorResponse_TooShort(t *testing.T) {
	if err := checkErrorResponse([]byte{0x11}); err == nil {
		t.Error("expected error for too-short frame")
	}
}
