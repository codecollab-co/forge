# Forge

AI-native Git host. Agents are first-class collaborators on a repository — assign an Issue to an Agent and a Pull Request appears.

Status: pre-MVP. Strategy and architecture locked; implementation not yet started.

## Where to start

- [`CONTEXT.md`](./CONTEXT.md) — positioning, glossary, service map.
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — repo layout and cross-cutting rules.
- [`docs/PRD-mvp.md`](./docs/PRD-mvp.md) — MVP product requirements.
- [`docs/adr/`](./docs/adr/) — architectural decisions (ADRs 0001–0009).

## Layout

This is a monorepo. Each top-level service is named like the independent repo it would become if extracted:

- [`forge-platform/`](./forge-platform/) — Go. Git server, platform API, auth.
- [`forge-agent/`](./forge-agent/) — Python + FastAPI. Run lifecycle, agent loop, reviewer.
- [`forge-web/`](./forge-web/) — TypeScript + Next.js. Product UI.
- [`forge-infra/`](./forge-infra/) — Terraform. AWS topology.

## Local development

```bash
# Generate an RS256 keypair and write it to .env
./scripts/gen-jwt-keys.sh > .env

# Bring up Postgres + all three services
docker compose up --build

# Verify
curl http://localhost:8080/healthz                          # platform
curl http://localhost:8081/healthz                          # agent
open  http://localhost:3000                                 # web

# Round-trip a ping event through the EventBus
curl -X POST http://localhost:8080/ping
docker compose logs forge-agent | grep "event received"
```

## License

Apache-2.0. See [`LICENSE`](./LICENSE).
