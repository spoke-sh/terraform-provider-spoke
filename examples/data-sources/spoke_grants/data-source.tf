# Use organization-only provider configuration for this parent collection.
# A denied or incomplete page fails the entire read.
data "spoke_grants" "available" {}

output "grants" {
  value = data.spoke_grants.available.items
}
