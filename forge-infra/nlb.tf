// NLB on port 22 for git-over-SSH. Forwards to forge-platform's :2222.
// IP target group; the platform service registers itself.

resource "aws_lb" "nlb" {
  name               = "${local.prefix}-nlb"
  load_balancer_type = "network"
  subnets            = aws_subnet.public[*].id
  security_groups    = [aws_security_group.nlb.id]
}

resource "aws_lb_target_group" "ssh" {
  name        = "${local.prefix}-ssh"
  port        = 2222
  protocol    = "TCP"
  target_type = "ip"
  vpc_id      = aws_vpc.this.id

  health_check {
    protocol            = "TCP"
    interval            = 30
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }
}

resource "aws_lb_listener" "ssh" {
  load_balancer_arn = aws_lb.nlb.arn
  port              = 22
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ssh.arn
  }
}
