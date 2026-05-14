package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Channel type constants — single source of truth for all channel type names.
const (
	ChannelPico           = "pico"
	ChannelPicoClient     = "pico_client"
	ChannelTelegram       = "telegram"
	ChannelDiscord        = "discord"
	ChannelFeishu         = "feishu"
	ChannelWeixin         = "weixin"
	ChannelWeCom          = "wecom"
	ChannelDingTalk       = "dingtalk"
	ChannelSlack          = "slack"
	ChannelMatrix         = "matrix"
	ChannelLINE           = "line"
	ChannelOneBot         = "onebot"
	ChannelQQ             = "qq"
	ChannelIRC            = "irc"
	ChannelVK             = "vk"
	ChannelMaixCam        = "maixcam"
	ChannelWhatsApp       = "whatsapp"
	ChannelWhatsAppNative = "whatsapp_native"
	ChannelTeamsWebHook   = "teams_webhook"
	ChannelMQTT           = "mqtt"
	ChannelSlackWebHook   = "slack_webhook"
)

func initChannel() {
	registerSingletonChannel(ChannelPico)
	registerSingletonChannel(ChannelPicoClient)
}

// singletonRegistry stores which channel types are singletons (only allow one instance).
// Each channel type should call registerSingletonChannel in its init() if it's a singleton.
var singletonRegistry = make(map[string]struct{})

// registerSingletonChannel marks a channel type as singleton (only one instance allowed).
// Should be called from the channel type's init() function.
func registerSingletonChannel(channelType string) {
	singletonRegistry[channelType] = struct{}{}
}

// IsSingletonChannel returns true if the channel type only allows one instance.
func IsSingletonChannel(channelType string) bool {
	_, ok := singletonRegistry[channelType]
	return ok
}

// RawNode stores raw configuration data as JSON bytes, supporting both JSON and YAML.
// Internally uses json.RawMessage, so Decode always uses json.Unmarshal
// which correctly respects json struct tags.
type RawNode json.RawMessage

// UnmarshalJSON implements json.Unmarshaler: stores raw JSON bytes.
// NOTE: yaml.Unmarshal may call this when unmarshaling into RawNode fields.
// We detect if the input looks like YAML (not JSON) and handle it.
func (r *RawNode) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "{}" || trimmed == "[]" {
		*r = nil
		return nil
	}

	// If it doesn't look like JSON (starts with {, [, ", digit, n, t, f),
	// it's probably YAML data passed through yaml.Unmarshal.
	// Try to parse as YAML and convert to JSON.
	if len(trimmed) > 0 {
		first := trimmed[0]
		if first != '{' && first != '[' && first != '"' && first != '-' &&
			!(first >= '0' && first <= '9') && first != 'n' && first != 't' && first != 'f' {
			// Looks like YAML, not JSON. Parse as YAML and convert to JSON.
			var v any
			if err := yaml.Unmarshal(data, &v); err != nil {
				return err
			}
			jsonData, err := json.Marshal(v)
			if err != nil {
				return err
			}
			*r = jsonData
			return nil
		}
	}

	*r = append((*r)[:0:0], data...)
	return nil
}

// MarshalJSON implements json.Marshaler: outputs stored JSON bytes.
func (r RawNode) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

// UnmarshalYAML implements yaml.Unmarshaler: converts YAML node to JSON bytes.
// Merges the incoming YAML values with existing data, with YAML taking precedence.
func (r *RawNode) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == 0 {
		//*r = nil
		return nil
	}
	var v1, v2 map[string]any
	if len(*r) > 0 {
		if err := json.Unmarshal(*r, &v1); err != nil {
			return err
		}
	}
	if err := value.Decode(&v2); err != nil {
		return err
	}
	v := mergeMap(v1, v2)
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*r = data
	return nil
}

