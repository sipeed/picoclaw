package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// NewCLIPermissionFunc creates a PermissionFunc that prompts the user on a terminal.
func NewCLIPermissionFunc(reader io.Reader, writer io.Writer) PermissionFunc {
	scanner := bufio.NewScanner(reader)
	return func(ctx context.Context, path string) (bool, error) {
		fmt.Fprintf(writer, "\n⚠ Agent wants to access: %s\nAllow access to this directory? [y/N]: ", path)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return false, err
			}
			return false, nil
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes", nil
	}
}
