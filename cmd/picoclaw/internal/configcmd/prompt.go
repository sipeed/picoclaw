package configcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// IsTTY returns true if stdin is a terminal (interactive).
func IsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Prompt reads a line from stdin after printing the prompt label. The returned string is trimmed.
func Prompt(label string) (string, error) {
	fmt.Print(label)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return strings.TrimSpace(scanner.Text()), nil
}
