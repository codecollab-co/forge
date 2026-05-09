# ADR 0002 — Agent execution model

**Status:** Accepted — 2026-05-09

## Context

The wedge (ADR-0001) is AI-as-author: assign an Issue to an Agent, get a PR. That phrase hides three independent design axes — trigger model, execution location, and capability boundary — each of which significantly shapes the infrastructure, the security posture, and the product's feel.

## Decision

- **Trigger model: async-default with hop-in.** The Agent works in the background and opens a PR when done. The user can optionally "join" an active Run to chat / steer mid-flight.
- **Execution location: platform-owned Sandboxes.** Every Run gets a fresh microVM-class sandbox that the platform provisions, monitors, and bills for. Execution does *not* live in user CI or on user laptops.
- **Capability boundary at MVP: code-only.** Inside the sandbox the Agent has shell + git + file access. No outbound internet, no user secrets, no deploy targets. Boundary expands to "code + internet" post-MVP.

## Consequences

- **Sandbox infrastructure is core, not optional.** Building/operating this is a meaningful chunk of total engineering effort. We accept this as the cost of the wedge.
- **Async-default sets the UX.** The product must feel useful even when the user isn't watching. PR quality + clear Run summaries matter more than live-streaming diffs.
- **Code-only at MVP makes the first useful tasks narrow** (unit-test fixes, small refactors, isolated features). Tasks needing live API calls or staging deploys are post-MVP.
- **Self-hostable (ADR-0001 stage 2) becomes harder.** A customer hosting Forge must also operate sandbox infra — we will need a "bring-your-own-runner" mode eventually.

## Alternatives considered

- **Pure async (Devin-like)** — feels remote in 2026; rejected.
- **Pure sync (Cursor-background-agent-like)** — much harder infra and we lose the async-friendly enterprise/team workflows; rejected.
- **Execute in user CI (GitHub Actions)** — cheap, but reduces the platform to a thin layer; the wedge collapses.
- **Execute on user laptop** — defeats SaaS positioning.
- **Code + internet at MVP** — much more useful, but security model isn't ready. Capability boundaries are easier to expand than retract.
