locals {
  interface_endpoint_subnet_ids = length(var.vpc_endpoint_subnet_ids) > 0 ? var.vpc_endpoint_subnet_ids : aws_subnet.private[*].id

  ecr_repository_arns = concat([
    aws_ecr_repository.server.arn,
    aws_ecr_repository.web.arn
  ], var.additional_ecr_repository_arns)

  secretsmanager_secret_arns = concat([
    aws_secretsmanager_secret.app_secrets.arn,
    aws_secretsmanager_secret.database_url.arn,
    aws_secretsmanager_secret.redis_url.arn,
    aws_secretsmanager_secret.db_password.arn
  ], var.additional_secretsmanager_secret_arns)

  managed_log_group_arns = compact([
    aws_cloudwatch_log_group.server.arn,
    # The scheduled refresh task uses its own log group; include it when enabled so
    # the CloudWatch Logs VPC endpoint policy allows CreateLogStream/PutLogEvents.
    try(aws_cloudwatch_log_group.news_refresh[0].arn, ""),
  ])

  additional_cloudwatch_log_group_arns_with_streams = flatten([
    for arn in var.additional_cloudwatch_log_group_arns : [arn, "${arn}:*"]
  ])

  cloudwatch_log_resource_arns = concat(
    flatten([
      for arn in local.managed_log_group_arns : [arn, "${arn}:*"]
    ]),
    local.additional_cloudwatch_log_group_arns_with_streams
  )

  s3_gateway_bucket_arns = concat(
    ["arn:aws:s3:::prod-${var.aws_region}-starport-layer-bucket"],
    var.additional_s3_gateway_bucket_arns
  )

  interface_endpoint_definitions = merge(
    {
      ecr_api = {
        service_name = "com.amazonaws.${var.aws_region}.ecr.api"
        policy       = data.aws_iam_policy_document.vpc_endpoint_ecr.json
      }
      ecr_dkr = {
        service_name = "com.amazonaws.${var.aws_region}.ecr.dkr"
        policy       = data.aws_iam_policy_document.vpc_endpoint_ecr.json
      }
      logs = {
        service_name = "com.amazonaws.${var.aws_region}.logs"
        policy       = data.aws_iam_policy_document.vpc_endpoint_logs.json
      }
      secretsmanager = {
        service_name = "com.amazonaws.${var.aws_region}.secretsmanager"
        policy       = data.aws_iam_policy_document.vpc_endpoint_secretsmanager.json
      }
      rekognition = {
        service_name = "com.amazonaws.${var.aws_region}.rekognition"
        policy       = data.aws_iam_policy_document.vpc_endpoint_rekognition.json
      }
      sts = {
        service_name = "com.amazonaws.${var.aws_region}.sts"
        policy       = data.aws_iam_policy_document.vpc_endpoint_sts.json
      }
    },
    var.enable_kms_vpc_endpoint ? {
      kms = {
        service_name = "com.amazonaws.${var.aws_region}.kms"
        policy       = data.aws_iam_policy_document.vpc_endpoint_kms.json
      }
    } : {}
  )
}

# NOTE: Endpoint policies below use wildcard principals ("*"). This is intentional:
# - Interface endpoints are only reachable from within the VPC and are further restricted by the endpoint SG
#   (ingress 443 from the ECS tasks security group).
# - The S3 gateway endpoint is route-table scoped to the private subnets and is least-privilege restricted by bucket ARNs.
data "aws_iam_policy_document" "vpc_endpoint_s3" {
  statement {
    sid    = "AllowLayerDownloads"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = ["s3:GetObject"]
    resources = [
      for arn in local.s3_gateway_bucket_arns : "${arn}/*"
    ]
  }

  statement {
    sid    = "AllowLayerBucketListing"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions   = ["s3:ListBucket"]
    resources = local.s3_gateway_bucket_arns
  }
}

data "aws_iam_policy_document" "vpc_endpoint_ecr" {
  statement {
    sid    = "AllowECRAuthorization"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }

  statement {
    sid    = "AllowECRImagePull"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:DescribeImages",
      "ecr:GetDownloadUrlForLayer"
    ]
    resources = local.ecr_repository_arns
  }
}

data "aws_iam_policy_document" "vpc_endpoint_logs" {
  statement {
    sid    = "AllowTaskLogIngestion"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "logs:CreateLogStream",
      "logs:DescribeLogStreams",
      "logs:PutLogEvents"
    ]
    resources = local.cloudwatch_log_resource_arns
  }
}

data "aws_iam_policy_document" "vpc_endpoint_secretsmanager" {
  statement {
    sid    = "AllowReadAppSecrets"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "secretsmanager:DescribeSecret",
      "secretsmanager:GetSecretValue"
    ]
    resources = local.secretsmanager_secret_arns
  }
}

data "aws_iam_policy_document" "vpc_endpoint_sts" {
  statement {
    sid    = "AllowSTSIdentityAndRoleAssumption"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "sts:AssumeRole",
      "sts:AssumeRoleWithWebIdentity",
      "sts:GetCallerIdentity"
    ]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "vpc_endpoint_rekognition" {
  statement {
    sid    = "AllowModerationCalls"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "rekognition:DetectModerationLabels"
    ]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "vpc_endpoint_kms" {
  statement {
    sid    = "AllowKMSCryptoOperations"
    effect = "Allow"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "kms:Decrypt",
      "kms:DescribeKey",
      "kms:Encrypt",
      "kms:GenerateDataKey"
    ]
    resources = ["*"]
  }
}

resource "aws_security_group" "vpc_endpoints" {
  count = var.enable_vpc_endpoints ? 1 : 0

  name        = "${var.app_name}-vpce-sg"
  description = "Security group for interface VPC endpoints"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "HTTPS from ECS tasks"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
  }

  egress {
    description = "Allow responses within VPC"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  tags = {
    Name        = "${var.app_name}-vpce-sg"
    Component   = "networking"
    CostProfile = "nat-bypass"
  }
}

resource "aws_vpc_endpoint" "s3" {
  count = var.enable_vpc_endpoints ? 1 : 0

  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.${var.aws_region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = aws_route_table.private[*].id
  policy            = data.aws_iam_policy_document.vpc_endpoint_s3.json

  tags = {
    Name        = "${var.app_name}-s3-vpce"
    Component   = "networking"
    CostProfile = "nat-bypass"
  }
}

resource "aws_vpc_endpoint" "interface" {
  for_each = var.enable_vpc_endpoints ? local.interface_endpoint_definitions : {}

  vpc_id              = aws_vpc.main.id
  service_name        = each.value.service_name
  vpc_endpoint_type   = "Interface"
  subnet_ids          = local.interface_endpoint_subnet_ids
  security_group_ids  = [aws_security_group.vpc_endpoints[0].id]
  private_dns_enabled = true
  policy              = each.value.policy

  tags = {
    Name        = "${var.app_name}-${replace(each.key, "_", "-")}-vpce"
    Component   = "networking"
    CostProfile = "nat-bypass"
  }
}
