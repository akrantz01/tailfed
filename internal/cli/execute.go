package cli

// Executable is a command that can be run with some pre-provided arguments/options
type Executable interface {
	Execute(args []string)
}

// Execute runs the root command
func Execute(exit func(int), args []string) {
	NewRoot(exit).Execute(args)
}
