variable "region" {
  description = "AWS region for the Forge deployment."
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Logical environment name. Used as a tag and resource-name suffix."
  type        = string
  default     = "beta"
}

variable "name" {
  description = "Project name. Resources get prefixed with this."
  type        = string
  default     = "forge"
}

variable "vpc_cidr" {
  description = "VPC CIDR. /16 leaves room for many subnets."
  type        = string
  default     = "10.42.0.0/16"
}

variable "az_count" {
  description = "Number of AZs to span (subnets are created in each)."
  type        = number
  default     = 2
}

variable "domain_name" {
  description = "Public domain for the web app, e.g. forge.example.com. Used in ECS env vars; Route53/ACM are not managed by this stack."
  type        = string
}

variable "acm_certificate_arn" {
  description = "ACM cert ARN for the ALB HTTPS listener. Issue + validate it separately, then pass it in."
  type        = string
}

variable "db_password" {
  description = "Initial RDS master password. Move to Secrets Manager rotation post-MVP."
  type        = string
  sensitive   = true
}

variable "db_instance_class" {
  description = "RDS instance class."
  type        = string
  default     = "db.t4g.small"
}

variable "platform_image" {
  description = "Container image for forge-platform (e.g. <account>.dkr.ecr.<region>.amazonaws.com/forge-platform:v0.1.0)."
  type        = string
}

variable "agent_image" {
  description = "Container image for forge-agent."
  type        = string
}

variable "web_image" {
  description = "Container image for forge-web."
  type        = string
}

variable "anthropic_api_key" {
  description = "Anthropic API key for the agent. Pass via Secrets Manager once we wire that up."
  type        = string
  default     = ""
  sensitive   = true
}

variable "e2b_api_key" {
  description = "E2B API key for the agent's sandbox provider."
  type        = string
  default     = ""
  sensitive   = true
}

variable "jwt_private_key" {
  description = "RS256 private key (PEM) used by forge-platform to sign session + s2s tokens."
  type        = string
  sensitive   = true
}

variable "jwt_public_key" {
  description = "Public counterpart of jwt_private_key, for verifiers."
  type        = string
  sensitive   = true
}

variable "tags" {
  description = "Extra tags applied to every taggable resource."
  type        = map(string)
  default     = {}
}
