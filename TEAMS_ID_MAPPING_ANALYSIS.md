# Teams Channel Integration & ID Mapping Analysis

## Executive Summary

**Teams Channel Implementation Status**: ❌ **NOT YET IMPLEMENTED**
- Search results show no Teams/MSTeams channel in `pkg/channels/`
- Only reference found: migration config reference in `pkg/migrate/sources/openclaw/openclaw_config.go:123`
- **Foundry Integration**: Only implemented as an LLM **provider** (Azure AI Foundry), not as a channel

---

## InboundMessage Structure (Bus Layer)

**Location**: [pkg/bus/types.go](pkg/bus/types.go)

### Core Fields Available

```go
type InboundMessage struct {
	Channel    string            // Channel name (e.g., "teams", "slack", "telegram")
	SenderID   string            // Platform-specific sender identifier
	Sender     SenderInfo        // Structured sender information
	ChatID     string            // Conversation/chat identifier (CRITICAL FOR ISOLATION)
	Content    string            // Message text content
	Media      []string          // Media references (attachments)
	Peer       Peer              // Routing peer information
	MessageID  string            // Platform-specific message ID
	MediaScope string            // Media lifecycle tracking scope
	SessionKey string            // Session key (optional, can be auto-resolved)
	Metadata   map[string]string // Platform-specific metadata
}
```

### SenderInfo Sub-structure

```go
type SenderInfo struct {
	Platform    string // "telegram", "discord", "slack", "teams", etc.
	PlatformID  string // Raw platform ID (e.g., Teams UserID "29:...")
	CanonicalID string // Normalized "platform:id" format (e.g., "teams:29:...")
	Username    string // Display username (e.g., "@alice")
	DisplayName string // Full display name
}
```

### Peer Sub-structure

```go
type Peer struct {
	Kind string // "direct" | "group" | "channel" | ""
	ID   string // Peer identifier (user_id, group_id, channel_id, etc.)
}
```

---

## ID Mapping for Hypothetical Teams Implementation

### What Teams Would Need to Provide

If Teams were to be integrated, the following IDs should map as follows:

| Teams ID | InboundMessage Field | Notes |
|----------|----------------------|-------|
| User ID (e.g., `29:1ABC123`) | `SenderID`, `Sender.PlatformID` | Teams uses format `29:uuid` |
| Conversation ID | `ChatID` | CRITICAL: Identifies conversation scope |
| Team ID | `Metadata["team_id"]`, potentially routing input | Can be used for team-level routing |
| Channel ID | `Peer.ID` (if channel) | When in Team channel |
| Service URL | `Metadata["service_url"]` | Teams service endpoint |
| Activity ID | `MessageID` | Platform message identifier |

### Canonical ID Format

**Pattern**: `platform:platform_id`

**Example for Teams**: 
```
"teams:29:1ABC123" = Canonical ID for Teams user 29:1ABC123
```

Built via: [pkg/identity/identity.go](pkg/identity/identity.go)
```go
func BuildCanonicalID(platform, platformID string) string {
	p := strings.ToLower(strings.TrimSpace(platform))
	id := strings.TrimSpace(platformID)
	if p == "" || id == "" {
		return ""
	}
	return p + ":" + id  // "teams:29:abc123"
}
```

---

## ChatID Usage & Session Isolation

