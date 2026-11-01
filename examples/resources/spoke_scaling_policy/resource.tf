# Provisional mock-v5 declaration. Referenced services are owned independently.
resource "spoke_scaling_policy" "example" {
  management_key      = "example-scaling_policy"
  name                = "example"
  service_id          = "service-example"
  min_instances       = 1
  max_instances       = 3
  target_cpu_percent  = 70
  cooldown_seconds    = 60
  deletion_protection = true
}
