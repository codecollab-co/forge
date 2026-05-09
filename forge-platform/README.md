# forge-platform

Go service. Git server (HTTPS + SSH), platform REST/GraphQL API, auth issuer, owns the `platform.*` schema in Postgres.

**Only writer to Git on disk.** **Never imports an LLM or sandbox SDK.**

See [`../ARCHITECTURE.md`](../ARCHITECTURE.md) and the ADRs at [`../docs/adr/`](../docs/adr/).

## Status

Pre-implementation. Scaffolding lands as part of issue #1 (walking skeleton).
