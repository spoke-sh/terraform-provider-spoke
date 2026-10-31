# 8. Container services and endpoints

[Roadmap](README.md) · **Status:** In Progress

Deploy an application by immutable image digest, inspect its rollout and expose
an explicit endpoint while retaining separately owned services on retirement.

See the [implementation plan](milestones-5-8-implementation-plan.md#milestone-8-containers-endpoints-and-the-complete-graph)
for contract work, delivery steps and full-graph qualification.

## Prerequisites and scope

Use [scope/access](04-scope-and-access.md), [registries](05-artifact-registries.md),
[data services](06-databases-and-object-storage.md) and
[bindings](07-secret-references-and-bindings.md). Platform owns reconciliation,
instances, health and ingress. Custom domains and private connectivity are
separate [later scope](15-domains-and-private-connectivity.md).

## Delivery sequence

1. Define `spoke_container_service` with logical identity, immutable digest,
   resource limits, non-secret configuration, reference/binding inputs, desired
   revision, active revision and inspectable rollout outcome. Physical instances
   and individual build/push operations are not managed resources.
2. Specify readiness and bounded waiting separately from acceptance. Preserve
   the desired revision and original operation on failed rollout or timeout;
   never infer rollback from a failed provider call. Define replacement semantics.
3. Implement the service against strict fixture state transitions, including
   missing digest, denied pull, unavailable dependencies and failed health checks.
   GET only observes transitions controlled by the fixture controller.
4. Implement `spoke_endpoint` and exact reads with explicit private/public
   routing, target identity and TLS state. Unsupported routing choices fail
   validation; exposure never defaults to public by accident.
5. Compose scope → identity/grants → registry/database/bucket → service/bindings
   → endpoint. Use acyclic Terraform dependencies with the binding contract.
   Update the digest, revoke access, then retire endpoint/compute/bindings while
   retaining separately owned artifacts, databases and buckets.

## Acceptance criteria

- Each resource passes lifecycle/import/empty-plan/drift/revision and interrupted
  create/update/delete tests. Server records establish the requested digest and
  revision; desired, active and healthy are not conflated.
- The full graph works through real Terraform against mock endpoints, including
  recovery of a partially applied graph without duplicate services.
- Denied pull/access, failed health and pending timeout are distinguishable;
  diagnostics retain safe recovery context and known resource state.
- Endpoint deletion and compute retirement respect dependency order and leave
  independently owned data, artifacts, secrets and grants intact.
- Generated docs and examples identify each draft contract and mock-qualified
  operation. Real image pull, workload access and endpoint traffic remain gates
  in [live qualification](10-live-platform-qualification.md).
