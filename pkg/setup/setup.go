package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

type Setup struct {
	ConfigPath    string
	Cfg           *config.Config
	Steps         []string
	Confirmed     bool
	OAuthProvider string
}

func NewSetup(configPath string) (*Setup, error) {
	s := &Setup{ConfigPath: configPath}

	if _, err := os.Stat(configPath); err == nil {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		s.Cfg = cfg
	} else {
		s.Cfg = config.DefaultConfig()
	}

	s.buildSteps()
	return s, nil
}

func (s *Setup) buildSteps() {
	s.Steps = []string{}
}

func (s *Setup) Run() error {
	m := newTuiModel(s)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui start failed: %w", err)
	}

	if err := saveConfigPath(s.ConfigPath, s.Cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if s.OAuthProvider != "" {
		fmt.Println("\nRunning OAuth login for", s.OAuthProvider, "...")
		if err := runOAuthLogin(s.OAuthProvider); err != nil {
			return fmt.Errorf("oauth login failed: %w", err)
		}
	}

	return nil
}

func runOAuthLogin(provider string) error {
	switch strings.ToLower(provider) {
	case "openai":
		cfg := auth.OpenAIOAuthConfig()
		cred, err := auth.LoginBrowser(cfg)
		if err != nil {
			return fmt.Errorf("openai login failed: %w", err)
		}
		if err := auth.SetCredential("openai", cred); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		fmt.Println("OpenAI login successful!")
	case "anthropic":
		fmt.Println("Anthropic OAuth not available. Please run: picoclaw auth login --provider anthropic")
	case "google-antigravity", "antigravity":
		cfg := auth.GoogleAntigravityOAuthConfig()
		cred, err := auth.LoginBrowser(cfg)
		if err != nil {
			return fmt.Errorf("google-antigravity login failed: %w", err)
		}
		cred.Provider = "google-antigravity"
		if err := auth.SetCredential("google-antigravity", cred); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		fmt.Println("Google Antigravity login successful!")
	default:
		return fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
	return nil
}

func (s *Setup) AskMissing() error {
	return nil
}

func (s *Setup) RunNonInteractive() error {
	cfg := s.Cfg

	if _, err := os.Stat(s.ConfigPath); err == nil {
		fmt.Print("Config already exists. Overwrite and reset to default? (yes/no): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "yes" && answer != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
		cfg = config.DefaultConfig()
		s.Cfg = cfg
	}

	if cfg.Agents.Defaults.Workspace == "" {
		cfg.Agents.Defaults.Workspace = "~/.picoclaw/workspace"
	}

	workspace := cfg.WorkspacePath()

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("╔═══════════════════════════════════════════════════════╗")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("║              PicoClaw Setup Complete                  ║")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("╚═══════════════════════════════════════════════════════╝")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(successStyle.Render("✓ Workspace config created and is working!")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(sectionStyle.Render("📁 Workspace:")))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render(workspace)))
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(sectionStyle.Render("⚙  Config:")))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render(s.ConfigPath)))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(sectionStyle.Render("IMPORTANT COMMANDS")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(highlightStyle.Render("  picoclaw agent             - Start chatting")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  picoclaw onboard --interactive - Full setup")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  picoclaw auth login       - Login to providers")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  picoclaw status          - Show status")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(sectionStyle.Render("CHANNELS (in config.json)")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(highlightStyle.Render("  telegram   : Telegram bot")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  slack      : Slack")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  discord    : Discord")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  + more: whatsapp, feishu, dingtalk, line, qq, etc.")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(sectionStyle.Render("NEXT STEPS")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(highlightStyle.Render("  1. Add API key in config.json")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  2. Run 'picoclaw agent' to start!")))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  Run 'picoclaw onboard --interactive' for guided setup")))
	b.WriteString("\n")
	// b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	// b.WriteString("\n\n")

	fmt.Println(b.String())

	return nil
}

