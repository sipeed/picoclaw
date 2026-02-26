package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
)

// SkillRecommendation represents a recommended skill with metadata.
type SkillRecommendation struct {
	Name        string  `json:"name"`        // Skill name
	Score       float64 `json:"score"`       // Recommendation score (0-100)
	Reason      string  `json:"reason"`      // Why this skill is recommended
	Confidence  float64 `json:"confidence"`  // Confidence level (0-1)
	ChannelType string  `json:"channelType"` // Channel type that triggered this recommendation
}

// SkillRecommender provides context-based skill recommendations using LLM reasoning.
type SkillRecommender struct {
	skillsLoader *skills.SkillsLoader
	llmProvider  providers.LLMProvider
	model        string

	// Weights for scoring algorithm
	channelWeight float64 // Weight for channel-based matching (default: 0.4)
	keywordWeight float64 // Weight for keyword matching (default: 0.3)
	historyWeight float64 // Weight for historical usage (default: 0.2)
	recencyWeight float64 // Weight for recency (default: 0.1)
}

// NewSkillRecommender creates a new skill recommender.
func NewSkillRecommender(
	skillsLoader *skills.SkillsLoader,
	llmProvider providers.LLMProvider,
	model string,
) *SkillRecommender {
	return &SkillRecommender{
		skillsLoader:  skillsLoader,
		llmProvider:   llmProvider,
		model:         model,
		channelWeight: 0.4,
		keywordWeight: 0.3,
		historyWeight: 0.2,
		recencyWeight: 0.1,
	}
}

// SetWeights sets the weights for the scoring algorithm.
// All weights should be between 0 and 1, and ideally sum to 1.0.
func (sr *SkillRecommender) SetWeights(channel, keyword, history, recency float64) {
	sr.channelWeight = channel
	sr.keywordWeight = keyword
	sr.historyWeight = history
	sr.recencyWeight = recency
}

// RecommendSkillsForContext recommends skills based on the current context.
// It uses a hybrid approach:
// 1. Rule-based pre-filtering (channel type, keywords)
// 2. LLM-based reasoning for final selection
// 3. Scoring based on multiple factors
//
// Parameters:
//   - channel: Channel type (e.g., "telegram", "wecom", "slack")
//   - chatID: Chat identifier
//   - userMessage: Current user message
//   - history: Recent conversation history
//
// Returns:
//   - List of recommended skills with scores and reasons
func (sr *SkillRecommender) RecommendSkillsForContext(
	channel, chatID, userMessage string,
	history []providers.Message,
) ([]SkillRecommendation, error) {
	startTime := time.Now()

	// Get all available skills
	allSkills := sr.skillsLoader.ListSkills()
	if len(allSkills) == 0 {
		logger.DebugCF("agent", "No skills available for recommendation", nil)
		return []SkillRecommendation{}, nil
	}

	// Step 1: Rule-based pre-filtering and scoring
	preFilteredSkills := sr.preFilterAndScore(allSkills, channel, userMessage, history)

	if len(preFilteredSkills) == 0 {
		logger.DebugCF("agent", "No skills passed pre-filtering", nil)
		return []SkillRecommendation{}, nil
	}

	// Step 2: Use LLM for intelligent selection if we have multiple candidates
	var finalRecommendations []SkillRecommendation
	if len(preFilteredSkills) > 1 {
		var err error
		finalRecommendations, err = sr.llmBasedSelection(preFilteredSkills, channel, userMessage, history)
		if err != nil {
			logger.ErrorCF("agent", "LLM-based selection failed, using pre-filtered results",
				map[string]any{"error": err.Error()})
			finalRecommendations = preFilteredSkills
		}
	} else {
		finalRecommendations = preFilteredSkills
	}

	// Sort by score descending
	sortRecommendationsByScore(finalRecommendations)

	duration := time.Since(startTime)
	logger.DebugCF("agent", "Skill recommendation completed",
		map[string]any{
			"duration_ms":    duration.Milliseconds(),
			"total_skills":   len(allSkills),
			"pre_filtered":   len(preFilteredSkills),
			"final_selected": len(finalRecommendations),
			"channel":        channel,
		})

	return finalRecommendations, nil
}

