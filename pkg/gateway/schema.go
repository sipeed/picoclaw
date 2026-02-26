package gateway

import (
	"reflect"
	"strings"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
)

// SchemaField describes a single configuration field.
type SchemaField struct {
	Key      string      `json:"key"`
	Label    string      `json:"label"`
	Type     string      `json:"type"`
	Secret   bool        `json:"secret"`
	Default  interface{} `json:"default"`
}

// SchemaSection describes a top-level configuration section.
type SchemaSection struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Fields []SchemaField `json:"fields"`
}

// SchemaResponse is the top-level schema response.
type SchemaResponse struct {
	Sections []SchemaSection `json:"sections"`
}

// acronyms maps lowercase abbreviations to their uppercase forms for label generation.
var acronyms = map[string]string{
	"api": "API", "llm": "LLM", "url": "URL", "ws": "WS",
	"id": "ID", "mcp": "MCP",
}

// secretKeys lists JSON keys that contain sensitive values.
var secretKeys = map[string]bool{
	"api_key":              true,
	"token":                true,
	"bot_token":            true,
	"app_token":            true,
	"channel_secret":       true,
	"channel_access_token": true,
}

// BuildSchema generates a SchemaResponse by reflecting over a default Config.
func BuildSchema(defaultCfg *config.Config) SchemaResponse {
	var sections []SchemaSection

	cfgType := reflect.TypeOf(defaultCfg).Elem()
	cfgVal := reflect.ValueOf(defaultCfg).Elem()

	for i := 0; i < cfgType.NumField(); i++ {
		field := cfgType.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := jsonKey(field)
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		section := SchemaSection{
			Key:   jsonTag,
			Label: snakeToTitle(jsonTag),
		}

		fieldVal := cfgVal.Field(i)
		section.Fields = buildFields(field.Type, fieldVal, "")
		sections = append(sections, section)
	}

	return SchemaResponse{Sections: sections}
}

// buildFields recursively collects fields from a struct type, flattening nested structs
// with dot-separated key prefixes.
func buildFields(t reflect.Type, v reflect.Value, prefix string) []SchemaField {
	var fields []SchemaField

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if v.IsValid() && !v.IsNil() {
			v = v.Elem()
		}
	}

	if t.Kind() != reflect.Struct {
		return fields
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		jk := jsonKey(sf)
		if jk == "" || jk == "-" {
			continue
		}

		fullKey := jk
		if prefix != "" {
			fullKey = prefix + "." + jk
		}

		fieldVal := v.Field(i)
		ft := sf.Type

		// Dereference pointer types
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
			if fieldVal.IsValid() && !fieldVal.IsNil() {
				fieldVal = fieldVal.Elem()
			}
		}

		schemaType := goTypeToSchema(ft)
		if schemaType == "object" {
			// Nested struct: recurse and flatten
			fields = append(fields, buildFields(ft, fieldVal, fullKey)...)
			continue
		}

		var defVal interface{}
		if fieldVal.IsValid() {
			defVal = fieldVal.Interface()
		}

		fields = append(fields, SchemaField{
			Key:     fullKey,
			Label:   snakeToTitle(jk),
			Type:    schemaType,
			Secret:  secretKeys[jk],
			Default: defVal,
		})
	}

	return fields
}

// goTypeToSchema maps a Go reflect.Type to a schema type string.
func goTypeToSchema(t reflect.Type) string {
	// Check for FlexibleStringSlice by name
	if t.Name() == "FlexibleStringSlice" {
		return "[]string"
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return "[]string"
		}
		return "[]any"
	case reflect.Map:
		return "map"
	case reflect.Struct:
		return "object"
	default:
		return "any"
	}
}

// jsonKey extracts the JSON field name from a struct field's tag.
func jsonKey(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	parts := strings.SplitN(tag, ",", 2)
	return parts[0]
}

// snakeToTitle converts a snake_case string to Title Case, applying acronym rules.
func snakeToTitle(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if upper, ok := acronyms[strings.ToLower(p)]; ok {
			parts[i] = upper
		} else if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
