# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_volume" "example" {
  management_key      = "example-volume"
  name                = "example"
  profile             = "filesystem"
  access_mode         = "single_writer"
  region              = "region-a"
  capacity_bytes      = 1073741824
  deletion_protection = true
}