**Location**: [pkg/agent/loop.go](pkg/agent/loop.go#L1250-L1270)

### Current ChatID Role

The `ChatID` field is **THE PRIMARY KEY** for conversation isolation:

1. **Session Binding**: Each unique `ChatID` can map to a separate session depending on DMScope
2. **Workspace Isolation**: When non-empty and not "direct", creates isolated agent workspace:
   ```go
   if isolationID != "" && isolationID != "direct" {
       // Create transient isolated instance for this chat session
       agent = NewAgentInstance(ac, cfg, baseAgent.Provider, isolationID)
   }
   ```
3. **State Persistence**: Last ChatID tracked for workspace continuity

**Example mapping**:
- Single direct message with user → `ChatID = "teams:29:1ABC123"`
- Team channel conversation → `ChatID = "teams-channel:xyz789"`
- Group chat → `ChatID = "teams-groupchat:123abc"`

---

## Session Key Construction & Resolution

**Location**: [pkg/routing/session_key.go](pkg/routing/session_key.go) + [pkg/routing/route.go](pkg/routing/route.go)

### RouteInput (What Channel Provides to Router)

```go
type RouteInput struct {
	Channel    string      // "teams" (if implemented)
	AccountID  string      // Bot account/app ID
	Peer       *RoutePeer  // Who message is from (user)
	ParentPeer *RoutePeer  // Parent context (e.g., Team)
	GuildID    string      // Guild/workspace ID (if applicable)
	TeamID     string      // Teams Team ID (would go here)
}
```

### ResolvedRoute Output

```go
type ResolvedRoute struct {
	AgentID        string // Which agent handles this message
	SessionKey     string // Session identifier pattern
	MainSessionKey string // Main session fallback
	MatchedBy      string // How routing was matched
}
```

### Session Key Patterns

**DMScope** configuration determines how sessions are keyed:

| DMScope Mode | Format | Example | Use Case |
|--------------|--------|---------|----------|
| `DMScopeMain` | `agent:agentid:main` | `agent:teams-bot:main` | Single shared session |
| `DMScopePerPeer` | `agent:agentid:direct:peerid` | `agent:teams-bot:direct:user123` | Per-user sessions |
| `DMScopePerChannelPeer` | `agent:agentid:channel:direct:peerid` | `agent:teams-bot:teams:direct:user123` | Per-channel-per-user |
| `DMScopePerAccountChannelPeer` | `agent:agentid:channel:account:direct:peerid` | `agent:teams-bot:teams:acct1:direct:user123` | Per-account-channel-user |

**Location**: [pkg/routing/session_key.go:40-100](pkg/routing/session_key.go#L40-L100)

```go
// For Teams direct message:
BuildAgentPeerSessionKey(SessionKeyParams{
	AgentID:   "teams-bot",
	Channel:   "teams",
	AccountID: "bot-app-id",
	Peer:      &RoutePeer{Kind: "direct", ID: "29:abc123"},
	DMScope:   DMScopePerChannelPeer,
})
// Returns: "agent:teams-bot:teams:direct:29:abc123"
```

---

## ID Priority Cascade for Agent Routing

**Location**: [pkg/routing/route.go:68-126](pkg/routing/route.go#L68-L126)

The agent resolver uses this **7-level priority**:

1. **Peer binding** → Match on specific user/peer ID
2. **Parent peer binding** → Match on parent context (Team, Guild, etc.)
3. **Guild binding** → Match on Guild/Workspace ID
4. **Team binding** → Match on Team ID ← **TEAMS WOULD USE THIS**
5. **Account binding** → Match on account/app ID
6. **Channel wildcard** → Match on channel with wildcard
7. **Default agent** → Fallback

**For Teams, routing would likely use**:
- Level 2: ParentPeer = Team
- Level 3: GuildID = Team ID  
- Level 4: TeamID = Team ID

---

## Foundry Integration Status

**Locations**: 
- [pkg/providers/factory_provider.go:196](pkg/providers/factory_provider.go#L196)
- [pkg/providers/openai_compat/provider.go:432](pkg/providers/openai_compat/provider.go#L432)

### Current Foundry Support

**Type**: LLM **Provider Only** (NOT Channel)

```go
case "azure-ai", "azure-foundry":
    // Azure AI Foundry / Studio compatible with OpenAI API format
    // Used for LLM backend, not message channeling
```

**What's Missing for Teams/Foundry Integration**:
- ❌ No Teams Channel handler
- ❌ No Foundry Agent channel integration
- ❌ No Teams webhook receiver
- ❌ No Teams message routing

**What Exists**:
- ✅ Azure AI Foundry as LLM provider backend
- ✅ OpenAI-compatible API handling
- ✅ Generic inbound message bus infrastructure

---

## Metadata Field Usage

All channels populate `InboundMessage.Metadata` with platform-specific data:

### Example: WeCom (for comparison)
**Location**: [pkg/channels/wecom/app.go:605-620](pkg/channels/wecom/app.go#L605-L620)

```go
metadata := map[string]string{
	"msg_type":    msg.MsgType,
	"msg_id":      fmt.Sprintf("%d", msg.MsgId),
	"agent_id":    fmt.Sprintf("%d", msg.AgentID),
	"platform":    "wecom",
	"media_id":    msg.MediaId,
	"create_time": fmt.Sprintf("%d", msg.CreateTime),
}
```

### For Teams Implementation, Would Include:

```go
metadata := map[string]string{
	"team_id":        msg.TeamsTeamID,
	"channel_id":     msg.TeamsChannelID,
	"service_url":    msg.ServiceURL,
	"activity_id":    msg.ActivityID,
	"conversation_id": msg.ConversationID,
	"from_user_id":   msg.FromUserID,
	"platform":       "teams",
	...
}
```

---

## Identity Matching System

**Location**: [pkg/identity/identity.go](pkg/identity/identity.go)

The framework provides legacy-compatible and modern identity matching:

### Allowed Formats in Config

```yaml
allow_from:
  - "29:abc123"              # Raw Teams user ID
  - "teams:29:abc123"        # Canonical format
  - "@alice"                 # Username format
  - "29:abc123|alice"        # Compound format
```

### Matching Logic

```go
func MatchAllowed(sender bus.SenderInfo, allowed string) bool {
	// 1. Try canonical "platform:id" first
	if platform, id, ok := ParseCanonicalID(allowed); ok {
		if sender.CanonicalID == BuildCanonicalID(platform, id) {
			return true
		}
	}
	
	// 2. Fall back to PlatformID or Username
	if sender.PlatformID == allowed { return true }
	if sender.Username == "@" + allowed { return true }
	
	return false
}
```

---

## What a Teams Channel Implementation Would Need

### Minimum Required Fields in InboundMessage

```go
InboundMessage{
	Channel: "teams",
	SenderID: userID,  // Teams: "29:uuid"
	Sender: bus.SenderInfo{
		Platform:    "teams",
		PlatformID:  userID,  // "29:uuid"
		CanonicalID: "teams:29:uuid",
		Username:    userName,
		DisplayName: displayName,
	},
	ChatID: conversationID,  // Teams ConversationReference.conversation_id
	Content: messageContent,
	Peer: bus.Peer{
		Kind: "direct" || "channel",
		ID: channelID || userID,
	},
	MessageID: activityID,  // Teams Activity ID
	Metadata: map[string]string{
		"team_id": teamID,
		"channel_id": channelID,
		"service_url": serviceURL,
		// ... other Teams-specific fields
	},
}
```

### Routing Setup in Config

```yaml
agents:
  routing:
    - agent_id: "teams-agent"
      match:
        channel: "teams"
        team_id: "team-xyz"        # Route by Teams Team ID
```

---

## Key Takeaways for Teams + Foundry Integration

1. **Framework is Ready**: GenericBus message structure can handle Teams IDs
2. **ChatID is Primary**: Use Teams `ConversationReference.conversation_id` as ChatID for isolation
3. **SessionKey Auto-Generated**: Routing + DMScope automatically creates session keys
4. **Identity System Ready**: Canonical "teams:29:uuid" format supported
5. **No Channel Implementation Yet**: Need to implement webhook receiver + message publisher
6. **Foundry is Provider Only**: Currently only LLM backend, not messaging channel
7. **User ID Format**: Teams uses `29:uuid` format - should populate both PlatformID and CanonicalID
8. **Conversation Scope**: Teams conversation_id maps directly to InboundMessage.ChatID

---

## Reference Architecture Files

| Component | File | Key Types |
|-----------|------|-----------|
| Bus Types | [pkg/bus/types.go](pkg/bus/types.go) | InboundMessage, SenderInfo, Peer |
| Routing | [pkg/routing/route.go](pkg/routing/route.go) | RouteInput, ResolvedRoute |
| Session Keys | [pkg/routing/session_key.go](pkg/routing/session_key.go) | SessionKeyParams, DM scopes |
| Identity | [pkg/identity/identity.go](pkg/identity/identity.go) | BuildCanonicalID, MatchAllowed |
| Agent Loop | [pkg/agent/loop.go](pkg/agent/loop.go) | Message processing, session isolation |
| Example Channel | [pkg/channels/wecom/app.go](pkg/channels/wecom/app.go) | Channel implementation pattern |
