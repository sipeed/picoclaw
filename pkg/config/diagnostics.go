package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

func decodeJSONWithDiagnostics(data []byte, target any, label string) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return wrapJSONError(data, err, label)
	}

	unknownFields := collectUnknownJSONFields(raw, reflect.TypeOf(target), "")
	if len(unknownFields) > 0 {
		sort.Strings(unknownFields)
		return fmt.Errorf(
			"%s contains unknown field(s): %s",
			label,
			strings.Join(unknownFields, ", "),
		)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return wrapJSONError(data, err, label)
	}
	return nil
}

func DiagnosticSummary(err error) string {
	if err == nil {
		return ""
	}
	summary, _ := splitDiagnosticError(err.Error())
	return stripANSISequences(summary)
}

func formatDiagnosticLogMessage(prefix string, err error) string {
	if err == nil {
		return prefix
	}

	summary, preview := splitDiagnosticError(err.Error())
	summary = stripANSISequences(summary)
	if preview == "" {
		if summary == "" {
			return prefix
		}
		return prefix + ": " + summary
	}
	if summary == "" {
		return prefix + "\n" + preview
	}
	return prefix + ": " + summary + "\n" + preview
}

func wrapJSONError(data []byte, err error, label string) error {
	switch e := err.(type) {
	case *json.SyntaxError:
		line, column := lineAndColumnForOffset(data, e.Offset)
		preview := diagnosticPreviewForOffset(data, e.Offset)
		if preview != "" {
			return fmt.Errorf(
				"%s syntax error at line %d, column %d: %w\n%s",
				label,
				line,
				column,
				err,
				preview,
			)
		}
		return fmt.Errorf("%s syntax error at line %d, column %d: %w", label, line, column, err)
	case *json.UnmarshalTypeError:
		line, column := lineAndColumnForOffset(data, e.Offset)
		preview := diagnosticPreviewForOffset(data, e.Offset)
		field := strings.TrimSpace(e.Field)
		if field != "" {
			if preview != "" {
				return fmt.Errorf(
					"%s type error at line %d, column %d for field %q: expected %s but got %s\n%s",
					label,
					line,
					column,
					field,
					e.Type.String(),
					e.Value,
					preview,
				)
			}
			return fmt.Errorf(
				"%s type error at line %d, column %d for field %q: expected %s but got %s",
				label,
				line,
				column,
				field,
				e.Type.String(),
				e.Value,
			)
		}
		if preview != "" {
			return fmt.Errorf(
				"%s type error at line %d, column %d: expected %s but got %s\n%s",
				label,
				line,
				column,
				e.Type.String(),
				e.Value,
				preview,
			)
		}
		return fmt.Errorf(
			"%s type error at line %d, column %d: expected %s but got %s",
			label,
			line,
			column,
			e.Type.String(),
			e.Value,
		)
	default:
		return fmt.Errorf("failed to parse %s: %w", label, err)
	}
}

func splitDiagnosticError(message string) (string, string) {
	if idx := strings.IndexByte(message, '\n'); idx >= 0 {
		return message[:idx], message[idx+1:]
	}
	return message, ""
}

func stripANSISequences(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) || s[i+1] != '[' {
			continue
		}
		i += 2
		for i < len(s) {
			c := s[i]
			if c >= '@' && c <= '~' {
				break
			}
			i++
		}
	}

	return b.String()
}

func diagnosticPreviewForOffset(data []byte, offset int64) string {
	if len(data) == 0 {
		return ""
	}

	start, end := lineBoundsForOffset(data, offset)
	if start >= end {
		return ""
	}

	lineNumber, column := lineAndColumnForOffset(data, offset)
	line := strings.TrimRight(string(data[start:end]), "\r\n")
	if strings.TrimSpace(line) == "" {
		return ""
	}

	trimmedLine, trimOffset := trimDiagnosticLine(line, column)
	if trimmedLine == "" {
		return ""
	}

	prefix := fmt.Sprintf("%4d | ", lineNumber)
	caretColumn := column - trimOffset
	if caretColumn < 1 {
		caretColumn = 1
	}

	if diagnosticsUseColor() {
		linePrefix := "\x1b[2m" + prefix + "\x1b[0m"
		caretPrefix := "\x1b[2m" + strings.Repeat(" ", len(fmt.Sprintf("%4d", lineNumber))) + " | " + "\x1b[0m"
		highlighted := highlightDiagnosticColumn(trimmedLine, caretColumn)
		caretPad := strings.Repeat(" ", maxRuneCount(trimmedLine, caretColumn-1))
		return fmt.Sprintf(
			"  %s%s\n  %s%s\x1b[1;31m^\x1b[0m",
			linePrefix,
			highlighted,
			caretPrefix,
			caretPad,
		)
	}

	caretPrefix := strings.Repeat(" ", len(prefix))
	caretPad := strings.Repeat(" ", maxRuneCount(trimmedLine, caretColumn-1))
	return fmt.Sprintf(
		"  %s%s\n  %s%s^",
		prefix,
		trimmedLine,
		caretPrefix,
		caretPad,
	)
}

