// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package identity

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// ULIDType represents the type of ULID
type ULIDType string

const (
	ULIDTypeNode     ULIDType = "node"
	ULIDTypeTask     ULIDType = "task"
	ULIDTypeSession  ULIDType = "session"
	ULIDTypeMemory   ULIDType = "mem"
	ULIDTypeWorkflow ULIDType = "workflow"
	ULIDTypeDevice   ULIDType = "device"
	ULIDTypeUser     ULIDType = "user"
	ULIDTypeCustom   ULIDType = "custom"
)

// String returns the string representation of ULIDType
func (t ULIDType) String() string {
	return string(t)
}

// ULIDVariant represents the ULID format variant
type ULIDVariant string

const (
	ULIDVariantUUID     ULIDVariant = "uuid"     // Standard UUID v4: 8-4-4-4-4-4-7-4-4-4
	ULIDVariantULIDv7    ULIDVariant = "ulidv7"   // ULID v7: 8-4-4-4-4-4-7
	ULIDVariantULID     ULIDVariant = "ulid"     // ULID: 26 chars Crockford base32
	ULIDVariantNano     ULIDVariant = "nano"     // Nano: 21 chars
	ULIDVariantShort    ULIDVariant = "short"    // Short: 8 chars (first 8 of UUID)
	ULIDVariantBase62   ULIDVariant = "base62"   // Base62 encoded
	ULIDVariantCustom   ULIDVariant = "custom"   // Custom format
)

// String returns the string representation of ULIDVariant
func (v ULIDVariant) String() string {
	return string(v)
}

// ULID represents a ULID (Universal Unique IDentifier)
type ULID struct {
	Type     ULIDType
	Variant  ULIDVariant
	Value    string
	Prefix   string
	Time     time.Time
	Metadata map[string]string
}

// ULIDGenerator generates ULIDs
type ULIDGenerator struct {
	mu       sync.RWMutex
	entropy  *ulid.MonotonicEntropy
	prefix   string
}

// NewULIDGenerator creates a new ULID generator
func NewULIDGenerator() *ULIDGenerator {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return &ULIDGenerator{
		entropy: entropy,
	}
}

// WithPrefix sets the prefix for generated ULIDs
func (g *ULIDGenerator) WithPrefix(prefix string) *ULIDGenerator {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prefix = prefix
	return g
}

// Generate generates a new ULID with default UUID variant
func (g *ULIDGenerator) Generate() *ULID {
	return g.GenerateWithType(ULIDTypeNode, ULIDVariantUUID)
}

// GenerateWithType generates a ULID with specific type and variant
func (g *ULIDGenerator) GenerateWithType(ulidType ULIDType, variant ULIDVariant) *ULID {
	u := &ULID{
		Type:     ulidType,
		Variant:  variant,
		Time:     time.Now(),
		Metadata: make(map[string]string),
	}

	switch variant {
	case ULIDVariantUUID:
		// UUID v4 format
		id := uuid.New()
		u.Value = id.String()

	case ULIDVariantULIDv7:
		// ULID v7 format
		id := ulid.Make()
		u.Value = id.String()

	case ULIDVariantULID:
		// ULID (26 chars, Crockford base32)
		id := ulid.Make()
		u.Value = id.String()

	case ULIDVariantNano:
		// Nano format: 21 chars (21 chars from ULID without prefix)
		// Generate a ULID and extract components
		id, err := ulid.New(ulid.Timestamp(time.Now()), g.entropy)
		if err != nil {
			// Fallback to simple format
			id = ulid.MustNew(ulid.Timestamp(time.Now()), ulid.DefaultEntropy())
		}
		// Use full 26 chars from ULID, but store type prefix separately
		u.Value = id.String()
		u.Prefix = getNanoTypeChar(ulidType)

	case ULIDVariantShort:
		// Short format: 8 chars (UUID prefix)
		id := uuid.New()
		u.Value = strings.ReplaceAll(id.String(), "-", "")[:8]

	case ULIDVariantBase62:
		// Base62 encoded (using ULID as base)
		id := ulid.Make()
		// Encode the full ULID bytes as base62
		ulidBytes := id.Bytes()
		// Convert to base62
		var n uint64
		for i := 0; i < 8; i++ {
			n = n<<8 + uint64(ulidBytes[i])
		}
		u.Value = base62Encode(n)

	default:
		// Default to UUID v4
		id := uuid.New()
		u.Value = id.String()
	}

	// Add prefix if specified
	if g.prefix != "" {
		u.Prefix = g.prefix
	}

	return u
}

