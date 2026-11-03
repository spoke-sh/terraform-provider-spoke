# Use an explicit environment provider with that environment's machine token.
resource "spoke_static_target" "nightly" {
  management_key      = "site-nightly"
  binding_id          = "456"
  name                = "nightly"
  deletion_protection = true
  retention_days      = 30
}
# Selected bundle/generation and observed endpoints are read-only; no upload occurs.
