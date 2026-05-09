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

// CreateBranch points refs/heads/<name> at the commit referenced by `fromRef`.
// Errors if `name` already exists or `fromRef` doesn't resolve.
func (r *Repository) CreateBranch(ctx context.Context, owner, name, branch, fromRef string) error {
	if !r.Exists(owner, name) {
		return ErrNotFound
	}
	if err := ValidateOwnerOrName(branch); err != nil {
		return fmt.Errorf("branch: %w", err)
	}
	repoPath := r.Path(owner, name)
	if oid, _ := r.BranchOID(ctx, owner, name, branch); oid != "" {
		return errors.New("branch already exists")
	}
	revOut, err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--verify", fromRef+"^{commit}").Output()
	if err != nil {
		return fmt.Errorf("from %q: %w", fromRef, err)
	}
	fromOID := strings.TrimSpace(string(revOut))
	if out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "update-ref", "refs/heads/"+branch, fromOID).CombinedOutput(); err != nil {
		return fmt.Errorf("update-ref: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DeleteBranch removes refs/heads/<branch>. Refuses to delete the symbolic
// HEAD target so the default branch can't accidentally be removed.
func (r *Repository) DeleteBranch(ctx context.Context, owner, name, branch string) error {
	if !r.Exists(owner, name) {
		return ErrNotFound
	}
	defaultBranch, _ := r.DefaultBranch(ctx, owner, name)
	if branch == defaultBranch {
		return errors.New("cannot delete the default branch")
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Path(owner, name), "update-ref", "-d", "refs/heads/"+branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("update-ref -d: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
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
	headOID, err := r.BranchOID(ctx, owner, name, head)
	if err != nil {
		return "", err
	}
	if headOID == "" {
		return "", fmt.Errorf("head branch %q does not exist", head)
	}

	// Fast-forward case: base has no commits yet (e.g., default branch on a
	// freshly-initialised bare repo). Point base at head and call it a merge.
	if baseOID == "" {
		updArgs := []string{"-C", repoPath, "update-ref", "refs/heads/" + base, headOID}
		if out, err := exec.CommandContext(ctx, "git", updArgs...).CombinedOutput(); err != nil {
			return "", fmt.Errorf("update-ref (fast-forward): %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return headOID, nil
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

// MoveOwner relocates every repository under `oldOwner/` on disk to
// `newOwner/`. Called when a user renames their handle.
func (r *Repository) MoveOwner(ctx context.Context, oldOwner, newOwner string) error {
	if err := ValidateOwnerOrName(oldOwner); err != nil {
		return err
	}
	if err := ValidateOwnerOrName(newOwner); err != nil {
		return err
	}
	oldPath := filepath.Join(r.root, oldOwner)
	newPath := filepath.Join(r.root, newOwner)
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(newPath); err == nil {
		return errors.New("destination owner directory already exists")
	}
	return os.Rename(oldPath, newPath)
}

// RenameRepo moves /<owner>/<oldName>.git to /<owner>/<newName>.git.
func (r *Repository) RenameRepo(ctx context.Context, owner, oldName, newName string) error {
	if err := ValidateOwnerOrName(owner); err != nil {
		return err
	}
	if err := ValidateOwnerOrName(newName); err != nil {
		return err
	}
	oldPath := r.Path(owner, oldName)
	newPath := r.Path(owner, newName)
	if _, err := os.Stat(newPath); err == nil {
		return errors.New("destination repo already exists")
	}
	return os.Rename(oldPath, newPath)
}

// DeleteRepo removes the on-disk repo storage. Idempotent.
func (r *Repository) DeleteRepo(ctx context.Context, owner, name string) error {
	if err := ValidateOwnerOrName(owner); err != nil {
		return err
	}
	if err := ValidateOwnerOrName(name); err != nil {
		return err
	}
	return os.RemoveAll(r.Path(owner, name))
}

// CloneFromURL mirrors a remote repository into the on-disk bare layout.
// Used by "Import from Git URL". Times out after 5 minutes.
func (r *Repository) CloneFromURL(ctx context.Context, owner, name, sourceURL string) error {
	if err := ValidateOwnerOrName(owner); err != nil {
		return err
	}
	if err := ValidateOwnerOrName(name); err != nil {
		return err
	}
	path := r.Path(owner, name)
	if _, err := os.Stat(path); err == nil {
		return ErrAlreadyExists
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "clone", "--bare", "--", sourceURL, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(path)
		return fmt.Errorf("git clone --bare: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	cfg := exec.CommandContext(ctx, "git", "-C", path, "config", "http.receivepack", "true")
	if out, err := cfg.CombinedOutput(); err != nil {
		return fmt.Errorf("git config: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// FileChange describes a single file write inside a CreateCommit.
type FileChange struct {
	Path    string // forward-slash relative path
	Content []byte // full file contents
	Mode    string // "100644" if zero
}

// CreateCommit writes one or more files on top of the named branch (or
// creates the branch from baseBranch if it doesn't exist) and produces a
// single commit. Implemented entirely with plumbing commands so it works on
// a bare repo without a worktree:
//
//   1. hash-object --stdin for each file's content -> blob OIDs
//   2. start from base tree (read-tree), update-index for each path
//   3. write-tree -> tree OID
//   4. commit-tree with parent baseOID -> commit OID
//   5. update-ref refs/heads/<branch> -> commit OID
//
// We use a per-call temporary GIT_INDEX_FILE so concurrent calls don't stomp
// on each other's index.
func (r *Repository) CreateCommit(
	ctx context.Context,
	owner, name, branch, baseBranch string,
	files []FileChange,
	author Identity,
	message string,
) (string, error) {
	if !r.Exists(owner, name) {
		return "", ErrNotFound
	}
	if len(files) == 0 {
		return "", errors.New("at least one file is required")
	}
	repoPath := r.Path(owner, name)

	// Resolve parent OID. If the branch already exists, use it; otherwise
	// fall back to baseBranch.
	parentOID, _ := r.BranchOID(ctx, owner, name, branch)
	if parentOID == "" && baseBranch != "" {
		parentOID, _ = r.BranchOID(ctx, owner, name, baseBranch)
	}

	tmpIndex, err := os.CreateTemp("", "forge-index-")
	if err != nil {
		return "", err
	}
	tmpIndexPath := tmpIndex.Name()
	_ = tmpIndex.Close()
	defer os.Remove(tmpIndexPath)
	// `read-tree` requires the index file to not pre-exist.
	_ = os.Remove(tmpIndexPath)

	env := append(os.Environ(), "GIT_INDEX_FILE="+tmpIndexPath)

	if parentOID != "" {
		readTree := exec.CommandContext(ctx, "git", "-C", repoPath, "read-tree", parentOID)
		readTree.Env = env
		if out, err := readTree.CombinedOutput(); err != nil {
			return "", fmt.Errorf("read-tree: %w (%s)", err, strings.TrimSpace(string(out)))
		}
	}

	for _, f := range files {
		hash := exec.CommandContext(ctx, "git", "-C", repoPath, "hash-object", "-w", "--stdin")
		hash.Env = env
		hash.Stdin = strings.NewReader(string(f.Content))
		oidOut, err := hash.Output()
		if err != nil {
			return "", fmt.Errorf("hash-object %s: %w", f.Path, err)
		}
		blobOID := strings.TrimSpace(string(oidOut))
		mode := f.Mode
		if mode == "" {
			mode = "100644"
		}
		upd := exec.CommandContext(ctx, "git", "-C", repoPath,
			"update-index", "--add", "--cacheinfo", mode+","+blobOID+","+f.Path)
		upd.Env = env
		if out, err := upd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("update-index %s: %w (%s)", f.Path, err, strings.TrimSpace(string(out)))
		}
	}

	writeTree := exec.CommandContext(ctx, "git", "-C", repoPath, "write-tree")
	writeTree.Env = env
	treeOut, err := writeTree.Output()
	if err != nil {
		return "", fmt.Errorf("write-tree: %w", err)
	}
	treeOID := strings.TrimSpace(string(treeOut))

	commitArgs := []string{"-C", repoPath, "commit-tree", treeOID, "-m", message}
	if parentOID != "" {
		commitArgs = append(commitArgs, "-p", parentOID)
	}
	commit := exec.CommandContext(ctx, "git", commitArgs...)
	commit.Env = append(env,
		"GIT_AUTHOR_NAME="+author.Name,
		"GIT_AUTHOR_EMAIL="+author.Email,
		"GIT_COMMITTER_NAME="+author.Name,
		"GIT_COMMITTER_EMAIL="+author.Email,
	)
	commitOut, err := commit.Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitOID := strings.TrimSpace(string(commitOut))

	updArgs := []string{"-C", repoPath, "update-ref", "refs/heads/" + branch, commitOID}
	if existing, _ := r.BranchOID(ctx, owner, name, branch); existing != "" {
		updArgs = append(updArgs, existing)
	}
	updRef := exec.CommandContext(ctx, "git", updArgs...)
	if out, err := updRef.CombinedOutput(); err != nil {
		return "", fmt.Errorf("update-ref: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return commitOID, nil
}
