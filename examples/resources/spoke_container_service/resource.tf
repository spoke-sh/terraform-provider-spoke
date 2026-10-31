# Experimental mock-v4 declaration. Readiness advances independently.
resource "spoke_container_service" "example" {
  management_key      = "example-container-service"
  name                = "example"
  deletion_protection = true
  registry_id         = "registry_id"
  repository          = "app/web"
  digest              = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  service_account_id  = "account_id"
  cpu_millis          = 500
  memory_mib          = 256
  configuration       = { LOG_LEVEL = "info" }
  secret_references   = { CREDENTIAL = "secret_reference_id" }
  required_bindings   = ["database"]
}
