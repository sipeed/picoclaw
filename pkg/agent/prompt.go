package agent

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type PromptLayer string

const (
	PromptLayerKernel      PromptLayer = "kernel"
	PromptLayerInstruction PromptLayer = "instruction"
	PromptLayerCapability  PromptLayer = "capability"
	PromptLayerContext     PromptLayer = "context"
	PromptLayerTurn        PromptLayer = "turn"
)

type PromptSlot string

const (
	PromptSlotIdentity     PromptSlot = "identity"
	PromptSlotHierarchy    PromptSlot = "hierarchy"
	PromptSlotWorkspace    PromptSlot = "workspace"
	PromptSlotTooling      PromptSlot = "tooling"
	PromptSlotMCP          PromptSlot = "mcp"
	PromptSlotSkillCatalog PromptSlot = "skill_catalog"
	PromptSlotActiveSkill  PromptSlot = "active_skill"
	PromptSlotMemory       PromptSlot = "memory"
	PromptSlotRuntime      PromptSlot = "runtime"
	PromptSlotSummary      PromptSlot = "summary"
	PromptSlotMessage      PromptSlot = "message"
	PromptSlotSteering     PromptSlot = "steering"
	PromptSlotSubTurn      PromptSlot = "subturn"
	PromptSlotInterrupt    PromptSlot = "interrupt"
	PromptSlotOutput       PromptSlot = "output"
)

type PromptSourceID string

const (
	PromptSourceKernel         PromptSourceID = "runtime.kernel"
	PromptSourceHierarchy      PromptSourceID = "runtime.hierarchy"
	PromptSourceWorkspace      PromptSourceID = "workspace.definition"
	PromptSourceRuntime        PromptSourceID = "runtime.context"
	PromptSourceSummary        PromptSourceID = "context.summary"
	PromptSourceMemory         PromptSourceID = "memory:workspace"
	PromptSourceSkillCatalog   PromptSourceID = "skill:index"
	PromptSourceActiveSkills   PromptSourceID = "skill:active"
	PromptSourceToolRegistry   PromptSourceID = "tool_registry:native"
	PromptSourceToolDiscovery  PromptSourceID = "tool_registry:discovery"
	PromptSourceOutputPolicy   PromptSourceID = "runtime.output"
	PromptSourceSubTurnProfile PromptSourceID = "subturn.profile"
	PromptSourceUserMessage    PromptSourceID = "turn:user_message"
	PromptSourceSteering       PromptSourceID = "turn:steering"
	PromptSourceSubTurnResult  PromptSourceID = "turn:subturn_result"
	PromptSourceInterrupt      PromptSourceID = "turn:interrupt"
)

type PromptCachePolicy string

const (
	PromptCacheDefault   PromptCachePolicy = ""
	PromptCacheEphemeral PromptCachePolicy = "ephemeral"
	PromptCacheNone      PromptCachePolicy = "none"
)

type PromptPlacement struct {
	Layer PromptLayer
	Slot  PromptSlot
}

type PromptSourceDescriptor struct {
	ID              PromptSourceID
	Owner           string
	Description     string
	Allowed         []PromptPlacement
	StableByDefault bool
}

type PromptSource struct {
	ID   PromptSourceID
	Name string
	Path string
}

type PromptPart struct {
	ID      string
	Layer   PromptLayer
	Slot    PromptSlot
	Source  PromptSource
	Title   string
	Content string
	Stable  bool
	Cache   PromptCachePolicy
}

type PromptBuildRequest struct {
	History []providers.Message
	Summary string

	CurrentMessage string
	Media          []string

	Channel           string
	ChatID            string
	SenderID          string
	SenderDisplayName string

	ActiveSkills []string
	Overlays     []PromptPart
}

type PromptContributor interface {
	PromptSource() PromptSourceDescriptor
	ContributePrompt(ctx context.Context, req PromptBuildRequest) ([]PromptPart, error)
}

type PromptRegistry struct {
	mu           sync.RWMutex
	sources      map[PromptSourceID]PromptSourceDescriptor
	contributors []PromptContributor
	warned       map[PromptSourceID]struct{}
}

func NewPromptRegistry() *PromptRegistry {
	r := &PromptRegistry{
		sources: make(map[PromptSourceID]PromptSourceDescriptor),
		warned:  make(map[PromptSourceID]struct{}),
	}
	for _, desc := range builtinPromptSources() {
		if err := r.RegisterSource(desc); err != nil {
			logger.WarnCF("agent", "Failed to register builtin prompt source", map[string]any{
				"source": desc.ID,
				"error":  err.Error(),
			})
		}
	}
	return r
}

