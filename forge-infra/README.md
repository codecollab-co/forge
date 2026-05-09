# forge-infra

Terraform for the AWS topology described in [ADR-0009](../docs/adr/0009-aws-deployment-topology.md):

- ECS Fargate for `forge-agent` and `forge-web`; ECS (Fargate or EC2-backed, TBD in #1) for `forge-platform` due to stateful Git storage on EBS.
- ALB in front of `forge-platform` and `forge-web`; NLB on port 22 for SSH ingress to `forge-platform`.
- RDS Postgres (single-AZ at MVP, multi-AZ before GA).
- ElastiCache Redis.
- S3 for Run artifacts, attachments, future LFS objects.
- Secrets Manager for service secrets.
- CloudWatch logs + metrics; third-party tracing TBD.

See [`../ARCHITECTURE.md`](../ARCHITECTURE.md).

## Status

Pre-implementation. Module layout and root-module composition land as part of issue #1.
