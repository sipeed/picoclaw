// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import picolib "github.com/sipeed/picoclaw/pkg/pico"

// NewHandoffRequestPayload creates a NodePayload for a "handoff_request" action.
func NewHandoffRequestPayload(req *HandoffRequest) picolib.NodePayload {
	return picolib.NodePayload{
		picolib.PayloadKeyAction:  picolib.NodeActionHandoffRequest,
		picolib.PayloadKeyRequest: req,
	}
}

// HandoffResponseReply creates a reply payload carrying a HandoffResponse.
func HandoffResponseReply(resp *HandoffResponse) picolib.NodePayload {
	return picolib.NodePayload{picolib.PayloadKeyHandoffResp: resp}
}
