# Experimental mock-v5 contract; no live platform API is qualified.
resource "spoke_project" "application" {
  management_key = "application-project"
  name           = "application"
}

resource "spoke_environment" "production" {
  project_id     = spoke_project.application.resource_id
  management_key = "production-environment"
  name           = "production"
}

resource "spoke_service_account" "application" {
  management_key = "application-identity"
  name           = "application"
}

resource "spoke_grant" "read" {
  management_key = "application-read"
  name           = "application-read"
  subject_id     = spoke_service_account.application.resource_id
  target         = spoke_environment.production.target_path
  permission     = "read"
}
