package tools

import (
	"context"
	"fmt"

	"github.com/Knetic/govaluate"
)

// CalculatorTool evaluates mathematical expressions.
type CalculatorTool struct{}

// NewCalculatorTool creates a new CalculatorTool.
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "Evaluates mathematical expressions. Input should be a mathematical expression as a string."
}

func (t *CalculatorTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The mathematical expression to evaluate (e.g., '2 + 2', '10 * (5 - 3)').",
			},
		},
		"required": []string{"expression"},
	}
}

func (t *CalculatorTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	exprStr, ok := args["expression"].(string)
	if !ok || exprStr == "" {
		return ErrorResult("expression argument is required and must be a string")
	}

	expression, err := govaluate.NewEvaluableExpression(exprStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid expression: %v", err))
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("evaluation failed: %v", err))
	}

	// Format result to handle floats properly if needed
	resStr := fmt.Sprintf("%v", result)
	return UserResult(resStr)
}

func (t *CalculatorTool) RequiresApproval() bool {
	return false
}
