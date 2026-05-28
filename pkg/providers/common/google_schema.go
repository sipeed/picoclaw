package common

import (
	"strconv"
	"strings"
)

const maxGeminiSchemaDepth = 64

var geminiSupportedTypes = map[string]bool{
	"array":   true,
	"boolean": true,
	"integer": true,
	"number":  true,
	"object":  true,
	"string":  true,
}

// SanitizeSchemaForGoogle reduces a JSON Schema to the conservative subset
// accepted by Google/Gemini-style function declarations. It resolves local
// refs, collapses composition keywords like anyOf/oneOf/allOf, and strips
// advanced keywords that Gemini-compatible backends often reject.
func SanitizeSchemaForGoogle(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	sanitizer := geminiSchemaSanitizer{root: schema}
	result := sanitizer.sanitizeNode(schema, nil, 0)
	if len(result) == 0 {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	if _, hasProps := result["properties"]; hasProps {
		result["type"] = "object"
	}
	return result
}

// SanitizeSchemaForGemini is kept as a compatibility alias for the original
// Google/Gemini sanitizer name.
func SanitizeSchemaForGemini(schema map[string]any) map[string]any {
	return SanitizeSchemaForGoogle(schema)
}

type geminiSchemaSanitizer struct {
	root map[string]any
}

func (s geminiSchemaSanitizer) sanitizeNode(
	node map[string]any,
	refTrail map[string]struct{},
	depth int,
) map[string]any {
	if node == nil || depth > maxGeminiSchemaDepth {
		return map[string]any{}
	}

	normalized := s.normalizeNode(node, refTrail, depth)
	if len(normalized) == 0 {
		return map[string]any{}
	}

	result := make(map[string]any)

	if desc, ok := normalized["description"].(string); ok && strings.TrimSpace(desc) != "" {
		result["description"] = desc
	}

	if schemaType := sanitizeGeminiSchemaType(normalized["type"]); schemaType != "" {
		result["type"] = schemaType
	}

	if enumValues := sanitizeGeminiEnum(normalized["enum"]); len(enumValues) > 0 {
		result["enum"] = enumValues
	}

	if propsRaw, ok := normalized["properties"].(map[string]any); ok {
		props := make(map[string]any, len(propsRaw))
		for name, rawProp := range propsRaw {
			propSchema, ok := rawProp.(map[string]any)
			if !ok {
				continue
			}
			sanitizedProp := s.sanitizeNode(propSchema, refTrail, depth+1)
			if len(sanitizedProp) == 0 {
				sanitizedProp = map[string]any{}
			}
			props[name] = sanitizedProp
		}
		result["properties"] = props
		result["type"] = "object"
		if required := sanitizeGeminiRequired(normalized["required"], props); len(required) > 0 {
			result["required"] = required
		}
	}

	if itemsRaw, ok := normalized["items"].(map[string]any); ok {
		items := s.sanitizeNode(itemsRaw, refTrail, depth+1)
		if len(items) == 0 {
			items = map[string]any{}
		}
		result["items"] = items
		if _, hasType := result["type"]; !hasType {
			result["type"] = "array"
		}
	}

	return result
}

func (s geminiSchemaSanitizer) normalizeNode(
	node map[string]any,
	refTrail map[string]struct{},
	depth int,
) map[string]any {
	if node == nil || depth > maxGeminiSchemaDepth {
		return map[string]any{}
	}

	normalized := cloneGeminiSchemaMap(node)

	if ref, ok := normalized["$ref"].(string); ok {
		delete(normalized, "$ref")
		if _, seen := refTrail[ref]; !seen {
			if target, ok := s.resolveLocalSchemaRef(ref); ok {
				nextTrail := cloneRefTrail(refTrail)
				nextTrail[ref] = struct{}{}
				normalized = mergeGeminiSchemaMaps(
					s.normalizeNode(target, nextTrail, depth+1),
					normalized,
				)
			}
		}
	}

	if rawAllOf, ok := normalized["allOf"]; ok {
		delete(normalized, "allOf")
		for _, part := range schemaSlice(rawAllOf) {
			normalized = mergeGeminiSchemaMaps(
				normalized,
				s.normalizeNode(part, refTrail, depth+1),
			)
		}
	}

	if rawAnyOf, ok := normalized["anyOf"]; ok {
		delete(normalized, "anyOf")
		normalized = mergeGeminiSchemaMaps(
			s.mergeUnionBranches(schemaSlice(rawAnyOf), refTrail, depth+1),
			normalized,
		)
	}

	if rawOneOf, ok := normalized["oneOf"]; ok {
		delete(normalized, "oneOf")
		normalized = mergeGeminiSchemaMaps(
			s.mergeUnionBranches(schemaSlice(rawOneOf), refTrail, depth+1),
			normalized,
		)
	}

	return normalized
}

func (s geminiSchemaSanitizer) mergeUnionBranches(
	branches []map[string]any,
	refTrail map[string]struct{},
	depth int,
) map[string]any {
	if len(branches) == 0 {
		return map[string]any{}
	}

	objectBranches := make([]map[string]any, 0, len(branches))
	arrayBranches := make([]map[string]any, 0, len(branches))
	nonNullBranches := make([]map[string]any, 0, len(branches))
	sameType := ""
	sameTypeConsistent := true

	for _, branch := range branches {
		normalized := s.normalizeNode(branch, refTrail, depth+1)
		if len(normalized) == 0 {
			continue
		}

		branchType := geminiSchemaBranchType(normalized["type"])
		if branchType == "null" {
			continue
		}
		nonNullBranches = append(nonNullBranches, normalized)

		if sameType == "" {
			sameType = branchType
		} else if branchType != "" && branchType != sameType {
			sameTypeConsistent = false
		}

		if branchType == "object" || hasSchemaProperties(normalized) {
			objectBranches = append(objectBranches, normalized)
			continue
		}
		if branchType == "array" || hasSchemaItems(normalized) {
			arrayBranches = append(arrayBranches, normalized)
		}
	}

	if len(nonNullBranches) == 0 {
		return map[string]any{}
	}
	if len(objectBranches) > 0 {
		return mergeUnionObjectSchemas(objectBranches)
	}
	if len(arrayBranches) == len(nonNullBranches) && len(arrayBranches) > 0 {
		return mergeUnionArraySchemas(arrayBranches)
	}
	if sameTypeConsistent && sameType != "" {
		merged := map[string]any{}
		for _, branch := range nonNullBranches {
			merged = mergeGeminiSchemaMaps(merged, branch)
		}
		return merged
	}

	best := nonNullBranches[0]
	bestScore := geminiUnionBranchScore(best)
	for _, branch := range nonNullBranches[1:] {
		if score := geminiUnionBranchScore(branch); score > bestScore {
			best = branch
			bestScore = score
		}
	}
	return cloneGeminiSchemaMap(best)
}

func (s geminiSchemaSanitizer) resolveLocalSchemaRef(ref string) (map[string]any, bool) {
	if ref == "#" {
		return s.root, true
	}
	if !strings.HasPrefix(ref, "#/") {
		return nil, false
	}

	var current any = s.root
	for _, rawToken := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
		switch value := current.(type) {
		case map[string]any:
			next, ok := value[token]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(value) {
				return nil, false
			}
			current = value[index]
		default:
			return nil, false
		}
	}

	resolved, ok := current.(map[string]any)
	return resolved, ok
}