func saveConfigPath(path string, cfg *config.Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return config.SaveConfig(path, cfg)
}

type tuiModel struct {
	setup       *Setup
	state       string
	sessionIdx  int
	questionIdx int
	answers     map[string]string
	textInputs  map[string]textinput.Model
	selIdx      map[string]int
	registry    SessionRegistry
	errorMsg    string
}

func newTuiModel(s *Setup) tuiModel {
	return tuiModel{
		setup:      s,
		state:      "intro",
		answers:    make(map[string]string),
		textInputs: make(map[string]textinput.Model),
		selIdx:     make(map[string]int),
	}
}

func (t tuiModel) Init() tea.Cmd {
	return nil
}

func (t tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return t, tea.Quit
		case "esc":
			if t.state == "intro" || t.state == "done" {
				return t, tea.Quit
			}
			return t.prevQuestion()
		case "q":
			if t.state == "intro" || t.state == "done" {
				return t, tea.Quit
			}
		}

		switch t.state {
		case "intro":
			return t.handleIntro(msg)
		case "questions":
			return t.handleQuestions(msg)
		case "done":
			return t, tea.Quit
		}
	}
	return t, nil
}

func (t tuiModel) handleIntro(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		t.state = "questions"
		t.sessionIdx = 0
		t.questionIdx = 0
		t.errorMsg = ""
		t.registry = BuildSessionRegistry(t.setup.Cfg)
		t.initSession()
		return t, nil
	}
	return t, nil
}

func (t tuiModel) handleQuestions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	session := t.registry.Sessions[t.sessionIdx]
	visible := t.getVisibleQuestions(session)

	if len(visible) == 0 {
		return t, nil
	}

	if t.questionIdx >= len(visible) {
		t.questionIdx = len(visible) - 1
	}

	currentQ := visible[t.questionIdx]

	switch msg.String() {
	case "enter":
		// Save current answer
		t.saveAnswer(currentQ)
		t.errorMsg = ""

		// Ensure text inputs exist for newly visible questions
		t.ensureInputsExist()

		// Recalculate visible questions after saving (dependencies may change)
		session = t.registry.Sessions[t.sessionIdx]
		visible = t.getVisibleQuestions(session)

		// If not the last question, move to next question in same session
		if t.questionIdx < len(visible)-1 {
			t.questionIdx++
			t.focusCurrent()
			return t, nil
		}
		// If last question, try to proceed to next session
		return t.proceedToNextSession()

	case "tab":
		t.errorMsg = ""
		return t.nextQuestion()

	case "shift+tab":
		t.errorMsg = ""
		return t.prevQuestion()

	case "e":
		// Go back to edit from confirmation session
		if session.ID == "confirm" {
			t.sessionIdx = 0
			t.questionIdx = 0
			t.initSession()
			return t, nil
		}
	}

	// Handle input based on current question type
	switch currentQ.Type {
	case QuestionTypeText:
		// Ensure text input exists
		t.ensureTextInputExists(currentQ)

		if ti, ok := t.textInputs[currentQ.ID]; ok {
			var cmd tea.Cmd
			t.textInputs[currentQ.ID], cmd = ti.Update(msg)
			return t, cmd
		}

	case QuestionTypeSelect, QuestionTypeYesNo:
		switch msg.String() {
		case "up", "k":
			t.errorMsg = ""
			if t.selIdx[currentQ.ID] > 0 {
				t.selIdx[currentQ.ID]--
			}
		case "down", "j":
			t.errorMsg = ""
			if t.selIdx[currentQ.ID] < len(currentQ.Options)-1 {
				t.selIdx[currentQ.ID]++
			}
		}
	}

	return t, nil
}

