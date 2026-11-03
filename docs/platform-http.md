# Platform HTTP boundary

Every Terraform resource and data source uses the Go client in
[`internal/platformclient`](../internal/platformclient/). Platform owns service
execution and its Transit integration. The provider ships no Transit SDK, signer,
native library, ABI pin, principal sequencer or signed-envelope journal.

The original service inventory is **provisional mock-v5**: 23 resources
and 47 data sources. The separate `spoke_static_application` resource and exact
data source use `publication-v1`. All remain experimental. Platform now implements the narrower
[`spoke.platform.http.bucket.v1` development profile](https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/http/README.md)
and records local installed-provider qualification. This supplies bucket
declarations and recovery, with pending readiness and no object data plane.
The [current adoption plan](roadmap/platform-adoption-implementation-plan.md)
records its schema/limits, exact evidence and reviewed integration results;
the full provider inventory is not a published service contract.

## Current Terraform client and contracts

Provider configuration preserves explicit scope and transport settings. Service
adapters share the mock-v5 client; publication adapters instantiate the separately
negotiated publication client with the same credentials, CA and scope. Child scopes preserve the endpoint,
authentication and TLS settings. Credentials come from an external token file or
`SPOKE_TOKEN`. The implemented Platform bucket profile uses Bearer auth with
operator-provisioned token-to-principal mappings. Customer credential issuance
and scope/access lifecycle remain Platform prerequisites.

The client verifies TLS with system roots plus an optional custom CA, refuses
redirects, binds requests to organization and API version, bounds requests and
observations, and preserves explicit authorization/conflict/uncertainty outcomes.
Only authoritative resource absence can remove state. Reads never execute work.
Stable management keys, revision/generation preconditions and server receipts
recover accepted operations when a reply or Terraform state write is lost. Backend
sequencing and durable acceptance belong to Platform; HTTP does not weaken these
requirements or make identical-principal/key cloned states distinguishable.

[`contract.json`](../internal/platformclient/contract.json) binds the current wire
version, transport, publication status and schema SHA-256. Go embeds that identity;
release tooling reads the same source file. `terraform-provider-spoke -version`
reports it alongside the source/version/system identity. Schema changes must update
the digest together; a test verifies the exact schema bytes. A future Platform
handoff must supply a published schema, immutable Platform revision and actual
conformance/live results before changing the source's publication status.

[`profile.json`](../internal/platformclient/profile.json) separately pins the
development bucket profile, Platform source, schema and independent fingerprint
vectors. The executable reports this identity without promoting the full inventory
to published. The client binds the response profile across scoped copies, refuses
unknown or changing profiles, enforces the narrower bucket bounds, and supports
complete owner-filtered pending discovery. The provisional mock has no profile
header; it remains a separate development target, not a fallback after an error.

The protocol-level registry test enumerates every resource/data source, sends its
observation through a raw HTTPS peer and checks credentials, scope, version,
route and denial propagation. Managed state survives denial. An import boundary
check rejects service network/SDK dependencies outside the shared client, as well
as native SDK/process/SQL dependencies in the provider and client. Existing mock
Terraform and installed-process suites exercise successful CRUD and recovery.

## Publication declaration client

