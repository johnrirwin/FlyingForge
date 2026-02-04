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

# DynamoDB table for Terraform state locking (imported, not managed)
# The table was created manually before Terraform init
