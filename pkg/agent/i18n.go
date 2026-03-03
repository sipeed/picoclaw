package agent

import "fmt"

func tldrWithToolsMessage(toolCount int, tools string, iteration int) string {
	return fmt.Sprintf("Processed %d tool(s): %s. (%d iteration%s)", toolCount, tools, iteration, pluralize(iteration))
}

func tldrNoToolsMessage(iteration int) string {
	return fmt.Sprintf("Processed your request (%d iteration%s)", iteration, pluralize(iteration))
}

func tldrWithMessage(msg string) string {
	if len(msg) <= 50 {
		return fmt.Sprintf(". Message: \"%s\"", msg)
	}
	return fmt.Sprintf(". Message: \"%s...\"", msg[:50])
}

func pluralize(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
