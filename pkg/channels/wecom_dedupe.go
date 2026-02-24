package channels

import "sync"

const wecomMaxProcessedMessages = 1000

// markMessageProcessed marks msgID as processed and returns false for duplicates.
// All map reads/writes (including len) are protected by msgMu to avoid races.
func markMessageProcessed(msgMu *sync.RWMutex, processedMsgs *map[string]bool, msgID string, maxEntries int) bool {
	if maxEntries <= 0 {
		maxEntries = wecomMaxProcessedMessages
	}

	msgMu.Lock()
	defer msgMu.Unlock()

	if *processedMsgs == nil {
		*processedMsgs = make(map[string]bool)
	}

	if (*processedMsgs)[msgID] {
		return false
	}
	(*processedMsgs)[msgID] = true

	// When over limit, reset dedupe map but keep the current message.
	if len(*processedMsgs) > maxEntries {
		*processedMsgs = map[string]bool{msgID: true}
	}

	return true
}