func mergeUnionObjectSchemas(branches []map[string]any) map[string]any {
	merged := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}

	var commonRequired map[string]struct{}
	var requiredOrder []string

	for i, branch := range branches {
		merged = mergeGeminiSchemaMaps(merged, branch)

		required := requiredStrings(branch["required"])
		if i == 0 {
			commonRequired = make(map[string]struct{}, len(required))
			requiredOrder = append(requiredOrder, required...)
			for _, name := range required {
				commonRequired[name] = struct{}{}
			}
			continue
		}

		current := make(map[string]struct{}, len(required))
		for _, name := range required {
			current[name] = struct{}{}
		}
		for name := range commonRequired {
			if _, ok := current[name]; !ok {
				delete(commonRequired, name)
			}
		}
	}

	if len(commonRequired) > 0 {
		filtered := make([]string, 0, len(commonRequired))
		for _, name := range requiredOrder {
			if _, ok := commonRequired[name]; ok {
				filtered = append(filtered, name)
			}
		}
		if len(filtered) > 0 {
			merged["required"] = filtered
		}
	} else {
		delete(merged, "required")
	}

	return merged
}

func mergeUnionArraySchemas(branches []map[string]any) map[string]any {
	merged := map[string]any{
		"type": "array",
	}
	for _, branch := range branches {
		merged = mergeGeminiSchemaMaps(merged, branch)
	}
	return merged
}

func mergeGeminiSchemaMaps(base map[string]any, overlay map[string]any) map[string]any {
	if len(base) == 0 {
		return cloneGeminiSchemaMap(overlay)
	}
	if len(overlay) == 0 {
		return cloneGeminiSchemaMap(base)
	}

	result := cloneGeminiSchemaMap(base)
	for key, value := range overlay {
		switch key {
		case "properties":
			overlayProps, ok := value.(map[string]any)
			if !ok {
				continue
			}
			existing, _ := result["properties"].(map[string]any)
			mergedProps := cloneGeminiSchemaMap(existing)
			if mergedProps == nil {
				mergedProps = make(map[string]any, len(overlayProps))
			}
			for name, rawProp := range overlayProps {
				propSchema, ok := rawProp.(map[string]any)
				if !ok {
					continue
				}
				if existingProp, ok := mergedProps[name].(map[string]any); ok {
					mergedProps[name] = mergeGeminiSchemaMaps(existingProp, propSchema)
				} else {
					mergedProps[name] = cloneGeminiSchemaMap(propSchema)
				}
			}
			result["properties"] = mergedProps
		case "items":
			overlayItems, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if existingItems, ok := result["items"].(map[string]any); ok {
				result["items"] = mergeGeminiSchemaMaps(existingItems, overlayItems)
			} else {
				result["items"] = cloneGeminiSchemaMap(overlayItems)
			}
		case "required":
			if merged := mergeRequiredLists(result["required"], value); len(merged) > 0 {
				result["required"] = merged
			}
		case "type":
			if mergedType := mergeGeminiSchemaTypes(result["type"], value); mergedType != "" {
				result["type"] = mergedType
			} else {
				delete(result, "type")
			}
		case "description":
			desc, ok := value.(string)
			if ok && strings.TrimSpace(desc) != "" {
				result["description"] = desc
			}
		default:
			result[key] = cloneGeminiSchemaValue(value)
		}
	}

	return result
}

