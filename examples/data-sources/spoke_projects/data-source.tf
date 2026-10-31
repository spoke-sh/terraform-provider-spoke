# Use organization-only provider configuration for this parent collection.
# A denied or incomplete page fails the entire read.
data "spoke_projects" "available" {}

output "projects" {
  value = data.spoke_projects.available.items
}
