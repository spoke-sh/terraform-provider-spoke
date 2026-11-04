# Implementation plan: reproducible publication adoption

[Roadmap](README.md) · [Milestone 18](18-static-publication.md) · **Status:** In Progress

Make the merged publication declarations reproducible in upstream and customer
CI. Fix Platform's original-operation qualification, establish Saga's existing
consumer run in GitHub Actions, and reconcile the handoff with executed evidence.
This remains integration follow-up alongside the implemented
[governor client slice](ci-governor-implementation-plan.md#next-slice-for-review-governor-contract-and-http-client).
Native S3 and capacity qualification
then unlock the next service slice: immutable uploads and sealing.

## Current follow-up, 2026-11-04 02:55 UTC

The [new review](platform-adoption-implementation-plan.md#governor-submission-review-2026-11-04-0255-utc)
records all 17 publication API proof entries passing at current `33df0ac`;
its overall integration run remains pending. The earlier failure below remains unexplained; it is preserved
as historical evidence, not the latest completed API result. No consumer or
fixture pin changes. Saga's current Actions source acquisition still fails and
its repository secret metadata remains empty.

## Previous follow-up, 2026-11-03 23:43 UTC

Provider #18 and Platform #51/#53 have merged. The later main `8987581`
publication API job fails with `AssertionError`; its downloaded artifact is
incomplete, and detailed logs are not yet available while the overall run is
executing. The [current review](platform-adoption-implementation-plan.md#next-slice-review-2026-11-03-2343-utc)
records the run, report hash and limits. Obtain the failing assertion and a
focused reproducer before further fixture adoption; the completed history-proof
change and prior 42-case local pass do not explain this failure.

Saga's existing Actions consumer still needs authorized private-source access;
secret metadata remains empty. Preserve its provider/fixture pins until that
job reproduces. This supersedes the earlier next action to merge #53, while
retaining every historical execution below. No live or hosted gate advances.

## Implementation follow-up, 2026-11-03

Implementation is authorized. Refreshing upstream found Platform #49 had already
added original-key polling, so the initial immediate-receipt fix was superseded.
Main `b84815c` still failed when `expire-pending` was never accepted: polling
returned `resource_not_found` until its deadline. Platform
[PR #51](https://github.com/spoke-sh/platform/pull/51) completes bounded exact
replay after authoritative absence, validates scope/owner/receipt identity and
adds deterministic regressions. Its clean `9cf2b705149a1794bc25fc5e93935c395b32ab00`
passed 17 native HTTP groups and 14 focused tests; normal project checks passed.
The handoff now records merged provider #17 and Saga #9 without changing pins.

Platform #51 merged at `d73930d3a2a81dda259628085f42b673bdc13d47` while its
Actions run was still executing. That run later failed the history proof's
mailbox comparison after six groups. [PR #53](https://github.com/spoke-sh/platform/pull/53)
uses the caller's signed command sequence to prove reads are observational,
with actual mutations as positive controls. Its clean `9e4e377` passed all 17
native groups in 86.820 seconds and normal project checks. The two corrections
change qualification only; neither adopts a new runtime or provider artifact.
The latest main/push publication API artifact at clean `9f1a1df` also passes all
17 groups at 20:39:14 UTC. The [current review](platform-adoption-implementation-plan.md#qualification-follow-up-2026-11-03-2037-utc)
records its exact hash and the still-pending overall Check.

The provider qualification loader now resolves Platform's sibling test helpers
from the selected checkout. This is needed by the newer fixture's native-worker
and publication-recovery imports. Both client and Terraform qualification use
that loader; an isolated subprocess regression rejects a caller-directory helper.
The provider executable and consumer artifact remain unchanged.

The provider's locked-shell formatting, full checks and build passed, including
1,368.614 seconds of mock acceptance. The new
[adoption execution record](../../qualification/publication-adoption.md)
separately preserves two failed installed attempts and a focused diagnostic.
A subsequent serial run passed all 42 required cases at clean harness `8d8427a`
and fixture `9e4e377` at 20:35:03 UTC, with the same `bc186093` provider bytes and
unchanged deadlines. This qualifies that local tuple without explaining the
earlier timeouts. Saga remains pinned to `65f8284` until its existing Actions
job reproduces and any fixture cutover receives its own complete evidence.

Saga source access remains an external dependency: repository secret metadata
was empty at the 19:57 UTC review, and no source credential was supplied. The
existing Actions job remains failing rather than skipped or weakened. Its
optional fixture cutover is deferred until the existing pinned job reproduces
successfully. Local consumer evidence is preserved; no hosted claim follows.

## Reviewed baseline

The [2026-11-03 upstream review](platform-adoption-implementation-plan.md#post-merge-adoption-review-2026-11-03-1840-utc)
records exact local/remote revisions, issue reviews, contracts and workflow
artifacts. The relevant changes are:

- Provider [PR #17](https://github.com/spoke-sh/terraform-provider-spoke/pull/17)
  merged at `831cea4c479ae2da0da06fe50f95ec975e9afeef`. The previous
  [declaration plan](publication-terraform-implementation-plan.md) is delivered:
  three resources, six data sources, retained intent, an independent consumer
  and the explicit 42-case [declaration gate](../../qualification/publication-declarations.md).
- Saga [PR #9](https://github.com/spoke-sh/saga/pull/9) already delivers its own
  project/nightly/production consumer. Its
  [32-case local record](https://github.com/spoke-sh/saga/blob/a34eaffedf75b5392a4b74002e3856bdea1fc950/docs/evidence/milestone-4-publication-consumer.json)
  includes 24 provider-process deaths and separate retained-service restarts.
  Both projects select provider `bc1860930924ee5c0a88004273eb85bedf54cfe2`, binary
  SHA-256 `ad5897b486805411860785fc4bdf2689ab1f15e1d4bc1bf409c8f170d51646ea`,
  Platform `65f8284d0197ccc0ed450dc8c3e0b074947979cb` and Transit
  `94fa12ff6d8cd3e409ac6524f721a17d1a245abb`. There is no missing Saga consumer
  implementation or need to select a new provider executable for this slice.
- Platform's latest main [Check failed](https://github.com/spoke-sh/platform/actions/runs/41681221178).
  After four passing publication HTTP groups, `Fixture.mutate` asserted an
  immediate receipt ID for `unprotect`; the API returned HTTP 503
  `outcome_unknown` with the original key and lookup. The documented API permits
  this response. Recovery must determine the original result before concluding
  whether the service itself failed. Earlier runs failed at other mutations with
  the same fixture assumption; PR #47's distinct race fixes are already merged.
- Saga's latest main [Check failed](https://github.com/spoke-sh/saga/actions/runs/41587288341)
  before qualification: `PLATFORM_SOURCE_TOKEN` was unavailable to the source
  acquisition step. The publication lifecycle step and candidate job were
  skipped. Local evidence remains valid at its recorded sources; CI reproduction
  is still open.

## 1. Recover the original operation in Platform qualification

**Owner:** `spoke-sh/platform`; first implementation PR.

Read Platform's current instructions before editing. Change
`tests/publication-http.integration.py` and focused helper tests through the
existing test commands:

1. Handle successful receipts and the documented uncertain response explicitly.
   Preserve method, scope, principal, operation key, body and revision fence.
   Validate the original-key lookup against that scope and key before following
   it. Require an actual valid operation before entering settlement polling.
2. Resolve uncertainty through authorized original-key discovery. If acceptance
   is not yet discoverable, only the contract's exact same-request replay is
   eligible; bound the entire recovery by a monotonic deadline. Preserve receipt
   and resource IDs. An incomplete read or temporary absence cannot authorize a
   different mutation.
3. Keep denial, conflict, expired receipts, malformed responses and exhausted
   deadlines visible. Preserve existing expected terminal outcomes, including
   intentional conflict/capacity cases. Do not turn refusal into retry or replace
   this failure with a larger unconditional sleep.
4. Add deterministic cases for a lost/uncertain acceptance response, delayed
   original-key visibility, exact replay, refusal and timeout. Exercise the real
   normal API with existing fault controls as well as helper response cases.
   Assert one logical operation and no fresh key or duplicate resource.

**Exit evidence:** focused tests and Platform's required checks pass; the full
`pnpm test:publication:api` report completes every required group on a clean
commit. Attach sanitized failure details if an original operation remains
unresolved. A green rerun alone, with no regression for the observed failure,
does not finish this task. Confirm the merged-main publication job and inspect
its exact artifact. Preserve synthetic peer results as separate evidence.

## 2. Reproduce the existing Saga consumer in Actions

**Owner:** `spoke-sh/saga` and its repository administrator; independent of task 1.

The workflow already fails explicitly when private Platform source is unavailable.
Arrange repository-scoped, read-only source access for the existing
`PLATFORM_SOURCE_TOKEN` input, using an authorized credential. Do not copy a
developer's ambient GitHub credentials, commit a token, forward source access to
the fixture, or weaken the publication/candidate dependency. This administrative
input is currently unavailable to the workflow; code and local checks can proceed
without claiming the Actions gate passed.

Run the existing `publication` job with its committed dependency tuple first.
Require all 32 assertions/10 groups, readonly provider locks, separate state and
credentials, installed binary/schema hashes, and all 24 provider-death schedules.
Confirm a sanitized report is actually uploaded and the dependent candidate job
runs only after all required checks pass. If further runner failures appear after
source acquisition, address their evidence in a small Saga follow-up PR.

**Exit evidence:** a successful main/push run and downloaded declaration artifact
at the exact Saga commit, plus candidate provenance. A static candidate artifact
is not a published website. Source access alone is not qualification.

## 3. Record current adoption and qualify a new fixture only when selected

**Owners:** Platform and provider; documentation/evidence PRs after task 1.

- Update Platform's provider alignment, static handoff and HTTP overview to
  distinguish its still-qualified bucket artifact `67682f0` from the publication
  artifact `bc186093`. Replace stale statements that provider main has no static
  resources or that Saga lacks an installed consumer. Link the exact provider and
  Saga records; preserve older client and bucket qualification.
- Keep the bucket pin and publication schema/source pins unless an explicit
  adoption requires changing them. A merge or newer source revision is not a
  reason to rebuild the provider or rewrite an old qualification record.
- To select the corrected Platform fixture for consumers, first qualify the
  existing `bc186093` artifact against that exact clean candidate using the
  provider's 42-case `publication-declarations` gate. Record a new result alongside
  the `37c5265` harness / `65f8284` fixture record. Recheck schema and copied-source
  identity, complete history, original-operation recovery and compiler-free
  installation. The current API/schema has no executable delta from `65f8284`.
- A subsequent Saga fixture cutover must update its Platform flake input, lock,
  integration manifest and runtime binary hashes together, run all 32 cases,
  and pass Actions at that selected tuple. Keep provider constraints/checksums
  unchanged when the provider bytes are unchanged. Do not combine this with a
  Transit adoption.
- Reconcile implementation tracking with these records when issue updates are
  authorized, re-reading each issue first. Provider
  [#1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1) retains its
  original history; [#13](https://github.com/spoke-sh/terraform-provider-spoke/issues/13)
  has local declaration acceptance but remains open for the Saga delivery journey.
  Platform [#20](https://github.com/spoke-sh/platform/issues/20),
  [#24](https://github.com/spoke-sh/platform/issues/24),
  [#29](https://github.com/spoke-sh/platform/issues/29) and
  [#36](https://github.com/spoke-sh/platform/issues/36) need current consumer and
  CI evidence, with storage/hosting prerequisites still explicit.

**Exit evidence:** each adopted tuple has an attributable passing record and
current main CI result; docs and trackers distinguish implemented declarations,
local qualification, CI reproduction and delivered bytes. TF18 stays In Progress,
TF19 stays Planned, and release gates retain their existing required service cases.

## Review units and validation

| Unit | Logical commits or action | Required proof |
| --- | --- | --- |
| Platform recovery PR | Recover original publication acceptance; add deterministic regressions; record exact execution | Focused helper tests, required Platform checks, complete publication API qualification and merged-main artifact |
| Saga source access | Configure the existing read-only source credential; code PR only for an observed runner defect | Existing pinned consumer's full Actions job, retained report and gated candidate |
| Platform handoff PR | Refresh current provider/customer capability and evidence links | Contract/source comparison and documentation checks; installed bucket regression if its artifact is deliberately changed |
| Provider evidence PR | Qualify the selected corrected fixture; append evidence and adoption status | All 42 explicit declaration cases; required provider checks for any executable changes |
| Optional Saga fixture PR | Move only the fixture/runtime tuple after provider qualification | Complete lock/hash update, all 32 local cases and matching Actions evidence |

These units can land independently where their prerequisites permit. Task 2's
administrative dependency does not block Platform's recovery fix. For provider
code changes run locked-shell `just fmt`, `just check` and `just build`; Nix changes
also require four-system evaluation. For documentation-only changes validate
links/fragments, milestone statuses/dependencies, badges and whitespace.

## Following service slice and newly available downstream work

The critical service dependency is still
[Transit's real S3 connector #121](https://github.com/spoke-sh/transit/issues/121)
and Platform's [native connector/capacity contract](https://github.com/spoke-sh/platform/blob/72becc5915bf77fcd57f0ff1a2e4d870ef89054e/contracts/native-object-connector-gap.md).
Implement and qualify version-aware signed requests, protected temporary
credentials, actual network operations, original-effect recovery and bounded
retained storage before Platform's
[upload/sealing packages 2–8](https://github.com/spoke-sh/platform/blob/72becc5915bf77fcd57f0ff1a2e4d870ef89054e/docs/roadmap/publication-object-sealing-implementation.md).
Recent Transit query/retention work may help capacity; merged source or an open
PR does not supply the missing S3 contract. Platform owns the upload API; the
provider remains its HTTP declaration client. Live AWS qualification later needs
an explicitly selected isolated account/bucket/role and bounded test budget.

Dale can now plan its own installed declaration adoption from the same exact
provider artifact and Saga's consumer pattern. Its reviewed record is still
`installed-schema-only` with output-only roots; that is a useful independent
follow-up, not a prerequisite for Saga or storage. It needs its own lifecycle,
scope-denial and recovery evidence before claiming adoption.

Platform's saved-observation/computation proofs and deployment-cost plan add no
model HTTP API or static byte delivery. [Model tracker #26](https://github.com/spoke-sh/platform/issues/26)
and [TF19](19-model-resources.md) retain their existing prerequisites. Uploads,
sealing, previews, promotion/rollback, domains/TLS, hosted state and full provider
release remain separate gates; this integration slice adds no new resource.
