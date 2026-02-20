// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package relation

import (
	"fmt"
	"sync"
	"time"
)

// Registry manages relations between identities and resources
type Registry struct {
	mu        sync.RWMutex
	relations *RelationSet

	// Index for faster lookups
	bySubjectResource map[string]*Relation // key: "subject:resource"

	// Change tracking
	lastModified int64
}

// NewRegistry creates a new relation registry
func NewRegistry() *Registry {
	return &Registry{
		relations:        NewRelationSet(),
		bySubjectResource: make(map[string]*Relation),
		lastModified:     time.Now().UnixMilli(),
	}
}

// Add adds a relation to the registry
func (r *Registry) Add(rel *Relation) error {
	if rel == nil {
		return fmt.Errorf("cannot add nil relation")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	key := r.relationKey(rel.SubjectHID, rel.SubjectSID, rel.Resource, rel.RelationType)
	if _, exists := r.bySubjectResource[key]; exists {
		return fmt.Errorf("relation already exists: %s", key)
	}

	r.relations.Add(rel)
	r.bySubjectResource[key] = rel
	r.lastModified = time.Now().UnixMilli()

	return nil
}

// Update updates an existing relation
func (r *Registry) Update(rel *Relation) error {
	if rel == nil {
		return fmt.Errorf("cannot update nil relation")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	newKey := r.relationKey(rel.SubjectHID, rel.SubjectSID, rel.Resource, rel.RelationType)

	// Try to find the relation with any type for the same subject-resource pair
	oldRel, found := r.findBySubjectResource(rel.SubjectHID, rel.SubjectSID, rel.Resource)
	if !found {
		return fmt.Errorf("relation not found for subject-resource pair")
	}

	oldKey := r.relationKey(oldRel.SubjectHID, oldRel.SubjectSID, oldRel.Resource, oldRel.RelationType)

	// Remove old relation and add new one
	r.relations.Remove(oldRel.SubjectHID, oldRel.SubjectSID, oldRel.Resource, oldRel.RelationType)
	delete(r.bySubjectResource, oldKey)

	r.relations.Add(rel)
	r.bySubjectResource[newKey] = rel
	r.lastModified = time.Now().UnixMilli()

	return nil
}

// AddOrUpdate adds a relation or updates if it exists
func (r *Registry) AddOrUpdate(rel *Relation) error {
	if rel == nil {
		return fmt.Errorf("cannot add nil relation")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.relationKey(rel.SubjectHID, rel.SubjectSID, rel.Resource, rel.RelationType)

	// Check if relation exists
	if _, exists := r.bySubjectResource[key]; exists {
		// Update
		r.relations.Remove(rel.SubjectHID, rel.SubjectSID, rel.Resource, RelationAny)
		delete(r.bySubjectResource, key)
	}

	r.relations.Add(rel)
	r.bySubjectResource[key] = rel
	r.lastModified = time.Now().UnixMilli()

	return nil
}

// Remove removes a relation from the registry
func (r *Registry) Remove(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := r.relations.Remove(subjectHID, subjectSID, resource, relType)

	// Update index
	key := r.relationKey(subjectHID, subjectSID, resource, relType)
	if removed > 0 {
		delete(r.bySubjectResource, key)
		r.lastModified = time.Now().UnixMilli()
	}

	return nil
}

// Get retrieves a specific relation
func (r *Registry) Get(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) (*Relation, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rels := r.relations.Find(subjectHID, subjectSID, resource, relType)
	if len(rels) == 0 {
		return nil, false
	}
	return rels[0], true
}

// GetBySubject retrieves all relations for a subject
func (r *Registry) GetBySubject(subjectHID, subjectSID string) []*Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.FindBySubject(subjectHID, subjectSID)
}

// GetByResource retrieves all relations for a resource
func (r *Registry) GetByResource(resource *ResourceID) []*Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.FindByResource(resource)
}

// GetByType retrieves all relations of a specific type
func (r *Registry) GetByType(relType RelationType) []*Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.FindByType(relType)
}

// HasRelation checks if any relation exists between subject and resource
func (r *Registry) HasRelation(subjectHID string, resource *ResourceID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.HasAnyRelation(subjectHID, resource)
}

// GetPrivilege gets the highest privilege relation type for a subject on a resource
func (r *Registry) GetPrivilege(subjectHID, subjectSID string, resource *ResourceID) RelationType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.GetHighestPrivilege(subjectHID, subjectSID, resource)
}

// HasPrivilege checks if a subject has at least the required privilege on a resource
func (r *Registry) HasPrivilege(subjectHID, subjectSID string, resource *ResourceID, required RelationType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	privilege := r.relations.GetHighestPrivilege(subjectHID, subjectSID, resource)
	return HasPrivilege(privilege, required)
}

// GetAll returns all relations in the registry
func (r *Registry) GetAll() []*Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.All()
}

// Count returns the total number of relations
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.Len()
}

// Clear removes all relations from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.relations.Clear()
	r.bySubjectResource = make(map[string]*Relation)
	r.lastModified = time.Now().UnixMilli()
}

// LastModified returns the last modified timestamp
func (r *Registry) LastModified() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lastModified
}

// GetByOwner retrieves all relations where the subject is the owner
func (r *Registry) GetByOwner(ownerHID string) []*Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.relations.FindByType(RelationOwner)
}

