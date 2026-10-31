# Experimental mock-v4 contract; no live platform API is qualified.
# The organization-scoped provider can create both parent and child.
resource "spoke_project" "application" {
  management_key = "application-project"
  name           = "application"
}

resource "spoke_environment" "production" {
  project_id     = spoke_project.application.resource_id
  management_key = "production-environment"
  name           = "production"
}
