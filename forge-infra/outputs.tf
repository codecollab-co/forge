output "alb_dns_name" {
  description = "Public DNS for the ALB. CNAME var.domain_name to this."
  value       = aws_lb.alb.dns_name
}

output "nlb_dns_name" {
  description = "Public DNS for the NLB (SSH). CNAME ssh.<domain> to this if you want a friendlier name; git over SSH works on the raw NLB hostname too."
  value       = aws_lb.nlb.dns_name
}

output "rds_endpoint" {
  description = "RDS endpoint (host:port). Internal only; not reachable from outside the VPC."
  value       = aws_db_instance.postgres.endpoint
}

output "ecr_platform" {
  description = "Push forge-platform images here."
  value       = aws_ecr_repository.platform.repository_url
}

output "ecr_agent" {
  value = aws_ecr_repository.agent.repository_url
}

output "ecr_web" {
  value = aws_ecr_repository.web.repository_url
}

output "vpc_id" {
  value = aws_vpc.this.id
}

output "ecs_cluster" {
  value = aws_ecs_cluster.this.name
}
