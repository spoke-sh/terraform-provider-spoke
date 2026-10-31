# Milestones 5–8: the mock-backed application graph

Recorded 2026-10-31. This qualifies provider behavior against the provisional
mock-v3 management contract. It does not qualify a live service or data plane.

## Source and environment

Implementation and executable tests are committed at
[`201bf93`](https://github.com/spoke-sh/terraform-provider-spoke/commit/201bf93a3496cca475a3c5746d3114305a8bc645).
The logical slices are `601bde6` (typed contract/migration), `09152a9` (registries),
`ac71ba5` (databases/buckets), `c5d5d65` (references/bindings), `032b1a1`
(containers/endpoints) and `201bf93` (graph/recovery qualification).
The following documentation commit records this evidence without changing code.

Execution used the locked Nix shell, **Go 1.26.7, Terraform 1.15.3, Linux amd64**.
The refreshed platform assessment is
`5c170b77065981c079ed51e24c22026e341196be`; the
[capability matrix](../testdata/contracts/capabilities.md) records unavailable
customer provisioning APIs and their upstream owners. No platform implementation
or live environment was changed. Runtime evidence for other systems, Nix provider
packages, release artifacts and live services remains outside these milestones.

## Executed checks

- `nix develop --command bash -c 'just fmt && just check && just build'` passed:
  Nix/Just/Terraform/Go formatting, actionlint, host flake evaluation, Go vet,
  race-enabled unit/client/raw-schema tests, all **32 top-level mock Terraform
  acceptance tests** and their subtests, and the provider executable build.
- `nix develop --command ./scripts/generate-docs.sh` refreshed all resource/data
  source references while preserving the hand-maintained roadmap.
- `nix flake check --all-systems --no-build --no-update-lock-file` evaluated all
  four declared systems. Evaluation is not execution qualification.
- Relative Markdown links/fragments, contiguous milestone numbering, unique
  phase membership, index/header status consistency and `git diff --check` passed.
  Bare `just` still lists exactly five public tasks.

Ordinary unit tests leave `TF_ACC` unset. The required acceptance invocation sets
`TF_ACC=1` and `SPOKE_TEST_TARGET=mock`, uses local TLS endpoints and synthetic
credentials, and runs the harness with the race detector. Installed tests build
a normal provider binary into a private `0.0.1` filesystem mirror, verify an `h1`
checksum lock and readonly re-init, and execute Terraform without Go on its PATH
or development overrides. The test mirror is not a published release package.

## Evidence by milestone

| Milestone | Executable evidence | Observed outcome |
| --- | --- | --- |
| 5: registries | [Terraform service tests](../internal/provider/service_resource_test.go), [client service tests](../internal/platformclient/services_test.go), [raw wire tests](../internal/testutil/mockplatform/services_test.go) | Registry lifecycle/import/drift/empty-plan/denial/recreation; observational digest/tag lookup; missing versus denied reads; immutable digest metadata; quota/retention and retained-content deletion restrictions, including zero-byte artifacts. Moving a tag does not alter a pinned workload. |
| 6: databases/buckets | [Terraform service tests](../internal/provider/service_resource_test.go), [client service tests](../internal/platformclient/services_test.go), [installed migration](https://github.com/spoke-sh/terraform-provider-spoke/blob/dd75b117f249d482c06d77c94a3c97a5cc5ba579/internal/provider/migration_linux_test.go) | Stable IDs through capacity growth; shrink and unsafe retention refused; protected profile replacement leaves the existing database intact; nonempty deletion refused; existing bucket state gains settings without replacement or mutations. |
| 7: references/bindings | [Terraform service tests](../internal/provider/service_resource_test.go), [installed canary tests](../internal/provider/service_secrets_linux_test.go), [client service tests](../internal/platformclient/services_test.go) | Pinned version rotation retains the previous resolved version until resolution; exact same-environment relationships; slot/deletion reservations; explicit runtime authority independent of declaration; safe detach; reference changes invalidate the consumer's effective configuration without rewriting its desired spec. |
| 8: containers/endpoints | [Terraform lifecycle/validation](../internal/provider/service_resource_test.go), [installed service graph and crashes](../internal/provider/service_graph_linux_test.go), [client controllers](../internal/platformclient/services_test.go) | Stable declaration identity across digest changes, separately advanced readiness/active revision, missing image, pull denial, dependency/health/TLS failures, explicit exposure, grant revocation/restoration and retirement that retains independent services and grants. |

All six new resource types and the expanded bucket run shared Terraform lifecycle
checks for import, empty plans, rename, external drift, denied reads preserving
identity, authoritative absence and lost replies. New resources have exact and
bounded collection discovery; buckets gain exact reads. The provider now exposes
11 resources and 21 data sources. Schema validation and the composed graph cover
null/invalid settings and references whose IDs become known during apply.

The shared client/fixture tests additionally establish versioned canonical
fingerprints (sorted maps and binding-slot sets), stale revision rejection for
all seven service kinds, exact scope fences, concurrent binding/delete and slot
reservations, and settlement rechecking when artifact usage changes after an
update is accepted. Independent raw JSON requests are checked against the schema;
wrong-kind, missing/null required settings and secret-payload fields are rejected.

## Recovery and the installed graph

Client interruption tests cover **84 service CRUD interruptions**: seven service
kinds × create/update/delete × pre-dispatch, pre-acceptance, accepted/lost reply,
and completed/lost reply. Each cancels the actual HTTP caller, reloads the
fixture ledger, resumes the original intent and checks identities/effect counts.
This is caller cancellation evidence, distinct from killing a provider process.

Installed Terraform tests kill the actual provider process in **36 new service
cases**: six new kinds × CRUD × after durable acceptance/after settlement. They
restart the fixture and reload its ledger, then recover from Terraform state,
assert one new effect, preserve accepted IDs and retain independent prerequisites.
The original 12 bucket process-crash cases still run.

Two additional tests interrupt a partially applied application graph after
container acceptance and binding settlement. Recovery from the consumer's state
produces exactly 18 declarations (four bootstrap and fourteen service graph
objects), with no duplicate effect, followed by an empty plan. The tests use
fixture barriers and `-parallelism=1` to select the interruption, not timed sleeps.

The complete installed journey uses the checked-in
[service graph example](../examples/service-graph/README.md), separate bootstrap
and consumer roots, and real provider aliases. It creates all service kinds and
four runtime grants/bindings, publishes synthetic image metadata externally,
resolves pinned secret metadata, advances controllers explicitly and verifies
readiness. It then moves an external tag without changing the plan, updates the
configured digest and secret version, observes unavailable dependencies and failed
health/TLS while preserving the last active revision, and revokes/restores the
database grant. Setting `deploy = false` removes endpoint/bindings/compute while
retaining nonempty protected database/bucket services, registry artifacts,
reference metadata, the service account and all five independent grants. No
`-target` or manual state surgery is used.

Secret/credential canaries are checked against state, saved-plan JSON, failed-read
diagnostics and provider trace logs through reference success, rotation and
unavailable responses. Existing installed tests also cover raw saved plans,
pending timeout/recovery, independent principals/states, alias isolation,
self-revocation and denied refresh. The fixture stores only secret identifiers and
availability metadata; it has no secret-value endpoint.

## Explicit mock-v2 migration

[Captured historical state and ledger](https://github.com/spoke-sh/terraform-provider-spoke/blob/dd75b117f249d482c06d77c94a3c97a5cc5ba579/testdata/migration/README.md) were
produced by the installed provider at `a97dfca`, before the contract change.
`LoadV2Ledger` refuses pending v2 operations, migrates path syntax and preserves
IDs, owners, keys, generations and historical fingerprints. Terraform state
version 1 upgrades target paths and adds legacy bucket defaults. Installed tests
prove zero management mutations/replacements and empty plans for both resource-
derived grant targets and explicitly updated literal targets, including after
fixture restart. Unmigrated `/v2` paths are not accepted as new v3 targets.

Finish or recover pending work with the old provider before migrating. Literal
grant configuration must change to `/v3` alongside the explicit fixture cutover.
There is no automatic reinterpretation of an old pending receipt, production
migration endpoint or live compatibility guarantee.

## Remaining boundaries

The fixture models metadata, dependency/policy decisions and controller
observations. It implements no OCI registry, database queries, object transfer,
secret storage, container execution or ingress traffic. Apply establishes a
settled declaration, not usable service traffic. Live qualification remains
[milestone 10](../docs/roadmap/10-live-platform-qualification.md).

Fixture restart deserializes a synthetic ledger inside the Go harness; it is not
a separately killed platform process, power-failure test or production durability
proof. Cloned credentials plus management keys remain indistinguishable from
recovery. Ownership transfer and unmanaged adoption are unsupported; expired
receipts retain tombstones. Database/data migration, builds/uploads and secret
issuance stay outside Terraform. Nix packages and release/compatibility evidence
remain milestones 9 and 11. Other Terraform versions/platforms remain unqualified.
