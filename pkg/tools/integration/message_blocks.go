package integrationtools

import "fmt"

// CardBlock is implemented by every card block type.
// Parse validates and ingests raw LLM input; ToMap serializes back to the wire format.
type CardBlock interface {
	Parse(entry map[string]any, fieldPath string) error
	ToMap() map[string]any
}

type cardBlockFactory func() CardBlock

// cardBlockRegistry maps each known block type to its factory.
// "markdown" is an alias for "text". Custom block types are passed through by the dispatcher.
var cardBlockRegistry = map[string]cardBlockFactory{
	"text":     func() CardBlock { return &CardTextBlock{} },
	"markdown": func() CardBlock { return &CardTextBlock{} },
	"fields":   func() CardBlock { return &CardFieldsBlock{} },
	"badge":    func() CardBlock { return &CardBadgeBlock{} },
	"actions":  func() CardBlock { return &CardActionsBlock{} },
	"list":     func() CardBlock { return &CardListBlock{} },
	"table":    func() CardBlock { return &CardTableBlock{} },
	"image":    func() CardBlock { return &CardImageBlock{} },
	"divider":  func() CardBlock { return &CardDividerBlock{} },
	"json":     func() CardBlock { return &CardJSONBlock{} },
}

func normalizeCardBlock(entry map[string]any, fieldPath string) (map[string]any, error) {
	blockType, _ := entry["type"].(string)
	if blockType == "" {
		return nil, fmt.Errorf("%s.type is required when structured.type='card'", fieldPath)
	}
	if factory, ok := cardBlockRegistry[blockType]; ok {
		block := factory()
		if err := block.Parse(cloneStructuredEntry(entry), fieldPath); err != nil {
			return nil, err
		}
		return block.ToMap(), nil
	}
	// Custom block types are passed through as long as they declare a type.
	return cloneStructuredEntry(entry), nil
}

// CardTextBlock handles "text" and "markdown" block types.
type CardTextBlock struct{ raw map[string]any }

func (b *CardTextBlock) Parse(entry map[string]any, fieldPath string) error {
	blockType, _ := entry["type"].(string)
	text, _ := entry["text"].(string)
	if text == "" {
		return fmt.Errorf("%s.text is required for card block type '%s'", fieldPath, blockType)
	}
	b.raw = entry
	return nil
}
func (b *CardTextBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardFieldsBlock handles the "fields" block type.
type CardFieldsBlock struct{ raw map[string]any }

func (b *CardFieldsBlock) Parse(entry map[string]any, fieldPath string) error {
	fields, err := normalizeCardFieldItems(entry["fields"], fieldPath)
	if err != nil {
		return err
	}
	entry["fields"] = fields
	b.raw = entry
	return nil
}
func (b *CardFieldsBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardBadgeBlock handles the "badge" block type.
type CardBadgeBlock struct{ raw map[string]any }

func (b *CardBadgeBlock) Parse(entry map[string]any, fieldPath string) error {
	label, _ := entry["label"].(string)
	if label == "" {
		return fmt.Errorf("%s.label is required for card block type 'badge'", fieldPath)
	}
	b.raw = entry
	return nil
}
func (b *CardBadgeBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardActionsBlock handles the "actions" block type.
type CardActionsBlock struct{ raw map[string]any }

func (b *CardActionsBlock) Parse(entry map[string]any, fieldPath string) error {
	actions, _, err := normalizeActionItems(entry["actions"], fieldPath+".actions", "card")
	if err != nil {
		return err
	}
	entry["actions"] = actions
	b.raw = entry
	return nil
}
func (b *CardActionsBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardListBlock handles the "list" block type.
type CardListBlock struct{ raw map[string]any }

func (b *CardListBlock) Parse(entry map[string]any, fieldPath string) error {
	items, err := normalizeCardListItems(entry["items"], fieldPath)
	if err != nil {
		return err
	}
	entry["items"] = items
	b.raw = entry
	return nil
}
func (b *CardListBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardTableBlock handles the "table" block type.
type CardTableBlock struct{ raw map[string]any }

func (b *CardTableBlock) Parse(entry map[string]any, fieldPath string) error {
	rows, err := normalizeTableRows(entry["rows"], fieldPath)
	if err != nil {
		return err
	}
	entry["rows"] = rows
	b.raw = entry
	return nil
}
func (b *CardTableBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardImageBlock handles the "image" block type.
type CardImageBlock struct{ raw map[string]any }

func (b *CardImageBlock) Parse(entry map[string]any, fieldPath string) error {
	url, _ := entry["url"].(string)
	if url == "" {
		return fmt.Errorf("%s.url is required for card block type 'image'", fieldPath)
	}
	b.raw = entry
	return nil
}
func (b *CardImageBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardDividerBlock handles the "divider" block type (no required fields).
type CardDividerBlock struct{ raw map[string]any }

func (b *CardDividerBlock) Parse(entry map[string]any, fieldPath string) error {
	b.raw = entry
	return nil
}
func (b *CardDividerBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

// CardJSONBlock handles the "json" block type.
type CardJSONBlock struct{ raw map[string]any }

func (b *CardJSONBlock) Parse(entry map[string]any, fieldPath string) error {
	if _, ok := entry["data"]; !ok {
		return fmt.Errorf("%s.data is required for card block type 'json'", fieldPath)
	}
	b.raw = entry
	return nil
}
func (b *CardJSONBlock) ToMap() map[string]any { return cloneStructuredEntry(b.raw) }

func normalizeCardFieldItems(raw any, fieldPath string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.fields must be an array for card block type 'fields'", fieldPath)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s.fields must be a non-empty array for card block type 'fields'", fieldPath)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s.fields[%d] must be an object for card block type 'fields'", fieldPath, index)
		}
		label, _ := entry["label"].(string)
		value, _ := entry["value"].(string)
		if label == "" || value == "" {
			return nil, fmt.Errorf("%s.fields[%d].label and %s.fields[%d].value are required for card block type 'fields'", fieldPath, index, fieldPath, index)
		}
		normalized = append(normalized, cloneStructuredEntry(entry))
	}
	return normalized, nil
}

func normalizeCardListItems(raw any, fieldPath string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.items must be an array for card block type 'list'", fieldPath)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s.items must be a non-empty array for card block type 'list'", fieldPath)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		switch typed := item.(type) {
		case string:
			if typed == "" {
				return nil, fmt.Errorf("%s.items[%d] must not be empty for card block type 'list'", fieldPath, index)
			}
			normalized = append(normalized, typed)
		case map[string]any:
			text, _ := typed["text"].(string)
			if text == "" {
				return nil, fmt.Errorf("%s.items[%d].text is required for card block type 'list'", fieldPath, index)
			}
			normalized = append(normalized, cloneStructuredEntry(typed))
		default:
			return nil, fmt.Errorf("%s.items[%d] must be a string or object for card block type 'list'", fieldPath, index)
		}
	}
	return normalized, nil
}

func normalizeTableRows(raw any, fieldPath string) ([]any, error) {
	rows, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.rows must be an array for card block type 'table'", fieldPath)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s.rows must be a non-empty array for card block type 'table'", fieldPath)
	}
	return rows, nil
}

func normalizeFormFields(raw any, fieldPath string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.fields must be an array when %s.type='form'", fieldPath, fieldPath)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s.fields must be a non-empty array when %s.type='form'", fieldPath, fieldPath)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s.fields[%d] must be an object when %s.type='form'", fieldPath, index, fieldPath)
		}
		field := cloneStructuredEntry(entry)
		name, _ := field["name"].(string)
		label, _ := field["label"].(string)
		if name == "" || label == "" {
			return nil, fmt.Errorf("%s.fields[%d].name and %s.fields[%d].label are required when %s.type='form'", fieldPath, index, fieldPath, index, fieldPath)
		}
		normalized = append(normalized, field)
	}
	return normalized, nil
}

