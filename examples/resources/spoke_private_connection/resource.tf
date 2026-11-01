# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_private_connection" "example" {
  management_key       = "example-private_connection"
  name                 = "example"
  service_id           = "service-example"
  peer_organization_id = "peer-org"
  peer_network_id      = "peer-network"
  cidr                 = "10.0.0.0/24"
  deletion_protection  = true
}
