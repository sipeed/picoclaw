package aieos

import (
	"fmt"
	"strings"
)

// RenderToPrompt converts an AIEOS profile into natural-language text
// suitable for injection into a system prompt.
func RenderToPrompt(p *Profile) string {
	var sb strings.Builder

	// Identity section
	sb.WriteString("## Identity\n\n")
	fmt.Fprintf(&sb, "You are **%s**.", p.Identity.Name)
	if p.Identity.Description != "" {
		sb.WriteString(" " + p.Identity.Description + ".")
	}
	sb.WriteString("\n")
	if p.Identity.Purpose != "" {
		fmt.Fprintf(&sb, "\n**Purpose**: %s\n", p.Identity.Purpose)
	}
	if p.Identity.Philosophy != "" {
		fmt.Fprintf(&sb, "\n**Philosophy**: %s\n", p.Identity.Philosophy)
	}

	// Psychology (OCEAN traits)
	if p.Psychology != nil {
		sb.WriteString("\n## Personality\n\n")
		sb.WriteString(renderOCEAN(p.Psychology))
	}

	// Capabilities
	if len(p.Capabilities) > 0 {
		sb.WriteString("\n## Capabilities\n\n")
		for _, c := range p.Capabilities {
			if c.Description != "" {
				fmt.Fprintf(&sb, "- **%s**: %s\n", c.Name, c.Description)
			} else {
				fmt.Fprintf(&sb, "- %s\n", c.Name)
			}
		}
	}

	// Linguistics
	if p.Linguistics != nil {
		sb.WriteString("\n## Communication Style\n\n")
		sb.WriteString(renderLinguistics(p.Linguistics))
	}

	// Motivations
	if p.Motivations != nil {
		hasContent := len(p.Motivations.Values) > 0 || len(p.Motivations.Goals) > 0
		if hasContent {
			sb.WriteString("\n## Values & Goals\n\n")
			if len(p.Motivations.Values) > 0 {
				sb.WriteString("**Core values**: " + strings.Join(p.Motivations.Values, ", ") + "\n")
			}
			if len(p.Motivations.Goals) > 0 {
				sb.WriteString("**Goals**: " + strings.Join(p.Motivations.Goals, ", ") + "\n")
			}
		}
	}

	// Boundaries
	if p.Boundaries != nil {
		hasContent := len(p.Boundaries.HardLimits) > 0 || len(p.Boundaries.SoftLimits) > 0
		if hasContent {
			sb.WriteString("\n## Boundaries\n\n")
			if len(p.Boundaries.HardLimits) > 0 {
				sb.WriteString("**Hard limits** (never violate):\n")
				for _, l := range p.Boundaries.HardLimits {
					fmt.Fprintf(&sb, "- %s\n", l)
				}
			}
			if len(p.Boundaries.SoftLimits) > 0 {
				sb.WriteString("**Soft limits** (prefer to follow):\n")
				for _, l := range p.Boundaries.SoftLimits {
					fmt.Fprintf(&sb, "- %s\n", l)
				}
			}
		}
	}

	return sb.String()
}

// renderOCEAN converts OCEAN trait values (0.0-1.0) into natural language descriptions.
func renderOCEAN(psy *Psychology) string {
	var lines []string

	lines = append(lines, traitDescription(psy.Openness,
		"You are highly curious and open to new ideas.",
		"You have a balanced approach to novelty and tradition.",
		"You prefer familiar approaches and proven methods.",
	))
	lines = append(lines, traitDescription(psy.Conscientiousness,
		"You are highly organized, thorough, and detail-oriented.",
		"You balance thoroughness with flexibility.",
		"You take a relaxed approach to planning and organization.",
	))
	lines = append(lines, traitDescription(psy.Extraversion,
		"You are enthusiastic, talkative, and energetic in interactions.",
		"You balance engagement with thoughtful reflection.",
		"You are reserved and prefer concise, focused communication.",
	))
	lines = append(lines, traitDescription(psy.Agreeableness,
		"You are warm, cooperative, and prioritize the user's needs.",
		"You balance helpfulness with honest feedback.",
		"You are direct and straightforward, even when it may be uncomfortable.",
	))
	lines = append(lines, traitDescription(psy.Neuroticism,
		"You tend to be cautious and sensitive to potential issues.",
		"You have a balanced emotional response to challenges.",
		"You are calm, stable, and resilient under pressure.",
	))

	return strings.Join(lines, "\n") + "\n"
}

// traitDescription maps a 0.0-1.0 value to one of three descriptions: high (>=0.7), mid (0.3-0.7), low (<0.3).
func traitDescription(value float64, high, mid, low string) string {
	switch {
	case value >= 0.7:
		return high
	case value >= 0.3:
		return mid
	default:
		return low
	}
}

// renderLinguistics converts formality/verbosity into communication instructions.
func renderLinguistics(ling *Linguistics) string {
	var lines []string

	switch {
	case ling.Formality >= 0.7:
		lines = append(lines, "Use formal, professional language.")
	case ling.Formality >= 0.3:
		lines = append(lines, "Use a balanced, conversational but professional tone.")
	default:
		lines = append(lines, "Use casual, friendly language.")
	}

	switch {
	case ling.Verbosity >= 0.7:
		lines = append(lines, "Be thorough and detailed in your responses.")
	case ling.Verbosity >= 0.3:
		lines = append(lines, "Be moderately detailed — explain when helpful, be concise when possible.")
	default:
		lines = append(lines, "Be concise and to the point. Avoid unnecessary elaboration.")
	}

	if len(ling.Idiolect) > 0 {
		lines = append(
			lines,
			fmt.Sprintf(
				"Use these characteristic phrases when appropriate: %s.",
				strings.Join(ling.Idiolect, ", "),
			),
		)
	}

	return strings.Join(lines, "\n") + "\n"
}
