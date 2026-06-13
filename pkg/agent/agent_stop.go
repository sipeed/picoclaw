package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
)

func (al *AgentLoop) tryHandleStopCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	sessionKey string,
) bool {
	cmdName, ok := commands.CommandName(msg.Content)
	if !ok || cmdName != "stop" {
		return false
	}

	result, canceledEntries, err := al.stopActiveTurnForSession(sessionKey)

	// This function is only called when loaded=true (another turn already
	// claimed this session). If stopActiveTurnForSession found a pending
	// placeholder but didn't stop it, that placeholder belongs to the other
	// message's worker which hasn't started yet — arm a pending stop so the
	// worker will bail when it checks before running.
	if err == nil && !result.Stopped {
		if ts := al.getActiveTurnState(sessionKey); ts != nil {
			snap := ts.snapshot()
			if strings.HasPrefix(snap.TurnID, pendingTurnPrefix) {
				al.markPendingStop(sessionKey)
				result.Stopped = true
			}
		}
	}

	reply := commands.FormatStopReply(result)
	if err != nil {
		reply = "Failed to stop task: " + err.Error()
	}

	if al.channelManager != nil {
		al.channelManager.InvokeTypingStop(msg.Channel, msg.ChatID)
	}
	al.resetMessageToolRound(sessionKey)
	al.PublishResponseIfNeeded(ctx, msg.Channel, msg.ChatID, sessionKey, reply)

	stopStatus := turnDoneStatusOK
	if err != nil {
		stopStatus = turnDoneStatusError
	} else if result.Stopped || len(canceledEntries) > 0 {
		stopStatus = turnDoneStatusCanceled
	}
	al.notifyTurnDone(ctx, msg, stopStatus)
	al.notifyTurnDoneForSteeringEntries(ctx, canceledEntries, turnDoneStatusCanceled)
	return true
}

func (al *AgentLoop) stopActiveTurnForSession(
	sessionKey string,
) (commands.StopResult, []steeringQueueEntry, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return commands.StopResult{}, nil, fmt.Errorf("session key is required")
	}

	result := commands.StopResult{}
	clearedEntries := al.clearSteeringQueueEntriesForScope(sessionKey)
	al.clearPendingSkills(sessionKey)

	ts := al.getActiveTurnState(sessionKey)
	if ts == nil {
		result.Stopped = len(clearedEntries) > 0
		return result, clearedEntries, nil
	}

	snap := ts.snapshot()
	result.TaskName = snap.UserMessage

	if strings.HasPrefix(snap.TurnID, pendingTurnPrefix) {
		// A pending placeholder means this session is either idle (our own
		// placeholder from the /stop command) or another message is queued but
		// hasn't started yet. In both cases, we don't arm a pending stop here;
		// the caller (tryHandleStopCommand) handles the "another message queued"
		// case explicitly, since it knows loaded=true.
		return result, clearedEntries, nil
	}

	if err := al.HardAbort(sessionKey); err != nil {
		if al.getActiveTurnState(sessionKey) == nil {
			result.Stopped = len(clearedEntries) > 0
			return result, clearedEntries, nil
		}
		return commands.StopResult{}, clearedEntries, err
	}

	result.Stopped = true
	return result, clearedEntries, nil
}

func (al *AgentLoop) markPendingStop(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.pendingStops.Store(sessionKey, struct{}{})
}

func (al *AgentLoop) takePendingStop(sessionKey string) bool {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return false
	}
	_, ok := al.pendingStops.LoadAndDelete(sessionKey)
	return ok
}

func (al *AgentLoop) resetMessageToolRound(sessionKey string) {
	if strings.TrimSpace(sessionKey) == "" {
		return
	}
	if registry := al.GetRegistry(); registry != nil {
		if agent := registry.GetDefaultAgent(); agent != nil {
			if tool, ok := agent.Tools.Get("message"); ok {
				if resetter, ok := tool.(interface{ ResetSentInRound(sessionKey string) }); ok {
					resetter.ResetSentInRound(sessionKey)
				}
			}
		}
	}
}
