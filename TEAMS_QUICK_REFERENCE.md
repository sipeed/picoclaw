# Quick Reference: Teams Integration Questions

## Q1: Teams Channel Integration - Message Receiving & Processing

**Status**: ❌ NOT IMPLEMENTED

**Where it would go**: `pkg/channels/teams/` (currently doesn't exist)

**Current Similar Implementation**: See [pkg/channels/wecom/app.go](pkg/channels/wecom/app.go) for webhook pattern

**Expected Pattern**:
1. HTTP webhook receiver on configured port
2. Verify Teams Bot Framework signature
3. Parse activity/message payload
4. Build `InboundMessage` struct
5. Publish to bus via `channel.HandleMessage()` or `messageBus.PublishInbound()`

**Key Files to Reference**:
- [pkg/channels/base.go](pkg/channels/base.go) - Base channel interface
- [pkg/channels/manager.go](pkg/channels/manager.go) - Channel registration/lifecycle
- [pkg/channels/wecom/app.go:605-650](pkg/channels/wecom/app.go#L605-L650) - HandleMessage pattern

---

## Q2: InboundMessage Structure - All Available Fields

**Location**: [pkg/bus/types.go:18-35](pkg/bus/types.go#L18-L35)

### Complete Field List

| Field | Type | Purpose | Example |
|-------|------|---------|---------|
| `Channel` | string | Platform identifier | `"teams"` |
| `SenderID` | string | Raw user ID | `"29:1ABC123"` |
| `Sender` | SenderInfo | Structured identity | (see below) |
| `Sender.Platform` | string | Platform name | `"teams"` |
| `Sender.PlatformID` | string | User platform ID | `"29:1ABC123"` |
| `Sender.CanonicalID` | string | **Normalized format** | `"teams:29:1abc123"` |
| `Sender.Username` | string | Handle/username | `"alice"` |
| `Sender.DisplayName` | string | Full display name | `"Alice Smith"` |
| `ChatID` | string | **Conversation ID (PRIMARY)** | `"teams-conv-abc123"` |
| `Content` | string | Message text | `"Hello world"` |
| `Media` | []string | Media references | `["media://ref123"]` |
| `Peer.Kind` | string | Peer type | `"direct"` \| `"channel"` |
| `Peer.ID` | string | Peer ID | User/channel ID |
| `MessageID` | string | Platform message ID | `"activity-123"` |
| `MediaScope` | string | Media cleanup scope | `"teams:conv-abc123:msg-123"` |
| `SessionKey` | string | **Session identifier** | `"agent:bot:teams:direct:29:abc123"` |
| `Metadata` | map | Platform-specific data | (see below) |

### Metadata Map (Platform-Specific)

```go
metadata := map[string]string{
	"team_id":           "T12345",
	"channel_id":        "C12345", 
	"conversation_id":   "19:...",
	"service_url":       "https://smba.trafficmanager.net/...",
	"activity_id":       "...",
	"from_user_id":      "29:...",
	"from_user_name":    "alice",
	"recipient_id":      "28:...",
	"conversation_type": "personal|groupChat|channel",
	"platform":          "teams",
	// ... any other Teams-specific fields
}
```

---

## Q3: Unique User/Conversation ID Capture from Teams

### What Teams Provides vs. What PicoClaw Needs

**Teams → PicoClaw Mapping**:

```
Teams Activity Object
├── from.id → SenderID (raw), Sender.PlatformID
├── from.aadObjectId → (optional, use if available)
├── conversation.id → ChatID (THE KEY FIELD)
├── conversation.tenantId → Metadata["tenant_id"]
├── channelData.teamsChannelId → Peer.ID (if channel)
├── channelData.teamsTeamId → Metadata["team_id"], routing input
├── serviceUrl → Metadata["service_url"]
└── id → MessageID
```

### ID Construction

**User Identity Chain**:
```
Teams: from.id = "29:U123ABC"
  ↓
Stored as: SenderID = "29:U123ABC"
Stored as: Sender.PlatformID = "29:U123ABC"
Normalized as: Sender.CanonicalID = "teams:29:u123abc" (lowercased)
```

**Conversation Identity Chain**:
```
Teams: conversation.id = "19:abc123@thread.v2"
  ↓
Stored as: ChatID = "19:abc123@thread.v2" (conversation scope)
Used for: Session isolation, message routing, state persistence
```

**Team Identity Chain**:
```
Teams: channelData.teamsTeamId = "T12345678"
  ↓
Stored as: Metadata["team_id"] = "T12345678"
  ↓
Used in: Routing cascade (Level 4), agent selection
```

### Canonical ID Format

Built by [pkg/identity/identity.go:BuildCanonicalID()](pkg/identity/identity.go#L11-L20):

```go
BuildCanonicalID("teams", "29:U123ABC")
// Returns: "teams:29:u123abc" (normalized to lowercase)
```

**Used for**:
- Access control matching
- Cross-platform user linking (via identity_links in config)
- User identity validation

---

## Q4: Foundry Agent Integration Points & ID Provision

**Status**: ⚠️ PARTIAL - Foundry is an LLM Provider, NOT a Channel

### Current Foundry Support

**Location**: [pkg/providers/factory_provider.go:196](pkg/providers/factory_provider.go#L196)

Foundry is integrated **only as LLM backend** (OpenAI-compatible API):

```go
case "azure-ai", "azure_foundry":
    // Use for model calls, not messaging
```

**What Foundry Would Provide (if implemented as channel)**:
- Foundry Agent service/conversation IDs
- Foundry user session tracking
- Foundry-specific message format

**What's MISSING**:
1. ❌ Foundry Agent channel receiver
2. ❌ Foundry conversation → ChatID mapping
3. ❌ Foundry agent ID → Agent routing

### If Foundry Channel Were to Exist

Expected `InboundMessage` would be:

```go
InboundMessage{
	Channel: "foundry-agent",
	SenderID: foundryUserID,
	Sender: SenderInfo{
		Platform:    "foundry",
		PlatformID:  foundryUserID,
		CanonicalID: "foundry:" + foundryUserID,
		DisplayName: userName,
	},
	ChatID: foundryConversationID,  // Critical for isolation
	Content: message,
	Metadata: map[string]string{
		"foundry_agent_id":         agentID,
		"foundry_conversation_id":  conversationID,
		"foundry_message_id":       messageID,
		"platform":                 "foundry",
		// ... other Foundry fields
	},
}
```

### Foundry ID Mapping Table (Hypothetical)

| Foundry ID | InboundMessage Field | Purpose |
|----------|----------------------|---------|
| Agent ID | Routing/Config | Which agent handles |
| User ID | SenderID | Who sent message |
| Conversation ID | **ChatID** | Session isolation |
| Message ID | MessageID | For threading |
| Service Endpoint | Metadata | For API calls |

---

## Q5: How ChatID is Currently Used for Session ID Association

**Location**: [pkg/agent/loop.go:1248-1270](pkg/agent/loop.go#L1248-L1270)

### ChatID → SessionKey Conversion

**Process**:

```
1. InboundMessage arrives with ChatID
   ↓
2. Router resolves agent (via RouteInput)
   ↓
3. SessionKey built from:
   - Agent ID
   - Channel name
   - Peer information (ChatID wrapped as Peer.ID)
   - DMScope configuration
   ↓
4. Result: SessionKey = "agent:botname:team:type:id"
   ↓
5. SessionKey used to find/create workspace & history
```

### Session Key Patterns by DMScope

**From config `session.dm_scope`**:

| Setting | Session Behavior | Key Format |
|---------|------------------|-----------|
| Not set / `main` | Single shared session | `agent:bot:main` |
| `per_peer` | One session per user | `agent:bot:direct:user123` |
| `per_channel_peer` | One per channel+user | `agent:bot:teams:direct:user123` |
| `per_account_channel_peer` | One per account+channel+user | `agent:bot:teams:act1:direct:user123` |

**Code Reference**: [pkg/routing/session_key.go:40-100](pkg/routing/session_key.go#L40-L100)

### Session Isolation via ChatID

When ChatID is unique and non-"direct":

```go
// From pkg/agent/loop.go:1248-1270
if isolationID != "" && isolationID != "direct" {
	// Creates isolated agent instance with separate:
	// - Workspace directory
	// - Session history
	// - Memory storage
	// - State
	agent = NewAgentInstance(ac, cfg, baseAgent.Provider, isolationID)
}
```

**Isolation Example**:

```
ChatID = "teams-channel-abc123"
  ↓
Creates: workspace/teams-channel-abc123/
  ├── sessions/
  ├── memory/
  ├── skills/
  └── state/
  ↓
Each channel conversation has completely isolated history
```

### State Persistence

Tracks last ChatID:

```go
// Record last chat for workspace continuity
al.RecordLastChatID(chatID)  // pkg/agent/loop.go
```

Stored in: `workspace/state/state.json`:
```json
{
  "last_channel": "teams",
  "last_chat_id": "19:abc123@thread.v2",
  "timestamp": "2025-03-26T10:00:00Z"
}
```

---

## Summary Table: ID Field Mapping

| Concept | Field | Example | Used For |
|---------|-------|---------|----------|
| **User** | `SenderID` + `Sender.PlatformID` | `"29:U123ABC"` | Message author |
| **User (Normalized)** | `Sender.CanonicalID` | `"teams:29:u123abc"` | Access control |
| **Conversation** | `ChatID` | `"19:abc123@thread.v2"` | **Session isolation** |
| **Team** | `Metadata["team_id"]` | `"T12345678"` | Agent routing level |
| **Channel** | `Peer.Kind` + `Peer.ID` | `"channel:C12345"` | Routing peer |
| **Message** | `MessageID` | `"activity-123"` | Threading, dedup |
| **Workspace** | Derived from ChatID | `workspace/19:abc123@thread.v2/` | Data isolation |
| **Session** | `SessionKey` | `"agent:bot:teams:direct:29:u123abc"` | History tracking |

---

## File Cross-References

### For Teams Implementation
- Start: [pkg/channels/manager.go](pkg/channels/manager.go) - Channel registration
- Reference: [pkg/channels/wecom/app.go](pkg/channels/wecom/app.go) - Full implementation pattern  
- Base: [pkg/channels/base.go](pkg/channels/base.go) - Handler interface

### For Routing/Session
- Routing: [pkg/routing/route.go](pkg/routing/route.go) - 7-level cascade
- Keys: [pkg/routing/session_key.go](pkg/routing/session_key.go) - Key building
- Isolation: [pkg/agent/loop.go:1248+](pkg/agent/loop.go#L1248) - ChatID isolation

### For Identity
- Identity: [pkg/identity/identity.go](pkg/identity/identity.go) - CanonicalID logic
- Matching: Lines 28-100 - Access control matching

### For State
- State: [pkg/state/state.go](pkg/state/state.go) - LastChatID persistence
