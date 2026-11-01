# 10. Live platform qualification

[Roadmap](README.md) · **Status:** In Progress

Prove that the initial Terraform service graph controls real platform services,
recovers original accepted work and leaves retained data usable after compute teardown.

See the [remaining implementation plan](milestones-10-17-implementation-plan.md#milestone-10-platform-http-integration-and-complete-live-graph)
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

At the recorded Platform revision `7af27c2bb8083ab64348592f82247da1ff703596`,
native management v2 supplies seeded bucket metadata, settlement authorization,
terminal outcomes, tombstones, bounded discovery and local TLS placement. It uses
HTTP endpoints carrying Transit-signed commands/proofs. Platform must publish and
implement the HTTP management abstraction that an ordinary Go client can consume
without a Transit SDK, signer, ABI/library or native journal. Its backend runtime
choices remain Platform's responsibility. The public routes/auth/version are not
invented here, and mock-v5 is not automatically the production contract.

The [recorded source assessment](milestones-10-17-implementation-plan.md#source-basis-and-changes-since-pr-8)
also leaves real object/database/OCI/workload services and the full graph open.
This planning revision does not claim a fresh upstream review or a new live API.
Adoption uses hard cutovers under the [roadmap policy](README.md#development-compatibility-policy).

## Delivered foundation and remaining gates

The provider's Go HTTP client, mock-v5 lifecycle/recovery and installed-artifact
checks are delivered. Unregistered native-v2 application codecs and the blocked
native-only preflight are historical implementation work. The
[execution record](../../qualification/milestones-10-17.md) preserves their results;
they are not qualification of the selected HTTP architecture.

Remaining work: agree the public contract, adapt the client/fixtures, qualify
server-backed idempotency and original-operation lookup, remove native scaffolding,
replace live configuration/report/workflow assumptions, and execute installed
metadata followed by the real graph. Removing obsolete SDK checks alone cannot
make the live gate pass. The [previous boundary record](../../qualification/native-boundary.md)
remains historical; its SDK gaps are no longer provider prerequisites.

## Delivery sequence

1. Pin the published Platform HTTP API/schema and exact Platform revision.
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
2. Replace the current native-only preflight with an explicitly selected HTTP
   live lane through `just test-acc live` and a separate configured workflow. Require endpoint/CA/tenant/credential and
   isolated fixture configuration; fail if absent or if no tests are selected.
   Remove mandatory Transit definition/pin and native journal inputs. Cut over
   report/workflow consumers together, retaining artifact and required-case checks.
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
