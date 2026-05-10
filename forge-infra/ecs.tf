// ECS Fargate cluster + one task definition + service per backend service.
// All three services run in the private subnets behind the ALB / NLB.
//
// forge-platform: 1024 cpu / 2048 mem; runs both HTTP (8080) and SSH (2222).
// forge-agent:    512 cpu / 1024 mem; consumes events from Postgres.
// forge-web:      512 cpu / 1024 mem; calls forge-platform internally.

resource "aws_ecs_cluster" "this" {
  name = "${local.prefix}-cluster"
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

locals {
  db_url = format(
    "postgres://forge:%s@%s:5432/forge?sslmode=require",
    var.db_password, aws_db_instance.postgres.address,
  )
  agent_db_url = format(
    "postgresql://forge:%s@%s:5432/forge?sslmode=require",
    var.db_password, aws_db_instance.postgres.address,
  )

  platform_env = [
    { name = "DATABASE_URL", value = local.db_url },
    { name = "WEBSITE_DOMAIN", value = "https://${var.domain_name}" },
    { name = "API_DOMAIN", value = "https://${var.domain_name}" },
    { name = "AGENT_PUBLIC_URL", value = "https://${var.domain_name}" },
    { name = "REPOS_DIR", value = "/var/lib/forge/repos" },
    { name = "SSH_ADDR", value = ":2222" },
    { name = "SSH_HOST_KEY", value = "/home/forge/host_ed25519" },
    { name = "JWT_PRIVATE_KEY", value = var.jwt_private_key },
    { name = "JWT_PUBLIC_KEY", value = var.jwt_public_key },
    { name = "PORT", value = "8080" },
  ]

  agent_env = [
    { name = "DATABASE_URL", value = local.agent_db_url },
    { name = "PLATFORM_API_URL", value = "http://forge-platform.forge.local:8080" },
    { name = "JWT_PRIVATE_KEY", value = var.jwt_private_key },
    { name = "JWT_PUBLIC_KEY", value = var.jwt_public_key },
    { name = "ANTHROPIC_API_KEY", value = var.anthropic_api_key },
    { name = "E2B_API_KEY", value = var.e2b_api_key },
    { name = "WEBSITE_DOMAIN", value = "https://${var.domain_name}" },
    { name = "PORT", value = "8081" },
  ]

  web_env = [
    { name = "PLATFORM_API_URL", value = "http://forge-platform.forge.local:8080" },
    { name = "NEXT_PUBLIC_PLATFORM_API_URL", value = "https://${var.domain_name}" },
    { name = "NEXT_PUBLIC_AGENT_API_URL", value = "https://${var.domain_name}" },
    { name = "NEXT_PUBLIC_WEBSITE_DOMAIN", value = "https://${var.domain_name}" },
    { name = "AGENT_API_URL", value = "http://forge-agent.forge.local:8081" },
  ]
}

// ---- Service discovery (Cloud Map) -------------------------------------
// Lets services reach each other via forge-platform.forge.local.

resource "aws_service_discovery_private_dns_namespace" "this" {
  name = "forge.local"
  vpc  = aws_vpc.this.id
}

resource "aws_service_discovery_service" "platform" {
  name = "forge-platform"
  dns_config {
    namespace_id   = aws_service_discovery_private_dns_namespace.this.id
    routing_policy = "MULTIVALUE"
    dns_records {
      type = "A"
      ttl  = 10
    }
  }
  health_check_custom_config { failure_threshold = 1 }
}

resource "aws_service_discovery_service" "agent" {
  name = "forge-agent"
  dns_config {
    namespace_id   = aws_service_discovery_private_dns_namespace.this.id
    routing_policy = "MULTIVALUE"
    dns_records {
      type = "A"
      ttl  = 10
    }
  }
  health_check_custom_config { failure_threshold = 1 }
}

// ---- forge-platform task ----------------------------------------------

resource "aws_ecs_task_definition" "platform" {
  family                   = "${local.prefix}-platform"
  cpu                      = 1024
  memory                   = 2048
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([
    {
      name      = "forge-platform"
      image     = var.platform_image
      essential = true
      portMappings = [
        { containerPort = 8080, protocol = "tcp" },
        { containerPort = 2222, protocol = "tcp" },
      ]
      environment = local.platform_env
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.platform.name
          "awslogs-region"        = var.region
          "awslogs-stream-prefix" = "platform"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "platform" {
  name            = "forge-platform"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.platform.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.tasks.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.platform.arn
    container_name   = "forge-platform"
    container_port   = 8080
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.ssh.arn
    container_name   = "forge-platform"
    container_port   = 2222
  }

  service_registries {
    registry_arn = aws_service_discovery_service.platform.arn
  }

  depends_on = [aws_lb_listener.https, aws_lb_listener.ssh]
}

// ---- forge-agent task --------------------------------------------------

resource "aws_ecs_task_definition" "agent" {
  family                   = "${local.prefix}-agent"
  cpu                      = 512
  memory                   = 1024
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([
    {
      name         = "forge-agent"
      image        = var.agent_image
      essential    = true
      portMappings = [{ containerPort = 8081, protocol = "tcp" }]
      environment  = local.agent_env
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.agent.name
          "awslogs-region"        = var.region
          "awslogs-stream-prefix" = "agent"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "agent" {
  name            = "forge-agent"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.agent.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.tasks.id]
  }

  service_registries {
    registry_arn = aws_service_discovery_service.agent.arn
  }
}

// ---- forge-web task ---------------------------------------------------

resource "aws_ecs_task_definition" "web" {
  family                   = "${local.prefix}-web"
  cpu                      = 512
  memory                   = 1024
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([
    {
      name         = "forge-web"
      image        = var.web_image
      essential    = true
      portMappings = [{ containerPort = 3000, protocol = "tcp" }]
      environment  = local.web_env
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.web.name
          "awslogs-region"        = var.region
          "awslogs-stream-prefix" = "web"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "web" {
  name            = "forge-web"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.web.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.tasks.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.web.arn
    container_name   = "forge-web"
    container_port   = 3000
  }

  depends_on = [aws_lb_listener.https]
}
