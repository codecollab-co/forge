# ADR 0006 — LLM strategy

**Status:** Accepted — 2026-05-09

## Context

The Agent's intelligence is the wedge (ADR-0001). Two coupled choices: which model(s) power it, and who pays for the tokens. Both have long lock-in tails — model integrations and billing models are painful to retrofit.

## Decision

- **Model selection: multi-model with a thin router.** All model calls go through a single internal `ModelClient` interface. **Claude is the default** for agentic coding tasks at MVP; the interface allows swapping or routing per-task without code changes elsewhere.
- **Billing: hybrid.**
  - **Bundled tier (default):** subscription includes a monthly token allowance on platform-paid models.
  - **BYOK:** users (especially power users and enterprise) can attach their own Anthropic / OpenAI key.
  - Free closed beta uses platform-paid Claude with rate limits.
- **Not on the table at MVP:** open-weights / self-hosted models (quality gap for agentic coding); single-vendor lock-in (model leadership shifts too fast).

## Consequences

- All model calls must go through `ModelClient` — direct SDK imports outside that module are forbidden.
- We can chase model price/perf shifts without rewriting the agent loop.
- Bundled billing means we eat token-cost volatility on the default tier — must be priced with margin headroom.
- BYOK unlocks the path to ADR-0001 stage-2 enterprise self-host (where customers will not route code through our model account).
- "Why is Claude the default?" is answered by current agentic-coding benchmarks; this is a re-evaluable default, not a permanent choice.

## Alternatives considered

- **Single-model lock-in (Claude-only or GPT-only)** — simpler, slightly better quality from deep integration, but exposes us to leadership shifts. Rejected.
- **Open-weights self-hosted** — best margins at scale, but current open coding agents lag frontier closed models. Revisit in ~12 months.
- **BYOK-only** — zero token risk for us, but no product control and a weak "AI-native" claim if we're just a passthrough.
- **Bundled-only** — clean UX, but enterprise stage-2 will not accept it. BYOK has to exist eventually; better to design it in now.
