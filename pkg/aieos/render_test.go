package aieos

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderToPromptFull(t *testing.T) {
	p := &Profile{
		Version: "1.1",
		Identity: Identity{
			Name:        "PicoClaw",
			Description: "Ultra-lightweight AI assistant",
			Purpose:     "Provide helpful AI assistance",
			Philosophy:  "Simplicity over complexity",
		},
		Capabilities: []Capability{
			{Name: "web_search", Description: "Web search and content fetching"},
			{Name: "file_ops"},
		},
		Psychology: &Psychology{
			Openness:          0.8,
			Conscientiousness: 0.9,
			Extraversion:      0.5,
			Agreeableness:     0.85,
			Neuroticism:       0.1,
		},
		Linguistics: &Linguistics{
			Formality: 0.5,
			Verbosity: 0.3,
		},
		Motivations: &Motivations{
			Values: []string{"accuracy", "transparency"},
			Goals:  []string{"fast_assistant"},
		},
		Boundaries: &Boundaries{
			HardLimits: []string{"no_harmful_content"},
			SoftLimits: []string{"prefer_user_control"},
		},
	}

	result := RenderToPrompt(p)

	assert.Contains(t, result, "You are **PicoClaw**.")
	assert.Contains(t, result, "Ultra-lightweight AI assistant")
	assert.Contains(t, result, "**Purpose**: Provide helpful AI assistance")
	assert.Contains(t, result, "**Philosophy**: Simplicity over complexity")
	assert.Contains(t, result, "## Personality")
	assert.Contains(t, result, "## Capabilities")
	assert.Contains(t, result, "- **web_search**: Web search and content fetching")
	assert.Contains(t, result, "- file_ops")
	assert.Contains(t, result, "## Communication Style")
	assert.Contains(t, result, "## Values & Goals")
	assert.Contains(t, result, "accuracy, transparency")
	assert.Contains(t, result, "fast_assistant")
	assert.Contains(t, result, "## Boundaries")
	assert.Contains(t, result, "no_harmful_content")
	assert.Contains(t, result, "prefer_user_control")
}

func TestRenderToPromptMinimal(t *testing.T) {
	p := &Profile{
		Version: "1.1",
		Identity: Identity{
			Name: "MinimalBot",
		},
	}

	result := RenderToPrompt(p)

	assert.Contains(t, result, "You are **MinimalBot**.")
	assert.NotContains(t, result, "## Personality")
	assert.NotContains(t, result, "## Capabilities")
	assert.NotContains(t, result, "## Communication Style")
	assert.NotContains(t, result, "## Values & Goals")
	assert.NotContains(t, result, "## Boundaries")
}

func TestRenderOCEAN(t *testing.T) {
	tests := []struct {
		name     string
		psy      *Psychology
		contains []string
	}{
		{
			name: "high values",
			psy: &Psychology{
				Openness: 0.9, Conscientiousness: 0.8,
				Extraversion: 0.7, Agreeableness: 0.75, Neuroticism: 0.8,
			},
			contains: []string{
				"curious and open",
				"organized, thorough",
				"enthusiastic, talkative",
				"warm, cooperative",
				"cautious and sensitive",
			},
		},
		{
			name: "low values",
			psy: &Psychology{
				Openness: 0.1, Conscientiousness: 0.2,
				Extraversion: 0.1, Agreeableness: 0.2, Neuroticism: 0.1,
			},
			contains: []string{
				"familiar approaches",
				"relaxed approach",
				"reserved",
				"direct and straightforward",
				"calm, stable",
			},
		},
		{
			name: "mid values",
			psy: &Psychology{
				Openness: 0.5, Conscientiousness: 0.5,
				Extraversion: 0.5, Agreeableness: 0.5, Neuroticism: 0.5,
			},
			contains: []string{
				"balanced approach to novelty",
				"balance thoroughness",
				"balance engagement",
				"balance helpfulness",
				"balanced emotional response",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := renderOCEAN(tc.psy)
			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestRenderLinguistics(t *testing.T) {
	tests := []struct {
		name     string
		ling     *Linguistics
		contains []string
	}{
		{
			name:     "formal and verbose",
			ling:     &Linguistics{Formality: 0.9, Verbosity: 0.8},
			contains: []string{"formal, professional", "thorough and detailed"},
		},
		{
			name:     "casual and concise",
			ling:     &Linguistics{Formality: 0.1, Verbosity: 0.1},
			contains: []string{"casual, friendly", "concise and to the point"},
		},
		{
			name:     "balanced with idiolect",
			ling:     &Linguistics{Formality: 0.5, Verbosity: 0.5, Idiolect: []string{"hey there", "gotcha"}},
			contains: []string{"balanced, conversational", "moderately detailed", "hey there, gotcha"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := renderLinguistics(tc.ling)
			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestRenderEmptyMotivationsAndBoundaries(t *testing.T) {
	p := &Profile{
		Version:     "1.1",
		Identity:    Identity{Name: "TestBot"},
		Motivations: &Motivations{},
		Boundaries:  &Boundaries{},
	}

	result := RenderToPrompt(p)

	assert.False(t, strings.Contains(result, "## Values & Goals"))
	assert.False(t, strings.Contains(result, "## Boundaries"))
}
