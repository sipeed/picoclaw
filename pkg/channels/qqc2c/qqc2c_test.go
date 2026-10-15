package qqc2c

import (
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// keyboard / command panel
// ---------------------------------------------------------------------------

func TestBuildKeyboardRespectsPlatformLimits(t *testing.T) {
	// 7 rows requested, 7 buttons per row: must trim to 5x5.
	row := make([]Button, 0, 7)
	for i := 0; i < 7; i++ {
		row = append(row, CallbackButton("b", "L", "d"))
	}
	rows := make([][]Button, 0, 7)
	for i := 0; i < 7; i++ {
		rows = append(rows, row)
	}
	kb := BuildKeyboard(rows)
	if kb == nil || kb.Content == nil {
		t.Fatal("expected keyboard")
	}
	if len(kb.Content.Rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(kb.Content.Rows))
	}
	for _, r := range kb.Content.Rows {
		if len(r.Buttons) != 5 {
			t.Fatalf("buttons/row = %d, want 5", len(r.Buttons))
		}
	}
}

func TestBuildKeyboardEmptyIsNil(t *testing.T) {
	if kb := BuildKeyboard(nil); kb != nil {
		t.Fatal("expected nil keyboard for empty rows")
	}
	if kb := BuildKeyboard([][]Button{{}}); kb != nil {
		t.Fatal("expected nil keyboard when every row is empty")
	}
}

func TestCommandPanelTagsAreURLEncoded(t *testing.T) {
	panel := RenderCommandPanel("测试", []Command{{Name: "新闻", Text: "/news 今日", Desc: "拉取新闻"}})
	if !strings.Contains(panel, "<qqbot-cmd-enter text=") {
		t.Fatalf("panel missing cmd-enter tag: %s", panel)
	}
	// Must be url-encoded (spaces -> %20 / +), never raw.
	if strings.Contains(panel, `text="/news 今日"`) {
		t.Fatalf("tag text is not url-encoded: %s", panel)
	}
	if !strings.Contains(panel, "拉取新闻") {
		t.Fatalf("description missing: %s", panel)
	}
}

func TestCmdInputTagUsesShowFallback(t *testing.T) {
	tag := CmdInputTag("/status", "")
	// show falls back to text, and (per the platform docs) both are url-encoded.
	if !strings.Contains(tag, `show="%2Fstatus"`) {
		t.Fatalf("show should fall back to text and be url-encoded: %s", tag)
	}
	if !strings.Contains(tag, `text="%2Fstatus"`) {
		t.Fatalf("text should be url-encoded: %s", tag)
	}
	if !strings.Contains(tag, "reference=") {
		t.Fatalf("expected reference attr: %s", tag)
	}
}

// ---------------------------------------------------------------------------
// token payload decoding
// ---------------------------------------------------------------------------

func TestExpiresSecondsHandlesStringAndNumber(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{float64(7200), 7200},
		{"7200", 7200},
		{json.Number("3601"), 3601},
		{nil, 0},
	}
	for _, c := range cases {
		got := accessTokenResp{ExpiresIn: c.in}.expiresSeconds()
		if got != c.want {
			t.Errorf("expiresSeconds(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// error classification
// ---------------------------------------------------------------------------

func TestErrorClassifiers(t *testing.T) {
	notFound := &apiError{Status: 400, Body: `{"message":"请求的资源不存在(用户/群已注销)","code":11255,"err_code":40011028}`}
	if !isNotFoundErr(notFound) {
		t.Error("notFound payload should classify as not-found")
	}
	dup := &apiError{Status: 400, Body: `{"message":"msg_seq duplicate"}`}
	if !isMsgSeqErr(dup) {
		t.Error("msg_seq payload should classify as duplicate")
	}
	other := &apiError{Status: 500, Body: `{"message":"internal error"}`}
	if isNotFoundErr(other) || isMsgSeqErr(other) {
		t.Error("unrelated error must not be classified")
	}
	if isNotFoundErr(nil) || isMsgSeqErr(nil) {
		t.Error("nil error must not be classified")
	}
}

// ---------------------------------------------------------------------------
// passive-reply sequence numbers
// ---------------------------------------------------------------------------

func TestNextSeqIsMonotonicPerChat(t *testing.T) {
	c := &Channel{}
	a := c.nextSeq("user-a")
	b := c.nextSeq("user-a")
	if b <= a {
		t.Fatalf("msg_seq must increase for the same chat: %d then %d", a, b)
	}
	c1 := c.nextSeq("user-b")
	if c1 == b {
		t.Fatalf("different chats must keep independent counters (both %d)", c1)
	}
}

func TestNextSeqWithPrefilledCounter(t *testing.T) {
	c := &Channel{}
	s := new(atomic.Int64)
	s.Store(10)
	c.msgSeq.Store("u", s)
	if got := c.nextSeq("u"); got != 11 {
		t.Fatalf("nextSeq = %d, want 11", got)
	}
}

// ---------------------------------------------------------------------------
// streaming session behaviour (no network: Update must be a no-op when nothing
// changed or when the throttle window has not elapsed)
// ---------------------------------------------------------------------------

func TestStreamSessionSkipsUnchangedAndThrottles(t *testing.T) {
	s := &streamSession{
		openID:   "u",
		msgID:    "m",
		interval: time.Minute, // long window: nothing should be sent
		lastSent: time.Now(),
	}
	// Same content as last → no-op even with a nil api client.
	if err := s.Update(nil, "same"); err != nil {
		t.Fatalf("unchanged update must be a no-op, got %v", err)
	}
	// Empty content → no-op.
	if err := s.Update(nil, ""); err != nil {
		t.Fatalf("empty update must be a no-op, got %v", err)
	}
	// Throttled → no-op (still must not touch the api client).
	if err := s.Update(nil, "new content"); err != nil {
		t.Fatalf("throttled update must be a no-op, got %v", err)
	}
}

func TestStreamSessionCancelStopsFurtherWork(t *testing.T) {
	s := &streamSession{interval: 0}
	s.Cancel(nil)
	if !s.closed {
		t.Fatal("Cancel must mark the session closed")
	}
	if err := s.Update(nil, "x"); err != nil {
		t.Fatalf("update after cancel must be a no-op, got %v", err)
	}
}
