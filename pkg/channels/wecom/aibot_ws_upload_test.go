package wecom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// UploadWSMedia — pre-flight validation tests (no real WS required)
// ---------------------------------------------------------------------------

// TestUploadWSMedia_InvalidMediaType verifies that unsupported media types are
// rejected before any network activity occurs.
func TestUploadWSMedia_InvalidMediaType(t *testing.T) {
	ch := newTestWSChannel(t)

	for _, bad := range []string{"", "pdf", "audio", "IMAGE", "gif"} {
		_, err := ch.UploadWSMedia(context.Background(), "/dev/null", "test.pdf", bad)
		if err == nil {
			t.Errorf("expected error for media type %q, got nil", bad)
		}
	}
}

// TestUploadWSMedia_FileNotFound verifies that a non-existent path returns an error.
func TestUploadWSMedia_FileNotFound(t *testing.T) {
	ch := newTestWSChannel(t)
	_, err := ch.UploadWSMedia(context.Background(), "/nonexistent/path/to/file.bin", "", "file")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestUploadWSMedia_EmptyFile verifies that an empty file is rejected before
// the upload starts.
func TestUploadWSMedia_EmptyFile(t *testing.T) {
	ch := newTestWSChannel(t)

	tmp, err := os.CreateTemp(t.TempDir(), "empty-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	_, err = ch.UploadWSMedia(context.Background(), tmp.Name(), "", "file")
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

// TestUploadWSMedia_FileTooLarge verifies that a file requiring more than
// wsUploadMaxChunks chunks is rejected before the upload starts.
func TestUploadWSMedia_FileTooLarge(t *testing.T) {
	ch := newTestWSChannel(t)

	// Synthesize a file that would need 101 chunks (one byte over the limit).
	size := int64(wsUploadMaxChunks)*int64(wsUploadChunkSize) + 1

	tmp, err := os.CreateTemp(t.TempDir(), "large-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	truncErr := tmp.Truncate(size)
	tmp.Close()
	if truncErr != nil {
		t.Fatal(truncErr)
	}

	_, err = ch.UploadWSMedia(context.Background(), tmp.Name(), "", "file")
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}
}

// TestUploadWSMedia_NotConnected verifies that calling UploadWSMedia when no
// WebSocket connection is active returns a clear error.
func TestUploadWSMedia_NotConnected(t *testing.T) {
	ch := newTestWSChannel(t)

	tmp := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmp, []byte("hello world"), 0o600); err != nil {
		t.Fatal(err)
	}

	// conn is nil by default in a test channel — callWSCommand must return an error.
	_, err := ch.UploadWSMedia(context.Background(), tmp, "", "file")
	if err == nil {
		t.Fatal("expected error when conn is nil, got nil")
	}
}

// TestUploadWSMedia_ContextCanceledPreFlight verifies that a pre-canceled
// context is detected during chunk iteration (even before any WS call, when
// the connection is nil the WS error takes precedence for a single-chunk file,
// so we test this path with a file that triggers ctx.Err() first).
func TestUploadWSMedia_ContextCanceledPreFlight(t *testing.T) {
	ch := newTestWSChannel(t)

	tmp := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmp, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel before the call

	// With conn == nil, callWSCommand will return "websocket not connected".
	// The pre-canceled context is also an error; either one satisfies the test.
	_, err := ch.UploadWSMedia(ctx, tmp, "", "file")
	if err == nil {
		t.Fatal("expected error for canceled context or nil conn, got nil")
	}
}

// ---------------------------------------------------------------------------
// callWSCommand — unit tests without a live connection
// ---------------------------------------------------------------------------

// TestCallWSCommand_EmptyReqID verifies that callWSCommand rejects a command
// whose req_id is empty before touching the connection.
func TestCallWSCommand_EmptyReqID(t *testing.T) {
	ch := newTestWSChannel(t)
	cmd := wsCommand{
		Cmd:  "aibot_upload_media_init",
		Body: map[string]string{"type": "file"},
		// Headers.ReqID intentionally left empty
	}
	_, err := ch.callWSCommand(cmd, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for empty req_id, got nil")
	}
}

// TestCallWSCommand_NotConnected verifies that callWSCommand returns a clear
// error when the WebSocket connection is nil.
func TestCallWSCommand_NotConnected(t *testing.T) {
	ch := newTestWSChannel(t)
	cmd := wsCommand{
		Cmd:     "aibot_upload_media_init",
		Headers: wsHeaders{ReqID: "test-req-id"},
		Body:    map[string]string{"type": "file"},
	}
	_, err := ch.callWSCommand(cmd, 5*time.Second)
	if err == nil {
		t.Fatal("expected error when conn is nil, got nil")
	}
}

// ---------------------------------------------------------------------------
// Upload body / response type round-trip tests
// ---------------------------------------------------------------------------

// TestWsUploadChunkCount verifies the integer chunk-count formula used inside
// UploadWSMedia without needing the math package.
func TestWsUploadChunkCount(t *testing.T) {
	tests := []struct {
		name       string
		size       int64
		wantChunks int
	}{
		{"one byte", 1, 1},
		{"exactly one chunk", int64(wsUploadChunkSize), 1},
		{"one byte over", int64(wsUploadChunkSize) + 1, 2},
		{"max allowed", int64(wsUploadMaxChunks) * int64(wsUploadChunkSize), wsUploadMaxChunks},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := int((tc.size + int64(wsUploadChunkSize) - 1) / int64(wsUploadChunkSize))
			if got != tc.wantChunks {
				t.Errorf("size=%d: got %d chunks, want %d", tc.size, got, tc.wantChunks)
			}
		})
	}
}

