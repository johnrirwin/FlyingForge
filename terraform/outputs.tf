output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = aws_lb.main.dns_name
}

output "alb_zone_id" {
  description = "Zone ID of the Application Load Balancer"
  value       = aws_lb.main.zone_id
}

output "ecr_server_repository_url" {
  description = "ECR repository URL for server"
  value       = aws_ecr_repository.server.repository_url
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "ecs_server_service_name" {
  description = "ECS server service name"
  value       = aws_ecs_service.server.name
}

output "rds_endpoint" {
  description = "RDS instance endpoint"
  value       = aws_db_instance.main.endpoint
  sensitive   = true
}

output "redis_endpoint" {
  description = "ElastiCache Redis endpoint"
  value       = aws_elasticache_cluster.main.cache_nodes[0].address
  sensitive   = true
}

output "app_url" {
  description = "Application URL"
  value       = var.domain_name != "" ? "https://${var.domain_name}" : "https://${aws_cloudfront_distribution.web.domain_name}"
}

output "cloudfront_domain_name" {
  description = "CloudFront distribution domain name for the web frontend"
  value       = aws_cloudfront_distribution.web.domain_name
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID for cache invalidations"
  value       = aws_cloudfront_distribution.web.id
}

output "web_bucket_name" {
  description = "S3 bucket name that hosts the web frontend assets"
  value       = aws_s3_bucket.web.bucket
}

output "web_url" {
  description = "Web frontend URL"
  value       = var.domain_name != "" ? "https://${var.domain_name}" : "https://${aws_cloudfront_distribution.web.domain_name}"
}

output "vpc_endpoint_ids" {
  description = "VPC endpoint IDs used to keep private subnet AWS traffic off NAT"
  value = var.enable_vpc_endpoints ? merge(
    { s3 = aws_vpc_endpoint.s3[0].id },
    { for name, endpoint in aws_vpc_endpoint.interface : name => endpoint.id }
  ) : {}
}

output "vpc_endpoint_security_group_id" {
  description = "Security group attached to interface VPC endpoints"
  value       = var.enable_vpc_endpoints ? aws_security_group.vpc_endpoints[0].id : null
}

output "news_refresh_schedule_arn" {
  description = "ARN for the daily scheduled one-shot news refresh task"
  value       = var.enable_scheduled_news_refresh ? aws_scheduler_schedule.news_refresh[0].arn : null
}