func normalizeProgressSteps(raw any, fieldPath string) ([]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, false, fmt.Errorf("%s.steps must be an array when %s.type='progress'", fieldPath, fieldPath)
	}
	if len(items) == 0 {
		return nil, false, fmt.Errorf("%s.steps must be a non-empty array when %s.type='progress'", fieldPath, fieldPath)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("%s.steps[%d] must be an object when %s.type='progress'", fieldPath, index, fieldPath)
		}
		step := cloneStructuredEntry(entry)
		assignFirstStringAlias(step, "detail", "detail", "description", "message", "content")
		label, _ := step["label"].(string)
		if label == "" {
			return nil, false, fmt.Errorf("%s.steps[%d].label is required when %s.type='progress'", fieldPath, index, fieldPath)
		}
		normalized = append(normalized, step)
	}

	return normalized, true, nil
}

func normalizeTodoItems(raw any, fieldPath string) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.items must be an array when %s.type='todo'", fieldPath, fieldPath)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s.items must be a non-empty array when %s.type='todo'", fieldPath, fieldPath)
	}

	normalized := make([]any, 0, len(items))
	for index, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s.items[%d] must be an object when %s.type='todo'", fieldPath, index, fieldPath)
		}
		todoItem := cloneStructuredEntry(entry)
		assignFirstStringAlias(todoItem, "title", "title", "label", "text", "step")
		assignFirstStringAlias(todoItem, "detail", "detail", "description", "message")
		title, _ := todoItem["title"].(string)
		if title == "" {
			return nil, fmt.Errorf("%s.items[%d].title is required when %s.type='todo'", fieldPath, index, fieldPath)
		}
		status, _ := todoItem["status"].(string)
		if status == "" {
			if done, ok := todoItem["done"].(bool); ok {
				if done {
					todoItem["status"] = "completed"
				} else {
					todoItem["status"] = "not-started"
				}
			} else {
				todoItem["status"] = "not-started"
			}
		} else if status != "not-started" && status != "in-progress" && status != "completed" {
			return nil, fmt.Errorf("%s.items[%d].status must be one of not-started, in-progress, completed when %s.type='todo'", fieldPath, index, fieldPath)
		}
		normalized = append(normalized, todoItem)
	}
	return normalized, nil
}
