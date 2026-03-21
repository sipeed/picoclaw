package agent

import "fmt"

// MockEventBus - for POC
var MockEventBus = struct {
	Emit func(event any)
}{
	Emit: func(event any) {
		fmt.Printf("[Mock EventBus] %T %+v\n", event, event)
	},
}
