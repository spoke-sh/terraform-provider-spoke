# 10. Live platform qualification

[Roadmap](README.md) · **Status:** In Progress

Prove that the initial Terraform service graph controls real platform services,
recovers original accepted work and leaves retained data usable after compute teardown.

See the [remaining implementation plan](milestones-10-17-implementation-plan.md#milestone-10-native-integration-and-complete-live-graph)
for native signing, durable pre-dispatch recovery, backend prerequisites and the
explicit live lane.

## Prerequisites and scope

Contract coordination starts in [milestone 2](02-management-contracts-and-discovery.md).
Complete this milestone using the [recovery matrix](03-recovery-isolation-and-protection.md),
initial graph in [milestones 4](04-scope-and-access.md)–[8](08-container-services-and-endpoints.md)
and [installed artifact](09-nix-packages-and-consumers.md). Platform must provide
published versioned management APIs, automation auth, service controllers and
an isolated qualification environment. Provider mocks cannot satisfy those
upstream prerequisites. Qualify resources incrementally as their APIs become available.

## Current upstream boundary

Platform `7af27c2bb8083ab64348592f82247da1ff703596` now supplies native management
v2: seeded bucket metadata with capacity/retention, commit-time authorization,
terminal failure, expiry/tombstones, complete object/pending lists and local
operator/TLS placement. Its installed-provider harness is reusable, but the
recorded direct-Go-signing prototype does not satisfy this repository's SDK rule.
There is still no full customer graph or object data plane.

The [refreshed assessment](milestones-10-17-implementation-plan.md#source-basis-and-changes-since-pr-8)
compares that service with Transit `a000b8d18449fdee7901955d0c94417ceeab32eb`.
The new Zig context/SQL recovery store is not a Go Serving adapter; public
Serving/proof/custom-CA interfaces remain missing. Platform still pins Transit
`0e2fba4`, so adopting the newer native definition requires a coordinated cutover.
Adaptation uses hard cutovers under the [roadmap policy](README.md#development-compatibility-policy);
no legacy mock-to-native compatibility adapter is required.

## Delivered foundation and remaining gates

Native-v2 application command/read-context/result codecs now use the exact owner
schema and vectors, including terminal outcomes, tombstones and complete bounded
observations. Pending creates and bucket reads after receipt expiry have explicit
regression tests. The codecs remain unregistered: no signer or transport is
implemented outside the shared SDK.

The explicit live preflight emits blocked evidence and exits nonzero. The Go SDK
still lacks Serving/proof/TLS interfaces; native adapter/journal, installed metadata
and full-graph runs remain unqualified. The [current execution record](../../qualification/milestones-10-17.md)
separates these delivered application codecs from the remaining upstream and live
gates. The [previous boundary record](../../qualification/native-boundary.md)
preserves the original native-v1 assessment.

## Delivery sequence

1. Pin the platform API/schema and exact platform revision. Reconcile scope,
   automation credentials, IDs, errors, capability negotiation, revisions,
   acceptance/settlement, ownership, receipt retention and deletion semantics.
   Update boundary types, fixture contracts, client, Terraform schemas and docs
   together by hard cutover; record divergences without retaining legacy paths.
   Persist exact native prepared work and coordinate principal sequences before
   dispatch; Terraform state alone is not a pre-dispatch recovery journal.
2. Implement an explicitly selected live lane through `just test-acc live` and
   a separate configured workflow. Require endpoint/CA/tenant/credential and
   isolated fixture configuration; fail if absent or if no tests are selected.
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
- Interrupted operations and server/controller restart preserve original intent
  without duplicate logical resources. Uncertainty remains inspectable; timeout
  or lost authority does not imply rollback or authoritative absence.
- Real OCI, object, database and workload/endpoint assertions pass, including
  negative authorization and retained-data checks after compute removal.
- Ordinary PR CI remains credential-free. Live test selection, isolation and
  cleanup are explicit; skipped/missing cases cannot qualify a capability.
- Executed evidence and remaining upstream limits are recorded. Neither a
  successful management response nor a running process alone establishes a
  service guarantee, broader backend support or public distribution.