// mergeMap deeply merges two map[string]any.
// dst: base map
// src: override map (same keys overwrite dst, nested maps are merged recursively)
// Returns a new map without modifying the originals.
func mergeMap(dst, src map[string]any) map[string]any {
	// logger.Infof("mergeMap: dst: %v, src: %v", dst, src)
	// Create result map to avoid modifying originals
	result := make(map[string]any)

	// Copy all content from base map
	for k, v := range dst {
		result[k] = v
	}

	// Merge override map
	for k, srcVal := range src {
		dstVal, exists := result[k]

		if !exists {
			// Key doesn't exist in base, add directly
			result[k] = srcVal
			continue
		}

		// Both are maps → recursive merge
		dstMap, dstIsMap := toMap(dstVal)
		srcMap, srcIsMap := toMap(srcVal)

		if dstIsMap && srcIsMap {
			result[k] = mergeMap(dstMap, srcMap)
		} else {
			// Not both maps → override
			result[k] = srcVal
		}
	}

	return result
}

// toMap safely converts any value to map[string]any.
func toMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

// MarshalYAML implements yaml.ValueMarshaler: converts stored JSON back to a YAML-compatible value.
func (r RawNode) MarshalYAML() (any, error) {
	if len(r) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(r, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// Decode unmarshals the stored data into the given target struct using json.Unmarshal.
func (r *RawNode) Decode(target any) error {
	if len(*r) == 0 {
		return nil
	}
	return json.Unmarshal(*r, target)
}

// IsEmpty returns true if the node has not been populated.
func (r *RawNode) IsEmpty() bool {
	return len(*r) == 0
}

// Channel defines the common fields shared by all channel types.
// Channel-specific settings go into Settings (nested format only).
// The settings struct should use SecureString/SecureStrings for sensitive fields.
//
// Decode stores the settings pointer internally; subsequent modifications to the
// decoded struct are automatically reflected in MarshalJSON/MarshalYAML.
//
// MarshalJSON outputs nested format (common fields at top level, settings as sub-key).
// MarshalYAML outputs only secure fields (for .security.yml).
//
// Standard Go JSON/YAML unmarshaling handles nested format correctly:
//   - JSON: {"enabled": true, "type": "telegram", "settings": {"base_url": "..."}}
//   - YAML: settings: {token: xxx}  (for .security.yml)
//
//nolint:recvcheck
type Channel struct {
	name               string
	Enabled            bool                `json:"enabled"                 yaml:"-"`
	Type               string              `json:"type"                    yaml:"-"`
	AllowFrom          FlexibleStringSlice `json:"allow_from,omitempty"    yaml:"-"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    yaml:"-"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty" yaml:"-"`
	Typing             TypingConfig        `json:"typing,omitempty"        yaml:"-"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"   yaml:"-"`
	Settings           RawNode             `json:"settings,omitzero"       yaml:"settings,omitempty"`
	extend             any
}

// MarshalJSON implements json.Marshaler for Channel.
// Outputs nested format: common fields at top level, channel-specific in "settings".
// Secure fields (SecureString/SecureStrings) are removed from settings output.
func (b Channel) MarshalJSON() ([]byte, error) {
	var settings RawNode
	if b.extend != nil {
		raw, err := json.Marshal(b.extend)
		if err != nil {
			return nil, err
		}
		settings = raw
	} else {
		settings = b.Settings
	}

	out := b
	out.Settings = settings

	// Use type alias to bypass our custom MarshalJSON (infinite recursion)
	type Alias Channel
	return json.Marshal((*Alias)(&out))
}

// MarshalYAML implements yaml.ValueMarshaler for Channel.
// Outputs only secure fields in the Settings YAML (for .security.yml).
// If Decode was called, it serializes from the stored extend (reflecting any
// modifications); otherwise falls back to decoding Settings via the channel Type
// to extract secure fields.
func (b Channel) MarshalYAML() (any, error) {
	decoded, _ := b.GetDecoded()
	return struct {
		Settings any `json:"settings,omitzero" yaml:"settings,omitempty"`
	}{
		Settings: decoded,
	}, nil
}

// Name returns the channel name.
func (b *Channel) Name() string {
	return b.name
}

// SetName sets the channel name.
func (b *Channel) SetName(name string) {
	b.name = name
}

// SetSecretField sets a secure field value by field name in the Settings JSON.
// NOTE: This only operates on raw Settings. If Decode() has been called,
// prefer modifying the typed struct directly — MarshalJSON serializes from extend.
func (b *Channel) SetSecretField(fieldName string, value SecureString) {
	var m map[string]any
	if err := json.Unmarshal(b.Settings, &m); err != nil {
		return
	}
	m[fieldName] = value
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	b.Settings = data
}

// Decode decodes the Settings node into the given target struct and stores
// the pointer internally. Subsequent modifications to the target are
// automatically reflected in MarshalJSON/MarshalYAML (no explicit Encode needed).
func (b *Channel) Decode(target any) error {
	if target == nil {
		return fmt.Errorf("target is nil")
	}
	if err := b.Settings.Decode(target); err != nil {
		return err
	}
	b.extend = target
	return nil
}

// GetDecoded returns the previously decoded settings struct.
// If Decode hasn't been called yet, it lazily decodes using the channel Type prototype.
// Returns an error if decoding fails; the decoded value (possibly nil) is still returned
// so callers can distinguish between "not decoded" and "decode failed".
func (b *Channel) GetDecoded() (any, error) {
	if b.extend == nil {
		// fallback to prototype-based creation
		if target := newChannelSettings(b.Type); target != nil {
			if err := b.Decode(target); err != nil {
				return nil, fmt.Errorf("channel %q failed to decode settings: %w", b.name, err)
			}
		}
	}
	return b.extend, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for Channel.
// Merges the YAML node into the existing Channel.
// Supports both nested format (settings: {...}) and flat format (token: xxx).
func (b *Channel) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == 0 {
		return nil
	}

	type alias Channel
	a := alias(*b)
	err := value.Decode(&a)
	if err != nil {
		logger.Errorf("decode yaml error: %v", err)
		return err
	}

	*b = *(*Channel)(&a)

	if len(b.Settings) > 0 {
		b.extend = nil
	}

	return nil
}

// SettingsIsEmpty returns true if Settings has not been populated.
func (b *Channel) SettingsIsEmpty() bool {
	return b.Settings.IsEmpty()
}

// CollectSensitiveValues returns all sensitive string values from this Channel's
// decoded settings (extend). Used by the security filter system.
func (b Channel) CollectSensitiveValues() []string {
	if b.extend == nil {
		return nil
	}
	var values []string
	collectSensitive(reflect.ValueOf(b.extend), &values)
	return values
}

// ChannelsConfig maps channel name to its Channel configuration.
// Each Channel stores the full channel config in Settings and handles
// JSON/YAML serialization (removing/keeping secure fields automatically).
//
//nolint:recvcheck
type ChannelsConfig map[string]*Channel

// UnmarshalYAML implements yaml.Unmarshaler for ChannelsConfig.
// This ensures that when loading security.yml, existing Channel instances
// are properly merged rather than replaced with new ones.
func (c *ChannelsConfig) UnmarshalYAML(value *yaml.Node) error {
	// yaml.Node Content for a mapping contains alternating key-value nodes
	// We need to iterate through them in pairs
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node, got %v", value.Kind)
	}

	if *c == nil {
		*c = make(ChannelsConfig)
	}

	for i := 0; i < len(value.Content); i += 2 {
		if i+1 >= len(value.Content) {
			break
		}
		name := value.Content[i].Value
		node := value.Content[i+1]

		existingBC := (*c)[name]
		if existingBC != nil {
			// Channel already exists - call UnmarshalYAML on it
			// This merges security.yml settings into existing config
			if err := existingBC.UnmarshalYAML(node); err != nil {
				return err
			}
			// Ensure name is set (may have been empty before)
			existingBC.SetName(name)
		} else {
			// New channel - create and unmarshal
			newBC := &Channel{}
			if err := node.Decode(newBC); err != nil {
				return err
			}
			// Set the channel name from the map key
			newBC.SetName(name)
			(*c)[name] = newBC
		}
	}

	return nil
}

// UnmarshalJSON implements json.Unmarshaler for ChannelsConfig.
// Sets the channel name from the map key after unmarshaling.
func (c *ChannelsConfig) UnmarshalJSON(data []byte) error {
	// Use a type alias to avoid infinite recursion
	type channelsConfigAlias map[string]*Channel
	var raw channelsConfigAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if *c == nil {
		*c = make(ChannelsConfig)
	}

	for name, bc := range raw {
		if bc != nil {
			bc.SetName(name)
		}
		(*c)[name] = bc
	}

	return nil
}

// Get returns the Channel for the given channel name (map key), or nil if not found.
func (c ChannelsConfig) Get(name string) *Channel {
	if c == nil {
		return nil
	}
	return c[name]
}

// GetByType returns the Channel for the given channel type, or nil if not found.
func (c ChannelsConfig) GetByType(t string) *Channel {
	if c == nil {
		return nil
	}
	for _, bc := range c {
		if bc.Type == t {
			return bc
		}
	}
	return nil
}

// SetEnabled sets the Enabled field on the Channel with the given name.
// Returns false if no channel with that name exists.
func (c ChannelsConfig) SetEnabled(name string, enabled bool) bool {
	bc := c[name]
	if bc == nil {
		return false
	}
	bc.Enabled = enabled
	return true
}

// validateSingletonChannels checks that singleton channel types have at most
// one enabled instance. Returns an error if a singleton type has multiple enabled channels.
func validateSingletonChannels(channels ChannelsConfig) error {
	typeCount := make(map[string]int)
	typeNames := make(map[string][]string)
	for name, bc := range channels {
		if !bc.Enabled {
			continue
		}
		t := bc.Type
		if t == "" {
			t = name
		}
		if IsSingletonChannel(t) {
			typeCount[t]++
			typeNames[t] = append(typeNames[t], name)
		}
	}
	for t, count := range typeCount {
		if count > 1 {
			return fmt.Errorf(
				"channel type %q is singleton and does not support multiple instances, found %d enabled instances: %v",
				t,
				count,
				typeNames[t],
			)
		}
	}
	return nil
}

// BaseFieldNames are JSON keys that belong to Channel, not to channel-specific settings.
var BaseFieldNames = map[string]struct{}{
	"enabled":              {},
	"type":                 {},
	"allow_from":           {},
	"reasoning_channel_id": {},
	"group_trigger":        {},
	"typing":               {},
	"placeholder":          {},
}

// ─── Internal helpers ───

// extractSecureFieldNames uses reflection to find exported fields of type
// SecureString or SecureStrings and returns their JSON field names.
func extractSecureFieldNames(target any) map[string]struct{} {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	names := make(map[string]struct{})
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		ft := f.Type
		if ft == reflect.TypeOf(SecureString{}) || ft == reflect.TypeOf(&SecureString{}) ||
			ft == reflect.TypeOf(SecureStrings{}) || ft == reflect.TypeOf(&SecureStrings{}) {
			jsonTag := f.Tag.Get("json")
			name := strings.Split(jsonTag, ",")[0]
			if name == "" || name == "-" {
				name = f.Name
			}
			names[name] = struct{}{}
		}
	}
	return names
}

