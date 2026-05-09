// Package githttp serves the smart-HTTP Git protocol by exec'ing
// `git http-backend` as a CGI handler.
//
// Read (clone, fetch): allowed for public repos without auth.
// Write (push): HTTP Basic auth using the user's git secret (slice 4);
// proper named/revocable PATs land in slice 12. Authorization is delegated
// to permissions.PermissionChecker.
package githttp

import (
	"errors"
	"net/http"
	"net/http/cgi"
	"os/exec"
	"strings"

	"github.com/codecollab-co/forge/forge-platform/internal/git"
	"github.com/codecollab-co/forge/forge-platform/internal/permissions"
	"github.com/codecollab-co/forge/forge-platform/internal/repos"
	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

type Handler struct {
	Repos      *repos.Store
	Users      *users.Repo
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

	write := isWriteOp(sub, r.URL.RawQuery)

	actor, ok := h.authenticate(r)
	if !ok && (write || repo.Visibility == "private") {
		w.Header().Set("WWW-Authenticate", `Basic realm="forge"`)
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	resource := permissions.Repo{OwnerID: repo.OwnerID, Visibility: repo.Visibility}
	action := permissions.ActionRead
	if write {
		action = permissions.ActionPush
	}
	if !permissions.Allow(actor, resource, action) {
		// 404 on private repos to avoid leaking existence; 403 otherwise.
		if repo.Visibility == "private" && (actor.IsAnonymous || actor.UserID != repo.OwnerID) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
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
			"REMOTE_USER=" + actor.UserID,
		},
	}
	cgiHandler.ServeHTTP(w, r)
}

// authenticate parses HTTP Basic credentials. Returns (actor, ok). When the
// header is absent or invalid, ok=false (anonymous).
func (h *Handler) authenticate(r *http.Request) (permissions.Actor, bool) {
	handle, secret, hasAuth := r.BasicAuth()
	if !hasAuth || handle == "" || secret == "" {
		return permissions.Actor{IsAnonymous: true}, false
	}
	user, err := h.Users.VerifyGitSecret(r.Context(), handle, secret)
	if err != nil || user == nil {
		return permissions.Actor{IsAnonymous: true}, false
	}
	return permissions.Actor{UserID: user.ID}, true
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
