# Run after bootstrap, with concrete project/environment outputs.
# Credentials for the granted service account are issued externally.
terraform {
  required_providers {
    spoke = { source = "spoke-sh/spoke" }
  }
}
variable "endpoint" { type = string }
variable "organization_id" { type = string }
variable "project_id" { type = string }
variable "environment_id" { type = string }
variable "token_file" {
  type    = string
  default = null
}
variable "ca_file" {
  type    = string
  default = null
}
provider "spoke" {
  endpoint        = var.endpoint
  organization_id = var.organization_id
  project_id      = var.project_id
  environment_id  = var.environment_id
  token_file      = var.token_file
  ca_file         = var.ca_file
}
resource "spoke_bucket" "assets" {
  management_key = "application-assets"
  name           = "assets"
}
data "spoke_buckets" "available" {
  depends_on = [spoke_bucket.assets]
}
output "bucket_id" { value = spoke_bucket.assets.id }
