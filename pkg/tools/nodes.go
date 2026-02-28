package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/nodes"
)

// NodesTool exposes all connected openclaw node capabilities to agents.
//
// Supported actions match the original openclaw nodes-tool.ts and canvas-tool.ts:
//
//	status           – list all connected nodes
//	describe         – describe a specific node
//	pending          – list nodes pending manual approval (pairing queue)
//	approve          – approve a pairing request by requestId
//	reject           – reject a pairing request by requestId
//	notify           – send a push notification to a node (system.notify)
//	camera_snap      – take a photo from a node camera (camera.snap)
//	camera_list      – list cameras on a node (camera.list)
//	camera_clip      – record a short video clip (camera.clip)
//	screen_record    – record the node screen (screen.record)
//	location_get     – get current GPS location from a node (location.get)
//	run              – execute a shell command on a node (system.run)
//	device_torch     – turn device flashlight on/off via on:bool (device.torch)
//	canvas_present   – show/open node canvas WebView (canvas.present)
//	canvas_hide      – hide the canvas WebView (canvas.hide)
//	canvas_navigate  – navigate canvas to a URL (canvas.navigate)
//	canvas_eval      – execute JavaScript in the canvas (canvas.eval)
//	canvas_snapshot  – capture a screenshot of the canvas (canvas.snapshot)
//	canvas_a2ui_push – push A2UI JSONL messages to the canvas (canvas.a2ui.pushJSONL)
//	canvas_a2ui_reset– reset the A2UI canvas state (canvas.a2ui.reset)
type NodesTool struct {
	registry *nodes.Registry
}

// NewNodesTool creates a NodesTool that reads from the given Registry.
func NewNodesTool(registry *nodes.Registry) *NodesTool {
	return &NodesTool{registry: registry}
}

func (t *NodesTool) Name() string { return "nodes" }

func (t *NodesTool) Description() string {
	return "Discover and control paired nodes (status/describe/pairing/notify/camera/screen/location/run/device/canvas)."
}

