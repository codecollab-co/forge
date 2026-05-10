// Root module composition. Per ADR-0009: ECS Fargate + RDS Postgres + ALB
// + NLB(SSH) + S3 + Secrets Manager + CloudWatch.

provider "aws" {
  region = var.region
  default_tags {
    tags = merge(var.tags, {
      Project     = var.name
      Environment = var.environment
      ManagedBy   = "terraform"
    })
  }
}

locals {
  # Stable name prefix for everything we create.
  prefix = "${var.name}-${var.environment}"
}
