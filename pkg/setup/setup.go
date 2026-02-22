package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sipeed/picoclaw/pkg/config"
)

var Banner = `
*******************************************************************

  ██████╗ ██╗ ██████╗ ██████╗  ██████╗██╗      █████╗ ██╗    ██╗
  ██╔══██╗██║██╔════╝██╔═══██╗██╔════╝██║     ██╔══██╗██║    ██║
  ██████╔╝██║██║     ██║   ██║██║     ██║     ███████║██║ █╗ ██║
  ██╔═══╝ ██║██║     ██║   ██║██║     ██║     ██╔══██║██║███╗██║
  ██║     ██║╚██████╗╚██████╔╝╚██████╗███████╗██║  ██║╚███╔███╔╝
  ╚═╝     ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ 

*******************************************************************
`

var About = `Tiny, Fast, and Deployable anywhere — automate the mundane, unleash your creativity`

// Setup holds state for the interactive setup flow.
type Setup struct {
	ConfigPath string
	Cfg        *config.Config
	Steps      []string
}

// NewSetup creates a Setup, loading existing config if present or using defaults.
func NewSetup(configPath string) (*Setup, error) {
	s := &Setup{ConfigPath: configPath}

	// Try load existing config, otherwise use defaults
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

// BuildDefs creates the per-step tui definitions based on the current config.
// It only includes steps that are relevant (e.g., missing values) but always
// exposes the provider selector so users can change it.
func (s *Setup) BuildDefs() []stepDef {
	defs := []stepDef{}

	// Workspace: always allow the user to set/change workspace
	defs = append(defs, stepDef{id: "workspace", kind: "text", prompt: "1. Workspace path (e.g. ~/.picoclaw/workspace)", info: "always allow the user to set/change workspace"})

	// Provider: allow selection (always present so user can change)
	defs = append(defs, stepDef{id: "provider", kind: "select", prompt: "2. Choose default provider", options: []string{
		"openai", "openrouter", "anthropic", "ollama", "volcengine", "github_copilot", "groq", "gemini", "zhipu", "deepseek", "antigravity", "qwen",
	}})

	// Provider-specific API key prompts (only add when the provider is openai or when key is missing)
	// For now we only prompt for OpenAI API key if empty.
	if s.Cfg.Providers.OpenAI.APIKey == "" {
		defs = append(defs, stepDef{id: "openai_api_key", kind: "text", prompt: "3. Provider API key (leave empty to skip)"})
	}

	// Default model
	if s.Cfg.Agents.Defaults.Model == "" {
		defs = append(defs, stepDef{id: "default_model", kind: "text", prompt: "4. Default model name (e.g. gpt-5.2)"})
	}

	// Web search option for OpenAI (only relevant if provider is openai but safe to show)
	defs = append(defs, stepDef{id: "provider", kind: "select", prompt: "2. Choose channel", options: []string{
		"telegram", "whatsapp", "discord", "feishu", "maxicam", "qq", "dingtalk", "slack", "line", "onebot", "wecom",
	}})
	// Install example skills (optional)
	defs = append(defs, stepDef{id: "install_skills", kind: "yesno", prompt: "6. Install example skills now?", options: []string{"yes", "no"}})

	return defs
}

// buildSteps inspects the config and creates a list of remaining steps for the user.
func (s *Setup) buildSteps() {
	s.Steps = []string{}
	// If providers section is empty, prompt to add API keys
	// if s.Cfg.Providers.IsEmpty() {
	// 	s.Steps = append(s.Steps, "Choose workspace location")
	// }

	// Check model list for missing API keys
	missing := 0
	for _, m := range s.Cfg.ModelList {
		if m.APIKey == "" {
			missing++
		}
	}
	if missing > 0 {
		s.Steps = append(s.Steps, fmt.Sprintf("Add API keys for %d model(s) in model_list", missing))
	}

	// Workspace
	if s.Cfg.Agents.Defaults.Workspace == "" {
		s.Steps = append(s.Steps, "Configure workspace path")
	}
}

// Run shows a minimal Bubble Tea UI listing steps and waits for user to continue.
func (s *Setup) Run() error {
	// Create the TUI model which owns a pointer to this Setup so it can
	// update s.Cfg while the user answers questions.
	m := tuiModel{setup: s, state: "intro"}

	p := tea.NewProgram(m)
	if err := p.Start(); err != nil {
		return fmt.Errorf("tui start failed: %w", err)
	}

	// On exit, save the potentially-updated config back to disk.
	if err := saveConfigPath(s.ConfigPath, s.Cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// AskMissing is deprecated when using full TUI; keep for backward compatibility.
func (s *Setup) AskMissing() error { return nil }

func saveConfigPath(path string, cfg *config.Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return config.SaveConfig(path, cfg)
}

// trimNewline kept for potential future use
// func trimNewline(s string) string {
// 	if len(s) == 0 {
// 		return s
// 	}
// 	if s[len(s)-1] == '\n' {
// 		s = s[:len(s)-1]
// 	}
// 	if len(s) > 0 && s[len(s)-1] == '\r' {
// 		s = s[:len(s)-1]
// 	}
// 	return s
// }

// --- TUI model ---
type tuiModel struct {
	setup  *Setup
	state  string // "intro", "questions", "done"
	cursor int
	// per-step definitions
	defs []stepDef
	// active text input (if current step is text)
	ti textinput.Model
	// selection index for select/yesno steps
	selIdx int
}

type stepDef struct {
	kind    string // "text", "select", "yesno"
	prompt  string
	info    string
	options []string // for select/yesno
	id      string   // logical identifier for mapping answers into config
}

func (t tuiModel) Init() tea.Cmd { return nil }

func (t tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Always allow immediate quit for sensitive interrupts
		if msg.String() == "esc" || msg.String() == "ctrl+c" {
			return t, tea.Quit
		}

		// If we're in questions state and the current step is text, let the textinput
		// handle keystrokes first. In that state we must not treat `q` as a global
		// quit key because the user may be typing the letter 'q' into the input.
		if t.state == "questions" && len(t.defs) > 0 && t.defs[t.cursor].kind == "text" {
			var cmd tea.Cmd
			ti, cmd := t.ti.Update(msg)
			t.ti = ti
			// propagate any command from textinput
			if cmd != nil {
				return t, cmd
			}
			// After textinput processed the key, do not interpret `q` as quit here.
			// Continue to next message handling (e.g. Enter) below.
		}

		// quit key 'q' should work when not typing into a text input (e.g., intro or select)
		if msg.String() == "q" {
			return t, tea.Quit
		}

		// handle other keys (enter, left/right) below
		switch msg.String() {
		case "enter":
			if t.state == "intro" {
				// Build defs dynamically from the Setup
				t.defs = t.setup.BuildDefs()

				// initialize first input if text
				t.cursor = 0
				if t.defs[0].kind == "text" {
					ti := textinput.New()
					ti.Placeholder = t.defs[0].prompt
					ti.CharLimit = 512
					// prefill from config when we know mapping; try workspace first
					if t.setup.Cfg.Agents.Defaults.Workspace != "" {
						ti.SetValue(t.setup.Cfg.Agents.Defaults.Workspace)
					}
					ti.Focus()
					t.ti = ti
				}
				// initialize selection index from existing config when applicable
				t.selIdx = 0
				// if provider already configured, set selIdx accordingly
				// find first select/yesno and prefill from config
				for _, def := range t.defs {
					if def.kind == "select" {
						for i, o := range def.options {
							if o == t.setup.Cfg.Agents.Defaults.Provider {
								t.selIdx = i
								break
							}
						}
						break
					}
				}
				t.state = "questions"
				return t, nil
			}
			if t.state == "questions" {
				// commit current answer and advance; when past last, finish
				idx := t.cursor
				def := t.defs[idx]
				switch def.kind {
				case "text":
					ans := t.ti.Value()
					// Store mapped answers by matching prompt where possible.
					// This avoids relying on fixed indices since defs are now dynamic.
					// Prefer id mapping when available
					switch def.id {
					case "workspace":
						t.setup.Cfg.Agents.Defaults.Workspace = ans
					case "openai_api_key":
						t.setup.Cfg.Providers.OpenAI.APIKey = ans
					case "default_model":
						t.setup.Cfg.Agents.Defaults.Model = ans
					default:
						// fallback to prompt matching for backward compatibility
						switch def.prompt {
						case "Workspace path (e.g. ~/.picoclaw/workspace)":
							t.setup.Cfg.Agents.Defaults.Workspace = ans
						case "OpenAI API key (leave empty to skip)":
							t.setup.Cfg.Providers.OpenAI.APIKey = ans
						case "Default model name (e.g. gpt-5.2)":
							t.setup.Cfg.Agents.Defaults.Model = ans
						}
					}
				case "yesno":
					// map yes/no to booleans
					yes := t.selIdx == 0
					// map by prompt instead of index
					if def.prompt == "Enable OpenAI web search?" {
						t.setup.Cfg.Providers.OpenAI.WebSearch = yes
					}
					// install-skills prompt left as a TODO action
				case "select":
					// set selected provider
					sel := def.options[t.selIdx]
					// prefer id-based mapping
					if def.id == "provider" {
						t.setup.Cfg.Agents.Defaults.Provider = sel
					} else {
						t.setup.Cfg.Agents.Defaults.Provider = sel
					}
				}

				if t.cursor < len(t.defs)-1 {
					t.cursor++
					// prepare next input
					if t.defs[t.cursor].kind == "text" {
						ti := textinput.New()
						ti.Placeholder = t.defs[t.cursor].prompt
						ti.CharLimit = 512
						// prefill from config by inspecting the prompt
						// prefill using id when present, otherwise prompt match
						switch t.defs[t.cursor].id {
						case "workspace":
							ti.SetValue(t.setup.Cfg.Agents.Defaults.Workspace)
						case "openai_api_key":
							ti.SetValue(t.setup.Cfg.Providers.OpenAI.APIKey)
						case "default_model":
							ti.SetValue(t.setup.Cfg.Agents.Defaults.Model)
						default:
							switch t.defs[t.cursor].prompt {
							case "Workspace path (e.g. ~/.picoclaw/workspace)":
								ti.SetValue(t.setup.Cfg.Agents.Defaults.Workspace)
							case "OpenAI API key (leave empty to skip)":
								ti.SetValue(t.setup.Cfg.Providers.OpenAI.APIKey)
							case "Default model name (e.g. gpt-5.2)":
								ti.SetValue(t.setup.Cfg.Agents.Defaults.Model)
							}
						}
						ti.Focus()
						t.ti = ti
					} else {
						t.ti.Blur()
						// set selection index from existing config if available
						if t.defs[t.cursor].kind == "select" {
							for i, o := range t.defs[t.cursor].options {
								if o == t.setup.Cfg.Agents.Defaults.Provider {
									t.selIdx = i
									break
								}
							}
						} else if t.defs[t.cursor].kind == "yesno" {
							// try to map known yesno prompts to config
							if t.defs[t.cursor].prompt == "Enable OpenAI web search?" {
								if t.setup.Cfg.Providers.OpenAI.WebSearch {
									t.selIdx = 0
								} else {
									t.selIdx = 1
								}
							} else {
								t.selIdx = 0
							}
						} else {
							t.selIdx = 0
						}
					}
					return t, nil
				}
				// last step committed
				t.state = "done"
				return t, tea.Quit
			}
		case "left":
			// For select/yesno steps, move selection left
			if t.state == "questions" && len(t.defs) > 0 {
				k := t.defs[t.cursor].kind
				if (k == "yesno" || k == "select") && t.selIdx > 0 {
					t.selIdx--
				}
			}
		case "right":
			// For select/yesno steps, move selection right
			if t.state == "questions" && len(t.defs) > 0 {
				k := t.defs[t.cursor].kind
				if k == "yesno" || k == "select" {
					if t.selIdx < len(t.defs[t.cursor].options)-1 {
						t.selIdx++
					}
				}
			}
		case "up":
			// allow up to also move selection up for select/yesno
			if t.state == "questions" && len(t.defs) > 0 {
				k := t.defs[t.cursor].kind
				if (k == "yesno" || k == "select") && t.selIdx > 0 {
					t.selIdx--
				}
			}
		case "down":
			// allow down to move selection down for select/yesno
			if t.state == "questions" && len(t.defs) > 0 {
				k := t.defs[t.cursor].kind
				if k == "yesno" || k == "select" {
					if t.selIdx < len(t.defs[t.cursor].options)-1 {
						t.selIdx++
					}
				}
			}
		}
	}
	return t, nil
}

func (t tuiModel) View() string {
	// Styles
	// Keep styles simple and avoid lipgloss alignment/layout, which can
	// cause flex-like behavior. We'll control left offset manually.
	bannerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	aboutStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	stepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

	out := ""
	// render banner and about with a small left offset
	out += lipgloss.NewStyle().PaddingLeft(1).Render(bannerStyle.Render(Banner)) + "\n"
	out += lipgloss.NewStyle().PaddingLeft(1).Render(aboutStyle.Render(About)) + "\n\n"

	// Show slideshow style: only current step prompt and progress
	if t.state == "questions" && len(t.defs) > 0 {
		total := len(t.defs)
		out += stepStyle.Render(fmt.Sprintf("Step [%d/%d]\n", t.cursor+1, total))
		// separator between step header and prompt
		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", 0))
		out += sep + "\n"
		cur := t.defs[t.cursor]
		promptStyle := lipgloss.NewStyle().Bold(true)
		out += promptStyle.Render(cur.prompt) + "\n\n"
		// optional informational label (may be empty)
		if cur.info != "" {
			// Label style: bold, no left margin
			// labelStyle := lipgloss.NewStyle().Bold(true)
			// Info style: secondary light color, small padding to separate from label
			infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).PaddingLeft(0)
			out += infoStyle.Render(cur.info) + "\n\n"
		}
		// show the input or selection
		if cur.kind == "text" {
			out += t.ti.View() + "\n"
		} else if cur.kind == "yesno" || cur.kind == "select" {
			for i, opt := range cur.options {
				if i == t.selIdx {
					out += selectedStyle.Render("→ "+opt) + "\n"
				} else {
					out += stepStyle.Render("  "+opt) + "\n"
				}
			}
		}
		out += "\n"
	} else {
		out += stepStyle.Render("Press Enter to begin the interactive setup") + "\n\n"
	}

	switch t.state {
	case "intro":
		out += lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true).Render("Press Enter to start interactive setup — q to quit")
	case "questions":
		out += lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Use Up/Down to move between steps; type and Enter to submit; q to quit.") + "\n\n"
		// (current step rendering done above)
	case "done":
		out += lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("Setup complete — saving config and exiting...")
	}

	// Apply global left padding so the whole UI shifts right by 3 columns.
	return lipgloss.NewStyle().PaddingLeft(3).Render(out)
}

// Where to add questions/forms/selectors:
// - Implement the interactive questions inside the Update/View branches for state == "questions".
// - Use the charmbracelet/bubbles components (textinput, list, radiobuttons, checkbox) to build forms.
// - Example plan:
//   1) Create a field list (or pages) for missing items: models without API keys, providers, workspace path.
//   2) For each missing item, show a text input (bubbles/textinput) and let the user type and submit.
//   3) Store answers directly in t.setup.Cfg (it's a pointer) so they persist.
//   4) After all fields are answered, transition to state "done" which will cause Run() to save the config.
