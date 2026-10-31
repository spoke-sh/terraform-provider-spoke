# Experimental mock-v3 declaration. Readiness advances independently.
resource "spoke_endpoint" "example" {
  management_key      = "example-endpoint"
  name                = "example"
  deletion_protection = true
  service_id          = "container_service_id"
  port                = 8080
  exposure            = "private"
  tls_policy          = "managed"
}