// mergeRawJSON merges two JSON objects (flat key-value) at the raw byte level.
// Overlay values override base values.
func mergeRawJSON(base, overlay RawNode) (RawNode, error) {
	var baseMap, overlayMap map[string]any
	if len(base) > 0 {
		if err := json.Unmarshal(base, &baseMap); err != nil {
			return base, err
		}
	}
	if len(overlay) > 0 {
		if err := json.Unmarshal(overlay, &overlayMap); err != nil {
			return base, err
		}
	}
	if baseMap == nil {
		baseMap = make(map[string]any)
	}
	for k, v := range overlayMap {
		baseMap[k] = v
	}
	data, err := json.Marshal(baseMap)
	if err != nil {
		return base, err
	}
	return RawNode(data), nil
}

// removeSecureFields removes secure fields from the raw JSON.
// If secureFields is nil or empty, returns the raw node as-is.
func removeSecureFields(r RawNode, secureFields map[string]struct{}) RawNode {
	if len(r) == 0 || len(secureFields) == 0 {
		return r
	}
	var m map[string]any
	if err := json.Unmarshal(r, &m); err != nil {
		return r
	}
	for name := range secureFields {
		delete(m, name)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return r
	}
	return RawNode(data)
}

// filterSecureFields keeps only secure fields in the raw JSON.
// If secureFields is nil or empty, returns nil (so omitzero/omitempty can omit it).
func filterSecureFields(r RawNode, secureFields map[string]struct{}) RawNode {
	if len(r) == 0 || len(secureFields) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(r, &m); err != nil {
		return nil
	}
	secureMap := make(map[string]any)
	for name := range secureFields {
		if val, ok := m[name]; ok {
			secureMap[name] = val
		}
	}
	if len(secureMap) == 0 {
		return nil
	}
	data, err := json.Marshal(secureMap)
	if err != nil {
		return nil
	}
	return data
}

