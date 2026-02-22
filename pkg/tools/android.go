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

// UI-interaction actions restricted to assistant/overlay clients.
var uiActions = map[string]bool{
	"screenshot":  true,
	"get_ui_tree": true,
	"tap":         true,
	"swipe":       true,
	"text":        true,
	"keyevent":    true,
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

// SetClientType restricts available actions based on the connected client.
// "main" (chat mode) hides UI-interaction actions; other values allow all.
func (t *AndroidTool) SetClientType(ct string) {
	t.clientType = ct
}

func (t *AndroidTool) Description() string {
	if t.clientType == "main" {
		return `Control the Android device. Available actions:
- search_apps: Search installed apps by name or package name (requires query)
- app_info: Get app details (requires package_name)
- launch_app: Launch an app (requires package_name)
- broadcast: Send a broadcast intent (requires intent_action; optional intent_extras)
- intent: Start an activity via intent (requires intent_action; optional intent_data, intent_package, intent_type, intent_extras)
`
	}
	return `Control the Android device. Available actions:
- search_apps: Search installed apps by name or package name (requires query)
- app_info: Get app details (requires package_name)
- launch_app: Launch an app (requires package_name)
- screenshot: Capture a screenshot of the current screen (no params)
- get_ui_tree: Get the accessibility UI tree (optional: resource_id, index, bounds_x/bounds_y, max_depth, max_nodes)
- tap: Tap a screen coordinate (requires x, y)
- swipe: Swipe between coordinates (requires x, y, x2, y2; optional duration_ms)
- text: Input text into the focused field (requires text)
- keyevent: Press a key (requires key: back/home/recents)
- broadcast: Send a broadcast intent (requires intent_action; optional intent_extras)
- intent: Start an activity via intent (requires intent_action; optional intent_data, intent_package, intent_type, intent_extras)
`
}

func (t *AndroidTool) Parameters() map[string]interface{} {
	allActions := []string{
		"search_apps", "app_info", "launch_app",
		"screenshot", "get_ui_tree",
		"tap", "swipe", "text", "keyevent",
		"broadcast", "intent",
	}
	actions := allActions
	if t.clientType == "main" {
		filtered := make([]string, 0, len(allActions))
		for _, a := range allActions {
			if !uiActions[a] {
				filtered = append(filtered, a)
			}
		}
		actions = filtered
	}
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        actions,
				"description": "The device action to perform",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query for app name or package name (for search_apps)",
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
			"resource_id": map[string]interface{}{
				"type":        "string",
				"description": "View resource ID to start UI tree from (for get_ui_tree, e.g. com.example:id/button)",
			},
			"index": map[string]interface{}{
				"type":        "integer",
				"description": "Which match to use when resource_id has multiple hits (for get_ui_tree, default 0)",
			},
			"bounds_x": map[string]interface{}{
				"type":        "number",
				"description": "X coordinate to find the containing node (for get_ui_tree, alternative to resource_id)",
			},
			"bounds_y": map[string]interface{}{
				"type":        "number",
				"description": "Y coordinate to find the containing node (for get_ui_tree, alternative to resource_id)",
			},
			"max_depth": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum traversal depth (for get_ui_tree, default 15)",
			},
			"max_nodes": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of nodes to output (for get_ui_tree, default 300)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *AndroidTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

func (t *AndroidTool) SetSendCallback(cb SendCallbackWithType) {
	t.sendCallback = cb
}

func (t *AndroidTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
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

	// Safety guard: reject UI actions from chat-mode clients
	if t.clientType == "main" && uiActions[action] {
		return ErrorResult(fmt.Sprintf("action %q is not available in chat mode", action))
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
	case "search_apps":
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("search_apps requires query")
		}
		params["query"] = query

	case "app_info", "launch_app":
		pkg, _ := args["package_name"].(string)
		if pkg == "" {
			return nil, fmt.Errorf("%s requires package_name", action)
		}
		if !packageNameRe.MatchString(pkg) {
			return nil, fmt.Errorf("invalid package_name: %s", pkg)
		}
		params["package_name"] = pkg

	case "screenshot":
		// No params needed

	case "get_ui_tree":
		// Start node selection: resource_id or bounds (mutually exclusive)
		hasResourceID := false
		hasBounds := false
		if rid, ok := args["resource_id"].(string); ok && rid != "" {
			params["resource_id"] = rid
			hasResourceID = true
			if idx, ok := toFloat64(args["index"]); ok {
				idxInt := int(idx)
				if idxInt < 0 {
					return nil, fmt.Errorf("get_ui_tree: index must be non-negative, got %d", idxInt)
				}
				params["index"] = idxInt
			}
		}
		if bx, bxOk := toFloat64(args["bounds_x"]); bxOk {
			if by, byOk := toFloat64(args["bounds_y"]); byOk {
				params["bounds_x"] = bx
				params["bounds_y"] = by
				hasBounds = true
			}
		}
		if hasResourceID && hasBounds {
			return nil, fmt.Errorf("get_ui_tree: cannot specify both resource_id and bounds_x/bounds_y")
		}
		if md, ok := toFloat64(args["max_depth"]); ok {
			params["max_depth"] = int(md)
		}
		if mn, ok := toFloat64(args["max_nodes"]); ok {
			mnInt := int(mn)
			if mnInt < 1 {
				return nil, fmt.Errorf("get_ui_tree: max_nodes must be at least 1, got %d", mnInt)
			}
			params["max_nodes"] = mnInt
		}

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
		// Screenshot returns base64 JPEG data — wrap as multimodal result
		if action == "screenshot" {
			return &ToolResult{
				ForLLM: "Screenshot captured.",
				Media:  []string{"data:image/jpeg;base64," + content},
				Silent: true,
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
