// Package cli is the black-box entry point for the forge CLI.
// All tests call Run(); the cmd/forge main is just os.Exit(cli.Run(os.Args[1:]...)).
package cli

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/codecollab-co/forge/forge-cli/internal/auth"
	authcmd "github.com/codecollab-co/forge/forge-cli/internal/cmd/auth"
	repocmd "github.com/codecollab-co/forge/forge-cli/internal/cmd/repo"
)

// Version is overridden at build time via -ldflags="-X .../cli.Version=v0.1.0"
var Version = "dev"

// ConfigDir returns the directory where credentials.json lives. Honors
// XDG_CONFIG_HOME and FORGE_CONFIG_DIR (test escape hatch).
func ConfigDir() string {
	if v := os.Getenv("FORGE_CONFIG_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "forge")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "forge")
	}
	return ".forge"
}

// Run executes the forge CLI with the supplied args, writing output to the
// provided streams. Returns nil on success or a non-nil error on failure.
// Designed for black-box testing — callers can capture stdout/stderr.
func Run(args []string, stdout, stderr io.Writer) error {
	store := auth.NewStore(ConfigDir())

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
	root.AddCommand(authcmd.New(stdout, stderr, store))
	root.AddCommand(repocmd.New(stdout, stderr, store))
	return root.Execute()
}