// channelSettingsFactory maps channel type to a zero-value prototype of the
// corresponding Settings struct. InitChannelList uses reflect.New to create
// fresh instances, avoiding repeated closure boilerplate.
var channelSettingsFactory = map[string]any{
	ChannelPico:           (PicoSettings{}),
	ChannelPicoClient:     (PicoClientSettings{}),
	ChannelTelegram:       (TelegramSettings{}),
	ChannelDiscord:        (DiscordSettings{}),
	ChannelFeishu:         (FeishuSettings{}),
	ChannelWeixin:         (WeixinSettings{}),
	ChannelWeCom:          (WeComSettings{}),
	ChannelDingTalk:       (DingTalkSettings{}),
	ChannelSlack:          (SlackSettings{}),
	ChannelMatrix:         (MatrixSettings{}),
	ChannelLINE:           (LINESettings{}),
	ChannelOneBot:         (OneBotSettings{}),
	ChannelQQ:             (QQSettings{}),
	ChannelIRC:            (IRCSettings{}),
	ChannelVK:             (VKSettings{}),
	ChannelMaixCam:        (MaixCamSettings{}),
	ChannelWhatsApp:       (WhatsAppSettings{}),
	ChannelWhatsAppNative: (WhatsAppSettings{}),
	ChannelTeamsWebHook:   (TeamsWebhookSettings{}),
	ChannelMQTT:           (MQTTSettings{}),
	ChannelSlackWebHook:   (SlackWebhookSettings{}),
}

