# 17. Scaling and placement policies

[Roadmap](README.md) · **Status:** Planned

Declare capacity and placement intent while letting the platform reconcile
instances within explicit policy, quota and data-location constraints.

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
