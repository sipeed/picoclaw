// PicoClaw - Ultra-lightweight personal AI agent
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/identity"
)

// Partition manages H-id based partitioning for NATS subjects
type Partition struct {
	hid      string
	subjects map[string]string // original subject -> partitioned subject
	mu       sync.RWMutex
}

// NewPartition creates a new partition manager
func NewPartition(hid string) *Partition {
	return &Partition{
		hid:      hid,
		subjects: make(map[string]string),
	}
}

// NewPartitionWithIdentity creates a partition manager from an identity
func NewPartitionWithIdentity(id *identity.Identity) *Partition {
	if id == nil {
		return NewPartition("")
	}
	return NewPartition(id.HID)
}

// Partitionize adds H-id partitioning to a subject
func (p *Partition) Partitionize(subject string) string {
	if p.hid == "" {
		return subject // No partitioning
	}

	p.mu.RLock()
	if cached, ok := p.subjects[subject]; ok {
		p.mu.RUnlock()
		return cached
	}
	p.mu.RUnlock()

	// Parse and add H-id
	parsed := ParseSubject(subject)
	if parsed == nil {
		return subject
	}

	// Skip if already partitioned or is cross-domain
	if parsed.HID != "" || parsed.IsCrossDomain() {
		return subject
	}

	// Add H-id to domain part
	builder := NewSubjectBuilder().WithHID(p.hid)
	partitioned := builder.Build(parsed.Domain, parsed.Parts...)

	p.mu.Lock()
	p.subjects[subject] = partitioned
	p.mu.Unlock()

	return partitioned
}

// Departitionize removes H-id partitioning from a subject
func (p *Partition) Departitionize(subject string) string {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return subject
	}

	// Not a partitioned subject
	if parsed.HID == "" {
		return subject
	}

	// Remove H-id from parts
	parsed.HID = ""

	return parsed.String()
}

// IsInPartition checks if a subject belongs to this partition's H-id
func (p *Partition) IsInPartition(subject string) bool {
	if p.hid == "" {
		return true // No partitioning means all subjects match
	}

	parsed := ParseSubject(subject)
	if parsed == nil {
		return false
	}

	// Check cross-domain subjects
	if parsed.IsCrossDomain() {
		return parsed.ToHID == p.hid
	}

	// Check regular subjects
	return parsed.HID == p.hid || parsed.HID == ""
}

// GetHID returns the H-id for this partition
func (p *Partition) GetHID() string {
	return p.hid
}

// SetHID sets the H-id for this partition and clears the cache
func (p *Partition) SetHID(hid string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hid = hid
	p.subjects = make(map[string]string)
}

// SubscribeFilter creates a wildcard subscription subject for this partition
func (p *Partition) SubscribeFilter(domain SubjectDomain, suffix string) string {
	builder := NewSubjectBuilder().WithHID(p.hid)
	return builder.BuildWildcard(domain, suffix)
}

// GetAllHIDSubjects gets all subject variants for a given base subject
// This is useful for publishing to multiple partitions
func GetAllHIDSubjects(subject string, hids []string) []string {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return []string{subject}
	}

	if parsed.HID != "" || parsed.IsCrossDomain() {
		return []string{subject} // Already has H-id or is cross-domain
	}

	subjects := make([]string, 0, len(hids))
	for _, hid := range hids {
		builder := NewSubjectBuilder().WithHID(hid)
		partitioned := builder.Build(parsed.Domain, parsed.Parts...)
		subjects = append(subjects, partitioned)
	}

	return subjects
}

// IsSubjectForHID checks if a subject is intended for a specific H-id
func IsSubjectForHID(subject, hid string) bool {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return false
	}

	if parsed.IsCrossDomain() {
		return parsed.ToHID == hid
	}

	return parsed.HID == hid
}

// ExtractHIDFromSubject extracts the H-id from a subject
func ExtractHIDFromSubject(subject string) string {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return ""
	}

	if parsed.IsCrossDomain() {
		return parsed.FromHID
	}

	return parsed.HID
}

// SubjectWithHID creates a new subject with the given H-id
func SubjectWithHID(subject, hid string) string {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return subject
	}

	if parsed.IsCrossDomain() {
		// For cross-domain, modify from HID
		parsed.FromHID = hid
		return parsed.String()
	}

	if parsed.HID != "" {
		// Already has H-id, replace it
		parsed.HID = hid
		return parsed.String()
	}

	// Add H-id
	builder := NewSubjectBuilder().WithHID(hid)
	return builder.Build(parsed.Domain, parsed.Parts...)
}

