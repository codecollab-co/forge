resource "aws_cloudwatch_log_group" "platform" {
  name              = "/ecs/${local.prefix}/forge-platform"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "agent" {
  name              = "/ecs/${local.prefix}/forge-agent"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "web" {
  name              = "/ecs/${local.prefix}/forge-web"
  retention_in_days = 14
}
