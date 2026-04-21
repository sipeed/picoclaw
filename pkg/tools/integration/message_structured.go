package integrationtools

import "fmt"

// StructuredPart is implemented by every canonical and alias structured part type.
// Parse validates and ingests raw LLM input; ToMap serializes back to the wire format.
// This mirrors VS Code's ChatResponsePart design: each kind owns its own schema.
type StructuredPart interface {
	Parse(entry map[string]any, fieldPath string) error
	ToMap() map[string]any
}

// structuredPartFactory creates a fresh zero-value StructuredPart for a given type name.
// Using a factory (rather than storing instances) keeps each normalisation call isolated.
type structuredPartFactory func() StructuredPart

// structuredPartRegistry maps each accepted type name to its factory.
// Canonical types (options, card) produce their own parts.
// Alias input types (form, progress, todo, alert) produce alias parts that
// canonicalise to type:card + kind on ToMap.
var structuredPartRegistry = map[string]structuredPartFactory{
	// Canonical types
	"options": func() StructuredPart { return &StructuredOptionsPart{} },
	"card":    func() StructuredPart { return &StructuredCardPart{} },
	// Alias input types — accepted for backward compatibility, normalised to card+kind
	"form":     func() StructuredPart { return &StructuredFormPart{} },
	"progress": func() StructuredPart { return &StructuredProgressPart{} },
	"todo":     func() StructuredPart { return &StructuredTodoPart{} },
	"alert":    func() StructuredPart { return &StructuredAlertPart{} },
}

func normalizeStructuredEntry(entry map[string]any, fieldPath string) (map[string]any, error) {
	msgType, _ := entry["type"].(string)
	if msgType == "" {
		return nil, fmt.Errorf("%s.type is required", fieldPath)
	}

	if factory, ok := structuredPartRegistry[msgType]; ok {
		part := factory()
		if err := part.Parse(cloneStructuredEntry(entry), fieldPath); err != nil {
			return nil, err
		}
		return part.ToMap(), nil
	}

	// Unknown types are passed through to the frontend for custom rendering.
	// The frontend will render them via a registered custom renderer or fall back
	// to an "unknown" panel that displays the raw JSON.
	return cloneStructuredEntry(entry), nil
}

// OptionItem is the canonical wire representation of a single selectable option.
// Mirrors VS Code ChatResponseQuestionCarouselPart's item shape — each item owns
// its label and a stable value that the reply handler receives.
type OptionItem struct {
	Label       string
	Value       string
	Description string // optional
}

func (o OptionItem) toMap() map[string]any {
	m := map[string]any{
		"label": o.Label,
		"value": o.Value,
	}
	if o.Description != "" {
		m["description"] = o.Description
	}
	return m
}

// StructuredOptionsPart is the canonical type for an options part.
// Mirrors VS Code ChatResponseMarkdownPart / ChatResponseConfirmationPart — the struct
// is the schema; Parse is the only place input validation lives.
type StructuredOptionsPart struct {
	Options           []OptionItem
	Mode              string // "single" | "multiple"; defaults to "single"
	AllowCustom       bool
	CustomPlaceholder string
	SubmitLabel       string
}

func (p *StructuredOptionsPart) Parse(entry map[string]any, fieldPath string) error {
	rawOptions, ok := entry["options"]
	if !ok || rawOptions == nil {
		return fmt.Errorf("%s.options must be an array when %s.type='options'", fieldPath, fieldPath)
	}
	items, ok := rawOptions.([]any)
	if !ok {
		return fmt.Errorf("%s.options must be an array when %s.type='options'", fieldPath, fieldPath)
	}
	if len(items) == 0 {
		return fmt.Errorf("%s.options must not be empty", fieldPath)
	}

	options := make([]OptionItem, 0, len(items))
	for i, item := range items {
		raw, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("%s.options[%d] must be an object", fieldPath, i)
		}
		label, _ := raw["label"].(string)
		value, _ := raw["value"].(string)
		if label == "" || value == "" {
			return fmt.Errorf("%s.options[%d].label and .value are required", fieldPath, i)
		}
		description, _ := raw["description"].(string)
		options = append(options, OptionItem{Label: label, Value: value, Description: description})
	}

	mode, _ := entry["mode"].(string)
	if mode == "" {
		mode = "single"
	}

	p.Options = options
	p.Mode = mode
	p.AllowCustom, _ = entry["allowCustom"].(bool)
	p.CustomPlaceholder, _ = entry["customPlaceholder"].(string)
	p.SubmitLabel, _ = entry["submitLabel"].(string)
	return nil
}

