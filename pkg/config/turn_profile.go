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

type TurnProfileConfig struct {
	Enabled      bool             `json:"enabled"`
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

func (d *AgentDefaults) ResolveTurnProfile() (EffectiveTurnProfile, bool, error) {
	if d == nil {
		return EffectiveTurnProfile{}, false, nil
	}
	profile := d.TurnProfile
	if !profile.Enabled {
		return EffectiveTurnProfile{}, false, nil
	}
	if err := validateTurnProfile(profile); err != nil {
		return EffectiveTurnProfile{}, false, err
	}
	return EffectiveTurnProfile{
		Enabled:          true,
		HistoryMode:      profile.History.Mode.Effective(),
		SystemPromptMode: profile.SystemPrompt.Mode.Effective(),
		SkillsMode:       profile.Skills.Mode.Effective(),
		ToolsMode:        profile.Tools.Mode.Effective(),
		AllowedSkills:    cleanStringList(profile.Skills.Allow),
		AllowedTools:     cleanStringList(profile.Tools.Allow),
	}, true, nil
}

func (c *Config) ValidateTurnProfile() error {
	if c == nil {
		return nil
	}
	return validateTurnProfile(c.Agents.Defaults.TurnProfile)
}

func validateTurnProfile(profile TurnProfileConfig) error {
	if !profile.Enabled {
		return nil
	}
	if err := validateTurnProfileBlock("history", profile.History, false); err != nil {
		return err
	}
	if err := validateTurnProfileBlock("system_prompt", profile.SystemPrompt, false); err != nil {
		return err
	}
	if err := validateTurnProfileBlock("skills", profile.Skills, true); err != nil {
		return err
	}
	if err := validateTurnProfileBlock("tools", profile.Tools, true); err != nil {
		return err
	}
	return nil
}

func validateTurnProfileBlock(field string, block TurnProfileBlock, allowCustom bool) error {
	mode := block.Mode.Effective()
	switch mode {
	case TurnProfileModeDefault, TurnProfileModeOff:
		return nil
	case TurnProfileModeCustom:
		if allowCustom {
			return nil
		}
		return fmt.Errorf("turn_profile.%s.mode custom is not supported in this version", field)
	default:
		return fmt.Errorf("turn_profile.%s.mode has unsupported mode %q", field, block.Mode)
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
