# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_custom_domain" "example" {
  management_key      = "example-custom_domain"
  name                = "example"
  domain              = "app.example.com"
  endpoint_id         = "endpoint-example"
  deletion_protection = true
}
