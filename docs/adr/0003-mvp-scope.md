# ADR 0003 — MVP scope

**Status:** Accepted — 2026-05-09

## Context

A naive "GitHub clone" MVP attempts feature parity and ships in 2027. A too-thin MVP ("agent + repo viewer") has no Git host for the Agent to integrate with. We need the smallest scope that demonstrates the wedge end-to-end to the beachhead segment (ADR-0001).

## Decision

**MVP includes exactly six things:**

1. **Git server** — push/pull over HTTPS+SSH, repo CRUD, branches, basic web file viewer.
2. **PR primitive** — branch → diff → merge. No advanced review UI.
3. **Issue primitive** — title, body, status, **assignable to an Agent**.
4. **Agent runtime** — Sandboxes + the "assign Issue → Agent opens PR" loop (per ADR-0002).
5. **Reviewer Agent** — auto-review on every PR.
6. **Auth** — sign-up, login, repo-level permissions.

**Explicitly deferred past MVP:** CI/Actions, Releases, Packages, Wiki, Projects/Boards, Org/Team management, Marketplace, Webhooks, GitHub-import, Mobile, Billing.

- **CI/Actions deferred** — users bring their own CI (point Actions/CircleCI at the repo) until we ship native CI. Painful but unavoidable; native CI is a 6-month project.
- **Billing deferred** — free during closed beta. Saves ~1–2 months for zero contribution to the wedge.

## Consequences

- The MVP is shippable in roughly two quarters of focused work, not eighteen months.
- Some users will bounce because "no Actions" feels broken. Acceptable: the beachhead segment already has CI elsewhere.
- "Free during beta" gives us a clean reason to gate access and curate the first cohort.
- Anything proposed for MVP that isn't on the list of six should be challenged against the wedge before being added.

## Alternatives considered

- **Feature-parity MVP** — multi-year project; competing on parity loses by definition.
- **Agent + thin repo viewer (no real Git host)** — Agent has nothing to integrate with; the wedge can't be demoed.
- **Include native CI in MVP** — adds ~6 months for table-stakes parity, not differentiation.
