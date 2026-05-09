# PRD — Forge MVP

**Status:** Draft — 2026-05-09
**Scope:** Closed-beta MVP of Forge, the AI-native Git host (per ADR-0001).
**Companion docs:** [`CONTEXT.md`](../CONTEXT.md), ADR-0001 through ADR-0009.

## Problem Statement

Developers on small AI-first teams (already heavy users of Cursor, Devin, Aider) treat their Git host as a passive substrate — they push code to GitHub and *separately* run AI tools against it. The integration is manual: copy issue text into Cursor, ask the agent to do the work, push the branch, open a PR, paste context back into the issue. The Git host knows nothing about the AI workflow that actually produced the code, and the AI workflow knows nothing about the team's review and merge process. Every team has reinvented the same brittle glue.

## Solution

Forge makes the Agent a first-class collaborator on a Repository, not a tool the developer runs alongside it. A user creates an Issue, clicks "Assign to Agent," and an Author Agent runs asynchronously in a platform-owned Sandbox (per ADR-0002), opening a Pull Request when finished. A Reviewer Agent then reviews that PR — and every other PR — by default. The user can "hop in" to a live Run mid-flight to steer the Agent.

At MVP this product ships as a public SaaS; stage-2 enterprise self-hosting is on the roadmap (per ADR-0001) and constrains today's choices. The MVP deliberately omits CI, billing, releases, packages, wikis, projects, orgs, and password-based auth (per ADR-0003).

## User Stories

**Authentication and identity**
1. As a developer, I want to sign in with my GitHub account, so I can start using Forge without creating yet another password.
2. As a developer, I want to sign in with my Google account, so I can use Forge if I don't want to involve GitHub at all.
3. As a developer, I want to register an SSH key on my profile, so I can `git push` and `git pull` over SSH from my laptop.
4. As a developer, I want to mint a Personal Access Token, so I can `git` over HTTPS from environments where SSH is awkward.
5. As a developer, I want to revoke an SSH key or PAT, so I can rotate credentials safely.
6. As a developer, I want to sign out of the web UI, so I can hand my laptop to someone else.

**Repositories**
7. As a developer, I want to create a new Repository under my user account, so I have a place to push code.
8. As a developer, I want to choose whether a Repository is public or private at creation, so I control who sees my code.
9. As a developer, I want to see a list of my Repositories, so I can navigate to the one I'm working on.
10. As a developer, I want to `git clone` a Repository over HTTPS, so I can work on it locally.
11. As a developer, I want to `git clone` a Repository over SSH, so I don't have to type a token every time.
12. As a developer, I want to `git push` a branch, so my code is available on Forge.
13. As a developer, I want a clear permission denial when I try to push to a Repository I don't own, so I'm not confused about what failed.
14. As a viewer of a public Repository, I want to browse files in the web UI, so I can read code without cloning.
15. As a viewer, I want to see the README rendered on the Repository home page, so I understand the project at a glance.
16. As a viewer, I want to switch which branch I'm browsing, so I can inspect work-in-progress.

**Pull Requests**
17. As a developer, I want to open a Pull Request from a branch into another branch, so my changes can be reviewed before merging.
18. As a reviewer, I want to see a unified diff of a Pull Request, so I can understand what changed.
19. As a reviewer, I want to leave a top-level comment on a Pull Request, so I can give feedback.
20. As a maintainer, I want to merge a Pull Request from the web UI, so the branch can be integrated.
21. As a maintainer, I want the destination branch to update on merge, so subsequent clones get the new state.
22. As a developer, I want to see a list of open and closed PRs on a Repository, so I can find one to work on.

**Issues**
23. As a developer, I want to create an Issue with a title and body on a Repository, so I can track work.
24. As a developer, I want to mark an Issue as open or closed, so the list reflects active work.
25. As a developer, I want to see all Issues on a Repository, so I can see what's outstanding.
26. As a developer, I want to view an Issue's details, so I can decide what to do about it.

**Agent — assignment and Run lifecycle**
27. As a developer, I want to assign an Agent to an Issue, so the Agent will attempt the work without me writing any code.
28. As a developer, I want to see that a Run has started for an Issue, so I know the Agent is working.
29. As a developer, I want to see the Run's current state (queued, running, succeeded, failed), so I know whether to wait or intervene.
30. As a developer, I want a link to the Pull Request the Agent opened, so I can review the work.
31. As a developer, I want to see a list of past Runs on an Issue, so I can refer back to earlier attempts.
32. As a developer, I want to cancel a Run that's clearly going wrong, so I'm not paying for a wasted Sandbox.

**Agent — live interaction (hop-in)**
33. As a developer, I want to "hop into" an active Run, so I can watch it work in real time.
34. As a developer in a live Run, I want to see the Agent's tool calls and thoughts streaming in, so I trust what it's doing.
35. As a developer in a live Run, I want to send a message to the Agent, so I can correct it without aborting.

