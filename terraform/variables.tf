variable "aws_region" {
  description = "AWS region to deploy to"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (e.g., production, staging)"
  type        = string
  default     = "production"
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "flyingforge"
}

# Database
variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.micro"
}

variable "db_allocated_storage" {
  description = "RDS allocated storage in GB"
  type        = number
  default     = 20
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "flyingforge"
}

variable "db_username" {
  description = "Database master username"
  type        = string
  default     = "flyingforge"
  sensitive   = true
}

# Redis
variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.micro"
}

# ECS
variable "server_cpu" {
  description = "CPU units for server task (1024 = 1 vCPU)"
  type        = number
  default     = 512
}

variable "server_memory" {
  description = "Memory for server task in MB"
  type        = number
  default     = 1024
}

variable "web_cpu" {
  description = "CPU units for web task"
  type        = number
  default     = 256
}

variable "web_memory" {
  description = "Memory for web task in MB"
  type        = number
  default     = 512
}

variable "server_desired_count" {
  description = "Desired number of server tasks"
  type        = number
  default     = 1
}

variable "web_desired_count" {
  description = "Desired number of web tasks"
  type        = number
  default     = 1
}

variable "news_refresh_cpu" {
  description = "CPU units for the scheduled one-shot news refresh task"
  type        = number
  default     = 256
}

variable "news_refresh_memory" {
  description = "Memory for the scheduled one-shot news refresh task in MB"
  type        = number
  default     = 512
}

variable "enable_scheduled_news_refresh" {
  description = "Whether to run scheduled one-shot ECS tasks to refresh news feeds"
  type        = bool
  default     = true
}

variable "news_refresh_schedule_expression" {
  description = "EventBridge Scheduler expression for the daily news refresh task"
  type        = string
  default     = "cron(0 16 * * ? *)"
}

variable "news_refresh_schedule_timezone" {
  description = "Timezone used by the EventBridge Scheduler news refresh task"
  type        = string
  default     = "America/New_York"
}

# Networking cost controls
variable "enable_nat_gateway" {
  description = "Whether to provision NAT gateways for private subnet internet egress"
  type        = bool
  default     = false
}

# NAT cost optimization (AWS service access without traversing NAT)
variable "enable_vpc_endpoints" {
  description = "Whether to create VPC endpoints for private subnet AWS service traffic"
  type        = bool
  default     = true
}

variable "enable_kms_vpc_endpoint" {
  description = "Whether to create an interface VPC endpoint for AWS KMS"
  type        = bool
  default     = true
}

variable "vpc_endpoint_subnet_ids" {
  description = "Optional subnet IDs for interface VPC endpoints (defaults to private subnets)"
  type        = list(string)
  default     = []
}

variable "additional_ecr_repository_arns" {
  description = "Additional ECR repository ARNs allowed by the ECR VPC endpoint policy"
  type        = list(string)
  default     = []
}

variable "additional_cloudwatch_log_group_arns" {
  description = "Additional CloudWatch log group ARNs allowed by the Logs VPC endpoint policy"
  type        = list(string)
  default     = []
}

variable "additional_secretsmanager_secret_arns" {
  description = "Additional Secrets Manager secret ARNs allowed by the Secrets Manager VPC endpoint policy"
  type        = list(string)
  default     = []
}

variable "additional_s3_gateway_bucket_arns" {
  description = "Additional S3 bucket ARNs (no /* suffix) allowed by the S3 gateway endpoint policy"
  type        = list(string)
  default     = []
}

# Domain (optional)
variable "domain_name" {
  description = "Domain name for the application (optional)"
  type        = string
  default     = ""
}

variable "create_dns_record" {
  description = "Whether to create Route53 DNS record"
  type        = bool
  default     = false
}

# Secrets (passed via GitHub Actions)
variable "google_client_id" {
  description = "Google OAuth client ID"
  type        = string
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth client secret"
  type        = string
  sensitive   = true
}

variable "encryption_key" {
  description = "Encryption key for sensitive data"
  type        = string
  sensitive   = true
}
