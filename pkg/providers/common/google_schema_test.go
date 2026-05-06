package common

import "testing"

func TestSanitizeSchemaForGemini_DereferencesRefsAndFlattensUnions(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"parent": map[string]any{
				"anyOf": []any{
					map[string]any{"$ref": "#/$defs/pageParent"},
					map[string]any{"$ref": "#/$defs/databaseParent"},
				},
			},
			"icon": map[string]any{
				"anyOf": []any{
					map[string]any{"$ref": "#/$defs/emoji"},
					map[string]any{"type": "null"},
				},
			},
			"data": map[string]any{
				"$ref": "#/$defs/dataPayload",
			},
		},
		"required": []any{"parent", "icon", "missing"},
		"$defs": map[string]any{
			"pageParent": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{
						"type": "string",
					},
				},
				"required": []any{"page_id"},
			},
			"databaseParent": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"database_id": map[string]any{
						"type": "string",
					},
				},
				"required": []any{"database_id"},
			},
			"emoji": map[string]any{
				"type":    "string",
				"pattern": "^:[a-z_]+:$",
			},
			"dataPayload": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"name": map[string]any{
						"type":      "string",
						"minLength": 1,
					},
					"count": map[string]any{
						"type":    "integer",
						"minimum": 1,
					},
				},
				"required": []any{"name"},
			},
		},
	}

	got := SanitizeSchemaForGemini(schema)
	assertSchemaKeyAbsent(t, got, "$defs")
	assertSchemaKeyAbsent(t, got, "$ref")
	assertSchemaKeyAbsent(t, got, "anyOf")
	assertSchemaKeyAbsent(t, got, "oneOf")
	assertSchemaKeyAbsent(t, got, "allOf")
	assertSchemaKeyAbsent(t, got, "additionalProperties")
	assertSchemaKeyAbsent(t, got, "pattern")
	assertSchemaKeyAbsent(t, got, "minLength")
	assertSchemaKeyAbsent(t, got, "minimum")

	if got["type"] != "object" {
		t.Fatalf("top-level type = %#v, want object", got["type"])
	}

	props, ok := got["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %#v, want map", got["properties"])
	}

	parent, ok := props["parent"].(map[string]any)
	if !ok {
		t.Fatalf("parent schema = %#v, want map", props["parent"])
	}
	if parent["type"] != "object" {
		t.Fatalf("parent.type = %#v, want object", parent["type"])
	}
	parentProps, ok := parent["properties"].(map[string]any)
	if !ok {
		t.Fatalf("parent.properties = %#v, want map", parent["properties"])
	}
	if _, found := parentProps["page_id"]; !found {
		t.Fatalf("parent.properties missing page_id: %#v", parentProps)
	}
	if _, found := parentProps["database_id"]; !found {
		t.Fatalf("parent.properties missing database_id: %#v", parentProps)
	}
	if _, hasRequired := parent["required"]; hasRequired {
		t.Fatalf("parent.required = %#v, want omitted for merged anyOf branches", parent["required"])
	}

	icon, ok := props["icon"].(map[string]any)
	if !ok {
		t.Fatalf("icon schema = %#v, want map", props["icon"])
	}
	if icon["type"] != "string" {
		t.Fatalf("icon.type = %#v, want string", icon["type"])
	}

	data, ok := props["data"].(map[string]any)
	if !ok {
		t.Fatalf("data schema = %#v, want map", props["data"])
	}
	if data["type"] != "object" {
		t.Fatalf("data.type = %#v, want object", data["type"])
	}
	dataProps, ok := data["properties"].(map[string]any)
	if !ok {
		t.Fatalf("data.properties = %#v, want map", data["properties"])
	}
	if _, found := dataProps["name"]; !found {
		t.Fatalf("data.properties missing name: %#v", dataProps)
	}
	if _, found := dataProps["count"]; !found {
		t.Fatalf("data.properties missing count: %#v", dataProps)
	}

	required, ok := got["required"].([]string)
	if !ok {
		t.Fatalf("required = %#v, want []string", got["required"])
	}
	if len(required) != 2 || required[0] != "parent" || required[1] != "icon" {
		t.Fatalf("required = %#v, want [parent icon]", required)
	}
}

func TestSanitizeSchemaForGemini_MergesAllOfAndFiltersRequired(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"payload": map[string]any{
				"allOf": []any{
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id": map[string]any{
								"type": "string",
							},
						},
						"required": []any{"id"},
					},
					map[string]any{
						"properties": map[string]any{
							"name": map[string]any{
								"type": "string",
							},
							"count": map[string]any{
								"type":    "integer",
								"minimum": 1,
							},
						},
						"required": []any{"name", "missing"},
					},
				},
			},
		},
	}

	got := SanitizeSchemaForGemini(schema)
	props := got["properties"].(map[string]any)
	payload := props["payload"].(map[string]any)

	if payload["type"] != "object" {
		t.Fatalf("payload.type = %#v, want object", payload["type"])
	}
	payloadProps, ok := payload["properties"].(map[string]any)
	if !ok {
		t.Fatalf("payload.properties = %#v, want map", payload["properties"])
	}
	for _, key := range []string{"id", "name", "count"} {
		if _, found := payloadProps[key]; !found {
			t.Fatalf("payload.properties missing %q: %#v", key, payloadProps)
		}
	}

	required, ok := payload["required"].([]string)
	if !ok {
		t.Fatalf("payload.required = %#v, want []string", payload["required"])
	}
	if len(required) != 2 || required[0] != "id" || required[1] != "name" {
		t.Fatalf("payload.required = %#v, want [id name]", required)
	}

	assertSchemaKeyAbsent(t, payload, "allOf")
	assertSchemaKeyAbsent(t, payload, "minimum")
}

func TestSanitizeSchemaForGemini_HandlesRecursiveRefs(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tree": map[string]any{
				"$ref": "#/$defs/node",
			},
		},
		"$defs": map[string]any{
			"node": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
					"child": map[string]any{
						"$ref": "#/$defs/node",
					},
				},
			},
		},
	}

	got := SanitizeSchemaForGemini(schema)
	props := got["properties"].(map[string]any)
	tree := props["tree"].(map[string]any)
	if tree["type"] != "object" {
		t.Fatalf("tree.type = %#v, want object", tree["type"])
	}
	assertSchemaKeyAbsent(t, tree, "$ref")
}

func assertSchemaKeyAbsent(t *testing.T, value any, key string) {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		if _, found := typed[key]; found {
			t.Fatalf("schema still contains key %q: %#v", key, typed)
		}
		for _, nested := range typed {
			assertSchemaKeyAbsent(t, nested, key)
		}
	case []any:
		for _, nested := range typed {
			assertSchemaKeyAbsent(t, nested, key)
		}
	case []string:
		return
	}
}