func (t tuiModel) proceedToNextSession() (tea.Model, tea.Cmd) {
	session := t.registry.Sessions[t.sessionIdx]
	visible := t.getVisibleQuestions(session)

	// Save all answers first
	for _, q := range visible {
		t.saveAnswer(q)
	}

	// Validate all required fields are filled
	missing := t.validateSession(visible)
	if len(missing) > 0 {
		t.errorMsg = fmt.Sprintf("Please fill: %s", strings.Join(missing, ", "))
		// Move to first missing field
		for i, q := range visible {
			if t.isQuestionEmpty(q) {
				t.questionIdx = i
				t.focusCurrent()
				break
			}
		}
		return t, nil
	}

	t.errorMsg = ""

	if session.ID == "confirm" {
		confirmVal := t.answers["confirm"]
		if confirmVal == "yes" {
			t.applyAnswersToConfig()
			t.state = "done"
			return t, tea.Quit
		}
		t.state = "intro"
		t.sessionIdx = 0
		t.questionIdx = 0
		t.answers = make(map[string]string)
		t.textInputs = make(map[string]textinput.Model)
		t.selIdx = make(map[string]int)
		return t, nil
	}

	if t.sessionIdx < len(t.registry.Sessions)-1 {
		t.sessionIdx++
		t.questionIdx = 0
		t.initSession()
	}

	return t, nil
}

func (t tuiModel) nextQuestion() (tea.Model, tea.Cmd) {
	session := t.registry.Sessions[t.sessionIdx]
	visible := t.getVisibleQuestions(session)

	if t.questionIdx < len(visible)-1 {
		t.questionIdx++
		t.ensureInputsExist()
		t.focusCurrent()
	}
	return t, nil
}

func (t tuiModel) prevQuestion() (tea.Model, tea.Cmd) {
	if t.questionIdx > 0 {
		t.questionIdx--
		t.focusCurrent()
	} else if t.sessionIdx > 0 {
		// Go to previous session
		t.sessionIdx--
		// Get the last question of the previous session
		prevSession := t.registry.Sessions[t.sessionIdx]
		prevVisible := t.getVisibleQuestions(prevSession)
		if len(prevVisible) > 0 {
			t.questionIdx = len(prevVisible) - 1
		} else {
			t.questionIdx = 0
		}
		t.initSession()
	}
	return t, nil
}

func (t *tuiModel) initSession() {
	t.ensureInputsExist()
	t.focusCurrent()
}

func (t *tuiModel) ensureInputsExist() {
	session := t.registry.Sessions[t.sessionIdx]
	visible := t.getVisibleQuestions(session)

	for _, q := range visible {
		switch q.Type {
		case QuestionTypeText:
			t.ensureTextInputExists(q)
		case QuestionTypeSelect, QuestionTypeYesNo:
			if _, ok := t.selIdx[q.ID]; !ok {
				t.selIdx[q.ID] = t.getDefaultSelIdx(q)
			}
		}
	}
}

func (t *tuiModel) ensureTextInputExists(q Question) {
	if q.Type != QuestionTypeText {
		return
	}

	// Only create if doesn't exist
	if _, exists := t.textInputs[q.ID]; !exists {
		ti := textinput.New()

		// Determine placeholder based on question type
		switch q.ID {
		case "provider_api_key":
			// Get API key for the selected provider (from answers)
			if provider := t.answers["provider"]; provider != "" {
				ti.Placeholder = GetProviderAPIKey(t.setup.Cfg, provider)
			} else if q.DefaultValue != "" {
				ti.Placeholder = q.DefaultValue
			} else {
				ti.Placeholder = q.Info
			}
		case "provider_api_base":
			// Get API base for the selected provider (from answers)
			if provider := t.answers["provider"]; provider != "" {
				ti.Placeholder = GetProviderAPIBase(t.setup.Cfg, provider)
			} else if q.DefaultValue != "" {
				ti.Placeholder = q.DefaultValue
			} else {
				ti.Placeholder = q.Info
			}
		default:
			if q.DefaultValue != "" {
				ti.Placeholder = q.DefaultValue
			} else {
				ti.Placeholder = q.Info
			}
		}

		ti.CharLimit = 512
		ti.Width = 40

		if val, ok := t.answers[q.ID]; ok && val != "" {
			ti.SetValue(val)
		} else if q.DefaultValue != "" {
			ti.SetValue(q.DefaultValue)
		}
		t.textInputs[q.ID] = ti
	}
}

