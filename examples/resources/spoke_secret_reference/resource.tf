# References only; credentials and secret payloads remain external.
resource "spoke_secret_reference" "application" {
  management_key      = "application-secret-reference"
  name                = "application-credential"
  external_owner      = "external-secret-store"
  external_id         = "application-credential"
  version             = "version-1"
  deletion_protection = true
}
