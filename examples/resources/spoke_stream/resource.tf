# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_stream" "example" {
  management_key      = "example-stream"
  name                = "example"
  partitions          = 2
  ordering            = "partition"
  capacity_bytes      = 1073741824
  retention_days      = 7
  deletion_protection = true
}
