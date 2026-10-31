# Experimental mock-v4 management; build/upload is an explicit external step.
resource "spoke_artifact_registry" "images" {
  management_key      = "application-images"
  name                = "application-images"
  format              = "oci"
  capacity_bytes      = 1073741824
  retention_days      = 30
  deletion_protection = true
}
