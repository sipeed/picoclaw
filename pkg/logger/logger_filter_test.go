package logger

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestSetComponentFilter(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Reset filter
	SetComponentFilter("")

	// Test 1: No filter
	InfoC("comp1", "msg1")
	if !strings.Contains(buf.String(), "msg1") {
		t.Error("Expected msg1 to be logged")
	}
	buf.Reset()

	// Test 2: Filter comp1
	SetComponentFilter("comp1")
	InfoC("comp1", "msg2") // Should be logged
	InfoC("comp2", "msg3") // Should NOT be logged

	output := buf.String()
	if !strings.Contains(output, "msg2") {
		t.Error("Expected msg2 to be logged")
	}
	if strings.Contains(output, "msg3") {
		t.Error("Expected msg3 NOT to be logged")
	}
	buf.Reset()

	// Test 3: Multiple filters
	SetComponentFilter("comp1,comp2")
	InfoC("comp1", "msg4") // Logged
	InfoC("comp2", "msg5") // Logged
	InfoC("comp3", "msg6") // Not logged

	output = buf.String()
	if !strings.Contains(output, "msg4") {
		t.Error("Expected msg4 to be logged")
	}
	if !strings.Contains(output, "msg5") {
		t.Error("Expected msg5 to be logged")
	}
	if strings.Contains(output, "msg6") {
		t.Error("Expected msg6 NOT to be logged")
	}

	// Reset filter at end
	SetComponentFilter("")
}
