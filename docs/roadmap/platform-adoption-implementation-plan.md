# Implementation plan: adopt current Platform capabilities

[Roadmap](README.md) · **Status:** In Progress

Use Platform's implemented HTTP bucket profile to establish a reproducible
provider integration, then qualify real services as their APIs arrive. The bucket
profile and explicit metadata lane are implemented. Static publication now has
a locally qualified declaration HTTP client, Terraform application/reservation/target
resources and an independent consumer gate. Delivered bytes remain separate. Model resources remain planned.
Implementation was authorized on 2026-11-02. The
[execution record](../../qualification/platform-adoption.md) records delivered
profile/client work, executable metadata qualification and remaining service gates.

This plan supersedes the upstream assessment, Platform numbering and immediate
ordering in the [10–17 plan](milestones-10-17-implementation-plan.md). Its resource
acceptance criteria remain in force. Provider milestones 1–9 retain their recorded
completion scope; 10–17 remain In Progress. Platform's broader roadmap does not
silently add resources or change the provider's release requirements.

## Qualification follow-up: 2026-11-03 20:37 UTC

The [new installed declaration record](../../qualification/publication-adoption.md)
passes all 42 required cases at clean provider harness
`8d8427a43b219b95bbc4a115637d25d5ffebaf85` and Platform fixture
`9e4e377b7086db1788a1c6a0a63c2909db04f6a9`. It finished at 20:35:03 UTC after
944.363 seconds of fixture execution, including all 24 provider-process death
schedules and separate retained-service restarts. Provider `bc186093`, its binary
hash, all five consumer locks and the schema are unchanged. Two preceding failed
attempts and a narrower diagnostic remain recorded; the serial pass does not
establish their cause or qualify performance under competing workloads.

