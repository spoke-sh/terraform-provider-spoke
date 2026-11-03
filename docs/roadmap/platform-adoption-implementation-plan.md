# Implementation plan: adopt current Platform capabilities

[Roadmap](README.md) · **Status:** In Progress

Use Platform's implemented HTTP bucket profile to establish a reproducible
provider integration, then qualify real services as their APIs arrive. The bucket
profile and explicit metadata lane are implemented. Static publication now has
a locally qualified declaration HTTP client; Terraform resources and delivery
remain separate gates. Model resources remain planned.
Implementation was authorized on 2026-11-02. The
[execution record](../../qualification/platform-adoption.md) records delivered
profile/client work, executable metadata qualification and remaining service gates.

This plan supersedes the upstream assessment, Platform numbering and immediate
ordering in the [10–17 plan](milestones-10-17-implementation-plan.md). Its resource
acceptance criteria remain in force. Provider milestones 1–9 retain their recorded
completion scope; 10–17 remain In Progress. Platform's broader roadmap does not
silently add resources or change the provider's release requirements.

## Implementation review: 2026-11-03 01:02 UTC

Platform local HEAD/tree and fetched remote main are clean at
`65f8284d0197ccc0ed450dc8c3e0b074947979cb` (the checkout advanced independently
since the earlier `f43f70d` inspection). [PR #41](https://github.com/spoke-sh/platform/pull/41)
adds the immutable upload/sealing implementation plan and clarifies that no new
routes are enabled. Publication schema bytes still match this provider's pin;
the source profile copy remains attributed to its original committed source.
Trackers #20/#24/#26/#29 were re-read; object transfer and hosted delivery remain
Platform work, while installed Terraform declarations are provider work.

- [Check main/push](https://github.com/spoke-sh/platform/actions/runs/41553977933)
  is running: billing and publication proof passed; publication API qualification
  is running; the main integration job is still executing checks, with installed
  bucket qualification and browser/container steps pending. The previous
  `f43f70d` Check was cancelled, not successful.
- [Native adoption main/push](https://github.com/spoke-sh/platform/actions/runs/41553978091)
  passed all steps. Its downloaded artifact has six passing checks, but
  `native_management_continuation` and `revoked_caller_terminal_receipt` remain
  blocked; HTTP provider parity was not run and `adoption_ready` is false.
- [UX main/push](https://github.com/spoke-sh/platform/actions/runs/41553977838)
  passed all steps. The downloaded manifest matches clean `65f8284`; its 11
  accessibility audits passed with zero violations. Its declared scope is the
  built React app with a local synthetic workspace, not service qualification.

The provider's retained publication-key construction passed real normal-API
protocol qualification at clean provider `8e225b6289977dc655575e4974460e3ac8f39d93`
against Platform `f43f70d9defc34b9415bb28a6a3bdfceb147c726`; see the
[identity record](../../qualification/publication-identity.json) and
[decision](../publication-identity.md). Application lifecycle and nine process-death
schedules also pass against the separate synthetic publication fixture. Next:
qualify the exact committed installed application artifact against the normal
API before registering bindings and targets. No pin adoption or milestone closure
follows from these upstream workflow results.

## Post-merge review: 2026-11-03 00:00 UTC

[Provider PR #14](https://github.com/spoke-sh/terraform-provider-spoke/pull/14)
is merged at `0eda3a7d873f160b3e61b7a23669640beaf0cc26` (current clean local
and remote main). [Platform PR #40](https://github.com/spoke-sh/platform/pull/40)
is merged at `f43f70d9defc34b9415bb28a6a3bdfceb147c726` (remote main).
The local Platform checkout remains clean at `e1f4daa`; it was not advanced or
modified. The merged publication schema and profile are byte-identical to this
provider's existing pins. No contract or artifact pin is changed by this review.

The publication-api job in [PR Check 41544993901](https://github.com/spoke-sh/platform/actions/runs/41544993901)
passed all its steps. Its downloaded report records 15 scenario-group entries
across normal/peer profiles in 144.98 seconds, against GitHub's test merge
`cb58190013ce4c0b4544359928aa27735dad54c5`; the schema digest matches this client.
That run's publication and billing jobs also passed, while the general platform
job remains in progress. The report explicitly leaves installed static Terraform,
real uploads/sealing, preview/domain delivery, Saga Actions and scale unqualified.

Latest Platform main/push [Check 41549175291](https://github.com/spoke-sh/platform/actions/runs/41549175291)
and [native feasibility 41549175286](https://github.com/spoke-sh/platform/actions/runs/41549175286)
are in progress; [UX 41549175348](https://github.com/spoke-sh/platform/actions/runs/41549175348)
passed. Provider [merge-head CI 41549231269](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41549231269)
and PR #14 CI are also still running. No pending checks or absent artifacts are
treated as passing qualification. The previous attributed local client/installed
evidence remains unchanged; no executable suite was rerun for this source review.

**Next provider work:** the [TF18 resource sequence](18-static-publication.md#next-provider-delivery-sequence)
starts with a proved per-mutation recovery/ownership contract and one real
application resource, then adds binding/target lifecycle and a checksum-pinned
installed consumer. The available API removes the blanket missing-declarations
blocker. Original operation receipts do not by themselves define Terraform's
resource ownership, replacement or recovery after unsaved state; prove these
semantics before registering a resource and assign any missing durable behavior
to Platform's HTTP contract. Eight applications, sixteen bindings, eight targets
per environment and 64 receipts per owner are unreclaimed lifetime capacities,
so this remains a bounded development profile.

The [publication Terraform implementation plan](publication-terraform-implementation-plan.md)
now expands that recommendation into four review units, proposed resource schemas,
the durable recovery contract decision and exact installed acceptance. It remains
Planned pending review; no Terraform static resource or new API is introduced by
the planning document.

**Planning follow-up, 2026-11-03 00:15 UTC:** the local Platform checkout is now
clean at merged main `f43f70d`, matching its remote. It advanced after the source
review above; this planning task did not modify it. Provider main remains
`0eda3a7` with roadmap-only planning edits. At the 00:12 UTC recheck, Platform
Check 41549175291 and provider CI 41549231269 were still in progress.

**Next Platform work:** change C of its
[Saga handoff plan](https://github.com/spoke-sh/platform/blob/f43f70d9defc34b9415bb28a6a3bdfceb147c726/docs/roadmap/saga-publication-handoff-implementation.md#c-deliver-the-p6-object-slice-and-immutable-sealing)
delivers real immutable object transfer, independent version/digest verification
and sealing. Preview/CDN delivery, promotion/rollback, domains and real Saga
Actions follow their own gates. Terraform declaration work can advance using
operator-provisioned scopes without waiting for customer onboarding or all of
P6; website delivery still requires those actual object bytes. Models remain planned.

Read Platform's current instructions, roadmap/alignment, contract/source and
qualification records, trackers #20/#24/#26/#29/#36 and provider #13. GitHub's
connector returned 404; authenticated `gh` reads supplied remote verification.
No issue, service or credential was changed.

## Implementation update: 2026-11-02 23:30 UTC

The Platform checkout remains clean at `e1f4daa`, with remote main still
`71d43c1f522657280c5c5bb1e5187cc617b13d95`. An existing independent Platform
worktree implemented the machine API while this provider change was prepared;
it was neither modified nor duplicated. [Platform PR #40](https://github.com/spoke-sh/platform/pull/40)
is open at `e0b8bb3bdd94a467c8281159f595792f5262573d`, with installed API
qualification attributed to `c56f82d41ec9b4fc3617f732e2c900773cc00080`.
Its normal declaration profile has no upload, sealing, promotion or domain API.

This provider now implements a separately negotiated `publication-v1` HTTP client
against the committed candidate schema from
`6627bed1179b7cb95f51831d26d35e207c65481f`. The fixture revision has identical
schema/profile bytes. Strict schema, decimal and complete-snapshot checks bind
project/application and environment/target reads; original keys, principal/receipt
identity and exact HTTP replay preserve uncertain mutation intent. Native codecs,
synthetic peer selection and runtime heads stay outside the client.

The [execution record](../../qualification/publication-client.md) and
[machine-readable report](../../qualification/publication-client.json) record
six passing case groups against a new isolated installed Platform fixture, using
the actual Go HTTP client at `67c9a7c52338ed5ed41c90d197b1f9eedae66b67`.
The killed caller and replacement process recover the same retained operation.
These results qualify client declarations, not Terraform apply/import or static
hosting. TF18 is now In Progress, with no new registered resource and no consumer
artifact upgrade. Full service/release gates remain unchanged.

Next, implement and qualify Terraform's per-mutation identity and import/ownership
adapter, including update/delete, A → B → A and death before state save. Keep
declarative policy independent of runtime promotion. Platform continues with
immutable upload/sealing and actual delivered bytes; those prerequisites remain
separate from the available declaration API. The next consumer proof is an exact
installed Terraform artifact followed by independent preview/stable-domain fetches.

The three latest main/push runs remain those recorded below: Check
[41538217263](https://github.com/spoke-sh/platform/actions/runs/41538217263) is
still in progress; native feasibility and UX passed with the same stated limits.
PR #40's [Check 41544993901](https://github.com/spoke-sh/platform/actions/runs/41544993901)
is also in progress. Local qualification does not substitute for either CI result.

**Follow-up workflow review, 23:42 UTC:** main Check 41538217263 completed
successfully at 23:33 UTC. All platform, billing and publication steps passed;
none was skipped. Downloaded artifacts bind installed bucket metadata to the
same exact `67682f0` provider and record twelve schedules, plus ten publication
proof and eleven billing cases at main `71d43c1`. Browser management admission
remains explicitly blocked; publication upload/sealing/hosting and actual Stripe
sandbox funding remain unqualified. This retires the pending merge-head check,
not the prior failed runs or service gates. PR #40's publication-api and platform
jobs remain in progress; its proof, billing, feasibility and UX jobs passed.

## Progress review: 2026-11-02 22:50 UTC

This earlier review describes merged main before the API candidate and provider
client above; its source identities and workflow observations remain historical.

Reviewed clean local Platform `e1f4daa9a2718c16be321a87e8b1dd065396f667`
and fetched remote main
[`71d43c1f522657280c5c5bb1e5187cc617b13d95`](https://github.com/spoke-sh/platform/commit/71d43c1f522657280c5c5bb1e5187cc617b13d95).
Remote main includes publication PRs #37–39 absent from the local checkout;
the checkout was not advanced or modified. Provider local/remote main is
`05083677bf1f93021841d5491fc37694a5e068b0`, which includes merged PR #12.
Platform's handoff documents still cite older provider main `9bc3cfd`; their
installed consumer correctly pins our qualified `67682f0` artifact. Production
Transit remains `94fa12ff6d8cd3e409ac6524f721a17d1a245abb`. Bucket gateway source,
profile schema, vectors and full inventory schema are unchanged from the initial
review below; their hashes still match this provider.

Source, contracts, qualification records, recent merged PRs, Platform trackers
#20/#24/#26/#29, consumer #36 and provider #13 were inspected through local Git
and authenticated GitHub reads. The five new deterministic HTTP recovery tests
passed in the locked provider shell. Native publication/billing and installed
integration suites were not rerun in this review; their results below retain
the upstream reports' exact source attribution. No issue or pin was changed.

| Addition | Verified capability and limit | What it unlocks / next owner and proof |
| --- | --- | --- |
| Publication native proof, merged PR #38 | [Ten local scenarios at `fec5c12`](https://github.com/spoke-sh/platform/blob/71d43c1f522657280c5c5bb1e5187cc617b13d95/docs/qualification/static-publication.md): retained decisions, current authority, actual process death, unknown effects, rollback and stale-ETag refusal. Two fixture bundles, four operation slots and one recovery read per operation; no public API, uploads or AWS delivery. | Platform P7 can proceed to its machine contract without waiting for general compute, hosted builds or paid onboarding. No reusable native blocker was observed in this finite slice. TF18 can coordinate the contract now; actual provider resources still require the implemented API. |
| Publication machine API plan, merged PR #39 | [Change B](https://github.com/spoke-sh/platform/blob/71d43c1f522657280c5c5bb1e5187cc617b13d95/docs/roadmap/publication-machine-api-implementation.md) specifies application/target policy, project/environment authority, recovery and bounded history. Documentation only; no publication HTTP schema/routes are implemented. | Agree canonical schemas/vectors, distinguish declaration ownership from runtime heads, and prove cross-owner binding/deletion races plus framed-command bounds. Then implement and qualify HTTP declarations before adapting the provider. Upload/sealing, domains and delivered bytes retain separate gates. |
| Native billing foundation, merged PR #34 | [Eleven local cases at `4ee1f41`](https://github.com/spoke-sh/platform/blob/71d43c1f522657280c5c5bb1e5187cc617b13d95/docs/qualification/billing-foundation.md): retained purchases, verified event admission and independently captured payment observations against a deterministic HTTPS peer. Totals are non-spendable; no actual Stripe sandbox funding or customer billing. | Platform P9 can implement the selected real sandbox payment/event slice. This adds no Terraform resource, paid storage admission or usable customer credit. |
| HTTP fixture recovery follow-up | [Installed report at `d7af062`](https://github.com/spoke-sh/platform/blob/71d43c1f522657280c5c5bb1e5187cc617b13d95/docs/qualification/http-recovery-local.json) passes against the same exact `67682f0` binary. Receipt-first backoff replaces three rapid replay attempts within the existing deadline; incomplete observations never permit replay. | Requalify the existing metadata lane against a selected newer clean Platform fixture when adopting it. No provider schema or artifact update is required by this fixture-only correction. |

Publication's proposed `publication-v1` transport differs from bucket `mock-v5`:
one operation key identifies one mutation, revisions are decimal strings, and
recovery observations are explicit child operations. Do not reuse the bucket
management-key/generation algorithm or expose guessed resource names. Keep
Terraform application/target policy independent of runtime promotions. The
provider remains HTTP-only; billing/publication native codecs stay in Platform.
Consumer [#13](https://github.com/spoke-sh/terraform-provider-spoke/issues/13)
permits an independently qualified immutable Linux mirror for Saga; a broad
Registry release is not a prerequisite for that narrow static consumer.

Latest `main`/`push` evidence at `71d43c1`, observed at the review time above:

| Workflow | Observed result and scope |
| --- | --- |
| [Check 41538217263](https://github.com/spoke-sh/platform/actions/runs/41538217263) | In progress; main checks and independent billing/publication jobs have not established a passing merge-head integration result. |
| [Native adoption 41538217276](https://github.com/spoke-sh/platform/actions/runs/41538217276) | Passed. Downloaded artifact has six passing feasibility checks, `adoption_ready=false`, blocked native continuation and revoked-caller terminal receipt gates, and HTTP parity not run. No owner replacement is unlocked. |
| [UX 41538217294](https://github.com/spoke-sh/platform/actions/runs/41538217294) | Passed; no new publication UI, browser command or hosted-service qualification. |

The later failed [Check 41511895226](https://github.com/spoke-sh/platform/actions/runs/41511895226)
at `e1f4daa` exhausted three cleanup replays. PR #38 addresses that fixture limit;
its local pass does not relabel this failed run. Checks at `ee80731` and `d9b92ff`
were cancelled. Models remain planned, and real object/database/OCI/workload,
hosted delivery and full release gates remain open. TF18 remains Planned while
Platform P7 is now In Progress; the two statuses describe different delivered work.

## Reviewed baseline

The following is the initial review, retained with its historical identities and
workflow outcomes. The progress review above supersedes its current-state claims.

Verified review: **2026-11-02 07:57 UTC** (2026-11-01 Pacific). Source inspection,
GitHub issue reads and workflow/log inspection were performed; no upstream suite
was rerun. The GitHub connector returned 404, so authenticated `gh` reads supplied
remote verification. Both worktrees were clean at inspection.

| Input | Reviewed identity and meaning |
| --- | --- |
| Provider | `9bc3cfd90ff2656e3e98ce02a99bdd3025aff61c`; HTTP-only, 23 resources and 47 data sources, provisional mock-v5 |
| Local Platform | `d2a5a251be1b947e8cb4063b789d358ce2508cd7` |
| Platform remote main | [`ebf97fe086a99ba91f6b669b9f922f3964eb5bfa`](https://github.com/spoke-sh/platform/commit/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa), merge of static publication PR #30; its tree `44f935fe4a4f215a43b3b027cbc86dfe415c7a18` matches local Platform HEAD |
| Platform production Transit pin | `94fa12ff6d8cd3e409ac6524f721a17d1a245abb`; a backend input, not a provider dependency |
| Platform's installed provider | Revision `9bc3cfd90ff2656e3e98ce02a99bdd3025aff61c`, Linux amd64; SHA-256 `393040cbaedfeb804d56de87f9c58c16aca354dc2b25b24fe70aa041f2ccdd9d`, from [provider.json][provider-pin] |
| Provider inventory schema | SHA-256 `a3100081694a125dee8779179d1a85fb3f5ff1cdd8af6f7a8dab46dd35333c0b`; Platform's copy matches the provider contract identity |
| Platform bucket profile schema | SHA-256 `8568a6483a450b79464386d9cd6e5be5c7ad0c4163ceecd0ffe3a7682b4341bd`; narrower than the inventory |
| Platform canonical vectors | SHA-256 `48367b59715053a70a40979ddb57f615dcd9e2d38b29923cc48f14d948a1fcc3`; create/update/delete fingerprints |

The independent provider consumer still pins `b3db62a`, as recorded in
[provider qualification](../../qualification/platform-http.md). It is a different
artifact from Platform's `9bc3cfd` consumer. Do not combine their hashes or claim
that either result qualifies a newly built candidate.

Reviewed Platform trackers: [native adoption #20](https://github.com/spoke-sh/platform/issues/20),
[Saga workloads #24](https://github.com/spoke-sh/platform/issues/24),
[models #26](https://github.com/spoke-sh/platform/issues/26) and
[static publication #29](https://github.com/spoke-sh/platform/issues/29), plus the
provider [implementation tracker #1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1).
No issue was edited by this review.

### Evidence and current workflow results

Platform [PR #19](https://github.com/spoke-sh/platform/pull/19) delivers the HTTP
boundary. [PR #28](https://github.com/spoke-sh/platform/pull/28) adds the owned
schema/vectors, bounded pending discovery and expanded failure qualification.
The [HTTP record][http-evidence] attributes local P4 execution to committed source
`2106e7314dc3390243cb8f32e3eaf49257f1cd62`; subsequent runs need their own reports.
It records CRUD/import, rename, empty plans, denied refresh, protected deletion,
lost responses, retained server restart, token rotation, quota/owner races and
four actual Terraform/provider process-death or expiry schedules. Those are
local metadata results, not storage or hosted deployment evidence.

Latest `main`/`push` runs at reviewed remote revision:

| Workflow | Observed result | Interpretation and next action |
| --- | --- | --- |
| [Platform checks, run 41397981615](https://github.com/spoke-sh/platform/actions/runs/41397981615) | Failed | Installed HTTP provider step fails in `tests/http-api.integration.py:403`: concurrency cleanup DELETE `/buckets/6` expected 202, received 503 `unavailable`. Subsequent browser/container steps were skipped. Reproduce and classify before claiming this revision qualifies. |
| [Native adoption, run 41397981575](https://github.com/spoke-sh/platform/actions/runs/41397981575) | Passed | Feasibility workflow succeeds independently of adoption gates. The [adoption record][native-evidence] still blocks replacement on durable continuation and authorized refusal/cleanup after revocation. |
| [UX review, run 41397981921](https://github.com/spoke-sh/platform/actions/runs/41397981921) | Passed | Automated UI checks/capture; no proof of browser management admission or usable services. |

The failed DELETE's root cause is not established by the assertion alone.
Preserve the failure and inspect admission/receipt state; do not call it harmless
flakiness or fix it with blind write retries. Platform's scripts execute raw HTTP
checks before the installed Terraform matrix, so the workflow step name does
not prove that every installed-provider case ran.

## Contract differences to reconcile

The [HTTP contract][http-contract] and [shared management profile][profile] now
resolve routes, bearer authentication, ownership and recovery for one bounded
profile. The old blanket prerequisite “Platform HTTP API does not exist” is
obsolete. The provider's complete schema is still provisional.

| Area | Implemented Platform profile | Provider action |
| --- | --- | --- |
| Version and inventory | Wire version `mock-v5`; response header `X-Spoke-Management-Profile: spoke.platform.http.bucket.v1`; capabilities advertise only `bucket` | Track wire version, profile/schema and provider inventory separately. Do not mark the whole `contract.json` published because this subset works. |
| Scope/authentication | One operator-provisioned organization/project/environment; external bearer token maps to a server-side native principal; same-owner token rotation preserves ownership | Retain external token/CA files and explicit targets. No public project/account/grant bootstrap is available. Browser cookies/Origin are refused. |
| Input bounds | Names 3–63 lowercase ASCII letters/digits/hyphens, initial letter and final alphanumeric; keys 1–128 ASCII identifier bytes; object profile, capacity 1–2^50 bytes and grow-only; retention 0–36500 days | Compare client/schema validators and fixtures field by field. Test boundaries, null/unknown values, shrink rejection and server refusals without guessing new syntax. |
| Numeric/concurrency limits | Counters at most 2^53−1; expected generation leaves room for the next receipt; 32 buckets/reservations and 32 principals; 64 KiB mutations; one concurrent request per principal | Keep client work bounded and distinguish quota, contention and uncertain transport results. Do not interpret same-signer collision tests as replica support. |
| Fingerprints | Canonical ordered JSON plus `mock-v5` and newline; generation is a separate fence; delete omits `spec` | Consume Platform golden vectors independently of provider-generated expectations; verify import, replay, A → B → A and conflict semantics. |
| Observations | Complete single-page bucket snapshot; bounded owner-filtered pending discovery at `GET S/management/bucket?state=accepted`; changing cuts or invalid continuation fail | Retain complete-or-error reads and strict response decoding. Add typed bounded pending discovery alongside the existing exact-key lookup; do not register a new Terraform data source without a caller requirement. Pending discovery is not terminal history. |
| Recovery | Durable owner/key/generation receipts, original-operation lookup, settlement authorization, terminal tombstones; expiry never permits reuse | Preserve original identity across lost state/replies, denial and death. Import cannot transfer ownership; identical credentials/keys in cloned states remain indistinguishable from recovery. |
| Service readiness | Declaration settles with `readiness=pending`, `dependency_unavailable`, empty endpoint and no active revision | Record metadata success separately; never infer object storage, data access or full release readiness. |

Richer payload admission/retention, bounded terminal history and browser command
delegation remain P4 work. Platform's native owner replacement is separate from
using the current HTTP owner. Do not wait for browser or native replacement work
to qualify the available machine bucket profile.

## Implementation sequence

### 1. Reconcile the profile and failing integration (TF10)

Delivered in the [bucket adoption record](../../qualification/platform-adoption.md).
The requirements below remain the basis for subsequent profile cutovers.

Provider changes: pin an immutable Platform profile/schema/vector source with
provenance under `testdata/contracts/`; document the inventory/profile distinction
in `internal/platformclient/contract.go` and the qualification identity model.
Use one current contract interpretation, with no previous-state fallback. Keep
the full inventory provisional until its own publication requirements are met.

Add focused conformance cases in `internal/platformclient/` for the table above,
using independently pinned vectors. Adapt `internal/testutil/mockplatform/` only
where the agreed current semantics differ; keep other fixture-only families
explicitly experimental. Check unsupported families, bad scope/version/profile,
denied/incomplete observations and bounded 429/503 handling. GET/plan/refresh
must remain observational and denied reads must preserve Terraform state.

Platform follow-up: reproduce the recorded concurrency cleanup failure with its
existing `pnpm test:provider:installed` lane. Determine whether the server or test
harness violates the owned contract. Any recovery from an uncertain mutation must
inspect the original key/operation and reuse its intent, never allocate a fresh
generation just to make cleanup pass. Changes belong in Platform after reading
its instructions, with a regression case and a new committed run.

**Exit:** a reviewed field-level mapping, passing pinned-vector/client tests and
a resolved or precisely isolated Platform failure. Provider contract work may
proceed while the failure is investigated; qualification may not borrow an older
passing result.

### 2. Qualify an exact installed bucket consumer (TF10)

Delivered for the exact artifact in the [bucket adoption record](../../qualification/platform-adoption.md).
New artifacts/fixtures require fresh evidence; prior passes do not transfer.

Reuse Platform's `scripts/installed-provider.mjs`, `tests/http-provider-consumer.py`
and retained HTTP fixture. They already exercise a normal filesystem-mirror
installation; avoid another native adapter or a second management harness.
Select a committed provider candidate, update Platform's provider revision,
version, schema and binary digest together, and execute with no compiler in
Terraform's PATH, no development override and readonly checksums.

Reproduce CRUD, import without ownership transfer, drift/rename, empty plans,
protection, scope/owner denial, revision/generation conflicts and token rotation.
Retain the four executed interruption schedules: before admission forwarding,
after acceptance with delivery stopped, after commit before reply, and terminal
expiry before recovery. Assert original resource/operation IDs, unchanged state
on expiry, and no replacement. Distinguish provider death from server restart;
retain separately owned declarations and synthetic credential-canary checks.

Where the provider's recovery contract requires additional CRUD death windows,
add them explicitly. The four Platform schedules do not inherit the mock suite's
108 cases or the historical native client's twelve schedules.

**Exit:** a clean committed Platform report and workflow run bound to the exact
provider bytes, both schema digests, profile, CLI/system and executed cases.
No change to TF10 completion or public-release status follows from metadata alone.

### 3. Enable the provider's explicit metadata lane (TF10/TF11)

Delivered in the [HTTP guide](../platform-http.md#explicit-qualification-configuration).
The requirements below continue to separate metadata from full release evidence.

Replace blocked-only behavior for the selected profile in
`internal/testutil/liveplatform/` and `internal/provider/live_test.go` with actual
installed HTTP cases. Extend the existing `just test-acc live` metadata selection
and `.github/workflows/live.yml`; keep the five public Just tasks. Require an
explicit isolated endpoint, CA, credential file, scope, exact artifact and bounded
cleanup. Operator setup stays in the harness and never in provider lifecycle code.

Reports must distinguish local Transit-backed metadata, hosted management and
real data planes. Bind evidence to the profile, inventory/schema, Platform and
provider revisions and exact executable. Reconcile report/release validators
together so a partial development-profile pass cannot satisfy the full release
gate. Missing configuration, artifacts, required cases or upstream prerequisites
must fail; never select this lane from ambient credentials or fall back to mocks.

**Exit:** the explicit metadata lane really executes and reports its bounded
scope; negative tests reject partial, skipped, mismatched and provisional evidence
for publication. Ordinary mock CI stays credential-free.

### 4. Adopt service APIs in dependency order (TF10, then TF12–17)

Use the current Platform owners below. For each available slice: agree the public
schema and field ownership, adapt client/fixtures/Terraform together, qualify the
installed lifecycle and recovery, then test the real data plane independently.
Regenerate examples/reference docs only after actual schema changes.

| Provider work | Current Platform owner | Required next service proof |
| --- | --- | --- |
| TF4 scope/access, consumed by TF10 | P5 customer scope and access | Organization-only bootstrap, service-account/grant lifecycle and external credential issuance without configuration cycles |
| TF6 storage, consumed by TF10 | P6 managed databases and object storage | Real object upload/download and database read/write, revocation, quota/retention and protected teardown |
| TF5 registries, consumed by TF10 | P11 artifact registries and isolated builds | OCI push/pull and immutable digest discovery; hosted builds are an independent later slice |
| TF7 bindings/secrets and TF8 containers/endpoints | P12 workload isolation, bindings and endpoints | Consumer-ID bindings, explicit grants, real rollout/TLS and access, retained data after compute retirement |
| TF10 complete initial graph; TF11 release | P15 initial service graph qualification; P3 for hosted placement | Installed graph plus independent service probes, retained recovery and exact evidence; no automatic expansion to every new P15 product |
| TF12 jobs/schedules; TF13 streams/queues | P22 additional managed services, using P12/P13 execution where applicable | Real executions or message settlement/retention, authorization and original-operation recovery |
| TF14 volumes/attachments | P22 plus P20 durability | Written bytes survive replacement and stale writers are excluded |
| TF15 domains/private connectivity | P22 with P3/P12; P7 separately qualifies static HTTPS domains | Domain proof/TLS and independent private reachability, peer consent and revocation; static domains do not qualify all networking |
| TF16 backups/clones | P20 durability, recovery and upgrades | Whole-service recovery provenance, independently writable clone and retained data/access correctness |
| TF17 scaling/placement | P19 elastic capacity and P21 cells/placement | Real bounded transitions and field ownership; storage fencing/continuity for stateful movement |

These names/paths follow [Platform's alignment map][alignment] at the reviewed
revision. Historical P10/P11/P13/P17–20 references in earlier provider plans must
not be used as current owners. Independent families may advance as their actual
prerequisites land; neither general build infrastructure nor native owner
replacement is a blanket prerequisite.

### 5. Review new static-publication and model scope

Platform P7 and provider [milestone 18](18-static-publication.md) are **In Progress**:
the API candidate and provider declaration client are locally qualified. The
Terraform adapter and actual static bytes/domains remain open. Platform P14 and
provider [milestone 19](19-model-resources.md) remain **Planned**, with no model API.
These declarations do not expand TF10's initial graph or the current registered
surface. Keep the original 1–17 compatibility gate and completion evidence.

| Candidate | Proposed Terraform responsibility | API dependency and acceptance before support |
| --- | --- | --- |
| Static applications/assets, [P7 proposal][static-plan] / #29 | Application, binding, target policy and later domains plus bounded discovery; runtime heads/promotions stay independent of declaration refresh | Client foundation delivered. Implement Terraform per-intent recovery/ownership, then qualify exact installed import/drift/deletion. P5 customer grants, P6 immutable bytes, preview audience and domains remain separate prerequisites. Qualify HTML/JS/Wasm delivery, cache isolation and rollback without breaking an open page. |
| Models/connections, [P14 proposal][model-plan] / #26 | Model definitions and governed connection references/bindings; gateway routing is implicit in Platform | Agreed management schema, explicit invocation grants, bounded policies, external credentials and original-operation recovery. Qualify lifecycle/import/drift, revocation and protected retirement; real inference evidence belongs to independently selected adapter/account/protocol cases. |

Build/upload/seal operations, bulk files, prompts, transcripts, invocations,
individual job runs and Saga world state remain outside Terraform state.
Model/provider secrets stay server-side; Terraform references external identities.
Bindings do not create runtime grants, and tool output cannot grant execution
authority. Prebuilt static delivery can proceed before hosted Nix builds; a
finite model slice can proceed before general compute or every inference adapter.

Platform P15 now includes static and model profiles. The current provider registers
neither surface. The authorized implementation keeps today's TF10 graph fixed and
tracks the extensions in separate milestones. Do not invent resource names or
reuse buckets/OCI registries as stand-ins for these services.

### 6. Qualify distribution for the advertised surface (TF11)

Once the selected services pass, retain full required-case enforcement and
immutable exact-mirror release bytes. Update the independent consumer's flake,
exact provider constraint and both locks together, then execute a copy outside
the checkout. Verify clean-host installation, selected CLI/system combinations,
checksums and anonymous public downloads before claiming release completion.

The present artifact registers all later families, so bucket metadata alone
cannot release it. A narrower first release requires a separate decision and
matching registered-surface/required-case changes. Keep Linux amd64 as the only
packaged target until other systems actually execute qualification.

## Review units and validation

Steps 1–3 delivered contract conformance, installed recovery evidence and executable
metadata/report integration. Platform failure/harness changes stay in their own
repository. Step 4 is one contract and qualification change per family; step 5
has delivered the publication client foundation, with the Terraform adapter and
independent bytes/domain qualification still to follow. Model scope remains planned. Step 6
depends on the advertised surface's complete evidence. Use tracker #1 for provider
execution follow-up and the owning Platform trackers for upstream prerequisites.

Code changes require `nix develop --command just fmt`, `just check` and
`just build` in the locked shell, focused changed-behavior tests and relevant
installed/live cases. Nix changes also require all-system evaluation without lock
updates. Execute the exact installed artifact for aliases and process death.
Update generated docs through `scripts/generate-docs.sh` after schema/example
changes; never generate over the roadmap.

For documentation changes, validate relative links/fragments, milestone
numbering/status/dependencies, badge workflow paths/filters and `git diff --check`.
Runtime, installed and upstream qualification results belong in the execution
record, attributed to exact source and artifacts.

## Ongoing review

Follow [AGENTS.md](../../AGENTS.md#continually-review-platform-progress): review at
the start of work, at least every 24 hours while active, and before contract or
release changes. Update this baseline and affected milestones with exact source,
pins, run links, failures/skips, obsolete blockers and the next acceptance proof.
Preserve past evidence rather than relabeling it for the newest revision.

README badges expose the existing upstream workflows continuously as they run.
They do not schedule a semantic review, update dependencies or authorize
publication. Investigate a failed run and inspect a successful adoption report's
blocked gates before changing capability status.

[http-contract]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/http/README.md
[profile]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/http/management-profile.md
[provider-pin]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/http/provider.json
[http-evidence]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/docs/qualification/http-api.md
[native-evidence]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/docs/qualification/native-transit-adoption.md
[alignment]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/docs/roadmap/provider-alignment.md
[static-plan]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/docs/roadmap/static-publication-implementation.md
[model-plan]: https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/docs/roadmap/model-gateway-implementation.md
