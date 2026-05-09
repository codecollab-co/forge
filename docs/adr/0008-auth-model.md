# ADR 0008 — Auth model

**Status:** Accepted — 2026-05-09

## Context

Auth is hard to swap once users exist (sessions, password hashes, OAuth links, audit trails). It also has to satisfy two very different futures: a public SaaS where Twitter/HN-driven beachhead users sign up via OAuth, and a stage-2 self-hostable enterprise distribution (ADR-0001) where customers expect SSO and refuse to depend on a third-party auth vendor.

## Decision

- **Auth library: SuperTokens (self-hostable, OSS).** Hosted via their managed core during MVP for speed; the same code path works against a self-hosted core for stage-2 enterprise.
- **MVP login methods: OAuth only — GitHub and Google.** No email+password until post-MVP. SSH keys and Personal Access Tokens (PATs) for `git` clients are required and stored in `platform.*`.
- **Org / Team model: deferred past MVP.** Repos at MVP are owned only by Users. Orgs and Team-level permissions arrive in v1.1.

## Consequences

- We avoid running our own password storage at MVP — and avoid the breach surface that comes with it.
- The beachhead audience (every dev has a GitHub account) onboards in two clicks.
- Stage-2 self-host is unblocked: customers run their own SuperTokens core; the application code is unchanged.
- Deferring Orgs cuts ~3–4 weeks of permission-model work and removes a pile of edge cases (transfers, billing-on-org, member roles). Some users will ask for it; we say "soon."
- We accept the irony that users log into a GitHub-competitor with their GitHub account. Mitigated later by adding email+password and SSO providers.
- Auth0 / Clerk / WorkOS were rejected because the migration cost when stage-2 enterprise lands is too high.

## Alternatives considered

- **Email + password from day 1** — universal, but adds password reset, MFA, breach handling for zero MVP value. Rejected.
- **Auth0 / Clerk** — fastest to ship, but ripping out at stage 2 is expensive and the per-MAU cost grows quickly. Rejected.
- **Build on Ory / Keycloak** — comparable to SuperTokens; SuperTokens chosen for lower setup overhead and a friendlier hosted-core option during MVP.
- **Orgs in MVP** — feels like table stakes; in practice the beachhead is solo devs and small teams who can live with user-owned repos for v1.0.
