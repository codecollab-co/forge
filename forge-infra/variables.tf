variable "region" {
  description = "AWS region for the Forge deployment."
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Logical environment name (e.g. beta, staging, prod)."
  type        = string
  default     = "beta"
}
