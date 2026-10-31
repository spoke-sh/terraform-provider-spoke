# Illustrative configuration for the provisional mock-v1 API.
# Run `just test-acc` for an automatically configured local fixture.
provider "spoke" {
  endpoint        = "https://management.example.invalid"
  organization_id = "org-example"
  project_id      = "project-example"
  environment_id  = "env-example"

  # Supply credentials through SPOKE_TOKEN or an external token_file.
  # Set ca_file when the endpoint uses a private CA; TLS stays verified.
}
