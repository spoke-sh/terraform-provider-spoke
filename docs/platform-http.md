# Platform HTTP boundary

Every Terraform resource and data source uses the Go client in
[`internal/platformclient`](../internal/platformclient/). Platform owns service
execution and its Transit integration. The provider ships no Transit SDK, signer,
native library, ABI pin, principal sequencer or signed-envelope journal.

The current HTTP schema is **provisional mock-v5**. It is a provider development
contract, not a published Platform API. All 23 resources and 47 data sources remain
experimental. Removing native code does not create missing Platform endpoints or
qualify real backends.

## One client and one contract

Provider configuration creates one scoped HTTPS client, shared by managed
resources and all discovery/history adapters. Child scopes preserve the endpoint,
authentication and TLS settings. Credentials come from an external token file or
`SPOKE_TOKEN`; only the provisional schema currently specifies Bearer auth.
The public authentication method remains Platform's contract decision.

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

The protocol-level registry test enumerates every resource/data source, sends its
observation through a raw HTTPS peer and checks credentials, scope, version,
route and denial propagation. Managed state survives denial. An import boundary
check rejects service network/SDK dependencies outside the shared client, as well
as native SDK/process/SQL dependencies in the provider and client. Existing mock
Terraform and installed-process suites exercise successful CRUD and recovery.

## Explicit qualification configuration

`just test-acc live` requires `SPOKE_LIVE_CONFIG` and `SPOKE_LIVE_REPORT` to be
absolute paths outside the Nix store. The configuration has the following shape;
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

This lane currently validates configuration and artifact bytes, then returns a
nonzero blocked report: Platform has not published the selected HTTP abstraction,
installed real HTTP cases are not implemented, and real service backends remain
unqualified. It reads no credential/CA contents and sends no service requests while
those prerequisites are absent. A configured endpoint or forged passing mock
report cannot turn that preflight into live evidence. The missing HTTP contract
must be reconciled before enabling actual service calls.

Old `profile`, `transit_revision`, native numeric scope and `journal_directory`
configuration is rejected. There is no compatibility migration or native fallback.
The workflow retains blocked reports as useful evidence of the incomplete gate.

## Release and remaining work

Publication requires a source-pinned **published HTTP contract**, matching artifact
and report contract/schema/Platform identities, the exact binary digest and every
required passing case. Missing, duplicate, skipped, blocked, provisional, native
or partial reports fail. Go artifact portability is tested independently of
Platform's runtime dependencies; public release signing remains outside Nix.

Platform must supply HTTP authentication, scoped capabilities/discovery, typed
service operations, durable idempotency and original-operation lookup. This
repository then reconciles the provisional client/schema, implements isolated
installed live cases, verifies real data planes and qualifies public distribution.
See the [remaining plan](roadmap/milestones-10-17-implementation-plan.md) and
[execution record](../qualification/platform-http.md). Milestones 10–17 remain open.
