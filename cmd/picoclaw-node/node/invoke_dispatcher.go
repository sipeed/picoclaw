package node

import (
	"github.com/openclaw/go_node/config"
	"github.com/openclaw/go_node/gateway"
	"github.com/openclaw/go_node/node/handlers"
)

// InvokeDispatcher routes node.invoke.request to registered handlers
type InvokeDispatcher struct {
	handlers map[string]handlers.Handler
}

// NewInvokeDispatcher creates a dispatcher with default handlers using exec config
func NewInvokeDispatcher(execCfg config.ExecConfig) *InvokeDispatcher {
	return &InvokeDispatcher{
		handlers: map[string]handlers.Handler{
			"system.run":  handlers.NewSystemRunHandler(execCfg),
			"file.save":   handlers.NewFileSaveHandler(execCfg),
			"camera.snap": handlers.NewCameraSnapHandler(execCfg),
		},
	}
}

// Register adds or overrides a handler for command
func (d *InvokeDispatcher) Register(command string, h handlers.Handler) {
	if d.handlers == nil {
		d.handlers = make(map[string]handlers.Handler)
	}
	d.handlers[command] = h
}

// HandleInvoke dispatches to the handler for the command, or returns error if unknown
func (d *InvokeDispatcher) HandleInvoke(req gateway.InvokeRequest) gateway.InvokeResult {
	h, ok := d.handlers[req.Command]
	if !ok {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "UNAVAILABLE",
				Message: "command not supported",
			},
		}
	}
	return h.Handle(req)
}
