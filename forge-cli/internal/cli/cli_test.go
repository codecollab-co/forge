package cli_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/codecollab-co/forge/forge-cli/internal/cli"
)

func TestVersionFlagPrintsBuildVersion(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run([]string{"--version"}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "forge") {
		t.Errorf("expected output to contain 'forge', got %q", got)
	}
}

func TestNoArgsPrintsHelpWithoutError(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(nil, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Forge CLI") {
		t.Errorf("expected help text containing 'Forge CLI', got %q", stdout.String())
	}
}
