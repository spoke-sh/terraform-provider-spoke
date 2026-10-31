# 12. Jobs and schedules

[Roadmap](README.md) · **Status:** Planned

Declare repeatable job definitions and schedules while treating individual
executions as observations owned by the platform scheduler.

## Prerequisites and scope

Use [scope/access](04-scope-and-access.md), [artifacts](05-artifact-registries.md),
[bindings](07-secret-references-and-bindings.md) and the workload model in
[milestone 8](08-container-services-and-endpoints.md). Platform must define job
admission, scheduling and execution recovery. This is subsequent service scope,
not a prerequisite for the first provider release.

## Delivery sequence

1. Specify job definition identity, immutable image digest, inputs/references,
   execution authority, schedule/time-zone interpretation, concurrency limits,
   missed-run behavior, retries and enable/disable semantics.
2. Define update and retirement with active executions. Stop future admission
   explicitly; cancellation requests and their outcomes are distinct from a
   completed cancellation or rollback of external effects.
3. Add versioned contract and deterministic fixture clock/controller behavior,
   then resource adapters and bounded execution-status discovery. Plan and
   refresh must not start executions or advance the fixture schedule.

## Acceptance criteria

- Definition/schedule lifecycle, import, empty-plan, revision conflict, denied
  access and interrupted mutation pass through Terraform and HTTP.
- Fake-clock tests cover schedule boundaries, missed runs, overlapping work,
  disabled admission and update/delete with active executions. Provider retries
  cannot accidentally submit an additional execution.
- Retirement preserves execution evidence, separately owned artifacts/data and
  explicit uncertainty about outstanding work. Execution outputs contain no
  secret material exposed to Terraform.
- Apply the installed and [live qualification](10-live-platform-qualification.md)
  gates to the real scheduler before advertising support; require observable
  execution and recovery evidence, not just a stored schedule definition.
