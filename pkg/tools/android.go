package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const androidToolTimeout = 15 * time.Second

var (
	packageNameRe  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)*$`)
	intentActionRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.]*$`)
)

// SendCallbackWithType is like SendCallback but includes a message type field.
type SendCallbackWithType func(channel, chatID, content, msgType string) error

// toolRequest is the JSON payload sent to the Android device via WebSocket.
type toolRequest struct {
	RequestID string                 `json:"request_id"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

type AndroidTool struct {
	sendCallback SendCallbackWithType
	channel      string
	chatID       string
	clientType   string
}

func NewAndroidTool() *AndroidTool {
	return &AndroidTool{}
}

func (t *AndroidTool) Name() string { return "android" }

func (t *AndroidTool) Description() string {
	return `Control the Android device. Available actions:
- list_apps: List installed apps
- app_info: Get app details (requires package_name)
- launch_app: Launch an app (requires package_name)
- current_activity: Get the currently active app/window
- tap: Tap a screen coordinate (requires x, y)
- swipe: Swipe between coordinates (requires x, y, x2, y2; optional duration_ms)
- text: Input text into the focused field (requires text)
- keyevent: Press a key (requires key: back/home/recents)
- broadcast: Send a broadcast intent (requires intent_action; optional intent_extras)
- intent: Start an activity via intent (requires intent_action; optional intent_data, intent_package, intent_type, intent_extras)

Only available in assistant mode.`
}

func (t *AndroidTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"list_apps", "app_info", "launch_app", "current_activity",
					"tap", "swipe", "text", "keyevent",
					"broadcast", "intent",
				},
				"description": "The device action to perform",
			},
			"package_name": map[string]interface{}{
				"type":        "string",
				"description": "Android package name (for app_info, launch_app)",
			},
			"x": map[string]interface{}{
				"type":        "number",
				"description": "X coordinate (for tap, swipe start)",
			},
			"y": map[string]interface{}{
				"type":        "number",
				"description": "Y coordinate (for tap, swipe start)",
			},
			"x2": map[string]interface{}{
				"type":        "number",
				"description": "End X coordinate (for swipe)",
			},
			"y2": map[string]interface{}{
				"type":        "number",
				"description": "End Y coordinate (for swipe)",
			},
			"duration_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Swipe duration in milliseconds (default 300)",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to input (for text action)",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"back", "home", "recents"},
				"description": "Key to press (for keyevent action)",
			},
			"intent_action": map[string]interface{}{
				"type":        "string",
				"description": "Intent action string (for broadcast, intent)",
			},
			"intent_data": map[string]interface{}{
				"type":        "string",
				"description": "Intent data URI (for intent)",
			},
			"intent_package": map[string]interface{}{
				"type":        "string",
				"description": "Target package for intent (for intent)",
			},
			"intent_type": map[string]interface{}{
				"type":        "string",
				"description": "MIME type for intent (for intent)",
			},
			"intent_extras": map[string]interface{}{
				"type":        "object",
				"description": "Extra key-value pairs for broadcast/intent",
			},
		},
		"required": []string{"action"},
	}
}

func (t *AndroidTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

func (t *AndroidTool) SetClientType(clientType string) {
	t.clientType = clientType
}

func (t *AndroidTool) SetSendCallback(cb SendCallbackWithType) {
	t.sendCallback = cb
}

func (t *AndroidTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.clientType == "" {
		return ErrorResult("android tool: not available (no assistant session)")
	}
	if t.clientType != "assistant" {
		return ErrorResult("android tool is only available in assistant mode")
	}
	if t.sendCallback == nil {
		return ErrorResult("android tool: send callback not configured")
	}
	if t.channel == "" || t.chatID == "" {
		return ErrorResult("android tool: no active channel context")
	}

	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	params, err := t.validateAndBuildParams(action, args)
	if err != nil {
		return ErrorResult(err.Error())
	}

	return t.sendAndWait(ctx, action, params)
}

func (t *AndroidTool) validateAndBuildParams(action string, args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	switch action {
	case "list_apps":
		// No params needed

	case "app_info", "launch_app":
		pkg, _ := args["package_name"].(string)
		if pkg == "" {
			return nil, fmt.Errorf("%s requires package_name", action)
		}
		if !packageNameRe.MatchString(pkg) {
			return nil, fmt.Errorf("invalid package_name: %s", pkg)
		}
		params["package_name"] = pkg

	case "current_activity":
		// No params needed

	case "tap":
		x, xOk := toFloat64(args["x"])
		y, yOk := toFloat64(args["y"])
		if !xOk || !yOk {
			return nil, fmt.Errorf("tap requires x and y coordinates")
		}
		params["x"] = x
		params["y"] = y

	case "swipe":
		x, xOk := toFloat64(args["x"])
		y, yOk := toFloat64(args["y"])
		x2, x2Ok := toFloat64(args["x2"])
		y2, y2Ok := toFloat64(args["y2"])
		if !xOk || !yOk || !x2Ok || !y2Ok {
			return nil, fmt.Errorf("swipe requires x, y, x2, y2 coordinates")
		}
		params["x"] = x
		params["y"] = y
		params["x2"] = x2
		params["y2"] = y2
		if dur, ok := toFloat64(args["duration_ms"]); ok {
			params["duration_ms"] = int(dur)
		}

	case "text":
		text, _ := args["text"].(string)
		if text == "" {
			return nil, fmt.Errorf("text action requires text parameter")
		}
		params["text"] = text

	case "keyevent":
		key, _ := args["key"].(string)
		if key == "" {
			return nil, fmt.Errorf("keyevent requires key parameter")
		}
		switch key {
		case "back", "home", "recents":
			// valid
		default:
			return nil, fmt.Errorf("invalid key: %s (must be back, home, or recents)", key)
		}
		params["key"] = key

	case "broadcast":
		intentAction, _ := args["intent_action"].(string)
		if intentAction == "" {
			return nil, fmt.Errorf("broadcast requires intent_action")
		}
		if !intentActionRe.MatchString(intentAction) {
			return nil, fmt.Errorf("invalid intent_action: %s", intentAction)
		}
		params["intent_action"] = intentAction
		if extras, ok := args["intent_extras"].(map[string]interface{}); ok {
			params["intent_extras"] = extras
		}

	case "intent":
		intentAction, _ := args["intent_action"].(string)
		if intentAction == "" {
			return nil, fmt.Errorf("intent requires intent_action")
		}
		if !intentActionRe.MatchString(intentAction) {
			return nil, fmt.Errorf("invalid intent_action: %s", intentAction)
		}
		params["intent_action"] = intentAction
		if data, ok := args["intent_data"].(string); ok && data != "" {
			params["intent_data"] = data
		}
		if pkg, ok := args["intent_package"].(string); ok && pkg != "" {
			if !packageNameRe.MatchString(pkg) {
				return nil, fmt.Errorf("invalid intent_package: %s", pkg)
			}
			params["intent_package"] = pkg
		}
		if mimeType, ok := args["intent_type"].(string); ok && mimeType != "" {
			params["intent_type"] = mimeType
		}
		if extras, ok := args["intent_extras"].(map[string]interface{}); ok {
			params["intent_extras"] = extras
		}

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	return params, nil
}

func (t *AndroidTool) sendAndWait(ctx context.Context, action string, params map[string]interface{}) *ToolResult {
	requestID := uuid.New().String()

	req := toolRequest{
		RequestID: requestID,
		Action:    action,
		Params:    params,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal tool request: %v", err))
	}

	// Register waiter before sending to avoid race
	respCh := DeviceResponseWaiter.Register(requestID)

	if err := t.sendCallback(t.channel, t.chatID, string(reqJSON), "tool_request"); err != nil {
		DeviceResponseWaiter.Cleanup(requestID)
		return ErrorResult(fmt.Sprintf("failed to send tool request: %v", err))
	}

	// Wait for response with timeout
	select {
	case content := <-respCh:
		// Check if the response indicates accessibility_required
		if strings.HasPrefix(content, "accessibility_required") {
			return &ToolResult{
				ForUser: "この機能にはユーザー補助の設定が必要です",
				ForLLM:  "accessibility_required: The accessibility service is not enabled. The settings dialog has been shown to the user. Do not retry automatically - wait for the user to enable the service and try again.",
			}
		}
		return SilentResult(content)
	case <-time.After(androidToolTimeout):
		DeviceResponseWaiter.Cleanup(requestID)
		return ErrorResult("android tool request timed out (15s)")
	case <-ctx.Done():
		DeviceResponseWaiter.Cleanup(requestID)
		return ErrorResult("android tool request cancelled")
	}
}

// toFloat64 extracts a float64 from an interface{} (handles both float64 and int from JSON).
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
