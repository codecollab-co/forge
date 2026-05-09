# forge-agent

Python + FastAPI service. Run lifecycle, agent loop, `ModelClient`, `SandboxProvider`, Reviewer Agent. Owns the `agent.*` schema in Postgres.

**Only caller of LLMs and sandboxes.** **Never shells out to `git`** — asks `forge-platform` to create branches and open PRs over HTTP.

See [`../ARCHITECTURE.md`](../ARCHITECTURE.md) and the ADRs at [`../docs/adr/`](../docs/adr/).

## Status

Pre-implementation. Scaffolding lands as part of issue #1 (walking skeleton).
