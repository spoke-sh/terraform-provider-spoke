# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_backup_policy" "example" {
  management_key      = "example-backup_policy"
  name                = "example"
  database_id         = "database-example"
  cron                = "0 2 * * *"
  time_zone           = "Etc/UTC"
  enabled             = false
  retention_days      = 30
  deletion_protection = true
}
