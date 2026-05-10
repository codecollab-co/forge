# ADR 0010 — `forge` CLI scope

**Status:** Accepted — 2026-05-10

## Context

`git clone`, `git push`, etc. already work against Forge. The argument for shipping a CLI was repeatedly raised; the original recommendation was to defer it on the basis that the wedge (ADR-0001) is the agent loop, not a CLI. After three rounds of pushback the user committed to **option B — full GitHub-`gh`-style coverage**, accepting a 6–8 week scope and the displacement of remaining MVP-six work and agent-loop polish.

## Decision

Build `forge`, a Go binary, with `gh`-style coverage:

- `forge auth` — login / logout / status / refresh / token
- `forge repo` — create / clone / list / view / edit / delete
- `forge issue` — create / list / view / comment / close / reopen / edit / assign-agent
- `forge pr` — create / list / view / checkout / diff / comment / merge / close
- `forge run` — list / view / watch / cancel
- `forge browse` — open the current repo in a browser
- `forge api` — raw passthrough for scripting

Distribution:

- GitHub Releases per platform (macOS arm64/x86_64, Linux arm64/x86_64, Windows x86_64).
- Homebrew tap (`brew install codecollab-co/forge/forge`).
- `curl … | sh` install script for non-Homebrew Unix.
- Scoop manifest for Windows.

Auth: RFC 8628 device-code flow (see ADR-0011). Token stored at
`$XDG_CONFIG_HOME/forge/credentials.json` (file `0600`) for v0.1; OS keychain
integration in v0.2.

Output: TTY-aware coloring; `--json` flag on every read command for scripting.

Build process: TDD, one vertical slice per command. No horizontal slicing.

## Consequences

- Every API change in `forge-platform` is a coupled CLI release. Versioning matters.
- 6–8 weeks of focused work delays remaining MVP-six (SSH transport, PATs, visibility tightening) and agent-loop improvements (real E2B sandboxes, `run_shell`, multi-fixture eval set).
- We accept this displacement. The user explicitly chose option B over my recommendation of option 1 (agent-only, ~5 commands, ~1 week) and option C (phased option 1 → option 3).
- A minor, related risk: `forge` *as a CLI brand* may end up competing with the "Forge is a Git host" brand. Mitigated by keeping the CLI commands `gh`-shaped (familiar) rather than inventing new vocabulary.

## Alternatives considered

- **Option 1 (agent-only, ~5 commands).** Recommended path. Defers full coverage in favour of `forge agent assign`, `forge run watch`, `forge issue create`, plus auth. Rejected by user who chose option B.
- **Option 2 (thin wrapper around `git`).** Dishonest UX (`forge push` would just be `git push`). Rejected during grilling.
- **Option C (phased option 1 → option 3).** Ship option 1 first, grow command-by-command driven by demand. Rejected by user in favour of full coverage now.