// ToMap serializes the canonical part back to a wire map sent to the frontend.
func (p *StructuredOptionsPart) ToMap() map[string]any {
	items := make([]any, len(p.Options))
	for i, opt := range p.Options {
		items[i] = opt.toMap()
	}
	m := map[string]any{
		"type":    "options",
		"options": items,
		"mode":    p.Mode,
	}
	if p.AllowCustom {
		m["allowCustom"] = true
	}
	if p.CustomPlaceholder != "" {
		m["customPlaceholder"] = p.CustomPlaceholder
	}
	if p.SubmitLabel != "" {
		m["submitLabel"] = p.SubmitLabel
	}
	return m
}

// StructuredCardPart is the canonical type for a rich card.
type StructuredCardPart struct {
	raw map[string]any // retains all fields after validation
}

func (p *StructuredCardPart) Parse(entry map[string]any, fieldPath string) error {
	title, _ := entry["title"].(string)
	kind, _ := entry["kind"].(string)

	blocks, hasBlocks, err := normalizeCardBlocks(entry["blocks"], fieldPath)
	if err != nil {
		return err
	}
	actions, hasActions, err := normalizeActionItems(entry["actions"], fieldPath+".actions", "card")
	if err != nil {
		return err
	}

	if title == "" && kind == "" {
		return fmt.Errorf("%s.title is required when %s.type='card' unless %s.kind identifies a custom card", fieldPath, fieldPath, fieldPath)
	}
	if !hasBlocks && !hasActions {
		return fmt.Errorf("%s.blocks or %s.actions must be a non-empty array when %s.type='card'", fieldPath, fieldPath, fieldPath)
	}
	if hasBlocks {
		entry["blocks"] = blocks
	}
	if hasActions {
		entry["actions"] = actions
	}
	p.raw = entry
	return nil
}

func (p *StructuredCardPart) ToMap() map[string]any {
	return cloneStructuredEntry(p.raw)
}

// StructuredFormPart is an alias input type that normalises to type:card + kind:form.
type StructuredFormPart struct {
	raw map[string]any
}

func (p *StructuredFormPart) Parse(entry map[string]any, fieldPath string) error {
	assignFirstStringAlias(entry, "content", "content", "description", "message")
	title, _ := entry["title"].(string)
	content, _ := entry["content"].(string)
	if title == "" && content == "" {
		return fmt.Errorf("%s.title or %s.content is required when %s.type='form'", fieldPath, fieldPath, fieldPath)
	}

	fields, err := normalizeFormFields(entry["fields"], fieldPath)
	if err != nil {
		return err
	}
	entry["fields"] = fields

	actions, hasActions, err := normalizeActionItems(entry["actions"], fieldPath+".actions", "form")
	if err != nil {
		return err
	}
	if hasActions {
		entry["actions"] = actions
	}
	p.raw = canonicalizeStructuredAliasEntry(entry, "form")
	return nil
}

func (p *StructuredFormPart) ToMap() map[string]any { return cloneStructuredEntry(p.raw) }

// StructuredProgressPart is an alias input type that normalises to type:card + kind:progress.
type StructuredProgressPart struct {
	raw map[string]any
}

func (p *StructuredProgressPart) Parse(entry map[string]any, fieldPath string) error {
	assignFirstStringAlias(entry, "content", "content", "description", "message", "detail")
	status, _ := entry["status"].(string)
	if status == "" {
		return fmt.Errorf("%s.status is required when %s.type='progress'", fieldPath, fieldPath)
	}

	title, _ := entry["title"].(string)
	content, _ := entry["content"].(string)
	steps, hasSteps, err := normalizeProgressSteps(entry["steps"], fieldPath)
	if err != nil {
		return err
	}
	if title == "" && content == "" && !hasSteps {
		return fmt.Errorf("%s.title, %s.content, or %s.steps must be provided when %s.type='progress'", fieldPath, fieldPath, fieldPath, fieldPath)
	}
	if hasSteps {
		entry["steps"] = steps
	}
	p.raw = canonicalizeStructuredAliasEntry(entry, "progress")
	return nil
}

