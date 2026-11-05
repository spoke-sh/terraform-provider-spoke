# Experimental operator declaration. Requires an existing operator scope and
# authorized references. The provisional profile does not launch workers.
resource "spoke_ci_governor" "internal_ci" {
  management_key        = "operator-ci-v1"
  name                  = "internal-ci"
  admission_policy_id   = "internal-entitlements"
  budget_policy_id      = "pilot-budget"
  cache_policy_id       = "tenant-cache"
  execution_profile_ids = ["linux-x86_64-v1"]
  max_active_jobs       = 2
  max_allocated_vcpus   = 16
  max_warm_runners      = 0
  max_job_seconds       = 3600
  max_queue_seconds     = 60
  retirement_policy     = "drain"
  deletion_protection   = true
}
