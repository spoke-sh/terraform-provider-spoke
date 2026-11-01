# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_volume_attachment" "example" {
  management_key      = "example-volume_attachment"
  name                = "example"
  volume_id           = "volume-example"
  service_id          = "service-example"
  mount_path          = "/data/files"
  deletion_protection = true
}