func (p *StructuredProgressPart) ToMap() map[string]any { return cloneStructuredEntry(p.raw) }

// StructuredTodoPart is an alias input type that normalises to type:card + kind:todo.
type StructuredTodoPart struct {
	raw map[string]any
}

func (p *StructuredTodoPart) Parse(entry map[string]any, fieldPath string) error {
	assignFirstStringAlias(entry, "content", "content", "description", "message")
	items, err := normalizeTodoItems(entry["items"], fieldPath)
	if err != nil {
		return err
	}
	entry["items"] = items
	p.raw = canonicalizeStructuredAliasEntry(entry, "todo")
	return nil
}

func (p *StructuredTodoPart) ToMap() map[string]any { return cloneStructuredEntry(p.raw) }

// StructuredAlertPart is an alias input type that normalises to type:card + kind:alert.
type StructuredAlertPart struct {
	raw map[string]any
}

func (p *StructuredAlertPart) Parse(entry map[string]any, fieldPath string) error {
	assignFirstStringAlias(entry, "level", "level", "severity", "statusLevel")
	assignFirstStringAlias(entry, "content", "content", "description", "message", "detail")

	level, _ := entry["level"].(string)
	if level == "" {
		return fmt.Errorf("%s.level is required when %s.type='alert'", fieldPath, fieldPath)
	}
	content, _ := entry["content"].(string)
	if content == "" {
		return fmt.Errorf("%s.content is required when %s.type='alert'", fieldPath, fieldPath)
	}

	actions, hasActions, err := normalizeActionItems(entry["actions"], fieldPath+".actions", "alert")
	if err != nil {
		return err
	}
	if hasActions {
		entry["actions"] = actions
	}
	p.raw = canonicalizeStructuredAliasEntry(entry, "alert")
	return nil
}

func (p *StructuredAlertPart) ToMap() map[string]any { return cloneStructuredEntry(p.raw) }

func canonicalizeStructuredAliasEntry(entry map[string]any, aliasType string) map[string]any {
	canonical := cloneStructuredEntry(entry)
	canonical["type"] = "card"
	if kind, _ := canonical["kind"].(string); kind == "" {
		canonical["kind"] = aliasType
	}
	return canonical
}

func cloneStructuredEntry(entry map[string]any) map[string]any {
	cloned := make(map[string]any, len(entry))
	for key, value := range entry {
		cloned[key] = value
	}
	return cloned
}

func assignFirstStringAlias(entry map[string]any, target string, aliases ...string) {
	for _, alias := range aliases {
		value, ok := entry[alias].(string)
		if ok && value != "" {
			entry[target] = value
			return
		}
	}
}

func normalizeActionItems(raw any, fieldPath string, parentType string) ([]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, false, fmt.Errorf("%s must be an array when structured.type='%s'", fieldPath, parentType)
	}
	if len(items) == 0 {
		return nil, false, fmt.Errorf("%s must be a non-empty array when structured.type='%s'", fieldPath, parentType)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("%s[%d] must be an object when structured.type='%s'", fieldPath, index, parentType)
		}
		label, _ := entry["label"].(string)
		if label == "" {
			return nil, false, fmt.Errorf("%s[%d].label is required when structured.type='%s'", fieldPath, index, parentType)
		}
		normalized = append(normalized, cloneStructuredEntry(entry))
	}

	return normalized, true, nil
}

func normalizeCardBlocks(raw any, fieldPath string) ([]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, false, fmt.Errorf("%s.blocks must be an array when %s.type='card'", fieldPath, fieldPath)
	}
	if len(items) == 0 {
		return nil, false, fmt.Errorf("%s.blocks must be a non-empty array when %s.type='card'", fieldPath, fieldPath)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("%s.blocks[%d] must be an object when %s.type='card'", fieldPath, index, fieldPath)
		}
		block, err := normalizeCardBlock(entry, fmt.Sprintf("%s.blocks[%d]", fieldPath, index))
		if err != nil {
			return nil, false, err
		}
		normalized = append(normalized, block)
	}

	return normalized, true, nil
}

// CardBlock is implemented by every card block type.
// Parse validates and ingests raw LLM input; ToMap serializes back to the wire format.
