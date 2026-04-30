package audio

import (
	"bytes"
	"fmt"
	"io"
)

// DecodeOggOpus reads an Ogg format stream and extracts individual Opus payloads.
// It calls onFrame for every complete Opus frame found in the stream.
func DecodeOggOpus(r io.Reader, onFrame func([]byte) error) error {
	var packet bytes.Buffer
	header := make([]byte, 27)
	segment := make([]byte, 255)

	for {
		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("failed to read ogg header: %w", err)
		}
		if string(header[:4]) != "OggS" {
			return fmt.Errorf("invalid ogg magic string")
		}

		pageSegments := int(header[26])
		segmentTable := make([]byte, pageSegments)
		if _, err := io.ReadFull(r, segmentTable); err != nil {
			return fmt.Errorf("failed to read segment table: %w", err)
		}

		for _, lacing := range segmentTable {
			if _, err := io.ReadFull(r, segment[:lacing]); err != nil {
				return fmt.Errorf("failed to read segment data: %w", err)
			}

			packet.Write(segment[:lacing])

			// If lacing is less than 255, the packet is complete
			if lacing < 255 {
				if packet.Len() > 0 {
					raw := packet.Bytes()
					// Ignore Ogg Opus headers
					if !bytes.HasPrefix(raw, []byte("OpusHead")) &&
						!bytes.HasPrefix(raw, []byte("OpusTags")) {
						// Copy the frame data: packet.Reset() reuses the
						// underlying array, so the slice would be
						// overwritten by subsequent packets before the
						// consumer (e.g. OpusSend channel) reads it.
						frame := make([]byte, len(raw))
						copy(frame, raw)
						if err := onFrame(frame); err != nil {
							return err
						}
					}
					// Start new packet
					packet.Reset()
				}
			}
		}
	}
}
