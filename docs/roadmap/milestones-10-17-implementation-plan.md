# Implementation plan: remaining milestones 10–17

[Roadmap](README.md) · **Status:** In Progress

Complete Platform HTTP integration, real-service qualification and independent
releases for the initial graph and six later service families. Milestones 10–17
remain **In Progress**. The [execution record](../../qualification/milestones-10-17.md)
identifies delivered mock-v5 resources, installed artifacts and historical native
codec/preflight work; it does not establish live support.

## Architecture decision and scope

Revised 2026-11-01 following the user's direction: all provider management
interaction uses Platform's public HTTP contract through a thin Go HTTPS client.

```text
Terraform → Go provider → Platform HTTPS management API → Transit / service backends
```

Platform owns Transit integration, native authorization/protocol translation,
internal sequencing and durable backend execution. The provider owns Terraform
schemas, HTTP authentication/request handling, bounded operation observation and
recovery through that public API. Do not add a Transit SDK, cgo/native library,
Serving signer, ABI pin, native principal coordinator or signed-envelope journal
to the provider. A shared native runtime is not a provider packaging prerequisite.

This decision supersedes the native integration requirements in the
[9–11 plan](milestones-9-11-implementation-plan.md) and the earlier version of this
plan. Preserve their executed evidence as history. Milestone 9 remains complete
for its recorded mock distribution scope; requalify installed artifacts when the
HTTP adapter or packaging changes. Hard cutovers continue until all 17 milestones
are complete; this revision introduces no compatibility layer.

The [HTTP boundary implementation](../../qualification/platform-http.md) now
removes native codecs/configuration, enforces all registered surfaces through the
shared HTTP client and replaces live/release identity checks. The HTTP contract
remains provisional; no real-service qualification, publication permission or
milestone completion follows from that implementation. The sequences below retain
the complete acceptance criteria; published contract adoption and live runs remain
upstream-dependent work.

## Source basis and changes since PR 8

The earlier review fetched the following upstream revisions. This revision uses
that recorded assessment and the merged provider source; it does not claim a fresh
upstream review or an already delivered public HTTP management API.

