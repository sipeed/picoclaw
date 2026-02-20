// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// NATS subject patterns
const (
	SubjectHeartbeat         = "picoclaw.swarm.heartbeat.%s"      // {node_id}
	SubjectDiscoveryAnnounce = "picoclaw.swarm.discovery.announce"
	SubjectDiscoveryQuery    = "picoclaw.swarm.discovery.query"
	SubjectTaskAssign        = "picoclaw.swarm.task.assign.%s"    // {node_id}
	SubjectTaskBroadcast     = "picoclaw.swarm.task.broadcast.%s" // {capability}
	SubjectTaskResult        = "picoclaw.swarm.task.result.%s"    // {task_id}
	SubjectTaskProgress      = "picoclaw.swarm.task.progress.%s"  // {task_id}
	SubjectSystemShutdown    = "picoclaw.swarm.system.shutdown.%s" // {node_id}
)

// NATSBridge connects local MessageBus to NATS for swarm communication
type NATSBridge struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	nc       *nats.Conn
	localBus *bus.MessageBus
	nodeInfo *NodeInfo
	cfg      *config.SwarmConfig
	subs     []*nats.Subscription
	mu       sync.RWMutex
	running  bool

	// Callbacks
	onTaskReceived func(*SwarmTask)
	onNodeJoin     func(*NodeInfo)
	onNodeLeave    func(nodeID string)
}

// NewNATSBridge creates a new NATS bridge
func NewNATSBridge(cfg *config.SwarmConfig, localBus *bus.MessageBus, nodeInfo *NodeInfo) *NATSBridge {
	return &NATSBridge{
		localBus: localBus,
		nodeInfo: nodeInfo,
		cfg:      cfg,
		subs:     make([]*nats.Subscription, 0),
	}
}

// Connect establishes connection to NATS server(s)
func (nb *NATSBridge) Connect(ctx context.Context) error {
	opts := []nats.Option{
		nats.Name(fmt.Sprintf("picoclaw-%s", nb.nodeInfo.ID)),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.WarnCF("swarm", "NATS disconnected", map[string]interface{}{
				"error": fmt.Sprintf("%v", err),
			})
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.InfoCF("swarm", "NATS reconnected", map[string]interface{}{
				"url": nc.ConnectedUrl(),
			})
		}),
	}

	if nb.cfg.NATS.Credentials != "" {
		opts = append(opts, nats.UserCredentials(nb.cfg.NATS.Credentials))
	}

	urls := nats.DefaultURL
	if len(nb.cfg.NATS.URLs) > 0 {
		urls = strings.Join(nb.cfg.NATS.URLs, ",")
	}

	conn, err := nats.Connect(urls, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	nb.conn = conn
	nb.nc = conn

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	nb.js = js
	logger.InfoCF("swarm", "Connected to NATS", map[string]interface{}{
		"url": conn.ConnectedUrl(),
	})

	return nil
}

