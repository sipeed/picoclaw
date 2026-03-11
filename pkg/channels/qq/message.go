package qq

import "github.com/tencent-connect/botgo/dto"

// RichMediaMessage rich media message.
// It is recommended to upload first, then send using message type 7.
type RichMediaMessage struct {
	FileType uint64 `json:"file_type,omitempty"` // file type: 1-image, 2-video, 3-voice (currently voice only supports silk format)
	URL      string `json:"url,omitempty"`       // rich media file to send, HTTP or HTTPS link
	FileName string `json:"file_name,omitempty"` // file name for files sent via FileData
	FileData []byte `json:"file_data,omitempty"` // file binary data for files sent via FileData
}

// GetEventID event ID
func (msg RichMediaMessage) GetEventID() string {
	return ""
}

// GetSendType message type
func (msg RichMediaMessage) GetSendType() dto.SendType {
	return dto.RichMedia
}

// MessageAttachment attachment definition
type MessageAttachment struct {
	URL          string `json:"url,omitempty"`
	FileName     string `json:"filename,omitempty"`
	ContentType  string `json:"content_type,omitempty"`   // voice: audio, image/xxx: image, video/xxx: video
	AsrReferText string `json:"asr_refer_text,omitempty"` // ASR reference text
}
