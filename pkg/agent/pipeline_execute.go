// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const repeatedFatalToolErrorStreakLimit = 3

type mcpServerTool interface {
	MCPServerName() string
}

func toolErrorSummary(result *tools.ToolResult) string {
	if result == nil || !result.IsError {
		return ""
	}
	content := strings.TrimSpace(result.ContentForLLM())
	if content == "" && result.Err != nil {
		content = strings.TrimSpace(result.Err.Error())
	}
	return utils.Truncate(content, 200)
}

func isFatalMCPTransportErrorSummary(summary string) bool {
	summary = strings.ToLower(strings.TrimSpace(summary))
	if summary == "" || !strings.Contains(summary, "mcp tool execution failed") {
		return false
	}
	return strings.Contains(summary, "client is closing") ||
		strings.Contains(summary, "connection closed: calling \"tools/call\"") ||
		strings.Contains(summary, "invalid character") ||
		strings.Contains(summary, "broken pipe") ||
		strings.Contains(summary, "eof")
}

func repeatedFatalToolErrorReply(toolName string) string {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return "I hit repeated backend tool transport errors and stopped instead of retrying indefinitely. Please try again."
	}
	return fmt.Sprintf(
		"I hit repeated backend tool transport errors while using `%s` and stopped instead of retrying indefinitely. Please try again.",
		toolName,
	)
}

func fatalMCPServerErrorReply(serverName, toolName string) string {
	serverName = strings.TrimSpace(serverName)
	toolName = strings.TrimSpace(toolName)
	if serverName != "" {
		return fmt.Sprintf(
			"I hit a backend MCP transport error while using the `%s` server and stopped instead of trying workarounds. Please restart or fix that MCP server, then try again.",
			serverName,
		)
	}
	if toolName != "" {
		return fmt.Sprintf(
			"I hit a backend MCP transport error while using `%s` and stopped instead of trying workarounds. Please restart or fix that MCP server, then try again.",
			toolName,
		)
	}
	return "I hit a backend MCP transport error and stopped instead of trying workarounds. Please restart or fix that MCP server, then try again."
}

func mcpServerNameForTool(ts *turnState, toolName string) string {
	if ts == nil || ts.agent == nil || ts.agent.Tools == nil {
		return ""
	}
	tool, ok := ts.agent.Tools.Get(toolName)
	if !ok || tool == nil {
		return ""
	}
	mcpTool, ok := tool.(mcpServerTool)
	if !ok {
		return ""
	}
	return strings.TrimSpace(mcpTool.MCPServerName())
}

func inferSkillNamesFromToolCall(ts *turnState, toolName string, toolArgs map[string]any) []string {
	if ts == nil || toolName != "read_file" {
		return nil
	}

	rawPath, ok := toolArgs["path"].(string)
	if !ok {
		return nil
	}
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return nil
	}

	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(ts.workspace, cleanPath)
	}
	if filepath.Base(cleanPath) != "SKILL.md" {
		return nil
	}

	var roots []string
	if ts.agent != nil && ts.agent.ContextBuilder != nil {
		roots = ts.agent.ContextBuilder.skillRoots()
	}
	if len(roots) == 0 && strings.TrimSpace(ts.workspace) != "" {
		roots = []string{filepath.Join(ts.workspace, "skills")}
	}

	found := make(map[string]struct{})
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(root), cleanPath)
		if err != nil {
			continue
		}
		if rel == "." || rel == "" || strings.HasPrefix(rel, "..") {
			continue
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) != 2 || parts[1] != "SKILL.md" {
			continue
		}

		skillName := strings.TrimSpace(parts[0])
		if skillName == "" {
			continue
		}
		if ts.agent != nil && ts.agent.ContextBuilder != nil {
			if canonical, ok := ts.agent.ContextBuilder.ResolveSkillName(skillName); ok {
				skillName = canonical
			}
		}
		found[skillName] = struct{}{}
	}

	if len(found) == 0 {
		return nil
	}

	names := make([]string, 0, len(found))
	for skillName := range found {
		names = append(names, skillName)
	}
	sort.Strings(names)
	return names
}

