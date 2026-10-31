resource "spoke_bucket" "assets" {
  management_key      = "application-assets"
  name                = "assets"
  capacity_bytes      = 1073741824
  retention_days      = 30
  deletion_protection = true
}
