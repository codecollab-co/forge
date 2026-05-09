-- Bootstrap schema for slice 1. Loaded by Postgres on first container start
-- via docker-compose's docker-entrypoint-initdb.d hook. Real migrations move
-- to atlas in a follow-up.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS platform;
CREATE SCHEMA IF NOT EXISTS agent;

CREATE TABLE IF NOT EXISTS platform.events (
    id          BIGSERIAL PRIMARY KEY,
    type        TEXT NOT NULL,
    payload     JSONB NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    locked_at   TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS events_pending_idx
    ON platform.events (id)
    WHERE status = 'pending';
