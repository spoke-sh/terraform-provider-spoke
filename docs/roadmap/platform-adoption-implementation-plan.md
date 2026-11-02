# Implementation plan: adopt current Platform capabilities

[Roadmap](README.md) · **Status:** Planned · Implementation proposal for review

Use Platform's implemented HTTP bucket profile to establish a reproducible
provider integration, then qualify real services as their APIs arrive. Reconcile
the contract and current failed integration run first. Static publication and
model resources are proposed additions with separate API and review gates.
This refresh changes documentation and monitoring instructions only.

This plan supersedes the upstream assessment, Platform numbering and immediate
ordering in the [10–17 plan](milestones-10-17-implementation-plan.md). Its resource
acceptance criteria remain in force. Provider milestones 1–9 retain their recorded
completion scope; 10–17 remain In Progress. Platform's broader roadmap does not
silently add resources or change the provider's release requirements.

## Reviewed baseline

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

Platform's new P7 and P14 are **Planned**. They are not implemented endpoints or
existing Terraform resources. Recommend separate provider milestones if their
declaration scope is approved; add numbered files and index/dependencies together
at that point. Keep the current 1–17 compatibility policy and completed evidence
unchanged during this review.

| Candidate | Proposed Terraform responsibility | API dependency and acceptance before support |
| --- | --- | --- |
| Static applications/assets, [P7 proposal][static-plan] / #29 | Application, target/configuration and domain declarations plus bounded discovery; agree ownership of selected revisions/promotions before defining schemas | Versioned management lifecycle, P5 grants, P6 real immutable object versions, preview audience, revision fences and recoverable promotion. Qualify import/drift, protected deletion, independent HTML/JS/Wasm delivery, cache isolation and rollback without breaking an open page. |
| Models/connections, [P14 proposal][model-plan] / #26 | Model definitions and governed connection references/bindings; gateway routing is implicit in Platform | Agreed management schema, explicit invocation grants, bounded policies, external credentials and original-operation recovery. Qualify lifecycle/import/drift, revocation and protected retirement; real inference evidence belongs to independently selected adapter/account/protocol cases. |

Build/upload/seal operations, bulk files, prompts, transcripts, invocations,
individual job runs and Saga world state remain outside Terraform state.
Model/provider secrets stay server-side; Terraform references external identities.
Bindings do not create runtime grants, and tool output cannot grant execution
authority. Prebuilt static delivery can proceed before hosted Nix builds; a
finite model slice can proceed before general compute or every inference adapter.

Platform P15 now includes static and model profiles. The current provider has
neither. Review whether to expand its initial graph or qualify those later;
**recommend keeping today's TF10 graph fixed and adding separately reviewed
milestones**. Do not invent resource names or reuse buckets/OCI registries as
stand-ins for these services.

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

Implement steps 1–3 as separate reviewable changes: contract conformance,
installed recovery evidence, then executable metadata/report integration. Platform
failure/harness changes stay in their own repository. Step 4 is one contract and
qualification change per family; step 5 first needs a scope decision; step 6
depends on the advertised surface's complete evidence. Use tracker #1 for provider
execution follow-up and the owning Platform trackers for upstream prerequisites.

Code changes require `nix develop --command just fmt`, `just check` and
`just build` in the locked shell, focused changed-behavior tests and relevant
installed/live cases. Nix changes also require all-system evaluation without lock
updates. Execute the exact installed artifact for aliases and process death.
Update generated docs through `scripts/generate-docs.sh` after schema/example
changes; never generate over the roadmap.

For this documentation-only proposal, validate relative links/fragments,
milestone numbering/status/dependencies, badge workflow paths/filters and
`git diff --check`. No runtime, installed, live or upstream qualification is
claimed by these edits.

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