| Repository | Reviewed revision | Relevant state |
| --- | --- | --- |
| Provider | [`739619d`](https://github.com/spoke-sh/terraform-provider-spoke/commit/739619d085185f7e3bc72270d650cf3263f0db34), merged PR 9 | Go HTTP client, mock-v5, 23 resources/47 data sources, installed Nix consumers, unregistered native-v2 codecs and native-only blocked live/release gates. |
| Transit | [`a000b8d`](https://github.com/spoke-sh/transit/commit/a000b8d18449fdee7901955d0c94417ceeab32eb) | Runtime primitives available to Platform; no direct provider dependency is selected. |
| Platform | [`7af27c2`](https://github.com/spoke-sh/platform/commit/7af27c2bb8083ab64348592f82247da1ff703596) | Native management v2 bucket metadata, local operator/TLS placement and installed-provider harness. |
| Platform's Transit input | [`0e2fba4`](https://github.com/spoke-sh/transit/commit/0e2fba44727ca858bd1e8e4a775b62fbcf1a665d) | Backend build input in Platform's [lockfile][p-lock]; upgrades remain Platform's responsibility. |

### Delivered evidence and limits

- Platform's [v2 contract][p-contract] already uses HTTP, but its public request
  shape exposes Transit-signed sends and subject proofs. Its capacity/retention,
  settlement authorization, terminal failure, tombstones and bounded discovery
  are useful service semantics. They do not yet establish the Platform HTTP
  abstraction selected here. HTTP transport alone does not require a native SDK.
- [Platform qualification][p-evidence] and [placement tools][p-placement] supply
  real metadata restart, revocation, TLS/bootstrap and fault-control evidence.
  Reuse applicable harness controls after adapting them to the HTTP contract;
  results from another provider prototype do not qualify this artifact.
- The [capability inventory][p-capabilities] advertises only bucket metadata,
  with no object bytes or complete customer service graph. Other initial and
  later services remain backend prerequisites at these reviewed revisions.
- Transit [messaging][t-messaging] includes Queue claim waits; atomic dead-letter
  transfer, mixed transactions and integrated qualification remain open. Platform
  selects and qualifies the messaging profile behind its management API.
- A [single-Table read-only branch][t-branches] does not supply whole-database
  backups, writable clones or complete restore. Platform must provide and qualify
  those service guarantees separately.

### What remains blocking

Platform must publish and implement a versioned HTTP management contract usable
by an ordinary Go HTTP client, including machine authentication, scoped discovery,
resource operations, durable idempotency and original-operation lookup. Exact
routes, authentication scheme, version identifiers and wire fields remain owned
by Platform; do not promote the provisional mock contract to production by fiat.

The provider must reconcile that contract, adapt its HTTP client and fixtures,
replace obsolete native qualification assumptions, and execute installed live
acceptance. Real service backends and publication evidence remain required.
Transit SDK/ABI alignment and backend recovery internals belong to Platform's
implementation; they are not direct provider dependencies or release gates.

## Ownership, ordering and cutover decisions

`TF` means this provider roadmap; `P` means Platform and `T` means Transit.
Matching numbers in different repositories do not imply matching scope. Transit
references below explain Platform's backend dependencies, never provider calls.

| Work | Required upstream handoff | Provider delivery gate |
| --- | --- | --- |
| TF10 HTTP boundary | P4 public HTTP schema, machine auth, durable acceptance/recovery, discovery and isolated endpoint | Installed Go HTTP bucket lifecycle and original-operation recovery. |
| TF10 initial graph | P5 scope/IAM, P6 data, P10A OCI, P11 workloads/secrets/endpoints, P13 isolated graph; P3 for hosted claims | Independent OCI/rows/bytes/runtime/TLS and retained-data evidence. |
| TF11 publication | Qualified advertised HTTP service surface and publisher access | Exact public Go artifacts and anonymous locked installation on advertised systems. |
| TF12 jobs/schedules | P20 HTTP management and scheduler over P11 execution/P10A images | Real schedule/admission/retirement and interrupted-execution evidence. |
| TF13 streams/queues | P20 HTTP management over selected T6 messaging profile | Real produce/read or claim/settle, retention, revocation and recovery. |
| TF14 volumes/attachments | P20 HTTP storage/attachment management over P11; selected T22/P18 durability/fencing profile | Data across replacement and exclusion of stale writers. |
| TF15 domains/private connections | P20 HTTP management plus P3/P11 network/endpoint owners | Actual ownership, DNS/TLS/private traffic, revocation and retained dependencies. |
| TF16 policies/clones | P18 HTTP recovery APIs; selected T19/T24 backup capabilities | Exact independently restored data and source/destination isolation. |
| TF17 scaling/placement | P17/P19 HTTP policy APIs and controllers; T22–25 where distributed state is needed | Actual bounds/placement, one field owner, fencing and continuity. |

Upstream service ownership is recorded in [Platform's alignment map][p-alignment],
[additional services][p-later], [recovery][p-recovery], [capacity][p-scaling] and
[placement][p-cells]. Their recorded native-provider assumptions are superseded
for this repository. Cross-repository contract and implementation changes require
work in those owners; this documentation edit does not change their APIs.

Delivery order is HTTP contract → client/recovery adaptation → installed metadata
→ full initial graph → publication. Release preparation and later family
integration can proceed on their actual dependencies. TF13 does not depend on
TF12; TF15 does not depend on TF14. Storage-aware TF17 requires qualified storage
profiles; stateless compute policy need not wait for distributed storage.

An initial release may advertise a narrower qualified surface before TF12–17
complete. The current expanded artifact registers all six later families, so its
publication requires their real evidence too. A smaller release must explicitly
narrow its registered/advertised surface and required-case manifest together;
partial evidence cannot qualify the expanded artifact.

Replace superseded fixture shapes, client mappings, state schemas and gate
configuration together. Reject incompatible state/API versions before mutation.
An operator can inspect outstanding work with its original immutable artifact
and environment; a cutover never permits erasing uncertainty or retained data.
For each family, exact routes and wire fields come from the owning contract.
Keep one current provisional fixture version until that contract is adopted.

## Milestone 10: Platform HTTP integration and complete live graph

### 10A — Publish and reconcile the Platform HTTP contract

Platform P4 owns the public contract and independent server conformance. The
provider contributes its existing lifecycle requirements and a field-level mapping
from mock-v5. Agree the following before claiming a live adapter:

| Contract area | Required handoff |
| --- | --- |
| HTTP and authentication | Versioned schema, documented routes/methods/statuses, machine credential issuance/rotation/revocation, HTTPS trust and automation authority independent of browser cookies or operator keys. No Transit envelopes/proofs or native runtime in the consumer. |
| Scope and discovery | Organization-first bootstrap, stable scoped IDs, target/capability discovery, supported resource fields and operations, authoritative errors, complete bounded pagination and revision/snapshot rules. |
| Intent and concurrency | Stable management ownership/key before mutation, logical request identity and generation, canonical intent comparison, expected revision preconditions, and atomic server rejection of conflicting intent or independent owners. |
| Acceptance and recovery | Durable acceptance before effects, inspectable original operations after lost replies, distinct pending/succeeded/failed outcomes, receipt retention/expiry/tombstones and authorized pending lookup even when Terraform never saved a resource/operation ID. |
| Lifecycle and readiness | Current authority at admission/settlement/disclosure, safe import, dependency-restricted/protected deletion, separate declaration completion and controller readiness, retained data after retirement. |
| Limits and errors | Bounded request/response/pages/waits, documented throttling/retry semantics and explicit denied/absent/conflict/unsupported/incomplete/unavailable distinctions. |

Exact header names, URI paths, auth scheme and contract version are unresolved
until Platform publishes them. Pin the agreed schema and Platform revision; do
not require a provider Transit ABI pin. Existing bearer mock credentials are a
provisional choice, not evidence of Platform's chosen authentication scheme.

**Gate:** reviewed field mapping and independent raw-HTTP conformance vectors;
Platform implements the selected initial slice. Metadata capabilities are labeled
explicitly and cannot claim a usable object endpoint or service readiness.

### 10B — Adapt the Go HTTP client and cut over obsolete native scaffolding

Native removal and HTTP boundary enforcement are implemented. Reconciliation of
the provisional client to a published Platform schema remains pending.

1. Extend [platformclient](../../internal/platformclient/) using Go `net/http`
   and schema-derived types. Configure only the public endpoint, verified system
   or custom CA, external machine credential, scope assertions and bounded waits.
   Keep credentials out of URLs, state and logs; do not forward them on redirects.
2. Reconcile IDs/imports, required/immutable fields, protection, capability checks,
   errors and operation observations. Bound bytes, retries, pages and aggregate
   items; cancellation stops waiting without asserting remote rollback. Unsupported
   kinds/settings fail before dispatch. Denied or incomplete reads preserve state.
3. Hard-cut client, fixtures, Terraform schemas, examples and reference together
   where the published contract differs. Remove superseded mock wire/state paths;
   retain no native fallback or silently selected alternate protocol.
4. Retire [historical native codecs](https://github.com/spoke-sh/terraform-provider-spoke/tree/739619d085185f7e3bc72270d650cf3263f0db34/internal/nativecontract) and active
   schema/vector tests after their live/release consumers are replaced in 10D/11.
   Remove the remaining obsolete code/pins in the same change that detaches the
   final consumer; do not break existing fail-closed gates during an earlier
   client-only commit. Preserve past qualification reports with immutable source
   links when files are removed.

**Gate:** independent HTTP request/response tests and Terraform lifecycle/import/
discovery tests pass through the actual client. Verify wrong CA/hostname, revoked
credentials, scope/version mismatch, malformed responses, conflicts, bounded
throttling and cancellation. Provider build/install has no Transit native runtime.

### 10C — Prove original-operation recovery through the HTTP API

Platform persists accepted intent and outcomes and coordinates its own backend
sequences. The provider maintains stable logical ownership/request identity and
uses authorized HTTP lookup/retry semantics; no native principal lock or exported
signed-envelope journal is part of this architecture.

1. Establish a stable management key and operation identity before the first
   mutation. Specify how a fresh provider process discovers accepted work using
   configuration and the server's receipt/pending APIs if no Terraform state was
   saved. A resource ID or random request token received only after acceptance is
   insufficient. Terraform private state alone is not pre-dispatch persistence.
2. Define create/update/delete generations and revision preconditions so retries
   resume the original intent while a later A → B → A change is a distinct
   operation. Platform atomically binds key, principal, scope, generation and
   intent; conflicting payloads/owners cannot overwrite an existing operation.
3. After lost replies, deadlines or cancellation, inspect/resume the original
   operation. Preserve known outcomes and uncertain intent across provider/server
   restarts. Receipt expiry, denied lookup or unavailable history cannot trigger
   a fresh create or remove known state. Current authorization still applies.
4. Exercise aliases and concurrent Terraform processes against server-enforced
   ownership/revision conflicts. Keep the documented identical-principal/key
   cloned-state limitation; HTTP does not magically distinguish independent owners.
   Do not solve a server concurrency contract by adding client sequence allocation.

**Gate:** installed-provider deaths before dispatch, after acceptance and after
completion but before Terraform saves state, plus lost replies and real server/
controller restart across CRUD. Include delayed retries after later generations,
recreate after delete, retained receipts/tombstones and conflicting concurrent
intent. No duplicate logical resource or unsafe state removal is permitted. If
these cases cannot recover through the API, the Platform contract remains blocked;
do not weaken the gate or assume Terraform persisted something it did not.

### 10D — Execute the installed HTTP metadata lane

HTTP configuration/report/workflow cutover is implemented. Preflight refuses the
unpublished contract without credentials/network access; actual live cases remain
unimplemented until Platform's HTTP handoff is available.

Replace the always-blocked [live preflight](../../internal/testutil/liveplatform/qualification.go)
with actual HTTP cases as 10A–10C land. Select the explicit Platform API version,
schema/Platform revision, artifact identity, HTTPS endpoint/CA, scope and external
credential file. Remove mandatory Transit revision/definition, seeded native scope
and journal paths. Backend revisions may remain service provenance, not client
ABI compatibility requirements. Update `.github/workflows/live.yml` and report
schemas/names together. Retain rejection of missing/skipped/duplicate cases,
wrong artifacts, unsupported capabilities and mock evidence.

Reuse applicable [Platform harness][p-harness] controls after HTTP adaptation.
Execute installed Nix/mirror and extracted-release binaries without a compiler,
reattachment or development overrides. The harness owns isolated setup, identities,
restart/fault barriers, run budgets and bounded cleanup; customer Terraform never
bootstraps its own endpoint or receives operator keys. No ambient credential
selects live tests, and a live failure never falls back to a synthetic fixture.

Run CRUD/import/empty plans, capacity/retention, protection, drift, saved-plan
conflicts, complete object/pending discovery, authorization at settlement and
observation, terminal failures, expiry/tombstones and the 10C recovery matrix.
Inspect state/plans/logs for secret canaries. Track accepted identities and partial
apply leftovers; cleanup preserves unresolved work and independently owned data.

**Gate:** exact provider/Platform/HTTP-schema/artifact/CLI/system and case results.
This qualifies the metadata slice only; TF10 and publication still require the
complete initial service graph.

### 10E — Qualify the complete initial service graph

Consume real P5 project/environment/service-account/grant HTTP APIs, P6 database/
object services, P10A registry/import, and P11 secret/binding/workload/endpoint
controllers. Platform supplies typed service payloads, discovery and capabilities;
Terraform manages declarations over HTTP. Independent harness clients exercise
the actual data planes. P10B hosted builds are not needed for imported images.

P13 supplies the isolated graph, budgets, complete inventory and real management/
controller/worker fault controls. P3 supplies hosted placement/network evidence
where claimed. Technical fixture qualification does not imply billing readiness.

For the exact installed candidate, execute project/environment → identity/grants
→ registry/database/bucket → service/references/bindings → endpoint. Verify import,
empty plans, rename, drift, partial apply, refusal, stale revisions and safe teardown.
Independently push/pull OCI digests, write/read rows and object bytes, exercise
workload grants/secrets and verify TLS traffic. Revoke authority, fail rollouts,
kill providers and restart controllers at accepted-operation barriers. Retire
compute/bindings/endpoints, then reread retained data/artifacts. This full evidence
is required to close TF10.

## Milestone 11: current-profile qualification and publication

Reuse [distribution tooling](../distribution.md), exact-mirror ZIP construction,
checksums/provenance, the protocol manifest and publication refusal tests.

1. Qualify the Go HTTP provider in Nix packages, mirrors and independent consumers.
   Execute the extracted ZIP on a clean host without Nix, compilers, source or
   Transit libraries. Record actual executable dependencies and trust-store needs;
   no Nix store path may be an undeclared runtime dependency.
2. Select minimum/current Terraform versions deliberately; 1.15.3 is the executed
   baseline. Run protocol/schema loading, validation, saved plans, lifecycle/import
   and recovery for each claimed tuple. Start with Linux amd64; add arm64/macOS
   only with execution and suitable process-death harnesses. Evaluation and cross-
   builds remain separate results; Transit runtime support does not set this matrix.
3. Qualify the implemented HTTP checks in `scripts/release.py` and its tests with the
   published HTTP contract identity, Platform revision, exact artifact and complete
   advertised-family evidence. Update release workflow inputs/provenance as needed.
   Never accept mock results, blocked reports, missing cases or metadata-only
   evidence as the full live gate. Keep failure closed during the transition;
   remove obsolete native codecs/fixtures after the final consumer is detached.
4. Preserve independent immutable experimental versions and hard-cutover rejection.
   One version/system identifies the same bytes in ZIP and mirror; no historical
   mock/state compatibility promise is introduced before all TF1–17 complete.
5. Complete namespace/signing-key registration and access, checking Registry
   requirements at implementation time. Bind evidence to the exact tag/artifact,
   sign outside Nix, and retain least-privilege jobs and immutable publication retries.
6. Publish and verify anonymous Registry download, signed checksums, readonly locks,
   install/reinstall and tamper/missing-version failures in an external consumer.
   Execute its authenticated live journey separately and record URLs/CI evidence.

**Gate:** all advertised CLI/HTTP-API/system combinations, exact public bytes and
installed live services pass. For the currently expanded artifact, this includes
TF12–17 family evidence. An explicitly narrowed initial artifact can qualify the
initial graph first without completing those later milestones.

## Shared delivery contract for milestones 12–17

Mock-v5 contract/fixture and adapter/discovery slices are delivered for every
family. Remaining work is adoption of Platform-owned HTTP service contracts and
installed/live qualification; revise existing slices where the real schema differs.
The sequences below retain the full family design; reuse delivered mock/adapters
and add coverage only for contract differences or unresolved behavior. Use existing
[client](../../internal/platformclient/), [provider](../../internal/provider/) and
[fixture](../../internal/testutil/mockplatform/) owners. Extract common lifecycle
code only where actual new callers need it; service-specific invariants stay typed.
Keep fixture clock, demand, worker, fault and data controls outside the public API.

All Terraform management calls use the Platform HTTP API. Transit primitives and
service data-plane clients belong to Platform or the independent test harness;
no family adds a Transit SDK or service execution to the provider.

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

**Implemented provisional surface:** `spoke_job`, `spoke_schedule`, exact/collection discovery and
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

**Implemented provisional surface:** `spoke_stream`, `spoke_queue`, typed exact/collection reads
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

**Implemented provisional surface:** `spoke_volume`, `spoke_volume_attachment` and exact/collection
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

**Implemented provisional surfaces:** `spoke_custom_domain` and `spoke_private_connection`, each
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

**Implemented provisional surface:** `spoke_backup_policy`, `spoke_database_clone`, policy/clone
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

**Implemented provisional surfaces:** `spoke_scaling_policy`, `spoke_placement_policy` and bounded
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
| 1 | Platform: publish/implement the HTTP management contract | 10A schema, auth, recovery/discovery and independent server conformance; upstream work. |
| 2 | `feat: adopt the Platform HTTP management contract` | 10B client/fixture/schema cutover; keep remaining gates functional until their consumers change. |
| 3 | `test: qualify HTTP operation identity and recovery` | 10C lost-state/reply, generations, concurrent intent and installed-provider crash cases. |
| 4 | `test: replace native preflight with installed HTTP qualification` | 10D live config/report/workflow cutover and real metadata cases; missing prerequisites still fail. |
| 5 | Contract/adaptation commits per initial service family | P5/P6/P10A/P11 HTTP handoffs with coherent current-contract fixtures. |
| 6 | `test: qualify the installed live service graph` | 10E data planes, revocation, restart and retained data; close TF10 only here. |
| 7 | `build: qualify HTTP release gates and the provider matrix` | 11 HTTP evidence validation and mock refusal; remove final native scaffolding, then qualify clean-host artifacts and CLI/system tuples. |
| 8–13 | Per TF12–17 family: adopt HTTP contract → adapt fixtures/client → installed/live evidence | Existing provisional surfaces are retained or hard-cut to the agreed schema; family gates remain required. |
| 14 | `release: publish and verify the qualified provider` | Exact public artifacts/Registry and full advertised-surface evidence; a deliberately narrower initial release may precede 8–13. |
| 15 | `docs: publish the completed service capability matrix` | All TF1–17 criteria have linked evidence; decide future compatibility policy explicitly. |

Keep five public Just tasks and the default task listing. Extend existing
`test-acc` targets and live configuration for family selection; no live test runs
from ambient credentials. Implementation uses `just fmt`, `just check`,
`just build`, installed artifact tests and explicit configured live tests as
applicable. Nix changes also require all-system evaluation, with actual target execution
recorded separately. Regenerate reference docs from schemas/examples through
`scripts/generate-docs.sh`; preserve this hand-maintained roadmap.

Each qualification record names committed source, HTTP schema and dependency revisions,
artifact hashes, CLI/system, fixture kind, required cases, commands/CI runs,
failures/skips and limits. Preserve earlier failed/incomplete runs. Status changes
need actual milestone criteria, not merely merged adapters or mock passes.

For this planning-only change, validate relative links/fragments, the contiguous
1–17 index, phase membership, index/header status agreement, dependency order and
`git diff --check`. Runtime tests establish no new fact for a documentation edit.

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
