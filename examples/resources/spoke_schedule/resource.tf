# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_schedule" "example" {
  management_key      = "example-schedule"
  name                = "example"
  job_id              = "job-example"
  cron                = "0 * * * *"
  time_zone           = "Etc/UTC"
  enabled             = false
  missed_run          = "catch_up_once"
  overlap             = "forbid"
  deletion_protection = true
}
