// One ALB fronts both forge-web (root) and forge-platform (host-based or
// path-based — at MVP we route everything except /api/* to web). Caller
// supplies the ACM cert; this stack doesn't manage Route53.

resource "aws_lb" "alb" {
  name               = "${local.prefix}-alb"
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id
  idle_timeout       = 120
}

resource "aws_lb_target_group" "web" {
  name        = "${local.prefix}-web"
  port        = 3000
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.this.id

  health_check {
    path                = "/"
    matcher             = "200-399"
    interval            = 15
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_target_group" "platform" {
  name        = "${local.prefix}-plat"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.this.id

  health_check {
    path                = "/healthz"
    matcher             = "200"
    interval            = 15
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.alb.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.acm_certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.web.arn
  }
}

// Plain HTTP -> 301 to HTTPS.
resource "aws_lb_listener" "http_redirect" {
  load_balancer_arn = aws_lb.alb.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

// Anything starting with /api/, /auth/, /repos/, /me/, /oauth/, /internal/,
// /healthz, /config, /licenses, /gitignore, /runs/ → forge-platform.
// (Crude until we move to api.<domain> as a separate hostname.)
resource "aws_lb_listener_rule" "platform" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.platform.arn
  }

  condition {
    path_pattern {
      values = [
        "/api/*", "/auth/*", "/repos/*", "/me/*", "/oauth/*",
        "/internal/*", "/healthz", "/config",
        "/licenses", "/gitignore", "/runs/*",
      ]
    }
  }
}
