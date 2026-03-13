package feishu

type ParsedMessage struct {
	MessageID  string         `json:"message_id,omitempty"`
	ChatID     string         `json:"chat_id,omitempty"`
	MsgType    string         `json:"msg_type,omitempty"`
	CreateTime string         `json:"create_time,omitempty"`
	Sender     map[string]any `json:"sender,omitempty"`
	Content    any            `json:"content,omitempty"`
}

type CardComponent struct {
	Index   int            `json:"index"`
	Tag     string         `json:"tag,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type CardSummary struct {
	Title         string          `json:"title,omitempty"`
	Components    []CardComponent `json:"components,omitempty"`
	ImageKeys     []string        `json:"image_keys,omitempty"`
	TextContents  []string        `json:"text_contents,omitempty"`
	ActionButtons []string        `json:"action_buttons,omitempty"`
}

type MessageDetail struct {
	MessageID  string           `json:"message_id,omitempty"`
	ChatID     string           `json:"chat_id,omitempty"`
	RootID     string           `json:"root_id,omitempty"`
	ParentID   string           `json:"parent_id,omitempty"`
	MsgType    string           `json:"msg_type,omitempty"`
	Deleted    bool             `json:"deleted,omitempty"`
	Updated    bool             `json:"updated,omitempty"`
	CreateTime string           `json:"create_time,omitempty"`
	UpdateTime string           `json:"update_time,omitempty"`
	Sender     map[string]any   `json:"sender,omitempty"`
	Body       map[string]any   `json:"body,omitempty"`
	Mentions   []map[string]any `json:"mentions,omitempty"`
	Raw        map[string]any   `json:"raw,omitempty"`
	Parsed     *ParsedMessage   `json:"parsed,omitempty"`
	CardParsed *CardSummary     `json:"card_parsed,omitempty"`
}

type MessageList struct {
	Items     []MessageDetail `json:"items,omitempty"`
	HasMore   bool            `json:"has_more,omitempty"`
	PageToken string          `json:"page_token,omitempty"`
}

type UserSummary struct {
	ID        string         `json:"id,omitempty"`
	OpenID    string         `json:"open_id,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	UnionID   string         `json:"union_id,omitempty"`
	Name      string         `json:"name,omitempty"`
	EnName    string         `json:"en_name,omitempty"`
	Email     string         `json:"email,omitempty"`
	Mobile    string         `json:"mobile,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Status    map[string]any `json:"status,omitempty"`
	Raw       map[string]any `json:"raw,omitempty"`
}

type UserList struct {
	Items     []UserSummary `json:"items,omitempty"`
	HasMore   bool          `json:"has_more,omitempty"`
	PageToken string        `json:"page_token,omitempty"`
}

type ChatSummary struct {
	ChatID      string         `json:"chat_id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	ChatMode    string         `json:"chat_mode,omitempty"`
	ChatType    string         `json:"chat_type,omitempty"`
	OwnerID     string         `json:"owner_id,omitempty"`
	OwnerOpenID string         `json:"owner_open_id,omitempty"`
	External    bool           `json:"external,omitempty"`
	TenantKey   string         `json:"tenant_key,omitempty"`
	Avatar      string         `json:"avatar,omitempty"`
	Raw         map[string]any `json:"raw,omitempty"`
}

type ChatList struct {
	Items     []ChatSummary `json:"items,omitempty"`
	HasMore   bool          `json:"has_more,omitempty"`
	PageToken string        `json:"page_token,omitempty"`
}

type ShareLinkLookupResult struct {
	Token            string         `json:"token,omitempty"`
	DecodedMessageID string         `json:"decoded_message_id,omitempty"`
	Message          *MessageDetail `json:"message,omitempty"`
	FallbackError    string         `json:"fallback_error,omitempty"`
}

type DriveFileSummary struct {
	FileToken    string         `json:"file_token,omitempty"`
	Name         string         `json:"name,omitempty"`
	Type         string         `json:"type,omitempty"`
	ParentToken  string         `json:"parent_token,omitempty"`
	Size         int64          `json:"size,omitempty"`
	Extension    string         `json:"extension,omitempty"`
	MimeType     string         `json:"mime_type,omitempty"`
	URL          string         `json:"url,omitempty"`
	CreatedTime  string         `json:"created_time,omitempty"`
	ModifiedTime string         `json:"modified_time,omitempty"`
	Raw          map[string]any `json:"raw,omitempty"`
}

type DriveFolderSummary struct {
	FolderToken  string         `json:"folder_token,omitempty"`
	Name         string         `json:"name,omitempty"`
	ParentToken  string         `json:"parent_token,omitempty"`
	URL          string         `json:"url,omitempty"`
	CreatedTime  string         `json:"created_time,omitempty"`
	ModifiedTime string         `json:"modified_time,omitempty"`
	Raw          map[string]any `json:"raw,omitempty"`
}

type DriveFileList struct {
	Items     []DriveFileSummary `json:"items,omitempty"`
	HasMore   bool               `json:"has_more,omitempty"`
	PageToken string             `json:"page_token,omitempty"`
}

type MultipartUploadSession struct {
	FileToken string `json:"file_token,omitempty"`
	UploadID  string `json:"upload_id,omitempty"`
	BlockSize int    `json:"block_size,omitempty"`
}

type DownloadedFile struct {
	Name        string `json:"name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Data        []byte `json:"data,omitempty"`
}