func (t *NodesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"description": "Action to perform. " +
					"Node discovery: status, describe. " +
					"Pairing: pending, approve, reject. " +
					"Notification: notify. " +
					"Camera: camera_snap (photo), camera_list, camera_clip (video). " +
					"Screen: screen_record. " +
					"Location: location_get. " +
					"Shell (desktop/server nodes only, NOT mobile): run. " +
					"Device: device_torch (control flashlight/torch — on:true to turn on, on:false to turn off). " +
					"Canvas/WebView (mobile & desktop nodes): " +
					"canvas_present (open WebView, optional url+placement), " +
					"canvas_hide (hide WebView), " +
					"canvas_navigate (navigate to URL — use this to open a URL in the node browser), " +
					"canvas_eval (execute JavaScript and return result), " +
					"canvas_snapshot (screenshot the WebView), " +
					"canvas_a2ui_push (push A2UI JSONL UI messages), " +
					"canvas_a2ui_reset (reset A2UI state). ",
				"enum": []string{
					"status", "describe",
					"pending", "approve", "reject",
					"notify",
					"camera_snap", "camera_list", "camera_clip",
					"screen_record",
					"location_get",
					"run",
					"device_torch",
					"canvas_present", "canvas_hide", "canvas_navigate",
					"canvas_eval", "canvas_snapshot",
					"canvas_a2ui_push", "canvas_a2ui_reset",
				},
			},
			"node": map[string]any{
				"type":        "string",
				"description": "Target node ID or display name (required for most actions except status/pending).",
			},
			"requestId": map[string]any{
				"type":        "string",
				"description": "Pairing request ID for approve/reject actions.",
			},
			"timeoutMs": map[string]any{
				"type":        "number",
				"description": "Overall invoke timeout in milliseconds (default: 30000).",
			},
			// notify
			"title": map[string]any{
				"type":        "string",
				"description": "Notification title (notify action).",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Notification body text (notify action).",
			},
			"sound": map[string]any{
				"type":        "string",
				"description": "Notification sound name (notify action).",
			},
			"priority": map[string]any{
				"type":        "string",
				"description": "Notification priority: passive | active | timeSensitive (notify action).",
				"enum":        []string{"passive", "active", "timeSensitive"},
			},
			"delivery": map[string]any{
				"type":        "string",
				"description": "Notification delivery channel: system | overlay | auto (notify action).",
				"enum":        []string{"system", "overlay", "auto"},
			},
			// camera_snap / camera_clip
			"facing": map[string]any{
				"type":        "string",
				"description": "Camera facing: front | back | both (camera_snap: all three; camera_clip: front|back only).",
				"enum":        []string{"front", "back", "both"},
			},
			"maxWidth": map[string]any{
				"type":        "number",
				"description": "Maximum image width in pixels (camera_snap).",
			},
			"quality": map[string]any{
				"type":        "number",
				"description": "JPEG quality 0–100 (camera_snap).",
			},
			"delayMs": map[string]any{
				"type":        "number",
				"description": "Capture delay in milliseconds (camera_snap).",
			},
			"deviceId": map[string]any{
				"type":        "string",
				"description": "Specific camera device ID (camera_snap / camera_clip).",
			},
			"durationMs": map[string]any{
				"type":        "number",
				"description": "Recording duration in milliseconds (camera_clip / screen_record; default 3000/10000).",
			},
			"includeAudio": map[string]any{
				"type":        "boolean",
				"description": "Whether to include audio (camera_clip / screen_record; default true).",
			},
			// screen_record
			"fps": map[string]any{
				"type":        "number",
				"description": "Recording frame rate (screen_record; default 10).",
			},
			"screenIndex": map[string]any{
				"type":        "number",
				"description": "Screen index to record (screen_record; default 0).",
			},
			"outPath": map[string]any{
				"type":        "string",
				"description": "Optional output file path (screen_record; default: temp file).",
			},
			// location_get
			"maxAgeMs": map[string]any{
				"type":        "number",
				"description": "Accept a cached location up to this many ms old (location_get).",
			},
			"locationTimeoutMs": map[string]any{
				"type":        "number",
				"description": "Max time to wait for a fresh GPS fix (location_get).",
			},
			"desiredAccuracy": map[string]any{
				"type":        "string",
				"description": "GPS accuracy hint: coarse | balanced | precise (location_get).",
				"enum":        []string{"coarse", "balanced", "precise"},
			},
			// run
			"command": map[string]any{
				"type":        "array",
				"description": "Argv array for the command to run, e.g. [\"echo\",\"hello\"] (run action).",
				"items":       map[string]any{"type": "string"},
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory for the command (run action).",
			},
			"env": map[string]any{
				"type":        "array",
				"description": "Extra environment variables as KEY=VALUE strings (run action).",
				"items":       map[string]any{"type": "string"},
			},
			"commandTimeoutMs": map[string]any{
				"type":        "number",
				"description": "Timeout for the remote command itself (run action).",
			},
			// canvas_present
			"url": map[string]any{
				"type":        "string",
				"description": "URL to load in the canvas (canvas_present / canvas_navigate).",
			},
			"x": map[string]any{
				"type":        "number",
				"description": "Canvas placement x offset in pixels (canvas_present).",
			},
			"y": map[string]any{
				"type":        "number",
				"description": "Canvas placement y offset in pixels (canvas_present).",
			},
			"width": map[string]any{
				"type":        "number",
				"description": "Canvas placement width in pixels (canvas_present).",
			},
			"height": map[string]any{
				"type":        "number",
				"description": "Canvas placement height in pixels (canvas_present).",
			},
			// canvas_eval
			"javaScript": map[string]any{
				"type":        "string",
				"description": "JavaScript code to execute in the canvas (canvas_eval).",
			},
			// canvas_snapshot
			"outputFormat": map[string]any{
				"type":        "string",
				"description": "Snapshot image format: png | jpg | jpeg (canvas_snapshot; default png).",
				"enum":        []string{"png", "jpg", "jpeg"},
			},
			// device_torch
			"on": map[string]any{
				"type":        "boolean",
				"description": "Turn the torch on (true) or off (false) (device_torch action).",
			},
			// canvas_a2ui_push
			"jsonl": map[string]any{
				"type":        "string",
				"description": "A2UI JSONL payload string, one JSON object per line (canvas_a2ui_push).",
			},
		},
		"required": []string{"action"},
	}
}

