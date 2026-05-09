# ADR 0004 — Git server and storage

**Status:** Accepted — 2026-05-09

## Context

Forge needs a Git server. Three implementation strategies exist (wrap stock `git`, embed a Git library, reimplement the protocol on object storage), each tightly coupled to a storage choice (local disk / EFS / S3-backed). The wedge (ADR-0001) is the Agent loop, not Git internals — but Git correctness is non-negotiable, since incorrect Git would poison every Agent Run.

## Decision

- **Git server: wrap stock `git` binaries.** Serve `git-http-backend` and `git-upload-pack` / `git-receive-pack` behind our app, exec'ing the upstream `git` toolchain. No custom protocol implementation at MVP.
- **Storage at MVP: EBS-backed disk per Git node.** Bare repos live on a filesystem on attached block storage.
- **Migration path: S3-backed Git is a post-PMF project**, not an MVP one. The Git layer should be encapsulated behind a clear interface so the storage backend can be swapped later.

## Consequences

- Git correctness is inherited from upstream — we get years of edge-case fixes for free.
- Repo size and concurrent-clone throughput are bounded by a single node's disk and CPU. Acceptable for the beachhead segment (small teams, small repos).
- Horizontal scale requires sharding repos across nodes; we accept this as a known problem to solve later, not now.
- Self-hostable distribution (ADR-0001 stage 2) becomes easy — anyone with `git` and a disk can run it.
- We commit to *not* shipping Git-on-S3 as part of MVP, even if it would be more "scalable."

## Alternatives considered

- **Embed a Git library (libgit2 / go-git / JGit)** — more control, but we own every bug and gain little for the wedge. Possible later if we need protocol-level features.
- **Reimplement Git on S3 from day 1** — what GitHub and GitLab eventually moved to, but a multi-quarter project that pre-empts the actual product. CodeCommit took years to get this right and still felt awkward. Rejected for MVP.
