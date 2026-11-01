# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_queue" "example" {
  management_key      = "example-queue"
  name                = "example"
  ordering            = "fifo"
  visibility_seconds  = 30
  max_attempts        = 5
  retention_days      = 7
  deletion_protection = true
}
