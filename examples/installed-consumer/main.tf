terraform {
  required_version = "= 1.15.3"
  required_providers {
    spoke = {
      source  = "spoke-sh/spoke"
      version = "= 0.1.0-dev.g5708eb1dad834acf04d4c6f2b654a22af25696d9"
    }
    random = {
      source  = "hashicorp/random"
      version = "= 3.7.2"
    }
  }
}

variable "endpoint" { type = string }
variable "organization_id" { type = string }
variable "project_id" { type = string }
variable "environment_id" { type = string }
variable "token_file" { type = string }
variable "ca_file" { type = string }

provider "spoke" {
  endpoint        = var.endpoint
  organization_id = var.organization_id
  project_id      = var.project_id
  environment_id  = var.environment_id
  token_file      = var.token_file
  ca_file         = var.ca_file
}

# Prove the scoped mirror still permits a separately pinned Registry provider.
resource "random_id" "suffix" { byte_length = 4 }
resource "spoke_bucket" "assets" {
  management_key      = "external-consumer"
  name                = "assets-${random_id.suffix.hex}"
  deletion_protection = false
}
output "bucket_id" { value = spoke_bucket.assets.id }
