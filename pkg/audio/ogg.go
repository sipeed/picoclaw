package audio

import (
	"bytes"
	"fmt"
	"io"
)

// DecodeOggOpus reads an Ogg format stream and extracts individual Opus payloads.
// It calls onFrame for every complete Opus frame found in the stream.
func DecodeOggOpus(r io.Reader, onFrame func([]byte) error) error {
	var packet []byte
	header := make([]byte, 27)

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
			segment := make([]byte, lacing)
			if _, err := io.ReadFull(r, segment); err != nil {
				return fmt.Errorf("failed to read segment data: %w", err)
			}

			packet = append(packet, segment...)

			// If lacing is less than 255, the packet is complete
			if lacing < 255 {
				if len(packet) > 0 {
					// Ignore Ogg Opus headers
					if !bytes.HasPrefix(packet, []byte("OpusHead")) && !bytes.HasPrefix(packet, []byte("OpusTags")) {
						if err := onFrame(packet); err != nil {
							return err
						}
					}
					// Start new packet
					packet = nil
				}
			}
		}
	}
}
