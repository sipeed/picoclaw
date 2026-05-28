package events

// Stats reports aggregate EventBus counters.
type Stats struct {
	Published   uint64
	Matched     uint64
	Delivered   uint64
	Dropped     uint64
	Blocked     uint64
	Closed      bool
	Subscribers int

	SubscriberStats []SubscriberStats
}

// SubscriberStats reports counters for one subscription.
type SubscriberStats struct {
	ID       uint64
	Name     string
	Received uint64
	Handled  uint64
	Failed   uint64
	Dropped  uint64
	Panicked uint64
	TimedOut uint64
}
