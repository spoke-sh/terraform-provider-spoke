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

### Next slice for review: governor contract and HTTP client

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

Provider main is `b12e6a9fa15b1c7d6243d8c40c9a5d4c132d5a24`; reviewed Platform main
is `89875815497e4e29b4fd4b035faa318f2ba4d857` at the 23:43 UTC review on 2026-11-03. The current provider
still registers 26 resources/53 data sources and no governor. Native/API/artifact
pins remain unchanged. TF17 owns the proposed operator governor; TF12's original
job/schedule obligations and TF5's completed OCI mock scope remain unchanged.
This corrects the earlier plan's TF12 pool/cache extension; no milestone status
or sequence changes. Existing scope, artifact and lifecycle contracts precede
the governor. Its narrow execution profile does not require all generic services.

Platform P11 owns image/cache trust, P12 worker isolation/recovery, P19 supply
policy and P9/P10 paid usage. P4/P5 own management and identity. The workflow repo
owns customer integration and distribution, not supply policy or cloud effects.
Next provider action: review the governor schema/vectors alongside the workflow
contract, then fixture/client work. Transit can separately improve measured build
reuse now; declaring a governor does not itself accelerate compilation or tests.

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
