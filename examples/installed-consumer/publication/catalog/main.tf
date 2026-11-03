terraform {
  required_version = "= 1.15.3"
  required_providers {
    spoke = {
      source  = "spoke-sh/spoke"
      version = "= 0.1.0-dev.gbc1860930924ee5c0a88004273eb85bedf54cfe2"
    }
  }
}
variable "endpoint" { type = string }
variable "ca_file" { type = string }
variable "token_file" { type = string }
variable "protected" { default = true }
variable "name" { type = string }
provider "spoke" {
  endpoint        = var.endpoint
  ca_file         = var.ca_file
  token_file      = var.token_file
  organization_id = "example"
  project_id      = "saga"
}
resource "spoke_static_application" "site" {
  management_key      = "consumer-catalog"
  name                = var.name
  retention_days      = 1
  deletion_protection = var.protected
}
resource "spoke_static_target_binding" "nightly" {
  management_key = "consumer-nightly-binding"
  application_id = spoke_static_application.site.resource_id
  environment_id = "nightly"
}
resource "spoke_static_target_binding" "production" {
  management_key = "consumer-prod-binding"
  application_id = spoke_static_application.site.resource_id
  environment_id = "production"
}
output "nightly_binding" { value = spoke_static_target_binding.nightly.resource_id }
output "production_binding" { value = spoke_static_target_binding.production.resource_id }
output "application_import_id" { value = spoke_static_application.site.id }
