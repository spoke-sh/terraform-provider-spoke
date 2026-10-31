# Use organization-only provider configuration for this parent collection.
# A denied or incomplete page fails the entire read.
data "spoke_service_accounts" "available" {}

output "service_accounts" {
  value = data.spoke_service_accounts.available.items
}
