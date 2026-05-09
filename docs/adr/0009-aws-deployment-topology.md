# ADR 0009 — AWS deployment topology (MVP)

**Status:** Accepted — 2026-05-09

## Context

The project is committed to AWS. We need a topology that survives a closed beta (and a viral demo) without dragging in operational weight that pre-empts product work. We also need it to map cleanly onto a stage-2 self-hostable distribution (ADR-0001), so anything AWS-proprietary at the application layer is suspect.

## Decision

- **Compute: ECS Fargate** for `agent-orchestrator` and `web`; ECS on EC2 (or Fargate with attached EFS — TBD during build) for `platform-api` because Git storage on EBS is stateful.
- **Load balancer: ALB** in front of `platform-api` and `web`; SSH ingress on an NLB (port 22) routed to `platform-api`.
- **Database: RDS Postgres** (single-AZ at MVP, multi-AZ before GA). Single instance; logical schemas split between `platform.*` (Go) and `agent.*` (Python).
- **Cache / queue: ElastiCache Redis** (Redis Streams as the inter-service queue at MVP); migrate to SQS if/when fan-out grows.
- **Object store: S3** for LFS objects, attachments, and agent Run artifacts/logs.
- **Secrets: AWS Secrets Manager.**
- **Observability: CloudWatch logs + metrics; third-party tracing (Axiom or Better Stack) for app-level traces.**
- **Git storage: EBS** attached to `platform-api` nodes (per ADR-0004). MVP runs a small fixed number of Git nodes; sharding-by-repo is a known follow-up, not MVP work.
- **Sandboxes: out-of-AWS** via E2B / Modal (per ADR-0005), called over the public internet at MVP.

Explicitly **not** at MVP: EKS, Lambda for any hot path, Aurora Serverless, multi-region, CDN beyond CloudFront for static assets.

## Consequences

- Idle cost is in the low-hundreds-of-dollars/month range; the variable cost is sandbox provider usage and LLM tokens.
- `platform-api` is stateful; horizontal scaling is bounded until repo-sharding lands. Acceptable for MVP user volumes.
- All other services scale horizontally on Fargate without ceremony.
- Stage-2 self-host: the same containers run on customer ECS, EKS, or `docker compose`. Anything that *only* works on Fargate (e.g., Fargate-specific networking) is forbidden in app code.
- We accept the latency of calling sandbox providers over the public internet during MVP. If it becomes a UX problem, the answer is the in-house Firecracker fleet (ADR-0005), not VPC peering with the vendor.

## Alternatives considered

- **Single EC2 box** — fastest to start, fragile, no scaling story. Rejected.
- **EKS from day 1** — correct eventually, premature now; ops tax is real. Revisit when we have ≥5 services or a platform team.
- **Lambda + API Gateway + Aurora Serverless** — cold starts and the long-lived nature of Git/SSH connections make Lambda a bad fit on the hot path.
- **Fly.io / Railway / Render** — operationally simpler, but conflicts with the AWS commitment and the stage-2 customer expectation of AWS-native artifacts.
