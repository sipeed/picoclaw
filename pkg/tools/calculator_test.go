package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatorTool_Execute(t *testing.T) {
	calc := NewCalculatorTool()

	tests := []struct {
		name       string
		expression string
		want       string
		wantError  bool
	}{
		{
			name:       "simple addition",
			expression: "2 + 2",
			want:       "4",
		},
		{
			name:       "complex math",
			expression: "10 * (5 - 3)",
			want:       "20",
		},
		{
			name:       "floating point result",
			expression: "10 / 4",
			want:       "2.5",
		},
		{
			name:       "missing expression",
			expression: "",
			wantError:  true,
		},
		{
			name:       "invalid expression",
			expression: "2 + * 2",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{}
			if tt.expression != "" {
				args["expression"] = tt.expression
			}

			result := calc.Execute(context.Background(), args)

			if tt.wantError {
				assert.True(t, result.IsError)
			} else {
				assert.False(t, result.IsError)
				assert.Equal(t, tt.want, result.ForLLM)
			}
		})
	}
}

func TestCalculatorTool_Info(t *testing.T) {
	calc := NewCalculatorTool()
	assert.Equal(t, "calculator", calc.Name())
	assert.NotEmpty(t, calc.Description())
	assert.NotNil(t, calc.Parameters())
}
