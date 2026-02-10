package core

import (
	"context"
)

type SwarmStore interface {
	CreateSwarm(ctx context.Context, swarm *Swarm) error
	GetSwarm(ctx context.Context, id SwarmID) (*Swarm, error)
	UpdateSwarm(ctx context.Context, swarm *Swarm) error
	ListSwarms(ctx context.Context, status SwarmStatus) ([]*Swarm, error)
	
	CreateNode(ctx context.Context, node *Node) error
	GetNode(ctx context.Context, id NodeID) (*Node, error)
	UpdateNode(ctx context.Context, node *Node) error
	GetSwarmNodes(ctx context.Context, swarmID SwarmID) ([]*Node, error)
}

type SharedMemory interface {
	SaveFact(ctx context.Context, fact Fact) error
	SearchFacts(ctx context.Context, swarmID SwarmID, query string, limit int, global bool) ([]FactResult, error)
	ClearMemory(ctx context.Context, swarmID SwarmID) error
}

type EventBus interface {
	Publish(topic string, event Event) error
	Subscribe(topic string, handler func(Event)) (Subscription, error)
}

type Subscription interface {
	Unsubscribe() error
}

type LLMClient interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDef, model string) (*LLMResponse, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}