// String returns the full ULID string
func (u *ULID) String() string {
	if u == nil {
		return ""
	}

	parts := []string{}
	if u.Prefix != "" {
		parts = append(parts, u.Prefix, "-")
	}
	parts = append(parts, u.Value)

	return strings.Join(parts, "-")
}

// GenerateNodeID generates a node ULID
func (g *ULIDGenerator) GenerateNodeID() string {
	u := g.GenerateWithType(ULIDTypeNode, ULIDVariantUUID)
	return u.String()
}

// GenerateTaskID generates a task ULID
func (g *ULIDGenerator) GenerateTaskID() string {
	u := g.GenerateWithType(ULIDTypeTask, ULIDVariantUUID)
	return u.String()
}

// GenerateSessionID generates a session ULID
func (g *ULIDGenerator) GenerateSessionID() string {
	u := g.GenerateWithType(ULIDTypeSession, ULIDVariantUUID)
	return u.String()
}

// GenerateMemoryID generates a memory ULID
func (g *ULIDGenerator) GenerateMemoryID() string {
	u := g.GenerateWithType(ULIDTypeMemory, ULIDVariantUUID)
	return u.String()
}

// GetNanoTypeChar returns the type character for nano ULIDs
func getNanoTypeChar(ulidType ULIDType) string {
	switch ulidType {
	case ULIDTypeNode:
		return "n"
	case ULIDTypeTask:
		return "t"
	case ULIDTypeSession:
		return "s"
	case ULIDTypeMemory:
		return "m"
	case ULIDTypeWorkflow:
		return "w"
	case ULIDTypeDevice:
		return "d"
	case ULIDTypeUser:
		return "u"
	default:
		return "x"
	}
}

// base62Encode encodes a uint64 to base62 string
func base62Encode(n uint64) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}

	var result []byte
	for n > 0 {
		result = append(result, charset[n%62])
		n /= 62
	}

	// Reverse result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// ParseULID parses a ULID string into its components
func ParseULID(s string) (*ULID, error) {
	if s == "" {
		return nil, fmt.Errorf("empty ULID")
	}

	u := &ULID{
		Metadata: make(map[string]string),
	}

	// Detect format by checking string characteristics
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		switch len(parts) {
		case 5:
			// Standard UUID: 8-4-4-4-4
			if len(parts[0]) == 8 && len(parts[1]) == 4 && len(parts[2]) == 4 &&
			   len(parts[3]) == 4 && len(parts[4]) == 12 {
				u.Variant = ULIDVariantUUID
				u.Value = s
				return u, nil
			}

		case 4:
			// ULIDv7 or ULID: 8-4-4-4-4-4
			if len(parts[0]) == 8 && len(parts[1]) == 4 && len(parts[2]) == 4 &&
			   len(parts[3]) == 4 && len(parts[4]) == 4 {
				// Check if it's ULIDv7 (version 7) or ULID (base32)
				if parts[4][0] == '7' {
					u.Variant = ULIDVariantULIDv7
				} else {
					u.Variant = ULIDVariantULID
				}
				u.Value = s
				return u, nil
			}

		case 3:
			// Nano format or with prefix
			if len(parts[0]) == 1 && len(parts[2]) == 21 {
				u.Variant = ULIDVariantNano
				u.Value = parts[2]
				u.Prefix = parts[0]
				return u, nil
			}
		}
	}

	// Try to parse as ULID
	if _, err := ulid.Parse(s); err == nil {
		u.Variant = ULIDVariantULID
		u.Value = s
		return u, nil
	}

	// Try to parse as UUID
	if _, err := uuid.Parse(s); err == nil {
		u.Variant = ULIDVariantUUID
		u.Value = s
		return u, nil
	}

	// Fallback: treat as custom value
	u.Variant = ULIDVariantCustom
	u.Value = s
	return u, nil
}

