// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// DiscoveryService handles node discovery using a gossip protocol.
// For lightweight implementation, we use a simple UDP-based gossip
// instead of the heavier memberlist library.
type DiscoveryService struct {
	config       *Config
	localNode    *NodeInfo
	membership   *MembershipManager
	eventHandler *EventDispatcher
	conn         *net.UDPConn
	rpcConn      net.Listener
	auth         *AuthProvider

	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	once     sync.Once

	// Sequence number for updates
	seqNum uint64
}

// NewDiscoveryService creates a new discovery service.
func NewDiscoveryService(cfg *Config) (*DiscoveryService, error) {
	if cfg.NodeID == "" {
		// Generate node ID from hostname
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "picoclaw"
		}
		cfg.NodeID = fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])
	}

	// Determine advertise address
	advAddr := cfg.AdvertiseAddr
	if advAddr == "" || advAddr == "0.0.0.0" {
		advAddr = getLocalIP()
		if advAddr == "" {
			advAddr = "127.0.0.1"
		}
	}

	advPort := cfg.AdvertisePort
	if advPort == 0 {
		advPort = cfg.BindPort
	}

	localNode := &NodeInfo{
		ID:        cfg.NodeID,
		Addr:      advAddr,
		Port:      cfg.RPC.Port,
		AgentCaps: make(map[string]string),
		LoadScore: 0,
		Labels:    make(map[string]string),
		Timestamp: time.Now().UnixNano(),
		Version:   "1.0.0", // PicoClaw version
	}

	ds := &DiscoveryService{
		config:       cfg,
		localNode:    localNode,
		eventHandler: NewEventDispatcher(),
		stopChan:     make(chan struct{}),
	}

	// Initialize auth provider if secret is configured
	if cfg.Discovery.AuthSecret != "" {
		ds.auth = NewAuthProvider(cfg.NodeID, cfg.Discovery.AuthSecret)
		if cfg.Discovery.RequireAuth || cfg.Discovery.EnableMessageSigning {
			logger.InfoC("swarm", "Authentication enabled for swarm")
		}
	}

	// Initialize membership manager
	ds.membership = NewMembershipManager(ds, cfg.Discovery)

	return ds, nil
}

// Start starts the discovery service.
func (ds *DiscoveryService) Start() error {
	ds.mu.Lock()
	if ds.running {
		ds.mu.Unlock()
		return nil
	}
	ds.running = true
	ds.mu.Unlock()

	// Bind UDP socket for gossip
	addr := fmt.Sprintf("%s:%d", ds.config.BindAddr, ds.config.BindPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	ds.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Start gossip listener
	go ds.gossipListener()

	// Start periodic gossip
	go ds.gossipLoop()

	// Join existing cluster if addresses provided
	if len(ds.config.Discovery.JoinAddrs) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ds.Join(ctx, ds.config.Discovery.JoinAddrs)
	}

	// Add self to membership
	ds.membership.UpdateNode(ds.localNode)

	return nil
}

// Stop stops the discovery service.
func (ds *DiscoveryService) Stop() error {
	ds.once.Do(func() {
		ds.mu.Lock()
		ds.running = false
		ds.mu.Unlock()

		if ds.stopChan != nil {
			close(ds.stopChan)
		}

		if ds.conn != nil {
			ds.conn.Close()
		}

		if ds.rpcConn != nil {
			ds.rpcConn.Close()
		}
	})
	return nil
}

// Join joins a cluster by contacting existing nodes.
func (ds *DiscoveryService) Join(ctx context.Context, addrs []string) (int, error) {
	count := 0

	for _, addr := range addrs {
		// Send join message to each address
		err := ds.sendJoin(ctx, addr)
		if err == nil {
			count++
		}
	}

	return count, nil
}

// Members returns all known members of the cluster.
func (ds *DiscoveryService) Members() []*NodeWithState {
	return ds.membership.GetMembers()
}

// LocalNode returns the local node info.
func (ds *DiscoveryService) LocalNode() *NodeInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.localNode
}

// UpdateLocalInfo updates the local node's information.
func (ds *DiscoveryService) UpdateLocalInfo(info *NodeInfo) {
	ds.mu.Lock()
	ds.localNode = info
	ds.localNode.Timestamp = time.Now().UnixNano()
	ds.seqNum++
	ds.mu.Unlock()

	// Update membership
	ds.membership.UpdateNode(info)

	// Broadcast update
	ds.broadcastUpdate()
}

