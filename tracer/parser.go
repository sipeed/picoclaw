package tracer

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// logEntry maps the JSON fields we care about from each gateway log line.
type logEntry struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Time      string `json:"time"`
	EventKind string `json:"event_kind"`
	AgentID   string `json:"agent_id"`

	// Turn scope
	TurnID     string `json:"turn_id"`
	Channel    string `json:"channel"`
	ChatID     string `json:"chat_id"`
	SenderID   string `json:"sender_id"`
	SessionKey string `json:"session_key"`
	UserLen    int    `json:"user_len"`

	// LLM request
	Iteration       int     `json:"iteration"`
	Model           string  `json:"model"`
	Messages        int     `json:"messages"`
	Tools           int     `json:"tools"`
	MaxTokens       int     `json:"max_tokens"`
	SystemPromptLen int     `json:"system_prompt_len"`
	Temperature     float64 `json:"temperature"`
	MessagesJSON    string  `json:"messages_json"`
	ToolsJSON       string  `json:"tools_json"`

	// LLM response
	ContentLen   int  `json:"content_len"`
	ToolCalls    int  `json:"tool_calls"`
	HasReasoning bool `json:"has_reasoning"`

	// Tool exec
	Tool       string `json:"tool"`
	ArgsCount  int    `json:"args_count"`
	DurationMs int    `json:"duration_ms"`
	IsError    bool   `json:"is_error"`
	ForLLMLen  int    `json:"for_llm_len"`

	// Turn end
	Status          string `json:"status"`
	IterationsTotal int    `json:"iterations_total"`
}

func logTimestamp(iso string) string {
	if len(iso) >= 19 {
		return iso[11:19]
	}
	return iso
}

var (
	reMsgHdr  = regexp.MustCompile(`^\s+\[\d+\] Role: (\w+)\s*$`)
	reToolHdr = regexp.MustCompile(`^\s+\[\d+\] Type: function, Name: (\S+)\s*$`)
)

func parseMessages(s string) []Message {
	var out []Message
	var cur *Message
	var lines []string

	flush := func() {
		if cur != nil {
			cur.Content = strings.TrimSpace(strings.Join(lines, "\n"))
			out = append(out, *cur)
			cur, lines = nil, nil
		}
	}

	for _, line := range strings.Split(s, "\n") {
		if m := reMsgHdr.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Message{Role: strings.ToLower(m[1])}
		} else if cur != nil {
			stripped := strings.TrimSpace(line)
			switch stripped {
			case "[", "]", "":
				continue
			}
			if after, ok := strings.CutPrefix(stripped, "Content:"); ok {
				lines = append(lines, strings.TrimSpace(after))
			} else {
				lines = append(lines, strings.TrimRight(line, " \t"))
			}
		}
	}
	flush()
	return out
}

func parseTools(s string) []Tool {
	var out []Tool
	var cur *Tool

	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}

	for _, line := range strings.Split(s, "\n") {
		if m := reToolHdr.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Tool{Name: m[1]}
		} else if cur != nil {
			stripped := strings.TrimSpace(line)
			switch stripped {
			case "[", "]", "":
				continue
			}
			if after, ok := strings.CutPrefix(stripped, "Description:"); ok {
				cur.Description = strings.TrimSpace(after)
			} else if after, ok := strings.CutPrefix(stripped, "Parameters:"); ok {
				cur.Parameters = strings.TrimSpace(after)
			}
		}
	}
	flush()
	return out
}

type llmKey struct {
	turnID    string
	iteration int
}