// IsStandard returns true if this is a standard ULID format
func (u *ULID) IsStandard() bool {
	return u.Variant == ULIDVariantUUID ||
		u.Variant == ULIDVariantULIDv7 ||
		u.Variant == ULIDVariantULID
}

// IsValid returns true if the ULID has a valid value
func (u *ULID) IsValid() bool {
	if u == nil || u.Value == "" {
		return false
	}
	switch u.Variant {
	case ULIDVariantUUID:
		// UUID should be 36 chars with dashes
		return len(u.Value) == 36
	case ULIDVariantULIDv7, ULIDVariantULID:
		// ULID should be 26 chars
		return len(u.Value) == 26
	case ULIDVariantNano:
		// Nano ULID uses 26 chars from ULID, prefix stored separately
		return len(u.Value) == 26
	case ULIDVariantShort:
		// Short ULID should be 8 chars
		return len(u.Value) == 8
	case ULIDVariantBase62:
		// Base62 should be 1-22 chars
		return len(u.Value) >= 1 && len(u.Value) <= 22
	case ULIDVariantCustom:
		// Custom can be any non-empty string
		return len(u.Value) > 0
	default:
		return len(u.Value) > 0
	}
}

// GetPrefix returns the type prefix for this ULID
func (u *ULID) GetPrefix() string {
	return getNanoTypeChar(u.Type)
}

// ExtractType extracts the ULID type from the ULID string
func ExtractType(ulidStr string) ULIDType {
	// Look for type prefix in nano format or custom prefix
	if strings.HasPrefix(ulidStr, "n-") || strings.HasPrefix(ulidStr, "t-") {
		return ULIDTypeTask
	}
	if strings.HasPrefix(ulidStr, "s-") {
		return ULIDTypeSession
	}
	if strings.HasPrefix(ulidStr, "m-") {
		return ULIDTypeMemory
	}
	if strings.HasPrefix(ulidStr, "w-") {
		return ULIDTypeWorkflow
	}
	if strings.HasPrefix(ulidStr, "d-") {
		return ULIDTypeDevice
	}
	if strings.HasPrefix(ulidStr, "u-") {
		return ULIDTypeUser
	}
	return ULIDTypeNode
}

// FormatULID formats a ULID with the specified variant
func FormatULID(id string, variant ULIDVariant) (string, error) {
	switch variant {
	case ULIDVariantUUID:
		// Validate as UUID v4
		if _, err := uuid.Parse(id); err != nil {
			return "", fmt.Errorf("invalid UUID: %w", err)
		}
		return id, nil

	case ULIDVariantULIDv7:
		// Validate as ULID v7
		if _, err := ulid.Parse(id); err != nil {
			return "", fmt.Errorf("invalid ULID: %w", err)
		}
		return id, nil

	case ULIDVariantULID:
		// Validate as ULID
		if _, err := ulid.Parse(id); err != nil {
			return "", fmt.Errorf("invalid ULID: %w", err)
		}
		return id, nil

	case ULIDVariantNano:
		// Validate as nano format: t-timestamp-random (21 chars)
		if len(id) != 21 {
			return "", fmt.Errorf("invalid nano ULID: wrong length")
		}
		// Check first char is type char
		if id[0] < 'a' || id[0] > 'z' {
			return "", fmt.Errorf("invalid nano ULID: wrong type char")
		}
		return id, nil

	case ULIDVariantShort:
		// Validate as 8-char format
		if len(id) != 8 {
			return "", fmt.Errorf("invalid short ULID: wrong length")
		}
		// Check hex chars
		for _, c := range id {
			if !isHexChar(byte(c)) {
				return "", fmt.Errorf("invalid short ULID: non-hex character")
			}
		}
		return id, nil

	case ULIDVariantBase62:
		// Base62 format validation
		if len(id) < 1 || len(id) > 22 {
			return "", fmt.Errorf("invalid base62 ULID: wrong length")
		}
		return id, nil

	default:
		return "", fmt.Errorf("unknown ULID variant: %s", variant)
	}
}

