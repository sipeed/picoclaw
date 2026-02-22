// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElectionManager_SingleCandidate(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		em := NewElectionManager(tn.NC(), tn.JS(), "node-1", "test-hid", "test-sid")

		becameLeader := make(chan bool, 1)
		em.OnBecameLeader(func() {
			select {
			case becameLeader <- true:
			default:
			}
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cfg := &ElectionConfig{
			ElectionSubject:   "picoclaw.test.election",
			LeaseDuration:     2 * time.Second,
			ElectionInterval: 500 * time.Millisecond,
		}

		err := em.Start(ctx, cfg)
		require.NoError(t, err)

		// Should become leader immediately
		select {
		case <-becameLeader:
			assert.True(t, em.IsLeader())
			assert.Equal(t, "node-1", em.GetLeaderID())
		case <-time.After(1 * time.Second):
			t.Fatal("Did not become leader in time")
		}

		em.Stop()
	})
}

func TestElectionManager_MultipleCandidates(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg := &ElectionConfig{
			ElectionSubject:   "picoclaw.test.election.multi",
			LeaseDuration:     3 * time.Second,
			ElectionInterval: 1 * time.Second,
		}

		em1 := NewElectionManager(tn.NC(), tn.JS(), "node-1", "test-hid-multi", "sid-1")
		em2 := NewElectionManager(tn.NC(), tn.JS(), "node-2", "test-hid-multi", "sid-2")
		em3 := NewElectionManager(tn.NC(), tn.JS(), "node-3", "test-hid-multi", "sid-3")

		leaderChanges := make(chan string, 10)

		for _, em := range []*ElectionManager{em1, em2, em3} {
			em := em
			em.OnBecameLeader(func() {
				leaderChanges <- em.nodeID
			})
			em.OnNewLeader(func(leaderID string) {
				leaderChanges <- "new:" + leaderID
			})
		}

		// Start all with staggered delays
		require.NoError(t, em1.Start(ctx, cfg))

		time.Sleep(100 * time.Millisecond)
		require.NoError(t, em2.Start(ctx, cfg))

		time.Sleep(100 * time.Millisecond)
		require.NoError(t, em3.Start(ctx, cfg))

		// First node should be leader
		assert.Eventually(t, func() bool {
			return em1.IsLeader()
		}, 2*time.Second, 100*time.Millisecond, "node-1 should become leader")

		// Wait for all nodes to discover the leader
		assert.Eventually(t, func() bool {
			return em1.GetLeaderID() == "node-1" &&
				em2.GetLeaderID() == "node-1" &&
				em3.GetLeaderID() == "node-1"
		}, 2*time.Second, 100*time.Millisecond, "all nodes should discover node-1 as leader")

		// Stop leader, second should take over
		em1.Stop()

		assert.Eventually(t, func() bool {
			return !em1.IsLeader() && (em2.IsLeader() || em3.IsLeader())
		}, 5*time.Second, 200*time.Millisecond, "new leader should be elected")

		// Wait for all nodes to agree on the leader
		assert.Eventually(t, func() bool {
			l2 := em2.GetLeaderID()
			l3 := em3.GetLeaderID()
			return l2 != "" && l2 == l3
		}, 2*time.Second, 100*time.Millisecond, "all nodes should agree on leader")

		// Only one should be leader
		leaders := 0
		if em2.IsLeader() {
			leaders++
		}
		if em3.IsLeader() {
			leaders++
		}
		assert.Equal(t, 1, leaders, "Only one node should be leader")

		// All nodes should agree on the leader ID
		assert.Equal(t, em2.GetLeaderID(), em3.GetLeaderID())

		em2.Stop()
		em3.Stop()
	})
}

