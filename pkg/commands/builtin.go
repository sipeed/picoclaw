package commands

// BuiltinDefinitions returns all built-in command definitions.
// Each command group is defined in its own cmd_*.go file.
// Definitions are stateless — runtime dependencies are provided
// via the Runtime parameter passed to handlers at execution time.
func BuiltinDefinitions() []Definition {
	return []Definition{
		startCommand(),
		helpCommand(),
		showCommand(),
		listCommand(),
		useCommand(),
		btwCommand(),
		switchCommand(),
		checkCommand(),
		clearCommand(),
		subagentsCommand(),
		reloadCommand(),
	}
}
