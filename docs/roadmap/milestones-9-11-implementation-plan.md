# Milestones 9–11 implementation plan

[Roadmap](README.md) · **Plan status:** In Progress · **Review:** Implementation authorized

This plan records milestone 9's delivered packaging foundation and the remaining
HTTP integration/release work in milestones 10–11. Revised 2026-11-01: the provider
uses a thin Go client for Platform's public HTTPS management API. Platform owns
Transit integration; no provider native SDK, ABI/library packaging, native signing
or principal-sequence journal is required.

The [milestones 10–17 plan](milestones-10-17-implementation-plan.md) owns the
current detailed HTTP contract, recovery, live and release tasks. The
[original reviewed plan](https://github.com/spoke-sh/terraform-provider-spoke/blob/29230f8015adddb6c2010993aa57f4d3313d32b2/docs/roadmap/milestones-9-11-implementation-plan.md)
and [execution record](../../qualification/milestones-9-11.md) preserve historical
native assumptions and test results. Those assumptions are superseded, not
additional completion criteria. Milestone 9 is **Completed** at its recorded mock
scope; milestones 10 and 11 remain **In Progress**. This revision changes plans,
not the existing native-only preflight or publication implementation.

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
| Initial system | Linux amd64 first. Evaluate other Nix systems separately; advertise additional systems only after execution and recovery evidence on those systems. |
| HTTP adapter | Adopt Platform-owned HTTP schemas, auth, idempotency and operation lookup with Go `net/http`; keep Transit protocols and execution behind Platform. |
| Initial release | Milestone 11 can close for a qualified initial profile before milestones 12–17. It tests selected CLI/system/API combinations, not historical mock migrations. |

A new test contract uses fresh isolated fixture state and a new Terraform test
root. Incompatible old state/ledgers/receipts must fail before any platform
mutation, with an actionable diagnostic. Never automatically discard state,
replay an old envelope as new intent, or purge a remote service. An operator may
inspect/finish outstanding work using its original pinned environment; retaining
that artifact does not require maintaining its code path in the new provider.

## Baseline and refreshed upstream assessment

The following describes the historical PR 7 planning baseline. Current source and
HTTP prerequisites are in the [remaining plan](milestones-10-17-implementation-plan.md#source-basis-and-changes-since-pr-8).

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
public module release. That previously motivated the native plan. It is no
longer a provider prerequisite: Platform owns any such backend dependencies, and
the provider independently qualifies its Go HTTP executable on advertised systems.

## Delivery order and gates

| Step | Dependencies | Reviewable result |
| --- | --- | --- |
| 0. Hard-cutover cleanup | Merged 5–8 | Remove legacy compatibility; current-contract behavior and negative version tests remain green. |
| 9a. Package and mirror | Step 0 | Exact Nix-built Linux artifact, version/provenance and normal Terraform installation. |
| 9b. Independent consumer | 9a and completed 8 | Bucket plus full graph/recovery through the package, with no compiler or source imports in the consumer. |
| 10a. HTTP contract integration | Platform public HTTP schema, auth and recovery APIs | Go HTTP adapter and original-operation recovery against an isolated real service. |
| 10b. Full live graph | 9, 10a and platform service/controller/data-plane prerequisites | Installed provider controls real services and retains usable data after retirement. |
| 11a. Release preparation | 9; published HTTP profile from 10a | Executed CLI/system matrix, reproducible candidate artifacts and publication workflow. |
| 11b. Publication | 10b and 11a, namespace/signing access | Immutable public artifacts, Registry installation and installed/live verification of those exact bytes. |

Milestone 9 packaging and the hard cutover are delivered. Requalify exact
artifacts after HTTP adaptation. Release preparation can proceed alongside live
integration; only real service evidence qualifies the advertised surface. The
expanded artifact includes all TF12–17 families, so a narrower initial release
must explicitly narrow both its exposed surface and required evidence.

## Step 0: remove the compatibility layer

Completed in PR 8; the sequence below is retained as implementation history.

1. Remove [state_upgrade.go](https://github.com/spoke-sh/terraform-provider-spoke/blob/dd75b117f249d482c06d77c94a3c97a5cc5ba579/internal/provider/state_upgrade.go), its
   `ResourceWithUpgradeState` integration, the installed
   [v2 migration test](https://github.com/spoke-sh/terraform-provider-spoke/blob/dd75b117f249d482c06d77c94a3c97a5cc5ba579/internal/provider/migration_linux_test.go), and
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

Package the Go HTTP executable with its actual dependencies. Platform API/schema
pins identify the remote contract; they do not introduce Transit ABI or runtime
inputs. Repeat installed lifecycle/recovery after client or packaging changes.
Clean-host release tests must prove no undeclared Nix-store runtime dependencies.

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
tasks and the default listing. Cache keys must include source/toolchain/Go dependency
inputs; cache hits never replace artifact identity checks.

Exit: [milestone 9 acceptance](09-nix-packages-and-consumers.md#acceptance-criteria)
passes through the Nix-built artifact for the bucket and now-available full graph.
Record other systems as evaluated, built or executed, never interchangeably.

## Milestone 10: HTTP boundary and real service qualification

Use the current [HTTP integration sequence](milestones-10-17-implementation-plan.md#milestone-10-platform-http-integration-and-complete-live-graph):

1. Platform publishes/implements HTTP authentication, scope/capabilities, resource
   schemas, revision preconditions, durable idempotency and original-operation
   lookup. Exact routes/auth/versions are upstream decisions; mock-v5 is provisional.
2. Adapt the Go HTTP client and cut over fixtures/schemas/docs together. Remove
   obsolete native codecs/vectors after their live/release consumers are replaced;
   keep gates functional until then. Retain historical evidence via immutable links.
3. Qualify stable operation identity across lost replies, a provider death before
   Terraform saves state, later generations and concurrent conflicting intent.
   Platform owns backend sequencing and durable acceptance. An unavailable receipt
   never authorizes fresh work or state erasure; the API must support recovery.
4. Replace the native-only live preflight/configuration and workflow with explicit
   HTTP API/artifact/Platform selections and real installed metadata cases. The
   isolated harness owns credentials, setup, fault controls, budgets and cleanup.
   Missing/partial/mock evidence still fails; ordinary CI stays credential-free.
5. Execute the full initial service graph and independent OCI/object/database/
   workload/TLS checks, revocation, actual controller restart and retained-data
   assertions using exact installed artifacts without a compiler or source tree.

Exit: [milestone 10 acceptance](10-live-platform-qualification.md#acceptance-criteria)
passes. Metadata-only results do not close it. Preserve historical native results
as evidence of their original scope, never proof of the revised HTTP integration.

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

Qualify the Go provider on Linux amd64 first. Add Linux arm64 and macOS
amd64/arm64 only after actual execution and process-death tests there; port child
process control away from Linux `/proc` as needed. Report missing runners as
exclusions. A cross-build or Transit runtime support matrix cannot establish
provider support. API failures preserve state and produce bounded diagnostics.

Qualify extracted ZIPs on clean hosts without Nix, compilers, source or Transit
libraries. Verify trust-store handling and actual executable dependencies; do not
publish artifacts with hidden build-machine Nix-store requirements. Replace
native-profile publication checks with the exact published HTTP contract identity
and complete real evidence for every advertised family. This gate replacement
remains implementation work and must keep rejecting mock/blocked reports.

### Release construction and publication

1. Establish a release artifact owner: stage each qualified executable/runtime
   tree once, create its archive, and reuse that exact staged payload in the Nix
   mirror and published distribution. Check extracted file digests before signing.
   Nix-local source builds that differ use their own immutable prerelease identity;
   never teach consumers to accept differing bytes for the same version/platform.
2. Add versioned ZIPs, a protocol-6 Registry manifest, checksums, detached GPG
   checksum signature, source/Go-dependency provenance and generated docs.
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
| 4. HTTP boundary | `internal/platformclient/`, Platform schema, current fixtures; remove native scaffolding | Raw-HTTP conformance, stable operation identity, recovery before saved state and redaction |
| 5. Live lane | `internal/testutil/liveplatform/`, live acceptance, dedicated workflow | Explicit preflight, required-case enforcement, isolation and bounded cleanup |
| 6. Real graph qualification | Platform adapters, shared assertions, capability/evidence records | Complete real management/data-plane/restart/retained-data journey |
| 7. Matrix and release preparation | Go portability, HTTP evidence gates, CLI/system jobs, Registry manifest and artifact workflow | Executed targets, clean-host archive verification, signatures and immutable retries |
| 8. Published consumer evidence | Release docs, public install fixture, `qualification/`, roadmap | Anonymous installation and installed/live checks for the exact published revision |

Keep each slice reviewable and buildable. Record upstream blockers with the
concrete missing capability; do not close a milestone with skipped work. Phase IV
services are outside this implementation plan but remain part of the all-milestone
compatibility gate. No cross-repository code changes or publication are performed
by writing this plan.

## Validation and review gates

For implementation, run `just fmt`, `just check` and `just build` in the pinned
shell. Nix changes also require all-system evaluation and package builds and execution on
every advertised system. Generated reference updates use
`scripts/generate-docs.sh`, preserving `docs/roadmap/`. Keep ordinary checks free
of real credentials and retain the five-task/default-listing interface.

Evidence must identify provider/Platform/HTTP-schema revisions, binary/archive
hashes, Nix output/dependency identity, CLI/system, selected cases, outcomes and
limits. Update numbered milestone and index statuses only when their exit gates
have executed. Planning itself needs relative-link/fragment, numbering, dependency
and status checks plus `git diff --check`; it does not require code tests.

Review the hard-cutover removal inventory, exact artifact identity/portability
strategy, server-backed idempotency/recovery, installed HTTP and real-service
gates, and first release's supported matrix. Those decisions determine whether
subsequent implementation is ready; no old mock migration is a prerequisite.
