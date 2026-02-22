// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package relation

import (
	"fmt"
	"sync"
)

// AuthzRequest represents an authorization request
type AuthzRequest struct {
	// SubjectHID is the H-id of the subject requesting access
	SubjectHID string

	// SubjectSID is the optional S-id of the subject
	SubjectSID string

	// Action is the action being requested
	Action Action

	// Resource is the resource being accessed
	Resource *ResourceID

	// Context provides additional context for the decision
	Context map[string]interface{}
}

// AuthzResult represents the result of an authorization decision
type AuthzResult struct {
	// Allowed is true if access is granted
	Allowed bool

	// Reason is a human-readable explanation
	Reason string

	// MatchedRelation is the relation that matched (if any)
	MatchedRelation *Relation

	// MatchedPolicy is the policy that matched (if any)
	MatchedPolicy *Policy
}

// Authorizer handles authorization decisions
type Authorizer struct {
	registry *Registry
	policies *PolicySet
	mu       sync.RWMutex

	// DefaultDeny causes authorization to deny by default
	DefaultDeny bool
}

// NewAuthorizer creates a new authorizer
func NewAuthorizer() *Authorizer {
	return &Authorizer{
		registry: NewRegistry(),
		policies: DefaultPolicies(),
		DefaultDeny: true,
	}
}

// NewAuthorizerWithRegistry creates an authorizer with a specific registry
func NewAuthorizerWithRegistry(registry *Registry) *Authorizer {
	return &Authorizer{
		registry: registry,
		policies: DefaultPolicies(),
		DefaultDeny: true,
	}
}

// SetRegistry sets the relation registry
func (a *Authorizer) SetRegistry(registry *Registry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.registry = registry
}

// GetRegistry returns the relation registry
func (a *Authorizer) GetRegistry() *Registry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.registry
}

// SetPolicies sets the policy set
func (a *Authorizer) SetPolicies(policies *PolicySet) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.policies = policies
}

// GetPolicies returns the policy set
func (a *Authorizer) GetPolicies() *PolicySet {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.policies
}

