# Use environment provider configuration for this parent collection.
# A denied or incomplete page fails the entire read.
data "spoke_buckets" "available" {}

output "buckets" {
  value = data.spoke_buckets.available.items
}