func (t *tuiModel) getDefaultSelIdx(q Question) int {
	defaultIdx := 0
	if val, ok := t.answers[q.ID]; ok && val != "" {
		for i, opt := range q.Options {
			if opt == val {
				defaultIdx = i
				break
			}
		}
	} else if q.DefaultValue != "" {
		for i, opt := range q.Options {
			if opt == q.DefaultValue {
				defaultIdx = i
				break
			}
		}
	}
	return defaultIdx
}

func (t *tuiModel) focusCurrent() {
	session := t.registry.Sessions[t.sessionIdx]
	visible := t.getVisibleQuestions(session)

	// Blur all text inputs
	for id := range t.textInputs {
		if ti, ok := t.textInputs[id]; ok {
			ti.Blur()
			t.textInputs[id] = ti
		}
	}

	// Focus current text input if applicable
	if t.questionIdx < len(visible) {
		q := visible[t.questionIdx]
		if q.Type == QuestionTypeText {
			t.ensureTextInputExists(q)
			if ti, ok := t.textInputs[q.ID]; ok {
				ti.Focus()
				t.textInputs[q.ID] = ti
			}
		}
	}
}

func (t *tuiModel) getVisibleQuestions(session Session) []Question {
	var visible []Question
	for _, q := range session.Questions {
		if t.isQuestionVisible(q) {
			visible = append(visible, q)
		}
	}
	return visible
}

func (t *tuiModel) isQuestionVisible(q Question) bool {
	if q.DependsOn == "" {
		return true
	}
	depValue := t.answers[q.DependsOn]
	if q.DependsValue != "" {
		return depValue == q.DependsValue
	}
	return depValue != ""
}

func (t *tuiModel) buildAnswersSummary() []string {
	var summary []string

	// Workspace
	if val := t.answers["workspace"]; val != "" {
		summary = append(summary, fmt.Sprintf("Workspace: %s", val))
	}
	if val := t.answers["restrict_workspace"]; val != "" {
		summary = append(summary, fmt.Sprintf("Restrict to workspace: %s", val))
	}

	// Provider
	if val := t.answers["provider"]; val != "" {
		summary = append(summary, fmt.Sprintf("Provider: %s", val))
	}
	if val := t.answers["provider_api_key"]; val != "" {
		summary = append(summary, fmt.Sprintf("API Key: %s", maskSensitive(val)))
	}

	// Model
	if val := t.answers["model_select"]; val != "" {
		if val == "custom" {
			if customVal := t.answers["custom_model"]; customVal != "" {
				summary = append(summary, fmt.Sprintf("Model: %s (custom)", customVal))
			}
		} else {
			summary = append(summary, fmt.Sprintf("Model: %s", val))
		}
	}

	// Channel
	if val := t.answers["channel_select"]; val != "" {
		summary = append(summary, fmt.Sprintf("Channel: %s", val))
	}
	if val := t.answers["channel_token"]; val != "" {
		summary = append(summary, fmt.Sprintf("Channel Token: %s", maskSensitive(val)))
	}

	return summary
}

func maskSensitive(val string) string {
	if len(val) <= 4 {
		return "****"
	}
	return val[:4] + strings.Repeat("*", len(val)-4)
}