func shouldPublishAsyncToolResultToUser(result *tools.ToolResult) bool {
	return decideAsyncToolResultDelivery(result).PublishToUser
}

func shouldQueueAsyncToolResultForParent(result *tools.ToolResult) bool {
	return decideAsyncToolResultDelivery(result).QueueParent
}

func asyncResultContentLen(result *tools.ToolResult) int {
	return decideAsyncToolResultDelivery(result).ContentLen
}

func asyncResultForUserLen(result *tools.ToolResult) int {
	return decideAsyncToolResultDelivery(result).ForUserLen
}

func asyncResultMediaCount(result *tools.ToolResult) int {
	return decideAsyncToolResultDelivery(result).MediaCount
}

func recordCompletionMedia(exec *turnExecution, store media.MediaStore, refs []string) {
	if exec == nil || len(refs) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(exec.completionMedia)+len(refs))
	for _, item := range exec.completionMedia {
		seen[item.Ref] = struct{}{}
	}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		exec.completionMedia = append(exec.completionMedia, buildCompletionMedia(store, ref))
		seen[ref] = struct{}{}
	}
}

func buildCompletionMedia(store media.MediaStore, ref string) tools.CompletionMedia {
	item := tools.CompletionMedia{Ref: ref}
	if store == nil {
		return item
	}
	_, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		return item
	}
	item.Filename = meta.Filename
	item.ContentType = meta.ContentType
	item.Type = inferMediaType(meta.Filename, meta.ContentType)
	return item
}