// GetResourcesBySubject retrieves all resources that a subject has relations with
func (r *Registry) GetResourcesBySubject(subjectHID, subjectSID string) []*ResourceID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rels := r.relations.FindBySubject(subjectHID, subjectSID)
	resources := make([]*ResourceID, 0, len(rels))
	seen := make(map[string]bool)

	for _, rel := range rels {
		key := rel.Resource.String()
		if !seen[key] {
			seen[key] = true
			resources = append(resources, rel.Resource)
		}
	}

	return resources
}

// GetSubjectsByResource retrieves all subjects that have relations with a resource
func (r *Registry) GetSubjectsByResource(resource *ResourceID) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rels := r.relations.FindByResource(resource)
	subjects := make([]string, 0, len(rels))
	seen := make(map[string]bool)

	for _, rel := range rels {
		key := rel.SubjectHID
		if rel.SubjectSID != "" {
			key = rel.SubjectHID + "/" + rel.SubjectSID
		}
		if !seen[key] {
			seen[key] = true
			subjects = append(subjects, key)
		}
	}

	return subjects
}

// Clone creates a deep copy of the registry
func (r *Registry) Clone() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clone := NewRegistry()
	for _, rel := range r.relations.All() {
		clone.Add(rel.Clone())
	}

	return clone
}

// Merge merges another registry into this one
func (r *Registry) Merge(other *Registry) error {
	if other == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rel := range other.GetAll() {
		key := r.relationKey(rel.SubjectHID, rel.SubjectSID, rel.Resource, rel.RelationType)

		// Skip if already exists
		if _, exists := r.bySubjectResource[key]; exists {
			continue
		}

		r.relations.Add(rel.Clone())
		r.bySubjectResource[key] = rel
	}

	r.lastModified = time.Now().UnixMilli()
	return nil
}

// relationKey creates a unique key for a subject-resource-relation tuple
func (r *Registry) relationKey(subjectHID, subjectSID string, resource *ResourceID, relType RelationType) string {
	subject := subjectHID
	if subjectSID != "" {
		subject = subjectHID + "/" + subjectSID
	}
	return subject + ":" + resource.String() + ":" + string(relType)
}

// findBySubjectResource finds a relation for the given subject-resource pair (any type)
func (r *Registry) findBySubjectResource(subjectHID, subjectSID string, resource *ResourceID) (*Relation, bool) {
	for _, rel := range r.bySubjectResource {
		if rel.SubjectHID == subjectHID && rel.SubjectSID == subjectSID && rel.Resource.Matches(resource) {
			return rel, true
		}
	}
	return nil, false
}

// RelationFilter is used to filter relations
type RelationFilter struct {
	SubjectHID   string
	SubjectSID   string
	ResourceType ResourceType
	ResourceHID  string
	ResourceID   string
	RelationType RelationType
}

// Apply applies the filter to the registry
func (r *Registry) Apply(filter *RelationFilter) []*Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.relations.All()
	results := make([]*Relation, 0)

	for _, rel := range all {
		if filter.SubjectHID != "" && rel.SubjectHID != filter.SubjectHID {
			continue
		}
		if filter.SubjectSID != "" && rel.SubjectSID != filter.SubjectSID {
			continue
		}
		if filter.ResourceType != "" && filter.ResourceType != ResourceAny && rel.Resource.Type != filter.ResourceType {
			continue
		}
		if filter.ResourceHID != "" && rel.Resource.HID != filter.ResourceHID {
			continue
		}
		if filter.ResourceID != "" && rel.Resource.ID != filter.ResourceID {
			continue
		}
		if filter.RelationType != "" && filter.RelationType != RelationAny && rel.RelationType != filter.RelationType {
			continue
		}

		results = append(results, rel)
	}

	return results
}

// Export exports the registry as a map
func (r *Registry) Export() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data := make(map[string]interface{})
	data["last_modified"] = r.lastModified
	data["count"] = r.relations.Len()

	relations := make([]map[string]interface{}, 0, r.relations.Len())
	for _, rel := range r.relations.All() {
		relData := map[string]interface{}{
			"subject_hid":    rel.SubjectHID,
			"subject_sid":    rel.SubjectSID,
			"resource_type":  rel.Resource.Type,
			"resource_hid":   rel.Resource.HID,
			"resource_id":    rel.Resource.ID,
			"resource_namespace": rel.Resource.Namespace,
			"relation_type":  rel.RelationType,
			"attributes":     rel.Attributes,
		}
		relations = append(relations, relData)
	}
	data["relations"] = relations

	return data
}

// Import imports relations from a map
func (r *Registry) Import(data map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	relations, ok := data["relations"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid import data: missing relations")
	}

	for _, relData := range relations {
		relMap, ok := relData.(map[string]interface{})
		if !ok {
			continue
		}

		subjectHID, _ := relMap["subject_hid"].(string)
		subjectSID, _ := relMap["subject_sid"].(string)
		resourceType, _ := relMap["resource_type"].(string)
		resourceHID, _ := relMap["resource_hid"].(string)
		resourceID, _ := relMap["resource_id"].(string)
		resourceNamespace, _ := relMap["resource_namespace"].(string)
		relationType, _ := relMap["relation_type"].(string)

		rel := NewRelationWithSID(
			subjectHID,
			subjectSID,
			&ResourceID{
				Type:      ResourceType(resourceType),
				HID:       resourceHID,
				ID:        resourceID,
				Namespace: resourceNamespace,
			},
			RelationType(relationType),
		)

		r.relations.Add(rel)
	}

	r.lastModified = time.Now().UnixMilli()
	return nil
}