func (t *tuiModel) validateSession(questions []Question) []string {
	var missing []string
	for _, q := range questions {
		// Skip optional fields
		if q.ID == "provider_api_base" {
			continue
		}
		// Skip provider_api_key - it's optional (can be added in config.json later)
		if q.ID == "provider_api_key" {
			continue
		}
		// Skip channel token fields - they're optional based on channel selection
		if isChannelTokenField(q.ID) {
			continue
		}
		// Skip channel_enable if set to "no"
		if q.ID == "channel_enable" && t.answers["channel_enable"] == "no" {
			continue
		}
		if t.isQuestionEmpty(q) {
			missing = append(missing, q.Prompt)
		}
	}
	return missing
}

func isChannelTokenField(id string) bool {
	channelFields := []string{
		"telegram_token", "slack_bot_token", "slack_app_token",
		"discord_token", "whatsapp_bridge_url",
		"feishu_app_id", "feishu_app_secret",
		"dingtalk_client_id", "dingtalk_client_secret",
		"line_channel_access_token", "qq_app_id",
		"onebot_access_token",
		"wecom_token", "wecom_encoding_aes_key",
		"wecom_app_corp_id", "wecom_app_agent_id",
		"maixcam_device_address",
	}
	for _, f := range channelFields {
		if id == f {
			return true
		}
	}
	return false
}

func (t *tuiModel) isQuestionEmpty(q Question) bool {
	switch q.Type {
	case QuestionTypeText:
		if ti, ok := t.textInputs[q.ID]; ok {
			return strings.TrimSpace(ti.Value()) == ""
		}
		return true
	case QuestionTypeSelect, QuestionTypeYesNo:
		return false
	}
	return false
}

func (t *tuiModel) saveAnswer(q Question) {
	switch q.Type {
	case QuestionTypeText:
		if ti, ok := t.textInputs[q.ID]; ok {
			t.answers[q.ID] = ti.Value()
		}
	case QuestionTypeSelect, QuestionTypeYesNo:
		if idx, ok := t.selIdx[q.ID]; ok && idx >= 0 && idx < len(q.Options) {
			t.answers[q.ID] = q.Options[idx]
		}
	}

	// Update channel question prompts when channel is selected
	if q.ID == "channel_select" {
		t.updateChannelQuestionPrompt()
	}

	// Update provider auth method options when provider is selected
	if q.ID == "provider" {
		t.updateProviderAuthOptions()
		t.updateModelOptions()
	}
}

func (t *tuiModel) updateChannelQuestionPrompt() {
	channel := t.answers["channel_select"]
	if channel == "" {
		return
	}

	ch := strings.ToLower(channel)
	channelFieldIDs := getChannelFieldIDs(ch)

	for i := range t.registry.Sessions {
		if t.registry.Sessions[i].ID == "channel" {
			for j := range t.registry.Sessions[i].Questions {
				q := &t.registry.Sessions[i].Questions[j]
				for _, fieldID := range channelFieldIDs {
					if q.ID == fieldID {
						q.DefaultValue = GetChannelFieldToken(t.setup.Cfg, fieldID)
						if ti, ok := t.textInputs[fieldID]; ok {
							ti.SetValue(q.DefaultValue)
							t.textInputs[fieldID] = ti
						}
					}
				}
			}
			break
		}
	}
}

func getChannelFieldIDs(channel string) []string {
	switch channel {
	case "telegram":
		return []string{"telegram_token"}
	case "slack":
		return []string{"slack_bot_token", "slack_app_token"}
	case "discord":
		return []string{"discord_token"}
	case "whatsapp":
		return []string{"whatsapp_bridge_url"}
	case "feishu":
		return []string{"feishu_app_id", "feishu_app_secret"}
	case "dingtalk":
		return []string{"dingtalk_client_id", "dingtalk_client_secret"}
	case "line":
		return []string{"line_channel_access_token"}
	case "qq":
		return []string{"qq_app_id"}
	case "onebot":
		return []string{"onebot_access_token"}
	case "wecom":
		return []string{"wecom_token", "wecom_encoding_aes_key"}
	case "wecom_app":
		return []string{"wecom_app_corp_id", "wecom_app_agent_id"}
	case "maixcam":
		return []string{"maixcam_device_address"}
	}
	return nil
}

