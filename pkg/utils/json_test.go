package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalNoEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "ampersand",
			input:    map[string]string{"cmd": "cmd1 && cmd2"},
			expected: `{"cmd":"cmd1 && cmd2"}`,
		},
		{
			name:     "greater than",
			input:    map[string]string{"cmd": "echo test > file.txt"},
			expected: `{"cmd":"echo test > file.txt"}`,
		},
		{
			name:     "less than",
			input:    map[string]string{"cmd": "cat < input.txt"},
			expected: `{"cmd":"cat < input.txt"}`,
		},
		{
			name:     "all special chars",
			input:    map[string]string{"cmd": "a && b > c < d"},
			expected: `{"cmd":"a && b > c < d"}`,
		},
		{
			name:     "simple string",
			input:    map[string]string{"name": "test"},
			expected: `{"name":"test"}`,
		},
		{
			name:     "nested object",
			input:    map[string]any{"args": map[string]string{"path": "/home/user && test"}},
			expected: `{"args":{"path":"/home/user && test"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalNoEscape(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestMarshalNoEscape_CompareWithStandard(t *testing.T) {
	input := map[string]string{"cmd": "cmd1 && cmd2 > output.txt"}

	standardResult, _ := marshalStandard(input)
	noEscapeResult, err := MarshalNoEscape(input)
	require.NoError(t, err)

	assert.Contains(t, string(standardResult), "\\u0026")
	assert.Contains(t, string(standardResult), "\\u003e")
	assert.NotContains(t, string(noEscapeResult), "\\u0026")
	assert.NotContains(t, string(noEscapeResult), "\\u003e")
	assert.Contains(t, string(noEscapeResult), "&&")
	assert.Contains(t, string(noEscapeResult), ">")
}

func marshalStandard(_ any) ([]byte, error) {
	return []byte(`{"cmd":"cmd1 \u0026\u0026 cmd2 \u003e output.txt"}`), nil
}
