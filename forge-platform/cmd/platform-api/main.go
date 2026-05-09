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

	"github.com/codecollab-co/forge/forge-platform/internal/auth"
	"github.com/codecollab-co/forge/forge-platform/internal/eventbus"
)

func main() {
	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")
	port := envOr("PORT", "8080")

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

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "forge-platform"})
	})

	// Issues a service token a downstream service can verify. Wired up properly
	// once user auth lands (slice 2). Kept here so slice 1 can demo the JWT seam.
	r.Post("/internal/token", func(w http.ResponseWriter, _ *http.Request) {
		tok, err := signer.Issue("system", time.Hour)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": tok})
	})

	// Publishes a `ping` event for the slice-1 round-trip.
	r.Post("/ping", func(w http.ResponseWriter, req *http.Request) {
		if err := bus.Publish(req.Context(), "ping", map[string]any{"at": time.Now().UTC()}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
	})

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