// Start begins listening for swarm messages
func (nb *NATSBridge) Start(ctx context.Context) error {
	nb.mu.Lock()
	nb.running = true
	nb.mu.Unlock()

	// Subscribe to task assignments for this node
	taskSub, err := nb.conn.Subscribe(
		fmt.Sprintf(SubjectTaskAssign, nb.nodeInfo.ID),
		nb.handleTaskAssignment,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to task assignments: %w", err)
	}
	nb.subs = append(nb.subs, taskSub)

	// Subscribe to capability-based broadcast tasks using queue groups for load balancing
	for _, cap := range nb.nodeInfo.Capabilities {
		broadcastSub, err := nb.conn.QueueSubscribe(
			fmt.Sprintf(SubjectTaskBroadcast, cap),
			"workers", // Queue group for load balancing
			nb.handleTaskBroadcast,
		)
		if err != nil {
			return fmt.Errorf("failed to subscribe to broadcast %s: %w", cap, err)
		}
		nb.subs = append(nb.subs, broadcastSub)
	}

	// Subscribe to discovery queries
	discoverySub, err := nb.conn.Subscribe(SubjectDiscoveryQuery, nb.handleDiscoveryQuery)
	if err != nil {
		return fmt.Errorf("failed to subscribe to discovery: %w", err)
	}
	nb.subs = append(nb.subs, discoverySub)

	// Subscribe to discovery announcements
	announceSub, err := nb.conn.Subscribe(SubjectDiscoveryAnnounce, nb.handleDiscoveryAnnounce)
	if err != nil {
		return fmt.Errorf("failed to subscribe to announcements: %w", err)
	}
	nb.subs = append(nb.subs, announceSub)

	// Announce our presence
	if err := nb.AnnouncePresence(); err != nil {
		logger.WarnCF("swarm", "Failed to announce presence", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.InfoCF("swarm", "NATS bridge started", map[string]interface{}{
		"node_id":      nb.nodeInfo.ID,
		"capabilities": fmt.Sprintf("%v", nb.nodeInfo.Capabilities),
	})

	return nil
}

// Stop gracefully stops the bridge
func (nb *NATSBridge) Stop() error {
	nb.mu.Lock()
	nb.running = false
	nb.mu.Unlock()

	// Unsubscribe all
	for _, sub := range nb.subs {
		if err := sub.Unsubscribe(); err != nil {
			logger.WarnCF("swarm", "Failed to unsubscribe", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	if nb.conn != nil && !nb.conn.IsClosed() {
		// Announce shutdown
		shutdownSubject := fmt.Sprintf(SubjectSystemShutdown, nb.nodeInfo.ID)
		_ = nb.conn.Publish(shutdownSubject, []byte(nb.nodeInfo.ID))

		// Drain and close
		return nb.conn.Drain()
	}
	return nil
}

// IsConnected returns true if connected to NATS
func (nb *NATSBridge) IsConnected() bool {
	return nb.conn != nil && nb.conn.IsConnected()
}

// AnnouncePresence broadcasts this node's presence
func (nb *NATSBridge) AnnouncePresence() error {
	announce := DiscoveryAnnounce{
		Node:      *nb.nodeInfo,
		Timestamp: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(announce)
	if err != nil {
		return fmt.Errorf("failed to marshal announcement: %w", err)
	}
	return nb.conn.Publish(SubjectDiscoveryAnnounce, data)
}

// PublishTask sends a task to a specific node or broadcasts by capability
func (nb *NATSBridge) PublishTask(task *SwarmTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	var subject string
	if task.AssignedTo != "" {
		subject = fmt.Sprintf(SubjectTaskAssign, task.AssignedTo)
	} else {
		subject = fmt.Sprintf(SubjectTaskBroadcast, task.Capability)
	}

	return nb.conn.Publish(subject, data)
}

// PublishTaskResult publishes the result of a completed task
func (nb *NATSBridge) PublishTaskResult(result *TaskResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal task result: %w", err)
	}
	subject := fmt.Sprintf(SubjectTaskResult, result.TaskID)
	return nb.conn.Publish(subject, data)
}

// PublishTaskProgress publishes progress update for a task
func (nb *NATSBridge) PublishTaskProgress(progress *TaskProgress) error {
	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}
	subject := fmt.Sprintf(SubjectTaskProgress, progress.TaskID)
	return nb.conn.Publish(subject, data)
}

// PublishHeartbeat publishes a heartbeat message
func (nb *NATSBridge) PublishHeartbeat(hb *Heartbeat) error {
	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}
	subject := fmt.Sprintf(SubjectHeartbeat, hb.NodeID)
	return nb.conn.Publish(subject, data)
}

// SubscribeTaskResult subscribes to results for a specific task
func (nb *NATSBridge) SubscribeTaskResult(taskID string, handler func(*TaskResult)) (*nats.Subscription, error) {
	subject := fmt.Sprintf(SubjectTaskResult, taskID)
	return nb.conn.Subscribe(subject, func(msg *nats.Msg) {
		var result TaskResult
		if err := json.Unmarshal(msg.Data, &result); err == nil {
			handler(&result)
		}
	})
}

// SubscribeHeartbeat subscribes to heartbeats from a specific node
func (nb *NATSBridge) SubscribeHeartbeat(nodeID string, handler func(*Heartbeat)) (*nats.Subscription, error) {
	subject := fmt.Sprintf(SubjectHeartbeat, nodeID)
	return nb.conn.Subscribe(subject, func(msg *nats.Msg) {
		var hb Heartbeat
		if err := json.Unmarshal(msg.Data, &hb); err == nil {
			handler(&hb)
		}
	})
}

// SubscribeAllHeartbeats subscribes to all heartbeat messages
func (nb *NATSBridge) SubscribeAllHeartbeats(handler func(*Heartbeat)) (*nats.Subscription, error) {
	return nb.conn.Subscribe("picoclaw.swarm.heartbeat.*", func(msg *nats.Msg) {
		var hb Heartbeat
		if err := json.Unmarshal(msg.Data, &hb); err == nil {
			handler(&hb)
		}
	})
}

// SubscribeShutdown subscribes to shutdown notices from a specific node
func (nb *NATSBridge) SubscribeShutdown(handler func(nodeID string)) (*nats.Subscription, error) {
	return nb.conn.Subscribe("picoclaw.swarm.system.shutdown.*", func(msg *nats.Msg) {
		handler(string(msg.Data))
	})
}

// RequestDiscovery sends a discovery query and collects responses
func (nb *NATSBridge) RequestDiscovery(query *DiscoveryQuery, timeout time.Duration) ([]*NodeInfo, error) {
	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery query: %w", err)
	}

	var nodes []*NodeInfo
	var mu sync.Mutex

	inbox := nb.conn.NewRespInbox()
	sub, err := nb.conn.Subscribe(inbox, func(msg *nats.Msg) {
		var node NodeInfo
		if err := json.Unmarshal(msg.Data, &node); err == nil {
			mu.Lock()
			nodes = append(nodes, &node)
			mu.Unlock()
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to discovery inbox: %w", err)
	}

	if err := nb.conn.PublishRequest(SubjectDiscoveryQuery, inbox, data); err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("failed to publish discovery query: %w", err)
	}

	time.Sleep(timeout)
	sub.Unsubscribe()

	return nodes, nil
}

// SetOnTaskReceived sets the callback for when a task is received
func (nb *NATSBridge) SetOnTaskReceived(handler func(*SwarmTask)) {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	nb.onTaskReceived = handler
}

// SetOnNodeJoin sets the callback for when a node joins
func (nb *NATSBridge) SetOnNodeJoin(handler func(*NodeInfo)) {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	nb.onNodeJoin = handler
}

// SetOnNodeLeave sets the callback for when a node leaves
func (nb *NATSBridge) SetOnNodeLeave(handler func(nodeID string)) {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	nb.onNodeLeave = handler
}

// Message handlers

func (nb *NATSBridge) handleTaskAssignment(msg *nats.Msg) {
	var task SwarmTask
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		logger.ErrorCF("swarm", "Failed to unmarshal task", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.InfoCF("swarm", "Received task assignment", map[string]interface{}{
		"task_id":    task.ID,
		"capability": task.Capability,
	})

	nb.mu.RLock()
	handler := nb.onTaskReceived
	nb.mu.RUnlock()

	if handler != nil {
		handler(&task)
	}
}

func (nb *NATSBridge) handleTaskBroadcast(msg *nats.Msg) {
	var task SwarmTask
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		return
	}

	logger.InfoCF("swarm", "Received broadcast task", map[string]interface{}{
		"task_id":    task.ID,
		"capability": task.Capability,
	})

	nb.mu.RLock()
	handler := nb.onTaskReceived
	nb.mu.RUnlock()

	if handler != nil {
		handler(&task)
	}
}

func (nb *NATSBridge) handleDiscoveryQuery(msg *nats.Msg) {
	var query DiscoveryQuery
	if err := json.Unmarshal(msg.Data, &query); err != nil {
		return
	}

	// Check if we match the query criteria
	if query.Role != "" && query.Role != nb.nodeInfo.Role {
		return
	}
	if query.Capability != "" && !containsCapability(nb.nodeInfo.Capabilities, query.Capability) {
		return
	}

	// Reply with our node info
	response, err := json.Marshal(nb.nodeInfo)
	if err != nil {
		return
	}
	if err := msg.Respond(response); err != nil {
		logger.WarnCF("swarm", "Failed to respond to discovery query", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (nb *NATSBridge) handleDiscoveryAnnounce(msg *nats.Msg) {
	var announce DiscoveryAnnounce
	if err := json.Unmarshal(msg.Data, &announce); err != nil {
		return
	}

	// Skip our own announcements
	if announce.Node.ID == nb.nodeInfo.ID {
		return
	}

	logger.InfoCF("swarm", "Node joined swarm", map[string]interface{}{
		"node_id":      announce.Node.ID,
		"role":         string(announce.Node.Role),
		"capabilities": fmt.Sprintf("%v", announce.Node.Capabilities),
	})

	nb.mu.RLock()
	handler := nb.onNodeJoin
	nb.mu.RUnlock()

	if handler != nil {
		handler(&announce.Node)
	}
}

// containsCapability checks if a capability is in the list
func containsCapability(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}
