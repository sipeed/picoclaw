package bus

import (
	"sync"
	"github.com/google/uuid"
	"github.com/sipeed/picoclaw/pkg/swarm/core"
)

type ChannelBus struct {
	subs map[string]map[string]func(core.Event)
	mu   sync.RWMutex
}

func NewChannelBus() *ChannelBus {
	return &ChannelBus{subs: make(map[string]map[string]func(core.Event))}
}

func (b *ChannelBus) Publish(topic string, e core.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if topics, ok := b.subs[topic]; ok {
		for _, handler := range topics {
			go handler(e)
		}
	}
	return nil
}

func (b *ChannelBus) Subscribe(topic string, handler func(core.Event)) (core.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[topic] == nil { b.subs[topic] = make(map[string]func(core.Event)) }
	id := uuid.New().String()
	b.subs[topic][id] = handler
	return &sub{bus: b, topic: topic, id: id}, nil
}

type sub struct { bus *ChannelBus; topic, id string }
func (s *sub) Unsubscribe() error {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	delete(s.bus.subs[s.topic], s.id)
	return nil
}