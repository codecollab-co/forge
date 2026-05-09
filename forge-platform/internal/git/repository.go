// Package git is the GitRepository deep module from ADR-0007.
//
// All on-disk Git operations flow through this surface. The intent is that
// the EBS-backed bare repo storage today and a future S3-backed
// implementation (ADR-0004) sit behind the same interface.
package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrAlreadyExists = errors.New("repository already exists")
	ErrNotFound      = errors.New("repository not found")
	ErrInvalidName   = errors.New("invalid name")
)

var nameRE = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$`)

// ValidateOwnerOrName matches the same shape used for handles in the users
// repo: lowercase, alphanumeric + dash, 2-40 chars, no leading/trailing dash.
func ValidateOwnerOrName(s string) error {
	if !nameRE.MatchString(s) {
		return ErrInvalidName
	}
	return nil
}

type Repository struct {
	root string
}

func New(root string) (*Repository, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Repository{root: root}, nil
}

func (r *Repository) Path(owner, name string) string {
	return filepath.Join(r.root, owner, name+".git")
}

func (r *Repository) Exists(owner, name string) bool {
	if err := ValidateOwnerOrName(owner); err != nil {
		return false
	}
	if err := ValidateOwnerOrName(name); err != nil {
		return false
	}
	info, err := os.Stat(r.Path(owner, name))
	return err == nil && info.IsDir()
}

func (r *Repository) Init(ctx context.Context, owner, name string) error {
	if err := ValidateOwnerOrName(owner); err != nil {
		return fmt.Errorf("owner: %w", err)
	}
	if err := ValidateOwnerOrName(name); err != nil {
		return fmt.Errorf("name: %w", err)
	}
	path := r.Path(owner, name)
	if _, err := os.Stat(path); err == nil {
		return ErrAlreadyExists
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "init", "--bare", "--initial-branch=main", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(path)
		return fmt.Errorf("git init: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	// Allow git http-backend to serve this repo without explicit env config.
	cfg := exec.CommandContext(ctx, "git", "-C", path, "config", "http.receivepack", "true")
	if out, err := cfg.CombinedOutput(); err != nil {
		return fmt.Errorf("git config: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type Ref struct {
	Name string
	OID  string
}

func (r *Repository) ListRefs(ctx context.Context, owner, name string) ([]Ref, error) {
	if !r.Exists(owner, name) {
		return nil, ErrNotFound
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "for-each-ref", "--format=%(objectname) %(refname)")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		refs = append(refs, Ref{OID: parts[0], Name: parts[1]})
	}
	return refs, nil
}

// DefaultBranch returns the short branch name HEAD points at.
// Empty string + nil error if the repo has no commits yet.
func (r *Repository) DefaultBranch(ctx context.Context, owner, name string) (string, error) {
	if !r.Exists(owner, name) {
		return "", ErrNotFound
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "symbolic-ref", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repository) ListBranches(ctx context.Context, owner, name string) ([]string, error) {
	if !r.Exists(owner, name) {
		return nil, ErrNotFound
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

type TreeEntry struct {
	Mode string
	Type string // "blob" or "tree"
	OID  string
	Path string
}

// ReadTree returns the entries directly under `dir` at the given ref. Use
// dir == "" for the repository root.
func (r *Repository) ReadTree(ctx context.Context, owner, name, ref, dir string) ([]TreeEntry, error) {
	if !r.Exists(owner, name) {
		return nil, ErrNotFound
	}
	if ref == "" {
		ref = "HEAD"
	}
	target := ref + ":" + dir
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "ls-tree", "--full-name", "-z", target)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // no commits yet, or path doesn't exist
	}
	var entries []TreeEntry
	for _, raw := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if raw == "" {
			continue
		}
		// "<mode> <type> <oid>\t<path>"
		tab := strings.IndexByte(raw, '\t')
		if tab < 0 {
			continue
		}
		head, path := raw[:tab], raw[tab+1:]
		fields := strings.Fields(head)
		if len(fields) != 3 {
			continue
		}
		entries = append(entries, TreeEntry{Mode: fields[0], Type: fields[1], OID: fields[2], Path: path})
	}
	return entries, nil
}

// ReadBlob returns the contents of `path` at the given ref. Returns nil
// + nil error if the path doesn't exist.
func (r *Repository) ReadBlob(ctx context.Context, owner, name, ref, path string) ([]byte, error) {
	if !r.Exists(owner, name) {
		return nil, ErrNotFound
	}
	if ref == "" {
		ref = "HEAD"
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "cat-file", "-p", ref+":"+path)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	return out, nil
}

// BranchOID resolves a branch name to its current commit OID.
// Returns "" + nil if the branch does not exist.
func (r *Repository) BranchOID(ctx context.Context, owner, name, branch string) (string, error) {
	if !r.Exists(owner, name) {
		return "", ErrNotFound
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "rev-parse", "--verify", "refs/heads/"+branch)
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// Diff returns the unified diff that would result from merging head into base
// (i.e. what changes head introduces relative to the merge base).
func (r *Repository) Diff(ctx context.Context, owner, name, base, head string) ([]byte, error) {
	if !r.Exists(owner, name) {
		return nil, ErrNotFound
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "diff", base+"..."+head)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return out, nil
}

type Identity struct {
	Name  string
	Email string
}

// MergeBranches creates a merge commit on `base` that brings in `head`.
// Implemented with merge-tree (Git 2.38+) so it works on a bare repo without
// a worktree. Returns the merge commit OID.
//
// Returns ErrMergeConflict if the merge cannot be performed cleanly. The
// caller should report this back to the user; we do not attempt resolution.
func (r *Repository) MergeBranches(
	ctx context.Context,
	owner, name, base, head, message string,
	merger Identity,
) (string, error) {
	if !r.Exists(owner, name) {
		return "", ErrNotFound
	}
	repoPath := r.Path(owner, name)

	baseOID, err := r.BranchOID(ctx, owner, name, base)
	if err != nil {
		return "", err
	}
	if baseOID == "" {
		return "", fmt.Errorf("base branch %q does not exist", base)
	}
	headOID, err := r.BranchOID(ctx, owner, name, head)
	if err != nil {
		return "", err
	}
	if headOID == "" {
		return "", fmt.Errorf("head branch %q does not exist", head)
	}

	mergeTreeCmd := exec.CommandContext(ctx, "git", "-C", repoPath,
		"merge-tree", "--write-tree", "--no-messages", baseOID, headOID)
	treeOut, err := mergeTreeCmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrMergeConflict
		}
		return "", fmt.Errorf("merge-tree: %w", err)
	}
	treeOID := strings.TrimSpace(string(treeOut))
	if treeOID == "" {
		return "", ErrMergeConflict
	}

	commitCmd := exec.CommandContext(ctx, "git", "-C", repoPath,
		"commit-tree", treeOID, "-p", baseOID, "-p", headOID, "-m", message)
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+merger.Name,
		"GIT_AUTHOR_EMAIL="+merger.Email,
		"GIT_COMMITTER_NAME="+merger.Name,
		"GIT_COMMITTER_EMAIL="+merger.Email,
	)
	commitOut, err := commitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitOID := strings.TrimSpace(string(commitOut))

	updateCmd := exec.CommandContext(ctx, "git", "-C", repoPath,
		"update-ref", "refs/heads/"+base, commitOID, baseOID)
	if out, err := updateCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("update-ref: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return commitOID, nil
}

var ErrMergeConflict = errors.New("merge conflict")
