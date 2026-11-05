# Installed metadata consumer harness

The provider implements an explicit `hosted-metadata` consumer gate. Its fifteen
required cases pass against a **synthetic HTTPS bucket-profile peer**, including
nine actual installed provider-process deaths. This qualifies the consumer harness
against the development contract. **No hosted target was supplied or exercised.**
The provider did not deploy or administer Platform, restart a server, use native
operator credentials or change upstream source or pins.

## Exact inputs

| Input | Identity |
| --- | --- |
| Implementation and full-suite harness | `2d20a0843212ac30a3d85e153886028c54c43cb5` |
| Decoded-plan confidentiality harness | `78bf503cb08b9b32d6a6b74ad96b0631cac8f4b0` |
| Installed artifact | `0.1.0-dev.g2d20a0843212ac30a3d85e153886028c54c43cb5` |
| Executable SHA-256 | `20a292853a02634adb55aa7bbe6404f22b4c369c3a635515ea08e05272be9bbb` |
| HTTP contract | `mock-v5`, provisional; `a3100081694a125dee8779179d1a85fb3f5ff1cdd8af6f7a8dab46dd35333c0b` |
| Bounded profile | `spoke.platform.http.bucket.v1`, development metadata |
| Profile schema SHA-256 | `8568a6483a450b79464386d9cd6e5be5c7ad0c4163ceecd0ffe3a7682b4341bd` |
| Profile source | Platform `ebf97fe086a99ba91f6b669b9f922f3964eb5bfa` |
| Runtime | Linux amd64, Terraform 1.15.3, Go 1.26.7, locked Nix shell |

The mirror is
`/nix/store/m155xzsdr6v566n14nlygh7byzyjn7gv-terraform-provider-spoke-mirror-0.1.0-dev.g2d20a0843212ac30a3d85e153886028c54c43cb5`.
No independent bucket/publication/Saga consumer pin or lock changed. No public
resource schema, dependency or supported platform changed: the inventory remains
27 resources and 55 data sources. These artifact bytes are not published as a release.

## Executed consumer coverage

| Boundary | Proof |
| --- | --- |
| Direct lifecycle | Create/read/import, exact and complete collection discovery, empty plans and unavailable data-plane readiness. |
| Revision and drift | Rename A → B → A preserves identity with advancing revisions. External HTTP drift invalidates the saved Terraform plan and stale HTTP revision. Fresh apply converges. |
| Authority and protection | Invalid credentials fail refresh without changing state. Terraform and HTTP deletion reject protection. Explicit removal of protection permits deletion; authoritative absence removes state while retaining the receipt. |
| Observations | Installed Terraform discovery, plan and refresh through the relay dispatch no writes. |
| Nine process deaths | Create/update/delete each die before forwarding, after receipt arrival, and after a separate HTTP read confirms success. Retry retains operation ID/fingerprint/generation/owner and resource identity. |
| Retained checkpoints | Reopening completed steps with the original keys/state dispatches no additional mutations. |
| Confidentiality | External credential bytes are absent from state, diagnostics, reports, raw evidence and decoded saved-plan JSON. |

The transport verifies upstream TLS and never fabricates acceptance or controls
server settlement. A returned acceptance may already be terminal. The harness
kills the provider child and leaves Terraform alive to report the failure.
There is no server restart or backend durability claim in these cases.

The additional plan scan at `78bf503` changes test code only. It was executed
against the same immutable `2d20a08` artifact after both broad suites had started
from that clean commit. Source/artifact attribution remains separate.

## Validation and remaining gates

The [machine-readable record](hosted-metadata-consumer.json) records commands,
complete case inventories, durations, artifact identity and local log digests.
Locked-shell `just fmt`, `just check` and `just build` passed. The ordinary
acceptance suite passed all 47 groups in 1,075.020 seconds; `just test-acc installed`
passed the same 47 groups in 1,027.780 seconds. The final installed consumer matrix
with decoded-plan scanning and retained-step replay passed in 24.895 seconds.
The full check also covers workflow/format validation, Nix packaging, Go vet,
race unit tests and Python release-gate tests. No Nix source or lock changed.
The new gate's inventory is separate from the release manifest. Negative tests
reject mismatched targets/principals, mock or malformed responses, redirects,
untrusted TLS, missing artifacts, stale or incomplete reports and metadata reports
presented as release evidence. Existing local metadata/declaration validation and
process deadline tests remain covered by unit tests.

Hosted execution requires an externally supplied isolated management API and
consumer credential, scope, principal and TLS trust. The current website does not
supply that qualification target. Platform owns separate server restart, receipt
expiry, revocation and durable backend evidence. Its revision in a future report
is operator attribution until the API provides attestation. Metadata proves no
storage bytes, controller readiness, full service graph or Registry release.
TF10/TF11 remain In Progress. Follow the [run guide](../docs/hosted-metadata.md).
