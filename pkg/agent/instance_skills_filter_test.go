package agent

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentInstance_SetSkillsFilter(t *testing.T) {
	t.Parallel()

	// Create a minimal agent instance for testing
	agent := &AgentInstance{
		ID:           "test",
		SkillsFilter: nil,
	}

	t.Run("set initial filter", func(t *testing.T) {
		agent.SetSkillsFilter([]string{"skill1", "skill2"})

		filter := agent.GetSkillsFilter()
		assert.Equal(t, []string{"skill1", "skill2"}, filter)
	})

	t.Run("update existing filter", func(t *testing.T) {
		agent.SetSkillsFilter([]string{"skill3", "skill4"})

		filter := agent.GetSkillsFilter()
		assert.Equal(t, []string{"skill3", "skill4"}, filter)
	})

	t.Run("clear filter with nil", func(t *testing.T) {
		agent.SetSkillsFilter(nil)

		filter := agent.GetSkillsFilter()
		assert.Nil(t, filter)
	})

	t.Run("clear filter with empty slice", func(t *testing.T) {
		agent.SetSkillsFilter([]string{"skill1"})
		agent.SetSkillsFilter([]string{})

		filter := agent.GetSkillsFilter()
		assert.Empty(t, filter)
	})

	t.Run("returned slice is a copy", func(t *testing.T) {
		agent.SetSkillsFilter([]string{"skill1", "skill2"})

		filter1 := agent.GetSkillsFilter()
		filter1[0] = "modified"

		filter2 := agent.GetSkillsFilter()
		assert.Equal(t, "skill1", filter2[0], "original should not be modified")
	})

	t.Run("input slice is copied", func(t *testing.T) {
		input := []string{"skill1", "skill2"}
		agent.SetSkillsFilter(input)

		// Modify input after setting
		input[0] = "modified"

		filter := agent.GetSkillsFilter()
		assert.Equal(t, "skill1", filter[0], "filter should not be affected by input modification")
	})
}

func TestAgentInstance_GetSkillsFilter_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	agent := &AgentInstance{
		ID:           "test",
		SkillsFilter: []string{"skill1", "skill2", "skill3"},
	}

	// Run multiple concurrent reads
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			filter := agent.GetSkillsFilter()
			assert.Len(t, filter, 3)
		}()
	}

	wg.Wait()
}

func TestAgentInstance_SetSkillsFilter_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	agent := &AgentInstance{
		ID:           "test",
		SkillsFilter: []string{"skill1"},
	}

	// Run concurrent reads and writes
	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 100

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			agent.SetSkillsFilter([]string{"skill1", "skill2"})
		}(i)
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			filter := agent.GetSkillsFilter()
			// Should not panic, may see either old or new value
			_ = filter
		}()
	}

	wg.Wait()

	// Verify final state
	finalFilter := agent.GetSkillsFilter()
	assert.NotNil(t, finalFilter)
}

func TestAgentInstance_SkillsFilterPersistence(t *testing.T) {
	t.Parallel()

	agent := &AgentInstance{
		ID:           "test",
		SkillsFilter: nil,
	}

	// Set filter
	agent.SetSkillsFilter([]string{"customer-service", "faq", "order-tracking"})

	// Simulate multiple requests (concurrent reads)
	for i := 0; i < 10; i++ {
		filter := agent.GetSkillsFilter()
		assert.Equal(t, []string{"customer-service", "faq", "order-tracking"}, filter)
	}

	// Verify still persists after all reads
	finalFilter := agent.GetSkillsFilter()
	assert.Equal(t, []string{"customer-service", "faq", "order-tracking"}, finalFilter)
}