// Execute dispatches the requested action to the appropriate registry call.
func (t *NodesTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	timeoutMs := intArg(args, "timeoutMs", 30_000)

	switch action {

	// ── status ──────────────────────────────────────────────────────────────
	case "status":
		sessions := t.registry.ListConnected()
		if len(sessions) == 0 {
			return SilentResult("[]")
		}
		rows := make([]map[string]any, 0, len(sessions))
		for _, s := range sessions {
			rows = append(rows, map[string]any{
				"nodeId":        s.NodeID,
				"displayName":   s.DisplayName,
				"platform":      s.Platform,
				"deviceFamily":  s.DeviceFamily,
				"version":       s.Version,
				"remoteIp":      s.RemoteIP,
				"connectedAtMs": s.ConnectedAtMs,
				"caps":          s.Caps,
				"commands":      s.Commands,
			})
		}
		return jsonResult(rows)

	// ── describe ─────────────────────────────────────────────────────────────
	case "describe":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		s := t.registry.Get(nodeID)
		if s == nil {
			return ErrorResult(fmt.Sprintf("node %q not connected", nodeID))
		}
		return jsonResult(map[string]any{
			"nodeId":          s.NodeID,
			"displayName":     s.DisplayName,
			"platform":        s.Platform,
			"deviceFamily":    s.DeviceFamily,
			"modelIdentifier": s.ModelIdentifier,
			"version":         s.Version,
			"remoteIp":        s.RemoteIP,
			"connectedAtMs":   s.ConnectedAtMs,
			"caps":            s.Caps,
			"commands":        s.Commands,
		})

	// ── pending ───────────────────────────────────────────────────────────────
	case "pending":
		reqs := t.registry.ListPairingRequests()
		if len(reqs) == 0 {
			return SilentResult("[]")
		}
		rows := make([]map[string]any, 0, len(reqs))
		for _, r := range reqs {
			rows = append(rows, map[string]any{
				"requestId":     r.RequestID,
				"nodeId":        r.NodeID,
				"displayName":   r.DisplayName,
				"platform":      r.Platform,
				"deviceFamily":  r.DeviceFamily,
				"remoteIp":      r.RemoteIP,
				"requestedAtMs": r.RequestedAtMs,
			})
		}
		return jsonResult(rows)

	// ── approve ───────────────────────────────────────────────────────────────
	case "approve":
		reqID, _ := args["requestId"].(string)
		if reqID == "" {
			return ErrorResult("requestId is required for approve action")
		}
		if !t.registry.ApprovePairing(reqID) {
			return ErrorResult(fmt.Sprintf("pairing request %q not found", reqID))
		}
		return jsonResult(map[string]any{"ok": true, "requestId": reqID})

	// ── reject ────────────────────────────────────────────────────────────────
	case "reject":
		reqID, _ := args["requestId"].(string)
		if reqID == "" {
			return ErrorResult("requestId is required for reject action")
		}
		if !t.registry.RejectPairing(reqID) {
			return ErrorResult(fmt.Sprintf("pairing request %q not found", reqID))
		}
		return jsonResult(map[string]any{"ok": true, "requestId": reqID})

	// ── notify ────────────────────────────────────────────────────────────────
	case "notify":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		title, _ := args["title"].(string)
		body, _ := args["body"].(string)
		if title == "" && body == "" {
			return ErrorResult("title or body is required for notify action")
		}
		p := map[string]any{}
		if title != "" {
			p["title"] = title
		}
		if body != "" {
			p["body"] = body
		}
		if s, ok := args["sound"].(string); ok && s != "" {
			p["sound"] = s
		}
		if s, ok := args["priority"].(string); ok && s != "" {
			p["priority"] = s
		}
		if s, ok := args["delivery"].(string); ok && s != "" {
			p["delivery"] = s
		}
		result := t.registry.Invoke(nodeID, "system.notify", p, timeoutMs)
		if !result.Ok {
			return ErrorResult(invokeErrMsg(result))
		}
		return jsonResult(map[string]any{"ok": true})

	// ── camera_snap ───────────────────────────────────────────────────────────
	case "camera_snap":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		facing, _ := args["facing"].(string)
		if facing == "" {
			facing = "both"
		}
		facings := []string{}
		switch facing {
		case "front", "back":
			facings = []string{facing}
		default: // "both"
			facings = []string{"front", "back"}
		}

		p := map[string]any{"format": "jpg"}
		if v := floatArg(args, "maxWidth"); v > 0 {
			p["maxWidth"] = int(v)
		}
		if v := floatArg(args, "quality"); v > 0 {
			p["quality"] = int(v)
		}
		if v := floatArg(args, "delayMs"); v > 0 {
			p["delayMs"] = int(v)
		}
		if s, ok := args["deviceId"].(string); ok && s != "" {
			p["deviceId"] = s
		}

		type snapResult struct {
			facing string
			path   string
			width  any
			height any
		}
		var results []snapResult

		for _, f := range facings {
			p["facing"] = f
			res := t.registry.Invoke(nodeID, "camera.snap", p, timeoutMs)
			if !res.Ok {
				return ErrorResult(fmt.Sprintf("camera.snap facing=%s: %s", f, invokeErrMsg(res)))
			}
			payload, err := decodePayload(res)
			if err != nil {
				return ErrorResult(fmt.Sprintf("camera.snap parse: %v", err))
			}
			b64, _ := payload["base64"].(string)
			if b64 == "" {
				return ErrorResult("camera.snap: empty base64 in response")
			}
			imgBytes, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				imgBytes, err = base64.RawStdEncoding.DecodeString(b64)
				if err != nil {
					return ErrorResult("camera.snap: base64 decode failed")
				}
			}
			path := tempMediaPath("snap", f, "jpg")
			if err := os.WriteFile(path, imgBytes, 0o644); err != nil {
				return ErrorResult(fmt.Sprintf("camera.snap: write file: %v", err))
			}
			results = append(results, snapResult{
				facing: f,
				path:   path,
				width:  payload["width"],
				height: payload["height"],
			})
		}

		details := make([]map[string]any, 0, len(results))
		files := ""
		for _, r := range results {
			files += fmt.Sprintf("MEDIA:%s\n", r.path)
			details = append(details, map[string]any{
				"facing": r.facing,
				"path":   r.path,
				"width":  r.width,
				"height": r.height,
			})
		}
		detailJSON, _ := json.Marshal(details)
		return SilentResult(files + string(detailJSON))

	// ── camera_list ───────────────────────────────────────────────────────────
	case "camera_list":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		res := t.registry.Invoke(nodeID, "camera.list", map[string]any{}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, err := decodePayload(res)
		if err != nil {
			return jsonResult(map[string]any{})
		}
		return jsonResult(payload)

	// ── camera_clip ───────────────────────────────────────────────────────────
	case "camera_clip":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		facing, _ := args["facing"].(string)
		if facing == "" {
			facing = "front"
		}
		if facing != "front" && facing != "back" {
			return ErrorResult("camera_clip facing must be front or back")
		}
		durationMs := intArg(args, "durationMs", 3_000)
		includeAudio := true
		if v, ok := args["includeAudio"].(bool); ok {
			includeAudio = v
		}
		p := map[string]any{
			"facing":       facing,
			"durationMs":   durationMs,
			"includeAudio": includeAudio,
			"format":       "mp4",
		}
		if s, ok := args["deviceId"].(string); ok && s != "" {
			p["deviceId"] = s
		}
		res := t.registry.Invoke(nodeID, "camera.clip", p, timeoutMs+durationMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, err := decodePayload(res)
		if err != nil {
			return ErrorResult(fmt.Sprintf("camera.clip parse: %v", err))
		}
		b64, _ := payload["base64"].(string)
		if b64 == "" {
			return ErrorResult("camera.clip: empty base64 in response")
		}
		vidBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			vidBytes, err = base64.RawStdEncoding.DecodeString(b64)
			if err != nil {
				return ErrorResult("camera.clip: base64 decode failed")
			}
		}
		path := tempMediaPath("clip", facing, "mp4")
		if err := os.WriteFile(path, vidBytes, 0o644); err != nil {
			return ErrorResult(fmt.Sprintf("camera.clip: write file: %v", err))
		}
		return SilentResult(fmt.Sprintf("FILE:%s\n%s", path, mustJSON(map[string]any{
			"facing":     facing,
			"path":       path,
			"durationMs": payload["durationMs"],
			"hasAudio":   payload["hasAudio"],
		})))

	// ── screen_record ─────────────────────────────────────────────────────────
	case "screen_record":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		durationMs := intArg(args, "durationMs", 10_000)
		fps := intArg(args, "fps", 10)
		screenIndex := intArg(args, "screenIndex", 0)
		includeAudio := true
		if v, ok := args["includeAudio"].(bool); ok {
			includeAudio = v
		}
		p := map[string]any{
			"durationMs":   durationMs,
			"fps":          fps,
			"screenIndex":  screenIndex,
			"includeAudio": includeAudio,
			"format":       "mp4",
		}
		res := t.registry.Invoke(nodeID, "screen.record", p, timeoutMs+durationMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, err := decodePayload(res)
		if err != nil {
			return ErrorResult(fmt.Sprintf("screen.record parse: %v", err))
		}
		b64, _ := payload["base64"].(string)
		if b64 == "" {
			return ErrorResult("screen.record: empty base64 in response")
		}
		vidBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			vidBytes, err = base64.RawStdEncoding.DecodeString(b64)
			if err != nil {
				return ErrorResult("screen.record: base64 decode failed")
			}
		}
		outPath, _ := args["outPath"].(string)
		if outPath == "" {
			outPath = tempMediaPath("screen", fmt.Sprintf("idx%d", screenIndex), "mp4")
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return ErrorResult(fmt.Sprintf("screen.record: mkdir: %v", err))
		}
		if err := os.WriteFile(outPath, vidBytes, 0o644); err != nil {
			return ErrorResult(fmt.Sprintf("screen.record: write file: %v", err))
		}
		return SilentResult(fmt.Sprintf("FILE:%s\n%s", outPath, mustJSON(map[string]any{
			"path":        outPath,
			"durationMs":  payload["durationMs"],
			"fps":         payload["fps"],
			"screenIndex": payload["screenIndex"],
			"hasAudio":    payload["hasAudio"],
		})))

	// ── location_get ──────────────────────────────────────────────────────────
	case "location_get":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		p := map[string]any{}
		if v := floatArg(args, "maxAgeMs"); v > 0 {
			p["maxAgeMs"] = int(v)
		}
		if v := floatArg(args, "locationTimeoutMs"); v > 0 {
			p["timeoutMs"] = int(v)
		}
		if s, ok := args["desiredAccuracy"].(string); ok && s != "" {
			p["desiredAccuracy"] = s
		}
		res := t.registry.Invoke(nodeID, "location.get", p, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, _ := decodePayload(res)
		return jsonResult(payload)

	// ── run ───────────────────────────────────────────────────────────────────
	case "run":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		node := t.registry.Get(nodeID)
		if node == nil {
			return ErrorResult(fmt.Sprintf("node %q not connected", nodeID))
		}
		if !node.HasCommand("system.run") {
			return ErrorResult(fmt.Sprintf(
				"node %q does not support system.run (available commands: %v)",
				nodeID, node.Commands))
		}

		rawCmd, ok := args["command"]
		if !ok || rawCmd == nil {
			return ErrorResult("command is required (argv array, e.g. [\"echo\",\"hello\"])")
		}
		cmdSlice, ok := rawCmd.([]any)
		if !ok {
			return ErrorResult("command must be an array of strings")
		}
		cmd := make([]string, 0, len(cmdSlice))
		for _, v := range cmdSlice {
			cmd = append(cmd, fmt.Sprintf("%v", v))
		}
		if len(cmd) == 0 {
			return ErrorResult("command must not be empty")
		}

		// approved=true tells the node-host that picoclaw has already
		// authorized this invocation, bypassing its exec-approval prompt.
		p := map[string]any{"command": cmd, "approved": true}
		if v, ok := args["cwd"].(string); ok && v != "" {
			p["cwd"] = v
		}
		if v, ok := args["env"].([]any); ok && len(v) > 0 {
			envPairs := make([]string, 0, len(v))
			for _, e := range v {
				envPairs = append(envPairs, fmt.Sprintf("%v", e))
			}
			p["env"] = envPairs
		}
		if v := floatArg(args, "commandTimeoutMs"); v > 0 {
			p["timeoutMs"] = int(v)
		}

		invokeTimeout := timeoutMs
		if v := floatArg(args, "invokeTimeoutMs"); v > 0 {
			invokeTimeout = int(v)
		}

		res := t.registry.Invoke(nodeID, "system.run", p, invokeTimeout)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, _ := decodePayload(res)
		return jsonResult(payload)

	// ── device_torch ─────────────────────────────────────────────────────────
	case "device_torch":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}

		onVal, hasOn := args["on"].(bool)
		if !hasOn {
			return ErrorResult("device_torch requires on (boolean): true to turn on, false to turn off")
		}

		res := t.registry.Invoke(nodeID, "device.torch", map[string]any{"on": onVal}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		return jsonResult(map[string]any{"ok": true})

	// ── canvas_present ────────────────────────────────────────────────────────
	case "canvas_present":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		p := map[string]any{}
		if u, ok := args["url"].(string); ok && strings.TrimSpace(u) != "" {
			p["url"] = strings.TrimSpace(u)
		}
		// Optional placement object; only set if at least one coordinate is given.
		placement := map[string]any{}
		if v, ok := args["x"].(float64); ok {
			placement["x"] = int(v)
		}
		if v, ok := args["y"].(float64); ok {
			placement["y"] = int(v)
		}
		if v, ok := args["width"].(float64); ok {
			placement["width"] = int(v)
		}
		if v, ok := args["height"].(float64); ok {
			placement["height"] = int(v)
		}
		if len(placement) > 0 {
			p["placement"] = placement
		}
		res := t.registry.Invoke(nodeID, "canvas.present", p, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		return jsonResult(map[string]any{"ok": true})

	// ── canvas_hide ───────────────────────────────────────────────────────────
	case "canvas_hide":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		res := t.registry.Invoke(nodeID, "canvas.hide", map[string]any{}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		return jsonResult(map[string]any{"ok": true})

	// ── canvas_navigate ───────────────────────────────────────────────────────
	case "canvas_navigate":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		u, ok := args["url"].(string)
		if !ok || strings.TrimSpace(u) == "" {
			return ErrorResult("url is required for canvas_navigate")
		}
		res := t.registry.Invoke(nodeID, "canvas.navigate", map[string]any{"url": strings.TrimSpace(u)}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		return jsonResult(map[string]any{"ok": true})

	// ── canvas_eval ───────────────────────────────────────────────────────────
	case "canvas_eval":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		js, ok := args["javaScript"].(string)
		if !ok || strings.TrimSpace(js) == "" {
			return ErrorResult("javaScript is required for canvas_eval")
		}
		res := t.registry.Invoke(nodeID, "canvas.eval", map[string]any{"javaScript": js}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, _ := decodePayload(res)
		// Return the result string if present, otherwise ok.
		if result, ok := payload["result"].(string); ok && result != "" {
			return SilentResult(result)
		}
		return jsonResult(map[string]any{"ok": true})

	// ── canvas_snapshot ───────────────────────────────────────────────────────
	case "canvas_snapshot":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		// Normalise format: jpg/jpeg → jpeg; anything else → png.
		formatRaw := "png"
		if f, ok := args["outputFormat"].(string); ok {
			formatRaw = strings.ToLower(strings.TrimSpace(f))
		}
		format := "png"
		if formatRaw == "jpg" || formatRaw == "jpeg" {
			format = "jpeg"
		}
		p := map[string]any{"format": format}
		if v := floatArg(args, "maxWidth"); v > 0 {
			p["maxWidth"] = int(v)
		}
		if v := floatArg(args, "quality"); v > 0 {
			p["quality"] = int(v)
		}
		res := t.registry.Invoke(nodeID, "canvas.snapshot", p, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		payload, err := decodePayload(res)
		if err != nil {
			return ErrorResult(fmt.Sprintf("canvas.snapshot parse: %v", err))
		}
		b64, _ := payload["base64"].(string)
		if b64 == "" {
			return ErrorResult("canvas.snapshot: empty base64 in response")
		}
		imgBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			imgBytes, err = base64.RawStdEncoding.DecodeString(b64)
			if err != nil {
				return ErrorResult("canvas.snapshot: base64 decode failed")
			}
		}
		ext := format // "png" or "jpeg"
		if ext == "jpeg" {
			ext = "jpg"
		}
		path := tempMediaPath("canvas", "snapshot", ext)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return ErrorResult(fmt.Sprintf("canvas.snapshot: mkdir: %v", err))
		}
		if err := os.WriteFile(path, imgBytes, 0o644); err != nil {
			return ErrorResult(fmt.Sprintf("canvas.snapshot: write file: %v", err))
		}
		return SilentResult(fmt.Sprintf("MEDIA:%s\n%s", path, mustJSON(map[string]any{
			"format": format,
			"path":   path,
		})))

	// ── canvas_a2ui_push ──────────────────────────────────────────────────────
	case "canvas_a2ui_push":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		jsonl, ok := args["jsonl"].(string)
		if !ok || strings.TrimSpace(jsonl) == "" {
			return ErrorResult("jsonl is required for canvas_a2ui_push")
		}
		res := t.registry.Invoke(nodeID, "canvas.a2ui.pushJSONL", map[string]any{"jsonl": jsonl}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		return jsonResult(map[string]any{"ok": true})

	// ── canvas_a2ui_reset ─────────────────────────────────────────────────────
	case "canvas_a2ui_reset":
		nodeID, err := t.resolveNode(args)
		if err != nil {
			return ErrorResult(err.Error())
		}
		res := t.registry.Invoke(nodeID, "canvas.a2ui.reset", map[string]any{}, timeoutMs)
		if !res.Ok {
			return ErrorResult(invokeErrMsg(res))
		}
		return jsonResult(map[string]any{"ok": true})

	default:
		return ErrorResult(fmt.Sprintf("unknown action: %q", action))
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

// resolveNode resolves "node" arg to a nodeID by matching connected sessions.
// resolveNode resolves the "node" argument to a nodeID.
// Mirrors openclaw's resolveNodeIdFromList logic:
//   - "current" / "default" / empty → picks the only connected node, or the
//     first one when multiple are connected (auto-default).
//   - Exact nodeId match.
//   - Exact remoteIp match.
//   - Normalized display-name match (lower-case, non-alnum → "-").
//   - Partial nodeId prefix match (query length ≥ 6).
func (t *NodesTool) resolveNode(args map[string]any) (string, error) {
	ref, _ := args["node"].(string)
	sessions := t.registry.ListConnected()

	// Auto-default: "current", "default", or empty query → pick the single
	// connected node, or the first one if there are multiple.
	if ref == "" || ref == "current" || ref == "default" {
		if len(sessions) == 0 {
			return "", fmt.Errorf("no nodes are currently connected")
		}
		return sessions[0].NodeID, nil
	}

	qNorm := normalizeNodeKey(ref)

	var matches []*nodes.NodeSession
	for _, s := range sessions {
		if s.NodeID == ref {
			return s.NodeID, nil // exact ID → return immediately
		}
		if s.RemoteIP == ref {
			matches = append(matches, s)
			continue
		}
		if s.DisplayName != "" && normalizeNodeKey(s.DisplayName) == qNorm {
			matches = append(matches, s)
			continue
		}
		// Partial prefix match (at least 6 chars to avoid false positives).
		if len(ref) >= 6 && strings.HasPrefix(s.NodeID, ref) {
			matches = append(matches, s)
		}
	}

	if len(matches) == 1 {
		return matches[0].NodeID, nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, s := range matches {
			if s.DisplayName != "" {
				names[i] = s.DisplayName
			} else {
				names[i] = s.NodeID
			}
		}
		return "", fmt.Errorf("ambiguous node %q (matches: %s)", ref, strings.Join(names, ", "))
	}

	// Build a known-nodes hint for the error message.
	known := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if s.DisplayName != "" {
			known = append(known, s.DisplayName)
		} else {
			known = append(known, s.NodeID)
		}
	}
	hint := ""
	if len(known) > 0 {
		hint = fmt.Sprintf(" (connected: %s)", strings.Join(known, ", "))
	}
	return "", fmt.Errorf("no connected node matches %q%s", ref, hint)
}

// normalizeNodeKey converts a string to a lowercase slug (non-alnum → "-"),
// matching openclaw's normalizeNodeKey used for display-name comparison.
func normalizeNodeKey(s string) string {
	var b strings.Builder
	prevDash := true // suppress leading dashes
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	// Trim trailing dash.
	result := b.String()
	return strings.TrimRight(result, "-")
}

// jsonResult marshals v and wraps it in a SilentResult.
func jsonResult(v any) *ToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("json marshal error: %v", err))
	}
	return SilentResult(string(data))
}

// mustJSON marshals v, returning "{}" on error.
func mustJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// invokeErrMsg formats an error message from an InvokeResult.
func invokeErrMsg(res nodes.InvokeResult) string {
	if res.Error != nil {
		if res.Error.Message != "" {
			return res.Error.Message
		}
		return res.Error.Code
	}
	return "unknown error"
}

// decodePayload extracts the payload map from an InvokeResult.
// It tries PayloadJSON first, then Payload directly.
func decodePayload(res nodes.InvokeResult) (map[string]any, error) {
	if res.PayloadJSON != nil && *res.PayloadJSON != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(*res.PayloadJSON), &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	if res.Payload != nil {
		if m, ok := res.Payload.(map[string]any); ok {
			return m, nil
		}
		// Re-marshal round-trip.
		data, _ := json.Marshal(res.Payload)
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	return map[string]any{}, nil
}

// tempMediaPath returns a temp file path for node media files.
func tempMediaPath(kind, label, ext string) string {
	ts := time.Now().UnixMilli()
	name := fmt.Sprintf("node_%s_%s_%d.%s", kind, label, ts, ext)
	return filepath.Join(os.TempDir(), "picoclaw", name)
}

// intArg reads an integer parameter with a fallback default.
func intArg(args map[string]any, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok && v > 0 {
		return int(v)
	}
	return defaultVal
}

// floatArg reads a numeric parameter (JSON numbers arrive as float64).
func floatArg(args map[string]any, key string) float64 {
	v, _ := args[key].(float64)
	return v
}
