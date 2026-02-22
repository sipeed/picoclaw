// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package relation

import (
	"fmt"
	"strings"
)

// Action represents an action that can be performed on a resource
type Action string

const (
	// ActionRead is the read action
	ActionRead Action = "read"

	// ActionWrite is the write action
	ActionWrite Action = "write"

	// ActionDelete is the delete action
	ActionDelete Action = "delete"

	// ActionExecute is the execute action
	ActionExecute Action = "execute"

	// ActionAdmin is the admin action
	ActionAdmin Action = "admin"

	// ActionShare is the share/delegate action
	ActionShare Action = "share"

	// ActionAny is a wildcard for any action
	ActionAny Action = "*"
)

// ActionMapping defines which relation types can perform which actions
// Note: Admin is a read-only management role with delete/admin but NOT write permissions
var ActionMapping = map[Action][]RelationType{
	ActionRead: {
		RelationReader,
		RelationWriter,
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
		RelationMember, // Members can typically read
	},
	ActionWrite: {
		RelationWriter,
		RelationDelegate,
		RelationOwner,
		// Admin NOT included - Admin is read-only for content
	},
	ActionDelete: {
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
	},
	ActionExecute: {
		RelationExecutor,
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
	},
	ActionAdmin: {
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
	},
	ActionShare: {
		RelationDelegate,
		RelationOwner,
	},
}

// Effect defines the effect of a policy statement
type Effect int

const (
	// EffectAllow permits the action
	EffectAllow Effect = iota

	// EffectDeny denies the action
	EffectDeny
)

// String returns the string representation of the effect
func (e Effect) String() string {
	switch e {
	case EffectAllow:
		return "allow"
	case EffectDeny:
		return "deny"
	default:
		return "unknown"
	}
}

// ParseEffect parses a string into an Effect
func ParseEffect(s string) (Effect, error) {
	switch strings.ToLower(s) {
	case "allow":
		return EffectAllow, nil
	case "deny":
		return EffectDeny, nil
	default:
		return EffectDeny, fmt.Errorf("invalid effect: %s", s)
	}
}

// Statement is a single policy statement
type Statement struct {
	// Effect is either allow or deny
	Effect Effect `json:"effect"`

	// Action is the action this statement applies to
	Action Action `json:"action"`

	// Resource is the resource this statement applies to
	Resource *ResourceID `json:"resource"`

	// Condition is an optional condition that must be met
	Condition string `json:"condition,omitempty"`
}

// NewAllowStatement creates a new allow statement
func NewAllowStatement(action Action, resource *ResourceID) *Statement {
	return &Statement{
		Effect:   EffectAllow,
		Action:   action,
		Resource: resource,
	}
}

// NewDenyStatement creates a new deny statement
func NewDenyStatement(action Action, resource *ResourceID) *Statement {
	return &Statement{
		Effect:   EffectDeny,
		Action:   action,
		Resource: resource,
	}
}

// Matches checks if the statement matches the given action and resource
func (s *Statement) Matches(action Action, resource *ResourceID) bool {
	if s.Action != ActionAny && action != ActionAny && s.Action != action {
		return false
	}
	if s.Resource != nil && !s.Resource.Matches(resource) {
		return false
	}
	return true
}

// Policy is a collection of policy statements
type Policy struct {
	// ID is a unique identifier for the policy
	ID string `json:"id"`

	// Name is a human-readable name
	Name string `json:"name,omitempty"`

	// Description describes what this policy does
	Description string `json:"description,omitempty"`

	// Statements is the list of policy statements
	Statements []*Statement `json:"statements"`

	// Scope limits the policy to specific H-ids (empty = all)
	Scope []string `json:"scope,omitempty"`
}

// NewPolicy creates a new policy
func NewPolicy(id string) *Policy {
	return &Policy{
		ID:         id,
		Statements: make([]*Statement, 0),
		Scope:      make([]string, 0),
	}
}

// AddStatement adds a statement to the policy
func (p *Policy) AddStatement(stmt *Statement) {
	p.Statements = append(p.Statements, stmt)
}

// AddAllow adds an allow statement to the policy
func (p *Policy) AddAllow(action Action, resource *ResourceID) {
	p.AddStatement(NewAllowStatement(action, resource))
}

// AddDeny adds a deny statement to the policy
func (p *Policy) AddDeny(action Action, resource *ResourceID) {
	p.AddStatement(NewDenyStatement(action, resource))
}

// Evaluate evaluates the policy for a given action and resource
func (p *Policy) Evaluate(action Action, resource *ResourceID) Effect {
	// Default deny
	result := EffectDeny

	for _, stmt := range p.Statements {
		if !stmt.Matches(action, resource) {
			continue
		}

		// Deny takes precedence over allow
		if stmt.Effect == EffectDeny {
			return EffectDeny
		}

		result = EffectAllow
	}

	return result
}

