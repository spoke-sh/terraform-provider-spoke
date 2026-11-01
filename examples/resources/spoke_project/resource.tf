# Experimental mock-v5 contract; no live platform API is qualified.
# Use an organization-scoped provider (omit project_id and environment_id).
resource "spoke_project" "application" {
  management_key = "application-project"
  name           = "application"
}
