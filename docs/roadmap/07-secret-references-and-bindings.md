# 7. Secret references and service bindings

[Roadmap](README.md) · **Status:** Planned

Connect workloads to exact services and externally managed secrets without
placing secret values in Terraform or implicitly granting access.

See the [implementation plan](milestones-5-8-implementation-plan.md#milestone-7-secret-references-and-bindings)
and its [acyclic dependency contract](milestones-5-8-implementation-plan.md#acyclic-binding-and-workload-contract)
for contract work, delivery steps and test gates.

## Prerequisites and scope

Use [scope/access](04-scope-and-access.md) and the service identities from
[registries](05-artifact-registries.md) and [data services](06-databases-and-object-storage.md).
Define binding dependency rules with
[container services](08-container-services-and-endpoints.md) before either
adapter. Binding primitives can be tested using seeded consumers; the full
Terraform workload journey is milestone 8, avoiding a circular implementation gate.

## Delivery sequence

1. Specify `spoke_secret_reference` as an identifier/version reference to an
   external secret owner. Define validation, resolution authority and rotation
   behavior without returning secret bytes to Terraform.
2. Specify `spoke_service_binding` with exact consumer/service IDs, role or
   connection intent, dependency direction and deletion constraints. Binding
   records a connection; explicit grants authorize access separately.
3. Add fixture dependencies and test controls for missing services, denied secret
   resolution, changed reference versions, revoked grants and deleted consumers.
4. Implement resources and discovery with canonical imports, revision conditions
   and recoverable asynchronous mutations. Decide when references are immutable
   or replaceable and how consumers observe a changed configuration revision.
5. Define detach and service retirement order. Removing a binding does not purge
   the secret, revoke an independently owned grant or delete the target service.

## Acceptance criteria

- Reference/binding lifecycle, import, empty-plan, drift and interrupted mutation
  tests run through Terraform with independent fixture observations.
- Cross-scope dependencies and unauthorized resolution are rejected; a binding
  without an explicit grant cannot produce authorized access.
- A bound resource refuses deletion until its declared dependencies are removed.
  Detaching preserves the service, data, external secret and unrelated grants.
- Synthetic secret canaries are absent from state, saved plans, logs and error
  diagnostics, including rotation and failure cases.
- Reference changes and revocation are visible to the consumer without falsely
  claiming successful rollout. Actual workload access is proved in milestone 10.
