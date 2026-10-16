package qqc2c

import (
	"encoding/json"
	"sync/atomic"
	"testing"
)

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