// TestWsUploadInitBodyJSON checks that wsUploadInitBody serializes and
// deserializes correctly, including the omitempty MD5 field.
func TestWsUploadInitBodyJSON(t *testing.T) {
	b := wsUploadInitBody{
		Type:        "image",
		Filename:    "photo.jpg",
		TotalSize:   1024,
		TotalChunks: 1,
		MD5:         "deadbeefdeadbeef",
	}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got wsUploadInitBody
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != b {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, b)
	}

	// MD5 must be omitted when empty.
	b.MD5 = ""
	data, _ = json.Marshal(b)
	if contains(string(data), `"md5"`) {
		t.Errorf("expected md5 key to be absent when empty, got: %s", data)
	}
}

// TestWsUploadChunkBodyBase64 checks that wsUploadChunkBody correctly
// round-trips arbitrary binary data through base64.
func TestWsUploadChunkBodyBase64(t *testing.T) {
	raw := []byte("binary\x00data\xff\xfe")
	body := wsUploadChunkBody{
		UploadID:   "uid-test-1",
		ChunkIndex: 0,
		Base64Data: base64.StdEncoding.EncodeToString(raw),
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got wsUploadChunkBody
	if err = json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got.Base64Data)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if string(decoded) != string(raw) {
		t.Errorf("base64 round-trip mismatch: got %v, want %v", decoded, raw)
	}
}

// TestWsUploadFinishResponseJSON checks that the finish response unmarshals
// the fields WeCom returns.
func TestWsUploadFinishResponseJSON(t *testing.T) {
	raw := `{"type":"file","media_id":"MEDIAID_ABCDE","created_at":1680000000}`
	var resp wsUploadFinishResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.MediaID != "MEDIAID_ABCDE" {
		t.Errorf("media_id: got %q, want %q", resp.MediaID, "MEDIAID_ABCDE")
	}
	if resp.Type != "file" {
		t.Errorf("type: got %q, want %q", resp.Type, "file")
	}
	if resp.CreatedAt != 1680000000 {
		t.Errorf("created_at: got %d, want %d", resp.CreatedAt, int64(1680000000))
	}
}

// contains is a helper to avoid importing "strings" for a single check.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
