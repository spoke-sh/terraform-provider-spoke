# References only; credentials and secret payloads remain external.
resource "spoke_service_binding" "database" {
  management_key      = "application-database-binding"
  name                = "database"
  consumer_id         = "service-example"
  target_kind         = "database"
  target_id           = "database-example"
  slot                = "database"
  intent              = "database_connect"
  deletion_protection = true
}
