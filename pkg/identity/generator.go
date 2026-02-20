// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package identity

import (
	"fmt"
	"sync"
)

const (
	
	DefaultIDLength = 8
)

var (
	// Default ULID generator
	defaultULIDGen     *ULIDGenerator
	defaultULIDGenOnce sync.Once
)

func getDefaultULIDGenerator() *ULIDGenerator {
	defaultULIDGenOnce.Do(func() {
		defaultULIDGen = NewULIDGenerator()
	})
	return defaultULIDGen
}

// Generator handles identity generation using ULID
type Generator struct {
	hidPrefix     string
	sidPrefix     string
	idLength      int // For legacy mode
	ulidGen       *ULIDGenerator
	ulidVariant   ULIDVariant
	ulidType      ULIDType
	ulidEnabled   bool
	useLegacyMode bool // For backwards compatibility
}

// NewGenerator creates a new identity generator
func NewGenerator() *Generator {
	return &Generator{
		hidPrefix:     DefaultHIDPrefix,
		sidPrefix:     DefaultSIDPrefix,
		idLength:      DefaultIDLength,
		ulidGen:       getDefaultULIDGenerator(),
		ulidVariant:   ULIDVariantNano, // Default to 21-char Nano ULIDs
		ulidType:      ULIDTypeNode,
		ulidEnabled:   true,
		useLegacyMode: false,
	}
}

// WithHIDPrefix sets the H-id prefix for generation
func (g *Generator) WithHIDPrefix(prefix string) *Generator {
	g.hidPrefix = normalizeID(prefix)
	return g
}

// WithSIDPrefix sets the S-id prefix for generation
func (g *Generator) WithSIDPrefix(prefix string) *Generator {
	g.sidPrefix = normalizeID(prefix)
	return g
}

// WithIDLength sets the random ID length (DEPRECATED - use WithULIDVariant)
func (g *Generator) WithIDLength(length int) *Generator {
	g.idLength = length
	g.useLegacyMode = true
	return g
}

// WithULIDVariant sets the ULID variant for generation
func (g *Generator) WithULIDVariant(variant ULIDVariant) *Generator {
	g.ulidVariant = variant
	g.useLegacyMode = false
	g.ulidEnabled = true
	return g
}

// WithULIDType sets the ULID type for generation
func (g *Generator) WithULIDType(ulidType ULIDType) *Generator {
	g.ulidType = ulidType
	return g
}

// WithULIDGenerator sets a custom ULID generator
func (g *Generator) WithULIDGenerator(gen *ULIDGenerator) *Generator {
	g.ulidGen = gen
	g.ulidEnabled = true
	return g
}

// WithLegacyMode enables legacy random string generation (non-ULID)
func (g *Generator) WithLegacyMode() *Generator {
	g.useLegacyMode = true
	g.ulidEnabled = false
	return g
}

// GenerateHID generates a new H-id with the configured prefix and ULID suffix
func (g *Generator) GenerateHID() string {
	if g.useLegacyMode {
		suffix := randomString(g.idLength)
		return fmt.Sprintf("%s-%s", g.hidPrefix, suffix)
	}
	ulid := g.ulidGen.GenerateWithType(g.ulidType, g.ulidVariant)
	return fmt.Sprintf("%s-%s", g.hidPrefix, ulid.Value)
}

// GenerateSID generates a new S-id with the configured prefix and ULID suffix
func (g *Generator) GenerateSID() string {
	if g.useLegacyMode {
		suffix := randomString(g.idLength)
		return fmt.Sprintf("%s-%s", g.sidPrefix, suffix)
	}
	ulid := g.ulidGen.GenerateWithType(g.ulidType, g.ulidVariant)
	return fmt.Sprintf("%s-%s", g.sidPrefix, ulid.Value)
}

// Generate generates a complete new Identity
func (g *Generator) Generate() *Identity {
	hid := g.GenerateHID()
	sid := g.GenerateSID()
	return NewIdentity(hid, sid)
}

// GenerateWithHID generates an Identity with a specific H-id and random S-id
func (g *Generator) GenerateWithHID(hid string) *Identity {
	sid := g.GenerateSID()
	return NewIdentity(hid, sid)
}

// GenerateWithSID generates an Identity with a specific S-id and random H-id
func (g *Generator) GenerateWithSID(sid string) *Identity {
	hid := g.GenerateHID()
	return NewIdentity(hid, sid)
}

