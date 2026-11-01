# 16. Backup policies and database clones

[Roadmap](README.md) · **Status:** In Progress

Declare recovery policy and isolated database clones with exact source provenance,
current authorization and independently verified restored data.

The [implementation plan](milestones-10-17-implementation-plan.md#milestone-16-backup-policies-and-database-clones)
distinguishes Transit's delivered single-Table branch from the remaining managed
backup and writable database-clone requirements.

## Delivered provider slice

The mock-v5 declarations, scoped discovery, deterministic fixture controls and
Terraform lifecycle/recovery coverage are implemented. See the
[execution record](../../qualification/milestones-10-17.md) and
[provisional contract](../../testdata/contracts/lifecycles.md) for the exact profile,
checks and limits. This does not complete the real-service acceptance criteria
below; owning Platform/Transit APIs and installed live evidence remain required.

## Prerequisites and scope

Extend [databases/storage](06-databases-and-object-storage.md) and
[scope/access](04-scope-and-access.md). Platform owns backup execution, restore
consistency, encryption, retention and recovery guarantees. Individual backup
executions are observations unless a separate lifecycle contract justifies management.

## Delivery sequence

1. Specify backup policy identity, schedule, retention, recovery-point visibility,
   source version and deletion dependencies. Separate configured policy from
   completed backups and measured recovery guarantees.
2. Define clone identity and exact source snapshot/recovery point, destination
   scope, authorization, encryption and lineage. A clone is separately owned;
   mutable source changes must not silently retarget it.
3. Add mock policy/clone lifecycles with retained dependencies, failed/incomplete
   backups and interrupted restore. Implement provider adapters and observational
   discovery after these contracts are explicit.

## Acceptance criteria

- Lifecycle, import, empty-plan, revision conflict and interrupted clone/restore
  preserve original accepted identity and source provenance without duplicates.
- Unavailable recovery points, denied source access and incomplete restore never
  appear as successful ready clones. Cross-scope clones require explicit authority.
- Source, clone and policy deletion honor retained dependencies and protection;
  retiring one does not silently delete the others or their required backup data.
- Real installed/live tests restore known data, verify the selected source cut,
  enforce current access and demonstrate independent changes in source and clone.
  Publish measured recovery limits separately from policy configuration; mock
  metadata does not qualify a recovery-point or recovery-time guarantee.
