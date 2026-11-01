# Native boundary assessment and live gate

This is the historical PR 8 assessment and evidence for the native-v1 codecs.
For the refreshed native-v2 service and latest Transit SDK assessment, see the
[remaining milestones 10–17 plan](../docs/roadmap/milestones-10-17-implementation-plan.md#source-basis-and-changes-since-pr-8).
The provider's current codecs/preflight have not yet adopted that newer contract.

Inspected platform `230c2c4cee12c08aaeaaf65e3c252a6d50450a55` and its Transit
pin `0e2fba44727ca858bd1e8e4a775b62fbcf1a665d`. The platform's native bucket
metadata implementation is unchanged from the plan's `b13c86d` assessment.
This record establishes contract encoding and prerequisite checks, **not live
provider qualification**. No upstream service implementation is included here.

## Executed boundary work

[`internal/nativecontract`](../internal/nativecontract/) implements the platform
application's 64-byte command, precondition-bound SHA-256 fingerprint, bounded
read context and canonical `SV1R` result. The independent Python little-endian
[vectors](../testdata/contracts/native-v1-vectors.json) follow the owning
[command/read specification](https://github.com/spoke-sh/platform/blob/230c2c4cee12c08aaeaaf65e3c252a6d50450a55/packages/management/README.md),
[Owner](https://github.com/spoke-sh/platform/blob/230c2c4cee12c08aaeaaf65e3c252a6d50450a55/packages/management/native/Owner.zig)
and [Gateway](https://github.com/spoke-sh/platform/blob/230c2c4cee12c08aaeaaf65e3c252a6d50450a55/packages/management/native/Gateway.zig).
Tests reject invalid scope, names, target/revision combinations, selector leakage,
unsafe counters, malformed booleans, truncated results and business refusal text.
Changing scope, generation, revision or incarnation changes the fingerprint.
This package does not implement Transit signatures or submit any request.

| Current mock-v4 | Native profile | Adapter consequence |
| --- | --- | --- |
| Bearer REST with version/scope headers | Signed Serving commands and current subject proofs | Explicit profile selection; no retry through a different protocol. |
| String IDs, names up to 63 characters, typed storage | Seeded u32 scope, 16-byte key, bounded numeric incarnation, 3–16 ASCII names, metadata only | Reject unsupported settings before dispatch; no object-storage claim. |
| JSON fingerprint excludes management generation | Exactly 64 bytes including all preconditions and target | Never reuse a mock fingerprint or old envelope. |
| Operation response and owner-bound management receipt | Native acceptance, service admission, management acceptance, business result | HTTP 202 or native completion alone is not CRUD success. |
| Expiring receipts leave tombstones | Exact-generation receipts live with the native resource | Unavailable receipt is uncertainty, never permission to recreate. |
| Full resource collections | Caller-only pending pages at a resource revision | Cannot implement current resource collection data sources from pending pages. |
| Retained synthetic controller state | Bucket metadata without object bytes or full graph | Actual OCI, rows, object bytes, workloads and TLS remain unqualified. |

## Concrete upstream blockers

The pinned [Go client](https://github.com/spoke-sh/transit/blob/0e2fba44727ca858bd1e8e4a775b62fbcf1a665d/sdks/go/client/client.go)
exports raw `PrepareSend`, `RecoverSend`, `Dispatch`, generic mutation recovery
and authority capabilities. It does **not** expose Serving command preparation,
Serving operation/disposition reads, arbitrary native subject-proof creation or
the required management read-proof transport. Its
[configuration](https://github.com/spoke-sh/transit/blob/0e2fba44727ca858bd1e8e4a775b62fbcf1a665d/sdks/go/client/types.go)
also lacks the verified custom CA input required by the selected HTTPS fixture.
A raw authority send cannot stand in for a Serving command. Therefore an actual
SDK Serving round trip cannot be built at this pin without expanding the shared
boundary upstream. The provider adds neither a competing signer nor a native
runtime dependency it cannot use. The codecs are not registered resources.

Transit owns those shared binding/ABI additions, matching exported recovery
operations, verified TLS options, cancellation and supported native packaging.
The SDK currently retains a Nix runtime closure and offers no portable native
release. When those interfaces land, pin their exact Go/generated ABI/native
inputs and run the SDK integration against the actual platform placement.

The provider then owns the adapter and a durable external journal of the exact
exported prepared work **before dispatch**, plus exclusive principal sequencing,
identity/digest checks and fsync/atomic replacement. Opening a missing, corrupted
or mismatched journal must fail; it must not allocate fresh work after an uncertain
outcome. Crash-before-write, write-before-dispatch, lost acceptance and process
restart remain unexecuted native gates. A standalone file store without a working
SDK adapter would not qualify this behavior, so no unused journal API is shipped.

Platform P5/P6/P10/P11/P13 still own customer scope/IAM, object/database effects,
OCI, workload/secret/TLS behavior and the complete isolated graph. P4 owns native
placement and management lifecycle. Full milestone 10 and publication require
those real services. The mock remains a separate provisional profile until a
native kind can be adopted by hard cutover; there is no mock/native translator.

## Explicit live lane

`just test-acc live` selects only live cases and requires `SPOKE_LIVE_CONFIG` and
`SPOKE_LIVE_REPORT`, both absolute paths outside the Nix store. Ordinary CI never
selects this lane from credentials. Configuration schema is in
[`liveplatform.Config`](../internal/testutil/liveplatform/qualification.go).
It requires the exact profile/pins/artifact digest, HTTPS origin, external CA and
credential file paths, external journal directory, seeded numeric scope, isolated
run ID, deadline and object budget. `gate` is `metadata` or `service-graph`.
The release gate requires `service-graph`.

Preflight checks configuration and artifact identity without reading signing
seeds or sending network requests. It emits a machine-readable **blocked** report
and exits nonzero for the verified SDK/adapter/service gaps. It cannot pass from
configured credentials, a healthy endpoint, mock results or zero selected tests.
The case manifest rejects missing, duplicate, skipped and non-passing results and
mismatched artifact/profile/revision identities. The current workflow is a failing
prerequisite gate, not a completed live adapter or a claim of qualification.

The manual live workflow uses an operator-provisioned isolated runner labelled
`spoke-qualification` and repository variable `SPOKE_LIVE_CONFIG` pointing to its
external configuration. Bootstrap, fixture allocation, restart/fault controls,
cleanup inventory and credential issuance belong to that harness/operator. Before
implementing live calls, add explicit cleanup inventory and controls to the harness;
never substitute fixture controllers or broad forced deletion for real evidence.