**Reviewer Agent**
36. As a maintainer, I want every newly opened Pull Request to receive an automated review, so I don't have to be the first reader.
37. As a maintainer, I want the review to flag correctness, security, and style concerns inline on the diff where possible, so feedback lives next to the code.
38. As a maintainer, I want the review to clearly identify itself as Agent-generated, so I don't confuse it with a human reviewer.

**LLM and BYOK**
39. As a developer in closed beta, I want my Agent Runs to "just work" on platform-paid models, so I can evaluate the product without any setup.
40. As a power user, I want to attach my own Anthropic or OpenAI API key, so my Runs are billed to me directly and I'm not rate-limited by the platform.
41. As a power user, I want to remove a stored API key, so I can rotate credentials.

## Implementation Decisions

### Service topology (per ADR-0007)

Three services, one shared Postgres with two schemas, one queue.

- **`platform-api` (Go)** — Git server (HTTPS via `git-http-backend`, SSH via custom server fronting `git-upload-pack`/`git-receive-pack`), platform REST/GraphQL API, auth, owns `platform.*` schema. Only writer to Git on disk.
- **`agent-orchestrator` (Python + FastAPI)** — Run lifecycle, Agent loop, `ModelClient`, `SandboxProvider`, Reviewer Agent. Owns `agent.*` schema. Only caller of LLMs and sandboxes. Exposes SSE for live Run streams.
- **`web` (TypeScript + Next.js)** — product UI. Talks to `platform-api` for state, to `agent-orchestrator` for live Run streams.

### Deep modules

These four are the load-bearing modules; design care is concentrated here.

1. **`GitRepository` (Go)** — encapsulates all on-disk Git operations. Surface: `Init`, `MirrorClone`, `ListRefs`, `ReadTree`, `ReadBlob`, `CreateBranch`, `Diff`, `Merge`. Hides shell-out details, locking, and the future EBS→S3 backend swap (ADR-0004).
2. **`ModelClient` (Python)** — single interface for every LLM call. Hides routing, retry, vendor swap, BYOK key resolution, token accounting. No service outside `agent-orchestrator` may import a model SDK directly.
3. **`SandboxProvider` (Python)** — `Acquire(spec) → Sandbox`, `Release(sandbox)`. Backed by E2B/Modal at MVP (ADR-0005); replaceable with an in-house Firecracker fleet without code changes elsewhere.
4. **`RunStateMachine` (Python)** — pure logic over Run events. Inputs: events. Outputs: next state + side-effect intents (e.g., "open PR", "post review comment"). No I/O. Tested in isolation, no model or sandbox involvement.

Conventional but still extracted:

5. **`PermissionChecker` (Go)** — pure `(user, repo, action) → allow/deny`. The whole permission policy lives here.
6. **`AuthTokenIssuer` / `AuthTokenVerifier`** — JWT signing in Go (issuer); JWKS-based verification in Python.
7. **`EventBus`** — publish/consume abstraction. Postgres-as-queue at MVP; swappable to Redis Streams or SQS without changing publishers/consumers.
8. **`ReviewerAgent` (Python)** — `(diff, context) → []ReviewComment`. Pure over its inputs once a `ModelClient` is injected.

### Run lifecycle

Run states (owned by `RunStateMachine`):

```
queued → running → (succeeded | failed | cancelled)
```

- Created when a user assigns an Issue to an Agent (`POST /issues/:id/assign-agent` on `platform-api`). `platform-api` writes the Run row in `agent.runs` and publishes a `run.requested` event.
- `agent-orchestrator` consumes the event, transitions to `running`, acquires a `SandboxProvider`-backed Sandbox, runs the Agent loop.
- On success, the orchestrator calls `platform-api` to create a branch and open a PR; the resulting PR ID is recorded on the Run.
- On failure, the Run records an error category and message; no PR is opened.
- Cancellation is cooperative: the user requests cancel via `platform-api`, which publishes `run.cancel-requested`; the orchestrator releases the Sandbox at the next checkpoint.

### Cross-service contracts

- **Auth:** all `platform-api`-issued tokens are JWTs signed with a key whose JWKS is exposed for Python to verify. No shared secret.
- **Events on the bus:** `run.requested`, `run.cancel-requested`, `pr.opened`, `pr.merged`. Schema versioned with a `v` field; consumers ignore unknown versions rather than crashing.
- **Schema ownership:** `platform.*` is read-only from Python; `agent.*` is read-only from Go. Cross-schema writes are forbidden; violators caught in code review.
- **Sandbox isolation at MVP:** code-only — no outbound internet, no user secrets, no deploy targets (per ADR-0002). Boundary expansion is a post-MVP decision.

