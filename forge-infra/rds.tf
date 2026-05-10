// Postgres RDS for forge-platform + forge-agent. Single-AZ at MVP per
// ADR-0009; flip multi_az=true and bump backup retention before GA.

resource "aws_db_subnet_group" "this" {
  name       = "${local.prefix}-db"
  subnet_ids = aws_subnet.private[*].id
  tags       = { Name = "${local.prefix}-db" }
}

resource "aws_db_instance" "postgres" {
  identifier              = "${local.prefix}-pg"
  engine                  = "postgres"
  engine_version          = "16"
  instance_class          = var.db_instance_class
  allocated_storage       = 20
  max_allocated_storage   = 100
  db_name                 = "forge"
  username                = "forge"
  password                = var.db_password
  port                    = 5432
  publicly_accessible     = false
  multi_az                = false
  storage_encrypted       = true
  backup_retention_period = 7
  skip_final_snapshot     = true
  deletion_protection     = false
  apply_immediately       = true

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  tags = { Name = "${local.prefix}-pg" }
}