// preFilterAndScore performs rule-based pre-filtering and initial scoring.
func (sr *SkillRecommender) preFilterAndScore(
	allSkills []skills.SkillInfo,
	channel, userMessage string,
	history []providers.Message,
) []SkillRecommendation {
	recommendations := make([]SkillRecommendation, 0)

	for _, skill := range allSkills {
		score := 0.0
		reasons := make([]string, 0)

		// Channel-based scoring (40%)
		channelScore := sr.scoreByChannel(skill, channel)
		if channelScore > 0 {
			score += channelScore * sr.channelWeight
			reasons = append(reasons, fmt.Sprintf("channel match (%s)", channel))
		}

		// Keyword-based scoring (30%)
		keywordScore := sr.scoreByKeywords(skill, userMessage)
		if keywordScore > 0 {
			score += keywordScore * sr.keywordWeight
			reasons = append(reasons, "keyword match")
		}

		// History-based scoring (20%)
		historyScore := sr.scoreByHistory(skill, history)
		if historyScore > 0 {
			score += historyScore * sr.historyWeight
			reasons = append(reasons, "used in history")
		}

		// Recency scoring (10%)
		recencyScore := sr.scoreByRecency(skill, history)
		if recencyScore > 0 {
			score += recencyScore * sr.recencyWeight
			reasons = append(reasons, "recently used")
		}

		// Only include skills with positive score
		if score > 0 {
			recommendations = append(recommendations, SkillRecommendation{
				Name:        skill.Name,
				Score:       score * 100, // Normalize to 0-100
				Reason:      strings.Join(reasons, ", "),
				Confidence:  0.5, // Base confidence before LLM refinement
				ChannelType: channel,
			})
		}
	}

	return recommendations
}

// scoreByChannel scores a skill based on channel type compatibility.
func (sr *SkillRecommender) scoreByChannel(skill skills.SkillInfo, channel string) float64 {
	// Check if skill description or name mentions channel-specific keywords
	channelKeywords := map[string][]string{
		"telegram": {"telegram", "sticker", "poll", "inline"},
		"wecom":    {"wecom", "wechat", "企业微信", "approval", "meeting"},
		"slack":    {"slack", "huddle", "workflow", "emoji"},
		"discord":  {"discord", "server", "voice", "reaction"},
		"feishu":   {"feishu", "lark", "飞书", "doc", "bitable"},
		"dingtalk": {"dingtalk", "钉钉", "ding", "approval"},
	}

	keywords, exists := channelKeywords[channel]
	if !exists {
		return 0.3 // Default score for unknown channels
	}

	skillText := strings.ToLower(skill.Name + " " + skill.Description)
	for _, keyword := range keywords {
		if strings.Contains(skillText, strings.ToLower(keyword)) {
			return 1.0 // Perfect match
		}
	}

	return 0.0
}

// scoreByKeywords scores a skill based on user message keywords.
func (sr *SkillRecommender) scoreByKeywords(skill skills.SkillInfo, userMessage string) float64 {
	messageLower := strings.ToLower(userMessage)

	// Extract important keywords from skill description
	importantKeywords := extractKeywords(skill.Description)

	matchCount := 0
	for _, keyword := range importantKeywords {
		if strings.Contains(messageLower, strings.ToLower(keyword)) {
			matchCount++
		}
	}

	if matchCount == 0 {
		return 0.0
	}

	// Score based on keyword match ratio
	score := float64(matchCount) / float64(len(importantKeywords))
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// scoreByHistory scores a skill based on historical usage.
func (sr *SkillRecommender) scoreByHistory(skill skills.SkillInfo, history []providers.Message) float64 {
	// Check if this skill was used in recent history
	for _, msg := range history {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.Function != nil && toolCall.Function.Name == skill.Name {
					return 0.8 // High score for recently used skill
				}
			}
		}
	}
	return 0.0
}

// scoreByRecency scores a skill based on how recently it was used.
func (sr *SkillRecommender) scoreByRecency(skill skills.SkillInfo, history []providers.Message) float64 {
	// Simple recency: more recent = higher score
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.Function != nil && toolCall.Function.Name == skill.Name {
					// Linear decay based on position
					recency := float64(len(history)-i) / float64(len(history))
					return 1.0 - recency
				}
			}
		}
	}
	return 0.0
}

