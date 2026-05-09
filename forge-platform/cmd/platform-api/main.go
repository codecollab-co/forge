package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/supertokens"

	"github.com/codecollab-co/forge/forge-platform/internal/auth"
	"github.com/codecollab-co/forge/forge-platform/internal/eventbus"
	gitstorage "github.com/codecollab-co/forge/forge-platform/internal/git"
	"github.com/codecollab-co/forge/forge-platform/internal/githttp"
	"github.com/codecollab-co/forge/forge-platform/internal/issues"
	"github.com/codecollab-co/forge/forge-platform/internal/permissions"
	"github.com/codecollab-co/forge/forge-platform/internal/pulls"
	"github.com/codecollab-co/forge/forge-platform/internal/repos"
	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

func main() {
	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")
	port := envOr("PORT", "8080")
	websiteDomain := envOr("WEBSITE_DOMAIN", "http://localhost:3000")
	reposDir := envOr("REPOS_DIR", "/var/lib/forge/repos")

	signer, err := auth.NewSignerFromEnv()
	if err != nil {
		log.Fatalf("auth signer: %v", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer pool.Close()
	if err := waitForDB(ctx, pool); err != nil {
		log.Fatalf("postgres ready: %v", err)
	}

	bus := eventbus.New(pool)
	usersRepo := users.NewRepo(pool)
	reposStore := repos.NewStore(pool)
	pullsStore := pulls.NewStore(pool)
	issuesStore := issues.NewStore(pool)

	gitStorage, err := gitstorage.New(reposDir)
	if err != nil {
		log.Fatalf("git storage: %v", err)
	}

	if err := auth.InitSuperTokens(func(ctx context.Context, e auth.SignInUp) error {
		_, err := usersRepo.UpsertOnSignInUp(ctx, users.SignInUpInput{
			SuperTokensID: e.SuperTokensID,
			Provider:      e.Provider,
			ExternalID:    e.ExternalID,
			Email:         e.Email,
			DisplayName:   e.DisplayName,
			AvatarURL:     e.AvatarURL,
		})
		return err
	}); err != nil {
		log.Fatalf("supertokens init: %v", err)
	}

	gitHTTP := &githttp.Handler{Repos: reposStore, Users: usersRepo, GitStorage: gitStorage}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(websiteDomain))
	r.Use(supertokensMiddleware())

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "forge-platform"})
	})

	r.Post("/internal/token", func(w http.ResponseWriter, _ *http.Request) {
		tok, err := signer.Issue("system", time.Hour)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": tok})
	})

	r.Post("/ping", func(w http.ResponseWriter, req *http.Request) {
		if err := bus.Publish(req.Context(), "ping", map[string]any{"at": time.Now().UTC()}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
	})

	r.Get("/me", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		u, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if u == nil {
			http.Error(w, "user not provisioned", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, meResponse(u))
	}))

	r.Post("/repos", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		u, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || u == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}

		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Visibility  string `json:"visibility"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		body.Name = strings.TrimSpace(strings.ToLower(body.Name))
		if err := gitstorage.ValidateOwnerOrName(body.Name); err != nil {
			http.Error(w, "invalid name", http.StatusBadRequest)
			return
		}
		if body.Visibility == "" {
			body.Visibility = "public"
		}
		if body.Visibility != "public" && body.Visibility != "private" {
			http.Error(w, "invalid visibility", http.StatusBadRequest)
			return
		}

		row, err := reposStore.Create(req.Context(), repos.CreateInput{
			OwnerID:     u.ID,
			Name:        body.Name,
			Description: body.Description,
			Visibility:  body.Visibility,
		})
		if err != nil {
			if errors.Is(err, repos.ErrAlreadyExists) {
				http.Error(w, "repository already exists", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := gitStorage.Init(req.Context(), u.Handle, body.Name); err != nil && !errors.Is(err, gitstorage.ErrAlreadyExists) {
			http.Error(w, "git init: "+err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, repoResponse(row, u.Handle))
	}))

	r.Get("/repos", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		u, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || u == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		list, err := reposStore.ListByOwnerID(req.Context(), u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, repo := range list {
			out = append(out, repoResponse(repo, repo.OwnerHandle))
		}
		writeJSON(w, http.StatusOK, out)
	}))

	r.Get("/repos/{owner}/{name}", func(w http.ResponseWriter, req *http.Request) {
		owner := chi.URLParam(req, "owner")
		name := chi.URLParam(req, "name")
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), owner, name)
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, repoResponse(repo, repo.OwnerHandle))
	})

	r.Get("/repos/{owner}/{name}/branches", func(w http.ResponseWriter, req *http.Request) {
		owner := chi.URLParam(req, "owner")
		name := chi.URLParam(req, "name")
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), owner, name)
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		def, _ := gitStorage.DefaultBranch(req.Context(), repo.OwnerHandle, repo.Name)
		branches, _ := gitStorage.ListBranches(req.Context(), repo.OwnerHandle, repo.Name)
		writeJSON(w, http.StatusOK, map[string]any{"default": def, "branches": branches})
	})

	r.Get("/repos/{owner}/{name}/tree/{ref}", func(w http.ResponseWriter, req *http.Request) {
		owner := chi.URLParam(req, "owner")
		name := chi.URLParam(req, "name")
		ref := chi.URLParam(req, "ref")
		dir := strings.TrimPrefix(req.URL.Query().Get("path"), "/")
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), owner, name)
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries, err := gitStorage.ReadTree(req.Context(), repo.OwnerHandle, repo.Name, ref, dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			out = append(out, map[string]any{
				"path": e.Path, "type": e.Type, "mode": e.Mode, "oid": e.OID,
			})
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Get("/repos/{owner}/{name}/blob/{ref}", func(w http.ResponseWriter, req *http.Request) {
		owner := chi.URLParam(req, "owner")
		name := chi.URLParam(req, "name")
		ref := chi.URLParam(req, "ref")
		path := strings.TrimPrefix(req.URL.Query().Get("path"), "/")
		if path == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), owner, name)
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		blob, err := gitStorage.ReadBlob(req.Context(), repo.OwnerHandle, repo.Name, ref, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if blob == nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(blob)
	})

	r.Get("/me/git-secret", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		u, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || u == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		info, err := usersRepo.GitSecretInfo(req.Context(), u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"exists":       info.Exists,
			"created_at":   info.CreatedAt,
			"last_used_at": info.LastUsedAt,
			"username":     u.Handle,
		})
	}))

	r.Post("/me/git-secret", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		u, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || u == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		secret, err := usersRepo.GenerateGitSecret(req.Context(), u.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"username": u.Handle, "secret": secret})
	}))

	// ---- Pull Requests --------------------------------------------------

	r.Post("/repos/{owner}/{name}/pulls", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		actor, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || actor == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		if !permissions.Allow(permissions.Actor{UserID: actor.ID},
			permissions.Repo{OwnerID: repo.OwnerID, Visibility: repo.Visibility},
			permissions.ActionRead) {
			http.NotFound(w, req)
			return
		}

		var body struct {
			Title      string `json:"title"`
			Body       string `json:"body"`
			HeadBranch string `json:"head_branch"`
			BaseBranch string `json:"base_branch"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		body.Title = strings.TrimSpace(body.Title)
		if body.Title == "" || body.HeadBranch == "" || body.BaseBranch == "" {
			http.Error(w, "title, head_branch, base_branch are required", http.StatusBadRequest)
			return
		}
		if body.HeadBranch == body.BaseBranch {
			http.Error(w, "head_branch must differ from base_branch", http.StatusBadRequest)
			return
		}
		// Both branches must exist on the server.
		if oid, _ := gitStorage.BranchOID(req.Context(), repo.OwnerHandle, repo.Name, body.HeadBranch); oid == "" {
			http.Error(w, "head_branch does not exist", http.StatusBadRequest)
			return
		}
		if oid, _ := gitStorage.BranchOID(req.Context(), repo.OwnerHandle, repo.Name, body.BaseBranch); oid == "" {
			http.Error(w, "base_branch does not exist", http.StatusBadRequest)
			return
		}

		pr, err := pullsStore.Create(req.Context(), pulls.CreateInput{
			RepoID:     repo.ID,
			AuthorID:   actor.ID,
			Title:      body.Title,
			Body:       body.Body,
			HeadBranch: body.HeadBranch,
			BaseBranch: body.BaseBranch,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_ = bus.Publish(req.Context(), "pr.opened", map[string]any{
			"pr_id":     pr.ID,
			"repo_id":   repo.ID,
			"number":    pr.Number,
			"author_id": actor.ID,
			"head":      pr.HeadBranch,
			"base":      pr.BaseBranch,
		})

		writeJSON(w, http.StatusCreated, prResponse(pr, actor.Handle))
	}))

	r.Get("/repos/{owner}/{name}/pulls", func(w http.ResponseWriter, req *http.Request) {
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		state := pulls.State(req.URL.Query().Get("state"))
		list, err := pullsStore.ListByRepo(req.Context(), repo.ID, state)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, pr := range list {
			out = append(out, prResponse(pr, derefString(pr.AuthorHandle)))
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Get("/repos/{owner}/{name}/pulls/{number}", func(w http.ResponseWriter, req *http.Request) {
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		number, err := strconv.Atoi(chi.URLParam(req, "number"))
		if err != nil {
			http.Error(w, "invalid number", http.StatusBadRequest)
			return
		}
		pr, err := pullsStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
		if err != nil {
			if errors.Is(err, pulls.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		diff, _ := gitStorage.Diff(req.Context(), repo.OwnerHandle, repo.Name, pr.BaseBranch, pr.HeadBranch)
		comments, _ := pullsStore.ListComments(req.Context(), pr.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"pull_request": prResponse(pr, derefString(pr.AuthorHandle)),
			"diff":         string(diff),
			"comments":     commentResponses(comments),
		})
	})

	r.Post("/repos/{owner}/{name}/pulls/{number}/comments", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		actor, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || actor == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		number, err := strconv.Atoi(chi.URLParam(req, "number"))
		if err != nil {
			http.Error(w, "invalid number", http.StatusBadRequest)
			return
		}
		pr, err := pullsStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
		if err != nil {
			if errors.Is(err, pulls.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var body struct{ Body string `json:"body"` }
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Body) == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}

		c, err := pullsStore.AddComment(req.Context(), pr.ID, actor.ID, body.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c.AuthorHandle = &actor.Handle
		writeJSON(w, http.StatusCreated, commentResponse(c))
	}))

	r.Post("/repos/{owner}/{name}/pulls/{number}/merge", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		actor, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || actor == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		// Only the repo owner may merge at MVP (PermissionChecker.ActionPush).
		if !permissions.Allow(permissions.Actor{UserID: actor.ID},
			permissions.Repo{OwnerID: repo.OwnerID, Visibility: repo.Visibility},
			permissions.ActionPush) {
			http.Error(w, "only the repository owner may merge", http.StatusForbidden)
			return
		}
		number, err := strconv.Atoi(chi.URLParam(req, "number"))
		if err != nil {
			http.Error(w, "invalid number", http.StatusBadRequest)
			return
		}
		pr, err := pullsStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
		if err != nil {
			if errors.Is(err, pulls.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if pr.State != pulls.StateOpen {
			http.Error(w, "pull request is not open", http.StatusConflict)
			return
		}

		message := "Merge pull request #" + strconv.Itoa(pr.Number) + " from " + pr.HeadBranch
		mergeOID, err := gitStorage.MergeBranches(req.Context(),
			repo.OwnerHandle, repo.Name, pr.BaseBranch, pr.HeadBranch, message,
			gitstorage.Identity{Name: actor.Handle, Email: actor.Email})
		if err != nil {
			if errors.Is(err, gitstorage.ErrMergeConflict) {
				http.Error(w, "merge conflict", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := pullsStore.MarkMerged(req.Context(), pr.ID, actor.ID, mergeOID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_ = bus.Publish(req.Context(), "pr.merged", map[string]any{
			"pr_id":            pr.ID,
			"repo_id":          repo.ID,
			"number":           pr.Number,
			"merged_by":        actor.ID,
			"merge_commit_oid": mergeOID,
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"merge_commit_oid": mergeOID,
			"state":            "merged",
		})
	}))

	// ---- Issues ---------------------------------------------------------

	r.Post("/repos/{owner}/{name}/issues", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		actor, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || actor == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		if !permissions.Allow(permissions.Actor{UserID: actor.ID},
			permissions.Repo{OwnerID: repo.OwnerID, Visibility: repo.Visibility},
			permissions.ActionRead) {
			http.NotFound(w, req)
			return
		}

		var body struct {
			Title          string `json:"title"`
			Body           string `json:"body"`
			AssigneeUserID string `json:"assignee_user_id"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		body.Title = strings.TrimSpace(body.Title)
		if body.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		iss, err := issuesStore.Create(req.Context(), issues.CreateInput{
			RepoID: repo.ID, AuthorID: actor.ID,
			Title: body.Title, Body: body.Body, AssigneeUserID: body.AssigneeUserID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, issueResponse(iss))
	}))

	r.Get("/repos/{owner}/{name}/issues", func(w http.ResponseWriter, req *http.Request) {
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		state := issues.State(req.URL.Query().Get("state"))
		list, err := issuesStore.ListByRepo(req.Context(), repo.ID, state)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, iss := range list {
			out = append(out, issueResponse(iss))
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Get("/repos/{owner}/{name}/issues/{number}", func(w http.ResponseWriter, req *http.Request) {
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		number, err := strconv.Atoi(chi.URLParam(req, "number"))
		if err != nil {
			http.Error(w, "invalid number", http.StatusBadRequest)
			return
		}
		iss, err := issuesStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
		if err != nil {
			if errors.Is(err, issues.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		comments, _ := issuesStore.ListComments(req.Context(), iss.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"issue":    issueResponse(iss),
			"comments": issueCommentResponses(comments),
		})
	})

	r.Post("/repos/{owner}/{name}/issues/{number}/comments", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
		actor, err := usersRepo.BySuperTokensID(req.Context(), stID)
		if err != nil || actor == nil {
			http.Error(w, "user not provisioned", http.StatusUnauthorized)
			return
		}
		repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
		if err != nil {
			httpRepoErr(w, err)
			return
		}
		number, err := strconv.Atoi(chi.URLParam(req, "number"))
		if err != nil {
			http.Error(w, "invalid number", http.StatusBadRequest)
			return
		}
		iss, err := issuesStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
		if err != nil {
			if errors.Is(err, issues.ErrNotFound) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var body struct{ Body string `json:"body"` }
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Body) == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}
		c, err := issuesStore.AddComment(req.Context(), iss.ID, actor.ID, body.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c.AuthorHandle = &actor.Handle
		writeJSON(w, http.StatusCreated, issueCommentResponse(c))
	}))

	r.Post("/repos/{owner}/{name}/issues/{number}/close", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		issueStateChange(w, req, usersRepo, reposStore, issuesStore, "close")
	}))
	r.Post("/repos/{owner}/{name}/issues/{number}/reopen", session.VerifySession(nil, func(w http.ResponseWriter, req *http.Request) {
		issueStateChange(w, req, usersRepo, reposStore, issuesStore, "reopen")
	}))

	// Smart Git HTTP transport — last so it doesn't shadow API routes.
	// Matches /<owner>/<name>.git/* (git advertises and pushes here).
	r.Handle("/{owner}/{name}.git/*", gitHTTP)
	r.Handle("/{owner}/{name}.git", gitHTTP)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("forge-platform listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func httpRepoErr(w http.ResponseWriter, err error) {
	if errors.Is(err, repos.ErrNotFound) {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func prResponse(pr *pulls.PullRequest, authorHandle string) map[string]any {
	return map[string]any{
		"id":               pr.ID,
		"number":           pr.Number,
		"title":            pr.Title,
		"body":             pr.Body,
		"head_branch":      pr.HeadBranch,
		"base_branch":      pr.BaseBranch,
		"state":            pr.State,
		"author":           authorHandle,
		"merge_commit_oid": derefString(pr.MergeCommitOID),
		"merged_at":        pr.MergedAt,
		"created_at":       pr.CreatedAt,
	}
}

func commentResponse(c *pulls.Comment) map[string]any {
	return map[string]any{
		"id":          c.ID,
		"body":        c.Body,
		"author":      derefString(c.AuthorHandle),
		"author_kind": c.AuthorKind,
		"created_at":  c.CreatedAt,
	}
}

func commentResponses(cs []*pulls.Comment) []map[string]any {
	out := make([]map[string]any, 0, len(cs))
	for _, c := range cs {
		out = append(out, commentResponse(c))
	}
	return out
}

func issueResponse(i *issues.Issue) map[string]any {
	out := map[string]any{
		"id":         i.ID,
		"number":     i.Number,
		"title":      i.Title,
		"body":       i.Body,
		"state":      i.State,
		"author":     derefString(i.AuthorHandle),
		"created_at": i.CreatedAt,
		"closed_at":  i.ClosedAt,
	}
	if kind := i.AssigneeKind(); kind != "" {
		out["assignee"] = map[string]any{
			"kind":   string(kind),
			"id":     derefString(i.AssigneeUserID),
			"handle": derefString(i.AssigneeUserHandle),
		}
	} else {
		out["assignee"] = nil
	}
	return out
}

func issueCommentResponse(c *issues.Comment) map[string]any {
	return map[string]any{
		"id":         c.ID,
		"body":       c.Body,
		"author":     derefString(c.AuthorHandle),
		"created_at": c.CreatedAt,
	}
}

func issueCommentResponses(cs []*issues.Comment) []map[string]any {
	out := make([]map[string]any, 0, len(cs))
	for _, c := range cs {
		out = append(out, issueCommentResponse(c))
	}
	return out
}

func issueStateChange(
	w http.ResponseWriter, req *http.Request,
	usersRepo *users.Repo, reposStore *repos.Store, issuesStore *issues.Store,
	op string,
) {
	stID := session.GetSessionFromRequestContext(req.Context()).GetUserID()
	actor, err := usersRepo.BySuperTokensID(req.Context(), stID)
	if err != nil || actor == nil {
		http.Error(w, "user not provisioned", http.StatusUnauthorized)
		return
	}
	repo, err := reposStore.GetByOwnerHandleAndName(req.Context(), chi.URLParam(req, "owner"), chi.URLParam(req, "name"))
	if err != nil {
		httpRepoErr(w, err)
		return
	}
	number, err := strconv.Atoi(chi.URLParam(req, "number"))
	if err != nil {
		http.Error(w, "invalid number", http.StatusBadRequest)
		return
	}
	iss, err := issuesStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
	if err != nil {
		if errors.Is(err, issues.ErrNotFound) {
			http.NotFound(w, req)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Only the repo owner or the issue author may close/reopen.
	authorIsActor := iss.AuthorID != nil && *iss.AuthorID == actor.ID
	if !(authorIsActor || actor.ID == repo.OwnerID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	switch op {
	case "close":
		err = issuesStore.Close(req.Context(), iss.ID, actor.ID)
	case "reopen":
		err = issuesStore.Reopen(req.Context(), iss.ID)
	}
	if err != nil {
		if errors.Is(err, issues.ErrNotFound) {
			// state already correct (e.g., trying to close an already-closed issue)
			http.Error(w, "no state change applied", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-fetch to return the updated issue.
	updated, _ := issuesStore.GetByRepoAndNumber(req.Context(), repo.ID, number)
	writeJSON(w, http.StatusOK, issueResponse(updated))
}

func meResponse(u *users.User) map[string]any {
	return map[string]any{
		"id":           u.ID,
		"handle":       u.Handle,
		"email":        u.Email,
		"display_name": u.DisplayName,
		"avatar_url":   u.AvatarURL,
		"provider":     u.Provider,
	}
}

func repoResponse(r *repos.Repository, ownerHandle string) map[string]any {
	return map[string]any{
		"id":          r.ID,
		"owner":       ownerHandle,
		"name":        r.Name,
		"description": r.Description,
		"visibility":  r.Visibility,
		"created_at":  r.CreatedAt,
		"clone_url":   "/" + ownerHandle + "/" + r.Name + ".git",
	}
}

func corsMiddleware(websiteDomain string) func(http.Handler) http.Handler {
	allowedHeaders := strings.Join(append([]string{"content-type"}, supertokens.GetAllCORSHeaders()...), ", ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", websiteDomain)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func supertokensMiddleware() func(http.Handler) http.Handler {
	mw := supertokens.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handled := false
			recorder := &peekResponseWriter{ResponseWriter: w, captured: &handled}
			mw.ServeHTTP(recorder, r)
			if !*recorder.captured {
				next.ServeHTTP(w, r)
			}
		})
	}
}

type peekResponseWriter struct {
	http.ResponseWriter
	captured *bool
}

func (p *peekResponseWriter) WriteHeader(status int) {
	*p.captured = true
	p.ResponseWriter.WriteHeader(status)
}

func (p *peekResponseWriter) Write(b []byte) (int, error) {
	*p.captured = true
	return p.ResponseWriter.Write(b)
}

func waitForDB(ctx context.Context, pool *pgxpool.Pool) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := pool.Ping(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return lastErr
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if strings.TrimSpace(v) == "" {
		log.Fatalf("required env var %s is empty", k)
	}
	return v
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
