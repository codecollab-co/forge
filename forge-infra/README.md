# forge-infra

Terraform for the AWS topology described in [ADR-0009](../docs/adr/0009-aws-deployment-topology.md).

## What this stack includes

- **VPC** (10.42.0.0/16), 2 public + 2 private subnets, single NAT gateway
- **RDS Postgres 16** (single-AZ at MVP, encrypted, 7-day backups)
- **ECS Fargate cluster** with services for `forge-platform`, `forge-agent`, `forge-web`
- **ALB** (HTTPS-only, plain-HTTP → 301 to HTTPS) with path-based routing between web and platform
- **NLB** on port 22 → forge-platform's :2222 for git-over-SSH
- **ECR repos** for each service image
- **Cloud Map service discovery** (`forge.local`) so services reach each other by name
- **CloudWatch log groups** per service
- **Two IAM roles** per task (execution + task)
- **Security groups** for ALB, NLB, ECS tasks, RDS

## What this stack deliberately omits (for v0.1)

- **ACM certificate** — bring your own ARN via `acm_certificate_arn`
- **Route53 records** — `cname` `var.domain_name` to the ALB DNS yourself
- **Multi-AZ RDS** — flip `multi_az = true` in `rds.tf` before GA
- **Autoscaling** — `desired_count = 1` everywhere
- **Secrets Manager rotation** — secrets passed via Terraform variables for now
- **WAF, CloudFront, S3 (LFS), KMS-managed state encryption** — all post-MVP

## First deploy

```bash
# Prereqs: AWS credentials, Terraform 1.9+, an ACM cert in your region

# 1. State backend (one-off — substitute your S3 bucket name).
cat > backend.tfvars <<EOF
bucket = "your-state-bucket"
key    = "forge/terraform.tfstate"
region = "us-east-1"
EOF
terraform init -backend-config=backend.tfvars

# 2. Configure variables. terraform.tfvars goes in .gitignore.
cat > terraform.tfvars <<EOF
domain_name         = "forge.example.com"
acm_certificate_arn = "arn:aws:acm:us-east-1:111122223333:certificate/<uuid>"
db_password         = "<32+ random chars>"
platform_image      = "<account>.dkr.ecr.us-east-1.amazonaws.com/forge-beta/forge-platform:v0.1.0"
agent_image         = "<account>.dkr.ecr.us-east-1.amazonaws.com/forge-beta/forge-agent:v0.1.0"
web_image           = "<account>.dkr.ecr.us-east-1.amazonaws.com/forge-beta/forge-web:v0.1.0"

jwt_private_key = <<EEOF
-----BEGIN PRIVATE KEY-----
MII...
-----END PRIVATE KEY-----
EEOF
jwt_public_key  = <<EEOF
-----BEGIN PUBLIC KEY-----
MII...
-----END PUBLIC KEY-----
EEOF

anthropic_api_key = "..."
e2b_api_key       = "..."
EOF

terraform plan
# Inspect carefully. First apply provisions ECR repos but ECS will fail
# to pull until images exist. That's expected.
terraform apply

# 3. Push the three container images to the ECR repos terraform created.
aws ecr get-login-password | docker login --username AWS --password-stdin \
  $(terraform output -raw ecr_platform | cut -d/ -f1)

for svc in platform agent web; do
  docker build -t forge-$svc:v0.1.0 ../forge-$svc
  remote=$(terraform output -raw ecr_$svc):v0.1.0
  docker tag forge-$svc:v0.1.0 $remote
  docker push $remote
done

# 4. Force a fresh deploy now that images exist.
aws ecs update-service --cluster $(terraform output -raw ecs_cluster) \
  --service forge-platform --force-new-deployment
# (repeat for forge-agent, forge-web)

# 5. Point DNS.
echo "CNAME $(terraform output -raw alb_dns_name) → forge.example.com"
```

## Cost estimate (us-east-1, idle)

- RDS db.t4g.small ............ ~$25/mo
- NAT gateway ................. ~$32/mo + data transfer
- ALB ......................... ~$16/mo + LCU
- NLB ......................... ~$16/mo + LCU
- ECS Fargate (3 tasks tiny) .. ~$30/mo
- CloudWatch logs (light) ..... ~$5/mo
- **Total** ~$120/mo idle, before agent runs (E2B + Anthropic billed separately)
