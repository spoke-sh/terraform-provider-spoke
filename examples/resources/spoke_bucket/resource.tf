resource "spoke_bucket" "assets" {
  management_key      = "application-assets"
  name                = "assets"
  deletion_protection = true
}
