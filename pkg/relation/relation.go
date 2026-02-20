// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package relation

import (
	"fmt"
)

// RelationType defines the type of relationship
type RelationType string

const (
	// RelationOwner indicates ownership
	RelationOwner RelationType = "owner"

	// RelationMember indicates membership
	RelationMember RelationType = "member"

	// RelationReader indicates read access
	RelationReader RelationType = "reader"

	// RelationWriter indicates write access
	RelationWriter RelationType = "writer"

	// RelationAdmin indicates administrative access
	RelationAdmin RelationType = "admin"

	// RelationExecutor indicates execution permission
	RelationExecutor RelationType = "executor"

	// RelationDelegate indicates delegation permission
	RelationDelegate RelationType = "delegate"

	// RelationAny is a wildcard for any relation type
	RelationAny RelationType = "*"
)

// Relation represents a relationship between an identity and a resource
type Relation struct {
	// SubjectHID is the H-id of the subject (who has the relation)
	SubjectHID string `json:"subject_hid"`

	// SubjectSID is the optional S-id of the subject
	SubjectSID string `json:"subject_sid,omitempty"`

	// Resource is the resource being related to
	Resource *ResourceID `json:"resource"`

	// RelationType is the type of relation
	RelationType RelationType `json:"relation_type"`

	// Attributes contains additional relation attributes
	Attributes map[string]string `json:"attributes,omitempty"`
}

// NewRelation creates a new relation
func NewRelation(subjectHID string, resource *ResourceID, relType RelationType) *Relation {
	return &Relation{
		SubjectHID:   subjectHID,
		Resource:     resource,
		RelationType: relType,
		Attributes:   make(map[string]string),
	}
}

// NewRelationWithSID creates a new relation with a specific S-id
func NewRelationWithSID(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) *Relation {
	return &Relation{
		SubjectHID:   subjectHID,
		SubjectSID:   subjectSID,
		Resource:     resource,
		RelationType: relType,
		Attributes:   make(map[string]string),
	}
}

// String returns the string representation of the relation
func (r *Relation) String() string {
	subject := r.SubjectHID
	if r.SubjectSID != "" {
		subject = r.SubjectHID + "/" + r.SubjectSID
	}
	return fmt.Sprintf("%s %s %s", subject, r.RelationType, r.Resource.String())
}

// Matches checks if this relation matches the given criteria
func (r *Relation) Matches(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) bool {
	if r.SubjectHID != "" && subjectHID != "" && r.SubjectHID != subjectHID {
		return false
	}
	if r.SubjectSID != "" && subjectSID != "" && r.SubjectSID != subjectSID {
		return false
	}
	if r.RelationType != RelationAny && relType != RelationAny && r.RelationType != relType {
		return false
	}
	if resource != nil && !r.Resource.Matches(resource) {
		return false
	}
	return true
}

// IsDirect returns true if the relation is for a specific S-id
func (r *Relation) IsDirect() bool {
	return r.SubjectSID != ""
}

// IsTenant returns true if the relation is for an entire H-id
func (r *Relation) IsTenant() bool {
	return r.SubjectSID == ""
}

// GetAttribute returns an attribute value
func (r *Relation) GetAttribute(key string) (string, bool) {
	if r.Attributes == nil {
		return "", false
	}
	v, ok := r.Attributes[key]
	return v, ok
}

// SetAttribute sets an attribute value
func (r *Relation) SetAttribute(key, value string) {
	if r.Attributes == nil {
		r.Attributes = make(map[string]string)
	}
	r.Attributes[key] = value
}

// Clone returns a copy of the relation
func (r *Relation) Clone() *Relation {
	if r == nil {
		return nil
	}

	clone := &Relation{
		SubjectHID:   r.SubjectHID,
		SubjectSID:   r.SubjectSID,
		Resource:     r.Resource.Clone(),
		RelationType: r.RelationType,
	}

	if r.Attributes != nil {
		clone.Attributes = make(map[string]string, len(r.Attributes))
		for k, v := range r.Attributes {
			clone.Attributes[k] = v
		}
	}

	return clone
}