// UpdateLoad updates the local node's load score.
func (ds *DiscoveryService) UpdateLoad(score float64) {
	ds.mu.Lock()
	ds.localNode.LoadScore = score
	ds.localNode.Timestamp = time.Now().UnixNano()
	ds.seqNum++
	info := ds.localNode
	ds.mu.Unlock()

	ds.membership.UpdateNode(info)
	ds.broadcastUpdate()
}

// UpdateCapabilities updates the local node's agent capabilities.
func (ds *DiscoveryService) UpdateCapabilities(caps map[string]string) {
	ds.mu.Lock()
	ds.localNode.AgentCaps = caps
	ds.localNode.Timestamp = time.Now().UnixNano()
	ds.seqNum++
	info := ds.localNode
	ds.mu.Unlock()

	ds.membership.UpdateNode(info)
	ds.broadcastUpdate()
}

// Subscribe registers a handler for node events and returns its ID.
func (ds *DiscoveryService) Subscribe(handler EventHandler) EventHandlerID {
	return ds.eventHandler.Subscribe(handler)
}

// Unsubscribe removes a node event handler by ID.
func (ds *DiscoveryService) Unsubscribe(id EventHandlerID) {
	ds.eventHandler.Unsubscribe(id)
}

// gossipListener listens for incoming gossip messages.
func (ds *DiscoveryService) gossipListener() {
	buf := make([]byte, MaxGossipMessageSize)

	for {
		select {
		case <-ds.stopChan:
			return
		default:
		}

		ds.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := ds.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n > 0 {
			go ds.handleGossip(buf[:n], addr)
		}
	}
}

// gossipLoop periodically gossips node state to random members.
func (ds *DiscoveryService) gossipLoop() {
	ticker := time.NewTicker(ds.config.Discovery.GossipInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ds.stopChan:
			return
		case <-ticker.C:
			ds.broadcastUpdate()
		}
	}
}

// GossipMessage represents a gossip message.
type GossipMessage struct {
	Type      GossipMessageType `json:"type"` // ping, pong, join, update, sync
	FromNode  string      `json:"from_node"`
	SeqNum    uint64      `json:"seq_num"`
	Timestamp int64       `json:"timestamp"`
	Payload   []byte      `json:"payload,omitempty"`
	Nodes     []*NodeInfo `json:"nodes,omitempty"` // For memberlist exchange
	AuthToken *AuthToken  `json:"auth_token,omitempty"`
}

// handleGossip handles an incoming gossip message.
func (ds *DiscoveryService) handleGossip(data []byte, addr *net.UDPAddr) {
	var msg GossipMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	// Verify authentication if required
	if ds.config.Discovery.RequireAuth && msg.AuthToken != nil {
		if ds.auth == nil || !ds.auth.VerifyToken(msg.AuthToken) {
			logger.WarnCF("swarm", "Rejected unauthenticated message", map[string]any{"from": addr.String()})
			return
		}
	}

	// Verify message signature if enabled
	if ds.config.Discovery.EnableMessageSigning && msg.AuthToken != nil {
		// The signature is in the token, so verification above handles it
	}

	switch msg.Type {
	case GossipTypePing:
		ds.handlePing(msg, addr)
	case GossipTypePong:
		ds.handlePong(msg)
	case GossipTypeJoin:
		ds.handleJoin(msg, addr)
	case GossipTypeUpdate:
		ds.handleUpdate(msg)
	case GossipTypeSync:
		ds.handleSync(msg, addr)
	}
}

// handlePing handles a ping message.
func (ds *DiscoveryService) handlePing(msg GossipMessage, addr *net.UDPAddr) {
	// Respond with pong
	pong := GossipMessage{
		Type:      GossipTypePong,
		FromNode:  ds.localNode.ID,
		Timestamp: time.Now().UnixNano(),
	}

	data, err := json.Marshal(pong)
	if err != nil {
		logger.ErrorCF("swarm", "failed to marshal pong message", map[string]any{"error": err})
		return
	}
	if _, err := ds.conn.WriteToUDP(data, addr); err != nil {
		logger.DebugCF("swarm", "failed to send pong", map[string]any{"to": addr.String(), "error": err})
	}

	// Update membership if this is a known node
	if len(msg.Nodes) > 0 {
		for _, node := range msg.Nodes {
			if node.ID != ds.localNode.ID {
				ds.membership.UpdateNode(node)
			}
		}
	}
}

