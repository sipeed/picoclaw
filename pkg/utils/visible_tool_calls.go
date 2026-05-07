package utils

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type VisibleToolCall struct {
	ID           string                       `json:"id,omitempty"`
	Type         string                       `json:"type,omitempty"`
	Function     *VisibleToolCallFunction     `json:"function,omitempty"`
	ExtraContent *VisibleToolCallExtraContent `json:"extra_content,omitempty"`
}

type VisibleToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type VisibleToolCallExtraContent struct {
	ToolFeedbackExplanation string `json:"tool_feedback_explanation,omitempty"`
}

func BuildVisibleToolCalls(
	toolCalls []providers.ToolCall,
	maxArgsLen int,
) []VisibleToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	visible := make([]VisibleToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name, _ := VisibleToolCallNameAndArguments(tc)
		argsPreview := VisibleToolCallArgumentsPreview(tc, maxArgsLen)
		explanation := ""
		if tc.ExtraContent != nil {
			explanation = strings.TrimSpace(tc.ExtraContent.ToolFeedbackExplanation)
		}
		if name == "" && explanation == "" && argsPreview == "" {
			continue
		}

		visibleCall := VisibleToolCall{
			ID:   strings.TrimSpace(tc.ID),
			Type: strings.TrimSpace(tc.Type),
		}
		if visibleCall.Type == "" {
			visibleCall.Type = "function"
		}
		if name != "" || argsPreview != "" {
			visibleCall.Function = &VisibleToolCallFunction{
				Name:      name,
				Arguments: argsPreview,
			}
		}
		if explanation != "" {
			visibleCall.ExtraContent = &VisibleToolCallExtraContent{
				ToolFeedbackExplanation: explanation,
			}
		}

		visible = append(visible, visibleCall)
	}

	if len(visible) == 0 {
		return nil
	}
	return visible
}

func VisibleToolCallNameAndArguments(tc providers.ToolCall) (string, string) {
	name := strings.TrimSpace(tc.Name)
	argsJSON := ""
	if tc.Function != nil {
		if name == "" {
			name = strings.TrimSpace(tc.Function.Name)
		}
		argsJSON = strings.TrimSpace(tc.Function.Arguments)
	}
	if argsJSON == "" && len(tc.Arguments) > 0 {
		if encodedArgs, err := json.Marshal(tc.Arguments); err == nil {
			argsJSON = string(encodedArgs)
		}
	}
	return name, strings.TrimSpace(argsJSON)
}

func VisibleToolCallArgumentsPreview(tc providers.ToolCall, maxLen int) string {
	_, argsJSON := VisibleToolCallNameAndArguments(tc)
	if argsJSON == "" {
		return ""
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(argsJSON), "", "  "); err == nil {
		argsJSON = pretty.String()
	}
	if maxLen > 0 {
		return Truncate(argsJSON, maxLen)
	}
	return argsJSON
}