func (t *tuiModel) updateProviderAuthOptions() {
	provider := t.answers["provider"]
	if provider == "" {
		return
	}

	for i := range t.registry.Sessions {
		if t.registry.Sessions[i].ID == "provider" {
			for j := range t.registry.Sessions[i].Questions {
				q := &t.registry.Sessions[i].Questions[j]
				if q.ID == "provider_auth_method" {
					if isOAuthProvider(provider) {
						q.Options = []string{"oauth_login", "api_key"}
					} else {
						q.Options = []string{"api_key"}
					}
					// Reset selection if out of bounds
					if idx, ok := t.selIdx[q.ID]; ok && idx >= len(q.Options) {
						t.selIdx[q.ID] = 0
					}
				}
			}
			break
		}
	}
}

func (t *tuiModel) updateModelOptions() {
	provider := t.answers["provider"]
	if provider == "" {
		return
	}

	var modelIDs []string
	if models := config.GetModelsForProvider(provider); len(models) > 0 {
		for _, m := range models {
			modelIDs = append(modelIDs, m.ID)
		}
	} else {
		modelIDs = config.GetPopularModels(provider)
	}
	modelIDs = append(modelIDs, "custom")

	for i := range t.registry.Sessions {
		if t.registry.Sessions[i].ID == "model" {
			for j := range t.registry.Sessions[i].Questions {
				q := &t.registry.Sessions[i].Questions[j]
				if q.ID == "model_select" {
					q.Options = modelIDs
					// Reset selection if out of bounds
					if idx, ok := t.selIdx[q.ID]; ok && idx >= len(q.Options) {
						t.selIdx[q.ID] = 0
					}
				}
			}
			break
		}
	}
}