func builtinPromptSources() []PromptSourceDescriptor {
	return []PromptSourceDescriptor{
		{
			ID:              PromptSourceKernel,
			Owner:           "agent",
			Description:     "Core picoclaw identity and hard rules",
			Allowed:         []PromptPlacement{{Layer: PromptLayerKernel, Slot: PromptSlotIdentity}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceHierarchy,
			Owner:           "agent",
			Description:     "Prompt hierarchy rules",
			Allowed:         []PromptPlacement{{Layer: PromptLayerKernel, Slot: PromptSlotHierarchy}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceWorkspace,
			Owner:           "workspace",
			Description:     "Workspace and agent definition files",
			Allowed:         []PromptPlacement{{Layer: PromptLayerInstruction, Slot: PromptSlotWorkspace}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceToolDiscovery,
			Owner:           "tools",
			Description:     "Tool discovery instructions",
			Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotTooling}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceToolRegistry,
			Owner:           "tools",
			Description:     "Native provider tool definitions",
			Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotTooling}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceSkillCatalog,
			Owner:           "skills",
			Description:     "Installed skill catalog",
			Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotSkillCatalog}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceActiveSkills,
			Owner:           "skills",
			Description:     "Active skill instructions for the current request",
			Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotActiveSkill}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceMemory,
			Owner:           "memory",
			Description:     "Workspace memory context",
			Allowed:         []PromptPlacement{{Layer: PromptLayerContext, Slot: PromptSlotMemory}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceRuntime,
			Owner:           "agent",
			Description:     "Per-request runtime context",
			Allowed:         []PromptPlacement{{Layer: PromptLayerContext, Slot: PromptSlotRuntime}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceSummary,
			Owner:           "context_manager",
			Description:     "Conversation summary context",
			Allowed:         []PromptPlacement{{Layer: PromptLayerContext, Slot: PromptSlotSummary}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceOutputPolicy,
			Owner:           "agent",
			Description:     "Output formatting policy",
			Allowed:         []PromptPlacement{{Layer: PromptLayerContext, Slot: PromptSlotOutput}},
			StableByDefault: true,
		},
		{
			ID:              PromptSourceSubTurnProfile,
			Owner:           "subturn",
			Description:     "Child agent profile instructions",
			Allowed:         []PromptPlacement{{Layer: PromptLayerInstruction, Slot: PromptSlotWorkspace}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceUserMessage,
			Owner:           "turn",
			Description:     "Current user message for this turn",
			Allowed:         []PromptPlacement{{Layer: PromptLayerTurn, Slot: PromptSlotMessage}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceSteering,
			Owner:           "turn",
			Description:     "Steering message injected into a running turn",
			Allowed:         []PromptPlacement{{Layer: PromptLayerTurn, Slot: PromptSlotSteering}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceSubTurnResult,
			Owner:           "turn",
			Description:     "SubTurn result injected into a parent turn",
			Allowed:         []PromptPlacement{{Layer: PromptLayerTurn, Slot: PromptSlotSubTurn}},
			StableByDefault: false,
		},
		{
			ID:              PromptSourceInterrupt,
			Owner:           "turn",
			Description:     "Graceful interrupt hint injected into the terminal LLM call",
			Allowed:         []PromptPlacement{{Layer: PromptLayerTurn, Slot: PromptSlotInterrupt}},
			StableByDefault: false,
		},
	}
}

func (r *PromptRegistry) RegisterSource(desc PromptSourceDescriptor) error {
	if r == nil {
		return fmt.Errorf("prompt registry is nil")
	}
	desc.ID = PromptSourceID(strings.TrimSpace(string(desc.ID)))
	if desc.ID == "" {
		return fmt.Errorf("prompt source id is required")
	}
	if len(desc.Allowed) == 0 {
		return fmt.Errorf("prompt source %q must declare at least one placement", desc.ID)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[desc.ID] = clonePromptSourceDescriptor(desc)
	return nil
}

func (r *PromptRegistry) RegisterContributor(contributor PromptContributor) error {
	if r == nil {
		return fmt.Errorf("prompt registry is nil")
	}
	if contributor == nil {
		return fmt.Errorf("prompt contributor is nil")
	}
	desc := contributor.PromptSource()
	desc.ID = PromptSourceID(strings.TrimSpace(string(desc.ID)))
	if err := r.RegisterSource(desc); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributors = slices.DeleteFunc(r.contributors, func(existing PromptContributor) bool {
		return PromptSourceID(strings.TrimSpace(string(existing.PromptSource().ID))) == desc.ID
	})
	r.contributors = append(r.contributors, contributor)
	return nil
}

func (r *PromptRegistry) Collect(ctx context.Context, req PromptBuildRequest) ([]PromptPart, error) {
	if r == nil {
		return nil, nil
	}

	r.mu.RLock()
	contributors := append([]PromptContributor(nil), r.contributors...)
	r.mu.RUnlock()

	var parts []PromptPart
	for _, contributor := range contributors {
		contributed, err := contributor.ContributePrompt(ctx, req)
		if err != nil {
			return nil, err
		}
		for _, part := range contributed {
			if err := r.ValidatePart(part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		}
	}
	return parts, nil
}

func (r *PromptRegistry) ValidatePart(part PromptPart) error {
	if r == nil {
		return nil
	}
	sourceID := PromptSourceID(strings.TrimSpace(string(part.Source.ID)))
	if sourceID == "" {
		return fmt.Errorf("prompt part %q has empty source id", part.ID)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	desc, ok := r.sources[sourceID]
	if !ok {
		if _, warned := r.warned[sourceID]; !warned {
			r.warned[sourceID] = struct{}{}
			logger.WarnCF("agent", "Unregistered prompt source allowed in compatibility mode", map[string]any{
				"source": sourceID,
				"layer":  part.Layer,
				"slot":   part.Slot,
				"part":   part.ID,
			})
		}
		return nil
	}
	if promptPlacementAllowed(desc.Allowed, PromptPlacement{Layer: part.Layer, Slot: part.Slot}) {
		return nil
	}
	return fmt.Errorf("prompt source %q cannot write to %s/%s", sourceID, part.Layer, part.Slot)
}

func promptPlacementAllowed(allowed []PromptPlacement, placement PromptPlacement) bool {
	return slices.ContainsFunc(allowed, func(candidate PromptPlacement) bool {
		return candidate.Layer == placement.Layer && candidate.Slot == placement.Slot
	})
}

func clonePromptSourceDescriptor(desc PromptSourceDescriptor) PromptSourceDescriptor {
	desc.Allowed = append([]PromptPlacement(nil), desc.Allowed...)
	return desc
}

type PromptStack struct {
	registry *PromptRegistry
	parts    []PromptPart
	sealed   bool
}

func NewPromptStack(registry *PromptRegistry) *PromptStack {
	return &PromptStack{registry: registry}
}

func (s *PromptStack) Add(part PromptPart) error {
	if s == nil {
		return fmt.Errorf("prompt stack is nil")
	}
	if s.sealed {
		return fmt.Errorf("prompt stack is sealed")
	}
	if strings.TrimSpace(part.Content) == "" {
		return nil
	}
	if strings.TrimSpace(part.ID) == "" {
		return fmt.Errorf("prompt part id is required")
	}
	if s.registry != nil {
		if err := s.registry.ValidatePart(part); err != nil {
			return err
		}
	}
	s.parts = append(s.parts, part)
	return nil
}

func (s *PromptStack) Seal() {
	if s != nil {
		s.sealed = true
	}
}

func (s *PromptStack) Parts() []PromptPart {
	if s == nil || len(s.parts) == 0 {
		return nil
	}
	return append([]PromptPart(nil), s.parts...)
}

func renderPromptPartsLegacy(parts []PromptPart) string {
	textParts := make([]string, 0, len(parts))
	for _, part := range sortPromptParts(parts) {
		if strings.TrimSpace(part.Content) == "" {
			continue
		}
		textParts = append(textParts, part.Content)
	}
	return strings.Join(textParts, "\n\n---\n\n")
}

func sortPromptParts(parts []PromptPart) []PromptPart {
	sorted := append([]PromptPart(nil), parts...)
	slices.SortStableFunc(sorted, func(a, b PromptPart) int {
		if d := layerPriority(b.Layer) - layerPriority(a.Layer); d != 0 {
			return d
		}
		if d := slotPriority(b.Slot) - slotPriority(a.Slot); d != 0 {
			return d
		}
		if a.Source.ID != b.Source.ID {
			return strings.Compare(string(a.Source.ID), string(b.Source.ID))
		}
		return strings.Compare(a.ID, b.ID)
	})
	return sorted
}

func layerPriority(layer PromptLayer) int {
	switch layer {
	case PromptLayerKernel:
		return 100
	case PromptLayerInstruction:
		return 80
	case PromptLayerCapability:
		return 60
	case PromptLayerContext:
		return 40
	case PromptLayerTurn:
		return 20
	default:
		return 0
	}
}

func slotPriority(slot PromptSlot) int {
	switch slot {
	case PromptSlotIdentity:
		return 1000
	case PromptSlotHierarchy:
		return 990
	case PromptSlotWorkspace:
		return 900
	case PromptSlotTooling:
		return 800
	case PromptSlotMCP:
		return 790
	case PromptSlotSkillCatalog:
		return 780
	case PromptSlotActiveSkill:
		return 770
	case PromptSlotMemory:
		return 700
	case PromptSlotOutput:
		return 695
	case PromptSlotRuntime:
		return 690
	case PromptSlotSummary:
		return 680
	case PromptSlotMessage:
		return 600
	case PromptSlotSteering:
		return 590
	case PromptSlotSubTurn:
		return 580
	case PromptSlotInterrupt:
		return 570
	default:
		return 0
	}
}