// SubjectRouter routes subjects to appropriate H-id partitions
type SubjectRouter struct {
	partitions map[string]*Partition // hid -> partition
	mu         sync.RWMutex
	localHID   string
}

// NewSubjectRouter creates a new subject router
func NewSubjectRouter(localHID string) *SubjectRouter {
	return &SubjectRouter{
		partitions: make(map[string]*Partition),
		localHID:   localHID,
	}
}

// AddPartition adds a partition for an H-id
func (r *SubjectRouter) AddPartition(hid string) *Partition {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.partitions[hid]; !exists {
		r.partitions[hid] = NewPartition(hid)
	}

	return r.partitions[hid]
}

// RemovePartition removes a partition
func (r *SubjectRouter) RemovePartition(hid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.partitions, hid)
}

// GetPartition gets a partition for an H-id
func (r *SubjectRouter) GetPartition(hid string) *Partition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.partitions[hid]
}

// GetLocalPartition gets the local partition
func (r *SubjectRouter) GetLocalPartition() *Partition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.partitions[r.localHID]
}

// Route determines which partition(s) a subject should be routed to
func (r *SubjectRouter) Route(subject string) []string {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return []string{r.localHID}
	}

	// Cross-domain subjects route to specific H-id
	if parsed.IsCrossDomain() {
		return []string{parsed.ToHID}
	}

	// Subjects with H-id route to that H-id
	if parsed.HID != "" {
		return []string{parsed.HID}
	}

	// Broadcast subjects route to all partitions
	hids := make([]string, 0, len(r.partitions))
	r.mu.RLock()
	for hid := range r.partitions {
		hids = append(hids, hid)
	}
	r.mu.RUnlock()

	return hids
}

// IsLocal checks if a subject is for the local partition
func (r *SubjectRouter) IsLocal(subject string) bool {
	hids := r.Route(subject)
	for _, hid := range hids {
		if hid == r.localHID {
			return true
		}
	}
	return false
}

// TransformForPartition transforms a subject for a specific target partition
func (r *SubjectRouter) TransformForPartition(subject, targetHID string) string {
	parsed := ParseSubject(subject)
	if parsed == nil {
		return subject
	}

	// If it's a cross-domain subject, check if it's for us
	if parsed.IsCrossDomain() {
		if parsed.ToHID == r.localHID {
			// This is for us, transform to local
			builder := NewSubjectBuilder().WithHID(r.localHID)
			return builder.Build(parsed.Domain, parsed.Parts...)
		}
		return subject // Not for us, leave as-is
	}

	// Regular subject - add target H-id if needed
	if parsed.HID == "" {
		builder := NewSubjectBuilder().WithHID(targetHID)
		return builder.Build(parsed.Domain, parsed.Parts...)
	}

	return subject // Already has H-id
}

// CreateCrossSubject creates a cross-domain subject from local to remote
func (r *SubjectRouter) CreateCrossSubject(localSubject, remoteHID string) string {
	parsed := ParseSubject(localSubject)
	if parsed == nil {
		return localSubject
	}

	// Build cross-domain subject
	builder := NewSubjectBuilder()
	parts := append([]string{string(parsed.Domain)}, parsed.Parts...)
	return builder.BuildCross(r.localHID, remoteHID, parts...)
}

// ValidateSubject validates that a subject is properly formatted
func ValidateSubject(subject string) bool {
	if subject == "" {
		return false
	}

	parts := strings.Split(subject, ".")
	if len(parts) < 2 {
		return false
	}

	// Check for valid prefix
	if parts[0] != "picoclaw" {
		return false
	}

	// Check for valid domain
	validDomains := map[SubjectDomain]bool{
		DomainSwarm:      true,
		DomainMemory:     true,
		DomainTask:       true,
		DomainDiscovery:  true,
		DomainSystem:     true,
		DomainCross:      true,
	}

	if !validDomains[SubjectDomain(parts[1])] {
		return false
	}

	return true
}

// SanitizeSubject sanitizes a subject for safe use
func SanitizeSubject(subject string) string {
	// Remove any whitespace
	subject = strings.TrimSpace(subject)

	// Ensure it starts with picoclaw.
	if !strings.HasPrefix(subject, "picoclaw.") {
		return subject
	}

	// Convert to lowercase (H-ids are case-insensitive)
	parts := strings.Split(subject, ".")
	for i, part := range parts {
		if i > 1 && !strings.HasPrefix(part, ">") && !strings.HasPrefix(part, "*") {
			parts[i] = strings.ToLower(part)
		}
	}

	return strings.Join(parts, ".")
}
