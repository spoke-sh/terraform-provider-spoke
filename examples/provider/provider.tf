terraform {
  required_providers {
    spoke = {
      source = "spoke-sh/spoke"
      # No public release is available; acceptance uses an isolated test mirror.
    }
  }
}

# Experimental mock-v5 API. Supply SPOKE_TOKEN or an external token_file.
provider "spoke" {
  endpoint        = "https://management.example.invalid"
  organization_id = "org-example"
  # Omit project_id/environment_id to bootstrap scope and access resources.
  # Bucket resources require known project_id and environment_id assertions.
}
