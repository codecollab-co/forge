# Architecture

This repository is a **monorepo** containing four top-level units, each named like the independent repository it would become if extracted:

| Directory | Language | Role |
|---|---|---|
| `forge-platform/` | Go | `platform-api` — Git server, platform REST/GraphQL, auth, owns Postgres `platform.*` schema. Only writer to Git. |
| `forge-agent/` | Python + FastAPI | `agent-orchestrator` — Run lifecycle, agent loop, `ModelClient`, `SandboxProvider`, Reviewer Agent. Only caller of LLMs and sandboxes. |
| `forge-web/` | TypeScript + Next.js | Product UI and marketing site. |
| `forge-infra/` | Terraform | AWS deployment topology (per ADR-0009). |

See ADR-0007 for the service seams, and ADR-0009 for the AWS topology.

## Why monorepo

- Atomic cross-seam PRs (changing an event schema in Go and the consumer in Python in one diff).
- Single issue tracker, single CI configuration to reason about.
- Easy to extract any unit into its own repo later via `git filter-repo --subdirectory-filter <name>` — this is why the directories are named as if they were already separate repos.

## CI

Path-filtered: each service's CI runs only when its directory or shared infra changes. No matrix-of-everything on every PR.

## Local dev

`docker compose up` from the repo root brings up Postgres, Redis (used as a cache, not a queue at MVP — see below), and all three services. Sandboxes are external (E2B/Modal per ADR-0005) and called over the public internet during local dev.

## Queue

**Postgres-as-queue at MVP** (per ADR-0009 and confirmed during slice 1). Implemented behind the `EventBus` interface so the swap to Redis Streams or SQS is a single-module change.

## Cross-cutting rules

- **Go never imports an LLM or sandbox SDK.**
- **Python never shells out to `git`.**
- **Cross-schema writes are forbidden** — `platform.*` is read-only from `forge-agent`; `agent.*` is read-only from `forge-platform`.

Code review enforces these. See ADR-0007.