// ParseLog reads the gateway JSON-Lines log and returns turns newest-first.
func ParseLog(path string) ([]*Turn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	turns := map[string]*Turn{}
	var order []string
	llmIdx := map[llmKey]int{}

	// agent_id → active turn_id (for debug lines that lack turn_id)
	activeTurn := map[string]string{}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		var e logEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		t := logTimestamp(e.Time)

		if e.EventKind != "" {
			switch e.EventKind {

			case "agent.turn.start":
				if e.TurnID == "" {
					continue
				}
				activeTurn[e.AgentID] = e.TurnID
				turns[e.TurnID] = &Turn{
					TurnID:     e.TurnID,
					Timestamp:  t,
					Channel:    e.Channel,
					ChatID:     e.ChatID,
					SenderID:   e.SenderID,
					SessionKey: e.SessionKey,
					UserLen:    e.UserLen,
					Status:     "running",
					LLMCalls:   []LLMCall{},
					ToolExecs:  []ToolExec{},
				}
				order = append(order, e.TurnID)

			case "agent.llm.request":
				turn, ok := turns[e.TurnID]
				if !ok {
					continue
				}
				idx := len(turn.LLMCalls)
				turn.LLMCalls = append(turn.LLMCalls, LLMCall{
					Iteration:     e.Iteration,
					Model:         e.Model,
					MessagesCount: e.Messages,
					ToolsCount:    e.Tools,
					MaxTokens:     e.MaxTokens,
					Timestamp:     t,
					Messages:      []Message{},
					Tools:         []Tool{},
				})
				llmIdx[llmKey{e.TurnID, e.Iteration}] = idx

			case "agent.llm.response":
				turn, ok := turns[e.TurnID]
				if !ok {
					continue
				}
				if idx, ok := llmIdx[llmKey{e.TurnID, e.Iteration}]; ok {
					cl, tc, hr := e.ContentLen, e.ToolCalls, e.HasReasoning
					turn.LLMCalls[idx].ContentLen = &cl
					turn.LLMCalls[idx].ToolCallsCount = &tc
					turn.LLMCalls[idx].HasReasoning = &hr
				}

			case "agent.tool.exec_start":
				turn, ok := turns[e.TurnID]
				if !ok {
					continue
				}
				turn.ToolExecs = append(turn.ToolExecs, ToolExec{
					Tool:      e.Tool,
					ArgsCount: e.ArgsCount,
					Timestamp: t,
				})

			case "agent.tool.exec_end":
				turn, ok := turns[e.TurnID]
				if !ok {
					continue
				}
				for i := len(turn.ToolExecs) - 1; i >= 0; i-- {
					te := &turn.ToolExecs[i]
					if te.Tool == e.Tool && te.DurationMs == nil {
						dur, isErr, forLLM := e.DurationMs, e.IsError, e.ForLLMLen
						te.DurationMs = &dur
						te.IsError = &isErr
						te.ForLLMLen = &forLLM
						break
					}
				}

			case "agent.turn.end":
				turn, ok := turns[e.TurnID]
				if !ok {
					continue
				}
				dur := e.DurationMs
				turn.Status = e.Status
				turn.DurationMs = &dur
				turn.Iterations = e.IterationsTotal
				delete(activeTurn, e.AgentID)
			}
			continue
		}

		// Debug: LLM request — carries system_prompt_len and temperature.
		if e.Message == "LLM request" && e.Level == "debug" {
			agentID := e.AgentID
			if agentID == "" {
				agentID = "main"
			}
			turnID, ok := activeTurn[agentID]
			if !ok {
				continue
			}
			if idx, ok := llmIdx[llmKey{turnID, e.Iteration}]; ok {
				turn := turns[turnID]
				turn.LLMCalls[idx].SystemPromptLen = e.SystemPromptLen
				if e.Temperature != 0 {
					turn.LLMCalls[idx].Temperature = strconv.FormatFloat(e.Temperature, 'f', -1, 64)
				}
			}
			continue
		}

		// Debug: Full LLM request — carries messages_json and tools_json strings.
		if e.Message == "Full LLM request" && e.Level == "debug" {
			for _, turnID := range activeTurn {
				if idx, ok := llmIdx[llmKey{turnID, e.Iteration}]; ok {
					turn := turns[turnID]
					if len(turn.LLMCalls[idx].Messages) == 0 {
						turn.LLMCalls[idx].Messages = parseMessages(e.MessagesJSON)
						turn.LLMCalls[idx].Tools = parseTools(e.ToolsJSON)
					}
					break
				}
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	result := make([]*Turn, len(order))
	for i, id := range order {
		result[len(order)-1-i] = turns[id]
	}
	return result, nil
}
