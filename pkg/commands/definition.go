package commands

type Definition struct {
	Name        string
	Description string
	Usage       string
	Aliases     []string
	Channels    []string
	Handler     Handler
}
