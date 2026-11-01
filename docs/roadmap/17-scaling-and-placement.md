# 17. Scaling and placement policies

[Roadmap](README.md) · **Status:** In Progress

Declare capacity and placement intent while letting the platform reconcile
instances within explicit policy, quota and data-location constraints.

The [implementation plan](milestones-10-17-implementation-plan.md#milestone-17-scaling-and-placement-policies)
separates compute policy from storage-aware movement and defines field ownership,
upstream dependencies and transition evidence.

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

Extend [container services](08-container-services-and-endpoints.md) and
[scope/access](04-scope-and-access.md). Storage-aware policies additionally depend
on [data service](06-databases-and-object-storage.md) or
[volume](14-volumes-and-attachments.md) contracts. Platform owns scheduling,
autoscaling, relocation and physical instance lifecycle.

## Delivery sequence

1. Define policy identity, min/max capacity, metrics or schedule sources where
   supported, placement/residency constraints, quota and observation freshness.
   Assign one reconciler per mutable field: Terraform manages policy intent;
   platform controllers manage observed instance counts and placement.
2. Specify manual capacity versus autoscaling ownership, policy updates,
   constraint conflicts, unschedulable outcomes and safe retirement. Reject
   unsupported combinations instead of masking perpetual plan drift.
3. Add deterministic fixture demand/capacity transitions and interrupted
   relocation, then adapters and bounded status discovery. Reuse accepted-operation
   recovery and storage fencing; GET must not schedule or move work.

## Acceptance criteria

- Lifecycle/import/empty-plan/drift, denial, stale revision and interrupted
  updates preserve policy identity. Controller-owned observations do not create
  a Terraform overwrite loop.
- Bounds, quota, residency and incompatible attachment constraints are enforced;
  unsatisfied intent is inspectable without claiming successful placement.
- Removing a policy does not implicitly delete workloads or persistent data.
  Relocation retains original operation identity and storage writer fencing.
- Installed/live tests demonstrate policy-driven scale/placement, quota refusal,
  access control and workload/data continuity through supported transitions.
  Resource availability, performance and placement claims name the tested backend
  profile and measurements rather than relying on desired management state.