// isHexChar checks if a character is a hex digit
func isHexChar(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	default:
		return false
	}
}

// GenerateULID generates a new ULID with the specified variant
func GenerateULID(variant ULIDVariant) (string, error) {
	gen := NewULIDGenerator()
	u := gen.GenerateWithType(ULIDTypeNode, variant)
	return u.String(), nil
}

// MustGenerateULID generates a new ULID or panics
func MustGenerateULID(variant ULIDVariant) string {
	id, err := GenerateULID(variant)
	if err != nil {
		panic(err)
	}
	return id
}

// GenerateNodeULID generates a node ULID
func GenerateNodeULID() string {
	gen := NewULIDGenerator()
	return gen.GenerateNodeID()
}

// GenerateTaskULID generates a task ULID
func GenerateTaskULID() string {
	gen := NewULIDGenerator()
	return gen.GenerateTaskID()
}

// GenerateSessionULID generates a session ULID
func GenerateSessionULID() string {
	gen := NewULIDGenerator()
	return gen.GenerateSessionID()
}

// GenerateMemoryULID generates a memory ULID
func GenerateMemoryULID() string {
	gen := NewULIDGenerator()
	return gen.GenerateMemoryID()
}

// GenerateWorkflowULID generates a workflow ULID
func GenerateWorkflowULID() string {
	gen := NewULIDGenerator()
	u := gen.GenerateWithType(ULIDTypeWorkflow, ULIDVariantUUID)
	return u.String()
}

// GenerateCustomULID generates a custom ULID with specified prefix and variant
func GenerateCustomULID(prefix string, variant ULIDVariant) (string, error) {
	gen := NewULIDGenerator().WithPrefix(prefix)
	u := gen.GenerateWithType(ULIDTypeCustom, variant)
	return u.String(), nil
}

// ParseAndValidateULID parses a ULID string and validates it
func ParseAndValidateULID(id string) (*ULID, error) {
	u, err := ParseULID(id)
	if err != nil {
		return nil, err
	}

	// Additional validation based on variant
	switch u.Variant {
	case ULIDVariantUUID:
		if _, err := uuid.Parse(u.Value); err != nil {
			return nil, fmt.Errorf("invalid UUID: %w", err)
		}

	case ULIDVariantULIDv7, ULIDVariantULID:
		if _, err := ulid.Parse(u.Value); err != nil {
			return nil, fmt.Errorf("invalid ULID: %w", err)
		}

	case ULIDVariantNano:
		if len(u.Value) != 21 {
			return nil, fmt.Errorf("invalid nano ULID: wrong length %d", len(u.Value))
		}
	}

	return u, nil
}

// GenerateID generates a standard ULID (UUID v4 format)
func GenerateID() string {
	id := uuid.New()
	return id.String()
}

// GenerateIDWithPrefix generates a ULID with a custom prefix
func GenerateIDWithPrefix(prefix string) string {
	id := uuid.New()
	return fmt.Sprintf("%s-%s", prefix, id.String())
}

// GenerateShortID generates an 8-character ULID (first 8 chars of UUID)
func GenerateShortID() string {
	id := uuid.New()
	return strings.ReplaceAll(id.String(), "-", "")[:8]
}

// GenerateULIDv7 generates a ULID v7 format ULID
func GenerateULIDv7() string {
	id := ulid.Make()
	return id.String()
}