func lineAndColumnForOffset(data []byte, offset int64) (int, int) {
	if offset <= 0 {
		return 1, 1
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}

	line := 1
	column := 1
	for i := int64(0); i < offset-1; i++ {
		if data[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

func lineBoundsForOffset(data []byte, offset int64) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}

	if offset <= 0 {
		offset = 1
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}

	index := int(offset - 1)
	if index < 0 {
		index = 0
	}
	if index >= len(data) {
		index = len(data) - 1
	}

	start := index
	for start > 0 && data[start-1] != '\n' {
		start--
	}

	end := index
	for end < len(data) && data[end] != '\n' {
		end++
	}

	return start, end
}

func trimDiagnosticLine(line string, column int) (string, int) {
	runes := []rune(line)
	if len(runes) == 0 {
		return "", 0
	}

	if len(runes) <= 160 {
		return line, 0
	}

	const contextBefore = 60
	const maxWidth = 160

	start := column - 1 - contextBefore
	if start < 0 {
		start = 0
	}
	if start > len(runes)-maxWidth {
		start = len(runes) - maxWidth
	}
	if start < 0 {
		start = 0
	}

	end := start + maxWidth
	if end > len(runes) {
		end = len(runes)
	}

	trimmed := string(runes[start:end])
	trimOffset := start

	if start > 0 {
		trimmed = "..." + trimmed
		trimOffset -= 3
	}
	if end < len(runes) {
		trimmed += "..."
	}

	return trimmed, trimOffset
}

func diagnosticsUseColor() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func highlightDiagnosticColumn(line string, column int) string {
	runes := []rune(line)
	if column < 1 || column > len(runes) {
		return line
	}

	index := column - 1
	return string(runes[:index]) + "\x1b[31m" + string(runes[index]) + "\x1b[0m" + string(runes[index+1:])
}

func maxRuneCount(s string, count int) int {
	if count <= 0 {
		return 0
	}
	runes := []rune(s)
	if count > len(runes) {
		count = len(runes)
	}
	return utf8.RuneCountInString(string(runes[:count]))
}

func collectUnknownJSONFields(raw any, targetType reflect.Type, path string) []string {
	targetType = derefType(targetType)
	if targetType == nil {
		return nil
	}

	switch targetType.Kind() {
	case reflect.Struct:
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		fieldMap := jsonFieldTypeMap(targetType)
		var issues []string
		for key, value := range obj {
			fieldType, exists := fieldMap[key]
			fieldPath := appendJSONPath(path, key)
			if !exists {
				issues = append(issues, fieldPath)
				continue
			}
			issues = append(issues, collectUnknownJSONFields(value, fieldType, fieldPath)...)
		}
		return issues
	case reflect.Slice, reflect.Array:
		items, ok := raw.([]any)
		if !ok {
			return nil
		}
		var issues []string
		elemType := targetType.Elem()
		for i, item := range items {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			issues = append(issues, collectUnknownJSONFields(item, elemType, itemPath)...)
		}
		return issues
	case reflect.Map:
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		var issues []string
		elemType := targetType.Elem()
		for key, value := range obj {
			fieldPath := appendJSONPath(path, key)
			issues = append(issues, collectUnknownJSONFields(value, elemType, fieldPath)...)
		}
		return issues
	default:
		return nil
	}
}

func jsonFieldTypeMap(t reflect.Type) map[string]reflect.Type {
	result := make(map[string]reflect.Type)
	populateJSONFieldTypeMap(result, derefType(t))
	return result
}

func populateJSONFieldTypeMap(result map[string]reflect.Type, t reflect.Type) {
	if t == nil || t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			continue
		}

		if field.Anonymous && name == "" {
			populateJSONFieldTypeMap(result, derefType(field.Type))
			continue
		}

		if name == "" {
			name = field.Name
		}
		result[name] = field.Type
	}
}

func derefType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func appendJSONPath(path, segment string) string {
	if path == "" {
		return segment
	}
	return path + "." + segment
}
