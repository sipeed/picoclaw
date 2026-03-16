// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tts

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pion/opus/pkg/oggreader"
)

// ParseOggOpusPackets 从 Ogg Opus 流中解析完整 Opus 包并逐个回调。
//
// 正确处理 Ogg lacing：pion/oggreader 的 ParseNextPage 按原始 segment（最多 255 字节）
// 切割返回，一个 Opus 包若 > 255 字节会横跨多个 segment。本函数按 Ogg lacing 规则
// 拼接这些 segment，确保每次 onPacket 回调都是一个完整可解码的 Opus 包。
//
// oggreader.NewWith 内部会消耗第一个 Ogg 页（OpusHead），因此本函数从第一个
// ParseNextPage 开始，跳过 OpusTags 注释页后直接处理音频数据页。
func ParseOggOpusPackets(r io.Reader, onPacket func([]byte)) error {
	ogg, _, err := oggreader.NewWith(r)
	if err != nil {
		return fmt.Errorf("tts/ogg: init reader: %w", err)
	}
	var buf []byte
	for {
		segments, _, pageErr := ogg.ParseNextPage()
		if pageErr == io.EOF {
			break
		}
		if pageErr != nil {
			return fmt.Errorf("tts/ogg: parse page: %w", pageErr)
		}
		// 跳过 OpusTags 注释头页
		if len(segments) > 0 && bytes.HasPrefix(segments[0], []byte("OpusTags")) {
			continue
		}
		// 按 Ogg lacing 规则拼接 segments 为完整 Opus 包：
		// segment 长度 == 255 表示当前包未结束；< 255 表示当前包的最后一个 segment。
		for _, seg := range segments {
			buf = append(buf, seg...)
			if len(seg) < 255 {
				if len(buf) > 0 {
					pkt := make([]byte, len(buf))
					copy(pkt, buf)
					onPacket(pkt)
					buf = buf[:0]
				}
			}
		}
	}
	return nil
}
