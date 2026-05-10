// One ECR repo per service. Push images here from CI; image tags become
// the platform_image / agent_image / web_image variables.

resource "aws_ecr_repository" "platform" {
  name                 = "${local.prefix}/forge-platform"
  image_tag_mutability = "IMMUTABLE"
  image_scanning_configuration { scan_on_push = true }
}

resource "aws_ecr_repository" "agent" {
  name                 = "${local.prefix}/forge-agent"
  image_tag_mutability = "IMMUTABLE"
  image_scanning_configuration { scan_on_push = true }
}

resource "aws_ecr_repository" "web" {
  name                 = "${local.prefix}/forge-web"
  image_tag_mutability = "IMMUTABLE"
  image_scanning_configuration { scan_on_push = true }
}
