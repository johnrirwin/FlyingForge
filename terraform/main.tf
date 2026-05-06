terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "flyingforge-terraform-state"
    key            = "terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "flyingforge-terraform-locks"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "FlyingForge"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

locals {
  public_app_host                 = var.domain_name != "" ? var.domain_name : aws_cloudfront_distribution.web.domain_name
  public_app_url                  = "https://${local.public_app_host}"
  backend_path_patterns           = ["/api", "/api/*", "/health", "/mcp", "/oauth/*", "/.well-known/*"]
  mcp_resource_url                = "${local.public_app_url}/mcp"
  mcp_google_callback_url         = "${local.public_app_url}/oauth/google/callback"
  mcp_discovery_loopback_url      = "http://127.0.0.1:8080/.well-known/openid-configuration"
  mcp_authorization_jwks_loopback = "http://127.0.0.1:8080/oauth/jwks.json"
}

# DynamoDB table for Terraform state locking (imported, not managed)
# The table was created manually before Terraform init
