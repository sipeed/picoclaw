// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"strings"
	"testing"
)

func TestLastSentenceBreak_Empty(t *testing.T) {
	if got := lastSentenceBreak(""); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestLastSentenceBreak_NoTerminator(t *testing.T) {
	if got := lastSentenceBreak("你好世界 hello world"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestLastSentenceBreak_ChinesePeriod(t *testing.T) {
	s := "你好。"
	idx := lastSentenceBreak(s)
	if idx != len(s) {
		t.Errorf("got %d, want %d (end of string)", idx, len(s))
	}
}

func TestLastSentenceBreak_MultipleTerminators(t *testing.T) {
	// 应返回最后一个句子边界之后
	s := "你好。再见！"
	idx := lastSentenceBreak(s)
	if idx != len(s) {
		t.Errorf("got %d, want %d", idx, len(s))
	}
}

func TestLastSentenceBreak_TrailingText(t *testing.T) {
	// 断句后有剩余内容
	s := "你好。还有更多内容"
	// "你好。" = 9 bytes（每个汉字3字节，句号3字节）
	wantIdx := len("你好。")
	idx := lastSentenceBreak(s)
	if idx != wantIdx {
		t.Errorf("got %d, want %d", idx, wantIdx)
	}
	remaining := s[idx:]
	if remaining != "还有更多内容" {
		t.Errorf("remaining = %q, want '还有更多内容'", remaining)
	}
}

func TestLastSentenceBreak_EnglishPeriod(t *testing.T) {
	s := "Hello. World"
	idx := lastSentenceBreak(s)
	if idx != len("Hello.") {
		t.Errorf("got %d, want %d", idx, len("Hello."))
	}
}

func TestLastSentenceBreak_Newline(t *testing.T) {
	s := "line one\nline two"
	idx := lastSentenceBreak(s)
	if idx != len("line one\n") {
		t.Errorf("got %d, want %d", idx, len("line one\n"))
	}
}

func TestLastSentenceBreak_MixedScript(t *testing.T) {
	s := "你好！Hello.再见"
	// '！' 是全角感叹号，直接切；英文 '.' 后跟汉字（非空格），不满足切分条件。
	// 所以最后一个断点是 '！' 之后，即 "你好！" 的字节长度。
	want := len("你好！")
	if got := lastSentenceBreak(s); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestLastSentenceBreak_OnlyTerminator(t *testing.T) {
	s := "。"
	if got := lastSentenceBreak(s); got != len(s) {
		t.Errorf("got %d, want %d", got, len(s))
	}
}

// ---- thinking 状态机测试 ----
// 通过 simulateTokens 模拟 onToken 回调，验证 thinking 块过滤和断句行为。

func simulateTokens(tokens []string) (sentences []string, thinkingLogged bool) {
	const startTag = "<think>"
	const endTag = "</think>"

	var normalBuf strings.Builder
	thinkTail := make([]byte, 0, len(endTag)-1)
	inThinking := false
	gotThinking := false

	flushNormal := func() {
		text := normalBuf.String()
		if idx := lastSentenceBreak(text); idx > 0 {
			sentences = append(sentences, text[:idx])
			remaining := text[idx:]
			normalBuf.Reset()
			normalBuf.WriteString(remaining)
		}
	}

	onToken := func(token string) {
		for token != "" {
			if inThinking {
				scan := string(thinkTail) + token
				if idx := strings.Index(scan, endTag); idx >= 0 {
					gotThinking = true
					after := scan[idx+len(endTag):]
					thinkTail = thinkTail[:0]
					inThinking = false
					token = after
				} else {
					if len(scan) >= len(endTag)-1 {
						thinkTail = []byte(scan[len(scan)-(len(endTag)-1):])
					} else {
						thinkTail = []byte(scan)
					}
					token = ""
				}
			} else {
				normalBuf.WriteString(token)
				token = ""
				combined := normalBuf.String()
				if idx := strings.Index(combined, startTag); idx >= 0 {
					before := combined[:idx]
					after := combined[idx+len(startTag):]
					normalBuf.Reset()
					normalBuf.WriteString(before)
					flushNormal()
					inThinking = true
					thinkTail = thinkTail[:0]
					token = after
				} else {
					flushNormal()
				}
			}
		}
	}

	for _, t := range tokens {
		onToken(t)
	}
	// 处理剩余
	if !inThinking && normalBuf.Len() > 0 {
		sentences = append(sentences, normalBuf.String())
	}
	return sentences, gotThinking
}

func TestThinking_NoThinkTag(t *testing.T) {
	// 非 thinking 模式：所有内容正常断句
	tokens := []string{"你好，", "我是 AI。", "有什么可以帮您？"}
	sentences, thinking := simulateTokens(tokens)
	if thinking {
		t.Error("should not detect thinking")
	}
	// "我是 AI。" 和 "有什么可以帮您？" 各触发一次断句
	if len(sentences) != 2 {
		t.Errorf("want 2 sentences, got %d: %v", len(sentences), sentences)
	}
	if sentences[0] != "你好，我是 AI。" {
		t.Errorf("sentences[0] = %q", sentences[0])
	}
	if sentences[1] != "有什么可以帮您？" {
		t.Errorf("sentences[1] = %q", sentences[1])
	}
}

func TestThinking_ThinkBlockSkipped(t *testing.T) {
	// thinking 模式：<think>...</think> 内容不送 TTS
	tokens := []string{"<think>让我想一想。</think>", "你好！"}
	sentences, thinking := simulateTokens(tokens)
	if !thinking {
		t.Error("should detect thinking")
	}
	if len(sentences) != 1 {
		t.Errorf("want 1 sentence, got %d: %v", len(sentences), sentences)
	}
	if sentences[0] != "你好！" {
		t.Errorf("sentence[0] = %q", sentences[0])
	}
}

func TestThinking_TagSplitAcrossTokens(t *testing.T) {
	// </think> 跨 token 边界
	tokens := []string{"<think>thinking", "</thi", "nk>回答。"}
	sentences, thinking := simulateTokens(tokens)
	if !thinking {
		t.Error("should detect thinking")
	}
	if len(sentences) != 1 || sentences[0] != "回答。" {
		t.Errorf("sentences = %v", sentences)
	}
}

func TestThinking_ContentBeforeThink(t *testing.T) {
	// <think> 前已有正常内容
	tokens := []string{"先说一句。<think>思考中</think>然后继续。"}
	sentences, thinking := simulateTokens(tokens)
	if !thinking {
		t.Error("should detect thinking")
	}
	if len(sentences) != 2 {
		t.Fatalf("want 2 sentences, got %d: %v", len(sentences), sentences)
	}
	if sentences[0] != "先说一句。" {
		t.Errorf("sentences[0] = %q", sentences[0])
	}
	if sentences[1] != "然后继续。" {
		t.Errorf("sentences[1] = %q", sentences[1])
	}
}

func TestThinking_MultipleNewlinesInsideThink(t *testing.T) {
	// thinking 块内大量换行不触发断句
	tokens := strings.Split("<think>\n\n\n很多换行\n\n\n</think>好的。", "")
	sentences, thinking := simulateTokens(tokens)
	if !thinking {
		t.Error("should detect thinking")
	}
	if len(sentences) != 1 || sentences[0] != "好的。" {
		t.Errorf("sentences = %v", sentences)
	}
}
