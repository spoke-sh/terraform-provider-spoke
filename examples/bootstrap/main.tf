# Experimental configuration: only mock-v4 endpoints are currently qualified.
terraform {
  required_providers {
    spoke = { source = "spoke-sh/spoke" }
  }
}
variable "endpoint" { type = string }
variable "organization_id" { type = string }
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
  token_file      = var.token_file
  ca_file         = var.ca_file
}
resource "spoke_project" "application" {
  management_key = "application-project"
  name           = "application"
}
resource "spoke_environment" "production" {
  project_id     = spoke_project.application.resource_id
  management_key = "production-environment"
  name           = "production"
}
resource "spoke_service_account" "application" {
  management_key = "application-identity"
  name           = "application"
}
resource "spoke_grant" "manage_environment" {
  management_key = "application-environment-management"
  name           = "manage-environment"
  subject_id     = spoke_service_account.application.resource_id
  target         = spoke_environment.production.target_path
  permission     = "manage"
}
output "project_id" { value = spoke_project.application.resource_id }
output "environment_id" { value = spoke_environment.production.resource_id }
output "service_account_id" { value = spoke_service_account.application.resource_id }
