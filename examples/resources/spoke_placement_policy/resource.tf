# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_placement_policy" "example" {
  management_key      = "example-placement_policy"
  name                = "example"
  service_id          = "service-example"
  region              = "region-a"
  strategy            = "stopped"
  deletion_protection = true
}
