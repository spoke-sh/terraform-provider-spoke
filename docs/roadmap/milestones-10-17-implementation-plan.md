# Implementation plan: remaining milestones 10–17

[Roadmap](README.md) · **Status:** Planned

Finish native integration and the initial qualified release, then add the six
remaining service families through their owning platform APIs. This plan is for
review; it changes no runtime behavior and closes no milestone. Milestones 10/11
remain **In Progress** and 12–17 remain **Planned**.

This supersedes the remaining-work assessment in the
[9–11 plan](milestones-9-11-implementation-plan.md), preserving that plan and its
[executed evidence](../../qualification/milestones-9-11.md). Milestone 9 is already
complete for the mock artifact; repeat its installation checks whenever native
dependencies change the distribution.

## Source basis and changes since PR 8

Fetched `origin/main` for all three repositories during this review. Pin these
revisions for the assessment; branch names and local prototypes are not inputs to
an implementation or qualification claim.

| Repository | Reviewed revision | Relevant state |
| --- | --- | --- |
| Provider | [`29230f8`](https://github.com/spoke-sh/terraform-provider-spoke/commit/29230f8015adddb6c2010993aa57f4d3313d32b2), merged PR 8 | Mock-v4, 11 resources/21 data sources, installed Nix consumers, unregistered native-v1 codecs, failing live preflight, exact-byte release candidates. |
| Transit | [`a000b8d`](https://github.com/spoke-sh/transit/commit/a000b8d18449fdee7901955d0c94417ceeab32eb) | Shared Catalog context, SQL recovery storage/leases, Queue waits, immutable single-Table branches and startup work have landed. Relevant milestones remain open. |
| Platform | [`7af27c2`](https://github.com/spoke-sh/platform/commit/7af27c2bb8083ab64348592f82247da1ff703596), merged PR 16 | Native management v2, scoped bucket metadata, local operator/TLS placement and installed-provider test harness. |
| Platform's Transit input | [`0e2fba4`](https://github.com/spoke-sh/transit/commit/0e2fba44727ca858bd1e8e4a775b62fbcf1a665d) | Still the deployed-contract build input in Platform's [lockfile][p-lock]; latest Transit has not been adopted there. |

### What is now available

- Platform's [v2 contract][p-contract] supplies desired bucket capacity/retention,
  current authorization at acceptance and settlement, terminal failure, explicit
  receipt expiry with durable tombstones, and complete object and pending lists.
  [Native qualification][p-evidence] records actual worker/authority/gateway
  restart, revocation, saturation and packaged TLS/bootstrap checks. These are
  no longer missing platform primitives for the metadata slice.
- The [placement tools][p-placement] run against an existing authority with
  distinct operator, worker, gateway, courier and customer identities. The
  platform harness installs an explicitly supplied provider through a locked
  mirror and has CRUD, import, denial, concurrent-process and twelve CRUD/process
  death cases. Reuse those controls and assertions.
- Transit's [shared context][t-context] composes bounded Catalog and SQL facets.
  [LocalRecoveryStore and SqlRecovery][t-recovery] provide private Linux files,
  exclusive ownership, immutable original bytes, checksummed definition-bound
  records, revision-checked evidence, atomic publication and file/directory sync.
  [Recorded tests][t-sdk-evidence] cover actual files and scripted SQL peers;
  they do not establish Serving recovery, principal sequence allocation, installed
  Go exposure or a killed Terraform process against a real service.
- [Streams and Queues][t-messaging] have typed lifecycle/payload callers, retained
  outcomes, fenced claims/settlement and bounded waits, including public Queue
  claim waits. Atomic dead-letter transfer, mixed transactions and complete
  integrated qualification remain open; avoid copying earlier dated gap tables
  that still describe claim waits as absent.
- [Table branch work][t-branches] now publishes a durable read-only branch of one
  selected Table, including its unflushed frontier. Whole-database cuts, writable
  isolation, full restore and published branch deletion remain open. This does
  not supply a managed writable database clone or a complete backup service.

### What remains blocking

The public [Go SDK][t-go] still exposes raw authority `PrepareSend`, export,
recovery, dispatch and capabilities. Its public `client` implementation has no
changes from Platform's Transit pin. It lacks Serving preparation/recovery/result
APIs, arbitrary native subject proofs for management reads, and verified custom
CA configuration. The new Zig data context/SQL store does not fill those gaps.
The Linux cgo SDK still requires its Nix native closure; portable native release
artifacts remain unavailable.

The generated [client ABI][t-abi] definition changed from `d8196134…` at Platform's
pin to `04cd820c…` at latest Transit. A version label alone cannot establish a
matching native protocol. Upgrade the platform, shared SDK, generated resources
and native library coherently and rerun the real v2 tests. Do not combine a new
library with the old platform definition or merely change the provider's pin.

Platform's [capability inventory][p-capabilities] still advertises **only bucket
metadata**. Its other ten initial kinds and all later customer services need
backends. Metadata has no object bytes, usable storage endpoint or healthy runtime.
Hosted placement/network evidence remains separate from the local tools.

Platform's evidence also describes a local, unpublished provider prototype that
signs directly in Go. Its harness is useful prior work; its signer and results do
not satisfy this repository's shared-SDK rule or qualify the merged provider.
Do not incorporate that signer or assume unmerged provider work is delivered.

## Ownership, ordering and cutover decisions

`TF` below means this provider roadmap; `P` means Platform and `T` means Transit.
Matching numbers in different repositories do not imply matching scope.

| Work | Required upstream handoff | Provider delivery gate |
| --- | --- | --- |
| TF10 metadata | T shared Serving/proof/TLS bindings; matched P4 v2 placement | Installed SDK-based bucket lifecycle and original-operation recovery. |
| TF10 initial graph | P5 scope/IAM, P6 data, P10A OCI, P11 workloads/secrets/endpoints, P13 isolated graph; P3 for hosted claims | Independent OCI/rows/bytes/runtime/TLS and retained-data evidence. |
| TF11 publication | TF10 full graph, native portability and publisher access | Exact public artifacts and anonymous locked installation on advertised systems. |
| TF12 jobs/schedules | P20 scheduler over P11 execution and P10A images | Real schedule/admission/retirement and interrupted-execution evidence. |
| TF13 streams/queues | P20 management over selected T6 messaging profile | Real produce/read or claim/settle, retention, revocation and recovery. |
| TF14 volumes/attachments | P20 storage/attachment over P11 execution; selected T22/P18 durability/fencing profile | Data across replacement and exclusion of stale writers. |
| TF15 domains/private connections | P20 plus P3/P11 network/endpoint owners | Actual ownership, DNS/TLS/private traffic, revocation and retained dependencies. |
| TF16 policies/clones | P18 recovery APIs; selected T19/T24 data/backup capabilities | Exact independently restored data and source/destination isolation. |
| TF17 scaling/placement | P17 compute policy; P19 residency/movement; T22–25 when the profile needs distributed state | Actual bounds/placement, one field owner, fencing and continuity. |

Upstream ownership is recorded in [Platform's alignment map][p-alignment],
[additional services][p-later], [recovery][p-recovery], [capacity][p-scaling] and
[placement][p-cells]. Consume their concrete service contracts; do not implement
customer APIs by issuing raw SQL, editing Transit tables or invoking cloud CLIs.

Delivery order is TF10 contract/SDK/journal → installed metadata → full graph →
TF11 publication. Release preparation can proceed alongside TF10. TF12–17
contract/mock work can proceed on its actual dependencies before publication;
none is a prerequisite for TF11's initial release. TF13 does not depend on TF12;
TF15 does not depend on TF14. Storage-aware TF17 depends on qualified TF14/P18/P19
profiles; stateless compute policy need not wait for distributed storage.

Maintain hard cutovers until all milestones 1–17 are complete. Replace old native
application codecs, fixture shapes, saved formats and state schemas together;
retain no v1/v2 or old mock decoder fallback in the provider. Reject incompatible
state, ABI and saved work before mutation. An operator can reconcile outstanding
work with its original immutable artifact/environment; a cutover never permits
resetting a sequence, deleting a journal or purging retained resources.

For each new family, agree a versioned platform-owned contract first. Until that
exists, label the provider contract explicitly provisional. Proposed Terraform
names below are review candidates, not existing APIs. Exact endpoint routes and
wire fields must come from the owning contract; do not invent production routes.
Keep one current fixture contract, with a new version for incompatible changes.

## Milestone 10: native integration and complete live graph

### 10A — Adopt the published v2 application contract

1. Replace [native-v1 codecs](../../internal/nativecontract/codec.go) and vectors
   with the pinned v2 schema, capability inventory and independent golden vectors.
   Encode capacity/retention, expire and terminal outcomes. Keep logical intent
   fingerprint separate from exact signed transport identity: v2 excludes expected
   management generation from its domain-separated logical hash but includes
   revision/incarnation. Validate scope against discovered placement before writes.
2. Implement strict v2 observational decoding and bounded traversal. Carry `next`,
   revision and opaque cursor together; a changed cut or gateway restart requires
   a bounded restart, never a fabricated complete list. Pending work is distinct
   from objects. Denial, receipt expiry, incomplete pages and unavailable resources
   preserve known state; only authorized authoritative absence removes it.
3. Record a field-level mapping for the existing bucket schema/import ID and
   explicit native configuration: endpoint/CA, realm, ingress/service/resource,
   seeded scope, principal/credential epoch, external seed and journal paths.
   Preserve native name/numeric bounds and protection defaults. Reject unsupported
   kinds/settings before dispatch; do not ignore fields to fit the 64-byte intent.
   Metadata observations must expose no usable object endpoint or readiness claim.
4. Change active fixtures, schema versions, examples, generated reference and
   capability checks in the same cutover. Explicit profile selection must never
   trigger negotiation back to the mock protocol. Provisional families cannot be
   advertised as supported on a native endpoint merely because they have schemas.

**Gate:** independent binary/hash vectors, malformed/current/foreign-profile
checks and observational pagination pass without a signer or live-support claim.
This package can start before the SDK is extended.

### 10B — Obtain and qualify the shared SDK handoff

Transit owns a reviewable public Go/shared-native interface that can:

- Prepare/sign a Serving command, export its exact original work, restore it
  without creating a new identity, and dispatch it through the proper ingress.
- Inspect original Serving acceptance, admission, disposition and result with
  bounded cancellation/close semantics; a 202 or native completion is not CRUD
  success. Preserve finite business refusals and uncertain transport outcomes.
- Create the exact subject proofs for management observations, binding definition,
  realm, principal, credential epoch, role when used, ingress audience and context.
- Use the configured HTTPS origin and custom trust roots with host verification,
  no TLS downgrade, ambient credential selection or alternate protocol fallback.

Pin Go source, generated ABI resources and native libraries together. Coordinate
Platform's dependency upgrade and independent v2 signing/restart tests before
consuming a new Transit definition. Add installed SDK round trips, wrong-definition
and wrong-CA refusals, revocation, cancellation after dispatch and exact recovery.
Track these generic additions upstream; this provider remains an application
adapter and does not introduce its own canonical encoder or signer.

### 10C — Integrate durable recovery and principal coordination

Resolve store ownership with Transit before adding another generic storage
format. Prefer exposing the existing neutral `LocalRecoveryStore` through the
shared binding with a Serving adapter. It is currently Zig-only and its SQL
adapter cannot be used for management sends. If a provider-owned file adapter is
selected, document that decision and keep signing/protocol interpretation in the
SDK; meet the same original-byte, locking and durability contract.

The provider owns the mapping from Terraform intent to management key/generation,
native sequence, exported work and observed receipts:

1. Require an exclusive automation principal and a stable private local journal.
   Hold cross-process coordination across aliases and Terraform processes; an
   in-process mutex and read-then-increment capability call are insufficient.
   Detect unsupported concurrent use, retain unresolved sends before assigning
   later sequences, and document the same-principal/cloned-state limit.
2. Bind each record to profile/definition, endpoint/realm/ingress/scope, principal,
   key, generation, intent digest and exact exported envelope. Durably persist
   original work before the first network dispatch, then preserve later evidence
   without replacing the original. Seed bytes never enter records or state.
3. Initialize a new empty journal only through an explicit fresh-owner path.
   Missing/corrupt/foreign records during recovery must fail; they never authorize
   fresh work. Require private permissions, reject unsafe path substitutions,
   bound records/pages/storage, and sync file plus containing directory at atomic
   publication. The store lock alone does not allocate native principal sequences.
4. Preserve known remote outcomes if receipt persistence fails. Resume the exact
   original after cancellation, timeout or death. Tombstones and uncertain work
   cannot be collected under age/capacity pressure. Specify explicit retention
   decisions; expiry is an owner action and never a side effect of refresh.

**Gate:** actual-file failures and installed-process deaths before persistence,
after save/before send, after native acceptance, after business acceptance and
after terminal result. Concurrent aliases/processes cannot allocate different
work at one sequence. No duplicate resource, lost known outcome or state erasure
is permitted. Power-loss and network-filesystem guarantees require their own
evidence and are not inferred from SIGKILL tests.

### 10D — Execute the installed metadata lane

Replace the always-blocked
[live preflight](../../internal/testutil/liveplatform/qualification.go) with real
SDK calls only as 10B/10C land. Extend its configuration with exact placement,
native definition, journal identity and explicit fixture control inputs. Retain
artifact-hash checks and rejection of missing/skipped/duplicate required cases.

Reuse Platform's [installed-provider harness][p-harness] and local placement;
add provider-owned conformance assertions where needed. Run actual installed
Nix/mirror and extracted-release binaries, with no compiler, reattachment or
development overrides in the consumer. Requalify the complete runtime closure.
The harness provisions test identities and controls restart/fault barriers; the
customer provider never bootstraps its own endpoint or receives operator keys.

Execute CRUD/import/empty plans, capacity/retention, protection, drift, saved-plan
revision conflicts, both list kinds, current authorization at settlement and
disclosure, terminal failures, expiry/tombstones and process/server recovery.
Read state/plan/logs for secret canaries. Track all accepted fixture identities
and bounded cleanup, including partial apply and interrupted tests. Cleanup cannot
use broad forced deletion or discard unresolved original operations.

**Gate:** publish exact provider/platform/Transit/schema/artifact/CLI/system and
case results. This closes the metadata integration package and contributes to P4;
it does not close TF10 or authorize release publication.

### 10E — Qualify the complete initial service graph

Consume real P5 project/environment/service-account/grant APIs, P6 database/object
services, P10A registry/import, and P11 secret/binding/workload/endpoint controllers.
P10B hosted builds are not needed for the imported-image profile. P4 must supply
bounded service payloads and capability/collection contracts for these kinds;
the metadata command is not an arbitrary configuration transport.

P13 supplies the isolated graph with current credentials, budgets, complete
inventory and real management/controller/worker fault controls. P3 supplies the
selected hosted placement/network evidence where cloud support is claimed. An
explicitly unbilled technical fixture need not wait for commercial billing, and
technical qualification must not imply paid-service readiness.

For the exact installed candidate, execute project/environment → identity/grants
→ registry/database/bucket → service/references/bindings → endpoint. Verify import,
empty second plans, rename, drift, partial apply, refusal, stale revisions and
safe teardown. Independently push/pull an OCI digest, write/read database rows and
object bytes, exercise workload grants/secrets, and verify endpoint traffic/TLS.
Revoke authority, fail a rollout, kill providers and restart real controllers at
accepted-operation barriers. Retire endpoint/compute/bindings, then reread retained
data/artifacts independently. Only this full evidence closes TF10.

## Milestone 11: current-profile qualification and publication

Reuse [distribution tooling](../distribution.md), the exact-mirror ZIP builder,
checksums/provenance, protocol manifest and publication refusal tests. Remaining
work is qualification and actual distribution, not another release framework.

1. Qualify the new native runtime in Nix packages, mirrors and the independent
   consumer. Resolve redistributable native dependencies upstream and qualify
   the extracted ZIP on a clean host without Nix, compilers or the source tree.
   Shared-library discovery, required symbols and complete ABI identity must
   fail predictably on mismatches. Source publication alone is not portability.
2. Select a minimum Terraform CLI deliberately; 1.15.3 is the executed baseline.
   Pin minimum/current versions and execute protocol/schema loading, validation,
   saved plans, CRUD/import/empty plans and recovery for each claimed tuple.
   Start with Linux amd64. Add arm64/macOS only with native binaries and suitable
   process-death harnesses; shell evaluation and cross-builds are separate results.
3. Keep independent immutable experimental versions and hard-cutover rejection.
   Do not reintroduce historical state/migration promises before all TF1–17 close.
   One version/system must identify the same bytes in Registry ZIP and mirror.
4. Complete namespace/signing-key registration and publication access. Recheck
   current Registry requirements when implementing. Bind the full TF10 evidence
   to the exact tag/artifact/dependency tuple, and sign outside Nix. Preserve
   least-privilege workflow separation and immutable partial-publication retries.
5. Publish a qualified candidate and verify anonymous download/Registry discovery,
   signed checksums, readonly locks, install/reinstall and tamper/missing-version
   failures in a clean external consumer. Run its authenticated full live journey
   separately from anonymous installation. Record artifact URLs and CI evidence.

**Gate:** the advertised current CLI/API/system matrix, exact public bytes and
installed live graph all pass. Metadata-only evidence or a blocked report cannot
release an artifact. TF12–17 are not prerequisites for this initial publication;
each later advertised capability gets its own additive evidence gate.

## Shared delivery contract for milestones 12–17

For every family, deliver an independently reviewable contract/fixture slice,
then the adapter/discovery slice, then installed/live qualification. Use existing
[client](../../internal/platformclient/), [provider](../../internal/provider/) and
[fixture](../../internal/testutil/mockplatform/) owners. Extract common lifecycle
code only where actual new callers need it; service-specific invariants stay typed.
Keep fixture clock, demand, worker, fault and data controls outside the public API.

Each contract must define scope/authority, stable scoped import ID, supported and
immutable fields, expected revisions, canonical intent, acceptance/settlement,
receipt/tombstone behavior, complete bounded discovery, quotas, dependency
restrictions and protection. Exact and collection data sources are observational;
test denied/partial/repeated/stale pages and aggregate byte/item/time limits.

Each resource must pass validation/plan/apply/read/import/empty second plan,
rename or explicitly immutable-name refusal, drift, denied/absent distinction,
stale saved plans, protected/dependency-restricted deletion and interrupted CRUD.
Use separate states/principals, lost replies and installed-provider death; retain
original identities across real service/controller restart. Credentials, execution
inputs that contain secrets and secret outputs cannot enter Terraform state,
plans, logs or the Nix store. Bindings do not implicitly grant access.

For new releases, extend the required-case manifest by capability family, bind
it to the exact artifact and rerun the initial graph regression. An empty,
skipped, stale or missing family result cannot enable that capability. Mock and
installed-mock evidence permit development but never close a live criterion.

## Milestone 12: jobs and schedules

**Proposed surface:** `spoke_job`, `spoke_schedule`, exact/collection discovery and
bounded execution observations. Individual runs remain platform observations;
definition creation, plan and refresh never trigger work.

1. Agree with P20/P11 the image digest, non-secret arguments/references, runtime
   principal, required binding slots, concurrency/retry limits and job revision
   captured by an execution. Define cron/time-zone/DST interpretation, missed-run
   policy, overlap, enable/disable and schedule-to-definition update semantics.
2. Implement fake-clock fixture admission and explicit controller steps, then
   job/schedule adapters and canonical import. Separate definition settlement,
   scheduled admission, active execution and completion. Bound status history;
   never expose unrestricted output or use a computed status to drive a new run.
3. Define deletion with active work: disabling stops future admission; deleting
   definitions/schedules either restricts on dependencies or uses an explicit
   published retirement policy. Cancellation request, acknowledged cancellation
   and unresolved external effects remain distinct. Retain run evidence and data.
4. Qualify real scheduler clock boundaries/DST, missed and overlapping runs,
   update while running, revocation, disable/delete and scheduler/worker restart.
   Losing a Terraform reply must not submit an extra execution. An independent
   observer must witness the expected run and its attributed outcome.

**Gate:** shared lifecycle gates plus real execution/admission/recovery, with
named timing and concurrency limits. A stored schedule alone leaves TF12 open.

## Milestone 13: streams and queues

**Proposed surface:** `spoke_stream`, `spoke_queue`, typed exact/collection reads
and supported binding targets. Consumer groups are managed only if P20 publishes
their lifecycle; message production, claims and acknowledgements stay external.

1. Select the P20 profile over T6: partition/ordering identity, capacity, retention,
   producer/consumer grants, consumer ownership, queue visibility/retry and
   supported dead-letter policy. Reject unqualified atomic-DLQ or mixed-transaction
   guarantees. Expose immutable routing/schema/partition fields as the contract
   requires; do not assume all ALTER operations are available.
2. Model bounded deterministic payload/consumer/lease observations and fault
   controls in fixtures, then implement lifecycle/import/discovery and bindings.
   Test retention changes against retained data and pins; no configuration change
   implicitly authorizes purge. In-flight claims and dependent consumers restrict
   deletion according to the selected service contract.
3. Use independently authenticated data clients for append/read or enqueue/claim/
   settle and visibility expiry. Lose replies, restart service/workload, revoke
   access and test stale leases. Preserve exact original claim/settlement outcomes
   and retained data after consumer replacement. Fixture reads never advance time.

**Gate:** both declared service profiles pass management and real data-plane
gates, with actual ordering/delivery limits. A qualified stream can ship before
queues, but the combined milestone remains open until both criteria pass.

## Milestone 14: volumes and attachments

**Proposed surface:** `spoke_volume`, `spoke_volume_attachment` and exact/collection
discovery. P20 owns storage provisioning; P11 owns workload integration. A Transit
storage provider or bucket is not automatically a customer filesystem/block API.

1. Publish volume profile, capacity/growth, access mode, placement, protection,
   attachment identity and writer epoch. Separate desired attachment from observed
   mounted/ready/detached states. Reject unsupported shrink, move and access modes.
2. Implement explicit fixture attach/detach/fence transitions and adapters. Refuse
   competing incompatible writers. Treat node loss and detach timeout as uncertain
   ownership; they cannot authorize another writer or a successful deletion.
3. Qualify provider/controller death before/after attach and detach, stale epochs,
   lost replies and workload replacement against the actual storage backend.
   Write/reread bytes, verify the prior writer is excluded, and prove protected or
   attached storage cannot be deleted. Keep data after attachment/workload removal.

**Gate:** original-operation recovery, enforced writer fencing and retained bytes
pass for every advertised backend/access mode. Forced detach is unsupported until
the owning API supplies and qualifies an explicit safe policy.

## Milestone 15: custom domains and private connectivity

**Proposed surfaces:** `spoke_custom_domain` and `spoke_private_connection`, each
with typed discovery. Deliver these as two independent slices under P20/P3/P11.

1. Domain contract: normalized name, exact endpoint, owner/challenge generation,
   expiry/reverification, conflict handling and observable issuance/renewal/TLS
   state. Terraform may expose non-secret challenge records; DNS changes remain
   with the external DNS provider and keys remain with the certificate owner.
2. Private connection contract: local/peer scope and immutable target, both
   parties' authority, explicit acceptance, route/address constraints and teardown
   dependencies. If acceptance needs a separate Terraform resource, require that
   upstream lifecycle before adding it. Never infer peer consent from discovery.
3. Add fixture challenge/issuer/peer transitions and adapters with bounded waits
   and exact recovery. Test expired/replaced challenges, duplicate claims, denied
   targets, issuance failures, pending acceptance and conflicting addresses.
4. In the live harness, control isolated DNS/issuer/network fixtures externally.
   Verify actual certificate names, renewal and traffic withdrawal; prove private
   reachability, denied paths and revocation. Restart the owners after acceptance.
   Deleting either resource preserves external DNS and unrelated network/services;
   private failure never enables public exposure.

**Gate:** both slices have real lifecycle/traffic and recovery evidence. Domain
support does not qualify arbitrary private network providers or vice versa.

## Milestone 16: backup policies and database clones

**Proposed surface:** `spoke_backup_policy`, `spoke_database_clone`, policy/clone
discovery and exact/collection recovery-point observations. Backups are observed
executions unless Platform publishes a separately justified managed lifecycle.

1. P18 must specify policy schedule, retention, source identity, complete immutable
   recovery points, encryption/key references, integrity and recovery ownership.
   Separate desired policy from completed backup and measured recovery objectives.
   Require the selected [T24 backup/restore][t-operations] handoff for installation
   recovery claims; a read-only T19 Table branch is insufficient for this surface.
2. Bind clone create to an exact recovery point and destination scope/owner.
   Mutable selectors such as latest must resolve to a frozen identity before
   acceptance; retries never choose another point. Define independent writable
   destination, lineage, current source/destination authority and activation fences.
3. Add fixtures for incomplete/corrupt/missing points, key/access loss, interrupted
   copy/restore, retention dependencies and source changes. Implement adapters,
   import and observational discovery. No read starts a backup or retries a restore.
   Prevent deletion of dependencies needed by a retained backup/clone; policy
   retirement cannot silently delete its points, source or destination.
4. Qualify independently written known data, accepted unflushed writes within the
   advertised cut, integrity verification, interrupted restore, current authority
   and independent source/clone mutations. Old data must not revive revoked rights,
   retired workers or previously completed/uncertain external effects. Measure
   recovery time/window for the selected failure profile separately from settings.

**Gate:** exact-source writable clone and real policy/backup/restore evidence.
Do not mark TF16 complete from single-Table snapshots or configured retention.

## Milestone 17: scaling and placement policies

**Proposed surfaces:** `spoke_scaling_policy`, `spoke_placement_policy` and bounded
policy/status discovery. P17/P19 own reconciliation and physical resources.

1. Assign one writer per field: Terraform owns policy and constraints; controllers
   own desired/active instances and observed placement. Define how existing manual
   capacity relinquishes ownership when a policy attaches, and what explicit
   bounded mode remains on policy deletion. Unsupported ownership combinations
   fail validation instead of causing perpetual plans.
2. Specify min/max capacity, supported signal/source, freshness, cooldown/drain,
   quota and unschedulable states. Start with a qualified compute profile. Model
   stale/absent telemetry as uncertainty, never zero demand. Policy deletion must
   not implicitly delete a workload or persistent data.
3. Specify placement/residency capabilities and compatible volume/data profiles.
   For movement, retain logical identity and original operation across copy,
   verification, fencing, activation and routing. T22/T23/T24/T25 and P18/P19 gate
   distributed writer/failover claims; single-worker replacement is insufficient.
   Select stopped moves explicitly where that is the only qualified profile.
4. Add deterministic demand/capacity/placement fixtures and typed adapters, then
   run real scale-out/in, quota refusal, stale metrics, denied regions, drain and
   controller restart. Confirm empty plans despite changing observations. Where
   storage moves, verify data continuity and exclusion of displaced writers.

**Gate:** both policy families have real constraint/transition/recovery evidence
and measured limits for advertised profiles. Compute-only delivery can ship first
without claiming stateful movement or completing the whole milestone.

## Logical commit sequence and validation

Split changes at reviewable behavior boundaries; each implementation commit must
leave the current contract coherent. Prefix upstream changes with their owning
repository in tracking issues; they are not provider commits.

| Order | Suggested commit scope | Completion evidence |
| --- | --- | --- |
| 1 | `feat: cut native application contracts over to management v2` | 10A vectors, strict decoding, capability/field mapping and old-profile refusal. |
| 2 | Transit SDK + Platform pin handoff | 10B public APIs and matching native v2 integration; cannot be replaced by Go signing here. |
| 3 | `feat: integrate native SDK recovery and principal coordination` | 10C persistence barriers, exact recovery and cross-process sequencing. |
| 4 | `feat: enable native bucket lifecycle and discovery` | Current schema/config/import, authoritative reads, no unsupported service claims. |
| 5 | `test: qualify installed native metadata and crash recovery` | 10D real placement, artifact and all required fault cases. |
| 6 | Contract/adaptation commits per initial service family | P5/P6/P10A/P11 handoffs; whole-current-contract fixture and schema changes. |
| 7 | `test: qualify the installed live service graph` | 10E data planes, revocation, restart and retained data; close TF10 only here. |
| 8 | `build: package and qualify the native release matrix` | 11 runtime closure, clean non-Nix consumer and executed CLI/system tuples. |
| 9 | `release: publish and verify the qualified provider` | Exact public artifact/Registry and live consumer evidence; close TF11. |
| 10–15 | One series for each TF12–17 family: contract/fixture → adapter/discovery → installed/live evidence | Shared gates plus each family gate; split dual surfaces into smaller commits where useful. |
| 16 | `docs: publish the completed service capability matrix` | Every TF1–17 criterion has exact linked evidence; decide future compatibility policy explicitly. |

Keep five public Just tasks and the default task listing. Extend existing
`test-acc` targets and live configuration for family selection; no live test runs
from ambient credentials. Implementation uses `just fmt`, `just check`,
`just build`, installed artifact tests and explicit configured live tests as
applicable. Nix changes also require all-system evaluation, with native execution
recorded separately. Regenerate reference docs from schemas/examples through
`scripts/generate-docs.sh`; preserve this hand-maintained roadmap.

Each qualification record names committed source, exact dependency definitions,
artifact hashes, CLI/system, fixture kind, required cases, commands/CI runs,
failures/skips and limits. Preserve earlier failed/incomplete runs. Status changes
need actual milestone criteria, not merely merged adapters or mock passes.

For this planning-only change, validate relative links/fragments, the contiguous
1–17 index, phase membership, index/header status agreement, dependency order and
`git diff --check`. Runtime tests establish no new fact for a documentation edit.

[t-go]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/sdks/go/README.md
[t-abi]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/packages/zig/transit-client/abi-v1.json
[t-context]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/packages/zig/transit-data-client/docs/context.md
[t-recovery]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/packages/zig/transit-data-client/docs/recovery.md
[t-sdk-evidence]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/docs/history/qualification/sdk.md#local-sql-recovery-and-context-leases--2026-10-31
[t-messaging]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/docs/roadmap/06-streams-queues-and-durable-runtime.md
[t-branches]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/docs/roadmap/19-branches-merges-and-restore.md#local-table-branch-increment
[t-operations]: https://github.com/spoke-sh/transit/blob/a000b8d18449fdee7901955d0c94417ceeab32eb/docs/roadmap/24-operations-and-recovery.md
[p-lock]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/flake.lock
[p-contract]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/contracts/management-native-v2.md
[p-capabilities]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/contracts/management-native-v2-capabilities.json
[p-placement]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/packages/management/placement.md
[p-evidence]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/docs/qualification/native-v2.md
[p-harness]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/tests/native-provider-consumer.py
[p-alignment]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/docs/roadmap/provider-alignment.md
[p-later]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/docs/roadmap/20-additional-managed-services.md
[p-recovery]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/docs/roadmap/18-durability-recovery-and-upgrades.md
[p-scaling]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/docs/roadmap/17-elastic-capacity-and-fair-scheduling.md
[p-cells]: https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/docs/roadmap/19-cells-data-placement-and-dedicated-tenancy.md
