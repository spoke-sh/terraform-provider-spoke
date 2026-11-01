# Provisional mock-v5 management example; no live service is qualified.
# Apply examples/bootstrap first and supply its concrete scope/account IDs.
variable "endpoint" { type = string }
variable "organization_id" { type = string }
variable "project_id" { type = string }
variable "environment_id" { type = string }
variable "service_account_id" { type = string }
variable "token_file" { type = string }
variable "ca_file" { type = string }
variable "image_digest" { type = string }
variable "secret_version" { type = string }
variable "deploy" {
  type    = bool
  default = true
}
variable "grant_database" {
  type    = bool
  default = true
}

provider "spoke" {
  endpoint        = var.endpoint
  organization_id = var.organization_id
  project_id      = var.project_id
  environment_id  = var.environment_id
  token_file      = var.token_file
  ca_file         = var.ca_file
}
provider "spoke" {
  alias           = "organization"
  endpoint        = var.endpoint
  organization_id = var.organization_id
  token_file      = var.token_file
  ca_file         = var.ca_file
}
resource "spoke_artifact_registry" "images" {
  management_key      = "graph-images"
  name                = "images"
  retention_days      = 30
  deletion_protection = true
}
resource "spoke_database" "app" {
  management_key      = "graph-database"
  name                = "app"
  profile             = "shared"
  capacity_bytes      = 1073741824
  retention_days      = 30
  deletion_protection = true
}
resource "spoke_bucket" "assets" {
  management_key      = "graph-assets"
  name                = "assets"
  retention_days      = 30
  deletion_protection = true
}
resource "spoke_secret_reference" "credential" {
  management_key      = "graph-credential"
  name                = "credential"
  external_owner      = "external"
  external_id         = "credential"
  version             = var.secret_version
  deletion_protection = true
}
locals {
  targets = {
    images = {
      kind = "artifact_registry", id = spoke_artifact_registry.images.resource_id,
      path = spoke_artifact_registry.images.target_path, action = "artifact_pull"
    }
    database = {
      kind = "database", id = spoke_database.app.resource_id,
      path = spoke_database.app.target_path, action = "database_connect"
    }
    assets = {
      kind = "bucket", id = spoke_bucket.assets.resource_id,
      path = spoke_bucket.assets.target_path, action = "bucket_read"
    }
    credential = {
      kind = "secret_reference", id = spoke_secret_reference.credential.resource_id,
      path = spoke_secret_reference.credential.target_path, action = "secret_resolve"
    }
  }
}
resource "spoke_grant" "runtime" {
  provider            = spoke.organization
  for_each            = { for slot, target in local.targets : slot => target if slot != "database" || var.grant_database }
  management_key      = "graph-grant-${each.key}"
  name                = each.key
  subject_id          = var.service_account_id
  target              = each.value.path
  permission          = each.value.action
  deletion_protection = false
}
resource "spoke_container_service" "app" {
  count               = var.deploy ? 1 : 0
  management_key      = "graph-app"
  name                = "app"
  registry_id         = spoke_artifact_registry.images.resource_id
  repository          = "app/web"
  digest              = var.image_digest
  service_account_id  = var.service_account_id
  cpu_millis          = 500
  memory_mib          = 256
  configuration       = { LOG_LEVEL = "info" }
  secret_references   = { CREDENTIAL = spoke_secret_reference.credential.resource_id }
  required_bindings   = keys(local.targets)
  deletion_protection = false
}
resource "spoke_service_binding" "app" {
  for_each            = var.deploy ? local.targets : {}
  management_key      = "graph-binding-${each.key}"
  name                = each.key
  consumer_id         = spoke_container_service.app[0].resource_id
  target_kind         = each.value.kind
  target_id           = each.value.id
  slot                = each.key
  intent              = each.value.action
  deletion_protection = false
}
resource "spoke_endpoint" "app" {
  count               = var.deploy ? 1 : 0
  management_key      = "graph-endpoint"
  name                = "app"
  service_id          = spoke_container_service.app[0].resource_id
  port                = 8080
  exposure            = "private"
  tls_policy          = "managed"
  deletion_protection = false
  depends_on          = [spoke_service_binding.app, spoke_grant.runtime]
}
