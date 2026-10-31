# Milestones 9–11 implementation plan

[Roadmap](README.md) · **Plan status:** In Progress · **Review:** Implementation authorized

Package the provider for an independent consumer, adapt it to the platform-owned
management boundary, qualify the real application graph, and publish the exact
qualified artifacts. Start with a hard cutover away from the mock compatibility
layer. This document plans implementation; it does not qualify new behavior or
advance milestone completion statuses.

## Decisions and compatibility policy

The user's direction supersedes the earlier mock migration plan: **use hard
cutovers and maintain no backwards compatibility until all currently defined
roadmap milestones 1–17 are complete**. That includes mock wire formats, fixture
ledgers, fingerprints and Terraform state from previous development contracts.
Reassess compatibility commitments after that gate; do not promise them merely
because milestone 11 publishes an initial release.

| Decision | Planned approach |
| --- | --- |
| Development contract | One current contract per implemented profile. Change client, fixture, schemas, examples and tests together; remove superseded adapters and migration code. |
| Existing mock migration | Remove the v2-to-v3 state/ledger conversion and legacy bucket defaults first. Keep prior qualification as historical evidence, not an active compatibility gate. |
| Recovery | Preserve retries, imports, ownership, revisions, receipts and crash recovery within the current contract. A hard cutover never authorizes forgetting uncertain effects or deleting retained data. |
| Distribution | Immutable pre-1.0/experimental releases with exact provider/API/artifact identities. Breaking cutovers get new versions; published bytes are never overwritten. |
| Initial system | Linux amd64 first. Evaluate other Nix systems separately; advertise additional systems only after native execution and recovery evidence. |
| Native adapter | Evaluate the pinned Transit Go SDK and its native ABI before changing transport. Preserve its signing/recovery ownership and account for its runtime closure. |
| Initial release | Milestone 11 can close for a qualified initial profile before milestones 12–17. It tests selected CLI/system/API combinations, not historical mock migrations. |

A new test contract uses fresh isolated fixture state and a new Terraform test
root. Incompatible old state/ledgers/receipts must fail before any platform
mutation, with an actionable diagnostic. Never automatically discard state,
replay an old envelope as new intent, or purge a remote service. An operator may
inspect/finish outstanding work using its original pinned environment; retaining
that artifact does not require maintaining its code path in the new provider.

## Baseline and refreshed upstream assessment