func (t *tuiModel) applyAnswersToConfig() {
	cfg := t.setup.Cfg

	for qID, val := range t.answers {
		if val == "" {
			continue
		}

		switch qID {
		case "workspace":
			cfg.Agents.Defaults.Workspace = val
		case "restrict_workspace":
			cfg.Agents.Defaults.RestrictToWorkspace = (val == "yes")
		case "provider":
			cfg.Agents.Defaults.Provider = val
			// Set auth method if oauth_login was selected
			if t.answers["provider_auth_method"] == "oauth_login" {
				authMethod := "oauth"
				prov := strings.ToLower(val)
				switch prov {
				case "openai":
					cfg.Providers.OpenAI.AuthMethod = authMethod
				case "anthropic":
					cfg.Providers.Anthropic.AuthMethod = authMethod
				case "google-antigravity", "antigravity":
					cfg.Providers.Antigravity.AuthMethod = authMethod
				}
				// Also update model list
				for i := range cfg.ModelList {
					pfx := config.ParseProtocol(cfg.ModelList[i].Model)
					if pfx == "" {
						pfx = "openai"
					}
					if pfx == prov {
						cfg.ModelList[i].AuthMethod = authMethod
					}
				}
			}
		case "model_select":
			if val != "custom" {
				cfg.Agents.Defaults.Model = val
			}
		case "custom_model":
			if t.answers["model_select"] == "custom" {
				cfg.Agents.Defaults.Model = val
			}
		case "channel_select":
			if t.answers["channel_enable"] != "no" {
				t.applyChannelSelection(val, cfg)
			}
		case "channel_enable":
			if val == "no" {
				cfg.Channels.Telegram.Enabled = false
				cfg.Channels.Slack.Enabled = false
				cfg.Channels.Discord.Enabled = false
				cfg.Channels.WhatsApp.Enabled = false
				cfg.Channels.Feishu.Enabled = false
				cfg.Channels.DingTalk.Enabled = false
				cfg.Channels.LINE.Enabled = false
				cfg.Channels.QQ.Enabled = false
				cfg.Channels.OneBot.Enabled = false
				cfg.Channels.WeCom.Enabled = false
				cfg.Channels.WeComApp.Enabled = false
				cfg.Channels.MaixCam.Enabled = false
			}
		case "telegram_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.Telegram.Token = val
			}
		case "slack_bot_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.Slack.BotToken = val
			}
		case "slack_app_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.Slack.AppToken = val
			}
		case "discord_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.Discord.Token = val
			}
		case "whatsapp_bridge_url":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.WhatsApp.BridgeURL = val
			}
		case "feishu_app_id":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.Feishu.AppID = val
			}
		case "feishu_app_secret":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.Feishu.AppSecret = val
			}
		case "dingtalk_client_id":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.DingTalk.ClientID = val
			}
		case "dingtalk_client_secret":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.DingTalk.ClientSecret = val
			}
		case "line_channel_access_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.LINE.ChannelAccessToken = val
			}
		case "qq_app_id":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.QQ.AppID = val
			}
		case "onebot_access_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.OneBot.AccessToken = val
			}
		case "wecom_token":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.WeCom.Token = val
			}
		case "wecom_encoding_aes_key":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.WeCom.EncodingAESKey = val
			}
		case "wecom_app_corp_id":
			if t.answers["channel_enable"] != "no" {
				cfg.Channels.WeComApp.CorpID = val
			}
		case "wecom_app_agent_id":
			if t.answers["channel_enable"] != "no" {
				if agentID, err := strconv.ParseInt(val, 10, 64); err == nil {
					cfg.Channels.WeComApp.AgentID = agentID
				}
			}
		case "maixcam_device_address":
			if t.answers["channel_enable"] != "no" {
				val = strings.TrimPrefix(strings.TrimPrefix(val, "http://"), "https://")
				if idx := strings.Index(val, ":"); idx > 0 {
					cfg.Channels.MaixCam.Host = val[:idx]
					if port, err := strconv.Atoi(val[idx+1:]); err == nil {
						cfg.Channels.MaixCam.Port = port
					}
				} else {
					cfg.Channels.MaixCam.Host = val
				}
			}
		case "provider_api_key":
			prov := t.answers["provider"]
			if prov != "" && t.answers["provider_auth_method"] == "api_key" {
				SetProviderCredential(cfg, prov, "api_key", val)
			}
		case "provider_api_base":
			prov := t.answers["provider"]
			if prov != "" {
				SetProviderCredential(cfg, prov, "api_base", val)
			}
		case "provider_auth_method":
			prov := t.answers["provider"]
			if prov != "" && val == "oauth_login" {
				t.setup.OAuthProvider = prov
			}
		}
	}
}

func (t *tuiModel) applyChannelSelection(channel string, cfg *config.Config) {
	ch := strings.ToLower(channel)
	switch ch {
	case "telegram":
		cfg.Channels.Telegram.Enabled = true
	case "slack":
		cfg.Channels.Slack.Enabled = true
	case "discord":
		cfg.Channels.Discord.Enabled = true
	case "whatsapp":
		cfg.Channels.WhatsApp.Enabled = true
	case "feishu":
		cfg.Channels.Feishu.Enabled = true
	case "dingtalk":
		cfg.Channels.DingTalk.Enabled = true
	case "line":
		cfg.Channels.LINE.Enabled = true
	case "qq":
		cfg.Channels.QQ.Enabled = true
	case "onebot":
		cfg.Channels.OneBot.Enabled = true
	case "wecom":
		cfg.Channels.WeCom.Enabled = true
	case "wecom_app":
		cfg.Channels.WeComApp.Enabled = true
	case "maixcam":
		cfg.Channels.MaixCam.Enabled = true
	}
}

