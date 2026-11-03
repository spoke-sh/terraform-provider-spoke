# Implementation plan: publication declarations in Terraform

[Roadmap](README.md) · [Milestone 18](18-static-publication.md) · **Status:** In Progress

Implement a recoverable Terraform application lifecycle, then project bindings
and environment targets, using Platform's publication HTTP API. Finish this
delivery with an exact installed provider and independent consumer configurations.
This is the next declaration slice of TF18; it does not complete static hosting.

This document is proposed for review. Resource names and new recovery semantics
below are design proposals, not implemented APIs or supported Terraform schemas.
Milestone 18 remains In Progress; milestone 19 remains Planned.

## Baseline and what PR #40 unlocks

- [Platform PR #40](https://github.com/spoke-sh/platform/pull/40) is merged at
  `f43f70d9defc34b9415bb28a6a3bdfceb147c726`. Its
  [normal HTTP profile](https://github.com/spoke-sh/platform/blob/f43f70d9defc34b9415bb28a6a3bdfceb147c726/contracts/http/publication-profile.md)
  supports application policy, binding reservations, target policy, complete
  discovery and retained original operations. Scopes and roles are provisioned
  by the operator outside this provider.
- [Provider PR #14](https://github.com/spoke-sh/terraform-provider-spoke/pull/14)
  is merged at `0eda3a7d873f160b3e61b7a23669640beaf0cc26`. The typed HTTP client
  and its [local qualification](../../qualification/publication-client.md) are
  available; no static Terraform resource is registered. Merged Platform's
  schema/profile bytes match the existing client pins.
- The [post-merge review](platform-adoption-implementation-plan.md#post-merge-review-2026-11-03-0000-utc)
  records source identities and CI evidence. Rechecked at 2026-11-03 00:12 UTC,
  Platform [main Check](https://github.com/spoke-sh/platform/actions/runs/41549175291)
  and provider [main CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41549231269)
  were still running. Existing evidence remains attributed to its original
  sources; this planning change executes no new lifecycle qualification.

At the final source check (2026-11-03 00:15 UTC), the local Platform checkout was
clean at `f43f70d`, matching remote main. It had advanced since the earlier review;
this planning task made no changes to that checkout.

The API removes the missing declaration transport prerequisite. It lets us test
the actual Terraform adapter without waiting for uploads or customer onboarding.
It does not yet define durable Terraform resource ownership across multiple
mutations, replacement and state loss. Resolve that boundary first.

## Proposed delivery boundary

Use three resources and corresponding exact and collection data sources:

| Proposed resource | Scope and configured intent | Observed state and replacement |
| --- | --- | --- |
| `spoke_static_application` | Project; stable management key, name, deletion protection, retention days | Scoped ID, revision and readiness/reason. Name and policy update in place. Scope or management identity changes require explicit replacement. |
| `spoke_static_target_binding` | Project; stable management key, application ID and destination environment ID | Reservation ID/revision, lifecycle state and nullable target ID. Changing application or environment replaces the reservation. There is no binding update route. |
| `spoke_static_target` | Environment; stable management key, binding ID, name, deletion protection, retention days | Scoped ID/revision, binding/application identity and observed runtime status. A different binding requires replacement; policy updates in place. |

The proposed management key is a stable, non-secret resource identity. It is
**not** the current API's one-operation key. Its admission and ownership semantics
must pass the first gate below before appearing in a writable resource.

Use exact data-source names matching the resources and plural collection names
(`spoke_static_applications`, `spoke_static_target_bindings`,
`spoke_static_targets`). Binding collections require an application ID because
that is the implemented collection scope. Do not add unimplemented filters.

Require explicit name and retention values initially; deletion protection
defaults to true. Match the owning schema's identifier and retention bounds. Keep decimal
IDs/revisions as strings. Preserve null endpoints and pending readiness; successful
management settlement must not imply an available website. Runtime heads,
generations and bundle references are computed observations only.

Proposed canonical import forms are `organization/project/application_id`,
`organization/project/binding_id` and
`organization/project/environment/target_id`, interpreted by resource type.
Import validates scope and existing management ownership through authoritative
reads. It cannot claim an unmanaged declaration or transfer another principal's
ownership. Read-only data sources can inspect objects within their granted scope.

No upload, sealing, publication selection, preview, promotion, rollback or domain
resource belongs in this delivery. Do not expose the synthetic peer profile.
Per-file bytes, manifests and credentials stay outside Terraform state. General
scope bootstrap, a hosted state backend and public Registry release retain their
separate qualification gates.

## Implementation progress

The [retained-slot identity decision](../publication-identity.md) has executable
normal-API evidence at provider `8e225b6289977dc655575e4974460e3ac8f39d93`.
The application resource and exact data source now implement policy lifecycle,
principal-preserving import and revision fences. Synthetic Terraform lifecycle
and all nine create/update/delete process-death schedules passed. Exact installed
normal-API qualification is the next gate before binding/target registration.
TF18 remains open for the remaining declaration graph and delivered bytes.

## A. Prove durable identity and ownership

**Owners:** provider for Terraform semantics; Platform for durable admission and
the public HTTP contract. This is a prerequisite for registering writes.

Produce a contract decision and executable recovery cases before implementing
the application resource. The existing API atomically retains an operation after
the caller supplies its key; the caller still needs a recoverable way to select
that key and its exact original revision/intent. Application responses currently
contain no Terraform management claim or resource-incarnation record.

| Situation | Required behavior |
| --- | --- |
| First create; death before Terraform saves any ID | Configuration plus authorized server evidence locates the original accepted intent and resource. A retry cannot allocate a duplicate. |
| Update A → B → A | Each new logical update has a distinct identity; replay uses the original identity and precondition. A configuration hash cannot stand in for a mutation sequence. |
| Lost update/delete response; refresh before retry | Find pending/original work before admitting another mutation. Refresh cannot replace its original revision fence with a newer one. |
| Delete followed by recreation or replacement | Retain the old incarnation and receipts; create a separately fenced incarnation. An old saved plan or delayed request cannot mutate the replacement. |
| Import, token rotation, another principal or cloned state | Preserve principal-bound ownership. Rotation under the same principal recovers; another principal cannot claim the resource. Identical principal/key clones remain indistinguishable from recovery and require external state locking. |
| Expired receipt, partial history or lost parent access | Retain uncertainty and tombstones; never infer permission for a fresh mutation or child absence. |

**Recommended design if current primitives cannot prove these cases:** a
Platform-owned durable management record, addressed by scope, resource kind,
principal and a stable management key. It records resource incarnation,
management generation, resource ID, original pending/terminal operation and the
accepted normalized intent/preconditions. Admission atomically binds a mutation
to that record using explicit generation/revision preconditions; authorized
read-only lookup survives provider death. Retired records remain discoverable.
Platform must define how a new incarnation is authorized and how stale plans
are rejected, including both Terraform replacement orders.

This describes required semantics, not a proposed route asserted to exist.
First test whether current primitives provide an equivalent construction. If
they do, document the derivation and prove every case above. If they do not,
implement and qualify the smallest Platform HTTP extension before resource
writes proceed. Read Platform's own instructions before that cross-repository
work. Platform owns schema/version changes, authorization, durable storage and
native payload bounds; the provider consumes schema-derived HTTP types only.

Do not solve this with a provider journal, an unsaved random key, a configuration
hash alone, receipt-history guessing, or the bucket management-key algorithm.
Do not compute native fingerprints/codecs in the provider. Manually changing an
operation key for every Terraform apply is not the intended resource interface.

**Exit gate:** executable counterexample tests and passing supported construction
for all rows, including process death; documented import/recreation/replacement
rules; exact pinned contract; and explicit refusal for unsupported combinations.
Any required Platform change lands and qualifies before dependent provider writes.
Contract cutovers update client, fixtures and examples together and reject
incompatible installations before mutation. They do not erase retained services.

## B. Deliver one application lifecycle

1. Extend configuration in `internal/provider/provider.go` with a separate typed
   publication client using existing endpoint, external credentials, CA and
   timeout settings. A project alias has organization/project and no environment;
   a target alias also names its environment. Never silently broaden an
   environment-scoped configuration into project authority. Instantiate clients
   without provisioning or eager mutation. Negotiate each surface's normal
   profile/capabilities before its API work; reject a wrong or peer profile.
2. Implement the application resource, exact discovery and import in dedicated
   publication files under `internal/provider/`. Reuse the strict client in
   `internal/platformclient/publication*.go`; keep the existing bucket/mock-v5
   path independent. Resolve the client's current one-minute internal mutation
   cap against configured timeout semantics, with one documented bounded deadline.
3. Look up and recover accepted work before admitting a new mutation. Apply
   expected revisions, wait for management completion, then read authoritative
   state. Preserve the original operation on cancellation, outage or timeout.
   Read/refresh/data sources only observe. Remove state only for authorized,
   authoritative absence of that exact resource; permission loss is an error.
4. Make null/unknown handling, schema defaults and replacement rules explicit.
   Reject unknown provider scope at configuration; allow resource dependencies to
   become known at apply. Protection must be disabled by an explicit prior apply.
   Deletion cannot release bindings, remove independent data or force a cascade.
5. Add the separate publication profile to test fixtures under
   `internal/testutil/mockplatform/` and contract fixtures under
   `testdata/contracts/`. Update the protocol registry HTTP audit for each new
   resource/data source; do not bypass its coverage with special exemptions.
   Add application examples and regenerate reference docs using the repository
   script. Add the initial explicit installed application qualification lane and
   its process barriers now; expand it in C/D. Update required-case/release
   coverage with each registered surface. Record the precise experimental scope
   in the capability documentation.

**Exit gate:** actual Terraform validate/plan/apply/refresh/import/destroy and an
empty second plan pass against the normal installed Platform API, including
drift, revision conflicts, protection and killed-provider recovery. Complete
focused mock coverage and existing regression checks. Publish this evidence
before broadening the registered resource surface to bindings and targets.

## C. Add project reservations and environment targets

Implement binding and target resources plus the remaining exact/collection data
sources. These are separate ownership boundaries, not nested resources that
implicitly grant access across environments.

- A binding references the application ID and destination environment. Resolve
  the application's current revision when admitting a new reservation and retain
  it with that original intent. Do not make a changing application revision an
  ongoing Terraform dependency that replaces every existing binding on a policy
  edit. Recover an uncertain reservation before reading a new admission fence.
- Target creation consumes one exact reservation using its own environment
  principal. Policy updates own name/protection/retention only. Observe selected
  heads without trying to reset them or implementing the unavailable withdrawal.
- Target deletion must leave an authoritative tombstone before reservation
  release. A lost or denied release retains the conservative application hold.
  Binding deletion performs only the authorized server-defined cancellation or
  recovery; it cannot infer child absence from denied discovery or parent loss.
- Exercise cancellation racing with consumption, failed target creation with a
  retained hold, stale revisions, duplicate consumption, late effects and
  recovery of an interrupted release. Pending/unresolved or selected references
  that prevent deletion remain explicit errors; no force-delete workaround.
- Collections follow one complete bounded snapshot, recheck authority and
  restart within a bounded budget on stale cursors. Partial pages never establish
  absence. Keep operation discovery bounded and principal-aware.

**Exit gate:** installed Terraform manages a project application and distinct
nightly/production targets with separate credentials and state. Application
deletion is blocked until authorized child retirement/release completes. Repeated reads
and plans issue no mutations. Cross-environment access and ownership attempts
fail without losing state. Every new schema has examples, generated docs and
HTTP boundary coverage.

## D. Qualify the exact artifact and independent consumer

Expand the installed-process harness and explicit declaration qualification lane
started in B to the full graph; retain the five public Just tasks. Use a distinct
declaration evidence kind/gate in `internal/testutil/liveplatform/`, with an explicit clean
Platform fixture revision and publication schema/profile. Do not reinterpret the
existing bucket metadata gate or select a backend from ambient credentials.
The current Go client fixture is useful setup, but its test binary is not an
installed Terraform lifecycle result.

Build a clean, committed Linux amd64 artifact and mirror. Update the independent
consumer flake pin, exact provider constraint, Nix lock and Terraform lock
together. Run a copied consumer outside this checkout with readonly checksums,
no development override and no compiler on Terraform's PATH. Record the exact
binary hash; a missing artifact must fail rather than build a replacement.

Use three roots: a project catalog owns the application/reservations; nightly
and production own their respective targets and consume explicit catalog IDs.
Do not declare the application independently in both environment states. Use
read-only discovery or explicit output handoff without sharing credential values.
Test provider aliases with the installed binary; an in-process reattach server
shares configuration and cannot qualify isolation. Local independent roots do
not qualify a hosted encrypted/locked state service.

| Acceptance group | Required observable proof |
| --- | --- |
| Normal lifecycle | Validate, plan, apply, import, drift correction, policy update, refresh, empty plan and protected/dependency-restricted destroy through Terraform. |
| Identity and concurrency | A → B → A; same-intent replay; conflicting intent/principal; stale saved plan; supported replacement orders; no duplicate resource or stolen ownership. |
| Process death | Kill the separate provider before submission, after durable acceptance before reply, and after settlement before Terraform saves state; retry create/update/delete with the original operation/incarnation. |
| Service restart | Restart the isolated Platform fixture and recover its durable records separately from killing Terraform's provider. Instrumentation must expose a barrier only after durable acceptance. |
| Scope and authentication | Project/nightly/production isolation, denied reads, permission revocation, same-principal token rotation and incompatible profile rejection. Secrets absent from state, plans, diagnostics and logs. |
| Binding races and retention | Cancellation/consumption fencing, target tombstone and lost release, administrative recovery, application hold, retained independent data and blocked selected-target deletion. |
| Observational and bounded work | GET-only plan/refresh/discovery; complete pagination and stale snapshots; request/settlement deadlines; expiry tombstones; explicit capacity exhaustion without key reuse. |

Use a fresh isolated installation per bounded scenario group, budgeting operations
before execution: the current profile retains at most eight application IDs,
sixteen binding IDs, eight target IDs per environment and 64 receipt IDs per
owner. Deleting objects or expiring receipts does not reclaim these slots.
Exercise exhaustion deliberately in its own fixture; never reset a customer's
installation or prune receipts to make a test pass.

Record provider/Platform commits and cleanliness, API/schema hashes, Terraform
and Go versions, OS/architecture, artifact/mirror hashes, test commands, executed
cases, failures and unexecuted service gates in a new qualification record/report.
Update `internal/testutil/liveplatform/cases.json` and release validation for the
actual new advertised surface so declaration evidence cannot satisfy full static
delivery or Registry release. Preserve prior evidence and immutable artifacts.

**Exit gate:** every declaration case passes using the exact installed artifact
against the normal Platform API. The independent consumer reproduces the result.
Missing, skipped, mock or peer cases cannot be counted as installed qualification.
TF18 remains open for immutable bytes and hosted delivery.

## Review units, logical commits and checks

Deliver small dependent PRs rather than registering all resources at once:

| Review unit | Logical commits | Required gate before the next unit |
| --- | --- | --- |
| 1. Recovery contract | Contract decision and counterexamples; Platform companion if required; pinned client/types and recovery tests | Gate A, with the Platform dependency qualified before writable resources |
| 2. Application | Client/configuration plumbing; application lifecycle/discovery with fixtures, examples and generated docs; explicit installed application lane and evidence | Gate B |
| 3. Binding and target | Reservation lifecycle; target policy and dependency recovery; collection discovery and independent-root examples, each with tests/docs | Gate C |
| 4. Consumer qualification | Complete declaration qualification matrix and process cases; exact consumer artifact/locks; execution record and roadmap update | Gate D |

Keep each behavior commit reviewable with its focused tests and schema/docs.
Record immutable artifact evidence after committing the executable revision;
an evidence-only follow-up commit must retain that tested source identity.
Refresh upstream contracts, relevant trackers and main/push workflow artifacts
before each contract/lifecycle/qualification change as required by `AGENTS.md`.
Use provider trackers [#1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
and [#13](https://github.com/spoke-sh/terraform-provider-spoke/issues/13) for
implementation/evidence follow-up when implementation work is authorized.

For code deliveries, run focused behavior tests followed by `just fmt`,
`just check` and `just build` in the locked Nix shell. Run
`./scripts/generate-docs.sh` after schema/example changes and
`just test-acc installed` for the exact artifact; run the explicit Platform
declaration lane separately from credential-free ordinary CI. For Nix changes,
also run `nix flake check --all-systems --no-build --no-update-lock-file` and
verify shell entry. Evaluation does not establish additional runtime support.
Keep workflow actions pinned, permissions read-only and checkout credentials
unpersisted. This planning-only change needs documentation checks, not code tests.

## Parallel Platform work and the next milestone gate

Platform can independently deliver
[change C: object transfer and immutable sealing](https://github.com/spoke-sh/platform/blob/f43f70d9defc34b9415bb28a6a3bdfceb147c726/docs/roadmap/saga-publication-handoff-implementation.md#c-deliver-the-p6-object-slice-and-immutable-sealing).
That work must independently verify actual immutable object versions/digests;
provider declaration success cannot substitute for it. Preview/publication,
promotion/rollback, domain ownership/TLS and Saga Actions follow their own
implemented API and service gates.

Complete TF18 and the Saga consumer journey only after independent HTML/JS/Wasm/
asset fetches, audience/cache isolation, stale-effect fencing, TLS, warm-cache
rollback and open-page asset retention pass with exact artifact/API evidence.
This plan does not change TF10's existing service graph, the original milestones
1–17 compatibility gate, or milestone 19's model-resource prerequisites.