Start from merged [PR #7](https://github.com/spoke-sh/terraform-provider-spoke/pull/7),
provider revision
[`dd75b11`](https://github.com/spoke-sh/terraform-provider-spoke/commit/dd75b117f249d482c06d77c94a3c97a5cc5ba579).
Milestones 1–8 are completed against mock endpoints. The
[qualification record](../../qualification/milestones-5-8.md) covers 11 resources,
21 data sources, the installed application graph and process recovery on Go
1.26.7 / Terraform 1.15.3 / Linux amd64. It also records the now-superseded v2
migration work. Milestone 9 remains **In Progress**; 10 and 11 remain **Planned**.

Current implementation boundaries:

- [flake.nix](../../flake.nix) exports development shells and formatters only.
  It has no provider package, install mirror, release ZIP or consumer check.
- [main.go](../../main.go) embeds a provider version through `main.version` and
  serves protocol 6 at `registry.terraform.io/spoke-sh/spoke`. There is no release
  version/provenance policy or release workflow.
- [The process harness](../../internal/provider/process_linux_test.go) compiles a
  temporary Go binary itself, puts it in a `0.0.1` mirror and removes Go from the
  Terraform process PATH. It does not yet test a Nix-built distribution artifact.
  Its provider-child discovery uses Linux `/proc`.
- [The Just interface](../../justfile) has five public tasks; live selection fails
  explicitly. [CI](../../.github/workflows/ci.yml) runs credential-free mock tests
  and a local build on Linux. Keep that required lane.

The platform checkout was inspected at
[`b13c86d`](https://github.com/spoke-sh/platform/tree/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4).
Its [native management profile](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/packages/management/README.md)
and [local qualification](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/qualification/milestones-2-4.md)
now provide more than the previous provider assessment: signed native commands,
observational reads, durable acceptance and receipts for a **restricted bucket
metadata profile**. It uses seeded scope and provides no object data plane or
full service graph. Older mock-v1 comparisons in the platform handoff are
historical; compare its actual schema against this repository's mock-v3 code.

The platform pins Transit `0e2fba44727ca858bd1e8e4a775b62fbcf1a665d`.
The [Go SDK at that revision](https://github.com/spoke-sh/transit/blob/0e2fba44727ca858bd1e8e4a775b62fbcf1a665d/sdks/go/README.md)
uses cgo and a shared native library, with Linux amd64 execution and a retained
Nix runtime closure. It does not provide portable native release artifacts or a
public module release. This is a packaging/release prerequisite to resolve, not
permission to claim four-platform support from the current pure-Go provider.

## Delivery order and gates

| Step | Dependencies | Reviewable result |
| --- | --- | --- |
| 0. Hard-cutover cleanup | Merged 5–8 | Remove legacy compatibility; current-contract behavior and negative version tests remain green. |
| 9a. Package and mirror | Step 0 | Exact Nix-built Linux artifact, version/provenance and normal Terraform installation. |
| 9b. Independent consumer | 9a and completed 8 | Bucket plus full graph/recovery through the package, with no compiler or source imports in the consumer. |
| 10a. Native contract integration | Platform management profile; may start alongside 9 | Signed boundary and durable pre-dispatch recovery qualified against the real local metadata service. |
| 10b. Full live graph | 9, 10a and platform service/controller/data-plane prerequisites | Installed provider controls real services and retains usable data after retirement. |
| 11a. Release preparation | 9; native dependency decisions from 10a | Executed CLI/system matrix, reproducible candidate artifacts and publication workflow. |
| 11b. Publication | 10b and 11a, namespace/signing access | Immutable public artifacts, Registry installation and installed/live verification of those exact bytes. |

Package, native metadata and release preparation can progress independently.
After the native dependency lands, rerun milestone 9 with the changed binary and
runtime closure; the earlier pure-Go package evidence cannot qualify it. Missing
platform services block 10b and the live release claim, not packaging or honest
partial native-boundary evidence. Do not bypass those gates or mark 10 complete.

## Step 0: remove the compatibility layer

1. Remove [state_upgrade.go](../../internal/provider/state_upgrade.go), its
   `ResourceWithUpgradeState` integration, the installed
   [v2 migration test](../../internal/provider/migration_linux_test.go), and
   `LoadV2Ledger` plus its migration-only tests in the fixture. Remove the active
   `testdata/migration/` corpus and obsolete mock-v1/v2 contract documents/schemas;
   their committed history remains available as historical evidence.
2. Remove compatibility-only behavior such as accepting an untyped bucket spec,
   defaulting absent typed storage settings for old state, and preserving legacy
   intent encodings. Inventory each fallback before deletion so ordinary current
   defaults and safe current-version recovery remain covered.
3. Keep schema/contract versions explicit and monotonic. Reject unsupported
   prior state and ledger versions; do not reset a schema version to make old
   state look current. Bump the mock contract when its required wire shape changes.
   Recreate only isolated test fixtures/roots for the new version.
4. Add focused negative tests for old routes/headers, old ledger loads, stale
   state versions and old fingerprints. Assert rejection and zero management
   mutations. Keep positive current-version CRUD/import, empty-plan, scope,
   protection, pending-operation and process-restart tests.
5. Update active contract docs, examples, README and capability matrix. Preserve
   completed milestone/qualification records as history, using pinned historical
   links when removed files were cited. The old migration suite stops being a CI
   obligation. This plan changes policy now; cleanup is the first implementation
   commit, not code already removed by this documentation change.

Exit: one active mock contract with no migration adapter or legacy fallback;
current-contract acceptance/recovery still passes. Run the full existing check
and build gate after cleanup.

## Milestone 9: Nix packages and independent consumers

### Artifact definitions and identity

Add `nix/provider.nix` and `nix/mirror.nix`, exposed through
`packages.<system>.terraform-provider-spoke` and
`packages.<system>.terraform-provider-spoke-mirror`. Start with an executable
Linux amd64 profile; define other outputs only when their dependencies build.
Keep development shell evaluation separate from advertised package support.

Use the locked toolchain and `buildGoModule` with a verified `vendorHash`, a clean
source filter and explicit root-package build. Include files needed by tests and
generated contracts; exclude state, credentials and local build products. Dependency
fetch/build must not provision a service. Follow the
[Nixpkgs Go packaging guidance](https://nixos.org/manual/nixpkgs/stable/#sec-language-go).

Introduce one checked-in release-version source and embed version, commit and
contract/profile identity. Give development artifacts commit-qualified prerelease
versions such as `0.1.0-dev.g<revision>`; reject ambiguous dirty-tree publication.
Do not reuse the harness's `0.0.1` as a distribution identity. Record dependency
hash, derivation/output identity and binary digest in qualification evidence.

Package the executable with its exact versioned filename. The unpacked mirror
layout is `registry.terraform.io/spoke-sh/spoke/VERSION/OS_ARCH/`; configuration
includes only Spoke from that mirror and excludes Spoke from direct lookup while
allowing other providers normally. Use a scoped `TF_CLI_CONFIG_FILE`, without
changing the user's global configuration. These are Terraform's documented
[installation rules](https://developer.hashicorp.com/terraform/cli/config/config-file#explicit-installation-method-configuration).

The native integration must add pinned Go binding/ABI inputs and the native
runtime library to the closure. Build/install the matching SDK from immutable
source inputs rather than assuming its illustrative local archive version is a
public Go module release. Resolve library paths at packaging time and
check the ABI at startup. Neither a sibling checkout nor ambient library lookup
is a dependency source. A native package must fail clearly on unsupported systems;
do not switch to another protocol when native loading fails.

### Consumer and harness

1. Separate artifact construction from `newConsumer` and process-kill helpers.
   Let tests receive an exact binary/mirror/version and verify the launched
   executable. The Nix-artifact lane must never call `go build` implicitly.
2. Add `examples/installed-consumer/` with a flake pinned to a committed provider
   revision, exact Terraform provider constraint and checked lockfile. Export or
   copy the example to an isolated workspace outside this checkout for execution;
   no `path:../..`, Go source imports or `dev_overrides`.
3. Package a test-only fixture launcher separately from the provider. Start the
   synthetic TLS service in the harness, supply runtime file paths and scope IDs,
   and retain its ledger for restart tests. No credential or journal enters the
   Nix store. The consumer gets Terraform, the artifact/runtime closure and small
   shell utilities, with no Go/Zig/C compiler on its PATH.
4. Establish the lock using the trusted exact mirror and target platform, then
   require `terraform init -lockfile=readonly`. Mirror-generated checksums prove
   the selected local artifact; they are distinct from publisher-signed Registry
   checksums. Verify tampering, missing version/platform and accidental direct
   lookup fail. See the official
   [provider lock command](https://developer.hashicorp.com/terraform/cli/commands/providers/lock).
5. Run bootstrap then bucket and complete service graph lifecycle/import/empty
   plan, alias isolation, partial apply, provider death and fixture restart using
   the Nix artifact. Preserve selective retirement and canary checks. Reuse the
   assertions from the existing installed suite rather than substituting init-only
   checks. Test coexistence with an independently pinned second provider in the
   installation lane without adding a credential requirement.

Add package/check outputs for bounded build/install checks. Keep long acceptance
in the harness if the Nix build sandbox cannot host its loopback fixture; document
that distinction. Extend existing Just tasks and CI steps, keeping five public
tasks and the default listing. Cache keys must include source/toolchain/native
ABI inputs; cache hits never replace artifact identity checks.

Exit: [milestone 9 acceptance](09-nix-packages-and-consumers.md#acceptance-criteria)
passes through the Nix-built artifact for the bucket and now-available full graph.
Record other systems as evaluated, built or executed, never interchangeably.

## Milestone 10: native boundary and real service qualification

### Contract reconciliation and durable recovery

Build a field-by-field divergence matrix from the pinned
[native management schema](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/contracts/management-native-v1.schema.json)
and [Serving contract](https://github.com/spoke-sh/transit/blob/0e2fba44727ca858bd1e8e4a775b62fbcf1a665d/contracts/HTTP_SERVING_V1.md).
At minimum reconcile:

| Area | Work required from mock-v3 |
| --- | --- |
| Transport/authority | Replace bearer REST assumptions with native signed sends and current authorized read proofs; validate exact target, realm, ingress/binding and credential epoch. |
| Scope/shape | Reconcile seeded numeric scope/incarnation IDs, 16-byte management keys, bounded names and the fixed bucket command profile with current string/typed service specs. Unsupported fields/kinds fail before submission. |
| Intent and stages | Encode the native canonical payload and preconditions; distinguish Serving acceptance, service admission, management acceptance, terminal business result and runtime health. HTTP 202 alone is insufficient. |
| Receipts | Inspect exact original generation/operation, current binding and resource-lifetime retention. Receipt unavailability never authorizes a fresh create. Do not impose mock tombstone/expiry semantics on the native owner. |
| Reads | Implement the proof context, current authority checks, bounded pending pages and revision cuts. Do not claim the current pending-only API supplies full resource collection discovery. |

First run a small SDK integration against the actual local management package:
prepare/export/restore/dispatch, signed read-proof support, error/result decoding,
ABI loading and cancellation. Pin supported generated inputs from the same Transit
revision. Identify missing SDK or platform capabilities with their upstream owners
before adding an adapter; do not reconstruct a competing signing/authority engine
or access private platform tables.

Native sequence and exact signed work must survive provider death **before the
first dispatch**. Plan an explicit external recovery journal, outside Terraform
state and the Nix store, with atomic durable writes and exclusive locking. Key it
by exact native target/principal/management identity and generation, store the
exported prepared operation and its digest, and reconcile it before later intent.
Do not persist private signing seeds. Treat replayable envelopes as private files.
Use one dedicated native principal and shared locked journal per automation owner;
refuse conflicting local use. Shared-principal distributed sequence allocation
requires a supported coordinator and explicit qualification before support claims.
Test crash-before-write, write-before-dispatch, lost acceptance, restart, corrupted
journal, missing journal and principal/target mismatch. No missing-file fallback
may allocate a fresh sequence for an uncertain operation.

Align the current test contract to the selected platform profile by hard cutover.
Do not retain a mock-v3-to-native compatibility translator or infer a protocol
from a failed request. A limited native metadata capability can be qualified
explicitly while wider mock-only families remain experimental; their active mock
profile must be clearly separate from the native profile, with no transparent
fallback. For each adopted native kind, replace its old fixture/transport contract
rather than retaining the bearer implementation as a compatibility path. Remove
a superseded profile when its replacement is implemented.

### Live lane and backend gates

Add `just test-acc live` with explicit configuration and a separate workflow.
Use `internal/testutil/liveplatform/` and shared lifecycle assertions with a real
harness adapter; ordinary PR CI remains entirely synthetic. Require:

- Exact artifact, platform/Transit/API revisions and allowed capability profile.
- Explicit endpoint, verified CA, target scope, external credential file and
  native recovery-journal path; no ambient credentials selecting the lane.
- An isolated fixture allocation/run identifier, budget/deadlines, restart/fault
  controls and an inventory of fixture-owned objects for bounded cleanup.
- A required case manifest and machine-readable results. Missing configuration,
  unsupported required capabilities, zero selected tests or skipped required cases
  fail the selected gate. Partial metadata qualification is labeled partial and
  cannot satisfy full-graph acceptance.

Management bootstrap belongs to the harness/operator. Customer Terraform uses an
already available management service and known bootstrap outputs. Error paths
preserve retained data and uncertain receipts; cleanup reports leftovers and
recovery references instead of broadening scope or forcing deletion. Never expose
fault-control endpoints as public provider resources.

Platform prerequisites remain with their owners at the inspected revision:

| Provider gate | Platform owner |
| --- | --- |
| Native schema, automation credentials, receipt/sequence semantics and deployment placement | [P4 management](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/roadmap/04-management-api-and-operation-recovery.md) |
| Customer project/environment/account/grant APIs | [P5 scope/access](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/roadmap/05-customer-scope-and-access.md) |
| Actual database and object effects/data APIs | [P6 data](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/roadmap/06-managed-databases-and-object-storage.md) |
| OCI registry/import/digest policy | [P10 artifacts](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/roadmap/10-artifact-registries-and-isolated-builds.md) |
| Runtime secrets, bindings, workloads and TLS ingress | [P11 workloads](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/roadmap/11-workload-isolation-bindings-and-endpoints.md) |
| Isolated complete graph and independent data-plane evidence | [P13 graph qualification](https://github.com/spoke-sh/platform/blob/b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4/docs/roadmap/13-initial-service-graph-qualification.md) |

Refresh these contracts during implementation. Record actual service coverage and
exact owner agreements; this plan neither implements upstream services nor claims
that the metadata profile satisfies them.

### Executable live journey

Use the installed provider for organization bootstrap, explicit grants, registry,
database, bucket, secret reference, container, bindings and endpoint. Run CRUD,
import, empty plans, drift, revision/owner conflicts, cross-tenant denial and
protected/nonempty/in-use deletion. Inspect independent backend IDs and receipts.

External clients must push/pull an actual OCI digest, write/read bucket bytes and
database rows, resolve a pinned secret into the authorized workload without
exposing it to Terraform, and verify endpoint reachability/TLS. Update digest and
secret version, revoke/restore access, observe failed/pending rollout distinctly
from declaration outcome, then retire compute/bindings/endpoints and read the
retained data/artifacts again. Mock controller controls cannot establish any of
these data-plane results.

Kill the installed provider at deterministic acceptance/response barriers; lose
replies and restart actual management/controller processes over retained state.
Verify original signed-operation recovery, no duplicate resources and continued
authorization checks. Use backend fault hooks or a controlled transport harness,
not timing guesses or a passing process health check alone.

Exit: [milestone 10 acceptance](10-live-platform-qualification.md#acceptance-criteria)
passes for the complete declared profile. Publish a capability matrix and evidence
separating mock, local native metadata, real service/data plane and cloud placement.

## Milestone 11: current-profile compatibility and distribution

### Matrix and artifact portability

Choose exact CLI versions at implementation time: keep 1.15.3 as the recorded
baseline and select a pinned current version/minimum for the release. The old 1.10
candidate is historical, not a requirement to support older CLIs. Claim only
executed combinations; a newer version in upstream
documentation is not evidence that this provider supports it. Test protocol 6,
validation, import, saved plans, same-contract restart and the installed/live
journey. Reject incompatible API/state versions before mutation. Do not add
positive historical mock/state upgrade tests before the full-roadmap gate.

Qualify native Linux amd64 first. Add Linux arm64 and macOS amd64/arm64 only after
native dependencies and process-death tests work there; port child-process control
away from Linux `/proc` where needed. Missing runners or native SDK support remain
explicit exclusions. Native ABI/protocol failures must preserve known state and
produce bounded, redacted diagnostics.

Resolve portability before publishing a native-backed Registry artifact. A ZIP
that depends on a build machine's Nix store cannot be called a standalone portable
provider. Obtain a supported redistributable native library/runtime package or a
qualified portable link strategy from the upstream owner; include its libraries,
licenses and provenance as needed and verify on a clean host without Nix. Until
that passes, report Nix-only installation accurately and keep the public-portable
release gate open. Do not substitute an unqualified pure-Go protocol implementation.

### Release construction and publication

1. Establish a release artifact owner: stage each qualified executable/runtime
   tree once, create its archive, and reuse that exact staged payload in the Nix
   mirror and published distribution. Check extracted file digests before signing.
   Nix-local source builds that differ use their own immutable prerelease identity;
   never teach consumers to accept differing bytes for the same version/platform.
2. Add versioned ZIPs, a protocol-6 Registry manifest, checksums, detached GPG
   checksum signature, source/native-dependency provenance and generated docs.
   Recheck the exact namespace, key and asset requirements before publication;
   Terraform documents the required
   [release format and signing](https://developer.hashicorp.com/terraform/registry/providers/publishing).
3. Add `.github/workflows/release.yml` and a pinned artifact-builder configuration.
   Build/test jobs are read-only; only publication receives release-write access
   and signing credentials. Bind artifacts to the exact tag commit and tested
   dependency inputs. Keep signing keys out of Nix derivations and ordinary PR jobs.
   Validate the full asset manifest before upload; retry partial publication by
   checking existing digests, never replacing differing public assets.
4. Confirm `spoke-sh/spoke` namespace access, repository visibility and signing-key
   registration. Execute publication only after the concrete candidate passes
   the selected matrix and live gate. Public availability is a separate result
   from authenticated service access.
5. From a clean external consumer, fetch publicly without GitHub credentials or
   a local mirror, verify version/signatures/checksums and readonly initialization,
   then run the documented installed/live profile. Verify that the release mirror
   and public artifact agree. Test a changed checksum, missing version/platform,
   unsupported contract and interrupted download as explicit failures.
6. Publish the current supported operation/CLI/system/API matrix, exact artifact
   identities and exclusions. Document breaking hard cutovers and safe selection
   of an older immutable artifact without claiming that switching a binary rolls
   back accepted platform work. Remain experimental/pre-1.0 while the roadmap
   no-backwards-compatibility policy is in effect.

Exit: [milestone 11 acceptance](11-compatibility-and-releases.md#acceptance-criteria)
requires an actual publication and clean install/live evidence. A workflow file,
private draft, mirror-only install or successful cross-compile does not close it.

## Implementation owners and logical commits

| Commit group | Files/components | Validation |
| --- | --- | --- |
| 1. Hard mock cutover | Legacy state/ledger handlers and corpora; current contract/docs/tests | Unsupported-version rejection without effects; full current mock suite |
| 2. Package identity | `flake.nix`, `nix/`, version/provenance metadata | Locked dependency hash, package build, startup/version/closure checks |
| 3. Mirror and external consumer | Artifact-injected harness, test fixture launcher, `examples/installed-consumer/` | Checksum/readonly init, negative install tests, bucket/full graph and killed-provider recovery |
| 4. Native boundary | `internal/platformclient/`, pinned SDK/ABI inputs, external journal and current-profile fixtures | Independent encoding/proof vectors, native metadata calls, exact prepared-operation restart and redaction |
| 5. Live lane | `internal/testutil/liveplatform/`, live acceptance, dedicated workflow | Explicit preflight, required-case enforcement, isolation and bounded cleanup |
| 6. Real graph qualification | Platform adapters, shared assertions, capability/evidence records | Complete real management/data-plane/restart/retained-data journey |
| 7. Matrix and release preparation | Native portability, CLI/system jobs, Registry manifest and artifact workflow | Selected native targets, archive/runtime verification, signatures and immutable retries |
| 8. Published consumer evidence | Release docs, public install fixture, `qualification/`, roadmap | Anonymous installation and installed/live checks for the exact published revision |

Keep each slice reviewable and buildable. Record upstream blockers with the
concrete missing capability; do not close a milestone with skipped work. Phase IV
services are outside this implementation plan but remain part of the all-milestone
compatibility gate. No cross-repository code changes or publication are performed
by writing this plan.

## Validation and review gates

For implementation, run `just fmt`, `just check` and `just build` in the pinned
shell. Nix changes also require all-system evaluation and native package builds on
every advertised system. Generated reference updates use
`scripts/generate-docs.sh`, preserving `docs/roadmap/`. Keep ordinary checks free
of real credentials and retain the five-task/default-listing interface.

Evidence must identify provider/platform/Transit/API revisions, binary/archive
hashes, Nix output/dependency identity, CLI/system, selected cases, outcomes and
limits. Update numbered milestone and index statuses only when their exit gates
have executed. Planning itself needs relative-link/fragment, numbering, dependency
and status checks plus `git diff --check`; it does not require code tests.

Review the hard-cutover removal inventory, exact artifact identity/portability
strategy, external journal/sequence ownership, explicit native-versus-real-service
gates, and first release's supported matrix. Those decisions determine whether
subsequent implementation is ready; no old mock migration is a prerequisite.
