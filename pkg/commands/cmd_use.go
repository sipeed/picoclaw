package commands

func useCommand() Definition {
	return Definition{
		Name:        "use",
		Description: "Force a specific installed skill for one request",
		Usage:       "/use <skill> [message]",
	}
}
