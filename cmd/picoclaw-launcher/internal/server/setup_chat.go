package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// setupChat holds the agent loop used for AI-guided configuration.
type setupChat struct {
	mu        sync.Mutex
	agentLoop *agent.AgentLoop
	msgBus    *bus.MessageBus
	provider  providers.LLMProvider
}

var (
	activeSetupChat   *setupChat
	activeSetupChatMu sync.Mutex
)

// RegisterChatAPI registers the AI-guided configuration chat endpoint.
func RegisterChatAPI(mux *http.ServeMux, absPath string) {
	mux.HandleFunc("POST /api/setup/chat", func(w http.ResponseWriter, r *http.Request) {
		handleSetupChat(w, r, absPath)
	})
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Success  bool   `json:"success"`
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

func handleSetupChat(w http.ResponseWriter, r *http.Request, absPath string) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	sc, err := getOrCreateSetupChat(absPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to initialize chat: %v", err),
		})
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	resp, err := sc.agentLoop.ProcessDirect(ctx, req.Message, "cli:setup")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Success: false,
			Error:   fmt.Sprintf("Chat error: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse{
		Success:  true,
		Response: resp,
	})
}

func getOrCreateSetupChat(absPath string) (*setupChat, error) {
	activeSetupChatMu.Lock()
	defer activeSetupChatMu.Unlock()

	if activeSetupChat != nil {
		return activeSetupChat, nil
	}

	cfg, err := config.LoadConfig(absPath)
	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("provider creation failed: %w", err)
	}
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Prime the agent with a setup-assistant system context via first message.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	setupPrompt := buildChatSetupPrompt(absPath, cfg)
	_, _ = agentLoop.ProcessDirect(ctx, setupPrompt, "cli:setup")

	activeSetupChat = &setupChat{
		agentLoop: agentLoop,
		msgBus:    msgBus,
		provider:  provider,
	}

	return activeSetupChat, nil
}

func buildChatSetupPrompt(configPath string, cfg *config.Config) string {
	return fmt.Sprintf(`You are PicoClaw's setup assistant. The user just completed initial API setup.
Config file: %s
Current model: %s

Your role is to help them configure PicoClaw step by step:

1. **Communication Channels** — Telegram bot, Discord bot, WeChat, Slack, etc.
   Ask which channels they want and guide them to get bot tokens.

2. **Agent Identity** — Help create/edit SOUL.md (personality), IDENTITY.md (name/description), USER.md (user preferences) in the workspace.

3. **Tools & Skills** — web search, MCP servers, custom skills.

4. **Advanced settings** — scheduling, cron jobs, memory tuning.

You can read and modify the config file using your file tools.
Start by welcoming the user and asking what they'd like to set up.
Keep responses concise. Use Chinese if the user writes in Chinese.`, configPath, cfg.Agents.Defaults.GetModelName())
}
