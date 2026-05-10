package sshd

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	gssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/codecollab-co/forge/forge-platform/internal/git"
	"github.com/codecollab-co/forge/forge-platform/internal/permissions"
	"github.com/codecollab-co/forge/forge-platform/internal/repos"
	"github.com/codecollab-co/forge/forge-platform/internal/sshkeys"
)

// Server hosts the git-over-SSH transport. Run blocks until the listener
// stops; spin it up in its own goroutine.
type Server struct {
	Addr        string         // e.g. ":2222"
	HostKeyPath string         // OpenSSH-format private key on disk; auto-generated if missing
	Repos       *repos.Store
	Keys        *sshkeys.Store
	GitStorage  *git.Repository
}

func (s *Server) Run(ctx context.Context) error {
	if err := ensureHostKey(s.HostKeyPath); err != nil {
		return err
	}
	srv := &gssh.Server{
		Addr: s.Addr,
		Handler: func(sess gssh.Session) {
			s.handle(sess)
		},
		PublicKeyHandler: s.publicKeyHandler,
	}
	if err := srv.SetOption(gssh.HostKeyFile(s.HostKeyPath)); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	log.Printf("sshd listening on %s", s.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, gssh.ErrServerClosed) {
		return err
	}
	return nil
}

// publicKeyHandler is consulted on every public-key auth attempt. We
// resolve the offered key's fingerprint to a user; if found, we store
// the user on the connection context for the request handler.
func (s *Server) publicKeyHandler(ctx gssh.Context, key gssh.PublicKey) bool {
	fp := ssh.FingerprintSHA256(key)
	user, err := s.Keys.ByFingerprint(ctx, fp)
	if err != nil || user == nil {
		return false
	}
	ctx.SetValue("forge.user.id", user.ID)
	ctx.SetValue("forge.user.handle", user.Handle)
	ctx.SetValue("forge.user.email", user.Email)
	return true
}

func (s *Server) handle(sess gssh.Session) {
	rawCmd := sess.RawCommand()
	if rawCmd == "" {
		_, _ = io.WriteString(sess.Stderr(), "Forge SSH only supports git-upload-pack / git-receive-pack.\n")
		_ = sess.Exit(1)
		return
	}
	gc, err := ParseGitCommand(rawCmd)
	if err != nil {
		_, _ = io.WriteString(sess.Stderr(), err.Error()+"\n")
		_ = sess.Exit(1)
		return
	}

	ctx := sess.Context()
	userID, _ := ctx.Value("forge.user.id").(string)
	if userID == "" {
		_, _ = io.WriteString(sess.Stderr(), "unauthenticated\n")
		_ = sess.Exit(1)
		return
	}

	repo, err := s.Repos.GetByOwnerHandleAndName(ctx, gc.Owner, gc.Name)
	if err != nil {
		_, _ = io.WriteString(sess.Stderr(), "repository not found\n")
		_ = sess.Exit(1)
		return
	}

	action := permissions.ActionRead
	if gc.Write {
		action = permissions.ActionPush
	}
	if !permissions.Allow(permissions.Actor{UserID: userID},
		permissions.Repo{OwnerID: repo.OwnerID, Visibility: repo.Visibility},
		action,
	) {
		// Same hide-existence policy as HTTPS: private => "not found".
		_, _ = io.WriteString(sess.Stderr(), "repository not found\n")
		_ = sess.Exit(1)
		return
	}

	repoPath := s.GitStorage.Path(repo.OwnerHandle, repo.Name)
	gitBin, err := exec.LookPath("git")
	if err != nil {
		_, _ = io.WriteString(sess.Stderr(), "git not installed on server\n")
		_ = sess.Exit(1)
		return
	}
	cmd := exec.CommandContext(ctx, gitBin, gc.Op[len("git-"):], repoPath)
	cmd.Stdin = sess
	cmd.Stdout = sess
	cmd.Stderr = sess.Stderr()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = sess.Exit(exitErr.ExitCode())
			return
		}
		_, _ = io.WriteString(sess.Stderr(), err.Error()+"\n")
		_ = sess.Exit(1)
		return
	}
	_ = sess.Exit(0)
}

// ensureHostKey writes a fresh ed25519 host key to path on first start.
// Pure Go (no shell-out / no openssh package needed).
// Stable across restarts only if path lives in a persistent volume; for
// dev we accept regeneration and the resulting "host key changed" warning.
func ensureHostKey(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "forge-sshd")
	if err != nil {
		return err
	}
	return os.WriteFile(path, pem.EncodeToMemory(pemBlock), 0o600)
}
