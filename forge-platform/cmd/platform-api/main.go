package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/supertokens"

	"github.com/codecollab-co/forge/forge-platform/internal/auth"
	"github.com/codecollab-co/forge/forge-platform/internal/eventbus"
	"github.com/codecollab-co/forge/forge-platform/internal/users"
)

func main() {
	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")
	port := envOr("PORT", "8080")
	websiteDomain := envOr("WEBSITE_DOMAIN", "http://localhost:3000")

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
		writeJSON(w, http.StatusOK, map[string]any{
			"id":           u.ID,
			"handle":       u.Handle,
			"email":        u.Email,
			"display_name": u.DisplayName,
			"avatar_url":   u.AvatarURL,
			"provider":     u.Provider,
		})
	}))

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
	mw := supertokens.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// fall-through; chi handles non-supertokens routes via the next handler
	}))
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

// peekResponseWriter detects whether the SuperTokens middleware actually
// handled the request (by writing a status). If it didn't, we fall through
// to the chi router.
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
