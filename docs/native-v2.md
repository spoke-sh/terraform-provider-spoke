# Native v2 bucket metadata profile

The platform contract is pinned to [`73b4360`](https://github.com/spoke-sh/platform/blob/73b43609a1b493bafaee836bdc349c9e55db68ab/contracts/management-native-v2.md),
with its [schema](https://github.com/spoke-sh/platform/blob/73b43609a1b493bafaee836bdc349c9e55db68ab/contracts/management-native-v2.schema.json) and
[capability inventory](https://github.com/spoke-sh/platform/blob/73b43609a1b493bafaee836bdc349c9e55db68ab/contracts/management-native-v2-capabilities.json).
The coordinated integration requires Transit
`c2e50061fc8622300e3a517e6e3a82f5c6eb6db1` and native definition
`8e441078693f1373d69323bfeadd5cfe8c7024ec2015819bacfb2b04eaea3865`.
The earlier platform application-contract baseline used an incompatible runtime;
upgrade the platform consumers together before selecting this provider profile.
Shared intent vectors are copied byte-for-byte into `testdata/native-v2/vectors.json`;
the provider's canonical implementation is tested independently against them.

Select `profile = "spoke.management.native.v2"` explicitly. This profile connects
to the platform's seeded Station owner over verified HTTPS. Only bucket metadata
CRUD/import and exact/complete bucket discovery are available. Unsupported kinds
fail before mutation; project/environment creation and grants remain outside this
profile. Mock-v5 stays the default independent fixture protocol.

Use canonical positive decimal scope/resource IDs, a 32-character lowercase hex
management key (a nonzero 16-byte value), and a 3–16 character lowercase bucket name.
Deletion protection still defaults to true. Capacity and retention are recorded
explicitly; readiness remains `pending`, reason `metadata_only`, with no endpoint
or active revision. Applying Terraform does not create an object store.
`target_path` is a profile-specific identity label, not a mock-v5 REST route.

```hcl
provider "spoke" {
  profile            = "spoke.management.native.v2"
  endpoint           = "https://management.example.test"
  organization_id    = "1"
  project_id         = "2"
  environment_id     = "3"
  ca_file            = "/run/spoke/ca.pem"
  native_config_file = "/run/spoke/native.json"
}

resource "spoke_bucket" "data" {
  management_key = "c672be3cfcf843ee804bd8cac05b4f71a"
  name           = "review-data"
  capacity_bytes = 1073741824
  retention_days = 7
}
```

The shared Zig caller owns signing, recovery classification and verified TLS;
Go owns only the platform application codec and Terraform adaptation. The Nix
package supplies an immutable native caller path. An explicit absolute
`client_executable` may select a matching installed qualification artifact.
The selected TLS profile uses an RSA certificate, verified private CA and exact
hostname with TLS 1.2 or 1.3. The current native client refuses the local fixture's
Ed25519 TLS certificate. Native request signatures still use Ed25519; this
transport limitation never permits disabling certificate verification.

The external native JSON contains `realm`, `principal`, `credential_epoch`,
`ingress`, `service`, `initial_sequence`, `seed_file` and `journal_directory`.
Obtain the exact native IDs/epoch from operator bootstrap. `seed_file` holds a
32-byte hex Ed25519 seed; never place it in Terraform variables, state, plans,
source, or Nix inputs. `token_file` is rejected and ambient `SPOKE_TOKEN` is not
used in this profile. No native configuration action provisions services.

Precreate a private absolute **0700** journal directory on one durable local
filesystem. For a newly allocated, unused principal only, run the installed
`native-journal /private/fresh-principal.json` initializer. Its strict manifest
contains `fresh_unused_principal: true`, `endpoint`, `scope` (string
`organization_id`, `project_id`, `environment_id`) and `native` (the external
native configuration above, with `initial_sequence: 0`). Keep the bootstrap
witness separately. Initialization refuses an existing lock or journal, including
a partial initialization. Terraform never initializes or recreates these files. Every provider alias/process using the principal must use the same
directory, endpoint and native target. The journal contains private recovery material, including intents and
signed commands. It is bounded to 1 MiB and 256 retained operations and must be
retained and backed up. It uses
one persistent OS lock inode, fsync before dispatch, atomic replacement and
native sequence fences. Dispatch refuses a missing lock and never recreates it:
another process may still hold the original unlinked inode. The lock must be an
owned private regular file, and the journal must be a private regular file.
A saved unacknowledged send is retried exactly
before any later sequence is allocated. Concurrent unrelated writers using the
credential are outside the profile and fail closed at native admission.

This is a **single-host dispatch profile**. Distributed/NFS lock semantics,
journal loss/rollback recovery and automatic compaction are not qualified. Do not
copy or delete the lock/journal to resolve a conflict, or reset `initial_sequence`
to bypass it. An operator must reconcile the original native identity first.
Credential rotation retains the same principal and journal; update the external
epoch and key references and gateway public-key configuration. An unresolved send
that cannot be inspected or authenticated remains blocked rather than replaced.

Receipts preserve original generations and failed outcomes. Current native rights
are checked again at settlement. A pending read preserves Terraform state and may
require retry after the independent owner completes its continuation. Current
permission failures, incomplete pages, unknown errors and unavailable resources
never become authoritative absence. Expired owner/key tombstones fail permanently;
import uses the original key and cannot transfer ownership. V1/mock state is not
automatically migrated into this profile.

Concurrent Terraform observations queue under their caller deadline. A transient
read failure has at most three attempts with 100/200 ms backoff; denied, expired
and stale-page responses return directly. Mutations do not use this retry policy.

## Packaging and qualification

`nix build .#terraform-provider-spoke` builds a provider and
`nix build .#terraform-provider-spoke-mirror` exports a normal filesystem mirror. Its prerelease
version includes the source path's content identity; read the exact version with
`nix eval --raw .#terraform-provider-spoke.version`. Use that exact version and
`terraform providers lock -fs-mirror=…`, then `terraform init -lockfile=readonly`.
There are no dev overrides, registry release claims or credentials in the outputs.
Cross-system evaluation does not establish runtime support.

The platform owns bootstrap, TLS placement, native failure injection and the
installed-consumer lane. Its test runs the binary in Terraform with a locked
filesystem mirror and no Go toolchain in the consumer PATH. Provider-only tests
cover independent canonical vectors, bounded decoding and rejected lossy settings.
The full service graph and actual data planes retain milestone 10's gates.