// Authorize checks if a subject is authorized to perform an action on a resource
func (a *Authorizer) Authorize(req *AuthzRequest) *AuthzResult {
	if req == nil {
		return &AuthzResult{
			Allowed: false,
			Reason:  "nil request",
		}
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check for owner access first (always allowed)
	if a.isOwner(req) {
		return &AuthzResult{
			Allowed: true,
			Reason:  "owner has full access",
		}
	}

	// Check policy-based authorization
	policyResult := a.checkPolicies(req)
	if policyResult != nil {
		return policyResult
	}

	// Check relation-based authorization
	return a.checkRelations(req)
}

// isOwner checks if the subject is the owner of the resource
func (a *Authorizer) isOwner(req *AuthzRequest) bool {
	// Owner check: subject H-id matches resource HID
	return req.SubjectHID == req.Resource.HID
}

// checkPolicies checks if any policies allow or deny the request
func (a *Authorizer) checkPolicies(req *AuthzRequest) *AuthzResult {
	if a.policies == nil || a.policies.Count() == 0 {
		return nil
	}

	effect := a.policies.Evaluate(req.SubjectHID, req.Action, req.Resource)

	if effect == EffectDeny {
		return &AuthzResult{
			Allowed: false,
			Reason:  "denied by policy",
		}
	}

	if effect == EffectAllow {
		return &AuthzResult{
			Allowed: true,
			Reason:  "allowed by policy",
		}
	}

	return nil
}

// checkRelations checks if any relations allow or deny the request
func (a *Authorizer) checkRelations(req *AuthzRequest) *AuthzResult {
	if a.registry == nil {
		return a.defaultResult()
	}

	// Get the highest privilege relation
	privilege := a.registry.GetPrivilege(req.SubjectHID, req.SubjectSID, req.Resource)

	if privilege == "" || privilege == RelationAny {
		return a.defaultResult()
	}

	// Check if the relation type allows the action
	if ActionAllowedByRelation(req.Action, privilege) {
		return &AuthzResult{
			Allowed: true,
			Reason: fmt.Sprintf("allowed by %s relation", privilege),
		}
	}

	// Check for owner privileges (owner can do anything)
	if privilege == RelationOwner {
		return &AuthzResult{
			Allowed: true,
			Reason:  "owner has full access",
		}
	}

	return &AuthzResult{
		Allowed: false,
		Reason: fmt.Sprintf("%s relation does not allow %s", privilege, req.Action),
	}
}

// defaultResult returns the default authorization result
func (a *Authorizer) defaultResult() *AuthzResult {
	if a.DefaultDeny {
		return &AuthzResult{
			Allowed: false,
			Reason:  "default deny",
		}
	}
	return &AuthzResult{
		Allowed: true,
		Reason:  "default allow",
	}
}

// CanRead is a convenience method to check read authorization
func (a *Authorizer) CanRead(subjectHID, subjectSID string, resource *ResourceID) bool {
	req := &AuthzRequest{
		SubjectHID: subjectHID,
		SubjectSID: subjectSID,
		Action:     ActionRead,
		Resource:   resource,
	}
	return a.Authorize(req).Allowed
}

// CanWrite is a convenience method to check write authorization
func (a *Authorizer) CanWrite(subjectHID, subjectSID string, resource *ResourceID) bool {
	req := &AuthzRequest{
		SubjectHID: subjectHID,
		SubjectSID: subjectSID,
		Action:     ActionWrite,
		Resource:   resource,
	}
	return a.Authorize(req).Allowed
}

// CanDelete is a convenience method to check delete authorization
func (a *Authorizer) CanDelete(subjectHID, subjectSID string, resource *ResourceID) bool {
	req := &AuthzRequest{
		SubjectHID: subjectHID,
		SubjectSID: subjectSID,
		Action:     ActionDelete,
		Resource:   resource,
	}
	return a.Authorize(req).Allowed
}

// CanExecute is a convenience method to check execute authorization
func (a *Authorizer) CanExecute(subjectHID, subjectSID string, resource *ResourceID) bool {
	req := &AuthzRequest{
		SubjectHID: subjectHID,
		SubjectSID: subjectSID,
		Action:     ActionExecute,
		Resource:   resource,
	}
	return a.Authorize(req).Allowed
}

// CanAdmin is a convenience method to check admin authorization
func (a *Authorizer) CanAdmin(subjectHID, subjectSID string, resource *ResourceID) bool {
	req := &AuthzRequest{
		SubjectHID: subjectHID,
		SubjectSID: subjectSID,
		Action:     ActionAdmin,
		Resource:   resource,
	}
	return a.Authorize(req).Allowed
}

// CanShare is a convenience method to check share authorization
func (a *Authorizer) CanShare(subjectHID, subjectSID string, resource *ResourceID) bool {
	req := &AuthzRequest{
		SubjectHID: subjectHID,
		SubjectSID: subjectSID,
		Action:     ActionShare,
		Resource:   resource,
	}
	return a.Authorize(req).Allowed
}

// Grant grants a relation to a subject for a resource
func (a *Authorizer) Grant(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.registry == nil {
		return fmt.Errorf("no registry configured")
	}

	rel := NewRelationWithSID(subjectHID, subjectSID, resource, relType)
	return a.registry.Add(rel)
}

// Revoke revokes a relation from a subject for a resource
func (a *Authorizer) Revoke(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.registry == nil {
		return fmt.Errorf("no registry configured")
	}

	return a.registry.Remove(subjectHID, subjectSID, resource, relType)
}

// GetRelations returns all relations for a subject
func (a *Authorizer) GetRelations(subjectHID, subjectSID string) []*Relation {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.registry == nil {
		return nil
	}

	return a.registry.GetBySubject(subjectHID, subjectSID)
}

// GetResources returns all resources a subject has access to
func (a *Authorizer) GetResources(subjectHID, subjectSID string) []*ResourceID {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.registry == nil {
		return nil
	}

	return a.registry.GetResourcesBySubject(subjectHID, subjectSID)
}

// FilterAuthorized returns only the resources that the subject is authorized to access
func (a *Authorizer) FilterAuthorized(subjectHID, subjectSID string, resources []*ResourceID, action Action) []*ResourceID {
	a.mu.RLock()
	defer a.mu.RUnlock()

	authorized := make([]*ResourceID, 0)

	for _, resource := range resources {
		req := &AuthzRequest{
			SubjectHID: subjectHID,
			SubjectSID: subjectSID,
			Action:     action,
			Resource:   resource,
		}

		if a.Authorize(req).Allowed {
			authorized = append(authorized, resource)
		}
	}

	return authorized
}

// BatchAuthorize checks authorization for multiple requests
func (a *Authorizer) BatchAuthorize(requests []*AuthzRequest) []*AuthzResult {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make([]*AuthzResult, len(requests))

	for i, req := range requests {
		results[i] = a.checkRelations(req)
	}

	return results
}

// TransferOwnership transfers ownership of a resource to a new H-id
func (a *Authorizer) TransferOwnership(resource *ResourceID, currentOwner, newOwner string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.registry == nil {
		return fmt.Errorf("no registry configured")
	}

	// Remove old owner relations
	oldOwnerFilter := &RelationFilter{
		SubjectHID:   currentOwner,
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		RelationType: RelationOwner,
	}

	matching := a.registry.Apply(oldOwnerFilter)
	for _, rel := range matching {
		if rel.RelationType == RelationOwner {
			a.registry.Remove(rel.SubjectHID, rel.SubjectSID, rel.Resource, RelationOwner)
		}
	}

	// Add new owner relation
	newOwnerRel := NewRelation(newOwner, resource, RelationOwner)
	return a.registry.Add(newOwnerRel)
}

// AddPolicy adds a policy to the authorizer
func (a *Authorizer) AddPolicy(policy *Policy) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.policies == nil {
		a.policies = NewPolicySet()
	}

	a.policies.Add(policy)
}

// RemovePolicy removes a policy from the authorizer
func (a *Authorizer) RemovePolicy(policyID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.policies == nil {
		return
	}

	a.policies.Remove(policyID)
}

// GetPolicy returns a policy by ID
func (a *Authorizer) GetPolicy(policyID string) (*Policy, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.policies == nil {
		return nil, false
	}

	return a.policies.Get(policyID)
}

// GetAllPolicies returns all policies
func (a *Authorizer) GetAllPolicies() []*Policy {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.policies == nil {
		return nil
	}

	return a.policies.GetAll()
}

// ClearPolicies removes all policies
func (a *Authorizer) ClearPolicies() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.policies = NewPolicySet()
}

// Clear clears the registry and policies
func (a *Authorizer) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.registry != nil {
		a.registry.Clear()
	}

	a.policies = NewPolicySet()
}

// Stats returns statistics about the authorizer
func (a *Authorizer) Stats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := make(map[string]interface{})

	if a.registry != nil {
		stats["relation_count"] = a.registry.Count()
	}

	if a.policies != nil {
		stats["policy_count"] = a.policies.Count()
	}

	stats["default_deny"] = a.DefaultDeny

	return stats
}
