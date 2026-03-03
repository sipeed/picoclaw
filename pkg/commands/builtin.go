package commands

// BuiltinDefinitions returns all built-in command definitions.
// Each command group is defined in its own cmd_*.go file.
func BuiltinDefinitions(deps *Deps) []Definition {
	return []Definition{
		startCommand(),
		helpCommand(deps),
		showCommand(deps),
		listCommand(deps),
	}
}
