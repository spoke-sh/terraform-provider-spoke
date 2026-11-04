# Implementation plan: CI supply governor

**Status: Contract and unregistered client implemented; service delivery planned.**
The separately pinned provisional profile and persisted synthetic HTTPS peer pass
the [client conformance matrix](../../qualification/governor-client.md).
No Terraform schema, native governor service, live runner or billable service is added. This extends
[scaling and placement](17-scaling-and-placement.md), using existing
[scope/access](04-scope-and-access.md) and [artifact identity](05-artifact-registries.md).
[Jobs and schedules](12-jobs-and-schedules.md) retain their existing execution
semantics and evidence; no new customer CI resource is required there.

## Customer demand and operator supply

Customers use the workflow integration in
[`spoke-github-action`](https://github.com/spoke-sh/spoke-github-action). Their
repositories declare commands and optional runtime compatibility, with no capacity
Terraform, pool/cache provisioning, fleet count, machine size or governor ID.
GitHub produces queued-job demand and owns scheduling/attempts. Platform resolves
the authorized account and governs eligible supply, managed cache access and use.

A separately authorized operator uses this provider in an infrastructure root to
manage `spoke_ci_governor`. Terraform declares limits and policy, not demand or
actual worker counts. Only Platform translates that policy into physical supply.
The provider remains a Go client of the public Platform HTTPS API.

The [Platform demand/governor contract](https://github.com/spoke-sh/platform/blob/dcb40932b1e927b0525a3d6fa392142ed0b5044a/contracts/ci-demand-and-governor-v1.md)
and [delivery plan](https://github.com/spoke-sh/platform/blob/dcb40932b1e927b0525a3d6fa392142ed0b5044a/docs/roadmap/ci-runner-implementation.md)
own the service behavior. The [historical timing baseline](https://github.com/spoke-sh/platform/blob/563e0eeab3477c329a457227db436d5eca70d251/docs/qualification/ci-runner-baseline.md)
retains its original source attribution. These links preserve the refined proposal at `dcb4093`,
merged through Platform #59. The [executable HTTP profile](https://github.com/spoke-sh/platform/blob/af60b8a13d7b9f953010552096e33174c22d3981/contracts/http/ci-governor-profile.md)
now owns the separately versioned provisional management boundary. Track service
work in [Platform #56](https://github.com/spoke-sh/platform/issues/56) and provider
work in [implementation tracker #1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1).

## Proposed Terraform surface

| Surface | Policy or observation |
| --- | --- |
| `spoke_ci_governor` | Operator environment, admission/budget/managed-cache policy references, qualified execution profiles, active-job/vCPU/warm-capacity ceilings, deadlines and drain policy |
| Exact/collection governor data sources | Scoped stable identity, desired/active policy revisions, supported profiles, readiness/blockers and bounded observations |
| Demand, workers and usage | Platform/GitHub observations; never Terraform resources or desired-count fields |

The earlier proposed `spoke_github_runner_pool` and required `spoke_build_cache`
customer resources are removed from this slice. Managed cache selection is a
server policy; cache lifetime remains independent of governor/worker retirement.
No customer Terraform is needed to use the workflow.

The governor is the sole owner of its supply policy. Reject a generic scaling
resource attempting to own the same fields. Enforce current operator authority,
opaque scoped IDs, expected revisions and advertised profile combinations. Import
cannot transfer authority. Plan, refresh and discovery cannot generate demand,
register a runner, start a job, write cache data or repair cloud resources.

GitHub installation consent is separate onboarding. Never accept App keys,
installation/JIT tokens, AWS credentials or worker secrets as resource fields;
sensitive Terraform attributes still store values. Customer job credentials must
not mutate a governor or select another account's supply authority.

Lowering ceilings changes new admission and drains surplus; it does not silently
kill accepted work. Raising limits with zero demand launches nothing. Deletion
stops admission and drains; unresolved leases/usage can block completion. Preserve
receipts, tombstones and caches. Successor-governor handoff must fence the old
owner without duplicating reservations. Actual worker counts remain observations.

## Reviewable delivery changes

### Next slice: Terraform governor adapter and installed recovery

**Proposed for review — 2026-11-04; implementation has not started.** The
[05:05 UTC planning review](platform-adoption-implementation-plan.md#adapter-planning-review-2026-11-04-0505-utc)
confirms the committed schema and client prerequisites are delivered. Implement
the provider portion of change E against that provisional boundary, with
installed synthetic evidence. Platform change C remains the critical path to
native management and service qualification; this proposal does not reorder or
complete those service gates.

The outcome is one experimental `spoke_ci_governor` resource and exact/collection
data sources in an operator infrastructure root. Customers continue to use the
workflow contract without Terraform. Stop at management policy, discovery and
recovery; there is still no Platform governor endpoint or qualified execution
service to adopt. Keep execution, warming, handoff and billing unsupported.

Implementation starts from provider `1ec09faa16332ed9371e6c16c8fe57fb7ae5f221`
and retains Platform contract `af60b8a13d7b9f953010552096e33174c22d3981`.
The existing client, independent peer and immutable mirror harness are reused.
The final registration count would be 27 resources and 55 data sources; the
current 26/53 count remains unchanged until the code lands. No new numbered
milestone, live qualification mode or customer workflow is part of this slice.

#### Terraform interface decisions

Use the existing provider's endpoint, external credentials, CA, timeout and
organization/project/environment assertions. Governor configuration requires
all three existing operator scopes. Create the governor client only for governor
surfaces, so organization-only bootstrap and publication configurations retain
their own requirements. Rename the internal `publicationConfig` holder to a
neutral HTTP configuration name if shared; do not add another credential path.

| Surface / attributes | Proposed caller behavior |
| --- | --- |
| Resource `spoke_ci_governor` | Flat policy attributes matching the committed `Policy` schema; all policy inputs explicitly required, with no hidden defaults. |
| `management_key` | Required, non-secret, 1–64 characters using the pinned key pattern. Available before mutation; changing it creates a new logical owner. |
| `name`, `admission_policy_id`, `budget_policy_id`, `cache_policy_id` | Required IDs using the governor's narrower lowercase 1–63 character pattern. Admission entitlement is immutable while owned; changing it requires explicit retirement and a new management key. Budget/cache references remain service-authorized policy updates. |
| `execution_profile_ids`, `max_warm_runners`, `retirement_policy` | Exactly `["linux-x86_64-v1"]`, `0` and `"drain"` in this profile. Reject unsupported values; never silently normalize them. |
| `max_active_jobs`, `max_allocated_vcpus` | Required integers, respectively 1–256 and 1–4096, as defined by the schema. They are policy ceilings, not desired worker counts. |
| `max_job_seconds`, `max_queue_seconds`, `deletion_protection` | Required integers 1–86400 and 1–3600, and an explicit boolean. Protection must be disabled through a completed policy update before retirement. |
| Resource observations | Computed canonical `id`, opaque `resource_id`, `owner_id`, desired `revision`, `active_revision`, `readiness` from wire `state`, bounded `blockers`, `active_jobs`, `allocated_vcpus`, and `unsettled_usage`. |
| Recovery observations | Computed `operation_id`, `operation_sequence` and `operation_status`; protocol-versioned plan metadata retains the exact mutation fence. Pending progress must not be mistaken for a new mutation baseline. |
| Data source `spoke_ci_governor` | Takes the canonical scoped `id`; returns the authoritative declaration and observations. An absent requested object is a diagnostic, not a fabricated empty object. |
| Data source `spoke_ci_governors` | Uses the configured operator scope; returns a complete, stable-ID-ordered `governors` collection. No caller-controlled pagination or incomplete result is exposed. |

Canonical Terraform/import ID: `organization/project/environment/governor_id`.
Validate every component against the profile and configured scope. Import reads
the governor's management key and verifies its retained owner/request; it does
not create a key, transfer ownership or commit prepared work. An unresolved
mutation that cannot be represented safely must produce a recovery diagnostic.
Scope changes on an existing state fail validation. Retired keys are never
recycled, including replacement through `-replace`; recreation requires
the caller to supply a new configured key. Overlapping owners are refused even
when Terraform requests create-before-destroy.

Null required inputs fail validation. Unknown inputs may remain unknown during
planning, but all mutation inputs must be resolved before dispatch; resolution
cannot rebase a captured sequence or expected revision. Use signed Terraform
integers only after enforcing the unsigned-32-bit wire bounds. Credentials and
credential contents never become resource attributes or plan metadata.

#### 1. Prove Terraform mutation identity before registering the surface

Require a stable, non-secret management key in configuration, available before
the first provider process can die. Define and test the decision table using
configuration, prior Terraform state, the server's latest retained request and
the authoritative governor revision. Recover the exact prepared/accepted request
before selecting any successor. A sequence consumed by refusal or abort is not
reused; a retired key requires an explicit new key for recreation.

Cover create, update and retirement with no saved operation ID; a completed
request whose response/state write was lost; changed configuration while work
is pending; A → B → A; concurrent writers; stale saved plans; import; and cloned
state. Distinguish recovery of an already applied mutation from a genuinely new
logical change. Do not auto-increment on timeout, derive identity solely from
policy bytes, generate keys only in unsaved state, auto-abort pending work or
silently advance an expected revision. Identical credentials and copied keys
still cannot distinguish a clone from legitimate recovery.

**Gate:** every branch either recovers the original identity, proves a permitted
new mutation or returns an actionable conflict without mutation. If that cannot
be established through the pinned API, deliver the failing case and a Platform
contract amendment first; do not register an adapter around an unresolved case.

Implement a governor-specific selector in `internal/provider/governor_identity.go`
with focused decision tests. It may reuse transport and Terraform infrastructure,
but not the publication slot encoding/64-slot scan or bucket generation algorithm.
Use the governor's bounded `Management`, `Request` and exact `Governor` reads.

The plan captures `(profile version, scope, principal, key, sequence, kind,
target, expected revision, full policy)` in protocol-versioned private plan
metadata and exposes the selected sequence as computed progress where a resource
plan exists. Destruction also captures its fence even though its planned resource
value is null. This metadata is a checked cache of intent, not the durability
authority: configuration supplies the stable key and the server retains prepared
intent before commit. A fresh process/plan must reconstruct recovery from those
sources without relying on a previous private-state write.

| Observed condition | Plan/apply decision |
| --- | --- |
| Key authoritatively absent; create requested | Select sequence zero, null target and expected revision zero. Freeze full intent before PUT. A concurrent winner at this key must be an exact replay or a conflict. |
| Latest request prepared/accepted and matches the requested action, target and policy | Select that request's original sequence, revision and complete intent. Planning only observes; apply may resume it. Never commit a different retained intent via an unconditional `Resume`. |
| Latest request prepared/accepted but desired configuration/action differs | Return a conflict identifying the original operation. Restore its intent to recover it, or resolve it through an explicit owner-authorized operation; never auto-abort, retire over a pending update or allocate a successor. |
| Exact planned request already applied; response/state save was lost | Recover its result and authoritative resource, or exact retirement absence. Preserve its original sequence. A read must not turn replay into another update. |
| Terraform has no resource state, but the key already owns a live governor | Recover the same owned incarnation only when its retained/current policy matches the create configuration. Otherwise require import or restoration of the original intent before a fresh update plan; Create must not silently become Update. |
| Latest terminal request and current resource agree with desired policy | Refresh/import current observations and produce an empty plan. No new management request. |
| Latest terminal request; an owned resource's fresh plan requests a different policy or retirement | Select exactly the next sequence, with the resource revision observed by that plan. A → B → A therefore consumes three distinct logical mutation sequences. |
| Latest request refused/aborted | A saved plan for that request returns its retained outcome. Only a fresh plan may explicitly select its successor; retries inside an apply cannot do so. |
| Server sequence exceeds the selected request, ownership/target differs, or a new request's expected revision is stale | Refuse the stale plan before new intent is submitted. Do not rebase onto the newly observed revision, even if policy bytes happen to match. |
| Retired key, expired receipt, exhausted counters/capacity, denial or uncertain read | Preserve retained identity and surface the typed refusal/error. None is permission to reuse a key, skip a sequence or claim absence. |

Verify ownership and the saved-plan fence before selecting a branch. Exact
completed-request recovery uses its original intent; the revision increment
caused by that same request is not a stale-plan error. If independent drift has
since changed the result, retain authoritative observations and require a fresh
plan rather than reconstructing the old desired policy as current state.

Test these decisions through the real Framework protocol as well as the selector.
In particular, verify that null destroy plans, refresh after a crash and an
unknown-to-known input transition preserve their fences. Classify original
operation recovery before updating progress fields during refresh; otherwise an
accepted request could accidentally become the baseline for a successor.

#### 2. Add policy management and observational discovery

Use the existing governor client and exact profile/capability checks. Map policy
fields, authorized references and protection separately from computed active
revision, counts, readiness and blockers. Unknown/null input, unsupported profiles
and immutable ownership changes must have explicit Terraform behavior. Import
reads the existing scoped owner and key; it never transfers authority.

Plan, refresh and exact/collection reads remain observational. Incomplete
discovery, denied reads and outages are errors, not evidence of absence. Protect
active work during reductions and draining retirement; retain state until the
server proves authoritative absence. Runtime observation changes must not make
Terraform restore a prior worker count. Add the new surfaces to the HTTPS
registry audit without introducing Transit, GitHub or cloud clients.

Keep implementation in `internal/provider/ci_governor_resource.go` and
`ci_governor_data_source.go`, with shared typed mapping and configuration helpers
only where they remove duplicated validation. Register the three surfaces in
`provider.go`. Extend `TestEveryRegisteredSurfaceUsesPlatformHTTP` with explicit
governor scope/version/path expectations through an independent raw HTTPS peer;
do not exempt the new profile from the registry audit.

`Read` never calls `Apply`, `Resume` or `Abort`. If a request is still pending,
preserve the known resource and expose bounded pending information; do not clear
state while accepted work can still affect the target. A denied read, missing
parent or incomplete listing is never a removal signal. Drain timeout/cancellation
returns a diagnostic with the retained identity and keeps Terraform state.
`applied` management may legitimately yield `disabled`/`execution_unqualified`;
waiting for worker readiness would incorrectly turn this declaration into a
service qualification claim.

#### 3. Execute the installed Terraform recovery matrix

Extend the existing installed-binary harness with an isolated governor root and
the persisted HTTPS peer. Use exact Nix mirror bytes, readonly checksums and
Terraform subprocesses with no compiler in PATH. Keep the existing independent
bucket/publication consumers and their locks unchanged.

Kill the actual provider process during create/update/retire before dispatch,
after prepare, after acceptance and after completion before state save. Restart
Terraform independently from reloading the peer's persisted ledger. Assert one
original key/sequence/operation, no duplicate logical policy effect and preserved
receipts. The existing twelve client deaths do not satisfy this adapter matrix.

Use twelve required provider-death schedules: three mutation kinds × these four
barriers. A test proxy/peer blocks prepare before forwarding for the first case,
withholds the durably prepared response for the second, holds settlement after
durable commit for the third, and withholds the terminal response for the fourth.
Record the killed provider PID/executable, ledger state and request tuple. Reload
the peer from disk as a separate step and preserve the original TLS origin; do
not count killing Terraform itself or restarting an in-process provider as the
required death. Test saved-plan replay and fresh-plan recovery explicitly, with
fresh planning after changed intent requiring the decision table above.

Add test-only synchronization around committed peer transitions and extend the
existing Linux consumer harness to accept a governor barrier without translating
governor requests into mock-v5 operations. Assertions compare independently read
ledger/request identities and policy revisions. Fixture settlement is explicit;
GET cannot drive it. No fault-control route or hook enters the provider binary.

Also execute import/empty-plan/drift, unknown/null validation, cross-scope and
revoked authority, stale revisions, concurrent writers, A → B → A, retired-key
recreation, protected deletion, blocked drain and preserved cache/usage
references. Zero-demand apply and raised limits produce no fixture launch;
observational reads do not settle requests. These are synthetic assertions,
not evidence of physical workers, native cleanup or aggregate service quotas.

| Required group | Acceptance proof |
| --- | --- |
| Terraform lifecycle and schema | Create, update, import, refresh, empty plan and retirement; null/unknown/bounds, scoped ID parsing, incompatible state version and unsupported profiles fail before mutation. |
| Identity and concurrency | The complete decision table, stale saved plan, conflicting and identical concurrent requests, A → B → A, refusal/abort consumption, receipt expiry and new-key recreation. |
| Process and peer recovery | All twelve crash schedules execute with separately reloaded persistent peers; original identity survives and each logical policy revision changes at most once. |
| Authority and aliases | Current operator/reader/job/foreign-scope permissions, revocation before prepare/commit/settlement and import ownership; independent provider aliases cannot share configuration. |
| Policy and retirement | Zero-demand creates/raises, lowered ceilings with outstanding usage, blocked drain, protection, timeout and preserved independent cache/artifact/usage references. |
| Observational reads | Raw request audit sees GET only during plan/refresh/discovery; state removal requires authoritative target absence with no unresolved operation. Collection snapshot changes, cursor loops and missing pages fail. |
| Bounds and confidentiality | Deadline/cancellation, malformed/oversized replies, terminal capacity refusal; secret canaries absent from state, saved plans, logs and diagnostics. |
| Packaging and inventory | Exact mirror/version/checksums, compiler absent from Terraform PATH, required test manifest complete, all 27/55 registered surfaces covered by HTTPS audit after implementation. |

Create a separate `testdata/contracts/governor-terraform-cases.json` required-case
inventory rather than extending the historical client result into a Terraform
claim. Add governor lifecycle and process-recovery presence checks to
`scripts/test-installed.sh` and the existing mock task. Missing cases, artifacts
or unavailable required barriers fail; they must not silently skip. Preserve all
existing suites and the five-public-task limit. Investigate measured timeout or
resource failures before increasing the ordinary lane's limits.

#### 4. Package and record the bounded result

Generate reference docs and operator examples through `scripts/generate-docs.sh`.
Run locked-shell `just fmt`, `just check`, `just build` and the required installed
matrix; evaluate all systems if Nix changes. Record the exact provider artifact,
contract digests, required cases and failed/skipped/unqualified behavior. Keep
TF17 In Progress, live/release gates closed and customer workflow templates
disabled until their separate service evidence exists.

Use one provider PR with these logical commits; each leaves the existing registry
and checks coherent:

| Order / commit | Main files and review gate |
| --- | --- |
| 1. `feat(provider): define recoverable governor mutation selection` | Governor identity selector, protocol design and focused decision/Framework tests. Stop for a Platform contract amendment if any identity case remains ambiguous. |
| 2. `feat(provider): add experimental CI governor management` | Resource schema/mapping/CRUD/import, scoped configuration and resource HTTP-boundary coverage. Verify all policy and observation ownership rules. |
| 3. `feat(provider): add bounded governor discovery` | Exact/collection data sources, explicit registry expectations and read/error tests. Final inventory becomes 27/55; no request/worker/usage Terraform resources. |
| 4. `test(provider): qualify installed governor recovery against synthetic peers` | Persisted-peer barriers, process harness, case manifest and existing mock/installed task integration. Execute lifecycle, authority/drain and all twelve provider deaths. |
| 5. `docs(provider): record experimental governor management evidence` | Generated schema docs, operator examples, `docs/governor-identity.md`, and separate `qualification/governor-terraform.md`/JSON report. Update README, AGENTS, TF17 and tracking with executed evidence and remaining gates. |

After schema/example changes, run `nix develop --command ./scripts/generate-docs.sh`.
Final validation is `nix develop --command just fmt`, `just check`, `just build`
and `just test-acc installed`, each in the locked shell. Focused tests provide
earlier feedback but do not replace these required checks. Evaluate all systems
without builds if Nix changes. A clean committed implementation supplies the
immutable artifact; record artifact and harness revisions separately if the
final evidence commit follows the run. Do not publish a dirty snapshot.

Add the generated operator example under `examples/resources/spoke_ci_governor/`
and matching data-source paths, plus an isolated governor scenario in the process
harness. Existing committed independent consumers and Saga constraints/locks
stay pinned; this slice does not claim independent hosted adoption. No new Nix
dependency is expected: the governor's embedded contract files are already in
`nix/provider.nix`'s allowlist. Inspect that filter if implementation adds another
embedded file.

Track the implementation in [provider #1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1).
Before submitting the implementation PR, refresh the upstream review and re-read
the tracker, then attach exact required-case evidence. Keep TF17 **In Progress**.
The PR is ready when all provider gates above pass or when a concrete contract
gap has been isolated into a separately reviewable prerequisite; an incomplete
adapter must not be reported as the completed slice.

#### Upstream work and stop conditions

- **Platform #56 / #20:** choose and prove a retained native management owner,
  autonomous continuation and independently authorized refusal/cleanup after
  caller revocation. Then prove atomic operator/account admission across two
  repositories with deterministic external peers, original effect identities
  and retained uncertainty. No provider fixture can satisfy this gate.
- **Platform publication regression:** reproduce the recorded native
  `HttpCapacity` / `AuthorityOutcomeUnknown` acceptance failure on the existing
  pin before changing retry policy or accepting a fixture cutover. Transit #362
  changes co-hosted SQL lane waiting and diagnostics; it is not an established
  fix for that failure and supplies no governor owner.
- **Transit publication:** #364 repairs definition-bound fixtures and release
  shell setup, but the latest nightly is still the failed `2f8be4b` run with
  skipped image/installed packaging. Require new qualified artifact evidence and an
  explicit retained-state cutover before changing production or candidate pins.
- **Real workers and paid use:** explicit AWS/GitHub setup, isolated execution,
  cleanup, assignment, cache isolation and real usage remain later Platform
  gates. AWS login helpers and prepaid-credit shadow tests do not qualify them.
  Platform #64's merged sandbox funding plan and recorded sandbox credential
  handoff supersede the old missing-account prerequisite; actual payment,
  runtime-secret consumption and CI usage remain unqualified.

<a id="next-slice-for-review-governor-contract-and-http-client"></a>

### Delivered slice: governor contract and HTTP client

**Implemented scope:** turn the operator-governor proposal into a versioned,
executable contract and a tested Go client. Stop before registering Terraform
resources or implementing physical supply. This is the contract portion of
Platform #56's change B and the provider foundation for its later change E.
It does not attempt all six delivery changes below in one implementation.
The [execution record](../../qualification/governor-client.md) identifies exact
commits, required cases and remaining gates. The original plan follows for scope
and acceptance traceability.

The [2026-11-03 23:43 UTC review](platform-adoption-implementation-plan.md#next-slice-review-2026-11-03-2343-utc)
found provider #19 and workflow-repository #1 merged. At that review, Platform #59 remained open
at `dcb40932b1e927b0525a3d6fa392142ed0b5044a`; main `8987581` still contained the
earlier pool proposal. Neither revision had a governor HTTP schema, canonical
vectors or API implementation. Implementation reconciled merged #59 and added
the committed provisional contract; it did
not derive a supported endpoint from its illustrative Terraform example.

The useful outcome is an independently exercised management boundary for the
later `spoke_ci_governor` adapter. Customer workflows continue to describe work;
operator Terraform will own policy. Workers, admission accounting and managed
caches remain Platform responsibilities. This slice requires no cloud account,
GitHub installation token, hosted service or runtime-pin change.

#### 1. Freeze the Platform-owned management profile

Deliver a separately versioned, explicitly provisional `ci-governor-v1` profile
under Platform's `contracts/http/`, with schema, canonical request/response
vectors, capability matrix and executable validation. Platform owns the final
paths and wire types. Review the workflow contract at merged `a65062f` alongside
it; clarify caller semantics there if needed, without enabling its template.

The contract must resolve these decisions before client implementation:

- Exact operator organization/environment scope, current authorization and
  non-transferable ownership. Job/customer credentials cannot mutate governors;
  labels and submitted account IDs cannot confer authority. Define one policy
  owner per entitlement, including conflicts with generic scaling policy.
- Policy fields versus observations: supported profile and admission/cache/budget
  references, ceilings, deadlines and drain policy; desired/active revisions,
  readiness and bounded blockers remain separate. Define numeric/input limits,
  defaults, null/omitted fields and unsupported combinations, including the
  initial zero-warm profile. Policy acceptance is not worker readiness.
- Create/update/retire, exact and collection reads, original-operation lookup
  and bounded pending discovery. Specify headers, status/error mappings,
  authoritative absence, snapshot/cursor rules and bounded response/work limits.
- One recoverable identity per logical mutation, established before dispatch
  and discoverable through server-backed ownership/receipts after client death.
  Specify allocation, replay, generation/revision fences, expiry and tombstones
  for create, update and delete. Cover A → B → A, concurrent writers and recreation;
  neither an intent hash nor a fresh key in unsaved Terraform state is sufficient.
  Record that identical credentials and copied keys cannot distinguish cloned
  client state from legitimate recovery; import cannot transfer ownership.
  Reusing bucket or publication mechanics requires explicit contract evidence.
- Retirement stops admission and drains; uncertain leases or unsettled usage
  prevent destructive completion. Lowered ceilings affect new admissions while
  accepted work retains its policy. Preserve caches, artifacts and reservations;
  define fenced handoff or explicitly refuse it until qualified. Reads never
  advance these transitions.

Gate: the owner commits the schema/vectors and independent positive/negative
contract tests. All recovery decisions above must be specified, rather than
left as implementation TODOs. These files describe a provisional boundary;
they do not advertise an implemented Platform API or prove native persistence.

#### 2. Pin the profile and build deterministic HTTPS conformance fixtures

Copy the committed contract bytes into a dedicated provider contract directory,
with source revision, profile/version, schema and vector digests, status and
license attribution. Keep `mock-v5`, bucket and publication pins independent.
Use a dedicated test peer under `internal/testutil/mockplatform/`; fixture
controls remain outside the proposed public API.

Model only management semantics: operator ownership, desired/active revisions,
retained requests/receipts, drain blockers and bounded discovery. Seed synthetic
outstanding leases/usage to test policy changes without implementing a runner
scheduler in the provider. Persist accepted fixture state before exposing crash
barriers. Raw HTTPS assertions must independently check the peer and vectors,
so client and mock cannot validate only each other's interpretation.

#### 3. Implement the unregistered Go HTTP client

Add the governor client, schema-derived boundary types and strict validation
under `internal/platformclient/`. Reuse TLS, external credentials, deadlines and
bounded HTTP transport where appropriate; keep governor recovery/profile logic
separate. Reject unknown profiles and unsupported capabilities before mutation.
Retry uncertain work only through the agreed original identity; cancellation
stops waiting and does not manufacture rollback or terminal success.

No Terraform resource or data source is registered in this slice. Consequently
the existing 26-resource/53-data-source surface, installed consumer constraints
and locks stay unchanged. Framework null/unknown planning, import and actual
provider-process death tests belong to the subsequent adapter slice; a client
process restart is not evidence for those cases.

#### 4. Record conformance and hand off to native management proof

| Required proof in this slice | Observable result |
| --- | --- |
| Version, schema and authority | Exact pinned vectors pass; malformed fields, disallowed nulls, unknown fields, unsupported profiles, customer writes and foreign-scope receipts fail without mutation. |
| Recovery of create/update/retire | Before dispatch, lost acceptance, accepted-but-unsettled and lost completion schedules retain the same logical request/operation across client and fixture restarts. |
| Concurrent/repeated intent | Conflicting owners and stale revisions fail; identical retries converge; A → B → A and recreation do not replay an earlier logical change. |
| Observational reads | Repeated exact/collection/pending/operation reads leave accepted state unchanged; incomplete snapshots, denial and outages are errors, not absence. |
| Policy and drain | Zero-demand create/limit increase has no synthetic launch; seeded leases survive reductions and retirement uncertainty; terminal retirement retains receipts and independent cache/artifact references. |
| Bounded failure and secrets | Oversized/malformed replies, cursor loops, deadlines and cancellation terminate predictably; synthetic secret canaries stay out of diagnostics and captured output. |

Use a required-case manifest and report the exact provider/Platform-contract
commits, digests, command, fixture kind and pass/fail/not-run cases. These tests
establish synthetic HTTP conformance only. Actual atomic multi-repository quotas,
GitHub assignment, native continuation/cleanup ownership and cloud recovery must
be proved in Platform's change C before management/service qualification. Do not
implement their controller inside the provider test fixture to claim completion.

For provider code changes run `nix develop --command just fmt`, `just check` and
`just build` in the locked shell, including focused client/recovery tests and the
existing boundary audit. Extend existing tasks; retain credential-free ordinary
CI. No Nix change is planned; if needed, also evaluate all systems. Generate
resource reference documentation only when a later schema/example change needs
it. Update TF17 and issue #1 with executed evidence after re-reading the tracker;
keep Platform delivery in #56 and native gaps in #20.

#### Logical commits and PR boundaries

| Order / owner | Commit-sized change | Review gate |
| --- | --- | --- |
| 1 — Platform PR | `feat(contracts): define provisional CI governor management profile` | Reconcile #59, schema/vectors/authority/recovery decisions and independent contract validation. Any workflow-contract clarification stays in its own repository. |
| 2 — Provider PR | `feat(client): pin the CI governor contract` | Copy the committed owner bytes and verify digests, version and isolated profile selection. |
| 3 — Same provider PR | `test(client): add governor HTTPS conformance fixtures` | Deterministic persisted peer and independent wire/fault cases, with no physical execution. |
| 4 — Same provider PR | `feat(client): implement recoverable governor management` | Client and required conformance matrix pass, including original-key restart recovery and strict read/error behavior. |
| 5 — Same provider PR | `docs: record governor client conformance and next gates` | Required provider checks pass; source-specific evidence and handoff distinguish synthetic client proof from installed or live support. |

The provider pins committed Platform contract `af60b8a`; merge its
[owner PR #63](https://github.com/spoke-sh/platform/pull/63)
before the dependent provider PR. The contract commit remains an ancestor of
the Platform branch after merging current main. If schema or
recovery ownership cannot be resolved, finish the contract/reproducer handoff
and leave the dependent client work open; do not invent a compatibility fallback.
Contract revisions use hard cutovers while preserving current recovery evidence.

Completion unlocks implementation of the Terraform adapter against an agreed
boundary, coordinated with Platform's retained native-management proof. It does
not unlock live runners, cache acceleration, paid usage or a provider release.
Keep Transit timing/build-reuse work independent; no performance target is an
acceptance criterion for this provider slice.

### Full governor delivery sequence

1. **Pin the public governor HTTP profile.** Agree schema, routes, canonical
   intent/fingerprint vectors, operator authorization, revisions, bounded reads
   and capability refusals. Establish recoverable management identity before
   dispatch, including provider death before saving any ID to Terraform state.
2. **Implement deterministic fixtures.** Platform models global and tenant/account ceilings,
   multiple repositories sharing one allowance, queued/refused/expired demand,
   duplicate notifications, stale observations and uncertain launch/stop. Include
   zero-demand apply, lowered limits, overlapping governors and handoff fencing.
   Provider peers exercise their public management projections and refusals;
   scheduling and physical-effect simulation stay with Platform.
   Mocks cannot qualify GitHub routing, fleet isolation or billing.
3. **Implement HTTP client and governor adapters.** Cover unknown/null handling,
   original-key recovery and explicit drain semantics. Preserve runtime-owned
   counts and account-owned budget policy; no private Transit/GitHub/cloud clients.
4. **Document and package the exact provider.** Generate supported examples only
   after the schema exists; use the existing boundary audit, unit/race, mock
   acceptance and immutable installed-artifact workflow. Examples belong to an
   operator infrastructure root, never the customer workflow repository.
5. **Qualify installed management.** Run independent Terraform scopes with actual
   service retention: create/update/import/empty-plan/delete, authorization,
   stale revisions, provider death before/after acceptance/state save, service
   restart, policy reduction, drain and preserved cache/usage evidence.
6. **Qualify workflow demand against governed supply.** A customer repository with
   no Terraform calls the reusable workflow. Observe actual GitHub demand, one
   admitted job/lease, logs/artifacts and physical termination. Test a second
   repository sharing quota, cross-tenant refusal, cancellation, GitHub outage,
   orphan registrations and broken-control-plane recovery. Change supply policy
   without changing the consuming repository. Paid service still needs billing
   gates; cross-organization access requires independent qualification.

## Roadmap and next proof

Provider main is `1ec09faa16332ed9371e6c16c8fe57fb7ae5f221`; reviewed Platform main
is `dfb88b15e9b8a4bc11989e5d0296ddc685154e5a` at the 05:05 UTC review on 2026-11-04. The current provider
still registers 26 resources/53 data sources and no governor. Native/API/artifact
pins remain unchanged. TF17 owns the proposed operator governor; TF12's original
job/schedule obligations and TF5's completed OCI mock scope remain unchanged.
This corrects the earlier plan's TF12 pool/cache extension; no milestone status
or sequence changes. Existing scope, artifact and lifecycle contracts precede
the governor. Its narrow execution profile does not require all generic services.

Platform P11 owns image/cache trust, P12 worker isolation/recovery, P19 supply
policy and P9/P10 paid usage. P4/P5 own management and identity. The workflow repo
owns customer integration and distribution, not supply policy or cloud effects.
Next provider action: prove Terraform key/sequence selection, then implement the
adapter and installed synthetic matrix above. The schema/client prerequisites
are delivered; retained native service ownership remains open. Transit can
separately improve measured build reuse now; declaring a governor does not itself
accelerate compilation or tests.

## Required evidence before release claims

- No worker or demand from plan/apply without queued work in the zero-warm profile.
- One policy owner and atomic operator/tenant limits across repositories;
  uncertain workers keep reservations until proven stopped.
- Original-key management recovery across provider/service death, with no double
  launch identity or settlement caused by retry.
- Exact provider artifact/schema/service profile in the independent consumer;
  real workflow routing, cancellation/expiry, logs/artifacts and physical cleanup.
- Customer jobs cannot mutate supply policy or impersonate another account;
  secret canaries stay out of state and caches remain isolated.
- Physical cost and billable job usage remain distinct; operator warming is not
  billed as customer execution. Paid rates/reservations require separate proof.

The first pilot is internal Linux/x86_64 with non-spendable usage. ARM64, public
PRs, cross-organization callers and paid customers retain separate gates. The
100× ambition remains a measured build-reuse hypothesis, not a release claim.
