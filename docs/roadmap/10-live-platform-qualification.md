# 10. Live platform qualification

[Roadmap](README.md) · **Status:** In Progress

Prove that the initial Terraform service graph controls real platform services,
recovers original accepted work and leaves retained data usable after compute teardown.

See the [implementation plan](milestones-9-11-implementation-plan.md#milestone-10-native-boundary-and-real-service-qualification)
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

At platform revision `b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4`, a locally
qualified native management profile implements signed commands, observational
reads and durable receipts for seeded-scope bucket metadata. It does not implement
the full customer graph or object data plane. The
[plan's source assessment](milestones-9-11-implementation-plan.md#baseline-and-refreshed-upstream-assessment)
records its schema, SDK/native runtime constraints and qualification evidence.
Adaptation uses hard cutovers under the [roadmap policy](README.md#development-compatibility-policy);
no legacy mock-to-native compatibility adapter is required.

## Delivered foundation and remaining gates

Native application command/read-context/result codecs and an explicit configured
live preflight are implemented. Independent byte vectors pass. The live gate
emits blocked evidence and exits nonzero: the pinned Go SDK lacks Serving and
subject-proof interfaces, the installed native adapter/journal is unqualified,
and the platform exposes metadata only. No native request was sent or real graph
qualified. See the [boundary record](../../qualification/native-boundary.md).

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
