// Package sshd hosts the git-over-SSH server (slice 11).
//
// command.go is the pure command parser; server.go wires it to a real
// SSH listener and execs git-upload-pack / git-receive-pack.
package sshd

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnsupportedCommand = errors.New("only git-upload-pack and git-receive-pack are allowed")
	ErrInvalidPath        = errors.New("invalid repository path")
)

type GitCommand struct {
	Op    string // "git-upload-pack" | "git-receive-pack"
	Owner string
	Name  string
	Write bool // true for receive-pack
}

// ParseGitCommand handles the SSH original command, e.g.
//   git-upload-pack 'alice/foo.git'
//   git-receive-pack '/alice/foo.git'
func ParseGitCommand(cmd string) (GitCommand, error) {
	cmd = strings.TrimSpace(cmd)
	var op string
	switch {
	case strings.HasPrefix(cmd, "git-upload-pack "):
		op = "git-upload-pack"
	case strings.HasPrefix(cmd, "git-receive-pack "):
		op = "git-receive-pack"
	default:
		return GitCommand{}, ErrUnsupportedCommand
	}
	rest := strings.TrimSpace(strings.TrimPrefix(cmd, op))
	rest = strings.Trim(rest, "'\"")
	rest = strings.TrimPrefix(rest, "/")

	if !strings.HasSuffix(rest, ".git") {
		return GitCommand{}, fmt.Errorf("%w: missing .git suffix", ErrInvalidPath)
	}
	rest = strings.TrimSuffix(rest, ".git")

	if strings.Contains(rest, "..") {
		return GitCommand{}, fmt.Errorf("%w: contains ..", ErrInvalidPath)
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return GitCommand{}, fmt.Errorf("%w: expected <owner>/<name>", ErrInvalidPath)
	}
	return GitCommand{
		Op:    op,
		Owner: parts[0],
		Name:  parts[1],
		Write: op == "git-receive-pack",
	}, nil
}
