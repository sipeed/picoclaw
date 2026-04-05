package utils

import (
	"bytes"
	"encoding/json"
)

// MarshalNoEscape serializes a value to JSON without HTML escaping.
// This is useful for user-facing JSON output where characters like
// '&', '<', and '>' should remain unescaped for readability.
func MarshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}
