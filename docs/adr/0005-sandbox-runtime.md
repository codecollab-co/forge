# ADR 0005 — Sandbox runtime

**Status:** Accepted — 2026-05-09

## Context

ADR-0002 commits us to platform-owned Sandboxes for every Agent Run. Building isolation infrastructure (Firecracker microVMs, image management, networking, lifecycle) is a multi-quarter effort. Agent-generated code must be treated as untrusted, which rules out shared-kernel container hosting.

## Decision

- **MVP: outsource sandboxing to a managed provider (E2B as primary candidate; Modal as fallback).** Each Run calls the provider's API to obtain an isolated sandbox.
- **Post-PMF: replace the provider with our own Firecracker-based fleet.** The Sandbox layer is encapsulated behind a `SandboxProvider` interface so the swap is a contained migration, not a rewrite.
- **Not on the table at MVP:** plain Docker on shared hosts (unsafe for untrusted code), DIY Firecracker (months of work pre-empting the wedge).

## Consequences

- We ship the Agent loop in weeks instead of quarters.
- Per-Run margin is worse during MVP (we pay a sandbox vendor per second), but absolute volume is small during closed beta.
- We accept a vendor dependency on the most novel piece of our infra — mitigated by E2B being open source, which makes the eventual self-host migration concrete.
- ADR-0001's stage-2 self-hostable promise depends on the DIY replacement landing before we sell to enterprise. This is a tracked dependency, not a blocker today.
- All sandbox-touching code must go through the `SandboxProvider` interface — direct E2B/Modal SDK calls outside that boundary are forbidden.

## Alternatives considered

- **Docker on ECS/Fargate** — kernel-shared, unsafe for untrusted Agent-generated code. Rejected.
- **gVisor / Kata Containers** — better isolation, but still requires us to build orchestration. Provides middle-ground complexity without a clear win over "buy now, build later."
- **Firecracker DIY at MVP** — defensible if the sandbox itself is the moat (sub-second cold starts, persistent VM state across Runs). We're betting the moat is agent UX, not sandbox infra. Revisited if that bet looks wrong.