// ExecuteTools executes the tool loop, handling BeforeTool/ApproveTool/AfterTool hooks,
// tool execution with async callbacks, media delivery, and steering injection.
// Returns ToolControl indicating what the coordinator should do next:
//   - ToolControlContinue: all tool results handled, pendingMessages or steering exists, continue turn
//   - ToolControlBreak: tool loop exited, proceed to coordinator's hardAbort/finalContent/finalize
func (p *Pipeline) ExecuteTools(
	ctx context.Context,
	turnCtx context.Context,
	ts *turnState,
	exec *turnExecution,
	iteration int,
) ToolControl {
	al := p.al
	normalizedToolCalls := exec.normalizedToolCalls

	ts.setPhase(TurnPhaseTools)
	messages := exec.messages
	handledAttachments := make([]providers.Attachment, 0)

toolLoop:
	for i, tc := range normalizedToolCalls {
		if ts.hardAbortRequested() {
			exec.abortedByHardAbort = true
			return ToolControlBreak
		}

		toolName := tc.Name
		toolArgs := cloneStringAnyMap(tc.Arguments)

		if al.hooks != nil {
			toolReq, decision := al.hooks.BeforeTool(turnCtx, &ToolCallHookRequest{
				Meta:      ts.eventMeta("runTurn", "turn.tool.before"),
				Context:   cloneTurnContext(ts.turnCtx),
				Tool:      toolName,
				Arguments: toolArgs,
			})
			switch decision.normalizedAction() {
			case HookActionContinue, HookActionModify:
				if toolReq != nil {
					toolName = toolReq.Tool
					toolArgs = toolReq.Arguments
				}
			case HookActionRespond:
				if toolReq != nil && toolReq.HookResult != nil {
					hookResult := toolReq.HookResult

					argsJSON, _ := json.Marshal(toolArgs)
					argsPreview := utils.Truncate(string(argsJSON), 200)
					logger.InfoCF("agent", fmt.Sprintf("Tool call (hook respond): %s(%s)", toolName, argsPreview),
						map[string]any{
							"agent_id":  ts.agent.ID,
							"tool":      toolName,
							"iteration": iteration,
						})

					al.emitEvent(
						runtimeevents.KindAgentToolExecStart,
						ts.eventMeta("runTurn", "turn.tool.start"),
						ToolExecStartPayload{
							Tool:      toolName,
							Arguments: cloneEventArguments(toolArgs),
						},
					)

					if shouldPublishToolFeedback(al, ts) && ts.channel != "pico" {
						toolFeedbackMaxLen := al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength()
						toolFeedbackExplanation := toolFeedbackExplanationForToolCall(
							exec.response,
							tc,
							messages,
						)
						feedbackMsg := utils.FormatToolFeedbackMessageWithStyle(
							al.cfg.Agents.Defaults.GetToolFeedbackStyle(),
							toolName,
							toolFeedbackExplanation,
							toolFeedbackArgsPreview(toolArgs, toolFeedbackMaxLen),
						)
						if title := toolFeedbackTitleForTurn(ts); title != "" {
							feedbackMsg = utils.FormatToolFeedbackMessageWithStyleAndTitle(
								al.cfg.Agents.Defaults.GetToolFeedbackStyle(),
								title,
								toolName,
								toolFeedbackExplanation,
								toolFeedbackArgsPreview(toolArgs, toolFeedbackMaxLen),
							)
						}
						fbCtx, fbCancel := context.WithTimeout(turnCtx, 3*time.Second)
						_ = al.bus.PublishOutbound(fbCtx, outboundMessageForTurnWithOptions(
							ts,
							feedbackMsg,
							outboundTurnMessageOptions{kind: messageKindToolFeedback},
						))
						fbCancel()
					}

					toolDuration := time.Duration(0)

					if ts.opts.SuppressToolUserDelivery {
						hookResult.ResponseHandled = false
					}

					if !ts.opts.SuppressToolUserDelivery && hookResult.ResponseHandled {
						attachments, delivered, err := al.deliverToolResultToUser(ctx, ts, hookResult, toolName)
						if err != nil {
							hookResult.IsError = true
							hookResult.ForLLM = fmt.Sprintf("failed to deliver attachment: %v", err)
						} else if delivered {
							handledAttachments = append(handledAttachments, attachments...)
						} else if len(toolResultMediaRefs(hookResult)) > 0 {
							hookResult.ResponseHandled = false
						}
					}

					shouldSendForUser := !hookResult.ResponseHandled &&
						!ts.opts.SuppressToolUserDelivery &&
						!hookResult.Silent &&
						hookResult.ForUser != "" &&
						ts.opts.SendResponse
					if shouldSendForUser {
						al.bus.PublishOutbound(ctx, outboundMessageForTurn(ts, hookResult.ForUser))
					}

					if !hookResult.ResponseHandled {
						exec.allResponsesHandled = false
					}

					contentForLLM := hookResult.ContentForLLM()
					if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
						contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
					}

					toolResultMsg := providers.Message{
						Role:       "tool",
						Content:    contentForLLM,
						ToolCallID: tc.ID,
					}

					if len(hookResult.Media) > 0 && !hookResult.ResponseHandled {
						recordCompletionMedia(exec, al.mediaStore, hookResult.Media)
						hookResult.ArtifactTags = buildArtifactTags(al.mediaStore, hookResult.Media)
						contentForLLM = hookResult.ContentForLLM()
						if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
							contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
						}
						toolResultMsg.Content = contentForLLM
						toolResultMsg.Media = append(toolResultMsg.Media, hookResult.Media...)
					}

					al.emitEvent(
						runtimeevents.KindAgentToolExecEnd,
						ts.eventMeta("runTurn", "turn.tool.end"),
						ToolExecEndPayload{
							Tool:       toolName,
							Duration:   toolDuration,
							ForLLMLen:  len(contentForLLM),
							ForUserLen: len(hookResult.ForUser),
							IsError:    hookResult.IsError,
							Async:      hookResult.Async,
						},
					)
					ts.recordToolExecution(
						toolName,
						!hookResult.IsError,
						toolErrorSummary(hookResult),
						inferSkillNamesFromToolCall(ts, toolName, toolArgs),
					)

					messages = append(messages, toolResultMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
						ts.recordPersistedMessage(toolResultMsg)
						ts.ingestMessage(turnCtx, al, toolResultMsg)
					}

					if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
						exec.markAdditionalUserInputObserved()
						exec.pendingMessages = append(exec.pendingMessages, steerMsgs...)
					}

					skipReason := ""
					skipMessage := ""
					if len(exec.pendingMessages) > 0 {
						skipReason = "queued user steering message"
						skipMessage = "Skipped due to queued user message."
					} else if gracefulPending, _ := ts.gracefulInterruptRequested(); gracefulPending {
						skipReason = "graceful interrupt requested"
						skipMessage = "Skipped due to graceful interrupt."
					}

					if skipReason != "" {
						remaining := len(normalizedToolCalls) - i - 1
						if remaining > 0 {
							logger.InfoCF("agent", "Turn checkpoint: skipping remaining tools after hook respond",
								map[string]any{
									"agent_id":  ts.agent.ID,
									"completed": i + 1,
									"skipped":   remaining,
									"reason":    skipReason,
								})
							for j := i + 1; j < len(normalizedToolCalls); j++ {
								skippedTC := normalizedToolCalls[j]
								al.emitEvent(
									runtimeevents.KindAgentToolExecSkipped,
									ts.eventMeta("runTurn", "turn.tool.skipped"),
									ToolExecSkippedPayload{
										Tool:   skippedTC.Name,
										Reason: skipReason,
									},
								)
								skippedMsg := providers.Message{
									Role:       "tool",
									Content:    skipMessage,
									ToolCallID: skippedTC.ID,
								}
								messages = append(messages, skippedMsg)
								if !ts.opts.NoHistory {
									ts.agent.Sessions.AddFullMessage(ts.sessionKey, skippedMsg)
									ts.recordPersistedMessage(skippedMsg)
								}
							}
						}
						break toolLoop
					}

					if ts.pendingResults != nil {
						select {
						case result, ok := <-ts.pendingResults:
							if ok && result != nil && result.ForLLM != "" {
								content := al.cfg.FilterSensitiveData(result.ForLLM)
								msg := subTurnResultPromptMessage(content)
								messages = append(messages, msg)
								ts.agent.Sessions.AddFullMessage(ts.sessionKey, msg)
							}
						default:
						}
					}

					continue
				}
				logger.WarnCF("agent", "Hook returned respond action but no HookResult provided",
					map[string]any{
						"agent_id": ts.agent.ID,
						"tool":     toolName,
						"action":   "respond",
					})
			case HookActionDenyTool:
				exec.allResponsesHandled = false
				denyContent := hookDeniedToolContent("Tool execution denied by hook", decision.Reason)
				al.emitEvent(
					runtimeevents.KindAgentToolExecSkipped,
					ts.eventMeta("runTurn", "turn.tool.skipped"),
					ToolExecSkippedPayload{
						Tool:   toolName,
						Reason: denyContent,
					},
				)
				deniedMsg := providers.Message{
					Role:       "tool",
					Content:    denyContent,
					ToolCallID: tc.ID,
				}
				messages = append(messages, deniedMsg)
				if !ts.opts.NoHistory {
					ts.agent.Sessions.AddFullMessage(ts.sessionKey, deniedMsg)
					ts.recordPersistedMessage(deniedMsg)
				}
				continue
			case HookActionAbortTurn:
				exec.abortedByHook = true
				return ToolControlBreak
			case HookActionHardAbort:
				_ = ts.requestHardAbort()
				exec.abortedByHardAbort = true
				return ToolControlBreak
			}
		}

		if al.hooks != nil {
			approval := al.hooks.ApproveTool(turnCtx, &ToolApprovalRequest{
				Meta:      ts.eventMeta("runTurn", "turn.tool.approve"),
				Context:   cloneTurnContext(ts.turnCtx),
				Tool:      toolName,
				Arguments: toolArgs,
			})
			if !approval.Approved {
				exec.allResponsesHandled = false
				denyContent := hookDeniedToolContent("Tool execution denied by approval hook", approval.Reason)
				al.emitEvent(
					runtimeevents.KindAgentToolExecSkipped,
					ts.eventMeta("runTurn", "turn.tool.skipped"),
					ToolExecSkippedPayload{
						Tool:   toolName,
						Reason: denyContent,
					},
				)
				deniedMsg := providers.Message{
					Role:       "tool",
					Content:    denyContent,
					ToolCallID: tc.ID,
				}
				messages = append(messages, deniedMsg)
				if !ts.opts.NoHistory {
					ts.agent.Sessions.AddFullMessage(ts.sessionKey, deniedMsg)
					ts.recordPersistedMessage(deniedMsg)
				}
				continue
			}
		}

		argsJSON, _ := json.Marshal(toolArgs)
		argsPreview := utils.Truncate(string(argsJSON), 200)
		logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", toolName, argsPreview),
			map[string]any{
				"agent_id":  ts.agent.ID,
				"tool":      toolName,
				"iteration": iteration,
			})
		al.emitEvent(
			runtimeevents.KindAgentToolExecStart,
			ts.eventMeta("runTurn", "turn.tool.start"),
			ToolExecStartPayload{
				Tool:      toolName,
				Arguments: cloneEventArguments(toolArgs),
			},
		)

		if shouldPublishToolFeedback(al, ts) && ts.channel != "pico" {
			toolFeedbackMaxLen := al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength()
			toolFeedbackExplanation := toolFeedbackExplanationForToolCall(
				exec.response,
				tc,
				messages,
			)
			feedbackMsg := utils.FormatToolFeedbackMessageWithStyle(
				al.cfg.Agents.Defaults.GetToolFeedbackStyle(),
				toolName,
				toolFeedbackExplanation,
				toolFeedbackArgsPreview(toolArgs, toolFeedbackMaxLen),
			)
			if title := toolFeedbackTitleForTurn(ts); title != "" {
				feedbackMsg = utils.FormatToolFeedbackMessageWithStyleAndTitle(
					al.cfg.Agents.Defaults.GetToolFeedbackStyle(),
					title,
					toolName,
					toolFeedbackExplanation,
					toolFeedbackArgsPreview(toolArgs, toolFeedbackMaxLen),
				)
			}
			fbCtx, fbCancel := context.WithTimeout(turnCtx, 3*time.Second)
			_ = al.bus.PublishOutbound(fbCtx, outboundMessageForTurnWithOptions(
				ts,
				feedbackMsg,
				outboundTurnMessageOptions{kind: messageKindToolFeedback},
			))
			fbCancel()
		}

		toolCallID := tc.ID
		asyncToolName := toolName
		mcpServerName := mcpServerNameForTool(ts, toolName)
		var asyncAckDelivery AsyncDeliveryDecision
		if tool, ok := ts.agent.Tools.Get(toolName); ok {
			if _, isAsync := tool.(tools.AsyncExecutor); isAsync {
				if deliveryMode, err := asyncDeliveryModeFromToolArgs(toolName, toolArgs); err == nil {
					asyncAckDelivery = decideAsyncToolResultDelivery(
						tools.AsyncResult("").WithAsyncDelivery(deliveryMode),
					)
				}
			}
		}
		asyncCallback := func(_ context.Context, result *tools.ToolResult) {
			completionID := asyncCompletionID(ts.turnID, toolCallID, asyncToolName)
			delivery := decideAsyncToolResultDelivery(result)
			al.emitEvent(
				runtimeevents.KindAgentAsyncCompletion,
				ts.scope.meta(iteration, "runTurn", "turn.async.completion"),
				AsyncCompletionPayload{
					SourceTool:   asyncToolName,
					CompletionID: completionID,
					TaskID:       delivery.TaskID,
					DeliveryMode: string(delivery.DeliveryMode),
					ContentLen:   delivery.ContentLen,
					ForUserLen:   delivery.ForUserLen,
					MediaCount:   delivery.MediaCount,
					IsError:      delivery.IsError,
					WillUser:     delivery.PublishToUser,
					WillParent:   delivery.QueueParent,
				},
			)
			if result != nil && result.IsError {
				al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
					TurnState:    ts,
					ToolName:     asyncToolName,
					CompletionID: completionID,
					Result:       result,
					Decision:     delivery,
				})
				return
			}
			al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
				TurnState:    ts,
				ToolName:     asyncToolName,
				CompletionID: completionID,
				Result:       result,
				Decision:     delivery,
			})
		}

		toolStart := time.Now()
		execCtx := tools.WithToolInboundContext(
			turnCtx,
			ts.channel,
			ts.chatID,
			ts.opts.Dispatch.MessageID(),
			ts.opts.Dispatch.ReplyToMessageID(),
		)
		execCtx = tools.WithToolTopicID(execCtx, originTopicID(ts.opts.Dispatch.InboundContext))
		execCtx = tools.WithToolSessionContext(
			execCtx,
			ts.agent.ID,
			ts.sessionKey,
			ts.opts.Dispatch.SessionScope,
		)
		toolResult := ts.agent.Tools.ExecuteWithContext(
			execCtx,
			toolName,
			toolArgs,
			ts.channel,
			ts.chatID,
			asyncCallback,
		)
		if toolResult != nil && toolResult.Async && asyncAckDelivery.ParentHandled {
			toolResult.ResponseHandled = true
		}
		toolDuration := time.Since(toolStart)

		if ts.hardAbortRequested() {
			exec.abortedByHardAbort = true
			return ToolControlBreak
		}

		if al.hooks != nil {
			toolResp, decision := al.hooks.AfterTool(turnCtx, &ToolResultHookResponse{
				Meta:      ts.eventMeta("runTurn", "turn.tool.after"),
				Context:   cloneTurnContext(ts.turnCtx),
				Tool:      toolName,
				Arguments: toolArgs,
				Result:    toolResult,
				Duration:  toolDuration,
			})
			switch decision.normalizedAction() {
			case HookActionContinue, HookActionModify:
				if toolResp != nil {
					if toolResp.Tool != "" {
						toolName = toolResp.Tool
					}
					if toolResp.Result != nil {
						toolResult = toolResp.Result
					}
				}
			case HookActionAbortTurn:
				exec.abortedByHook = true
				return ToolControlBreak
			case HookActionHardAbort:
				_ = ts.requestHardAbort()
				exec.abortedByHardAbort = true
				return ToolControlBreak
			}
		}

		if toolResult == nil {
			toolResult = tools.ErrorResult("hook returned nil tool result")
		}

		toolSummary := strings.TrimSpace(toolResult.ForUser)
		if toolSummary != "" {
			exec.actionLog = appendTurnActionRecord(exec.actionLog, "tool_result", toolName, toolSummary, toolResult.IsError)
		}

		if ts.opts.SuppressToolUserDelivery {
			toolResult.ResponseHandled = false
		}

		if !ts.opts.SuppressToolUserDelivery && toolResult.ResponseHandled {
			attachments, delivered, err := al.deliverToolResultToUser(ctx, ts, toolResult, toolName)
			if err != nil {
				toolResult = tools.ErrorResult(fmt.Sprintf("failed to deliver attachment: %v", err)).WithError(err)
			} else if delivered {
				handledAttachments = append(handledAttachments, attachments...)
			} else if len(toolResultMediaRefs(toolResult)) > 0 {
				toolResult.ResponseHandled = false
			}
		}

		if len(toolResult.Media) > 0 && !toolResult.ResponseHandled {
			recordCompletionMedia(exec, al.mediaStore, toolResult.Media)
			toolResult.ArtifactTags = buildArtifactTags(al.mediaStore, toolResult.Media)
		}

		if !toolResult.ResponseHandled {
			exec.allResponsesHandled = false
		}

		shouldSendForUser := !toolResult.ResponseHandled &&
			!ts.opts.SuppressToolUserDelivery &&
			!toolResult.Silent &&
			toolResult.ForUser != "" &&
			ts.opts.SendResponse
		if shouldSendForUser {
			al.bus.PublishOutbound(ctx, outboundMessageForTurn(ts, toolResult.ForUser))
			logger.DebugCF("agent", "Sent tool result to user",
				map[string]any{
					"tool":        toolName,
					"content_len": len(toolResult.ForUser),
				})
		}
		contentForLLM := toolResult.ContentForLLM()

		if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
			contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
		}

		toolResultMsg := providers.Message{
			Role:       "tool",
			Content:    contentForLLM,
			ToolCallID: toolCallID,
		}
		if len(toolResult.Media) > 0 && !toolResult.ResponseHandled {
			toolResultMsg.Media = append(toolResultMsg.Media, toolResult.Media...)
		}
		al.emitEvent(
			runtimeevents.KindAgentToolExecEnd,
			ts.eventMeta("runTurn", "turn.tool.end"),
			ToolExecEndPayload{
				Tool:       toolName,
				Duration:   toolDuration,
				ForLLMLen:  len(contentForLLM),
				ForUserLen: len(toolResult.ForUser),
				IsError:    toolResult.IsError,
				Async:      toolResult.Async,
			},
		)
		ts.recordToolExecution(
			toolName,
			!toolResult.IsError,
			toolErrorSummary(toolResult),
			inferSkillNamesFromToolCall(ts, toolName, toolArgs),
		)

		if toolResult.IsError {
			errSummary := toolErrorSummary(toolResult)
			if isFatalMCPTransportErrorSummary(errSummary) {
				if mcpServerName != "" {
					logger.WarnCF("agent", "Fatal MCP server transport error; aborting turn to avoid workaround loop",
						map[string]any{
							"agent_id":   ts.agent.ID,
							"iteration":  iteration,
							"tool":       toolName,
							"mcp_server": mcpServerName,
							"error":      errSummary,
							"session_id": ts.sessionKey,
						})
					exec.finalContent = fatalMCPServerErrorReply(mcpServerName, toolName)
					exec.allResponsesHandled = false
					messages = append(messages, toolResultMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
						ts.recordPersistedMessage(toolResultMsg)
						ts.ingestMessage(turnCtx, al, toolResultMsg)
					}
					exec.messages = messages
					return ToolControlBreak
				}
				streak := ts.recentToolExecutionErrorStreak(toolName, func(rec ToolExecutionRecord) bool {
					return isFatalMCPTransportErrorSummary(rec.ErrorSummary)
				})
				if streak >= repeatedFatalToolErrorStreakLimit {
					logger.WarnCF("agent", "Repeated fatal tool transport errors; aborting turn to avoid retry loop",
						map[string]any{
							"agent_id":   ts.agent.ID,
							"iteration":  iteration,
							"tool":       toolName,
							"error":      errSummary,
							"streak":     streak,
							"session_id": ts.sessionKey,
						})
					exec.finalContent = repeatedFatalToolErrorReply(toolName)
					exec.allResponsesHandled = false
					messages = append(messages, toolResultMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
						ts.recordPersistedMessage(toolResultMsg)
						ts.ingestMessage(turnCtx, al, toolResultMsg)
					}
					exec.messages = messages
					return ToolControlBreak
				}
			}
		}
		messages = append(messages, toolResultMsg)
		if !ts.opts.NoHistory {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
			ts.recordPersistedMessage(toolResultMsg)
			ts.ingestMessage(turnCtx, al, toolResultMsg)
		}

		if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
			exec.markSteeringObserved()
			exec.pendingMessages = append(exec.pendingMessages, steerMsgs...)
		}

		skipReason := ""
		skipMessage := ""
		if len(exec.pendingMessages) > 0 {
			skipReason = "queued user steering message"
			skipMessage = "Skipped due to queued user message."
		} else if gracefulPending, _ := ts.gracefulInterruptRequested(); gracefulPending {
			skipReason = "graceful interrupt requested"
			skipMessage = "Skipped due to graceful interrupt."
		}

		if skipReason != "" {
			remaining := len(normalizedToolCalls) - i - 1
			if remaining > 0 {
				logger.InfoCF("agent", "Turn checkpoint: skipping remaining tools",
					map[string]any{
						"agent_id":  ts.agent.ID,
						"completed": i + 1,
						"skipped":   remaining,
						"reason":    skipReason,
					})
				for j := i + 1; j < len(normalizedToolCalls); j++ {
					skippedTC := normalizedToolCalls[j]
					al.emitEvent(
						runtimeevents.KindAgentToolExecSkipped,
						ts.eventMeta("runTurn", "turn.tool.skipped"),
						ToolExecSkippedPayload{
							Tool:   skippedTC.Name,
							Reason: skipReason,
						},
					)
					skippedMsg := providers.Message{
						Role:       "tool",
						Content:    skipMessage,
						ToolCallID: skippedTC.ID,
					}
					messages = append(messages, skippedMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, skippedMsg)
						ts.recordPersistedMessage(skippedMsg)
					}
				}
			}
			break toolLoop
		}

		if ts.pendingResults != nil {
			select {
			case result, ok := <-ts.pendingResults:
				if ok && result != nil && result.ForLLM != "" {
					content := al.cfg.FilterSensitiveData(result.ForLLM)
					msg := subTurnResultPromptMessage(content)
					messages = append(messages, msg)
					ts.agent.Sessions.AddFullMessage(ts.sessionKey, msg)
				}
			default:
			}
		}
	}

	exec.messages = messages

	// Continue if pending steering exists (regardless of allResponsesHandled).
	// This covers the case where tools were partially executed and skipped due to steering,
	// but one tool had ResponseHandled=false (so allResponsesHandled=false).
	if len(exec.pendingMessages) > 0 {
		exec.markAdditionalUserInputObserved()
		logger.InfoCF("agent", "Pending steering after partial tool execution; continuing turn",
			map[string]any{
				"agent_id":            ts.agent.ID,
				"pending_count":       len(exec.pendingMessages),
				"allResponsesHandled": exec.allResponsesHandled,
			})
		exec.allResponsesHandled = false
		return ToolControlContinue
	}

	// Poll for newly arrived steering
	if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
		exec.markSteeringObserved()
		logger.InfoCF("agent", "Steering arrived after tool delivery; continuing turn",
			map[string]any{
				"agent_id":       ts.agent.ID,
				"steering_count": len(steerMsgs),
			})
		exec.pendingMessages = append(exec.pendingMessages, steerMsgs...)
		exec.allResponsesHandled = false
		return ToolControlContinue
	}

	// No pending steering: finalize or break depending on allResponsesHandled
	if shouldFinalizeAfterToolLoopWithRender(al, exec) {
		logger.InfoCF("agent", "Tool loop completed; rendering terminal reply from accumulated turn context",
			map[string]any{
				"agent_id":   ts.agent.ID,
				"iteration":  iteration,
				"tool_count": len(normalizedToolCalls),
			})
		return ToolControlFinalize
	}

	if exec.allResponsesHandled {
		summaryMsg := providers.Message{
			Role:        "assistant",
			Content:     handledToolResponseSummary,
			Attachments: append([]providers.Attachment(nil), handledAttachments...),
		}
		if !ts.opts.NoHistory {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, summaryMsg)
			ts.recordPersistedMessage(summaryMsg)
			ts.ingestMessage(turnCtx, al, summaryMsg)
			if err := ts.agent.Sessions.Save(ts.sessionKey); err != nil {
				logger.WarnCF("agent", "Failed to save session after tool delivery",
					map[string]any{
						"agent_id": ts.agent.ID,
						"error":    err.Error(),
					})
			}
		}
		if ts.opts.EnableSummary {
			al.contextManager.Compact(turnCtx, &CompactRequest{
				SessionKey: ts.sessionKey,
				Reason:     ContextCompressReasonSummarize,
				Budget:     ts.agent.ContextWindow,
			})
		}
		ts.setPhase(TurnPhaseCompleted)
		ts.setFinalContent("")
		if al.channelManager != nil && ts.channel != "" {
			al.channelManager.DismissToolFeedback(ctx, ts.channel, ts.chatID, ts.opts.InboundContext)
		}
		logger.InfoCF("agent", "Tool output satisfied delivery; ending turn without follow-up LLM",
			map[string]any{
				"agent_id":   ts.agent.ID,
				"iteration":  iteration,
				"tool_count": len(normalizedToolCalls),
			})
		return ToolControlBreak
	}

	// allResponsesHandled=false and no pending steering: continue so coordinator
	// makes another LLM call. The tool result is in messages and the LLM will
	// return it as finalContent in the next iteration.
	ts.agent.Tools.TickTTL()
	logger.DebugCF("agent", "TTL tick after tool execution", map[string]any{
		"agent_id": ts.agent.ID, "iteration": iteration,
	})
	return ToolControlContinue
}
