# Experimental management metadata; this does not run SQL or database migrations.
resource "spoke_database" "application" {
  management_key      = "application-database"
  name                = "application"
  profile             = "shared"
  capacity_bytes      = 1073741824
  retention_days      = 30
  deletion_protection = true
}