// GenerateNanoULID generates a nano-format ULID (21 chars)
func GenerateNanoULID(ulidType ULIDType) string {
	gen := NewULIDGenerator()
	u := gen.GenerateWithType(ulidType, ULIDVariantNano)
	return u.String()
}

// GenerateBase62ULID generates a base62 encoded ULID
func GenerateBase62ULID() string {
	id := ulid.Make()
	return base62Encode(id.Time())
}

// ValidateULID validates a ULID string
func ValidateULID(id string) error {
	_, err := ParseAndValidateULID(id)
	return err
}

// ULIDVersion represents the version of ULID
type ULIDVersion string

const (
	ULIDVersion4 ULIDVersion = "v4" // UUID v4
	ULIDVersion7 ULIDVersion = "v7" // ULID v7
)

// Version returns the ULID version
func (u *ULID) Version() ULIDVersion {
	if u.Variant == ULIDVariantULIDv7 || u.Variant == ULIDVariantULID {
		return ULIDVersion7
	}
	return ULIDVersion4
}

// GetTimestamp extracts the timestamp from the ULID if available
func (u *ULID) GetTimestamp() (time.Time, error) {
	switch u.Variant {
	case ULIDVariantULIDv7, ULIDVariantULID:
		id, err := ulid.Parse(u.Value)
		if err != nil {
			return time.Time{}, err
		}
		return ulid.Time(id.Time()), nil

	case ULIDVariantUUID:
		// UUID v4 doesn't have embedded timestamp
		// Use creation time from metadata if available
		if !u.Time.IsZero() {
			return u.Time, nil
		}
		return time.Time{}, fmt.Errorf("UUID v4 does not contain timestamp")

	default:
		return u.Time, nil
	}
}

// Compare compares two ULIDs by timestamp (for sorting)
func Compare(a, b *ULID) int {
	timeA, _ := a.GetTimestamp()
	timeB, _ := b.GetTimestamp()

	if timeA.Before(timeB) {
		return -1
	} else if timeA.After(timeB) {
		return 1
	}
	return 0
}

// ULIDSet manages a set of ULIDs with efficient lookup
type ULIDSet struct {
	ulids map[string]*ULID
	mu    sync.RWMutex
}

// NewULIDSet creates a new ULID set
func NewULIDSet() *ULIDSet {
	return &ULIDSet{
		ulids: make(map[string]*ULID),
	}
}

// Add adds a ULID to the set
func (s *ULIDSet) Add(ulid *ULID) error {
	if ulid == nil {
		return fmt.Errorf("cannot add nil ULID")
	}

	// Validate first
	if err := ValidateULID(ulid.String()); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ulid.String()
	s.ulids[key] = ulid
	return nil
}

// Get retrieves a ULID by its string value
func (s *ULIDSet) Get(id string) (*ULID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ulid, ok := s.ulids[id]
	return ulid, ok
}

// Remove removes a ULID from the set
func (s *ULIDSet) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.ulids, id)
}

// List returns all ULIDs in the set
func (s *ULIDSet) List() []*ULID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ulids := make([]*ULID, 0, len(s.ulids))
	for _, ulid := range s.ulids {
		ulids = append(ulids, ulid)
	}

	// Sort by timestamp (newest first)
	for i := 0; i < len(ulids); i++ {
		for j := i + 1; j < len(ulids); j++ {
			if Compare(ulids[j], ulids[i]) < 0 {
				ulids[i], ulids[j] = ulids[j], ulids[i]
			}
		}
	}

	return ulids
}

// FilterByType returns ULIDs of a specific type
func (s *ULIDSet) FilterByType(ulidType ULIDType) []*ULID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ULID, 0)
	for _, ulid := range s.ulids {
		if ulid.Type == ulidType {
			result = append(result, ulid)
		}
	}

	return result
}

