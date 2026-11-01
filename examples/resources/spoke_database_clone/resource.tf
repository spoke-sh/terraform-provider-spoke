# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_database_clone" "example" {
  management_key      = "example-database_clone"
  name                = "example"
  source_database_id  = "database-example"
  recovery_point_id   = "point-example"
  capacity_bytes      = 1073741824
  deletion_protection = true
}
