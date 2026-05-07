package qq

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const qqVoiceMaxDuration = 60 * time.Second

func qqAudioDuration(localPath, filename, contentType string) (time.Duration, bool, error) {
	if localPath == "" {
		return 0, false, nil
	}

	switch qqAudioDurationFormat(localPath, filename, contentType) {
	case "wav":
		return qqWAVDuration(localPath)
	case "ogg":
		return qqOggDuration(localPath)
	default:
		return 0, false, nil
	}
}

func qqAudioDurationFormat(localPath, filename, contentType string) string {
	contentType = strings.ToLower(contentType)

	switch {
	case strings.HasPrefix(contentType, "audio/wav"), strings.HasPrefix(contentType, "audio/x-wav"):
		return "wav"
	case strings.HasPrefix(contentType, "audio/ogg"),
		contentType == "application/ogg",
		contentType == "application/x-ogg":
		return "ogg"
	}

	switch filepath.Ext(strings.ToLower(filename)) {
	case ".wav":
		return "wav"
	case ".ogg", ".opus":
		return "ogg"
	}

	switch filepath.Ext(strings.ToLower(localPath)) {
	case ".wav":
		return "wav"
	case ".ogg", ".opus":
		return "ogg"
	}

	return ""
}

func qqWAVDuration(localPath string) (time.Duration, bool, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()

	var header [12]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return 0, false, err
	}

	var order binary.ByteOrder
	switch string(header[:4]) {
	case "RIFF":
		order = binary.LittleEndian
	case "RIFX":
		order = binary.BigEndian
	default:
		return 0, false, nil
	}

	if string(header[8:12]) != "WAVE" {
		return 0, false, nil
	}

	var byteRate uint32
	var dataSize uint32
	var foundFmt bool
	var foundData bool

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(file, chunkHeader[:]); err != nil {
			if err == io.EOF {
				break
			}
			return 0, false, err
		}

		chunkSize := order.Uint32(chunkHeader[4:8])
		switch string(chunkHeader[:4]) {
		case "fmt ":
			chunkData := make([]byte, chunkSize)
			if _, err := io.ReadFull(file, chunkData); err != nil {
				return 0, false, err
			}
			if len(chunkData) >= 12 {
				byteRate = order.Uint32(chunkData[8:12])
				foundFmt = true
			}
		case "data":
			dataSize = chunkSize
			foundData = true
			if _, err := io.CopyN(io.Discard, file, int64(chunkSize)); err != nil {
				return 0, false, err
			}
		default:
			if _, err := io.CopyN(io.Discard, file, int64(chunkSize)); err != nil {
				return 0, false, err
			}
		}

		if chunkSize%2 == 1 {
			if _, err := io.CopyN(io.Discard, file, 1); err != nil {
				return 0, false, err
			}
		}

		if foundFmt && foundData {
			break
		}
	}

	if !foundFmt || !foundData || byteRate == 0 {
		return 0, false, nil
	}

	durationNS := int64(dataSize) * int64(time.Second) / int64(byteRate)
	return time.Duration(durationNS), true, nil
}

func qqOggDuration(localPath string) (time.Duration, bool, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()

	var firstPacket []byte
	var codec string
	var sampleRate uint32
	var lastGranule uint64
	var haveGranule bool

	for {
		var header [27]byte
		if _, err := io.ReadFull(file, header[:]); err != nil {
			if err == io.EOF {
				break
			}
			return 0, false, err
		}

		if string(header[:4]) != "OggS" {
			return 0, false, nil
		}

		pageSegments := int(header[26])
		segments := make([]byte, pageSegments)
		if _, err := io.ReadFull(file, segments); err != nil {
			return 0, false, err
		}

		payloadLen := 0
		for _, segLen := range segments {
			payloadLen += int(segLen)
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(file, payload); err != nil {
			return 0, false, err
		}

		granule := binary.LittleEndian.Uint64(header[6:14])
		if granule != ^uint64(0) {
			lastGranule = granule
			haveGranule = true
		}

		if codec == "" {
			offset := 0
			for _, segLen := range segments {
				firstPacket = append(firstPacket, payload[offset:offset+int(segLen)]...)
				offset += int(segLen)
				if segLen < 255 {
					codec, sampleRate = qqParseOggCodec(firstPacket)
					break
				}
			}
		}
	}

	if !haveGranule || codec == "" {
		return 0, false, nil
	}

	switch codec {
	case "opus":
		return time.Duration(lastGranule) * time.Second / 48000, true, nil
	case "vorbis":
		if sampleRate == 0 {
			return 0, false, nil
		}
		return time.Duration(lastGranule) * time.Second / time.Duration(sampleRate), true, nil
	default:
		return 0, false, nil
	}
}

func qqParseOggCodec(packet []byte) (string, uint32) {
	if len(packet) >= 8 && string(packet[:8]) == "OpusHead" {
		return "opus", 48000
	}

	if len(packet) >= 16 && packet[0] == 0x01 && string(packet[1:7]) == "vorbis" {
		sampleRate := binary.LittleEndian.Uint32(packet[12:16])
		if sampleRate > 0 {
			return "vorbis", sampleRate
		}
	}

	return "", 0
}