func TestElectionManager_LeaseRenewal(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		em := NewElectionManager(tn.NC(), tn.JS(), "node-1", "test-hid-renew", "test-sid")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg := &ElectionConfig{
			ElectionSubject:   "picoclaw.test.election.renew",
			LeaseDuration:     1 * time.Second,
			ElectionInterval: 300 * time.Millisecond,
		}

		require.NoError(t, em.Start(ctx, cfg))

		// Become leader
		assert.Eventually(t, func() bool {
			return em.IsLeader()
		}, 2*time.Second, 100*time.Millisecond)

		// Stay leader for multiple lease periods
		for i := 0; i < 5; i++ {
			time.Sleep(500 * time.Millisecond)
			assert.True(t, em.IsLeader(), "Should remain leader during lease period %d", i)
		}

		em.Stop()
	})
}

func TestElectionManager_StepDown(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		em1 := NewElectionManager(tn.NC(), tn.JS(), "node-1", "test-hid-stepdown", "sid-1")
		em2 := NewElectionManager(tn.NC(), tn.JS(), "node-2", "test-hid-stepdown", "sid-2")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg := &ElectionConfig{
			ElectionSubject:   "picoclaw.test.election.stepdown",
			LeaseDuration:     2 * time.Second,
			ElectionInterval: 500 * time.Millisecond,
		}

		require.NoError(t, em1.Start(ctx, cfg))
		require.NoError(t, em2.Start(ctx, cfg))

		// em1 should become leader
		assert.Eventually(t, func() bool {
			return em1.IsLeader() && !em2.IsLeader()
		}, 2*time.Second, 100*time.Millisecond)

		// em1 steps down
		em1.Stop()

		// em2 should take over
		assert.Eventually(t, func() bool {
			return !em1.IsLeader() && em2.IsLeader()
		}, 3*time.Second, 100*time.Millisecond)

		em2.Stop()
	})
}

func TestRoleSwitcher_PromotionToCoordinator(t *testing.T) {
	RunTestWithNATS(t, func(tn *TestNATS) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg := &ElectionConfig{
			ElectionSubject:   "picoclaw.test.role.switch",
			LeaseDuration:     2 * time.Second,
			ElectionInterval: 500 * time.Millisecond,
		}

		em := NewElectionManager(tn.NC(), tn.JS(), "node-1", "test-hid-switch", "sid-1")
		nodeInfo := CreateTestNodeInfo("node-1", string(RoleWorker), []string{"test"})

		// Create a minimal manager wrapper
		manager := &Manager{
			nodeInfo: nodeInfo,
		}

		rs := NewRoleSwitcher(em, nodeInfo, manager)
		rs.Start() // Start the role switcher to register callbacks

		require.NoError(t, em.Start(ctx, cfg))

		// Should become leader and promote to coordinator
		assert.Eventually(t, func() bool {
			return rs.GetCurrentRole() == RoleCoordinator
		}, 2*time.Second, 100*time.Millisecond)

		assert.Equal(t, RoleCoordinator, nodeInfo.Role)
		assert.Equal(t, string(RoleWorker), nodeInfo.Metadata["original_role"])

		em.Stop()
	})
}

func TestParseLeaderInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		nodeID   string
		sid      string
		expiry   int64
		ok       bool
	}{
		{
			name:   "valid input",
			input:  "node-1|sid-1|1234567890",
			nodeID: "node-1",
			sid:    "sid-1",
			expiry: 1234567890,
			ok:     true,
		},
		{
			name:  "invalid format",
			input: "invalid",
			ok:    false,
		},
		{
			name:  "missing parts",
			input: "node-1|sid-1",
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeID, sid, expiry, ok := parseLeaderInfo(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.nodeID, nodeID)
				assert.Equal(t, tt.sid, sid)
				assert.Equal(t, tt.expiry, expiry)
			}
		})
	}
}

func TestDefaultElectionConfig(t *testing.T) {
	cfg := DefaultElectionConfig()

	assert.Equal(t, "picoclaw.election", cfg.ElectionSubject)
	assert.Equal(t, 10*time.Second, cfg.LeaseDuration)
	assert.Equal(t, 3*time.Second, cfg.ElectionInterval)
	assert.Equal(t, time.Duration(0), cfg.PreVoteDelay)
}
