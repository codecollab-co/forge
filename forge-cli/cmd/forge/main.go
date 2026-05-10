package main

import (
	"fmt"
	"os"

	"github.com/codecollab-co/forge/forge-cli/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "forge:", err)
		os.Exit(1)
	}
}