// RelationTypeHierarchy defines the hierarchy of relation types
// Higher values indicate more privilege
var RelationTypeHierarchy = map[RelationType]int{
	RelationReader:   1,
	RelationMember:   2,
	RelationWriter:   3,
	RelationExecutor: 4,
	RelationAdmin:    5,
	RelationDelegate: 6,
	RelationOwner:    7,
}

// RelationActionPermissions defines which actions each relation type can perform
// This allows for non-linear permission models where e.g., Admin has read+delete but not write
var RelationActionPermissions = map[RelationType]map[Action]bool{
	RelationReader: {
		ActionRead: true,
	},
	RelationWriter: {
		ActionRead:  true,
		ActionWrite: true,
	},
	RelationAdmin: {
		ActionRead:   true,
		ActionDelete: true,
		ActionShare:  true,
		ActionAdmin:  true,
	},
	RelationExecutor: {
		ActionRead:    true,
		ActionExecute: true,
	},
	RelationDelegate: {
		ActionRead:   true,
		ActionWrite:  true,
		ActionDelete: true,
		ActionShare:  true,
		ActionAdmin:  true,
	},
	RelationOwner: {
		ActionRead:    true,
		ActionWrite:   true,
		ActionDelete:  true,
		ActionShare:   true,
		ActionExecute: true,
		ActionAdmin:   true,
	},
}

// HasPermission checks if a relation type has a specific action permission
func HasPermission(relType RelationType, action Action) bool {
	if relType == RelationOwner {
		return true // Owner has all permissions
	}
	perms, ok := RelationActionPermissions[relType]
	if !ok {
		return false
	}
	return perms[action]
}

// HasPrivilege returns true if the given relation type has at least the required privilege level
func HasPrivilege(has, requires RelationType) bool {
	if has == RelationOwner {
		return true // Owner has all privileges
	}
	if requires == RelationOwner {
		return false // Only owner has owner privilege
	}

	hasLevel, ok := RelationTypeHierarchy[has]
	if !ok {
		return false
	}

	requiredLevel, ok := RelationTypeHierarchy[requires]
	if !ok {
		return false
	}

	return hasLevel >= requiredLevel
}

// IsAtLeast returns true if this relation type is at least as privileged as the other
func (r RelationType) IsAtLeast(other RelationType) bool {
	return HasPrivilege(r, other)
}

// IsAtMost returns true if this relation type is at most as privileged as the other
func (r RelationType) IsAtMost(other RelationType) bool {
	return HasPrivilege(other, r)
}

// CommonRelationTypes returns relation types that typically imply read access
func ReadRelationTypes() []RelationType {
	return []RelationType{
		RelationReader,
		RelationWriter,
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
	}
}

// WriteRelationTypes returns relation types that typically imply write access
func WriteRelationTypes() []RelationType {
	return []RelationType{
		RelationWriter,
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
	}
}

// AdminRelationTypes returns relation types with administrative privileges
func AdminRelationTypes() []RelationType {
	return []RelationType{
		RelationAdmin,
		RelationDelegate,
		RelationOwner,
	}
}

// RelationSet is a collection of relations with efficient lookup
type RelationSet struct {
	relations []*Relation
	// indexes for fast lookup
	bySubject   map[string][]*Relation
	byResource  map[string][]*Relation
	byType      map[RelationType][]*Relation
}

// NewRelationSet creates a new relation set
func NewRelationSet() *RelationSet {
	return &RelationSet{
		relations: make([]*Relation, 0),
		bySubject: make(map[string][]*Relation),
		byResource: make(map[string][]*Relation),
		byType: make(map[RelationType][]*Relation),
	}
}

// Add adds a relation to the set
func (s *RelationSet) Add(rel *Relation) {
	s.relations = append(s.relations, rel)

	// Update indexes
	subjectKey := rel.SubjectHID
	if rel.SubjectSID != "" {
		subjectKey = rel.SubjectHID + "/" + rel.SubjectSID
	}
	s.bySubject[subjectKey] = append(s.bySubject[subjectKey], rel)

	resourceKey := rel.Resource.String()
	s.byResource[resourceKey] = append(s.byResource[resourceKey], rel)

	s.byType[rel.RelationType] = append(s.byType[rel.RelationType], rel)
}

