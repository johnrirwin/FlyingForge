# Application Secrets
resource "aws_secretsmanager_secret" "app_secrets" {
  name                    = "${var.app_name}/app-secrets"
  recovery_window_in_days = 7

  tags = {
    Name = "${var.app_name}-app-secrets"
  }
}

resource "aws_secretsmanager_secret_version" "app_secrets" {
  secret_id = aws_secretsmanager_secret.app_secrets.id
  secret_string = jsonencode({
    GOOGLE_CLIENT_ID     = var.google_client_id
    GOOGLE_CLIENT_SECRET = var.google_client_secret
    ENCRYPTION_KEY       = var.encryption_key
  })
}

# Database URL secret (constructed from RDS output)
resource "aws_secretsmanager_secret" "database_url" {
  name                    = "${var.app_name}/database-url"
  recovery_window_in_days = 7

  tags = {
    Name = "${var.app_name}-database-url"
  }
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id     = aws_secretsmanager_secret.database_url.id
  secret_string = "postgres://${var.db_username}:${random_password.db_password.result}@${aws_db_instance.main.endpoint}/${var.db_name}?sslmode=require"
}

# Redis URL secret
resource "aws_secretsmanager_secret" "redis_url" {
  name                    = "${var.app_name}/redis-url"
  recovery_window_in_days = 7

  tags = {
    Name = "${var.app_name}-redis-url"
  }
}

resource "aws_secretsmanager_secret_version" "redis_url" {
  secret_id     = aws_secretsmanager_secret.redis_url.id
  secret_string = "redis://${aws_elasticache_cluster.main.cache_nodes[0].address}:6379"
}
