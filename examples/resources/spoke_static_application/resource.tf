# Use a project-scoped provider without environment_id.
resource "spoke_static_application" "site" {
  management_key      = "site-catalog"
  name                = "site"
  retention_days      = 30
  deletion_protection = true
}
