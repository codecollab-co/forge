// Package eventbus is the publish side of the Postgres-as-queue EventBus
// described in ADR-0009. The interface is intentionally narrow so the
// backend can be swapped to Redis Streams or SQS later without touching
// callers.
package eventbus

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Bus struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Bus {
	return &Bus{pool: pool}
}

func (b *Bus) Publish(ctx context.Context, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = b.pool.Exec(ctx,
		`INSERT INTO platform.events (type, payload) VALUES ($1, $2::jsonb)`,
		eventType, string(body),
	)
	return err
}
