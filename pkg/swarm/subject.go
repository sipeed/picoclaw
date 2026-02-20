// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/identity"
)

// SubjectDomain represents different domains in the swarm
type SubjectDomain string

const (
	// DomainSwarm is for general swarm messages
	DomainSwarm SubjectDomain = "swarm"
	// DomainMemory is for memory-related messages
	DomainMemory SubjectDomain = "memory"
	// DomainTask is for task messages
	DomainTask SubjectDomain = "task"
	// DomainDiscovery is for node discovery
	DomainDiscovery SubjectDomain = "discovery"
	// DomainSystem is for system messages
	DomainSystem SubjectDomain = "system"
	// DomainCross is for cross H-id communication
	DomainCross SubjectDomain = "x"
)

// SubjectBuilder builds NATS subjects with H-id partitioning
type SubjectBuilder struct {
	// hid is the H-id for this instance (nil = no partitioning)
	hid string

	// prefix is the subject prefix (default: "picoclaw")
	prefix string
}

// NewSubjectBuilder creates a new subject builder
func NewSubjectBuilder() *SubjectBuilder {
	return &SubjectBuilder{
		prefix: "picoclaw",
	}
}

// WithHID sets the H-id for partitioning
func (b *SubjectBuilder) WithHID(hid string) *SubjectBuilder {
	b.hid = hid
	return b
}

// WithIdentity sets the H-id from an identity
func (b *SubjectBuilder) WithIdentity(id *identity.Identity) *SubjectBuilder {
	if id != nil {
		b.hid = id.HID
	}
	return b
}

// WithPrefix sets a custom prefix
func (b *SubjectBuilder) WithPrefix(prefix string) *SubjectBuilder {
	b.prefix = prefix
	return b

}

// Build builds a subject with the given domain and parts
// Format: {prefix}.{domain}[.{hid}].{parts...}
func (b *SubjectBuilder) Build(domain SubjectDomain, parts ...string) string {
	sb := strings.Builder{}
	sb.WriteString(b.prefix)
	sb.WriteString(".")
	sb.WriteString(string(domain))

	if b.hid != "" {
		sb.WriteString(".")
		sb.WriteString(b.hid)
	}

	for _, part := range parts {
		sb.WriteString(".")
		sb.WriteString(part)
	}

	return sb.String()
}

// BuildCross builds a subject for cross H-id communication
// Format: {prefix}.x.{from_hid}.{to_hid}.{parts...}
func (b *SubjectBuilder) BuildCross(fromHID, toHID string, parts ...string) string {
	sb := strings.Builder{}
	sb.WriteString(b.prefix)
	sb.WriteString(".")
	sb.WriteString(string(DomainCross))
	sb.WriteString(".")
	sb.WriteString(fromHID)
	sb.WriteString(".")
	sb.WriteString(toHID)

	for _, part := range parts {
		sb.WriteString(".")
		sb.WriteString(part)
	}

	return sb.String()
}

// BuildWildcard builds a wildcard subject for subscribing
// If HID is set, creates a HID-specific wildcard, otherwise creates a global wildcard
func (b *SubjectBuilder) BuildWildcard(domain SubjectDomain, suffix string) string {
	if b.hid != "" {
		return fmt.Sprintf("%s.%s.%s.%s", b.prefix, domain, b.hid, suffix)
	}
	return fmt.Sprintf("%s.%s.%s", b.prefix, domain, suffix)
}

