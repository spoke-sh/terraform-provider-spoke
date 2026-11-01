# 13. Streams and queues

[Roadmap](README.md) · **Status:** In Progress

Declare messaging services with explicit retention, capacity and producer/consumer
authority while keeping messages and delivery acknowledgements in the data plane.

The [implementation plan](milestones-10-17-implementation-plan.md#milestone-13-streams-and-queues)
maps current Transit messaging primitives to the remaining platform/provider work.

## Delivered provider slice

The mock-v5 declarations, scoped discovery, deterministic fixture controls and
Terraform lifecycle/recovery coverage are implemented. See the
[execution record](../../qualification/milestones-10-17.md) and
[provisional contract](../../testdata/contracts/lifecycles.md) for the exact profile,
checks and limits. This does not complete the real-service acceptance criteria
below; owning Platform/Transit APIs and installed live evidence remain required.

## Prerequisites and scope

Use [scope/access](04-scope-and-access.md) and
[service bindings](07-secret-references-and-bindings.md). Platform must publish
customer stream/queue service contracts over its selected runtime primitives;
the provider does not expose private Transit storage or implement message brokers.

## Delivery sequence

1. Specify stable stream/queue identities, ordering/delivery profiles, retention,
   capacity, consumer ownership, dead-letter policy where supported, immutable
   settings and protected deletion of retained or in-flight data.
2. Add strict contract fixtures for creation/readiness, grants, consumer
   dependencies, quota changes and in-use/nonempty deletion. Define destructive
   retention changes explicitly without creating an implicit purge path.
3. Implement service resources and bounded discovery; keep produce, consume,
   claim, acknowledgement and individual messages outside managed lifecycle.
   Define bindings without granting access as a side effect.

## Acceptance criteria

- Terraform lifecycle, canonical import, empty-plan, drift, denial, stale
  revision and interrupted mutation preserve identity and declared retention.
- Dependent consumers, retained data and in-flight deliveries prevent unsafe
  deletion; retiring workloads does not erase independent messaging services.
- Fixture reads are observational. Quota/refusal and incomplete discovery cannot
  be reported as empty or successfully deleted services.
- Installed/live checks prove the advertised produce/consume or claim/settle
  behavior, access revocation and retained data after workload replacement.
  Delivery guarantees are named per qualified backend profile, not inferred
  from successful management provisioning.