The separately negotiated `PublicationClient` implements Platform's
[`publication-v1` API candidate](https://github.com/spoke-sh/platform/pull/40).
Its [independent pin](../internal/platformclient/publication-contract/profile.json)
and copied schema/profile are separate from bucket `mock-v5`. The executable's
`publication_client_*` identity fields describe this HTTP integration, not a
newly registered Terraform resource or a published service inventory.

Project-scoped clients manage application policy and environment bindings;
environment-scoped clients manage target policy. Reads cover exact declarations,
complete bounded collections, original operation keys and filtered receipt
history, including expiry tombstones. Schema validation rejects omitted/null
required values, duplicate/unknown fields, oversized decimal counters, changing
snapshots and partial results. TLS, profile/version/scope and principal checks
apply independently of bucket negotiation. The synthetic peer profile is rejected.

Every mutation requires a caller-supplied key for exactly one logical intent.
The client inspects that key before dispatch and replays the exact HTTP request
to verify retained intent server-side. It never generates a new key, renews a
revision fence or computes Platform's native fingerprint. Accepted receipts retain
identity through settlement; denied, expired or incomplete discovery does not
authorize a fresh mutation. Uploads, sealing, runtime selection/promotion and
domains are outside this declaration client.

Run explicit local client qualification from a clean provider checkout against
the clean, pinned Platform fixture revision in the
[execution record](../qualification/publication-client.md):

```sh
nix develop --command python3 scripts/qualify-publication-client.py \
  --platform /absolute/clean/platform-checkout \
  --report /absolute/new/publication-client-report.json
```

The runner rejects a mismatched schema or dirty inputs, builds the actual Go client
test executable and uses Platform's installed native fixture, private loopback
TLS and synthetic roles. It kills an independent caller process after settlement
but before caller state save, then recovers with the original key in a new process.
Reports include source and binary identities; no ambient service credentials are
used. Ordinary CI runs conformance peers and skips these three externally selected
tests. This is client qualification, not Terraform apply/import or hosted delivery.

The experimental application adapter uses a bounded sequence of immutable
operation-key slots, proven against the normal API before registration. Read the
[identity decision](publication-identity.md) for ownership, import, expiry and
replacement constraints. Mutation deadlines honor the provider's configured
1–3600 seconds; standalone client calls default to 60 seconds.

The explicit installed application lane requires a clean committed provider,
its exact Linux amd64 Nix mirror and a clean Platform fixture:

```sh
nix develop --command python3 scripts/qualify-publication-terraform.py \
  --platform /absolute/clean/platform-checkout \
  --mirror /nix/store/exact-provider-mirror \
  --report /absolute/new/publication-terraform-report.json
```

It executes actual Terraform and killed-provider recovery through the normal API.
Missing artifacts fail without a source build fallback. The test harness builds
separately; Terraform has no compiler on PATH. This lane does not qualify uploads,
hosted services or Registry publication. The release manifest includes both the
new declaration surface and a separate delivered-byte gate.

## Explicit qualification configuration

`just test-acc live` requires `SPOKE_LIVE_CONFIG` and `SPOKE_LIVE_REPORT` to be
absolute paths outside the Nix store. The local metadata selection executes the
Platform-owned retained Transit fixture and its normal installed Terraform
consumer. It creates an isolated loopback HTTPS endpoint, private CA, seeded scope
and synthetic token files outside the Nix store. It neither uses ambient service
credentials nor provisions hosted infrastructure.

```json
{
  "api_version": "mock-v5",
  "api_schema_sha256": "a3100081694a125dee8779179d1a85fb3f5ff1cdd8af6f7a8dab46dd35333c0b",
  "gate": "metadata",
  "evidence_kind": "local-platform-metadata",
  "management_profile": "spoke.platform.http.bucket.v1",
  "profile_schema_sha256": "8568a6483a450b79464386d9cd6e5be5c7ad0c4163ceecd0ffe3a7682b4341bd",
  "fixture_checkout": "/absolute/clean/platform-checkout",
  "platform_revision": "<exact committed fixture revision>",
  "artifact": "/absolute/installed/terraform-provider-spoke",
  "artifact_sha256": "<SHA-256 of those exact bytes>",
  "provider_revision": "<exact committed provider revision>",
  "provider_version": "0.1.0-dev.g<exact committed provider revision>",
  "run_id": "isolated-http-check",
  "deadline_seconds": 1800,
  "max_objects": 32
}
```

The fixture checkout must include the installed-runner handoff in the
[adoption execution record](../qualification/platform-adoption.md). Its committed
`contracts/http/provider.json` must match the selected artifact, and its schema
and vectors must match this provider's pins. Dirty or mismatched inputs fail
before the fixture runs. The runner checks source again afterward and requires a
fresh report covering every schedule, including drift, uncertain cleanup,
process death and retained restart. Terraform uses readonly checksums and has no
compiler in its PATH. Failed fixtures and private logs are retained for recovery.
No old report can satisfy a new execution.

This local mode rejects hosted endpoint/credential fields. A passing result is
local metadata evidence and cannot satisfy publication. Hosted service selections
still run prerequisite checks only. Their configuration has the following shape;
replace the digest/path placeholders with the selected artifact's real values:

```json
{
  "api_version": "mock-v5",
  "api_schema_sha256": "<contract_schema_sha256 from the executable>",
  "gate": "service-graph",
  "endpoint": "https://isolated.example.invalid",
  "ca_file": "/absolute/external/ca.pem",
  "credential_file": "/absolute/external/token",
  "artifact": "/absolute/installed/terraform-provider-spoke",
  "artifact_sha256": "<SHA-256 of that exact executable>",
  "platform_revision": "<40-character immutable Platform revision>",
  "scope": {
    "organization_id": "org",
    "project_id": "project",
    "environment_id": "env"
  },
  "run_id": "isolated-http-check",
  "deadline_seconds": 60,
  "max_objects": 16
}
```

Supported selections are `metadata`, `initial-graph`, `service-graph`, `jobs`,
`messaging`, `volumes`, `networking`, `recovery` and `policies`. Family selections
include shared metadata cases. `service-graph` requires all sixteen current cases,
including the six later families exposed by this artifact. A partial selection
cannot qualify a full release.

The hosted selection validates configuration and artifact bytes, then returns a
nonzero blocked report: the full service contract and real backends remain
unqualified. It reads no credential/CA contents and sends no service requests
while blocked. The local metadata mode above must be selected explicitly.
A configured endpoint or forged passing mock report cannot turn that preflight
into live evidence. Reconcile the owned profile, report identity and required
cases before enabling actual calls; retain the full publication gate.

Old `profile`, `transit_revision`, native numeric scope and `journal_directory`
configuration is rejected. There is no compatibility migration or native fallback.
The workflow retains blocked reports as useful evidence of the incomplete gate.

## Release and remaining work

Publication requires a source-pinned **published HTTP contract**, matching artifact
and report contract/schema/Platform identities, the exact binary digest and every
required passing case. Missing, duplicate, skipped, blocked, provisional, native
or partial reports fail. Go artifact portability is tested independently of
Platform's runtime dependencies; public release signing remains outside Nix.

Platform supplies authentication, scoped discovery, bucket declarations and
original-operation recovery for its bounded development profile. Broader service
operations, customer scope/access and actual data planes remain prerequisites.
The pinned profile and explicit installed metadata lane are implemented here;
full service and distribution qualification remain open.
See the [adoption plan](roadmap/platform-adoption-implementation-plan.md) and
[execution record](../qualification/platform-adoption.md). Milestones 10–17 remain open.
