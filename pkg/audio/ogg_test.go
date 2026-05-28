package audio

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// buildOggPage helper creates an Ogg page for testing.
// lacingVals specifies the segment table, and data is the payload.
func buildOggPage(lacingVals []byte, data []byte) []byte {
	var buf bytes.Buffer
	// 27-byte Ogg header
	header := make([]byte, 27)
	copy(header[:4], "OggS")
	header[5] = 0 // type flag
	// For testing, we only care about OggS magic and page_segments (byte 26)
	header[26] = byte(len(lacingVals))
	buf.Write(header)
	buf.Write(lacingVals)
	buf.Write(data)
	return buf.Bytes()
}

func TestDecodeOggOpus_ValidParsing(t *testing.T) {
	var b bytes.Buffer

	// Packet 1: Single segment, length 50
	pkt1 := bytes.Repeat([]byte{1}, 50)
	// Packet 2: Multi-segment (255 + 10 = 265 bytes)
	pkt2Part1 := bytes.Repeat([]byte{2}, 255)
	pkt2Part2 := bytes.Repeat([]byte{2}, 10)
	// Packet 3: Continued across pages. Page 1 gets 255, Page 2 gets 20. Total 275 bytes.
	pkt3Part1 := bytes.Repeat([]byte{3}, 255)
	pkt3Part2 := bytes.Repeat([]byte{3}, 20)

	// Page 1: OpusHead (skip), OpusTags (skip), pkt1, pkt2, pkt3Part1
	page1Lacing := []byte{8, 8, 50, 255, 10, 255}
	page1Data := bytes.Join([][]byte{
		[]byte("OpusHead"),
		[]byte("OpusTags"),
		pkt1,
		pkt2Part1, pkt2Part2,
		pkt3Part1,
	}, nil)

	// Page 2: pkt3Part2, pkt4 (length 10)
	pkt4 := bytes.Repeat([]byte{4}, 10)
	page2Lacing := []byte{20, 10}
	page2Data := bytes.Join([][]byte{
		pkt3Part2,
		pkt4,
	}, nil)

	b.Write(buildOggPage(page1Lacing, page1Data))
	b.Write(buildOggPage(page2Lacing, page2Data))

	var frames [][]byte
	err := DecodeOggOpus(&b, func(frame []byte) error {
		// making a copy to store as DecodeOggOpus might reuse backing array
		cpy := make([]byte, len(frame))
		copy(cpy, frame)
		frames = append(frames, cpy)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFrames := [][]byte{
		pkt1,
		append(pkt2Part1, pkt2Part2...),
		append(pkt3Part1, pkt3Part2...),
		pkt4,
	}

	if len(frames) != len(expectedFrames) {
		t.Fatalf("expected %d frames, got %d", len(expectedFrames), len(frames))
	}

	for i, expected := range expectedFrames {
		if !reflect.DeepEqual(frames[i], expected) {
			t.Errorf("frame %d mismatch:\nexp: %v\ngot: %v", i, expected, frames[i])
		}
	}
}

func TestDecodeOggOpus_Errors(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		errContains string
	}{
		{
			name: "invalid magic string",
			data: []byte(
				"OggX\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
			),
			errContains: "invalid ogg magic string",
		},
		{
			name:        "short header",
			data:        []byte("Ogg"),
			errContains: "failed to read ogg header",
		},
		{
			name: "eof in segment table",
			data: func() []byte {
				h := make([]byte, 27)
				copy(h, "OggS")
				h[26] = 5 // expects 5 bytes of segment table, but none provided
				return h
			}(),
			errContains: "failed to read segment table",
		},
		{
			name: "eof in segment data",
			data: func() []byte {
				h := make([]byte, 27, 28)
				copy(h, "OggS")
				h[26] = 1
				return append(h, 100) // expects 100 bytes of data, but none provided
			}(),
			errContains: "failed to read segment data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DecodeOggOpus(bytes.NewReader(tt.data), func(b []byte) error { return nil })
			if tt.name == "short header" {
				if err != nil {
					t.Errorf("expected no error (io.EOF/ErrUnexpectedEOF swallowed), got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error to contain %q, got: %q", tt.errContains, err.Error())
			}
		})
	}
}