Platform [PR #53's publication API job](https://github.com/spoke-sh/platform/actions/runs/41711179602/job/126179323265)
also passed all 17 groups. Its downloaded report is clean tested merge
`9a90a0663a0855b2f4a4741b5c477990ad22b0b8` (main `9f1a1df` plus PR head
`a998441`), finished at 20:35:04 UTC in 139.450 seconds. Report SHA-256:
`47f94aded69775890c0304b740c62a3ae7121114b78fbf787b2c4fe1edd23c83`.
The PR's overall Check and latest main Check remain pending; a PR API pass is not
a merged-main or whole-platform qualification. The latest main native-adoption
and UX results below have both been downloaded and inspected.

Next action: land the history-proof follow-up, inspect its merged-main artifact,
and configure Saga's existing read-only source secret before reproducing its
currently pinned Actions consumer. Saga stays on fixture `65f8284`; any later
cutover still needs its own 32-case run and Actions evidence. TF18 remains In
Progress and TF19 Planned. No native/bucket pin, provider executable, hosted
service or release scope changes.

## Integration follow-up: 2026-11-03 20:22 UTC

Platform's user checkout and verified remote main are clean at
`9f1a1df33059993951384396dde103a45227ae00`. [PR #51](https://github.com/spoke-sh/platform/pull/51)
merged the bounded acceptance recovery and provider/Saga handoff at `d73930d`.
The separately merged #52 adds nightly deployment of checked revisions using
saved plans; that workflow change does not qualify a deployment or publication.
The user's checkout advanced independently and was not modified by this work.

The latest main/push [Check](https://github.com/spoke-sh/platform/actions/runs/41710072397)
and [Native adoption](https://github.com/spoke-sh/platform/actions/runs/41710072384)
are pending. Their current-head native artifacts are not yet available; the
older green feasibility result below is not substituted for them.
[UX](https://github.com/spoke-sh/platform/actions/runs/41710072493) passed with no
skipped steps. Its downloaded manifest matches clean `9f1a1df`, finished at
20:13:07 UTC and describes the built app's local synthetic workspace.
The earlier main Check `41702099117` ended canceled overall, with its publication
API job failed; no overall success is inferred from its other passing jobs.

PR #51 merged before its [publication API CI](https://github.com/spoke-sh/platform/actions/runs/41708011178/job/126168370621)
finished. That job failed after six groups on the owner-mailbox comparison during
history pagination, distinct from the earlier acceptance failure. Follow-up
[PR #53](https://github.com/spoke-sh/platform/pull/53) checks the caller's signed
command sequence while native background work runs, with real mutation controls.
Its clean `9e4e377b7086db1788a1c6a0a63c2909db04f6a9` passed all 17 native HTTP
groups and normal project checks. Its [PR Check](https://github.com/spoke-sh/platform/actions/runs/41711179602)
is pending; the local result does not establish current-main CI qualification.

Provider [PR #18](https://github.com/spoke-sh/terraform-provider-spoke/pull/18)
loads fixture helpers from the selected checkout. Formatting, full check and
build passed at clean harness `8d8427a43b219b95bbc4a115637d25d5ffebaf85`.
Its [new execution record](../../qualification/publication-adoption.md) preserves
two failed installed attempts and a passing create-recovery diagnostic; the
existing exact artifact and consumer fixture remain selected. The PR's current
[CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41708341618)
has a passing check/build job and pending installed-artifact recovery.

Saga remains blocked by absent `PLATFORM_SOURCE_TOKEN`; its repository secret
metadata is still empty. The implemented consumer does not need another feature
PR before source access can be configured. Its existing pinned Actions job must
pass before an optional fixture cutover. Native S3/capacity, hosted state,
upload/sealing and delivered bytes remain separate owned service prerequisites.

At 20:24 UTC, provider PR #18's two jobs at head `3ac191b` both passed; subsequent
documentation commits need their own current-head CI result. Transit remote main
advanced to `6db22d700db3db2b531e1bdd3da889f86568ca41` through ordered Document
storage #354; its local checkout remains clean at `5ea3b70`. The new component
qualification and coordinated definition/schema cutover do not establish direct
Document lifecycle, Station adoption or S3. Its S3 adapter, HTTP request and
exchange worker paths still have no delta from adopted `94fa12f`. No Platform
native pin or provider schema/artifact is adopted from this review.

At 20:27 UTC, the current-main native-adoption run above completed successfully
with no skipped steps. Its downloaded clean `9f1a1df` report passes six checks
against candidate Transit `24249947407a3d32a46b8a4591eb0061b301446d`, while still
blocking native management continuation and revoked-caller terminal receipt.
HTTP parity remains unrun and `adoption_ready=false`. This is a newly inspected
artifact for the current main, not reuse of the older green run.

## Implementation review: 2026-11-03 19:57 UTC

Platform's independently advanced local checkout and verified remote main are
clean at `b84815c7ea1c6151bbfcc938bbceab6c12a68673`. Merged #49 implements the
smaller AWS foundation/manual deployment workflow; #50 adds attended setup tasks.
They do not qualify AWS or expose customer services. #49 also added original-key
polling to publication qualification, superseding the earlier immediate-receipt
assertion. Publication API source/schema and adopted runtime/provider pins are
unchanged. Work for this provider handoff uses isolated worktrees; the user's
Platform checkout remains untouched.

Latest Platform main/push evidence, downloaded and inspected:

- [Check](https://github.com/spoke-sh/platform/actions/runs/41702099117) is still
  running overall, with **publication-api failed** after ten groups. Its log shows
  `expire-pending` remained absent through lookup-only recovery's deadline.
  Native publication, object storage and billing passed; the platform job is
  pending. This failure differs from the immediate-receipt assertion below.
- [Native adoption](https://github.com/spoke-sh/platform/actions/runs/41702099189)
  passed all steps and six checks. Its artifact matches `b84815c` but retains
  blocked native management continuation/revoked-caller terminal receipt,
  unexecuted HTTP parity and `adoption_ready=false`.
- [UX](https://github.com/spoke-sh/platform/actions/runs/41702099155) passed.
  Its downloaded manifest matches clean `b84815c`; the scope remains the built
  synthetic app. No browser-management qualification follows.

Provider main remains `831cea4c479ae2da0da06fe50f95ec975e9afeef`; its
[main CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41694454199)
now passed both jobs. Saga remains `a34eaffedf75b5392a4b74002e3856bdea1fc950`;
its [main run](https://github.com/spoke-sh/saga/actions/runs/41587288341) still
failed before declaration execution for missing `PLATFORM_SOURCE_TOKEN`.
Repository secret metadata was empty; an authorized source credential is required
to reproduce the existing job. Dale main remains `54e44f8`. Transit remote main
advanced to `86871748b34cc88da133ee40a2c3c88566e4b540` with native retention and
Table branch work. Its S3 adapter/finite HTTP worker/request/exchange paths still
have no delta from Platform's adopted `94fa12f`; native S3/capacity gates remain.

Platform trackers #20/#24/#26/#29/#36 were refreshed for review, preserving #20's
new AWS/load observations. The GitHub connector returned repository-access 404s;
authenticated `gh` successfully supplied current source/issues and workflow
artifacts, so this is a verified review rather than reuse of an older green run.

The [authorized implementation](publication-adoption-implementation-plan.md#implementation-follow-up-2026-11-03)
now has [Platform PR #51](https://github.com/spoke-sh/platform/pull/51), with 17
passing native HTTP groups and 14 focused tests at clean `9cf2b705`. The fix
recovers through the original key, permitting only exact-request replay after
authoritative absence. Normal project checks passed. Its
[PR Check](https://github.com/spoke-sh/platform/actions/runs/41708011178) is pending;
local execution does not stand in for PR or merged-main evidence. The provider
harness separately adds selected-checkout helper loading for this fixture.

## Post-merge adoption review: 2026-11-03 18:40 UTC

Reviewed GitHub issues and upstream source; remote revisions and workflow results
were reverified at 18:40:40 UTC. The new
[publication adoption plan](publication-adoption-implementation-plan.md) proposes
the next integration slice. This review runs no new service qualification,
changes no pins and makes no GitHub issue updates.

| Repository | Inspected local HEAD/tree | Verified remote main |
| --- | --- | --- |
| Provider | Clean `831cea4c479ae2da0da06fe50f95ec975e9afeef` before these planning edits | `831cea4c479ae2da0da06fe50f95ec975e9afeef` |
| Platform | Clean `72becc5915bf77fcd57f0ff1a2e4d870ef89054e` | `72becc5915bf77fcd57f0ff1a2e4d870ef89054e` |
| Transit | Clean `5ea3b70bb2848d25bf74c1ca3fafb3f2b0ee9770` | `0cfa0cec0983ef3c62224e897428eb42b390b64b` |
| Saga | Clean `a34eaffedf75b5392a4b74002e3856bdea1fc950` | `a34eaffedf75b5392a4b74002e3856bdea1fc950` |
| Dale | Clean `c2c383c9d9866b16957907196442e069337a659b` | `54e44f87034c96766e0cb15ef8ae99db5b39f2a3` |

The upstream checkouts were inspected without resetting or editing their trees.
Remote Transit source was read separately from its older local HEAD; Dale's
remote integration manifest was read through GitHub. Merged revisions identify
available source, not the exact artifacts qualified below.

Provider [PR #17](https://github.com/spoke-sh/terraform-provider-spoke/pull/17)
merged at 18:30:22 UTC. Both jobs in its
[PR CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41590827329)
passed; both jobs in the new
[main CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41694454199)
were still running at this review. The completed declaration plan retains its
42-case record at clean harness `37c5265`, provider artifact `bc186093` and
Platform fixture `65f8284`; the merge is not a new execution identity.

Saga [PR #9](https://github.com/spoke-sh/saga/pull/9) is also merged. Its
[local consumer record](https://github.com/spoke-sh/saga/blob/a34eaffedf75b5392a4b74002e3856bdea1fc950/docs/evidence/milestone-4-publication-consumer.json)
passed 32 assertions in 10 groups at clean Saga `0170bc2a37d86eaaada755dcc97518f07c7a8dce`,
including 24 actual provider deaths and separate native restarts. It uses the
same `bc186093` binary/hash and `65f8284` fixture as the provider's independent
consumer. Its latest
[main Actions run](https://github.com/spoke-sh/saga/actions/runs/41587288341)
failed acquiring private Platform source because `PLATFORM_SOURCE_TOKEN` was
unavailable; qualification and the dependent candidate job were skipped.
The check and station jobs passed. This retires the missing-consumer implementation
blocker while retaining CI reproduction, hosted state and byte-delivery gates.
Dale's remote manifest still records provider `68d0c53e` as installed-schema-only;
its output-only roots need their own installed lifecycle adoption.

Platform [PR #48](https://github.com/spoke-sh/platform/pull/48) adds a lower-cost
AWS deployment plan. It adds no publication API, cloud execution or customer
service qualification. Platform's publication gateway/owner source and schema
are unchanged from `65f8284`. The provider's schema matches current main at
SHA-256 `db04fc4ce0e14534c2e44cef932b197e6d9aeb48006f21f7a1523c21def0511f`.
Its copied profile remains byte-identical to pinned source `6627bed`; current
Platform adds only the upload-plan paragraph. Bucket schema/vectors also match.
Platform still qualifies provider `67682f0` for bucket metadata, with binary hash
`bd29d12d0c5eacf60bef716c64b99dde985a246c717653dabae008745c5016e8`
and Terraform schema hash
`a3100081694a125dee8779179d1a85fb3f5ff1cdd8af6f7a8dab46dd35333c0b`.
That adopted bucket pin is distinct from the qualified publication artifact; no
automatic pin update follows from this review.

Latest Platform main/push jobs and downloaded qualification artifacts at `72becc59`:

- [Check](https://github.com/spoke-sh/platform/actions/runs/41681221178) **failed**.
  The publication-api qualification step failed after four passing groups:
  `Fixture.mutate` required an immediate receipt for `unprotect`, but received
  `outcome_unknown` and its original-key lookup. It needs bounded recovery of
  that operation before its result can be assessed. The platform, object-storage,
  publication and billing jobs passed; no other job was skipped.
  Installed bucket qualification passed its 12 schedules at the unchanged
  `67682f0` artifact. The browser boundary report still says management admission
  is blocked and its mutation route returns 404.
- [Native adoption](https://github.com/spoke-sh/platform/actions/runs/41681221039)
  **passed** all steps and six checks, using candidate Transit `2424994`, not
  current Transit main or Platform's adopted `94fa12f` pin. Its artifact still blocks
  `native_management_continuation` and `revoked_caller_terminal_receipt`; HTTP
  provider parity is `not_run`, timing qualification is unrecorded, and
  `adoption_ready=false`.
- [UX](https://github.com/spoke-sh/platform/actions/runs/41681221092) **passed**
  all steps. The downloaded clean-source manifest matches `72becc59`, with 11
  accessibility audits and zero violations. Its scope is the built app with a
  local synthetic workspace, not browser management or hosted service proof.
- Check's object artifact passed nine scenario groups but still reports
  `blocked_on_native_s3_adapter_and_capacity_qualification`. AWS, upload/sealing,
  public API and full-envelope qualification are false. The executable S3 audit
  still lacks session-token/provider-version fields and actual network S3 proof.
  The native publication and billing reports also retain their unqualified
  hosted-service boundaries.

Provider issues [#1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
and [#13](https://github.com/spoke-sh/terraform-provider-spoke/issues/13), Platform
[#20](https://github.com/spoke-sh/platform/issues/20),
[#24](https://github.com/spoke-sh/platform/issues/24),
[#26](https://github.com/spoke-sh/platform/issues/26),
[#29](https://github.com/spoke-sh/platform/issues/29) and
[#36](https://github.com/spoke-sh/platform/issues/36), and Transit
[#121](https://github.com/spoke-sh/transit/issues/121) were re-read along with recent
merged/open PRs. Saga and Dale have no open issues at this review. Provider #13
and Platform's alignment/handoff still contain earlier client-only or missing
consumer observations; reconcile them with the bounded records above, preserving
their open hosted acceptance. PR #47 is merged, so it is no longer a pending
prerequisite; the immediate-receipt assumption remains a separate defect.

Transit main adds bounded native tooling, MIME/fixture work and psql inspection;
pending retention/query/branch PRs are not adopted evidence. The S3 adapter,
finite HTTP worker, HTTP Request and exchange Worker paths are unchanged between
Platform's adopted `94fa12f` and reviewed Transit main. Real version-aware S3 and
capacity qualification remain native prerequisites for Platform upload/sealing.
Saved-observation/computation work provides no model HTTP API; TF19 stays Planned.

**Next action:** repair Platform's publication qualification, reproduce Saga's
existing consumer in Actions, then record any deliberately selected fixture
adoption. The smallest provider change is a new exact qualification/adoption
record; there is no new provider resource or transport requirement. The following
service slice belongs to Transit/Platform's native S3 and capacity owners. The
three README badges still point to the current upstream workflow paths, filtered
to `branch=main` and `event=push`; their health does not close service milestones.

## Resumed qualification review: 2026-11-03 06:06 UTC

Platform's clean local checkout and verified remote main advanced independently
to `35cccf82a64462eca3d07f20824f6adfa09add6b`. Provider main remains merged
`68d0c53e`. Trackers #20/#24/#26/#29 were re-read at their latest 05:11 updates.
The publication HTTP source/schema is byte-identical to the selected `65f8284`
fixture. Its removed scratch checkout was recreated at that exact commit; the
user's Platform checkout was not modified. This upstream review does not change
the selected fixture or schema pins; the consumer separately selects the qualified
`bc186093` provider artifact.

- [PR #44](https://github.com/spoke-sh/platform/pull/44) delivers an installed
  operator computation-metadata reader, with 17 separately attributed PR-CI
  groups. [PR #46](https://github.com/spoke-sh/platform/pull/46) qualifies native
  saved-observation write/disclosure boundaries. These are useful P8/P13
  precursors, with no customer service/browser completion or new provider HTTP
  resource. A later provider integration needs a versioned, scoped HTTP surface;
  it must not import the native reader or SDK. Model API gates remain unchanged.
- [PR #47](https://github.com/spoke-sh/platform/pull/47) fixes history-quiescence
  and finite-worker QueryExpired races in Platform's tests, retaining original
  exchange identity and bounded reactivation. Its locally reported 15 HTTP,
  10 publication and nine object groups belong to `9d57c20`, not this review.
  Earlier failed workflow reports remain historical evidence.
- Latest main/push [Check](https://github.com/spoke-sh/platform/actions/runs/41587086196)
  was running at the initial review. Native feasibility was also pending then;
  its separately inspected completion is recorded below. Current
  [UX](https://github.com/spoke-sh/platform/actions/runs/41587086221) passed.
  Its downloaded manifest matches clean `35cccf82`; all 11 accessibility audits
  passed with zero violations against the built app and local synthetic workspace.
  It does not qualify browser service management. No earlier green run substitutes for these pending outcomes.

At the 06:19 UTC follow-up, [native feasibility](https://github.com/spoke-sh/platform/actions/runs/41587086209)
had passed all six checks. The downloaded artifact matches clean `35cccf82`, with
`native_management_continuation` and `revoked_caller_terminal_receipt` still
blocked, HTTP provider parity unrun, and `adoption_ready=false`. Check remained
in progress: native publication and billing had passed, publication API was
building, object verification was running, and the main integration job had not
yet reached installed bucket qualification. Provider merged-main
[CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41568491160)
passed at `68d0c53e`; its synthetic checks do not qualify hosted services.

At 06:29 UTC, the same Check run's **publication-api job failed**. Its
downloaded exact-main report completed six groups, then `Fixture.mutate` asserted
an immediate receipt ID while creating `capacity-app-0`. The API returned
`outcome_unknown` with the original key/lookup instead. The fixture must recover
that original operation without admitting another identity; Platform owns this
follow-up. PR #47's race fixes do not resolve this remaining receipt assumption.
The main integration job was still running, so no overall Check pass is claimed.

The downloaded object artifact passed nine groups at clean `35cccf82` in 193.322
seconds. It still reports `blocked_on_native_s3_adapter_and_capacity_qualification`;
AWS, upload/sealing, public API and full-envelope qualification are all false.
Its executable adapter audit has no session-token/provider-version fields and
no network S3 qualification. This is bounded native feasibility, not customer
storage or a reason to advance provider pins.

At the final remote review, 2026-11-03 06:32 UTC, clean local/remote Platform
remained `35cccf82` and provider remote main remained `68d0c53e`. These were still
the latest main/push runs; their results and pending integration step were
unchanged. No older green workflow substitutes for the failed publication API job.

The next publication service slice remains Transit/Platform native S3 and
capacity qualification, then immutable uploads/sealing. Saved-observation service
work can proceed under its separate Platform plan; it does not unlock Terraform
model resources or delivered static bytes. Provider consumer work remains bounded
to the implemented normal declaration profile.

## Consumer qualification review: 2026-11-03 03:00:55 UTC

Provider [#15](https://github.com/spoke-sh/terraform-provider-spoke/pull/15) and
[#16](https://github.com/spoke-sh/terraform-provider-spoke/pull/16) are merged;
remote main is `68d0c53e5011673900c82f04be1b359fb856ce74`. The consumer work builds
on that main. The exact installed artifact remains `bc186093`, whose
[normal graph proof](../../qualification/publication-graph.md) passed against clean
Platform `65f8284`; merge commits do not replace its executed identity.

Platform's clean local checkout and verified remote main are
`54090efc51769ef663b97b2c2675e73f5c09b5fe` (merged
[planning PR #43](https://github.com/spoke-sh/platform/pull/43)). It adds independent
computation-evidence planning for P8/P13, with no publication API/schema change.
The selected clean `65f8284` fixture and publication schema remain unchanged.
Trackers #20/#24/#26/#29 were re-read; their older provider-source observations
are superseded here by the merged provider declarations, not by hosted support.

- [Check main/push](https://github.com/spoke-sh/platform/actions/runs/41564695193)
  is running, with **publication-api failed**. Its downloaded failed report has
  four completed groups before `Fixture.mutate` asserted an immediate receipt for
  `delete-production`; the API instead returned `outcome_unknown` with the
  original lookup key. Platform owns the fixture recovery follow-up. The main
  integration/installed bucket steps were still pending or running. Object,
  native publication and billing jobs passed independently.
- [Native feasibility](https://github.com/spoke-sh/platform/actions/runs/41564695082)
  passed six checks. Its downloaded exact-main artifact still has blocked
  `native_management_continuation` and `revoked_caller_terminal_receipt`, unrun
  HTTP provider parity, and `adoption_ready=false`.
- [UX](https://github.com/spoke-sh/platform/actions/runs/41564695041) passed. Its
  downloaded manifest attributes a clean built app with local synthetic workspace
  to `54090efc`; it does not qualify browser service management.
- The successful object job remains bounded native feasibility, with the
  S3/capacity adoption gate blocked. Customer upload/sealing/delivery flags are
  false. The previous `28372acf` Check was cancelled, not a passing substitute.

Next: finish the provider consumer review with exact local evidence. Transit and
Platform own temporary-credential/versioned S3, placement/capacity and subsequent
upload/sealing APIs. After those qualify, the provider can consume the implemented
management surface while Saga verifies retained HTML/JS/Wasm bytes, audience
isolation and rollback. Model resources still require Platform's own invocation
contract; computation-inspection planning unlocks no Terraform model API.

## Binding/target review: 2026-11-03 01:39:39 UTC

Platform's independently advanced local checkout and fetched remote main are
clean at `28372acf687588c031430ac76a76727222e41cf7`, merged
[PR #42](https://github.com/spoke-sh/platform/pull/42). Its nine local object-proof
scenario groups qualify bounded native range verification and recovery at source
`56c2386659b75d84e6ca069dbd957c64cc19b9e3`; the S3/capacity adoption gate remains
blocked. No customer upload/sealing route, profile or provider pin is enabled.
The declaration API source/schema is unchanged from the selected clean `65f8284`
fixture. The only publication fixture delta adds an optional scratch parent.

The latest main/push [Check](https://github.com/spoke-sh/platform/actions/runs/41560878922)
and [native feasibility](https://github.com/spoke-sh/platform/actions/runs/41560878913)
were running at this review; [UX](https://github.com/spoke-sh/platform/actions/runs/41560878911)
passed. Check now has a separate object-storage feasibility job. Its future green
result cannot turn blocked object adoption into provider support. The earlier
`65f8284` publication-api failure remains recorded below. Follow the new native
connector requirements in Platform/Transit; continue bounded declaration work in
this provider without adopting test-only object interfaces.

## Application qualification and CI follow-up: 2026-11-03 01:18:33 UTC

The exact committed Linux amd64 provider `581cf627a777c26a27692aaa88508430669e9901`
passed [installed application qualification](../../qualification/publication-application.md)
against clean Platform `65f8284d0197ccc0ed450dc8c3e0b074947979cb`. All 15 reported
case groups passed, including nine actual provider-process kills followed by
separate native node/owner/courier/gateway restarts. The schema pin is unchanged;
bindings, targets and hosted delivery retain their subsequent gates.

The latest [Platform Check](https://github.com/spoke-sh/platform/actions/runs/41553977933)
remains in progress, but its publication-api job has **failed**. The downloaded
`publication-http-qualification-65f8284...` artifact reports failure. Its log shows
`Fixture.mutate` asserted an immediate receipt ID for `expire-pending`, while the
API returned `outcome_unknown` with the original key/lookup. This is not a green
main result or grounds to ignore upstream recovery coverage. Platform owns that
fixture follow-up; this provider's local installed declaration proof is separately
attributed. Native feasibility and UX retain the inspected results below.

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

**Next provider work:** the [TF18 resource sequence](18-static-publication.md#declaration-delivery-sequence)
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