// newChannelSettings creates a fresh zero-value pointer for the given channel type.
// Returns nil if the type is not registered.
func newChannelSettings(channelType string) any {
	proto, ok := channelSettingsFactory[channelType]
	if !ok {
		return nil
	}
	return reflect.New(reflect.TypeOf(proto)).Interface()
}

// isValidChannelType returns true if the channel type is a known, registered type.
func isValidChannelType(channelType string) bool {
	_, ok := channelSettingsFactory[channelType]
	return ok
}

// InitChannelList validates and initializes all channels in the ChannelsConfig.
// It performs three steps:
//  1. Validates that each channel has a non-empty Type
//  2. Validates singleton constraints
//  3. Decodes Settings into the correct typed struct based on Type,
//     so that b.extend contains the actual settings (e.g., PicoSettings)
//
// After calling this method, callers can safely use b.extend via Decode()
// without re-parsing raw Settings.
func InitChannelList(channels ChannelsConfig) error {
	// Step 1 & 3: validate type and decode into typed settings
	for name, bc := range channels {
		if bc == nil {
			delete(channels, name)
			continue
		}
		// Ensure channel name is set from the map key
		bc.SetName(name)
		// Infer Type from map key if not explicitly set
		if bc.Type == "" {
			bc.Type = name
		}
		if !isValidChannelType(bc.Type) {
			return fmt.Errorf("channel %q has unknown type %q", name, bc.Type)
		}
		// Decode into the correct typed settings
		if target := newChannelSettings(bc.Type); target != nil {
			if err := bc.Decode(target); err != nil {
				return fmt.Errorf("channel %q failed to decode settings: %w", name, err)
			}
			// Apply env overrides for channel-specific fields via struct tags
			if err := env.Parse(target); err != nil {
				// Non-fatal: some env vars may not apply
			}
		}
	}

	// Step 2: validate singleton constraints
	if err := validateSingletonChannels(channels); err != nil {
		return err
	}

	return nil
}
