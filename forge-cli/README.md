# forge-cli

`forge` — CLI for [Forge](https://github.com/codecollab-co/forge).

## Install

**macOS / Linux (Homebrew):**

```bash
brew install codecollab-co/forge/forge
```

**curl | sh:**

```bash
curl -fsSL https://raw.githubusercontent.com/codecollab-co/forge/main/scripts/install-forge.sh | sh
```

The install script downloads the latest GitHub release, verifies the
sha256 against the published `checksums.txt`, and installs to
`/usr/local/bin/forge` (override with `FORGE_INSTALL_DIR=$HOME/.local/bin`).
Pin a version with `FORGE_VERSION=v0.1.0`.

**Windows (Scoop):**

```powershell
scoop bucket add forge https://github.com/codecollab-co/scoop-forge
scoop install forge
```

**From source:**

```bash
go install github.com/codecollab-co/forge/forge-cli/cmd/forge@latest
```

> **macOS note:** binaries aren't notarised yet (ADR-0012). On first
> run you'll see a Gatekeeper warning — right-click the `forge`
> binary in Finder and choose **Open** once, or run
> `xattr -d com.apple.quarantine $(which forge)`.

## First-run

```bash
forge auth login --api-url https://forge.example.com
```

This starts the RFC 8628 device-code flow: open the URL it prints,
sign in to Forge, enter the user code, return to the terminal.

## Commands

| Group   | Subcommands |
|---------|-------------|
| `auth`  | `login`, `status`, `logout` |
| `repo`  | `list`, `view`, `create`, `edit`, `delete`, `clone` |
| `issue` | `list`, `view`, `create`, `comment`, `close`, `reopen`, `assign-agent` |
| `pr`    | `list`, `view`, `create`, `comment`, `merge` |
| `run`   | `list`, `view`, `watch`, `cancel` |
| `browse`| open a repo / issue / PR / run page |
| `api`   | raw passthrough for scripting |

`--help` works at every level.

## Verify the binary

Releases are signed via [Sigstore Cosign](https://docs.sigstore.dev/) using
GitHub OIDC. Verify any artefact with:

```bash
cosign verify-blob \
  --certificate-identity-regexp "https://github.com/codecollab-co/forge/.github/workflows/release-cli.yml@refs/tags/v.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature forge_<version>_<os>_<arch>.tar.gz.sig \
  forge_<version>_<os>_<arch>.tar.gz
```

## Tests

```bash
go test ./...
```

Black-box `cli.Run(args, stdout, stderr)` integration tests against
`httptest.Server`. No model API calls; no real network.
