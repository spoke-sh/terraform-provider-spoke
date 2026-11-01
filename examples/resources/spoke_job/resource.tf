# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_job" "example" {
  management_key      = "example-job"
  name                = "example"
  registry_id         = "registry-example"
  repository          = "app/jobs"
  digest              = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  service_account_id  = "account-example"
  max_concurrent      = 1
  retry_limit         = 1
  timeout_seconds     = 3600
  deletion_protection = true
}
