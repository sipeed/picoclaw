// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/config"
)

// TestNATS provides a test NATS environment with embedded server
type TestNATS struct {
	embedded *EmbeddedNATS
	nc       *nats.Conn
	js       nats.JetStreamContext
	url      string
}

// SetupTestNATS creates a new test NATS environment
// Call t.Cleanup(func() { tn.Stop() }) to ensure cleanup
func SetupTestNATS(t *testing.T) *TestNATS {
	t.Helper()

	// Find a free port
	port := freePort(t)

	cfg := &config.NATSConfig{
		EmbeddedPort: port,
	}

	embedded := NewEmbeddedNATS(cfg)
	if err := embedded.Start(); err != nil {
		t.Fatalf("Failed to start embedded NATS: %v", err)
	}

	// Give the server a moment to be fully ready
	time.Sleep(100 * time.Millisecond)

	url := embedded.ClientURL()

	// Connect to the embedded server
	nc, err := nats.Connect(url)
	if err != nil {
		embedded.Stop()
		t.Fatalf("Failed to connect to embedded NATS: %v", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		embedded.Stop()
		t.Fatalf("Failed to create JetStream context: %v", err)
	}

	tn := &TestNATS{
		embedded: embedded,
		nc:       nc,
		js:       js,
		url:      url,
	}

	// Register cleanup
	t.Cleanup(func() {
		tn.Stop()
	})

	return tn
}

// Stop stops the test NATS environment
func (tn *TestNATS) Stop() {
	if tn.nc != nil {
		tn.nc.Close()
	}
	if tn.embedded != nil {
		tn.embedded.Stop()
	}
}

// NC returns the NATS connection
func (tn *TestNATS) NC() *nats.Conn {
	return tn.nc
}

// JS returns the JetStream context
func (tn *TestNATS) JS() nats.JetStreamContext {
	return tn.js
}

// URL returns the NATS server URL
func (tn *TestNATS) URL() string {
	return tn.url
}

// PublishTestMessage publishes a test message
func (tn *TestNATS) PublishTestMessage(subject string, data []byte) error {
	return tn.nc.Publish(subject, data)
}

// CreateTestStream creates a test JetStream stream
func (tn *TestNATS) CreateTestStream(streamName string, subjects []string) error {
	_, err := tn.js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: subjects,
		MaxAge:   1 * time.Hour,
	})
	return err
}

// CreateTestConsumer creates a test JetStream consumer
func (tn *TestNATS) CreateTestConsumer(stream, consumer string) error {
	_, err := tn.js.AddConsumer(stream, &nats.ConsumerConfig{
		Durable:   consumer,
		AckPolicy: nats.AckExplicitPolicy,
	})
	return err
}

// WaitForMessage waits for a message on the given subject
func (tn *TestNATS) WaitForMessage(ctx context.Context, subject string, timeout time.Duration) (*nats.Msg, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sub, err := tn.nc.SubscribeSync(subject)
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()

	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// CreateTestNodeInfo creates a test NodeInfo
func CreateTestNodeInfo(id, role string, capabilities []string) *NodeInfo {
	return &NodeInfo{
		ID:           id,
		Role:         NodeRole(role),
		Capabilities: capabilities,
		Model:        "test-model",
		Status:       StatusOnline,
		MaxTasks:     5,
		StartedAt:    time.Now().UnixMilli(),
		Metadata:     make(map[string]string),
	}
}

// CreateTestTask creates a test SwarmTask
func CreateTestTask(id, taskType, prompt, capability string) *SwarmTask {
	return &SwarmTask{
		ID:         id,
		Type:       SwarmTaskType(taskType),
		Prompt:     prompt,
		Capability: capability,
		Status:     TaskPending,
		CreatedAt:  time.Now().UnixMilli(),
		Timeout:    5 * 60 * 1000, // 5 minutes in milliseconds
	}
}

// RunTestWithNATS runs a test function with a test NATS environment
func RunTestWithNATS(t *testing.T, testFn func(*TestNATS)) {
	t.Helper()

	tn := SetupTestNATS(t)
	testFn(tn)
}

// freePort finds a free port for testing
// Note: This duplicates helpers_test.go to avoid build issues when
// testutil is used in non-test contexts
func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

// AssertConnected asserts that NATS is connected
func AssertConnected(t *testing.T, tn *TestNATS) {
	t.Helper()

	if !tn.nc.IsConnected() {
		t.Error("NATS connection is not established")
	}
}

// AssertStreamExists asserts that a stream exists in JetStream
func AssertStreamExists(t *testing.T, tn *TestNATS, streamName string) {
	t.Helper()

	stream, err := tn.js.StreamInfo(streamName)
	if err != nil {
		t.Fatalf("Stream %s does not exist: %v", streamName, err)
	}

	if stream == nil {
		t.Errorf("Stream %s is nil", streamName)
	}
}

// PurgeTestStream purges a test stream
func PurgeTestStream(t *testing.T, tn *TestNATS, streamName string) {
	t.Helper()

	if err := tn.js.PurgeStream(streamName, &nats.StreamPurgeRequest{}); err != nil {
		t.Logf("Warning: failed to purge stream %s: %v", streamName, err)
	}
}

// DeleteTestStream deletes a test stream
func DeleteTestStream(t *testing.T, tn *TestNATS, streamName string) {
	t.Helper()

	if err := tn.js.DeleteStream(streamName); err != nil {
		t.Logf("Warning: failed to delete stream %s: %v", streamName, err)
	}
}

// GetStreamInfo safely gets stream info, returning nil if stream doesn't exist
func GetStreamInfo(t *testing.T, tn *TestNATS, streamName string) *nats.StreamInfo {
	t.Helper()

	stream, err := tn.js.StreamInfo(streamName)
	if err != nil {
		return nil
	}
	return stream
}
