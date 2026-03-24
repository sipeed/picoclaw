package pico

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestPicoChannel_PostSend_Unauthorized(t *testing.T) {
	t.Parallel()
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	body := `{"type":"message.send","session_id":"s1","id":"1","payload":{"content":"hi"}}`
	req := httptest.NewRequest(http.MethodPost, "/pico/send", strings.NewReader(body))
	rec := httptest.NewRecorder()
	ch.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestPicoChannel_PostSend_EmptyContent(t *testing.T) {
	t.Parallel()
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	body := `{"type":"message.send","session_id":"s1","id":"1","payload":{"content":"   "}}`
	req := httptest.NewRequest(http.MethodPost, "/pico/send", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	ch.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
	var m map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["error"] != "empty_content" {
		t.Fatalf("error = %q, want empty_content", m["error"])
	}
}

func TestPicoChannel_PostSend_MissingSession(t *testing.T) {
	t.Parallel()
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	body := `{"type":"message.send","id":"1","payload":{"content":"hi"}}`
	req := httptest.NewRequest(http.MethodPost, "/pico/send", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	ch.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestPicoChannel_PostSend_OK(t *testing.T) {
	t.Parallel()
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	body := `{"type":"message.send","session_id":"s1","id":"m1","payload":{"content":"hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/pico/send", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	ch.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204", rec.Code)
	}
}

func TestPicoChannel_GetEvents_Unauthorized(t *testing.T) {
	t.Parallel()
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	req := httptest.NewRequest(http.MethodGet, "/pico/events?session_id=s1", nil)
	rec := httptest.NewRecorder()
	ch.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestPicoChannel_SSE_ReceivesBroadcast(t *testing.T) {
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	ctxReq, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/pico/events?session_id=sse-s1", nil)
	req = req.WithContext(ctxReq)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch.ServeHTTP(rec, req)
	}()

	time.Sleep(200 * time.Millisecond)

	if err := ch.Send(ctx, bus.OutboundMessage{ChatID: "pico:sse-s1", Content: "broadcast-body"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	cancel()
	wg.Wait()

	out := rec.Body.String()
	if !strings.Contains(out, "broadcast-body") {
		t.Fatalf("expected outbound in body, got: %q", out)
	}
	if !strings.Contains(out, "message.create") {
		t.Fatalf("expected message type in stream, got: %q", out)
	}
}

func TestPicoChannel_SSE_SecondConnectionSameSessionReplacesFirst(t *testing.T) {
	ch, err := NewPicoChannel(config.PicoConfig{Token: "tok"}, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ch.Stop(ctx) })

	const sess = "shared-sse"

	ctx1, cancel1 := context.WithCancel(context.Background())
	req1 := httptest.NewRequest(http.MethodGet, "/pico/events?session_id="+sess, nil).WithContext(ctx1)
	req1.Header.Set("Authorization", "Bearer tok")
	rec1 := httptest.NewRecorder()
	go ch.ServeHTTP(rec1, req1)
	time.Sleep(200 * time.Millisecond)

	ctx2, cancel2 := context.WithCancel(context.Background())
	req2 := httptest.NewRequest(http.MethodGet, "/pico/events?session_id="+sess, nil).WithContext(ctx2)
	req2.Header.Set("Authorization", "Bearer tok")
	rec2 := httptest.NewRecorder()
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		ch.ServeHTTP(rec2, req2)
	}()
	time.Sleep(200 * time.Millisecond)

	if err := ch.Send(ctx, bus.OutboundMessage{ChatID: "pico:" + sess, Content: "only-once"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	n1 := strings.Count(rec1.Body.String(), "only-once")
	n2 := strings.Count(rec2.Body.String(), "only-once")
	if n2 != 1 {
		t.Fatalf("second SSE should receive exactly one broadcast, got n2=%d", n2)
	}
	if n1 != 0 {
		t.Fatalf("first SSE should be replaced and not receive broadcast, got n1=%d", n1)
	}

	cancel2()
	wg2.Wait()
	cancel1()
	time.Sleep(100 * time.Millisecond)
}
