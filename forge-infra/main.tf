# Placeholder root module. The real AWS topology described in ADR-0009
# (VPC, ECS Fargate cluster, ALB, NLB for SSH, RDS Postgres, ElastiCache,
# S3, Secrets Manager) lands as the AWS-deploy half of issue #1.
#
# Local development does not require this module — `docker compose up`
# from the repo root brings up the full stack against a local Postgres.

provider "aws" {
  region = var.region
}
