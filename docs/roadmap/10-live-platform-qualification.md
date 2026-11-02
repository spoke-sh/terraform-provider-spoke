# 10. Live platform qualification

[Roadmap](README.md) · **Status:** In Progress

Prove that the initial Terraform service graph controls real platform services,
recovers original accepted work and leaves retained data usable after compute teardown.

See the [Platform adoption plan](platform-adoption-implementation-plan.md#implementation-sequence)
for the Platform HTTP contract, stable request recovery, backend prerequisites
and explicit live lane. The provider uses Go `net/http`; Platform owns Transit
integration and internal sequencing.

## Prerequisites and scope

Contract coordination starts in [milestone 2](02-management-contracts-and-discovery.md).
Complete this milestone using the [recovery matrix](03-recovery-isolation-and-protection.md),
initial graph in [milestones 4](04-scope-and-access.md)–[8](08-container-services-and-endpoints.md)
and [installed artifact](09-nix-packages-and-consumers.md). Platform must provide
published versioned management APIs, automation auth, service controllers and
an isolated qualification environment. Provider mocks cannot satisfy those
upstream prerequisites. Qualify resources incrementally as their APIs become available.

## Current upstream boundary

Platform main `71d43c1f522657280c5c5bb1e5187cc617b13d95`, reviewed 2026-11-02 22:50 UTC,
implements `spoke.platform.http.bucket.v1`: ordinary bearer-authenticated HTTPS
JSON, an operator-seeded scope, bucket declarations, durable receipts/tombstones,
settlement authorization and bounded pending discovery. Its Platform-owned schema
and vectors narrow the provider's mock-v5 inventory. No native consumer SDK,
signer or journal is required. This is a development metadata profile, with no
usable object storage or customer scope/access lifecycle.

Platform records local installed-provider CRUD/import, drift, retained restart
and four process-death/expiry schedules against our exact `67682f0` artifact.
The newer installed report at Platform `d7af062` qualifies receipt-first cleanup
backoff; the bucket gateway and schemas are unchanged. Latest main integration
is still running at the review time. The
[reviewed evidence and next actions](platform-adoption-implementation-plan.md#progress-review-2026-11-02-2250-utc)
separate those local passes from earlier failed/cancelled CI. Native owner replacement and
browser command admission remain Platform work, not blockers for the existing
machine HTTP slice. Finite publication/billing proofs add no service routes to
this profile. Real object/database/OCI/workload services remain open.
Adoption uses hard cutovers under the [roadmap policy](README.md#development-compatibility-policy).

## Delivered foundation and remaining gates

The provider's Go HTTP client, mock-v5 lifecycle/recovery and installed-artifact
checks are delivered. Native codecs have been removed and live/release identity
checks now use the HTTP contract, schema and artifact. The
[HTTP boundary record](../../qualification/platform-http.md) tracks those changes.
Earlier native codecs and preflight are historical implementation work. The
[execution record](../../qualification/milestones-10-17.md) preserves their results;
they are not qualification of the selected HTTP architecture.

The [adoption execution record](../../qualification/platform-adoption.md) now
records pinned profile/schema/vectors, bounded pending discovery and a passing
explicit local metadata lane against an exact installed artifact. The Platform
companion fixes the cleanup harness's uncertainty handling and proves external
drift convergence. Earlier main failures retain their own attribution.

Remaining work: qualify the real graph as service APIs arrive. Hosted service
selections still emit blocked preflight evidence. The local metadata lane does
not enable them or satisfy the release gate.
The [previous boundary record](../../qualification/native-boundary.md)
remains historical; its SDK gaps are no longer provider prerequisites.

## Delivery sequence

1. Pin the owned Platform HTTP profile/schema and exact Platform revision;
   distinguish development metadata qualification from full service publication.
   Reconcile scope, automation credentials, IDs, errors, capabilities, revisions,
   acceptance/settlement, ownership, receipt retention and deletion semantics.
   Update boundary types, fixture contracts, client, Terraform schemas and docs
   together by hard cutover; record divergences without retaining legacy paths.
   Require stable logical identity before dispatch and server lookup when a
   process dies before Terraform saves state or a returned operation ID. Define
   retries versus later generations, including A → B → A and delete/recreate.
   Platform atomically binds intent, ownership and revision and persists acceptance
   before effects; the provider does not allocate native sequences or journal
   signed envelopes. Terraform private state alone cannot solve the lost-state case.
2. Implement actual HTTP live cases behind the current explicit preflight through
   `just test-acc live` and the separate configured workflow. Require endpoint/CA/tenant/credential and
   isolated fixture configuration; fail if absent or if no tests are selected.
   HTTP report/workflow consumers already exclude Transit definitions/pins and
   native journal inputs. Retain artifact and required-case checks when enabling
   actual calls. The implemented local metadata selection runs the Platform-owned
   isolated fixture; hosted service selections remain blocked pending their APIs.
   Never infer live mode from environment credentials or fall back to mocks.
3. Keep backend provisioning/setup in the harness. Use local/isolated Terraform
   state, fixture-owned identities and bounded cleanup. Do not require customer
   Terraform to create the management endpoint used to configure its provider.
4. Execute applicable contract and Terraform lifecycle scenarios on real APIs.
   Kill the provider, lose replies, restart platform/controller components and
   retry accepted CRUD. Verify authoritative backend receipts and resource IDs.
5. Exercise the real data plane independently: registry push/pull of the declared
   digest, bucket upload/download, database read/write, workload access through
   bindings, endpoint reachability/TLS and access revocation. Retire compute and
   read the separately retained data again through authorized clients.
6. Publish a capability matrix with exact provider/platform/API/artifact/CLI/system
   revisions, executed runs, failures and exclusions. Promote only qualified
   resource operations from experimental to supported documentation.

## Acceptance criteria

- The complete initial service graph passes lifecycle, import, empty-plan,
  drift, denied access, revision conflict and dependency-safe teardown against
  the real backend using the installed provider.
- Interrupted operations, including death before Terraform saves state, and
  server/controller restart preserve original intent without duplicate logical
  resources. Uncertainty remains inspectable; timeout
  or lost authority does not imply rollback or authoritative absence.
- Real OCI, object, database and workload/endpoint assertions pass, including
  negative authorization and retained-data checks after compute removal.
- The installed Go provider calls only the published Platform HTTP management
  API and needs no Transit SDK/library, signer, principal coordinator or native
  journal. Independent data-plane clients remain test-harness responsibilities.
- Ordinary PR CI remains credential-free. Live test selection, isolation and
  cleanup are explicit; skipped/missing cases cannot qualify a capability.
- Executed evidence and remaining upstream limits are recorded. Neither a
  successful management response nor a running process alone establishes a
  service guarantee, broader backend support or public distribution.
