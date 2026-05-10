// Package cli is the black-box entry point for the forge CLI.
// All tests call Run(); the cmd/forge main is just os.Exit(cli.Run(os.Args[1:]...)).
package cli

import (
	"io"

	"github.com/spf13/cobra"
)

// Version is overridden at build time via -ldflags="-X .../cli.Version=v0.1.0"
var Version = "dev"

// Run executes the forge CLI with the supplied args, writing output to the
// provided streams. Returns nil on success or a non-nil error on failure.
// Designed for black-box testing — callers can capture stdout/stderr.
func Run(args []string, stdout, stderr io.Writer) error {
	root := &cobra.Command{
		Use:           "forge",
		Short:         "Forge CLI — interact with Forge from your terminal",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	return root.Execute()
}
