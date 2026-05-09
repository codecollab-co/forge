# Forge — Context

AI-native Git host. The platform's reason to exist is that **agents are first-class collaborators on a repository**, not a feature bolted onto a passive Git server.

## Positioning

- **Wedge:** AI-as-author (assign an issue → agent opens a PR), with AI-as-reviewer as the companion default.
- **Beachhead:** small teams already using Cursor / Devin / Aider, frustrated that GitHub is a passive substrate for AI workflows.
- **Expansion:** small startups (2–10 devs).
- **Stage 2:** self-hostable distribution for enterprise (sovereignty, compliance).

## Glossary

### Agent
An autonomous coding entity that can be assigned work on a Repository. Two roles:
- **Author Agent** — assigned to an Issue, produces a PR.
- **Reviewer Agent** — runs automatically on every PR, posts review comments.

"Agent" without a qualifier means Author Agent (the headline product).

### Run
A single execution of an Agent against a task, inside one Sandbox, producing at most one PR. Runs are async by default; the user can "hop in" to chat with an active Run.

### Sandbox
An isolated execution environment (microVM-class) that the platform provisions per Run. The Agent has shell + git + file access inside it; **no internet, no secrets, no deploy at MVP** (capability boundary expands later).

### Repository
Standard Git repository, hosted on the platform. The unit of permissions, Issues, PRs, and Agent assignments.

### Issue
A unit of work on a Repository. Distinguishing feature vs. GitHub: an Issue is **assignable to an Agent** in the same way it's assignable to a human.

### PR
Proposed change to a Repository, produced by either a human or an Author Agent. Reviewer Agent runs on every PR by default.

### ModelClient
The single internal interface through which all LLM calls flow. Lets us route per-task and swap vendors without touching the agent loop. Lives in `agent-orchestrator` (per ADR-0006, ADR-0007).

### SandboxProvider
The single internal interface for obtaining a Sandbox. At MVP backed by E2B/Modal; post-PMF replaced by an in-house Firecracker fleet (per ADR-0005). Lives in `agent-orchestrator`.

## Services

- **`platform-api` (Go)** — Git server, platform API, auth, Postgres owner. Only writer to Git.
- **`agent-orchestrator` (Python/FastAPI)** — Run lifecycle, agent loop, ModelClient, SandboxProvider, Reviewer Agent. Only caller of LLMs and sandboxes.
- **`web` (TypeScript/Next.js)** — product UI, marketing site.

See ADR-0007 for the service seams.

## Out of scope at MVP

CI/Actions, Releases, Packages, Wiki, Projects/Boards, Org/Team management, Marketplace, Webhooks, GitHub-import, Mobile, Billing. See ADR-0003.
