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
  environment_id  = "nightly"
}
variable "binding_id" { type = string }
resource "spoke_static_target" "site" {
  management_key      = "consumer-target"
  binding_id          = var.binding_id
  name                = var.name
  deletion_protection = var.protected
  retention_days      = 1
}
output "target_import_id" { value = spoke_static_target.site.id }
output "target_id" { value = spoke_static_target.site.resource_id }
