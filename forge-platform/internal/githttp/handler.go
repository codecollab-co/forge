// Package githttp serves the smart-HTTP Git protocol by exec'ing
// `git http-backend` as a CGI handler. Slice 3: clone (read) only.
// Slice 4 will require auth on receive-pack (push).
package githttp

import (
	"errors"
	"net/http"
	"net/http/cgi"
	"os/exec"
	"strings"

	"github.com/codecollab-co/forge/forge-platform/internal/git"
	"github.com/codecollab-co/forge/forge-platform/internal/repos"
)

type Handler struct {
	Repos      *repos.Store
	GitStorage *git.Repository
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner, name, sub, ok := splitGitPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := git.ValidateOwnerOrName(owner); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := git.ValidateOwnerOrName(name); err != nil {
		http.NotFound(w, r)
		return
	}

	repo, err := h.Repos.GetByOwnerHandleAndName(r.Context(), owner, name)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Slice 3 only allows read; push lands in slice 4 with proper auth.
	if isWriteOp(sub, r.URL.RawQuery) {
		http.Error(w, "push not yet supported (slice 4)", http.StatusForbidden)
		return
	}

	gitBin, err := exec.LookPath("git")
	if err != nil {
		http.Error(w, "git not installed", http.StatusInternalServerError)
		return
	}

	cgiHandler := &cgi.Handler{
		Path: gitBin,
		Args: []string{"http-backend"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + h.GitStorage.Path(repo.OwnerHandle, repo.Name),
			"GIT_HTTP_EXPORT_ALL=1",
			"PATH_INFO=" + sub,
		},
	}
	cgiHandler.ServeHTTP(w, r)
}

func splitGitPath(p string) (owner, name, sub string, ok bool) {
	p = strings.TrimPrefix(p, "/")
	idx := strings.Index(p, ".git")
	if idx < 0 {
		return "", "", "", false
	}
	left, right := p[:idx], p[idx+len(".git"):]
	parts := strings.SplitN(left, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", false
	}
	if right == "" {
		right = "/"
	}
	return parts[0], parts[1], right, true
}

func isWriteOp(sub, rawQuery string) bool {
	if strings.HasSuffix(sub, "/git-receive-pack") {
		return true
	}
	if strings.HasSuffix(sub, "/info/refs") && strings.Contains(rawQuery, "service=git-receive-pack") {
		return true
	}
	return false
}
