package wecom

import "encoding/json"

const (
	wecomDefaultWebSocketURL = "wss://openws.work.weixin.qq.com"
	wecomCmdSubscribe        = "aibot_subscribe"
	wecomCmdPing             = "ping"
	wecomCmdMsgCallback      = "aibot_msg_callback"
	wecomCmdEventCallback    = "aibot_event_callback"
	wecomCmdRespondMsg       = "aibot_respond_msg"
	wecomCmdSendMsg          = "aibot_send_msg"
	wecomCmdUploadMediaInit  = "aibot_upload_media_init"
	wecomCmdUploadMediaChunk = "aibot_upload_media_chunk"
	wecomCmdUploadMediaEnd   = "aibot_upload_media_finish"
)

type wecomEnvelope struct {
	Cmd     string          `json:"cmd,omitempty"`
	Headers wecomHeaders    `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode int             `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
}

type wecomHeaders struct {
	ReqID string `json:"req_id,omitempty"`
}

type wecomCommand struct {
	Cmd     string       `json:"cmd"`
	Headers wecomHeaders `json:"headers"`
	Body    any          `json:"body,omitempty"`
}

type wecomSendMsgBody struct {
	ChatID       string                `json:"chatid"`
	ChatType     uint32                `json:"chat_type,omitempty"`
	MsgType      string                `json:"msgtype"`
	Markdown     *wecomMarkdownContent `json:"markdown,omitempty"`
	File         *wecomMediaRefContent `json:"file,omitempty"`
	Image        *wecomMediaRefContent `json:"image,omitempty"`
	Voice        *wecomMediaRefContent `json:"voice,omitempty"`
	Video        *wecomVideoContent    `json:"video,omitempty"`
	TemplateCard map[string]any        `json:"template_card,omitempty"`
}

type wecomRespondMsgBody struct {
	MsgType      string                `json:"msgtype"`
	Stream       *wecomStreamContent   `json:"stream,omitempty"`
	Markdown     *wecomMarkdownContent `json:"markdown,omitempty"`
	File         *wecomMediaRefContent `json:"file,omitempty"`
	Image        *wecomMediaRefContent `json:"image,omitempty"`
	Voice        *wecomMediaRefContent `json:"voice,omitempty"`
	Video        *wecomVideoContent    `json:"video,omitempty"`
	TemplateCard map[string]any        `json:"template_card,omitempty"`
}

type wecomStreamContent struct {
	ID      string `json:"id"`
	Finish  bool   `json:"finish"`
	Content string `json:"content,omitempty"`
}

type wecomMarkdownContent struct {
	Content string `json:"content"`
}

type wecomMediaRefContent struct {
	MediaID string `json:"media_id"`
}

type wecomVideoContent struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type wecomUploadMediaInitBody struct {
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	MD5         string `json:"md5,omitempty"`
}

type wecomUploadMediaInitResponse struct {
	UploadID string `json:"upload_id"`
}

type wecomUploadMediaChunkBody struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}

type wecomUploadMediaFinishBody struct {
	UploadID string `json:"upload_id"`
}

type wecomUploadMediaFinishResponse struct {
	Type      string          `json:"type"`
	MediaID   string          `json:"media_id"`
	CreatedAt json.RawMessage `json:"created_at"`
}

type wecomIncomingMessage struct {
	MsgID    string `json:"msgid"`
	AIBotID  string `json:"aibotid"`
	ChatID   string `json:"chatid,omitempty"`
	ChatType string `json:"chattype,omitempty"`
	From     struct {
		UserID string `json:"userid"`
	} `json:"from"`
	MsgType string `json:"msgtype"`
	Text    *struct {
		Content string `json:"content"`
	} `json:"text,omitempty"`
	Image *struct {
		URL    string `json:"url"`
		AESKey string `json:"aeskey,omitempty"`
	} `json:"image,omitempty"`
	File *struct {
		URL    string `json:"url"`
		AESKey string `json:"aeskey,omitempty"`
	} `json:"file,omitempty"`
	Video *struct {
		URL    string `json:"url"`
		AESKey string `json:"aeskey,omitempty"`
	} `json:"video,omitempty"`
	Voice *struct {
		Content string `json:"content"`
	} `json:"voice,omitempty"`
	Mixed *struct {
		MsgItem []struct {
			MsgType string `json:"msgtype"`
			Text    *struct {
				Content string `json:"content"`
			} `json:"text,omitempty"`
			Image *struct {
				URL    string `json:"url"`
				AESKey string `json:"aeskey,omitempty"`
			} `json:"image,omitempty"`
			File *struct {
				URL    string `json:"url"`
				AESKey string `json:"aeskey,omitempty"`
			} `json:"file,omitempty"`
		} `json:"msg_item"`
	} `json:"mixed,omitempty"`
	Quote *struct {
		MsgType string `json:"msgtype"`
		Text    *struct {
			Content string `json:"content"`
		} `json:"text,omitempty"`
	} `json:"quote,omitempty"`
	Event *struct {
		EventType string `json:"eventtype"`
	} `json:"event,omitempty"`
}

func incomingChatID(msg wecomIncomingMessage) string {
	if msg.ChatID != "" {
		return msg.ChatID
	}
	return msg.From.UserID
}

func incomingChatTypeCode(kind string) uint32 {
	if kind == "group" {
		return 2
	}
	return 1
}
