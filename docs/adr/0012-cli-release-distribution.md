# ADR 0012 — `forge` CLI release & distribution

**Status:** Accepted — 2026-05-10

## Context

ADR-0010 commits us to releasing `forge` via Homebrew, `curl … | sh`, and per-OS archives across macOS arm64/x86_64, Linux arm64/x86_64, and Windows x86_64. With the CLI feature-complete for v0.1, we need a release pipeline that produces those artefacts deterministically on every tag push, and an install path for each channel.

## Decision

- **Release tool: GoReleaser.** Single config drives the multi-arch builds, archives, checksums, Homebrew formula update, and Scoop manifest update.
- **Trigger: tag push (`v*`).** A GitHub Actions workflow runs `goreleaser release` on every `v*.*.*` tag.
- **Channels:**
  - **GitHub Releases** — primary channel. All tarballs + checksums live here.
  - **Homebrew tap** — `codecollab-co/homebrew-tap` (the existing org-level tap shared with other CodeCollab tools). GoReleaser writes `Formula/forge.rb` to that repo on each tag. Install: `brew install codecollab-co/tap/forge`.
  - **`curl … | sh` install script** — checked into this repo at `scripts/install-forge.sh`, fetches the latest GitHub release, verifies sha256 against the published checksums file, installs to `/usr/local/bin` (or `$FORGE_INSTALL_DIR`).
  - **Scoop bucket** — `codecollab-co/scoop-forge` for Windows. Auto-updated by GoReleaser. Install: `scoop install forge`.
- **Signing:**
  - **Cosign keyless signatures** on every artefact, using GitHub OIDC. Verifiable with `cosign verify-blob`.
  - **macOS notarization deliberately deferred** until we have an Apple Developer account ($99/year + cert provisioning). Until then macOS users will see a Gatekeeper warning the first time they run `forge` and need to right-click → Open. Documented in install instructions.
- **Versioning:** plain semver via Git tags. `v0.1.0` first; reserve `v0.x.x` for pre-1.0 breaking changes.

## Consequences

- The existing `codecollab-co/homebrew-tap` is reused — sharing one tap across all CodeCollab tools means users add the tap once. GoReleaser pushes `Formula/forge.rb` on each release using a `HOMEBREW_TAP_GITHUB_TOKEN` secret with `contents: write` on the tap repo.
- A new `codecollab-co/scoop-forge` repo is created for Windows (Scoop's convention is one bucket per tool family).
- Cosign keyless signatures don't require us to manage signing keys, but they do require Sigstore's public infrastructure to be available. Acceptable tradeoff for v0.1.
- macOS users get a worse first-run UX than `gh` users. Acceptable until volume justifies the Apple Developer account.
- Every release auto-publishes to all channels — there's no staging tier. Consider a `pre-release` flag in GoReleaser when we want to ship release candidates.

## Alternatives considered

- **Hand-rolled release script** — reinvents what GoReleaser solves. Rejected.
- **Manual Homebrew formula updates** — error-prone, no atomicity with the release. Rejected.
- **Notarisation now via fastlane / rcodesign** — possible but each path needs paid Apple infra. Deferred.
- **Snap / Flatpak / .deb / .rpm** — Linux distribution beyond the static tarball. Punted; the install script + apt-installable .deb is a follow-up if Linux users complain.
