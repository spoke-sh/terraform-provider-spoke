# Experimental mock-v5 contract; no live platform API is qualified.
# Credentials are issued externally; this resource stores no credential values.
resource "spoke_service_account" "application" {
  management_key = "application-identity"
  name           = "application"
}