// llmBasedSelection uses LLM to make intelligent skill selections.
func (sr *SkillRecommender) llmBasedSelection(
	candidates []SkillRecommendation,
	channel, userMessage string,
	history []providers.Message,
) ([]SkillRecommendation, error) {
	// Build prompt for LLM
	prompt := sr.buildLLMPrompt(candidates, channel, userMessage, history)

	// Create messages for LLM
	messages := []providers.Message{
		{
			Role:    "system",
			Content: "You are a skill recommendation assistant. Analyze the context and select the most appropriate skills.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call LLM
	ctx := context.Background()
	response, err := sr.llmProvider.Chat(ctx, messages, nil, sr.model, map[string]any{
		"temperature": 0.3,
		"max_tokens":  1000,
	})

	if err != nil {
		return nil, fmt.Errorf("LLM recommendation failed: %w", err)
	}

	// Parse LLM response
	return sr.parseLLMResponse(response, candidates)
}

// buildLLMPrompt builds a structured prompt for the LLM.
func (sr *SkillRecommender) buildLLMPrompt(
	candidates []SkillRecommendation,
	channel, userMessage string,
	history []providers.Message,
) string {
	var sb strings.Builder

	sb.WriteString("## Context\n")
	sb.WriteString(fmt.Sprintf("Channel: %s\n", channel))
	sb.WriteString(fmt.Sprintf("User Message: %s\n\n", userMessage))

	sb.WriteString("## Available Skills\n")
	for i, candidate := range candidates {
		sb.WriteString(fmt.Sprintf("%d. **%s** - Score: %.1f/100\n",
			i+1, candidate.Name, candidate.Score))
		sb.WriteString(fmt.Sprintf("   Description: %s\n", candidate.Reason))
	}

	sb.WriteString("\n## Task\n")
	sb.WriteString("Based on the context and available skills, recommend the most appropriate skills.\n")
	sb.WriteString("Consider:\n")
	sb.WriteString("- The user's intent from their message\n")
	sb.WriteString("- Channel-specific capabilities\n")
	sb.WriteString("- Historical context\n\n")
	sb.WriteString("Respond in JSON format:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"recommendations\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"skill_name\": \"skill-name\",\n")
	sb.WriteString("      \"confidence\": 0.9,\n")
	sb.WriteString("      \"reason\": \"why this skill is recommended\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n")

	return sb.String()
}

// parseLLMResponse parses the LLM's response and updates recommendations.
func (sr *SkillRecommender) parseLLMResponse(
	response *providers.LLMResponse,
	candidates []SkillRecommendation,
) ([]SkillRecommendation, error) {
	if response == nil || response.Content == "" {
		return candidates, fmt.Errorf("empty LLM response")
	}

	content := response.Content
	if content == "" {
		return candidates, fmt.Errorf("no content in LLM response")
	}

	// Try to extract JSON from response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		logger.WarnCF("agent", "No JSON found in LLM response",
			map[string]any{"content": content})
		return candidates, nil
	}

	jsonContent := content[jsonStart : jsonEnd+1]

	// Parse LLM response
	var llmResponse struct {
		Recommendations []struct {
			SkillName  string  `json:"skill_name"`
			Confidence float64 `json:"confidence"`
			Reason     string  `json:"reason"`
		} `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &llmResponse); err != nil {
		logger.WarnCF("agent", "Failed to parse LLM JSON",
			map[string]any{"error": err.Error(), "json": jsonContent})
		return candidates, nil
	}

	// Update candidate scores based on LLM feedback
	result := make([]SkillRecommendation, 0)
	for _, candidate := range candidates {
		for _, llmRec := range llmResponse.Recommendations {
			if llmRec.SkillName == candidate.Name {
				// Boost score and update confidence based on LLM recommendation
				candidate.Confidence = llmRec.Confidence
				candidate.Reason = llmRec.Reason
				candidate.Score = candidate.Score * (0.5 + candidate.Confidence*0.5) // Boost by up to 50%
				result = append(result, candidate)
				break
			}
		}
	}

	// If LLM didn't recommend anything, return original candidates
	if len(result) == 0 {
		return candidates, nil
	}

	return result, nil
}

// sortRecommendmentsByScore sorts recommendations by score in descending order.
func sortRecommendationsByScore(recommendations []SkillRecommendation) {
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
}

// extractKeywords extracts important keywords from text.
func extractKeywords(text string) []string {
	// Simple keyword extraction: remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "as": true,
		"into": true, "through": true, "during": true, "before": true,
		"after": true, "above": true, "below": true, "between": true,
		"and": true, "but": true, "or": true, "nor": true, "so": true,
		"yet": true, "both": true, "either": true, "neither": true,
		"not": true, "only": true, "own": true, "same": true, "than": true,
		"too": true, "very": true, "just": true, "can": true,
	}

	words := strings.Fields(text)
	keywords := make([]string, 0)

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:\"'()[]{}"))
		if len(word) > 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}
