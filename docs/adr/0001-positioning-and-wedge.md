# ADR 0001 — Positioning and wedge

**Status:** Accepted — 2026-05-09

## Context

The project is "a GitHub-like platform on AWS." That description spans at least three different products (public SaaS competitor, self-hosted enterprise, learning project) and several possible differentiation axes (price, integrated DevOps, AI-native, sovereignty, OSS, vertical, workflow UX). Without a sharp pick, every later decision becomes ambiguous.

## Decision

- **Product mode:** public SaaS first, self-hostable enterprise as stage 2.
- **Wedge:** AI-native Git host — specifically **AI-as-author (agent assigned to an Issue produces a PR)**, with AI-as-reviewer as the default companion feature.
- **Beachhead segment:** small teams already heavily using AI coding tools (Cursor, Devin, Aider). Expansion: small startups (2–10 devs).
- **Explicitly rejected wedges:** price ("cheaper GitHub" — graveyard), integrated DevOps (GitLab owns it), AI-as-search (Sourcegraph owns it), AI-as-IDE (Cursor owns it), AI-as-infrastructure (no distribution).

## Consequences

- The product is judged by how good the agent loop is, not by feature parity with GitHub. Feature gaps vs. GitHub are acceptable; a mediocre agent is not.
- The beachhead is small and loud. Distribution is Twitter/HN, not enterprise sales. Marketing voice and pricing must match.
- Self-hostable as stage 2 constrains every architectural choice: anything that *only* works at SaaS scale (e.g., proprietary managed-only infra) is a future migration cost.
- "Cheaper than GitHub" is not a marketing line we will use, even if true.

## Alternatives considered

- **Self-hosted enterprise first** — slower wedge, no consumer pull, harder to learn from users. Rejected for stage 1; revisited in stage 2.
- **Multiple wedges (AI + OSS + vertical + workflow)** — explicitly rejected. Four wedges is zero wedges.
- **AI-as-reviewer as the headline** — too narrow; CodeRabbit/Greptile already occupy it. Kept as companion feature instead.