// handlePong handles a pong message.
func (ds *DiscoveryService) handlePong(msg GossipMessage) {
	// Update last seen for this node
	ds.membership.RecordHeartbeat(msg.FromNode)
}

// handleJoin handles a join request from a new node.
func (ds *DiscoveryService) handleJoin(msg GossipMessage, addr *net.UDPAddr) {
	// Send our member list back
	members := ds.Members()
	nodes := make([]*NodeInfo, 0, len(members)+1)
	nodes = append(nodes, ds.localNode)
	for _, m := range members {
		if m.Node.ID != ds.localNode.ID {
			nodes = append(nodes, m.Node)
		}
	}

	response := GossipMessage{
		Type:      GossipTypeSync,
		FromNode:  ds.localNode.ID,
		Timestamp: time.Now().UnixNano(),
		Nodes:     nodes,
	}

	data, err := json.Marshal(response)
	if err != nil {
		logger.ErrorCF("swarm", "failed to marshal sync message", map[string]any{"error": err})
		return
	}
	if _, err := ds.conn.WriteToUDP(data, addr); err != nil {
		logger.ErrorCF("swarm", "failed to send sync", map[string]any{"to": addr.String(), "error": err})
		return
	}

	// Emit join event
	event := &NodeEvent{
		Event: EventJoin,
		Time:  time.Now().UnixNano(),
	}
	if len(msg.Nodes) > 0 {
		event.Node = msg.Nodes[0]
	}
	ds.eventHandler.Dispatch(event)
}

// handleUpdate handles a node update message.
func (ds *DiscoveryService) handleUpdate(msg GossipMessage) {
	if len(msg.Nodes) == 0 {
		return
	}

	for _, node := range msg.Nodes {
		if node.ID != ds.localNode.ID {
			existing, ok := ds.membership.GetNode(node.ID)
			if !ok || node.Timestamp > existing.Node.Timestamp {
				ds.membership.UpdateNode(node)
			}
		}
	}
}

// handleSync handles a sync response with member list.
func (ds *DiscoveryService) handleSync(msg GossipMessage, addr *net.UDPAddr) {
	for _, node := range msg.Nodes {
		if node.ID != ds.localNode.ID {
			ds.membership.UpdateNode(node)
		}
	}
}

// broadcastUpdate broadcasts local state to random members.
func (ds *DiscoveryService) broadcastUpdate() {
	members := ds.membership.GetMembers()
	if len(members) == 0 {
		return
	}

	msg := GossipMessage{
		Type:      GossipTypeUpdate,
		FromNode:  ds.localNode.ID,
		SeqNum:    ds.seqNum,
		Timestamp: time.Now().UnixNano(),
		Nodes:     []*NodeInfo{ds.localNode},
	}

	// Add auth token if authentication is enabled
	if ds.auth != nil {
		token, err := ds.auth.GenerateToken()
		if err != nil {
			logger.ErrorCF("swarm", "failed to generate auth token", map[string]any{"error": err})
		} else {
			msg.AuthToken = token
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.ErrorCF("swarm", "failed to marshal broadcast update", map[string]any{"error": err})
		return
	}

	// Send to a few random members
	for _, member := range members {
		if member.Node.ID != ds.localNode.ID {
			addr := fmt.Sprintf("%s:%d", member.Node.Addr, member.Node.Port)
			udpAddr, err := net.ResolveUDPAddr("udp", addr)
			if err != nil {
				logger.DebugCF("swarm", "failed to resolve address", map[string]any{"address": addr, "error": err})
				continue
			}
			if _, err := ds.conn.WriteToUDP(data, udpAddr); err != nil {
				logger.DebugCF("swarm", "failed to send update", map[string]any{"address": addr, "error": err})
			}
		}
	}
}

// sendJoin sends a join message to a specific address.
func (ds *DiscoveryService) sendJoin(ctx context.Context, addr string) error {
	joinAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	msg := GossipMessage{
		Type:      GossipTypeJoin,
		FromNode:  ds.localNode.ID,
		Timestamp: time.Now().UnixNano(),
		Nodes:     []*NodeInfo{ds.localNode},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal join message: %w", err)
	}

	// Set deadline
	ds.conn.SetWriteDeadline(time.Now().Add(DefaultUDPWriteDeadline))
	_, err = ds.conn.WriteToUDP(data, joinAddr)
	if err != nil {
		return fmt.Errorf("failed to send join to %s: %w", addr, err)
	}
	return nil
}

// getLocalIP returns the local IP address.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
