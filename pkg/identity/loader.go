// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package identity

import (
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultHIDPrefix is the default prefix for auto-generated H-ids
	DefaultHIDPrefix = "user"
	// DefaultSIDPrefix is the default prefix for auto-generated S-ids
	DefaultSIDPrefix = "node"
)

// Environment variables for identity configuration
const (
	EnvHID = "PICOCLAW_IDENTITY_HID"
	EnvSID = "PICOCLAW_IDENTITY_SID"
	EnvIdentity = "PICOCLAW_IDENTITY" // Full identity in "hid/sid" format
)

// Loader handles identity loading from multiple sources
type Loader struct {
	// configHID is the H-id from config file
	configHID string
	// configSID is the S-id from config file
	configSID string
	// cliHID is the H-id from CLI arguments
	cliHID string
	// cliSID is the S-id from CLI arguments
	cliSID string
}

// NewLoader creates a new identity loader
func NewLoader() *Loader {
	return &Loader{}
}

// SetConfig sets the identity from config file
func (l *Loader) SetConfig(hid, sid string) {
	l.configHID = hid
	l.configSID = sid
}

// SetCLI sets the identity from CLI arguments
func (l *Loader) SetCLI(hid, sid string) {
	l.cliHID = hid
	l.cliSID = sid
}

// Load loads the identity from available sources in priority order:
// 1. CLI arguments (highest priority)
// 2. Environment variables
// 3. Config file
// 4. Auto-generation (lowest priority)
func (l *Loader) Load() (*LoadedIdentity, error) {
	// Priority 1: CLI arguments
	if l.cliHID != "" || l.cliSID != "" {
		hid := l.cliHID
		sid := l.cliSID

		// If only one is provided, try to get the other from env
		if hid == "" {
			hid = os.Getenv(EnvHID)
		}
		if sid == "" {
			sid = os.Getenv(EnvSID)
		}

		// Generate missing parts if needed
		if hid == "" {
			hid = generateDefaultHID()
		}
		if sid == "" {
			sid = generateDefaultSID()
		}

		id := NewIdentity(hid, sid)
		if !id.IsValid() {
			return nil, fmt.Errorf("invalid CLI identity: %s", id.String())
		}
		return NewLoadedIdentity(id, SourceCLI), nil
	}

	// Priority 2: Environment variables
	envHID := os.Getenv(EnvHID)
	envSID := os.Getenv(EnvSID)
	envIdentity := os.Getenv(EnvIdentity)

	if envIdentity != "" {
		id, err := NewIdentityFromString(envIdentity)
		if err != nil {
			return nil, fmt.Errorf("invalid environment identity: %w", err)
		}
		if id.IsValid() {
			return NewLoadedIdentity(id, SourceEnv), nil
		}
	}

	if envHID != "" || envSID != "" {
		hid := envHID
		sid := envSID

		// Generate missing parts
		if hid == "" {
			hid = generateDefaultHID()
		}
		if sid == "" {
			sid = generateDefaultSID()
		}

		id := NewIdentity(hid, sid)
		if id.IsValid() {
			return NewLoadedIdentity(id, SourceEnv), nil
		}
	}

	// Priority 3: Config file
	if l.configHID != "" || l.configSID != "" {
		hid := l.configHID
		sid := l.configSID

		// Generate missing parts
		if hid == "" {
			hid = generateDefaultHID()
		}
		if sid == "" {
			sid = generateDefaultSID()
		}

		id := NewIdentity(hid, sid)
		if id.IsValid() {
			return NewLoadedIdentity(id, SourceConfig), nil
		}
	}

	// Priority 4: Auto-generate
	hid := generateDefaultHID()
	sid := generateDefaultSID()
	id := NewIdentity(hid, sid)
	return NewLoadedIdentity(id, SourceAuto), nil
}

// LoadOrGenerate loads the identity or generates one if loading fails
func (l *Loader) LoadOrGenerate() *LoadedIdentity {
	id, err := l.Load()
	if err != nil {
		// Generate on any error
		hid := generateDefaultHID()
		sid := generateDefaultSID()
		return NewLoadedIdentity(NewIdentity(hid, sid), SourceAuto)
	}
	return id
}

// MustLoad loads the identity or panics
func (l *Loader) MustLoad() *LoadedIdentity {
	id, err := l.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load identity: %v", err))
	}
	return id
}

// ParseIdentityString parses an identity from a string
// Supports formats: "hid/sid", "hid", "hid.sid"
func ParseIdentityString(s string) (*Identity, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty identity string")
	}

	// Try "hid/sid" format first
	if strings.Contains(s, "/") {
		return NewIdentityFromString(s)
	}

	// Try "hid.sid" format
	if strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		if len(parts) == 2 {
			return NewIdentity(parts[0], parts[1]), nil
		}
	}

	// Single value - treat as HID, auto-generate SID
	return NewIdentity(s, generateDefaultSID()), nil
}

// LoadFromConfigMap loads identity from a generic config map
func LoadFromConfigMap(cfg map[string]interface{}) (*Identity, error) {
	var hid, sid string

	if v, ok := cfg["hid"].(string); ok {
		hid = v
	}
	if v, ok := cfg["sid"].(string); ok {
		sid = v
	}

	// Check for nested identity config
	if identityCfg, ok := cfg["identity"].(map[string]interface{}); ok {
		if v, ok := identityCfg["hid"].(string); ok {
			hid = v
		}
		if v, ok := identityCfg["sid"].(string); ok {
			sid = v
		}
	}

	// At least HID should be present
	if hid == "" {
		return nil, fmt.Errorf("missing hid in config")
	}

	if sid == "" {
		sid = generateDefaultSID()
	}

	id := NewIdentity(hid, sid)
	if !id.IsValid() {
		return nil, fmt.Errorf("invalid identity from config: %s/%s", hid, sid)
	}

	return id, nil
}

// GetUsername derives a username from the current environment
// Used as a default HID when none is provided
func GetUsername() string {
	// Try common environment variables
	for _, env := range []string{"USER", "USERNAME", "LOGNAME", "LNAME"} {
		if user := os.Getenv(env); user != "" {
			return normalizeID(user)
		}
	}

	return DefaultHIDPrefix
}

// GetHostname derives a hostname from the current environment
// Used as a default SID when none is provided
func GetHostname() string {
	if host, err := os.Hostname(); err == nil {
		// Extract just the hostname part (remove domain)
		host = strings.Split(host, ".")[0]
		return normalizeID(host)
	}

	return DefaultSIDPrefix
}

// generateDefaultHID generates a default H-id
func generateDefaultHID() string {
	return fmt.Sprintf("%s-%s", DefaultHIDPrefix, GetUsername())
}

// generateDefaultSID generates a default S-id
func generateDefaultSID() string {
	return fmt.Sprintf("%s-%s", DefaultSIDPrefix, GetHostname())
}
