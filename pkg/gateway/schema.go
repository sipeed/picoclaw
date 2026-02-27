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
	Group    string      `json:"group,omitempty"`
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

// secretKeys lists JSON keys that contain sensitive values.
var secretKeys = map[string]bool{
	"api_key":              true,
	"token":                true,
	"bot_token":            true,
	"app_token":            true,
	"channel_secret":       true,
	"channel_access_token": true,
}

// directoryKeys lists full dot-separated JSON keys that represent directory paths.
// Fields matching these keys are reported as type "directory" so that
// Android can render a SAF directory-picker instead of a plain text field.
var directoryKeys = map[string]bool{
	"defaults.workspace": true,
	"defaults.data_dir":  true,
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

		// Gateway is managed via the Connection screen, not the Backend Config UI.
		if jsonTag == "gateway" {
			continue
		}

		section := SchemaSection{
			Key:   jsonTag,
			Label: labelTag(field),
		}

		fieldVal := cfgVal.Field(i)
		section.Fields = buildFields(field.Type, fieldVal, "", "")

		// If every field shares the same single group, the header is redundant — clear it.
		groups := map[string]bool{}
		for _, f := range section.Fields {
			if f.Group != "" {
				groups[f.Group] = true
			}
		}
		if len(groups) <= 1 {
			for j := range section.Fields {
				section.Fields[j].Group = ""
			}
		}

		sections = append(sections, section)
	}

	return SchemaResponse{Sections: sections}
}

// buildFields recursively collects fields from a struct type, flattening nested structs
// with dot-separated key prefixes. The group parameter propagates the label of the
// enclosing struct so that leaf fields can be grouped under a header in the UI.
func buildFields(t reflect.Type, v reflect.Value, prefix string, group string) []SchemaField {
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
			// Nested struct: recurse and flatten.
			// Use the nested struct's label tag as group; fall back to current group.
			childGroup := labelTag(sf)
			if childGroup == "" {
				childGroup = group
			}
			fields = append(fields, buildFields(ft, fieldVal, fullKey, childGroup)...)
			continue
		}

		var defVal interface{}
		if fieldVal.IsValid() {
			defVal = fieldVal.Interface()
		}

		// Override type for directory-path fields
		if schemaType == "string" && directoryKeys[fullKey] {
			schemaType = "directory"
		}

		fields = append(fields, SchemaField{
			Key:     fullKey,
			Label:   labelTag(sf),
			Group:   group,
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

// labelTag reads the "label" struct tag from a field.
func labelTag(f reflect.StructField) string {
	return f.Tag.Get("label")
}
