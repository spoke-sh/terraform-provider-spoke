# 15. Custom domains and private connectivity

[Roadmap](README.md) · **Status:** In Progress

Expose services through verified custom domains or explicitly authorized private
connections with observable TLS and routing lifecycle.

The [implementation plan](milestones-10-17-implementation-plan.md#milestone-15-custom-domains-and-private-connectivity)
sequences separate domain and connection slices with ownership and traffic gates.

## Delivered provider slice

The mock-v5 declarations, scoped discovery, deterministic fixture controls and
Terraform lifecycle/recovery coverage are implemented. See the
[execution record](../../qualification/milestones-10-17.md) and
[provisional contract](../../testdata/contracts/lifecycles.md) for the exact profile,
checks and limits. This does not complete the real-service acceptance criteria
below; owning Platform/Transit APIs and installed live evidence remain required.

## Prerequisites and scope

Extend [endpoints](08-container-services-and-endpoints.md) and
[scope/access](04-scope-and-access.md). Platform owns domain verification,
certificate issuance/renewal, routing and network attachment. External DNS/cloud
configuration remains with its provider unless a published platform API owns it.

## Delivery sequence

1. Define domain ownership challenge, expiry, re-verification, conflicting claims,
   TLS certificate state and binding to an exact endpoint. Non-secret challenge
   records may be outputs; private keys and credential bytes must not be.
2. Define private connection identities, both parties' authorization/acceptance,
   target scopes, routing constraints, address conflicts and deletion dependencies.
   Avoid implicit account or public-exposure changes.
3. Add independent domain and connection contract/mock slices, then adapters with
   asynchronous inspection, bounded timeouts and accepted-operation recovery.
   Renewal and routing reconciliation stay with the platform, not Terraform GET.

## Acceptance criteria

- Each slice passes lifecycle/import/empty-plan/drift, denial, revision conflict
  and interrupted mutation. Duplicate/unverified claims and cross-tenant targets
  are refused without taking over another owner's route.
- Expired challenges, issuance failures and pending peer acceptance remain
  explicit; a resource is not ready merely because its request was accepted.
- Removing a route or connection preserves external DNS, unrelated endpoints and
  separately owned network resources. Failed private routing does not fall back
  to public exposure.
- Installed/live evidence verifies DNS/TLS/traffic and private reachability plus
  denied paths and revocation. Advertise each supported network profile only
  after its real backend lifecycle passes the shared qualification gates.
