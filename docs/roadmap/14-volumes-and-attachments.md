# 14. Volumes and attachments

[Roadmap](README.md) · **Status:** In Progress

Attach persistent filesystem or block storage to workloads with explicit
ownership, writer fencing and data retention across replacement.

The [implementation plan](milestones-10-17-implementation-plan.md#milestone-14-volumes-and-attachments)
defines the proposed resources, attachment/fencing sequence and real-data gates.

## Delivered provider slice

The mock-v5 declarations, scoped discovery, deterministic fixture controls and
Terraform lifecycle/recovery coverage are implemented. See the
[execution record](../../qualification/milestones-10-17.md) and
[provisional contract](../../testdata/contracts/lifecycles.md) for the exact profile,
checks and limits. This does not complete the real-service acceptance criteria
below; Platform HTTP service APIs, their real backends and installed live
evidence remain required. All provider management calls use the
[shared HTTP contract](milestones-10-17-implementation-plan.md#shared-delivery-contract-for-milestones-1217).
Transit integration stays behind Platform; independent data-plane checks belong
to the qualification harness.

## Prerequisites and scope

Use [scope/access](04-scope-and-access.md) and
[container services](08-container-services-and-endpoints.md). Platform owns the
storage, mount/attach lifecycle and enforcement of writer authority. Buckets are
not POSIX volumes, and provider-side scripts cannot supply missing backend fencing.

## Delivery sequence

1. Define volume profiles, capacity/growth, supported access modes, placement
   compatibility, protection and separate attachment identity. Specify which
   settings can change in place without data loss.
2. Define attach/detach settlement, stale-writer fencing, failed-node recovery and
   forced-detach policy. A provider timeout does not establish detachment or make
   it safe to attach another writer.
3. Add fixture attachment ownership and fault transitions, then volume and
   attachment adapters with exact imports, revisions and recoverable operations.
   Keep workload replacement separate from volume deletion.

## Acceptance criteria

- Lifecycle/import/empty-plan/drift and interrupted attach/detach recover original
  operations. Concurrent incompatible writers are refused by the declared policy.
- Stale revisions, lost replies and unavailable workloads cannot bypass fencing
  or cause the provider to report a still-attached volume as safely detached.
- Workload replacement retains volume identity/data; protected or attached volume
  deletion is refused. Unsupported shrink or migration fails without mutation.
- Real installed/live tests write and reread data across replacement and test
  stale-writer exclusion. Mock attachment state is not evidence of filesystem
  durability, block semantics or backend fencing.
