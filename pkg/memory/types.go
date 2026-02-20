// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package memory

// MemoryScope defines the visibility scope of a memory item
type MemoryScope int

const (
	// ScopePrivate indicates memory is only accessible by the owning S-id
	ScopePrivate MemoryScope = iota

	// ScopeShared indicates memory is accessible by all S-ids within the same H-id
	ScopeShared

	// ScopePublic indicates memory is accessible by any H-id (with authorization)
	ScopePublic
)

// String returns the string representation of the scope
func (s MemoryScope) String() string {
	switch s {
	case ScopePrivate:
		return "private"
	case ScopeShared:
		return "shared"
	case ScopePublic:
		return "public"
	default:
		return "unknown"
	}
}

// ParseMemoryScope parses a string into a MemoryScope
func ParseMemoryScope(s string) (MemoryScope, error) {
	switch s {
	case "private":
		return ScopePrivate, nil
	case "shared":
		return ScopeShared, nil
	case "public":
		return ScopePublic, nil
	default:
		return ScopePrivate, ErrInvalidScope{s}
	}
}

// MarshalJSON implements json.Marshaler
func (s MemoryScope) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (s *MemoryScope) UnmarshalJSON(data []byte) error {
	str, err := unquoteString(data)
	if err != nil {
		return err
	}
	parsed, err := ParseMemoryScope(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// MemoryType defines the type of memory content
type MemoryType int

const (
	// TypeConversation stores conversation history
	TypeConversation MemoryType = iota

	// TypeKnowledge stores facts and knowledge
	TypeKnowledge

	// TypeContext stores temporary context information
	TypeContext

	// TypeToolResult stores tool execution results
	TypeToolResult

	// TypeUserPreference stores user preferences
	TypeUserPreference

	// TypeSystem stores system-level information
	TypeSystem
)

// String returns the string representation of the memory type
func (t MemoryType) String() string {
	switch t {
	case TypeConversation:
		return "conversation"
	case TypeKnowledge:
		return "knowledge"
	case TypeContext:
		return "context"
	case TypeToolResult:
		return "tool_result"
	case TypeUserPreference:
		return "user_preference"
	case TypeSystem:
		return "system"
	default:
		return "unknown"
	}
}

// ParseMemoryType parses a string into a MemoryType
func ParseMemoryType(s string) (MemoryType, error) {
	switch s {
	case "conversation":
		return TypeConversation, nil
	case "knowledge":
		return TypeKnowledge, nil
	case "context":
		return TypeContext, nil
	case "tool_result", "tool-result":
		return TypeToolResult, nil
	case "user_preference", "user-preference":
		return TypeUserPreference, nil
	case "system":
		return TypeSystem, nil
	default:
		return TypeContext, ErrInvalidType{s}
	}
}

// MarshalJSON implements json.Marshaler
func (t MemoryType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (t *MemoryType) UnmarshalJSON(data []byte) error {
	str, err := unquoteString(data)
	if err != nil {
		return err
	}
	parsed, err := ParseMemoryType(str)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// MemoryItemType is an alias for backward compatibility
type MemoryItemType = MemoryType

// PermissionError represents a permission check failure
type PermissionError struct {
	Operation string
	Scope     MemoryScope
	Reason    string
}

func (e PermissionError) Error() string {
	if e.Reason != "" {
		return e.Reason
	}
	return "permission denied: " + e.Operation + " on " + e.Scope.String()
}

// ErrInvalidScope is returned when an invalid scope is provided
type ErrInvalidScope struct {
	Scope string
}

func (e ErrInvalidScope) Error() string {
	return "invalid memory scope: " + e.Scope
}

// ErrInvalidType is returned when an invalid memory type is provided
type ErrInvalidType struct {
	Type string
}

func (e ErrInvalidType) Error() string {
	return "invalid memory type: " + e.Type
}

// helper function to unquote JSON strings
func unquoteString(data []byte) (string, error) {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		return string(data[1 : len(data)-1]), nil
	}
	return "", ErrInvalidType{"not a quoted string"}
}

// ScopeLevel returns the numeric level of a scope for comparison
func (s MemoryScope) ScopeLevel() int {
	switch s {
	case ScopePrivate:
		return 0
	case ScopeShared:
		return 1
	case ScopePublic:
		return 2
	default:
		return 0
	}
}

// IsMorePermissive returns true if this scope is more permissive than the other
func (s MemoryScope) IsMorePermissive(other MemoryScope) bool {
	return s.ScopeLevel() > other.ScopeLevel()
}

// IsLessPermissive returns true if this scope is less permissive than the other
func (s MemoryScope) IsLessPermissive(other MemoryScope) bool {
	return s.ScopeLevel() < other.ScopeLevel()
}
