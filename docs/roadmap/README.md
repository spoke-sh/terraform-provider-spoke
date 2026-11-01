# Roadmap

This roadmap owns milestone scope, dependencies, acceptance criteria and status
for the Spoke Terraform provider. The sequence follows delivery dependencies,
not dates. It mirrors the numbered milestones, phase index, delivery checkpoints
and evidence-based completion rules in the
[Transit roadmap](https://github.com/spoke-sh/transit/tree/c43fef806eebf3c04fbfa3de504e0d00883b772c/docs/roadmap).

Current usage and supported behavior belong in the [repository README](../../README.md)
and [generated provider reference](../index.md). The
[implementation tracker](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
retains the original plan; this roadmap decomposes its remaining work after
[PR #2](https://github.com/spoke-sh/terraform-provider-spoke/pull/2). A planned
milestone does not establish support. Mock, process-recovery, installed-consumer
and live-service evidence must remain distinguishable.

Planning lives in `docs/roadmap/`. Use
[`scripts/generate-docs.sh`](../../scripts/generate-docs.sh) to regenerate the
provider reference while preserving hand-maintained plans.

The supporting [milestones 5–8 implementation plan](milestones-5-8-implementation-plan.md)
defines the contract/fixture changes, resource delivery steps, logical commits
and qualification gates for phase II. Milestone 4 is the completed baseline;
milestones 5–8 are completed against mock-v3; live service qualification remains
separate.

The [milestones 9–11 implementation plan](milestones-9-11-implementation-plan.md)
starts from merged PR #7 and covers Nix artifacts, independent consumers, native
management adaptation, real-service qualification and publication. Implementation
has completed the hard mock-v4 cutover and milestone 9; milestones 10 and 11 have
codec/live-gate and release-candidate foundations with explicit upstream blockers.
See the [execution record](../../qualification/milestones-9-11.md).

The [remaining milestones 10–17 plan](milestones-10-17-implementation-plan.md)
uses provider `29230f8`, Transit `a000b8d` and Platform `7af27c2`. It supersedes
the earlier remaining-work assessment: Platform now supplies native v2 bucket
metadata, settlement authorization, tombstones, complete discovery and local TLS
placement. Shared SDK adoption, the real service graph and publication remain
open. The plan sequences those gates and all six later service families without
introducing backwards compatibility. Implementation now delivers native-v2
application codecs and the mock-v5 provider surfaces for milestones 12–17.
Milestones 10–17 remain **In Progress**: the [execution record](../../qualification/milestones-10-17.md)
separates provider/mock evidence from missing SDK, service and publication gates.

The [current native integration](../../qualification/native-boundary.md) now adds
the shared Serving SDK, explicit native metadata profile and durable dispatch
journal on top of mock-v5. This supersedes the earlier missing-SDK assessment;
all full-graph and later-family live gates remain open.

## Development compatibility policy

Use hard cutovers and maintain no backwards compatibility until all currently
defined milestones 1–17 are complete. This covers old mock contracts, fixture
ledgers and Terraform state; remove superseded migration/fallback code as part of
each cutover. Preserve current-contract recovery and data protection,
reject unsupported versions before mutation, and keep published artifacts immutable.
Earlier migration evidence remains historical, not a continuing requirement.
Milestone 11 may publish an experimental initial release and qualify its current
CLI/API/system matrix without promising historical upgrades.

## Source baseline and delivered scope

The provider baseline is merged revision
[`ac0995d`](https://github.com/spoke-sh/terraform-provider-spoke/commit/ac0995d1c1f4b205427b5340fce3f0ac560591d0),
assessed on 2026-10-31. It implements protocol 6, an HTTPS client, a provisional
mock-v1 contract, bucket lifecycle, exact-ID project/environment discovery,
credential-free mock CI and a Linux provider-process recovery test using an
installed binary. [Milestone 1](01-mock-provider-foundation.md) records that
bounded outcome and its executed CI evidence.

At that baseline the mock ledger was in memory, with no published platform
management contract, qualified live service, Nix provider package or Registry
release established. The test mirror is built by the test harness; it does
not complete downstream Nix packaging. Only Go 1.26.7 / Terraform 1.15.3 on
Linux amd64 has recorded execution evidence. [HANDOFF.md](../../HANDOFF.md) preserves
the earlier source assessment; its platform observations are historical and must
be refreshed during contract reconciliation.

Milestones 2–4 now extend that foundation with mock-v2 contracts/discovery,
persisted fixture restart and interrupted CRUD evidence, plus scope/access
resources. See the [qualification record](../../qualification/milestones-2-4.md)
for implementation revision `32d6894`, executed checks and remaining limits.

Milestones 5–8 add typed mock-v3 services, preserved v2 state migration,
registries/digests, databases/buckets, references/bindings and containers/endpoints.
The [service graph qualification](../../qualification/milestones-5-8.md) records
revision `201bf93`, installed graph/recovery evidence and the live-service limits.

## Delivery phases

| Phase | Milestones | Exit outcome |
| --- | --- | --- |
| I: Contract and recovery foundation | 1–3 | The existing bucket slice has explicit discovery, ownership and recovery contracts plus deterministic failure and restart evidence. |
| II: Initial service graph | 4–8 | Terraform declares scope, access, registries, databases, buckets, bindings, container services and endpoints through the mocked management boundary. |
| III: Installation, live qualification and release | 9–11 | An external consumer installs immutable artifacts, the initial graph passes real management and data-plane tests, and a qualified independent release is published. |
| IV: Additional service lifecycles | 12–17 | Jobs, messaging, volumes, advanced networking, recovery services and placement are added only with their own backend contracts and qualification. |

Each milestone appears in exactly one phase. Phases group user outcomes; they
do not add implicit dependencies. Contract review starts immediately. Packaging
can proceed from the existing foundation while resource work continues. Phase IV
is subsequent scope, not a prerequisite for the initial release in milestone 11.

## Milestone sequence

### Phase I: Contract and recovery foundation

| Order | Milestone | Status | Intended outcome |
| --- | --- | --- | --- |
| 1 | [Mock provider foundation](01-mock-provider-foundation.md) | Completed | One observable bucket lifecycle, scoped discovery, required mock CI and a separate-process create recovery proof. |
| 2 | [Management contracts and bounded discovery](02-management-contracts-and-discovery.md) | Completed | Capability matrix, explicit ownership and receipt rules, complete bounded discovery and a path to the platform-owned schema. |
| 3 | [Recovery, isolation and data protection](03-recovery-isolation-and-protection.md) | Completed | Interrupted CRUD, durable fixture restart, independent-state conflicts, denial and bounded failure tests before expanding resources. |

### Phase II: Initial service graph

| Order | Milestone | Status | Intended outcome |
| --- | --- | --- | --- |
| 4 | [Scope and access](04-scope-and-access.md) | Completed | Projects, environments, service accounts and explicit grants without credential values or a provider-configuration cycle. |
| 5 | [Artifact registries and digest discovery](05-artifact-registries.md) | Completed | Scoped registries, retention and immutable artifact lookup with independently controlled upload and deployment. |
| 6 | [Databases and object storage](06-databases-and-object-storage.md) | Completed | Managed database lifecycle and expanded buckets with explicit capacity, retention and data protection. |
| 7 | [Secret references and service bindings](07-secret-references-and-bindings.md) | Completed | Exact service dependencies and externally managed secret references without implicit grants. |
| 8 | [Container services and endpoints](08-container-services-and-endpoints.md) | Completed | Digest-based deployment, inspectable rollout, explicit ingress and dependency-safe retirement of the complete mocked graph. |

### Phase III: Installation, live qualification and release

| Order | Milestone | Status | Intended outcome |
| --- | --- | --- | --- |
| 9 | [Nix packages and installed consumers](09-nix-packages-and-consumers.md) | Completed | Versioned provider packages and mirrors installed by an external consumer without a compiler or development overrides. |
| 10 | [Live platform qualification](10-live-platform-qualification.md) | In Progress | Published API reconciliation and real management, recovery, authorization and data-plane evidence for the initial graph. |
| 11 | [Compatibility and release distribution](11-compatibility-and-releases.md) | In Progress | Tested current CLI/API/system matrix, hard-cutover policy, independent versions and verified public installation. |

### Phase IV: Additional service lifecycles

| Order | Milestone | Status | Intended outcome |
| --- | --- | --- | --- |
| 12 | [Jobs and schedules](12-jobs-and-schedules.md) | In Progress | Declarative job definitions and schedules with observable executions and safe retirement. |
| 13 | [Streams and queues](13-streams-and-queues.md) | In Progress | Scoped messaging services with retention, consumer authority and protected deletion. |
| 14 | [Volumes and attachments](14-volumes-and-attachments.md) | In Progress | Persistent storage with explicit attachment, fencing and retained data across workload replacement. |
| 15 | [Custom domains and private connectivity](15-domains-and-private-connectivity.md) | In Progress | Verified domains, certificate lifecycle and explicitly authorized private network connections. |
| 16 | [Backup policies and database clones](16-backups-and-clones.md) | In Progress | Declarative recovery policy and isolated clones with verified restore provenance and retention. |
| 17 | [Scaling and placement policies](17-scaling-and-placement.md) | In Progress | Bounded capacity and placement intent with one reconciler per field and observable transitions. |

## Delivery checkpoints

| Phase | Checkpoint | Owners | Execution and exit gate |
| --- | --- | --- | --- |
| I | Finish the reusable boundary | 2–3 | Resolve draft contract gaps, then run every failure case through actual HTTP and Terraform callers; retain uncertainty and original identity across process and fixture restarts. |
| II | Declare ownership and services | 4–6 | Start with organization-scoped bootstrap and grants; add independent registry/database/bucket lifecycle slices with import, drift and deletion tests. |
| II | Compose an application | 7–8 | Connect exact service IDs, deploy an immutable digest, inspect readiness, update, revoke and retire compute while preserving separately owned services. |
| III | Install outside this checkout | 9 | Build the Nix artifact and versioned mirror, initialize a clean consumer with checksums, then run lifecycle and crash recovery using that artifact. |
| III | Prove real service behavior | 10 | Reconcile the published API and execute the graph, interrupted operations and real OCI/object/database/endpoint checks in an isolated live fixture. |
| III | Publish one qualified revision | 11 | Run the compatibility matrix, package exact bytes, publish immutable assets and verify anonymous installation and the installed consumer. |
| IV | Extend independently | 12–17 | Add a versioned service contract and mock slice first; apply the same installed/live/release gates before claiming support for each addition. |

## Ownership and contract gates

This repository owns Terraform schemas and lifecycle adaptation, the management
client, mocks, tests, examples, generated documentation, Nix packaging and
provider releases. `spoke-sh/platform` owns service APIs, product policy,
provisioning and service guarantees. `spoke-sh/transit` owns generic runtime,
storage and authorization primitives. Link upstream prerequisites when verified;
do not implement missing service APIs by accessing private tables or cloud CLIs.

Each resource needs a recorded management operation, authentication and scope,
stable identity, revision conditions, acceptance/settlement behavior, receipt
recovery, deletion restrictions and executable evidence. Draft mock contracts
allow provider development before the backend exists; they do not satisfy the
live gate. Read APIs remain observational. Configure an already available
management service; Terraform must not require a service to create itself.

The draft binds management keys to authenticated principals and verifies
independent-owner conflicts. Identical principal credentials and keys still cannot
distinguish a cloned state from legitimate recovery; that case remains explicitly
unsupported. Milestones 2–3 define and test these limits; milestone 10 requires
platform agreement before live adoption. No roadmap wording grants the draft
stronger isolation than it has.

## Shared implementation gates

- Add contract and independent raw-HTTP fixture checks before resource adapters.
  Keep mock controls outside the proposed public API. Reads never advance work.
- For each resource, exercise validate, plan, apply, inspect, import, empty
  second plan, update, drift, denied access, stale revisions, interrupted work and
  restricted deletion through real Terraform and the actual HTTP client.
- Establish mutation identity before dispatch. Cancellation stops waiting;
  acceptance is not completion. Only authorized authoritative absence removes
  state. Keep response sizes, pages, retries and settlement bounded.
- Preserve separately owned data and artifacts. No implicit purge, force-empty
  or cascade. Keep credentials and secret values out of state, saved plans,
  logs, diagnostics, fixtures checked into Git and Nix outputs.
- Ordinary CI uses synthetic loopback fixtures with no platform credentials.
  Keep five public Just tasks and the private default listing recipe. Live
  acceptance requires explicit selection and fails on absent configuration;
  it never falls back to mocks. Missing or skipped required cases fail the gate.
- Publish evidence with exact source/artifact, API, Terraform and system
  identities. Nix evaluation and cross-compilation do not establish execution
  support; management state alone does not prove a working data plane.

## Relationship to the implementation tracker

| Issue #1 work | Milestone owner |
| --- | --- |
| Step 1: provider scaffold | 1, delivered |
| Steps 2–3: draft contract, fixture and HTTP client | 1, delivered subset; 2–3, completed draft contract and fault coverage; live adoption remains in 10 |
| Step 4: first Terraform lifecycle | 1, delivered bucket/discovery slice |
| Step 5: uncertainty, isolation and protection | 3; service-specific extensions completed in 4–8 against mocks |
| Step 6: initial resource graph | 4–8, completed against mock-v3; live behavior remains in 10 |
| Step 7: packaging and installed consumer | 9; release identity and publication in 11 |
| Step 8: platform reconciliation and qualification | 2, early contract coordination; 10–11, live and release evidence |
| Later service candidates | 12–17; optional bounded bucket-object resources remain a decision in 6 |

The original tracker is not complete because the foundation merged. Its broad
checkboxes must not be marked done when only a narrower slice has evidence.

## Maintaining the sequence

Keep one numbered, descriptively named file per milestone and one ordered index.
Use `Planned`, `In Progress` and `Completed` consistently in the header and index.
Record actual prerequisites; a lower number alone is not a technical dependency.
When inserting or reordering, update filenames, inbound links, phases and
checkpoints together. Split only independently reviewable outcomes.

Keep delivered scope separate from remaining work. Completion requires the
milestone's acceptance criteria, linked executed results and current behavior
documented in the README/reference or its owning component. Evidence should
identify a committed revision, command or CI run, fixture kind and limitations.
Keep completed milestone files. Reopen unmet criteria as `In Progress` while
preserving prior evidence. Moving work or merging code does not complete a gate.
