# Milestones 10–17: provider delivery and remaining live gates

> **Planning correction, 2026-11-01:** the
> [Platform HTTP plan](../docs/roadmap/milestones-10-17-implementation-plan.md)
> supersedes this record's native SDK/ABI, signer, principal-sequence journal
> and native-runtime release prerequisites. Platform owns Transit integration;
> this provider remains a Go HTTP client. Results below are preserved for their
> original revisions and scopes. Existing native-only live/release checks still
> need implementation changes; this note supplies no new live evidence or status.

Execution began 2026-10-31 and continued 2026-11-01, against the reviewed
[implementation plan](../docs/roadmap/milestones-10-17-implementation-plan.md).
**Milestones 10–17 remain In Progress.** Provider-owned codecs and provisional
service surfaces are delivered. Neither mock results nor blocked preflight are
real-service qualification or permission to publish a release.

## Revisions and delivered scope

| Component | Exact source / result |
| --- | --- |
| Provider baseline | `29230f8015adddb6c2010993aa57f4d3313d32b2` (merged PR 8) |
| Reviewed plan | `fce10f1` |
| Native-v2 codec cutover | `9a696d9`; pending-create/expired-bucket decoder correction `ae99879` |
| Mock-v5 implementation | `67f5aca`: 23 resources, 47 data sources; Terraform resource schema 3 |
| Installed recovery and live gates | `6091f416f1763507f09ff6e7d3ea10e25378cef2` |
| Independent consumer / fixture review | `b18db52` pins the artifact; `d07ae31` binds fixture peer consent to an exact scoped connection |
| Transit reviewed origin/main | `a000b8d18449fdee7901955d0c94417ceeab32eb` |
| Platform origin/main | `7af27c2bb8083ab64348592f82247da1ff703596` |
| Platform's Transit build input | `0e2fba44727ca858bd1e8e4a775b62fbcf1a665d` |
| Native application profile | `spoke.management.native.v2`, unregistered codec only |
| Executed runtime tuple | Go 1.26.7, Terraform 1.15.3, protocol 6, Linux amd64, mock-v5 |

Native codecs consume the exact Platform schema/vectors, capacity/retention,
terminal failure, expiry and bounded object/pending observations. Decoder tests
cover pending creates with no settled attributes and bucket inspection after
receipt expiry. Scope/principal/selector binding, canonical receipt intent,
unknown/duplicate members, completeness and traversal budgets fail closed. No
native signer, transport or unused journal abstraction was added.

The [mock-v5 contract](../testdata/contracts/lifecycles.md) delivers typed job and
schedule, stream and queue, volume and attachment, domain and private connection,
backup policy and database clone, scaling and placement declarations. All share
current-contract ownership, revision fences, asynchronous receipts, protection,
canonical import, exact/complete discovery and recovery. Bindings distinguish
job/container consumers and messaging intents. Histories expose bounded metadata
without execution outputs or backup contents. Schemas/examples/reference were
regenerated through the repository script.

## Executed checks

The provider runs below select committed source `6091f41` and its exact Nix
artifact. The subsequent fixture-only consent correction in `d07ae31` passed the
full fixture race suite, targeted isolation/revocation tests and vet; it does not
change provider executable sources. Documentation and consumer-pin commits are
recorded separately above.

| Check | Result and boundary |
| --- | --- |
| `just fmt`, `just build`, `go vet ./...` | Passed; formatting and local Go build, no deployment effects |
| `just check` | Passed: formatting, workflow lint, five Python release tests, Nix install check, vet, race unit tests and all 37 top-level mock acceptance tests (972.999 seconds for acceptance) |
| `nix flake check --all-systems --no-build --no-update-lock-file` | Passed evaluation on four systems; no non-Linux execution claim |
| `just test-acc installed` | Passed all 37 top-level mock tests in 936.957 seconds with the exact Nix mirror; separate-process cases have no compiler fallback |
| Installed crash matrix | 108 cases: 18 typed service kinds × create/update/delete × after acceptance/after completion, with killed provider process and reloaded fixture ledger |
| Independent copied consumer | Passed outside the checkout: readonly checksums, ordinary Random provider, CRUD/import/empty plans and cleanup, no compiler on PATH |
| Extracted release ZIP | Exact-byte verification and installed graph/checksum/version/platform cases passed; 11.105 seconds |
| Explicit native preflight | Expected nonzero result; all 16 cases blocked, no service requests or credentials read |
| Roadmap/docs checks | Relative links/fragments, contiguous 1–17 numbering, index/header statuses and `git diff --check` passed; five public Just tasks retained |

All twelve added resource kinds passed Terraform create, lost reply, update,
import, empty plan, drift, denial, authoritative absence and cleanup. Additional
validation/protection and controller-observation cases prove reads do not change
policy declarations. Schema generation preserved the hand-maintained roadmap.

