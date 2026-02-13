# CloudWatch Logs for scheduled one-shot refresh task
resource "aws_cloudwatch_log_group" "news_refresh" {
  count = var.enable_scheduled_news_refresh ? 1 : 0

  name              = "/ecs/${var.app_name}-news-refresh"
  retention_in_days = 14

  tags = {
    Name        = "${var.app_name}-news-refresh-logs"
    Component   = "news-refresh"
    CostProfile = "scheduled-ingest"
  }
}

# ECS Task Definition used by EventBridge Scheduler to refresh feeds once per run.
resource "aws_ecs_task_definition" "news_refresh" {
  count = var.enable_scheduled_news_refresh ? 1 : 0

  family                   = "${var.app_name}-news-refresh"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.news_refresh_cpu
  memory                   = var.news_refresh_memory
  execution_role_arn       = aws_iam_role.ecs_task_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name      = "news-refresh"
      image     = "${aws_ecr_repository.server.repository_url}:latest"
      essential = true
      command   = ["./flyingforge", "-refresh-once"]

      environment = [
        {
          name  = "LOG_LEVEL"
          value = "info"
        },
        {
          name  = "AWS_REGION"
          value = var.aws_region
        },
        {
          name  = "DB_HOST"
          value = aws_db_instance.main.address
        },
        {
          name  = "DB_PORT"
          value = "5432"
        },
        {
          name  = "DB_USER"
          value = var.db_username
        },
        {
          name  = "DB_NAME"
          value = var.db_name
        },
        {
          name  = "DB_SSLMODE"
          value = "require"
        },
        {
          name  = "CACHE_BACKEND"
          value = "redis"
        },
        {
          name  = "REDIS_ADDR"
          value = "${aws_elasticache_cluster.main.cache_nodes[0].address}:6379"
        },
        {
          name  = "RATE_LIMIT"
          value = "1s"
        }
      ]

      secrets = [
        {
          name      = "DB_PASSWORD"
          valueFrom = aws_secretsmanager_secret.db_password.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.news_refresh[0].name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name        = "${var.app_name}-news-refresh-task"
    Component   = "news-refresh"
    CostProfile = "scheduled-ingest"
  }
}

data "aws_iam_policy_document" "scheduler_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "news_refresh_scheduler" {
  count = var.enable_scheduled_news_refresh ? 1 : 0

  name               = "${var.app_name}-news-refresh-scheduler"
  assume_role_policy = data.aws_iam_policy_document.scheduler_assume_role.json

  tags = {
    Name        = "${var.app_name}-news-refresh-scheduler"
    Component   = "news-refresh"
    CostProfile = "scheduled-ingest"
  }
}

data "aws_iam_policy_document" "news_refresh_scheduler" {
  statement {
    sid    = "RunRefreshTask"
    effect = "Allow"

    actions = [
      "ecs:RunTask"
    ]
    resources = [
      aws_ecs_task_definition.news_refresh[0].arn,
      "${aws_ecs_task_definition.news_refresh[0].arn_without_revision}:*"
    ]

    condition {
      test     = "ArnEquals"
      variable = "ecs:cluster"
      values   = [aws_ecs_cluster.main.arn]
    }
  }

  statement {
    sid    = "PassTaskRoles"
    effect = "Allow"

    actions = [
      "iam:PassRole"
    ]
    resources = [
      aws_iam_role.ecs_task_execution.arn,
      aws_iam_role.ecs_task.arn
    ]
  }
}

resource "aws_iam_role_policy" "news_refresh_scheduler" {
  count = var.enable_scheduled_news_refresh ? 1 : 0

  name   = "${var.app_name}-news-refresh-scheduler"
  role   = aws_iam_role.news_refresh_scheduler[0].id
  policy = data.aws_iam_policy_document.news_refresh_scheduler.json
}

resource "aws_scheduler_schedule" "news_refresh" {
  count = var.enable_scheduled_news_refresh ? 1 : 0

  name                         = "${var.app_name}-news-refresh-4pm"
  description                  = "Run one-shot news refresh task daily at 4 PM"
  schedule_expression          = var.news_refresh_schedule_expression
  schedule_expression_timezone = var.news_refresh_schedule_timezone
  state                        = "ENABLED"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_ecs_cluster.main.arn
    role_arn = aws_iam_role.news_refresh_scheduler[0].arn

    ecs_parameters {
      task_definition_arn = aws_ecs_task_definition.news_refresh[0].arn
      launch_type         = "FARGATE"
      platform_version    = "LATEST"

      network_configuration {
        subnets          = aws_subnet.private[*].id
        security_groups  = [aws_security_group.ecs_tasks.id]
        assign_public_ip = false
      }
    }

    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 1
    }
  }
}