// Remove removes relations matching the given criteria
func (s *RelationSet) Remove(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) int {
	removed := 0
	newRelations := make([]*Relation, 0, len(s.relations))

	// Clear indexes
	s.bySubject = make(map[string][]*Relation)
	s.byResource = make(map[string][]*Relation)
	s.byType = make(map[RelationType][]*Relation)

	for _, rel := range s.relations {
		if rel.Matches(subjectHID, subjectSID, resource, relType) {
			removed++
			continue
		}
		newRelations = append(newRelations, rel)

		// Rebuild indexes
		subjectKey := rel.SubjectHID
		if rel.SubjectSID != "" {
			subjectKey = rel.SubjectHID + "/" + rel.SubjectSID
		}
		s.bySubject[subjectKey] = append(s.bySubject[subjectKey], rel)

		resourceKey := rel.Resource.String()
		s.byResource[resourceKey] = append(s.byResource[resourceKey], rel)

		s.byType[rel.RelationType] = append(s.byType[rel.RelationType], rel)
	}

	s.relations = newRelations
	return removed
}

// FindBySubject finds all relations for a given subject
func (s *RelationSet) FindBySubject(subjectHID, subjectSID string) []*Relation {
	key := subjectHID
	if subjectSID != "" {
		key = subjectHID + "/" + subjectSID
	}

	if rels, ok := s.bySubject[key]; ok {
		return rels
	}

	return nil
}

// FindByResource finds all relations for a given resource
func (s *RelationSet) FindByResource(resource *ResourceID) []*Relation {
	if rels, ok := s.byResource[resource.String()]; ok {
		return rels
	}
	return nil
}

// FindByType finds all relations of a given type
func (s *RelationSet) FindByType(relType RelationType) []*Relation {
	if rels, ok := s.byType[relType]; ok {
		return rels
	}
	return nil
}

// Find finds relations matching all given criteria
func (s *RelationSet) Find(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) []*Relation {
	results := make([]*Relation, 0)

	for _, rel := range s.relations {
		if rel.Matches(subjectHID, subjectSID, resource, relType) {
			results = append(results, rel)
		}
	}

	return results
}

// HasAnyRelation returns true if the subject has any relation to the resource
func (s *RelationSet) HasAnyRelation(subjectHID string, resource *ResourceID) bool {
	for _, rel := range s.relations {
		if rel.SubjectHID == subjectHID && rel.Resource.Matches(resource) {
			return true
		}
	}
	return false
}

// GetHighestPrivilege returns the highest privilege relation type for a subject on a resource
func (s *RelationSet) GetHighestPrivilege(subjectHID, subjectSID string, resource *ResourceID) RelationType {
	highest := RelationType("")

	for _, rel := range s.relations {
		if !rel.Matches(subjectHID, subjectSID, resource, RelationAny) {
			continue
		}

		if highest == "" || HasPrivilege(rel.RelationType, highest) {
			highest = rel.RelationType
		}
	}

	return highest
}

// All returns all relations in the set
func (s *RelationSet) All() []*Relation {
	return s.relations
}

// Len returns the number of relations in the set
func (s *RelationSet) Len() int {
	return len(s.relations)
}

// Clear removes all relations from the set
func (s *RelationSet) Clear() {
	s.relations = make([]*Relation, 0)
	s.bySubject = make(map[string][]*Relation)
	s.byResource = make(map[string][]*Relation)
	s.byType = make(map[RelationType][]*Relation)
}

// Clone creates a deep copy of the relation set
func (s *RelationSet) Clone() *RelationSet {
	clone := NewRelationSet()

	for _, rel := range s.relations {
		clone.Add(rel.Clone())
	}

	return clone
}

// Merge merges another relation set into this one
func (s *RelationSet) Merge(other *RelationSet) {
	if other == nil {
		return
	}

	for _, rel := range other.All() {
		s.Add(rel.Clone())
	}
}
