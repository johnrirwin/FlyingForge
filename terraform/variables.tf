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
  default     = 2
}

variable "web_desired_count" {
  description = "Desired number of web tasks"
  type        = number
  default     = 2
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
