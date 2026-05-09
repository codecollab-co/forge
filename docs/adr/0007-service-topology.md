# ADR 0007 — Service topology and language choice

**Status:** Accepted — 2026-05-09

## Context

The product needs three things that pull in different directions: a correct Git server (systems-level, concurrent, single-binary friendly), an agent orchestrator (rich AI/ML ecosystem, model SDKs, sandbox SDKs), and a product UI (full-stack typed web). No single language is best at all three. Forcing one language onto all three areas would either compromise the agent loop (no Go AI ecosystem) or compromise the platform service (Python's concurrency story for a Git server is poor).

## Decision

Three services, three languages, with strict seams.

- **`platform-api` (Go)** — Git server (HTTPS + SSH wrappers around stock `git`), platform REST/GraphQL API (users, orgs, repos, issues, PRs, permissions, webhooks), auth (token issuer), Postgres owner. **Only writer to Git.**
- **`agent-orchestrator` (Python + FastAPI)** — Run lifecycle, agent loop, prompt assembly, tool use, `ModelClient` (per ADR-0006), `SandboxProvider` (per ADR-0005), Reviewer Agent. **Only caller of LLMs and sandboxes.**
- **`web` (TypeScript + Next.js)** — product UI and marketing site. Talks to `platform-api` for state, to `agent-orchestrator` only for live Run streams (SSE).

### Seams

- **Queue between Go and Python.** Postgres-as-queue or Redis Streams at MVP; SQS later. Not Kafka.
- **Shared Postgres, separate schemas.** `platform.*` owned by Go, `agent.*` owned by Python. Cross-schema writes are forbidden.
- **One auth token format** signed by Go; Python validates with the public key.
- **Python never shells out to `git`.** It asks Go to create branches / open PRs.
- **Go never imports an LLM or sandbox SDK.** All AI cost and execution lives in Python.

## Consequences

- Three deploy targets from day 1. Local dev requires `docker-compose` with Postgres + Redis + both backends.
- Each service can hire / be reasoned about by people fluent in its language.
- Code review must enforce the seams: any cross-seam call (Python touching `git`, Go importing `anthropic`) is a red flag.
- Replacing the sandbox provider (ADR-0005) or the model vendor (ADR-0006) is a single-service change, not a platform-wide one.
- Migration to S3-backed Git (ADR-0004) is contained inside `platform-api`.

## Alternatives considered

- **Single-language (TypeScript/Node) full stack** — simplest ops, acceptable AI ecosystem, but wrapping Git and serving SSH at scale is awkward and the Python AI ecosystem still leads. Kept as a fallback if "three services" becomes too heavy.
- **Single-language (Go) end-to-end** — strong platform service, but the agent loop would be reimplementing libraries that exist as `pip install`. Rejected.
- **Single-language (Python) end-to-end** — strongest agent loop, weakest Git server. Rejected.
- **Rust for `platform-api`** — better performance and safety, slower to ship. Revisit for hot paths post-PMF.