func (t tuiModel) View() string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	optionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("╔════════════════════════════════════════════════╗")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("║           PicoClaw Interactive Setup           ║")))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("╚════════════════════════════════════════════════╝")))
	b.WriteString("\n\n")

	switch t.state {
	case "intro":
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("Press Enter to start the configuration wizard")))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("Press q or Ctrl+C to exit")))

	case "questions":
		session := t.registry.Sessions[t.sessionIdx]
		visible := t.getVisibleQuestions(session)
		totalSessions := len(t.registry.Sessions)

		header := fmt.Sprintf("Step %d of %d: %s", t.sessionIdx+1, totalSessions, session.Title)
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render(header)))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
		b.WriteString("\n\n")

		if t.errorMsg != "" {
			b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(errorStyle.Render("⚠ " + t.errorMsg)))
			b.WriteString("\n\n")
		}

		for i, q := range visible {
			isActive := i == t.questionIdx

			prompt := q.Prompt
			if isActive {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(activeStyle.Render("▶ " + prompt)))
			} else {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(promptStyle.Render("  " + prompt)))
			}
			b.WriteString("\n")

			switch q.Type {
			case QuestionTypeText:
				t.ensureTextInputExists(q)
				if ti, ok := t.textInputs[q.ID]; ok {
					if isActive {
						b.WriteString(lipgloss.NewStyle().PaddingLeft(6).Render(ti.View()))
					} else {
						val := ti.Value()
						if val == "" {
							b.WriteString(lipgloss.NewStyle().PaddingLeft(6).Render(dimStyle.Render("(not filled)")))
						} else {
							b.WriteString(lipgloss.NewStyle().PaddingLeft(6).Render(optionStyle.Render(val)))
						}
					}
				}

			case QuestionTypeSelect, QuestionTypeYesNo:
				idx, ok := t.selIdx[q.ID]
				if !ok {
					idx = 0
				}
				for j, opt := range q.Options {
					var line string
					if j == idx {
						if isActive {
							line = "  ◆ " + opt
							b.WriteString(lipgloss.NewStyle().PaddingLeft(6).Render(selectedStyle.Render(line)))
						} else {
							line = "  ○ " + opt
							b.WriteString(lipgloss.NewStyle().PaddingLeft(6).Render(optionStyle.Render(line)))
						}
					} else {
						line = "    " + opt
						b.WriteString(lipgloss.NewStyle().PaddingLeft(6).Render(dimStyle.Render(line)))
					}
					b.WriteString("\n")
				}
			}
			b.WriteString("\n")
		}

		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("─────────────────────────────────────────────────")))
		b.WriteString("\n")

		// Show summary in confirmation session
		if session.ID == "confirm" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("Configuration Summary:")))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
			b.WriteString("\n\n")

			summary := t.buildAnswersSummary()
			for _, line := range summary {
				b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render(line)))
				b.WriteString("\n")
			}
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("Press 'e' to go back and edit, or confirm below:")))
			b.WriteString("\n\n")
		}

		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("─────────────────────────────────────────────────")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("↑/↓: Select option | Tab/Enter: Next | Esc: Back | e: Edit (in confirm) | q: Quit")))

	case "done":
		workspace := t.setup.Cfg.WorkspacePath()
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(successStyle.Render("✓ picoclaw is ready!")))
		b.WriteString("\n\n")
		for _, line := range BuildSummary(t.setup.Cfg) {
			b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render(line)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("Workspace Path:")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render(workspace)))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(titleStyle.Render("Workspace structure:")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  📁 memory/       - Persistent memory")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  📁 skills/       - Custom skills")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  📄 AGENT.md     - Agent configuration")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  📄 IDENTITY.md  - Agent identity")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  📄 SOUL.md      - Agent personality")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("  📄 USER.md      - User profile")))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("Next: Run 'picoclaw agent' to start chatting!")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("Config: ~/.picoclaw/config.json")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("─────────────────────────────────────────────────")))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(dimStyle.Render("Press Enter or q to exit")))
	}

	return b.String()
}
