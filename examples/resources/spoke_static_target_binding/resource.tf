# Use a project-only provider. Hand the reservation ID to the environment root.
resource "spoke_static_target_binding" "nightly" {
  management_key = "site-nightly-reservation"
  application_id = "123"
  environment_id = "nightly"
}
