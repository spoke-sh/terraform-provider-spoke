# Complete service graph (experimental mock-v4)

This example declares management metadata. No live platform service, OCI push,
database/object access, external secret resolution, workload execution or endpoint
traffic is qualified. The installed-provider acceptance test uses `main.tf` with
synthetic credentials, a checksum-locked filesystem mirror and no Go on its PATH.

Apply [the bootstrap root](../bootstrap/main.tf) first. Pass its `project_id`,
`environment_id` and `service_account_id` outputs as concrete variables here,
alongside the explicit endpoint/organization and external token/CA file paths.
Do not configure this provider from resources created in this same root.
The operator credential needs management authority; the bootstrap management
grant alone does not authorize the application's runtime data access.

Build the application from pinned source/Nix inputs, then explicitly upload or
promote it to the registry outside Terraform. Use a verified immutable SHA-256
image digest as `image_digest`. A Nix store path identifies a local output, not a
remote image. The fixture substitutes published synthetic metadata for that
external upload. Plan/refresh never builds or pushes images.

Supply an externally managed identifier and pinned `secret_version`; the example
reference uses owner `external` and identifier `credential`. Change these to the
intended external identifiers before first apply. Never substitute secret bytes.
The `configuration` map is ordinary Terraform state: it must contain only
non-secret strings. Credentials belong in external files/environment, not Nix
expressions, Terraform variables or resource outputs.

The four explicit runtime grants authorize artifact pull, database connection,
object reads and secret resolution. Bindings declare the matching connection
slots; creating them grants no authority. A workload returns its identity before
bindings exist. `required_bindings` names slots rather than their resource IDs,
so the graph remains acyclic. The endpoint explicitly selects private exposure
and managed TLS and depends on the bindings/grants.

A successful apply settles declarations. Inspect `readiness`, `reason`,
`desired_revision`, `effective_revision`, `active_revision` and `tls_state`
separately. Pending dependencies, denied access and health/TLS failures do not
roll back the declaration or erase the last active revision. Fixture controllers
advance explicitly in tests; provider reads never advance them.

Update `image_digest` or `secret_version` explicitly for a rollout. Moving an
external tag cannot change a pinned digest. Setting `grant_database = false`
revokes that one grant and leaves the declared database intact.

Set `deploy = false` and apply to remove the endpoint, detach bindings and retire
compute while retaining the registry, database, bucket, secret reference and
independent grants. No `-target` or state edits are needed. Data resources retain
protection. Even disabling it cannot force nonempty/in-use deletion; data
migration, artifact retention and external cleanup remain separate operations.

Database profiles are service metadata, not a PostgreSQL/MySQL compatibility
claim. Database schema migrations and rows are outside Terraform. Buckets are
object services, not POSIX volumes or internal Transit segment stores.
