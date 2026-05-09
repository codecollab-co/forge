# ADR 0008 — Auth model

**Status:** Accepted — 2026-05-09. **Revised** — 2026-05-10 (login methods scope).

## Context

Auth is hard to swap once users exist (sessions, password hashes, OAuth links, audit trails). It also has to satisfy two very different futures: a public SaaS where Twitter/HN-driven beachhead users sign up, and a stage-2 self-hostable enterprise distribution (ADR-0001) where customers expect SSO and refuse to depend on a third-party auth vendor.

The original revision (2026-05-09) shipped OAuth-only with GitHub + Google. Revisiting after a hands-on first run: the irony of using a GitHub account to sign into a GitHub competitor was awkward, and email/password is table stakes for a closed beta where invitees may not have a Google account on the device they're testing on.

## Decision

- **Auth library: SuperTokens (self-hostable, OSS).** Hosted via the local SuperTokens core during MVP for speed; the same code path works against a self-hosted core for stage-2 enterprise.
- **MVP login methods (revised 2026-05-10):**
  - **Email + password** via the SuperTokens `EmailPassword` recipe — primary path.
  - **Google OAuth** via the SuperTokens `ThirdParty` recipe — secondary, only enabled when `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` are configured.
  - **GitHub OAuth: removed.** Optical conflict with the product positioning, and Google + email/password already cover the beachhead onboarding flows.
- **Identity model:** email is the SuperTokens-side identifier. The `handle` (the `@username` in repo URLs like `alice/repo`) is derived from the email's local part on first sign-in, deduplicated with a numeric suffix when needed. There is no separate "username" field at MVP.
- **SSH keys** and **Personal Access Tokens (PATs)** for `git` clients are required and stored in `platform.*` (slice 11 / 12).
- **Org / Team model: deferred past MVP.** Repos at MVP are owned only by Users. Orgs and Team-level permissions arrive in v1.1.

## Consequences

- Email-only signup means we own password reset and breach-handling surface — accepted as the cost of an auth flow that "just works" for invited beta users without requiring Google/GitHub. SuperTokens handles password hashing (Argon2) and reset flows.
- The beachhead audience (every dev has a Google account) onboards in two clicks via OAuth when configured.
- Stage-2 self-host is unblocked: customers run their own SuperTokens core; the application code is unchanged.
- Deferring Orgs cuts ~3–4 weeks of permission-model work and removes a pile of edge cases (transfers, billing-on-org, member roles). Some users will ask for it; we say "soon."
- Auth0 / Clerk / WorkOS were rejected because the migration cost when stage-2 enterprise lands is too high.
- Removing GitHub OAuth before launch loses ~zero adoption from the beachhead segment (every developer also has a Google account or can use email) while removing the optical awkwardness.

## Alternatives considered

- **OAuth-only (original 2026-05-09 decision)** — clean breach-surface story but worse onboarding UX and the GitHub-into-GitHub-competitor optic. Reverted.
- **Username + password (separate from email)** — possible, but SuperTokens has no native "username" recipe; would require a custom layer. Email is the universal identifier and the `handle` already serves the "username" role for URLs and mentions. Reconsider only if a real user need surfaces.
- **Auth0 / Clerk** — fastest to ship, but ripping out at stage 2 is expensive and the per-MAU cost grows quickly. Rejected.
- **Build on Ory / Keycloak** — comparable to SuperTokens; SuperTokens chosen for lower setup overhead and a friendlier hosted-core option during MVP.
- **Orgs in MVP** — feels like table stakes; in practice the beachhead is solo devs and small teams who can live with user-owned repos for v1.0.
