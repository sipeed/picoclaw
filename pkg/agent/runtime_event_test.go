package agent

import (
	"testing"
	"time"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
)

func subscribeRuntimeEventsForTest(
	t *testing.T,
	al *AgentLoop,
	buffer int,
	kinds ...runtimeevents.Kind,
) (<-chan runtimeevents.Event, func()) {
	t.Helper()

	if al == nil {
		t.Fatal("agent loop is nil")
	}
	channel := al.RuntimeEvents()
	if channel == nil {
		t.Fatal("runtime event channel is nil")
	}
	if len(kinds) > 0 {
		channel = channel.OfKind(kinds...)
	}
	sub, ch, err := channel.SubscribeChan(
		t.Context(),
		runtimeevents.SubscribeOptions{Name: "agent-runtime-test", Buffer: buffer},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}
	return ch, func() {
		if err := sub.Close(); err != nil {
			t.Errorf("runtime subscription close failed: %v", err)
		}
	}
}

func waitForRuntimeEvent(
	t *testing.T,
	ch <-chan runtimeevents.Event,
	timeout time.Duration,
	match func(runtimeevents.Event) bool,
) runtimeevents.Event {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				t.Fatal("runtime event stream closed before expected event arrived")
			}
			if match(evt) {
				return evt
			}
		case <-timer.C:
			t.Fatal("timed out waiting for expected runtime event")
		}
	}
}

func collectRuntimeEventStream(ch <-chan runtimeevents.Event) []runtimeevents.Event {
	var events []runtimeevents.Event
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, evt)
		default:
			return events
		}
	}
}

func findRuntimeEvent(
	events []runtimeevents.Event,
	kind runtimeevents.Kind,
) (runtimeevents.Event, bool) {
	for _, evt := range events {
		if evt.Kind == kind {
			return evt, true
		}
	}
	return runtimeevents.Event{}, false
}

func filterRuntimeEvents(events []runtimeevents.Event, kind runtimeevents.Kind) []runtimeevents.Event {
	var filtered []runtimeevents.Event
	for _, evt := range events {
		if evt.Kind == kind {
			filtered = append(filtered, evt)
		}
	}
	return filtered
}