Fixture tests cover fake-clock admission/restart, DST gap/fold, known-failure
retry, unknown/timeout retention, frozen admitted retry policy, disabled admission,
atomic budget refusal, history authorization/completeness, retained messaging,
explicit runtime grants, detach/replacement fence rejection, retained volume data,
external proof expiry and peer revocation, frozen clone provenance/independent
writes, backup retention, stale telemetry, quota/cooldown and pending placement
reservations. They operate synthetic state and do not run real customer services.

## Exact candidate artifact

- Committed artifact: `6091f416f1763507f09ff6e7d3ea10e25378cef2`.
- Version: `0.1.0-dev.g6091f416f1763507f09ff6e7d3ea10e25378cef2`.
- Binary SHA-256: `8226eb6a483ecd38eebfb684cec1a445e824f8ab037529cfc7dfd7285826f243`.
- ZIP SHA-256: `8bae9460fc5ba2dffc01d4399ddbc6061bbae2cc47456cb67db052a2719e6fdc`.
- The release candidate ZIP reuses exact Nix mirror bytes with fixed archive
  timestamps and executable mode; it is neither signed nor published.
- The independent consumer deliberately pins this commit and both locks; later
  documentation commits do not change that immutable qualification subject.
- No cross-platform binary or native SDK closure is qualified by shell evaluation.

## Historical completion criteria under the native plan

Use the linked HTTP plan for current remaining work. This table records the
requirements assessed during the original execution, including the now-superseded
native SDK/provider prerequisites.

| Milestone | Delivered here | Originally assessed completion work |
| --- | --- | --- |
| 10 | Native-v2 application codecs, exact schema/vectors and strict discovery; explicit prerequisite report | Shared Go SDK Serving preparation/recovery/result, subject proofs and verified custom CA support; matched ABI/library/Platform pin; durable original-work journal and principal sequencing; installed real metadata and complete initial service graph |
| 11 | Immutable mock artifact, expanded installed recovery, exact-byte candidate and fail-closed publication inventory | Portable native runtime closure and advertised system qualification; full real service evidence; signing/public Registry publication and anonymous locked installation |
| 12 | Job/schedule adapters, synthetic scheduler and execution metadata | Real execution, current runtime authority, scheduling/admission, retirement and killed-process recovery against Platform P20/P11 |
| 13 | Stream/queue metadata, bindings, explicit permissions and retention restrictions | Platform management plus actual produce/read/claim/settle, retention and revocation; selected Transit messaging profile. Atomic dead-letter/mixed transactions remain unsupported |
| 14 | Volume/attachment metadata, synthetic retained data and fencing controls | Durable volume service, real mounts/replacement, exclusion of stale writers, detach uncertainty and durability/failure-domain evidence |
| 15 | Domain/connection metadata, proof/TLS/consent observations | Real DNS ownership, certificate lifecycle, private routing/traffic and revocation; hosted placement evidence remains separate |
| 16 | Backup policy, completed-point discovery and exact same-scope synthetic clone | Whole-database consistent cuts, real backup/restore, writable isolated clones and cross-scope policy where advertised; a single read-only Transit Table branch is insufficient |
| 17 | Scaling/placement policy ownership, freshness/quota/cooldown and volume constraints | Real demand/admission/drain, residency/placement and restart; physical movement, data continuity and distributed fencing where advertised |

The SDK ABI at latest Transit (`04cd820c…`) differs from Platform's pin
(`d8196134…`). Upgrade those owners coherently; swapping only this provider's
library or copying the unpublished direct-Go signer does not meet the plan.
Platform still advertises only bucket metadata, with no usable object bytes,
SQL rows, OCI runtime or the later customer service APIs.

The live case manifest now includes all six later families. A partial family or
initial-graph-only result cannot qualify this expanded artifact. The release gate
also rejects a mock artifact even if supplied an otherwise passing native report.
No credentials are read, service requests dispatched or fixture fallback selected
by the blocked preflight. The explicit run returned nonzero and saved
[all sixteen blocked cases](milestones-10-17-live-blocked.json), using the exact
artifact above and a deliberately unconfigured isolated endpoint. This establishes
prerequisite refusal only; no actual placement was tested. No milestone was marked
Completed.

## Failures retained during development

The first mock-v5 test run rejected the old version-separated fingerprint vector;
it was independently recalculated from the literal canonical JSON. Initial new
fixture tests exposed a missing history schema reference and grant operations
using environment rather than organization scope; both were corrected before the
passing focused run. The first Nix build rejected the previous vendor hash after
adding the pinned cron parser; the expected fixed-output hash was updated from
the actual dependency build. Deliberately changing the independent provider pin
required regenerating its Spoke checksum entry; readonly verification remains
mandatory for normal consumers. Final fixture review also found that peer consent
was keyed only by remote network; `d07ae31` scopes it to the exact local connection,
with a race regression proving a second relationship cannot inherit approval.
This changes only the synthetic fixture, not the pinned provider executable.