### LLM strategy at MVP (per ADR-0006)

- All LLM calls flow through `ModelClient`.
- Default model for the Author Agent: Claude (current frontier coding model). Default for the Reviewer Agent: same family, possibly a smaller variant.
- Bundled tier (closed beta): platform-paid Claude with per-user rate limits.
- BYOK: optional Anthropic/OpenAI key on user profile, stored encrypted at rest in `platform.user_secrets`. When present, `ModelClient` uses the user's key for that user's Runs.

### Storage (per ADR-0004 and ADR-0009)

- Git: bare repos on EBS attached to `platform-api` nodes. MVP runs a small fixed number of Git nodes; sharding-by-repo is post-MVP.
- Postgres: RDS, single-AZ at MVP, multi-AZ before GA.
- Object store: S3 for Run artifacts, attachments, future LFS objects.
- Queue: Postgres-as-queue at MVP; migration to Redis Streams or SQS is contained behind `EventBus`.

### Out-of-scope at MVP (per ADR-0003)

CI / Actions, Releases, Packages, Wiki, Projects/Boards, Orgs/Teams, Marketplace, Webhooks, GitHub-import, Mobile, Billing, email+password auth.

## Testing Decisions

A good test here exercises **external behavior of a deep module** through its public interface, with collaborators replaced by deterministic fakes (an in-memory `EventBus`, a recorded-response `ModelClient`, an in-memory `SandboxProvider`). Tests never hit real LLM vendors, real sandbox vendors, or the network. Tests assert observable outcomes (returned values, emitted events, written rows) — not internal call sequences.

### Modules with thorough unit/component tests

- **`GitRepository`** — integration tests against a real `git` binary on a temp directory. Cover empty-repo init, push of one commit, branch creation, merge, ref listing.
- **`PermissionChecker`** — table-driven tests covering the full `(actor × repo-visibility × action)` matrix.
- **`RunStateMachine`** — pure-function tests over event sequences; covers happy path, mid-Run cancel, failure, and unknown event handling.
- **`ReviewerAgent`** — tests against fixed diffs with a recorded-response `ModelClient`. Asserts comment shape and anchoring on the diff, not exact wording.

### Modules tested at their seam (with fakes)

- **`ModelClient`** — recorded-response fake. Tests verify the Agent loop's behavior given canned model outputs, including retry/fallback.
- **`SandboxProvider`** — in-memory fake. Tests verify lifecycle (`Acquire` → use → `Release`) and that Runs do not leak sandboxes on error paths.
- **`EventBus`** — in-memory implementation used by every other test that needs to observe events.

### End-to-end coverage

One e2e test per tracer-bullet slice from the issues breakdown, walking `web → platform-api → agent-orchestrator → fakes`. The wedge tracer ("assign Issue → PR appears") gets the most attention; e2e for SSH, PATs, and visibility flags can rely on a single golden-path test each.

### Skipped at MVP

Per-handler unit tests on `platform-api` HTTP handlers, per-component tests on `web`. Coverage comes from the e2e slice tests; revisit when handlers or components grow real internal logic.

### Prior art

None — `forge/` is greenfield. As real test patterns emerge in the first two slices, capture them in a short `TESTING.md` so later slices follow the same shape.

## Out of Scope

- All MVP-deferred items listed in ADR-0003 (CI, releases, packages, wikis, projects, orgs/teams, marketplace, webhooks, GitHub-import, mobile, billing, email+password auth).
- Sandbox capability expansion beyond code-only (no outbound internet, no secrets, no deploy at MVP).
- In-house Firecracker fleet (deferred until post-PMF; covered by ADR-0005).
- S3-backed Git storage (deferred; covered by ADR-0004).
- Native CI/CD; users keep using GitHub Actions or CircleCI on their existing repos in parallel.
- Multi-region deployment; MVP runs single-region.
- High availability beyond single-AZ RDS; multi-AZ is a pre-GA concern, not an MVP one.
- Compliance certifications (SOC 2, ISO, etc.) — not blockers for closed beta.

## Further Notes

- The closed beta should curate its first cohort by hand. The beachhead segment (per ADR-0001) is small and loud — onboarding 20–50 the right teams matters more than reaching 5,000 wrong ones.
- The first slice in the breakdown (walking skeleton) is HITL because it locks in a number of small but irreversible choices (monorepo layout, queue choice, deploy story). Treat it as scaffolding work and don't let it drag.
- The "wedge tracer" slice (assign Issue → fake-but-real PR appears) should ship before the real Agent loop. Validating the seams with a stub Agent prevents weeks of debugging integration bugs while also tuning a model.
- Stage-2 self-host (per ADR-0001) is a constant constraint, not a future feature. Every MVP choice is checked against "does this still work when a customer runs it on their own AWS?" — the answer must remain yes.
