package tools

import (
	"context"
	"fmt"
)

type VoiceCallTool struct{}

// Compile-time check
var _ Tool = (*VoiceCallTool)(nil)

func NewVoiceCallTool() *VoiceCallTool {
	return &VoiceCallTool{}
}

func (t *VoiceCallTool) Name() string {
	return "voice_call"
}

func (t *VoiceCallTool) Description() string {
	return "Manage outbound and inbound voice calls via Twilio or Telnyx. Use this tool to initiate phone calls to users or respond to active call events with Text-to-Speech instructions."
}

func (t *VoiceCallTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"initiate_call", "speak_to_user", "end_call", "get_status"},
				"description": "The action to perform on the voice subsystem.",
			},
			"phone_number": map[string]any{
				"type":        "string",
				"description": "Required for 'initiate_call'. The E.164 phone number to call.",
			},
			"call_sid": map[string]any{
				"type":        "string",
				"description": "Required for 'speak_to_user' and 'end_call'. The unique call session ID.",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Required for 'speak_to_user'. The text to synthesize into speech for the user to hear.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *VoiceCallTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "initiate_call":
		number, _ := args["phone_number"].(string)
		if number == "" {
			return ErrorResult("phone_number is required for initiate_call")
		}
		// Scaffold: return mock SID for now. Next step requires Twilio client integration.
		msg := fmt.Sprintf("Initiated outbound call to %s. Call SID: mock_sid_12345", number)
		return &ToolResult{
			ForLLM:  msg,
			ForUser: msg,
		}

	case "speak_to_user":
		sid, _ := args["call_sid"].(string)
		text, _ := args["text"].(string)
		if sid == "" || text == "" {
			return ErrorResult("call_sid and text are required for speak_to_user")
		}
		// Scaffold: Update active call state to play TwiML <Say> on next webhook poll.
		msg := fmt.Sprintf("Queued speech for Call %s. Text: %s", sid, text)
		return &ToolResult{
			ForLLM:  msg,
			ForUser: msg,
		}

	case "end_call":
		sid, _ := args["call_sid"].(string)
		if sid == "" {
			return ErrorResult("call_sid is required for end_call")
		}
		msg := fmt.Sprintf("Ended call %s", sid)
		return &ToolResult{
			ForLLM:  msg,
			ForUser: msg,
		}

	case "get_status":
		sid, _ := args["call_sid"].(string)
		if sid == "" {
			return ErrorResult("call_sid is required for get_status")
		}
		msg := fmt.Sprintf("Call %s status: in-progress (scaffold)", sid)
		return &ToolResult{
			ForLLM:  msg,
			ForUser: msg,
		}

	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}
