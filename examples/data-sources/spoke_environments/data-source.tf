# Use project provider configuration for this parent collection.
# A denied or incomplete page fails the entire read.
data "spoke_environments" "available" {}

output "environments" {
  value = data.spoke_environments.available.items
}
