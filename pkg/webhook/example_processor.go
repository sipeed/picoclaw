package webhook

import (
	"context"
	"fmt"
	"time"
)

// ExampleProcessor demonstrates how to implement a custom processor function
func ExampleProcessor(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	// Simulate some processing work
	select {
	case <-time.After(2 * time.Second):
		// Processing completed
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Extract data from payload and perform processing
	inputData, ok := payload["data"]
	if !ok {
		return nil, fmt.Errorf("missing 'data' field in payload")
	}

	// Return processed result
	result := map[string]interface{}{
		"processed_data": inputData,
		"processed_at":   time.Now().Format(time.RFC3339),
		"message":        "Processing completed successfully",
	}

	return result, nil
}

// CreateDefaultProcessor creates a processor with the example processing function
func CreateDefaultProcessor() *Processor {
	return NewProcessor(ExampleProcessor)
}