// ParseSubject parses a NATS subject into its components
func ParseSubject(s string) *ParsedSubject {
	parts := strings.Split(s, ".")

	if len(parts) < 2 {
		return nil
	}

	ps := &ParsedSubject{
		Prefix: parts[0],
	}

	if len(parts) < 3 {
		return ps
	}

	ps.Domain = SubjectDomain(parts[1])

	// Check if this is a cross-domain subject
	if ps.Domain == DomainCross && len(parts) >= 4 {
		ps.FromHID = parts[2]
		ps.ToHID = parts[3]
		if len(parts) > 4 {
			ps.Parts = parts[4:]
		}
		return ps
	}

	// Regular subject - check if third part is an H-id
	if len(parts) >= 3 {
		// Heuristic: if third part looks like an H-id (contains "user-", "org-", "group-")
		// treat it as H-id, otherwise treat it as regular parts
		if isLikelyHID(parts[2]) {
			ps.HID = parts[2]
			if len(parts) > 3 {
				ps.Parts = parts[3:]
			}
		} else {
			ps.Parts = parts[2:]
		}
	}

	return ps
}

// isLikelyHID checks if a string is likely an H-id
func isLikelyHID(s string) bool {
	return strings.HasPrefix(s, "user-") ||
		strings.HasPrefix(s, "org-") ||
		strings.HasPrefix(s, "group-") ||
		strings.HasPrefix(s, "tenant-")
}

// ParsedSubject represents a parsed NATS subject
type ParsedSubject struct {
	Prefix  string
	Domain  SubjectDomain
	HID     string
	FromHID string // For cross-domain subjects
	ToHID   string // For cross-domain subjects
	Parts   []string
}

// String reconstructs the subject string
func (ps *ParsedSubject) String() string {
	sb := strings.Builder{}
	sb.WriteString(ps.Prefix)
	sb.WriteString(".")
	sb.WriteString(string(ps.Domain))

	if ps.FromHID != "" && ps.ToHID != "" {
		sb.WriteString(".")
		sb.WriteString(ps.FromHID)
		sb.WriteString(".")
		sb.WriteString(ps.ToHID)
		for _, part := range ps.Parts {
			sb.WriteString(".")
			sb.WriteString(part)
		}
		return sb.String()
	}

	if ps.HID != "" {
		sb.WriteString(".")
		sb.WriteString(ps.HID)
	}

	for _, part := range ps.Parts {
		sb.WriteString(".")
		sb.WriteString(part)
	}

	return sb.String()
}

// IsCrossDomain returns true if this is a cross-domain subject
func (ps *ParsedSubject) IsCrossDomain() bool {
	return ps.Domain == DomainCross
}

// GetHID returns the H-id from the subject (either FromHID or HID)
func (ps *ParsedSubject) GetHID() string {
	if ps.IsCrossDomain() {
		return ps.FromHID
	}
	return ps.HID
}

// Helper functions for common subjects

// HeartbeatSubject builds a heartbeat subject for a node
func HeartbeatSubject(hid, nodeID string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainSwarm, "heartbeat", nodeID)
}

// DiscoverySubject builds a discovery subject
func DiscoverySubject(hid string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainDiscovery, "announce")
}

// TaskAssignSubject builds a task assignment subject
func TaskAssignSubject(hid, nodeID string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainTask, "assign", nodeID)
}

// TaskResultSubject builds a task result subject
func TaskResultSubject(hid, taskID string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainTask, "result", taskID)
}

// MemoryStoreSubject builds a memory store subject
func MemoryStoreSubject(hid, memoryID string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainMemory, "store", memoryID)
}

// MemoryQuerySubject builds a memory query subject
func MemoryQuerySubject(hid string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainMemory, "query")
}

// SystemShutdownSubject builds a system shutdown subject
func SystemShutdownSubject(hid, nodeID string) string {
	b := NewSubjectBuilder()
	if hid != "" {
		b = b.WithHID(hid)
	}
	return b.Build(DomainSystem, "shutdown", nodeID)
}

// CrossHIDSubject builds a cross H-id communication subject
func CrossHIDSubject(fromHID, toHID string, action string) string {
	b := NewSubjectBuilder()
	return b.BuildCross(fromHID, toHID, action)
}

// LegacySubjectBuilder converts new-style subjects to legacy format
func LegacySubject(subject string) string {
	// For backward compatibility, return subject as-is
	// When legacy clients are removed, this can be deleted
	return subject
}
