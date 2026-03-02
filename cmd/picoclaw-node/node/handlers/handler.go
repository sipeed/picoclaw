package handlers

import "github.com/openclaw/go_node/gateway"

// Handler handles a specific invoke command
type Handler interface {
	Handle(req gateway.InvokeRequest) gateway.InvokeResult
}
