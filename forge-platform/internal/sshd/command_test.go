package sshd_test

import (
	"errors"
	"testing"

	"github.com/codecollab-co/forge/forge-platform/internal/sshd"
)

func TestParseGitCommand_UploadPack(t *testing.T) {
	got, err := sshd.ParseGitCommand("git-upload-pack 'alice/foo.git'")
	if err != nil {
		t.Fatal(err)
	}
	if got.Op != "git-upload-pack" || got.Owner != "alice" || got.Name != "foo" || got.Write {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseGitCommand_ReceivePackIsWrite(t *testing.T) {
	got, err := sshd.ParseGitCommand("git-receive-pack 'alice/foo.git'")
	if err != nil {
		t.Fatal(err)
	}
	if got.Op != "git-receive-pack" || !got.Write {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseGitCommand_LeadingSlash(t *testing.T) {
	got, err := sshd.ParseGitCommand("git-upload-pack '/alice/foo.git'")
	if err != nil {
		t.Fatal(err)
	}
	if got.Owner != "alice" || got.Name != "foo" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseGitCommand_NoQuotes(t *testing.T) {
	got, err := sshd.ParseGitCommand("git-upload-pack alice/foo.git")
	if err != nil {
		t.Fatal(err)
	}
	if got.Owner != "alice" || got.Name != "foo" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseGitCommand_RejectsUnknownCommand(t *testing.T) {
	_, err := sshd.ParseGitCommand("ls -la")
	if !errors.Is(err, sshd.ErrUnsupportedCommand) {
		t.Fatalf("expected ErrUnsupportedCommand, got %v", err)
	}
}

func TestParseGitCommand_RejectsTraversal(t *testing.T) {
	for _, c := range []string{
		"git-upload-pack '../etc/passwd.git'",
		"git-upload-pack 'a/../../b.git'",
		"git-upload-pack 'evil'",          // no slash, no .git
		"git-upload-pack 'too/many/parts.git'",
	} {
		_, err := sshd.ParseGitCommand(c)
		if err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}