// HasMatch returns true if the policy has any statements that match the given action and resource
func (p *Policy) HasMatch(action Action, resource *ResourceID) bool {
	for _, stmt := range p.Statements {
		if stmt.Matches(action, resource) {
			return true
		}
	}
	return false
}

// IsInScope checks if the given H-id is in the policy's scope
func (p *Policy) IsInScope(hid string) bool {
	if len(p.Scope) == 0 {
		return true // No scope restriction
	}

	for _, s := range p.Scope {
		if s == hid {
			return true
		}
	}

	return false
}

// Clone creates a deep copy of the policy
func (p *Policy) Clone() *Policy {
	clone := &Policy{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Statements:  make([]*Statement, len(p.Statements)),
		Scope:       make([]string, len(p.Scope)),
	}

	copy(clone.Scope, p.Scope)

	for i, stmt := range p.Statements {
		clone.Statements[i] = &Statement{
			Effect:     stmt.Effect,
			Action:     stmt.Action,
			Resource:   stmt.Resource.Clone(),
			Condition:  stmt.Condition,
		}
	}

	return clone
}

// PolicySet is a collection of policies
type PolicySet struct {
	policies map[string]*Policy
}

// NewPolicySet creates a new policy set
func NewPolicySet() *PolicySet {
	return &PolicySet{
		policies: make(map[string]*Policy),
	}
}

// Add adds a policy to the set
func (s *PolicySet) Add(policy *Policy) {
	s.policies[policy.ID] = policy
}

// Get retrieves a policy by ID
func (s *PolicySet) Get(id string) (*Policy, bool) {
	policy, ok := s.policies[id]
	return policy, ok
}

// Remove removes a policy from the set
func (s *PolicySet) Remove(id string) {
	delete(s.policies, id)
}

// Evaluate evaluates all policies for a given action, resource, and H-id
func (s *PolicySet) Evaluate(hid string, action Action, resource *ResourceID) Effect {
	// Default deny
	result := EffectDeny

	for _, policy := range s.policies {
		// Check scope
		if !policy.IsInScope(hid) {
			continue
		}

		// Skip policies that don't have matching statements for this action/resource
		if !policy.HasMatch(action, resource) {
			continue
		}

		effect := policy.Evaluate(action, resource)

		// Deny takes precedence
		if effect == EffectDeny {
			return EffectDeny
		}

		result = EffectAllow
	}

	return result
}

// GetAll returns all policies
func (s *PolicySet) GetAll() []*Policy {
	policies := make([]*Policy, 0, len(s.policies))
	for _, p := range s.policies {
		policies = append(policies, p)
	}
	return policies
}

// Count returns the number of policies
func (s *PolicySet) Count() int {
	return len(s.policies)
}

// Clear removes all policies
func (s *PolicySet) Clear() {
	s.policies = make(map[string]*Policy)
}

// DefaultPolicies returns the default policy set
func DefaultPolicies() *PolicySet {
	ps := NewPolicySet()

	// Owner can do anything on their own resources
	// Note: This is checked via isOwner() in Authorizer, not via policies
	// Default policies should only allow access when explicitly granted via relations
	// So we don't add a blanket allow policy here

	return ps
}

// GetRequiredRelationType returns the minimum relation type required for an action
func GetRequiredRelationType(action Action) RelationType {
	switch action {
	case ActionRead:
		return RelationReader
	case ActionWrite:
		return RelationWriter
	case ActionDelete:
		return RelationAdmin
	case ActionExecute:
		return RelationExecutor
	case ActionAdmin:
		return RelationAdmin
	case ActionShare:
		return RelationDelegate
	default:
		return RelationOwner
	}
}

// ActionAllowedByRelation checks if an action is allowed by a given relation type
func ActionAllowedByRelation(action Action, relType RelationType) bool {
	allowedTypes, ok := ActionMapping[action]
	if !ok {
		return false
	}

	for _, allowed := range allowedTypes {
		if allowed == relType {
			return true
		}
	}

	return false
}

// Built-in policies

// OwnerPolicy creates a policy that gives owners full access
func OwnerPolicy() *Policy {
	p := NewPolicy("owner-policy")
	p.Description = "Owners have full access to their own resources"
	p.AddAllow(ActionAny, &ResourceID{HID: "$owner", ID: "*"})
	return p
}

// ReaderPolicy creates a policy that gives read-only access
func ReaderPolicy() *Policy {
	p := NewPolicy("reader-policy")
	p.Description = "Readers can read resources"
	p.AddAllow(ActionRead, &ResourceID{ID: "*"})
	return p
}

// AdminPolicy creates a policy that gives admin access
func AdminPolicy() *Policy {
	p := NewPolicy("admin-policy")
	p.Description = "Admins have administrative access"
	p.AddAllow(ActionRead, &ResourceID{ID: "*"})
	p.AddAllow(ActionWrite, &ResourceID{ID: "*"})
	p.AddAllow(ActionDelete, &ResourceID{ID: "*"})
	p.AddAllow(ActionAdmin, &ResourceID{ID: "*"})
	return p
}