// GenerateULID generates a typed ULID
func (g *Generator) GenerateULID(ulidType ULIDType) *ULID {
	return g.ulidGen.GenerateWithType(ulidType, g.ulidVariant)
}

// randomString generates a random string of the given length
// DEPRECATED: Use ULID generation instead
func randomString(length int) string {
	// Use ULID short variant for legacy compatibility
	ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeCustom, ULIDVariantShort)
	if len(ulid.Value) >= length {
		return ulid.Value[:length]
	}
	return ulid.Value
}

// GenerateShortIDL generates a short random ID with specified length
// Note: Use GenerateShortID() from ulid.go for default 8-char IDs
func GenerateShortIDL(length int) string {
	if length <= 0 {
		length = DefaultIDLength
	}
	if length <= 8 {
		// Use short ULID (8 chars)
		ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeCustom, ULIDVariantShort)
		if len(ulid.Value) >= length {
			return ulid.Value[:length]
		}
		return ulid.Value
	}
	// Use Nano ULID for longer requests
	ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeCustom, ULIDVariantNano)
	if len(ulid.Value) >= length {
		return ulid.Value[:length]
	}
	return ulid.Value
}

// GenerateNodeID generates a node ID using ULID: "claw-{ulid}"
func GenerateNodeID() string {
	ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeNode, ULIDVariantNano)
	return fmt.Sprintf("claw-%s", ulid.Value)
}

// GenerateTaskID generates a task ID using ULID: "task-{ulid}"
func GenerateTaskID() string {
	ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeTask, ULIDVariantNano)
	return fmt.Sprintf("task-%s", ulid.Value)
}

// GenerateSessionID generates a session ID using ULID: "session-{ulid}"
func GenerateSessionID() string {
	ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeSession, ULIDVariantNano)
	return fmt.Sprintf("session-%s", ulid.Value)
}

// GenerateSwarmID generates a swarm ID using ULID: "swarm-{ulid}"
func GenerateSwarmID() string {
	ulid := getDefaultULIDGenerator().GenerateWithType(ULIDTypeCustom, ULIDVariantNano)
	return fmt.Sprintf("swarm-%s", ulid.Value)
}

// GenerateTypedULID generates a ULID with the specified type and variant
// This is a typed wrapper around the ULIDGenerator
func GenerateTypedULID(ulidType ULIDType, variant ULIDVariant) *ULID {
	return getDefaultULIDGenerator().GenerateWithType(ulidType, variant)
}

// ParseOrGenerate parses an identity string or generates a new one if parsing fails
func ParseOrGenerate(s string) *Identity {
	id, err := ParseIdentityString(s)
	if err != nil {
		return NewGenerator().Generate()
	}
	if !id.IsValid() {
		return NewGenerator().Generate()
	}
	return id
}

// Pool manages a pool of pre-generated identities for performance
// Now uses ULIDPool internally
type Pool struct {
	hids     chan string
	sids     chan string
	ulidPool *ULIDPool
}

// NewPool creates a new identity pool
func NewPool(size int) *Pool {
	ulidGen := getDefaultULIDGenerator()
	p := &Pool{
		hids:     make(chan string, size),
		sids:     make(chan string, size),
		ulidPool: NewULIDPool(size, ulidGen),
	}
	p.fill(size)
	return p
}

func (p *Pool) fill(count int) {
	gen := NewGenerator()
	for i := 0; i < count; i++ {
		p.hids <- gen.GenerateHID()
		p.sids <- gen.GenerateSID()
	}
}

// GetHID gets an H-id from the pool or generates a new one if empty
func (p *Pool) GetHID() string {
	select {
	case hid := <-p.hids:
		return hid
	default:
		return NewGenerator().GenerateHID()
	}
}

// GetSID gets an S-id from the pool or generates a new one if empty
func (p *Pool) GetSID() string {
	select {
	case sid := <-p.sids:
		return sid
	default:
		return NewGenerator().GenerateSID()
	}
}

// GenerateFromPool generates an Identity using IDs from the pool
func (p *Pool) GenerateFromPool() *Identity {
	return NewIdentity(p.GetHID(), p.GetSID())
}

// GetULID gets a ULID from the internal pool
func (p *Pool) GetULID(ulidType ULIDType) *ULID {
	return p.ulidPool.GetWithType(ulidType)
}
