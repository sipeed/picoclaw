package config

import (
	"fmt"
	"strings"
)

type TurnProfileMode string

const (
	TurnProfileModeDefault TurnProfileMode = "default"
	TurnProfileModeOff     TurnProfileMode = "off"
	TurnProfileModeCustom  TurnProfileMode = "custom"
)

type TurnProfilesConfig map[string]TurnProfileConfig

type TurnProfileConfig struct {
	History      TurnProfileBlock `json:"history,omitempty"`
	SystemPrompt TurnProfileBlock `json:"system_prompt,omitempty"`
	Skills       TurnProfileBlock `json:"skills,omitempty"`
	Tools        TurnProfileBlock `json:"tools,omitempty"`
}

type TurnProfileBlock struct {
	Mode  TurnProfileMode `json:"mode,omitempty"`
	Allow []string        `json:"allow,omitempty"`
}

type EffectiveTurnProfile struct {
	Enabled          bool
	Name             string
	HistoryMode      TurnProfileMode
	SystemPromptMode TurnProfileMode
	SkillsMode       TurnProfileMode
	ToolsMode        TurnProfileMode
	AllowedSkills    []string
	AllowedTools     []string
}

func (m TurnProfileMode) Effective() TurnProfileMode {
	switch TurnProfileMode(strings.ToLower(strings.TrimSpace(string(m)))) {
	case "", TurnProfileModeDefault:
		return TurnProfileModeDefault
	case TurnProfileModeOff:
		return TurnProfileModeOff
	case TurnProfileModeCustom:
		return TurnProfileModeCustom
	default:
		return TurnProfileMode(strings.ToLower(strings.TrimSpace(string(m))))
	}
}

func (d *AgentDefaults) ResolveTurnProfile(name string) (EffectiveTurnProfile, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return EffectiveTurnProfile{}, false, nil
	}
	if d == nil {
		return EffectiveTurnProfile{}, false, fmt.Errorf("unknown turn profile %q", name)
	}
	if d.TurnProfiles == nil {
		return EffectiveTurnProfile{}, false, fmt.Errorf("unknown turn profile %q", name)
	}
	profile, ok := d.TurnProfiles[name]
	if !ok {
		return EffectiveTurnProfile{}, false, fmt.Errorf("unknown turn profile %q", name)
	}
	if err := validateTurnProfile(name, profile); err != nil {
		return EffectiveTurnProfile{}, false, err
	}
	return EffectiveTurnProfile{
		Enabled:          true,
		Name:             name,
		HistoryMode:      profile.History.Mode.Effective(),
		SystemPromptMode: profile.SystemPrompt.Mode.Effective(),
		SkillsMode:       profile.Skills.Mode.Effective(),
		ToolsMode:        profile.Tools.Mode.Effective(),
		AllowedSkills:    cleanStringList(profile.Skills.Allow),
		AllowedTools:     cleanStringList(profile.Tools.Allow),
	}, true, nil
}

func (c *Config) ValidateTurnProfiles() error {
	if c == nil {
		return nil
	}
	for name, profile := range c.Agents.Defaults.TurnProfiles {
		if err := validateTurnProfile(name, profile); err != nil {
			return err
		}
	}
	return nil
}

func validateTurnProfile(name string, profile TurnProfileConfig) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("turn profile name is required")
	}

	if err := validateTurnProfileBlock(name, "history", profile.History, false); err != nil {
		return err
	}
	if err := validateTurnProfileBlock(name, "system_prompt", profile.SystemPrompt, false); err != nil {
		return err
	}
	if err := validateTurnProfileBlock(name, "skills", profile.Skills, true); err != nil {
		return err
	}
	if err := validateTurnProfileBlock(name, "tools", profile.Tools, true); err != nil {
		return err
	}
	return nil
}

func validateTurnProfileBlock(name, field string, block TurnProfileBlock, allowCustom bool) error {
	mode := block.Mode.Effective()
	switch mode {
	case TurnProfileModeDefault, TurnProfileModeOff:
		return nil
	case TurnProfileModeCustom:
		if allowCustom {
			return nil
		}
		return fmt.Errorf("turn profile %q %s.mode custom is not supported in this version", name, field)
	default:
		return fmt.Errorf("turn profile %q %s.mode has unsupported mode %q", name, field, block.Mode)
	}
}

func cleanStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