func mergeGeminiSchemaTypes(left any, right any) string {
	leftType := geminiSchemaBranchType(left)
	rightType := geminiSchemaBranchType(right)

	switch {
	case leftType == "":
		return rightType
	case rightType == "":
		return leftType
	case leftType == rightType:
		return leftType
	case leftType == "null":
		return rightType
	case rightType == "null":
		return leftType
	default:
		return ""
	}
}

func sanitizeGeminiSchemaType(raw any) string {
	typeName := geminiSchemaBranchType(raw)
	if typeName == "null" {
		return ""
	}
	return typeName
}

func geminiSchemaBranchType(raw any) string {
	switch value := raw.(type) {
	case string:
		if value == "null" {
			return value
		}
		if geminiSupportedTypes[value] {
			return value
		}
		return ""
	case []string:
		return geminiSchemaBranchType(stringSliceToAny(value))
	case []any:
		candidate := ""
		sawNull := false
		for _, item := range value {
			typeName, ok := item.(string)
			if !ok {
				continue
			}
			if typeName == "null" {
				sawNull = true
				continue
			}
			if !geminiSupportedTypes[typeName] {
				continue
			}
			if candidate == "" {
				candidate = typeName
				continue
			}
			if candidate != typeName {
				return ""
			}
		}
		if candidate == "" && sawNull {
			return "null"
		}
		return candidate
	default:
		return ""
	}
}

func sanitizeGeminiEnum(raw any) []any {
	values, ok := raw.([]any)
	if !ok {
		if stringValues, ok := raw.([]string); ok {
			return stringSliceToAny(stringValues)
		}
		return nil
	}

	sanitized := make([]any, 0, len(values))
	for _, value := range values {
		switch value.(type) {
		case string, bool, float64, int, int32, int64:
			sanitized = append(sanitized, value)
		}
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}

func sanitizeGeminiRequired(raw any, properties map[string]any) []string {
	required := requiredStrings(raw)
	if len(required) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(required))
	seen := make(map[string]struct{}, len(required))
	for _, name := range required {
		if _, ok := properties[name]; !ok {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		filtered = append(filtered, name)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func requiredStrings(raw any) []string {
	switch value := raw.(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		required := make([]string, 0, len(value))
		for _, item := range value {
			name, ok := item.(string)
			if ok {
				required = append(required, name)
			}
		}
		return required
	default:
		return nil
	}
}

func mergeRequiredLists(left any, right any) []string {
	merged := make([]string, 0)
	seen := map[string]struct{}{}

	for _, name := range append(requiredStrings(left), requiredStrings(right)...) {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		merged = append(merged, name)
	}

	return merged
}

func geminiUnionBranchScore(schema map[string]any) int {
	score := 0
	if hasSchemaProperties(schema) {
		score += 20
	}
	if hasSchemaItems(schema) {
		score += 10
	}
	if _, ok := schema["enum"]; ok {
		score += 5
	}
	if _, ok := schema["description"]; ok {
		score += 2
	}
	score += len(schema)
	return score
}

func hasSchemaProperties(schema map[string]any) bool {
	props, ok := schema["properties"].(map[string]any)
	return ok && len(props) > 0
}

func hasSchemaItems(schema map[string]any) bool {
	_, ok := schema["items"].(map[string]any)
	return ok
}

func schemaSlice(raw any) []map[string]any {
	switch value := raw.(type) {
	case []map[string]any:
		return append([]map[string]any(nil), value...)
	case []any:
		schemas := make([]map[string]any, 0, len(value))
		for _, item := range value {
			schema, ok := item.(map[string]any)
			if ok {
				schemas = append(schemas, schema)
			}
		}
		return schemas
	default:
		return nil
	}
}

func cloneGeminiSchemaMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneGeminiSchemaValue(value)
	}
	return out
}

func cloneGeminiSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneGeminiSchemaMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneGeminiSchemaValue(item)
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	default:
		return typed
	}
}

func cloneRefTrail(in map[string]struct{}) map[string]struct{} {
	if len(in) == 0 {
		return make(map[string]struct{})
	}
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

func stringSliceToAny(values []string) []any {
	if len(values) == 0 {
		return nil
	}
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}