// FilterByVariant returns ULIDs of a specific variant
func (s *ULIDSet) FilterByVariant(variant ULIDVariant) []*ULID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ULID, 0)
	for _, ulid := range s.ulids {
		if ulid.Variant == variant {
			result = append(result, ulid)
		}
	}

	return result
}

// Count returns the number of ULIDs in the set
func (s *ULIDSet) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.ulids)
}

// Len returns the number of ULIDs in the set (alias for Count)
func (s *ULIDSet) Len() int {
	return s.Count()
}

// Clear removes all ULIDs from the set
func (s *ULIDSet) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ulids = make(map[string]*ULID)
}

// ToSlice converts the set to a slice of ULID strings
func (s *ULIDSet) ToSlice() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.ulids))
	for id := range s.ulids {
		ids = append(ids, id)
	}
	return ids
}

// Contains checks if a ULID is in the set
func (s *ULIDSet) Contains(ulid *ULID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.ulids[ulid.String()]
	return ok
}

// Merge merges another ULIDSet into this one and returns a new set
func (s *ULIDSet) Merge(other *ULIDSet) *ULIDSet {
	result := NewULIDSet()

	s.mu.RLock()
	for k, v := range s.ulids {
		result.ulids[k] = v
	}
	s.mu.RUnlock()

	other.mu.RLock()
	for k, v := range other.ulids {
		result.ulids[k] = v
	}
	other.mu.RUnlock()

	return result
}

// Difference returns a new ULIDSet with elements in s but not in other
func (s *ULIDSet) Difference(other *ULIDSet) *ULIDSet {
	result := NewULIDSet()

	s.mu.RLock()
	defer s.mu.RUnlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for k, v := range s.ulids {
		if _, exists := other.ulids[k]; !exists {
			result.ulids[k] = v
		}
	}

	return result
}

// ULIDPool manages a pool of pre-generated ULIDs
type ULIDPool struct {
	gen    *ULIDGenerator
	ulids  chan *ULID
	mu     sync.Mutex
	closed bool
}

// NewULIDPool creates a new ULID pool
func NewULIDPool(size int, gen *ULIDGenerator) *ULIDPool {
	if gen == nil {
		gen = NewULIDGenerator()
	}

	p := &ULIDPool{
		gen:   gen,
		ulids: make(chan *ULID, size),
	}

	// Pre-fill the pool
	go p.fill(size)

	return p
}

// fill fills the pool with ULIDs
func (p *ULIDPool) fill(count int) {
	for i := 0; i < count; i++ {
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()

		if closed {
			return
		}

		select {
		case p.ulids <- p.gen.Generate():
		default:
			// Pool full, stop filling
			return
		}
	}
}

// Get gets a ULID from the pool or generates a new one
func (p *ULIDPool) Get() *ULID {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return p.gen.Generate()
	}

	select {
	case ulid := <-p.ulids:
		// Async refill
		go func() {
			p.fill(1)
		}()
		return ulid
	default:
		return p.gen.Generate()
	}
}

// GetWithType gets a ULID of specific type from the pool
func (p *ULIDPool) GetWithType(ulidType ULIDType) *ULID {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return p.gen.GenerateWithType(ulidType, ULIDVariantUUID)
	}

	// Try to get from pool
	for {
		select {
		case ulid := <-p.ulids:
			if ulid.Type == ulidType {
				// Async refill
				go func() {
					p.fill(1)
				}()
				return ulid
			}
			// Wrong type, put back and try next
			p.ulids <- ulid

		default:
			// Pool empty, generate new
			return p.gen.GenerateWithType(ulidType, ULIDVariantUUID)
		}
	}
}

// Close closes the pool
func (p *ULIDPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	close(p.ulids)

	// Drain remaining ULIDs
	for ulid := range p.ulids {
		_ = ulid
	}
}

// Size returns the current pool size
func (p *ULIDPool) Size() int {
	return len(p.ulids)
}

// IsClosed returns true if the pool is closed
func (p *ULIDPool) IsClosed() bool {
	return p.closed
}
