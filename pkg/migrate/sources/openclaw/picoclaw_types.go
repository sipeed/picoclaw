package openclaw

type ModelConfig struct {
	ModelName string `json:"model_name"`
	Model     string `json:"model"`
	APIBase   string `json:"api_base,omitempty"`
	APIKey    string `json:"api_key"`
	Proxy     string `json:"proxy,omitempty"`
}

type PicoClawConfig struct {
	Agents    AgentsConfig   `json:"agents"`
	Bindings  []AgentBinding `json:"bindings,omitempty"`
	Channels  ChannelsConfig `json:"channels"`
	ModelList []ModelConfig  `json:"model_list"`
	Gateway   GatewayConfig  `json:"gateway"`
	Tools     ToolsConfig    `json:"tools"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

type AgentDefaults struct {
	Workspace           string   `json:"workspace"`
	RestrictToWorkspace bool     `json:"restrict_to_workspace"`
	Provider            string   `json:"provider"`
	ModelName           string   `json:"model_name"`
	Model               string   `json:"model,omitempty"`
	ModelFallbacks      []string `json:"model_fallbacks,omitempty"`
	ImageModel          string   `json:"image_model,omitempty"`
	ImageModelFallbacks []string `json:"image_model_fallbacks,omitempty"`
	MaxTokens           int      `json:"max_tokens"`
	Temperature         *float64 `json:"temperature,omitempty"`
	MaxToolIterations   int      `json:"max_tool_iterations"`
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
}

type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

type AgentBinding struct {
	AgentID string       `json:"agent_id"`
	Match   BindingMatch `json:"match"`
}

type BindingMatch struct {
	Channel   string     `json:"channel"`
	AccountID string     `json:"account_id,omitempty"`
	Peer      *PeerMatch `json:"peer,omitempty"`
	GuildID   string     `json:"guild_id,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
}

type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	MaixCam  MaixCamConfig  `json:"maixcam"`
	QQ       QQConfig       `json:"qq"`
	DingTalk DingTalkConfig `json:"dingtalk"`
	Slack    SlackConfig    `json:"slack"`
	Matrix   MatrixConfig   `json:"matrix"`
	LINE     LINEConfig     `json:"line"`
}

type WhatsAppConfig struct {
	Enabled   bool     `json:"enabled"`
	BridgeURL string   `json:"bridge_url"`
	AllowFrom []string `json:"allow_from"`
}

type TelegramConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"token"`
	Proxy     string   `json:"proxy"`
	AllowFrom []string `json:"allow_from"`
}

type DiscordConfig struct {
	Enabled     bool     `json:"enabled"`
	Token       string   `json:"token"`
	MentionOnly bool     `json:"mention_only"`
	AllowFrom   []string `json:"allow_from"`
}

type MaixCamConfig struct {
	Enabled   bool     `json:"enabled"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	AllowFrom []string `json:"allow_from"`
}

type QQConfig struct {
	Enabled   bool     `json:"enabled"`
	AppID     string   `json:"app_id"`
	AppSecret string   `json:"app_secret"`
	AllowFrom []string `json:"allow_from"`
}

type DingTalkConfig struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AllowFrom    []string `json:"allow_from"`
}

type SlackConfig struct {
	Enabled   bool     `json:"enabled"`
	BotToken  string   `json:"bot_token"`
	AppToken  string   `json:"app_token"`
	AllowFrom []string `json:"allow_from"`
}

type MatrixConfig struct {
	Enabled     bool     `json:"enabled"`
	Homeserver  string   `json:"homeserver"`
	UserID      string   `json:"user_id"`
	AccessToken string   `json:"access_token"`
	AllowFrom   []string `json:"allow_from"`
}

type LINEConfig struct {
	Enabled            bool     `json:"enabled"`
	ChannelSecret      string   `json:"channel_secret"`
	ChannelAccessToken string   `json:"channel_access_token"`
	WebhookHost        string   `json:"webhook_host"`
	WebhookPort        int      `json:"webhook_port"`
	WebhookPath        string   `json:"webhook_path"`
	AllowFrom          []string `json:"allow_from"`
}

type GatewayConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ToolsConfig struct {
	Web  WebToolsConfig `json:"web"`
	Cron CronConfig     `json:"cron"`
	Exec ExecConfig     `json:"exec"`
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave"`
	Tavily     TavilyConfig     `json:"tavily"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
	Perplexity PerplexityConfig `json:"perplexity"`
	Proxy      string           `json:"proxy,omitempty"`
}

type BraveConfig struct {
	Enabled    bool     `json:"enabled"`
	APIKey     string   `json:"api_key"`
	APIKeys    []string `json:"api_keys"`
	MaxResults int      `json:"max_results"`
}

type TavilyConfig struct {
	Enabled    bool     `json:"enabled"`
	APIKey     string   `json:"api_key"`
	APIKeys    []string `json:"api_keys"`
	BaseURL    string   `json:"base_url"`
	MaxResults int      `json:"max_results"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled"`
	MaxResults int  `json:"max_results"`
}

type PerplexityConfig struct {
	Enabled    bool     `json:"enabled"`
	APIKey     string   `json:"api_key"`
	APIKeys    []string `json:"api_keys"`
	MaxResults int      `json:"max_results"`
}

type CronConfig struct {
	ExecTimeoutMinutes int `json:"exec_timeout_minutes"`
}

type ExecConfig struct {
	EnableDenyPatterns bool     `json:"enable_deny_patterns"`
	CustomDenyPatterns []string `json:"custom_deny_patterns"`
}
